package cdn_checker

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/ivpusic/grpool"
	"io"
	llog "log"
	"math/rand"
	"mirrors_status/pkg/config"
	"mirrors_status/pkg/log"
	"mirrors_status/pkg/modules/model"
	"mirrors_status/pkg/modules/service"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type CDNChecker struct {
	CheckTool CheckTool
}

func NewCDNChecker(conf *configs.CdnCheckerConf) *CDNChecker {
	return &CDNChecker{
		CheckTool: NewCheckTool(conf),
	}
}

var (
	optMirror string
	optNoTestHidden bool
	optDevEnv bool

	maxNumOfRetries int

	clientNormal *http.Client
	clientHidden *http.Client

	numMirrorsTotal int
	numMirrorsFinished int
	numMirrorsMu sync.Mutex
)

var regErrLookupTimeout = regexp.MustCompile("lookup.*on.*read udp.*i/o timeout")

var dialTcpTimeoutMap = make(map[string]int)
var dialTcpTimeoutMapMu sync.Mutex

var regDialTcpTimeout = regexp.MustCompile(`dial tcp (\S+): i/o timeout`)

var dnsCache = make(map[string][]string)

type cdnTestResultSlice []*TestResult

func (v cdnTestResultSlice) Len() int {
	return len(v)
}

func (v cdnTestResultSlice) Less(i, j int) bool {
	return v[i].Percent < v[j].Percent
}

func (v cdnTestResultSlice) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func GetHttpClient(mirrorWeight int) *http.Client {
	if mirrorWeight >= 0 {
		return clientNormal
	}
	return clientHidden
}

func ParseContentRange(str string) (posBegin, posEnd, total int, err error) {
	_, err = fmt.Sscanf(str, "bytes %d-%d/%d", &posBegin, &posEnd, &total)
	if err != nil {
		err = fmt.Errorf("parseContentRange: %q error %s", str, err.Error())
	}
	return
}

func CheckFileReq0(filePath string, req *http.Request, client *http.Client) (*FileValidateInfo, error) {
	size := 4 * 1024
	// 第一次请求
	req.Header.Set("Range", "bytes=0-" + strconv.Itoa(size-1))
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	code := resp.StatusCode / 100
	if code != 2 {
		return nil, fmt.Errorf("response status is %s", resp.Status)
	}

	modTime := resp.Header.Get("Last-Modified")
	contentRange := resp.Header.Get("Content-Range")
	posStart, postEnd, total, err := ParseContentRange(contentRange)
	if err != nil {
		return nil, err
	}

	if posStart != 0 {
		return nil, errors.New("posStart != 0")
	}

	buf := make([]byte, size)
	n, err := io.ReadFull(resp.Body, buf)
	if err == io.ErrUnexpectedEOF {
		if n != postEnd+1 {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	md5hash := md5.New()
	_, err = md5hash.Write(buf[:n])
	if err != nil {
		return nil, err
	}
	vi := &FileValidateInfo{
		FilePath: filePath,
		Size:     total,
		ModTime:  modTime,
		URL:      req.URL.String(),
	}

	if total <= size {
		vi.MD5Sum = md5hash.Sum(nil)
		return vi, nil
	}

	secondPosBegin := total - size
	if total < size*2 {
		secondPosBegin = size
	}

	// 第二次请求
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", secondPosBegin, total-1))
	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	contentRange = resp.Header.Get("Content-Range")
	posStart2, postEnd2, total2, err2 := ParseContentRange(contentRange)
	if err2 != nil {
		return nil, err2
	}
	if posStart2 != secondPosBegin {
		return nil, errors.New("2rd req posStart not match")
	}
	if postEnd2 != total-1 {
		return nil, errors.New("2rd req posEnd not match")
	}
	if total != total2 {
		return nil, errors.New("total not match")
	}

	buf2Size := total - 1 - secondPosBegin + 1
	if n+buf2Size > total {
		panic("assert failed: n + buf2size <= total")
	}

	buf2 := buf[:buf2Size]

	_, err = io.ReadFull(resp.Body, buf2)
	if err != nil {
		return nil, err
	}

	_, err = md5hash.Write(buf2)
	if err != nil {
		return nil, err
	}

	vi.MD5Sum = md5hash.Sum(nil)
	return vi, nil
}

func checkFileReq(filePath string, req *http.Request, allowRetry bool,
	client *http.Client) (vi *FileValidateInfo, err error) {
	retryDelay := func() {
		ms := rand.Intn(3000) + 100
		time.Sleep(time.Duration(ms) * time.Millisecond)
	}
	n := 1
	if allowRetry {
		n += maxNumOfRetries
	}

loop0:
	for i := 0; i < n; i++ {
		if i > 0 {
			log.Infof("Retry %s for %d times", i, req.URL)
		}

		vi, err = CheckFileReq0(filePath, req, client)

		if err != nil {
			log.Errorf("Check file of:%s found error:%v", req.URL, err)
			if !allowRetry {
				return
			}

			allowRetryErrMessages := []string{
				"connection reset by peer",
				"Client.Timeout exceeded while reading body",
				"network is unreachable",
				"TLS handshake timeout",
				"connection refused",
				"connection timed out",
				"Service Temporarily Unavailable",
				"Internal Server Error",
				"Bad Gateway",
			}

			errMsg := err.Error()
			for _, msg := range allowRetryErrMessages {
				if strings.Contains(errMsg, msg) {
					retryDelay()
					continue loop0
				}
			}

			match := regDialTcpTimeout.FindStringSubmatch(errMsg)
			if len(match) >= 2 {
				host := match[1]
				dialTcpTimeoutMapMu.Lock()
				num := dialTcpTimeoutMap[host]
				if num > 25 {
					dialTcpTimeoutMapMu.Unlock()
					return
				}
				dialTcpTimeoutMap[host]++
				dialTcpTimeoutMapMu.Unlock()
				retryDelay()
				continue loop0
			}
			if regErrLookupTimeout.MatchString(errMsg) {
				retryDelay()
				continue loop0
			}
			return
		}
		if i > 0 {
			log.Infof("Retry %s for %d times success", i, req.URL)
		}
		return
	}
	log.Infof("Retry failed for:%s", req.URL)
	return
}

func CheckFile(urlPrefix string, filePath string, allowRetry bool,
	client *http.Client) (*FileValidateInfo, error) {
	if !strings.HasSuffix(urlPrefix, "/") {
		urlPrefix += "/"
	}

	url0 := urlPrefix + filePath
	log.Infof("Ckeck file for:%s", url0)
	req, err := http.NewRequest(http.MethodGet, url0, nil)
	if err != nil {
		log.Errorf("Check file found error:%v", err)
		return nil, err
	}
	return checkFileReq(filePath, req, allowRetry, client)
}

func (checker CDNChecker) GetValidateInfoList(files []string) ([]*FileValidateInfo, error) {
	var validateInfoList []*FileValidateInfo
	var mu sync.Mutex
	client := GetHttpClient(9999)
	pool := grpool.NewPool(3, 1)
	defer pool.Release()
	pool.WaitCount(len(files))

	for _, file := range files {
		fileCopy := file
		pool.JobQueue <- func() {
			defer pool.JobDone()

			vi, err := CheckFile(checker.CheckTool.Conf.SourceUrl, fileCopy, true, client)
			if err != nil {
				return
			}
			mu.Lock()
			validateInfoList = append(validateInfoList, vi)
			mu.Unlock()
		}
	}

	pool.WaitAll()
	return validateInfoList, nil
}

func (checker *CDNChecker) PrefetchCdnDns(host string) error {
	ips, ok := dnsCache[host]
	if ok {
		return nil
	}

	var err error
	ips, err = checker.CheckTool.TestDNS(host)
	if err != nil {
		return err
	}

	dnsCache[host] = ips
	return nil
}

func (checker *CDNChecker) GetCdnDns(host string) []string {
	ips, ok := dnsCache[host]
	if !ok {
		if host == checker.CheckTool.Conf.DefaultCdn {
			return []string{
				"1.192.192.70",
				"221.130.199.56",
				"42.236.10.34",
				"36.110.211.9",
				"52.0.26.226",
			}
		}
	}
	return ips
}

func (checker *CDNChecker) CheckFileCdn(fileInfo FileInfo, cdnIp string, client *http.Client) (*FileValidateInfo, error) {
	url0 := "http://" + cdnIp + "/deepin/" + fileInfo.FilePath
	log.Infof("Check file CDN for:%s", url0)
	req, err := http.NewRequest(http.MethodGet, url0, nil)
	if err != nil {
		log.Errorf("Check file CDN found error:%v", err)
		return nil, err
	}
	req.Host = checker.CheckTool.Conf.DefaultCdn
	vi, err := checkFileReq(fileInfo.FilePath, req, true, client)
	return vi, err
}

func (vi *FileValidateInfo) Equal(other *FileValidateInfo) bool {
	return vi.FilePath == other.FilePath &&
		vi.Size == other.Size &&
		bytes.Equal(vi.MD5Sum, other.MD5Sum)
}

func MakeResultDir() error {
	err := os.Mkdir("result", 0755)
	if err != nil && !os.IsExist(err) {
		return err
	}
	return nil
}

func (tr *TestResult) Save() error {
	err := MakeResultDir()
	if err != nil {
		return err
	}

	var filename string
	if tr.CdnNodeAddress == "" {
		filename = tr.Name + ".txt"
	} else {
		filename = fmt.Sprintf("%s-%s.txt", tr.Name, tr.CdnNodeAddress)
	}
	filename = filepath.Join("result", filename)
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	bw := bufio.NewWriter(f)

	_, _ = fmt.Fprintln(bw, "name:", tr.Name)
	_, _ = fmt.Fprintln(bw, "urlPrefix:", tr.UrlPrefix)

	if tr.CdnNodeAddress != "" {
		_, _ = fmt.Fprintln(bw, "cdn node address:", tr.CdnNodeAddress)
	}
	_, _ = fmt.Fprintf(bw, "percent: %.3f%%\n", tr.Percent)
	if tr.Percent == 100 {
		_, _ = fmt.Fprintln(bw, "sync completed")
	}

	// err
	for _, record := range tr.Records {
		if record.Err != nil {

			_, _ = fmt.Fprintln(bw, "has error")
			break
		}
	}
	_, _ = fmt.Fprintln(bw, "\n# Error:")
	for _, record := range tr.Records {
		if record.Err == nil {
			continue
		}

		_, _ = fmt.Fprintln(bw, "file path:", record.Standard.FilePath)
		_, _ = fmt.Fprintln(bw, "standard url:", record.Standard.URL)
		_, _ = fmt.Fprintln(bw, "err:", record.Err)
		_, _ = fmt.Fprintln(bw, "errDump:", spew.Sdump(record.Err))
		_, _ = fmt.Fprintln(bw)
	}

	// not equal
	_, _ = fmt.Fprintln(bw, "\n# Not Equal:")
	for _, record := range tr.Records {
		if record.Result == nil || record.Equal {
			continue
		}

		// result is not nil and not equal
		_, _ = fmt.Fprintln(bw, "file path:", record.Standard.FilePath)
		_, _ = fmt.Fprintln(bw, "standard url:", record.Standard.URL)
		_, _ = fmt.Fprintln(bw, "url:", record.Result.URL)

		_, _ = fmt.Fprintln(bw, "standard size:", record.Standard.Size)
		_, _ = fmt.Fprintln(bw, "size:", record.Result.Size)

		_, _ = fmt.Fprintf(bw, "standard md5sum: %x\n", record.Standard.MD5Sum)
		_, _ = fmt.Fprintf(bw, "md5sum: %x\n", record.Result.MD5Sum)

		_, _ = fmt.Fprintln(bw, "standard mod time:", record.Standard.ModTime)
		_, _ = fmt.Fprintln(bw, "mod time:", record.Result.ModTime)

		_, _ = fmt.Fprintln(bw)
	}

	// equal
	_, _ = fmt.Fprintln(bw, "\n# Equal:")
	for _, record := range tr.Records {
		if !record.Equal {
			continue
		}

		_, _ = fmt.Fprintln(bw, "file path:", record.Standard.FilePath)
		_, _ = fmt.Fprintln(bw, "standard url:", record.Standard.URL)
		_, _ = fmt.Fprintln(bw, "url:", record.Result.URL)

		_, _ = fmt.Fprintln(bw, "size:", record.Result.Size)
		_, _ = fmt.Fprintf(bw, "md5sum: %x\n", record.Result.MD5Sum)

		_, _ = fmt.Fprintln(bw, "standard mod time:", record.Standard.ModTime)
		_, _ = fmt.Fprintln(bw, "mod time:", record.Result.ModTime)

		_, _ = fmt.Fprintln(bw)
	}

	err = bw.Flush()
	if err != nil {
		return err
	}

	return nil
}


func (checker *CDNChecker) TestCdnNode(mirrorId, urlPrefix, cdnNodeAddress string, validateInfoList []*FileValidateInfo) *TestResult {
	pool := grpool.NewPool(6, 1)
	defer pool.Release()
	var mu sync.Mutex
	records := make([]TestRecord, 0, len(validateInfoList))
	var good int
	var numErrs int

	client := GetHttpClient(1000)

	pool.WaitCount(len(validateInfoList))
	for _, validateInfo := range validateInfoList {
		vi := validateInfo
		pool.JobQueue <- func() {
			validateInfo1, err := checker.CheckFileCdn(FileInfo{
				FilePath: vi.FilePath,
			}, cdnNodeAddress, client)

			var record TestRecord
			record.Standard = vi
			mu.Lock()
			if err != nil {
				numErrs++
				log.Warningf("Check file CDN found error:%v", err)
				record.Err = err

			} else {
				record.Result = validateInfo1
				if vi.Equal(validateInfo1) {
					good++
					record.Equal = true
				}
			}
			records = append(records, record)

			mu.Unlock()
			pool.JobDone()
		}
	}

	pool.WaitAll()
	percent := float64(good) / float64(len(validateInfoList)) * 100.0

	r := &TestResult{
		Name:           mirrorId,
		UrlPrefix:      urlPrefix,
		CdnNodeAddress: cdnNodeAddress,
		Records:        records,
		Percent:        percent,
		NumErrs:        numErrs,
	}

	err := r.Save()
	if err != nil {
		log.Warningf("Save result found error:%v", err)
	}

	return r
}

func (checker *CDNChecker) TestMirrorCdn(mirrorId, urlPrefix string,
	validateInfoList []*FileValidateInfo) []*TestResult {
	u, err := url.Parse(urlPrefix)
	if err != nil {
		panic(err)
	}

	ips := checker.GetCdnDns(u.Hostname())
	log.Infof("Testing mirror:%s, IPs:%v", mirrorId, ips)
	if len(ips) == 0 {
		return []*TestResult{
			{
				Name: mirrorId,
			},
		}
	}

	pool := grpool.NewPool(len(ips), 1)
	defer pool.Release()

	var testResults cdnTestResultSlice
	var testResultsMu sync.Mutex

	pool.WaitCount(len(ips))
	for _, cdnAddress := range ips {
		cdnAddressCopy := cdnAddress
		pool.JobQueue <- func() {
			testResult := checker.TestCdnNode(mirrorId, urlPrefix, cdnAddressCopy, validateInfoList)
			testResultsMu.Lock()
			testResults = append(testResults, testResult)
			testResultsMu.Unlock()
			pool.JobDone()
		}
	}
	pool.WaitAll()

	sort.Sort(testResults)
	return testResults
}

func GetMirrorsTestProgressDesc() string {
	numMirrorsMu.Lock()
	str := fmt.Sprintf("[%d/%d]", numMirrorsFinished, numMirrorsTotal)
	numMirrorsMu.Unlock()
	return str
}

func TestMirrorCommon(mirrorId, urlPrefix string, mirrorWeight int,
	validateInfoList []*FileValidateInfo) *TestResult {
	if urlPrefix == "" {
		return &TestResult{
			Name: mirrorId,
		}
	}

	pool := grpool.NewPool(6, 1)
	defer pool.Release()
	var mu sync.Mutex
	numTotal := len(validateInfoList)
	records := make([]TestRecord, 0, numTotal)
	var good int
	var numErrs int
	var numCompleted int

	client := GetHttpClient(mirrorWeight)

	pool.WaitCount(numTotal)

	for _, validateInfo := range validateInfoList {
		vi := validateInfo
		pool.JobQueue <- func() {
			validateInfo1, err := CheckFile(urlPrefix, vi.FilePath, mirrorWeight >= 0, client)

			var record TestRecord
			record.Standard = vi
			mu.Lock()
			numCompleted++
			log.Infof("%s %s [%d/%d]", GetMirrorsTestProgressDesc(), mirrorId, numCompleted, numTotal)
			if err != nil {
				numErrs++
				log.Warningf("Check file found error:%v", err)
				record.Err = err

			} else {
				record.Result = validateInfo1
				if vi.Equal(validateInfo1) {
					good++
					record.Equal = true
				}
			}
			records = append(records, record)

			mu.Unlock()
			pool.JobDone()
		}
	}
	pool.WaitAll()
	percent := float64(good) / float64(len(validateInfoList)) * 100.0

	r := &TestResult{
		Name:      mirrorId,
		UrlPrefix: urlPrefix,
		Records:   records,
		Percent:   percent,
		NumErrs:   numErrs,
	}

	err := r.Save()
	if err != nil {
		log.Warningf("Save result found error:%v", err)
	}

	return r
}

func (checker *CDNChecker) TestMirror(mirrorId string, urlPrefix string, mirrorWeight int,
	validateInfoList []*FileValidateInfo) []*TestResult {
	log.Infof("Testing mirror:%q, Prefix:%q, Weight:%d", mirrorId, urlPrefix, mirrorWeight)

	if mirrorId == "default" {
		// is cdn
		return checker.TestMirrorCdn(mirrorId, urlPrefix, validateInfoList)
	}
	r := TestMirrorCommon(mirrorId, urlPrefix, mirrorWeight, validateInfoList)
	return []*TestResult{r}
}

func TestMirrorFinish() {
	numMirrorsMu.Lock()
	numMirrorsFinished++
	numMirrorsMu.Unlock()
}

func (checker *CDNChecker) PushAllMirrorsTestResults(testResults []*TestResult) error {
	var mirrorsPoints []model.MirrorsPoint
	var mirrorsCdnPoints []model.MirrorsCdnPoint

	var mirrorsPointsAppendedMap = make(map[string]struct{})
	for _, testResult := range testResults {
		if testResult.CdnNodeAddress == "" {
			if testResult.UrlPrefix != "" {
				mirrorsPoints = append(mirrorsPoints, model.MirrorsPoint{
					Name:     testResult.UrlPrefix,
					Progress: testResult.Percent / 100.0,
				})
			}
		} else {
			if _, ok := mirrorsPointsAppendedMap[testResult.Name]; !ok {
				mirrorsPoints = append(mirrorsPoints, model.MirrorsPoint{
					Name:     testResult.UrlPrefix,
					Progress: testResult.Percent / 100.0,
				})
				mirrorsPointsAppendedMap[testResult.Name] = struct{}{}
			}

			mirrorsCdnPoints = append(mirrorsCdnPoints, model.MirrorsCdnPoint{
				MirrorId:   testResult.Name,
				NodeIpAddr: testResult.CdnNodeAddress,
				Progress:   testResult.Percent / 100.0,
			})
		}
	}
	now := time.Now()
	err := configs.GetInfluxdbClient().PushMirrors(now, mirrorsPoints)
	if err != nil {
		log.Errorf("Push to mirrors found error:%v", err)
		return err
	}

	err = configs.GetInfluxdbClient().PushMirrorsCdn(now, mirrorsCdnPoints)
	if err != nil {
		log.Errorf("Push to mirrors_cdn found error:%v", err)
		return err
	}
	return nil
}

func (checker *CDNChecker) TestAllMirrors(mirrors0 mirrors, validateInfoList []*FileValidateInfo) {
	if optNoTestHidden {
		var tempMirrors mirrors
		for _, mirror := range mirrors0 {
			if mirror.Weight >= 0 {
				tempMirrors = append(tempMirrors, mirror)
			}
		}
		mirrors0 = tempMirrors
	}

	pool := grpool.NewPool(50, 1)
	defer pool.Release()
	pool.WaitCount(len(mirrors0))

	numMirrorsTotal = len(mirrors0)

	t0 := time.Now()
	var testResults []*TestResult
	var mu sync.Mutex

	var names []string
	for _, mirror := range mirrors0 {
		names = append(names, mirror.Name)
		mirrorCopy := mirror
		pool.JobQueue <- func() {
			t1 := time.Now()
			testResult := checker.TestMirror(mirrorCopy.Id, mirrorCopy.GetUrlPrefix(),
				mirrorCopy.Weight, validateInfoList)
			TestMirrorFinish()
			duration0 := time.Since(t0)
			duration1 := time.Since(t1)

			log.Infof("%s test finished for mirror:%q. Duration:%v, Since:%v", GetMirrorsTestProgressDesc(), mirrorCopy.Id, duration1, duration0)
			mu.Lock()
			testResults = append(testResults, testResult...)
			mu.Unlock()
			pool.JobDone()
		}
	}
	pool.WaitAll()

	for _, mirror := range mirrors0 {
		service.CreateOperation(configs.GetMySQLClient(), model.MirrorOperation{
			CreateDate: time.Now(),
			OperationType: model.SYNC,
			MirrorId: mirror.Id,
		})
	}

	checker.PushAllMirrorsTestResults(testResults)
}

func (checker *CDNChecker) Init(c *configs.CdnCheckerConf) error {
	checker.CheckTool = CheckTool{
		Conf: c,
	}

	rand.Seed(time.Now().UnixNano())
	flag.Parse()
	llog.SetFlags(llog.Lshortfile)

	tlsCfg := &tls.Config{InsecureSkipVerify: true}
	if optDevEnv {
		clientNormal = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
					DualStack: true,
				}).DialContext,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
				TLSClientConfig:       tlsCfg,
			},
			Timeout: 1 * time.Minute,
		}

		clientHidden = clientNormal
		maxNumOfRetries = 2
	} else {
		clientNormal = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   60 * time.Second,
					KeepAlive: 60 * time.Second,
					DualStack: true,
				}).DialContext,
				MaxIdleConns:          100,
				IdleConnTimeout:       120 * time.Second,
				TLSHandshakeTimeout:   20 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
				TLSClientConfig:       tlsCfg,
			},
			Timeout: 3 * time.Minute,
		}

		clientHidden = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
					DualStack: true,
				}).DialContext,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
				TLSClientConfig:       tlsCfg,
			},
			Timeout: 1 * time.Minute,
		}
		maxNumOfRetries = 4
	}
	// Get unpublished mirror list
	mirrors, err := GetUnpublishedMirrors(c.Target)
	if err != nil {
		log.Errorf("Get unpublished mirrors found error:%v", err)
		return err
	}

	changeFiles, err := GetChangeFiles(*checker.CheckTool.Conf)
	if err != nil {
		log.Errorf("Get change files found error:%v", err)
		return err
	}

	if len(changeFiles) == 0 {
		return nil
	}

	sort.Strings(changeFiles)

	validateInfoList, err := checker.GetValidateInfoList(changeFiles)
	if err != nil {
		log.Errorf("Get validate info list found error:%v", err)
		return err
	}

	if optMirror == "" {
		err = checker.PrefetchCdnDns(c.DefaultCdn)
		if err != nil {
			log.Warningf("Fetch CDN DNS found error:%v", err)
			return err
		}
		checker.TestAllMirrors(mirrors, validateInfoList)
	} else {
		var mirror0 *Mirror
		for _, mirror := range mirrors {
			if mirror.Id == optMirror {
				mirror0 = mirror
				break
			}
		}
		if mirror0 == nil {
			log.Errorf("Not found mirror:%v", optMirror)
			return errors.New("No such mirror named:" + optMirror)
		}
		checker.TestMirror(mirror0.Id, mirror0.GetUrlPrefix(), mirror0.Weight, validateInfoList)
		service.CreateOperation(configs.GetMySQLClient(), model.MirrorOperation{
			CreateDate: time.Now(),
			OperationType: model.SYNC,
			MirrorId: mirror0.Id,
		})

	}
	return nil
}

func (checker *CDNChecker) CheckAllMirrors(c *configs.CdnCheckerConf) error {
	return checker.Init(c)
}

func (checker *CDNChecker) CheckMirror(name string, c *configs.CdnCheckerConf) error {
	optMirror = name
	return checker.Init(c)
}
