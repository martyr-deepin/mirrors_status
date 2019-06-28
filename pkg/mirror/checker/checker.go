package checker

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/influxdata/influxdb/uuid"
	"github.com/ivpusic/grpool"
	llog "log"
	"math/rand"
	configs "mirrors_status/internal/config"
	"mirrors_status/internal/log"
	"mirrors_status/pkg/model/constants"
	"mirrors_status/pkg/model/mirror"
	"mirrors_status/pkg/model/operation"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
	"time"
)

type CDNChecker struct {
	CheckTool CheckTool
}

func NewCDNChecker() *CDNChecker {
	return &CDNChecker{
		CheckTool: NewCheckTool(),
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

func (checker CDNChecker) getValidateInfoList(files []string) ([]*FileValidateInfo, error) {
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

func (checker *CDNChecker) prefetchCdnDns(host string) error {
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

func (checker *CDNChecker) getCdnDns(host string) []string {
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

func (checker *CDNChecker) checkFileCdn(fileInfo FileInfo, cdnIp string, client *http.Client) (*FileValidateInfo, error) {
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


func (checker *CDNChecker) testCdnNode(mirrorId, urlPrefix, cdnNodeAddress string, validateInfoList []*FileValidateInfo) *TestResult {
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
			validateInfo1, err := checker.checkFileCdn(FileInfo{
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

func (checker *CDNChecker) testMirrorCdn(mirrorId, urlPrefix string,
	validateInfoList []*FileValidateInfo) []*TestResult {
	u, err := url.Parse(urlPrefix)
	if err != nil {
		log.Errorf("URL prefix: %v", urlPrefix)
		//panic(err)
	}

	ips := checker.getCdnDns(u.Hostname())
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
			testResult := checker.testCdnNode(mirrorId, urlPrefix, cdnAddressCopy, validateInfoList)
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

func (checker *CDNChecker) testMirror(mirrorId string, urlPrefix string, mirrorWeight int,
	validateInfoList []*FileValidateInfo) []*TestResult {
	log.Infof("Testing mirror:%q, Prefix:%q, Weight:%d", mirrorId, urlPrefix, mirrorWeight)
	if mirrorId == "default" {
		// is cdn
		return checker.testMirrorCdn(mirrorId, urlPrefix, validateInfoList)
	}
	r := TestMirrorCommon(mirrorId, urlPrefix, mirrorWeight, validateInfoList)
	return []*TestResult{r}
}

func TestMirrorFinish() {
	numMirrorsMu.Lock()
	numMirrorsFinished++
	numMirrorsMu.Unlock()
}

func (checker *CDNChecker) pushAllMirrorsTestResults(testResults []*TestResult) error {
	var mirrorsPoints []mirror.MirrorsPoint
	var mirrorsCdnPoints []mirror.MirrorsCdnPoint

	var mirrorsPointsAppendedMap = make(map[string]struct{})
	for _, testResult := range testResults {
		if testResult.CdnNodeAddress == "" {
			if testResult.UrlPrefix != "" {
				mirrorsPoints = append(mirrorsPoints, mirror.MirrorsPoint{
					Name:     testResult.UrlPrefix,
					Progress: testResult.Percent / 100.0,
				})
			}
		} else {
			if _, ok := mirrorsPointsAppendedMap[testResult.Name]; !ok {
				mirrorsPoints = append(mirrorsPoints, mirror.MirrorsPoint{
					Name:     testResult.UrlPrefix,
					Progress: testResult.Percent / 100.0,
				})
				mirrorsPointsAppendedMap[testResult.Name] = struct{}{}
			}

			mirrorsCdnPoints = append(mirrorsCdnPoints, mirror.MirrorsCdnPoint{
				MirrorId:   testResult.Name,
				NodeIpAddr: testResult.CdnNodeAddress,
				Progress:   testResult.Percent / 100.0,
			})
		}
	}
	err := mirror.PushMirrors(mirrorsPoints)
	if err != nil {
		log.Errorf("Push to mirrors found error:%v", err)
		return err
	}

	err = mirror.PushMirrorsCdn(mirrorsCdnPoints)
	if err != nil {
		log.Errorf("Push to mirrors_cdn found error:%v", err)
		return err
	}
	return nil
}

func (checker *CDNChecker) testAllMirrors(mirrors0 mirrors, validateInfoList []*FileValidateInfo, username string) {
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
			results := []*TestResult{}
			if mirror.UrlHttps != "" {
				results = append(results, checker.testMirror(mirrorCopy.Id, fmt.Sprintf("https://%s", mirror.UrlHttps),
					mirrorCopy.Weight, validateInfoList)...)
			}
			if mirror.UrlHttp != "" {
				results = append(results, checker.testMirror(mirrorCopy.Id, fmt.Sprintf("http://%s", mirror.UrlHttp),
					mirrorCopy.Weight, validateInfoList)...)
			}
			//if mirror.UrlFtp != "" {
			//	results = append(results, checker.testMirror(mirrorCopy.Id, fmt.Sprintf("ftp://%s", mirror.UrlFtp),
			//		mirrorCopy.Weight, validateInfoList)...)
			//}
			//if mirror.UrlRsync != "" {
			//	results = append(results, checker.testMirror(mirrorCopy.Id, fmt.Sprintf("rsync://%s", mirror.UrlRsync),
			//		mirrorCopy.Weight, validateInfoList)...)
			//}
			TestMirrorFinish()
			duration0 := time.Since(t0)
			duration1 := time.Since(t1)

			log.Infof("%s test finished for mirror:%q. Duration:%v, Since:%v", GetMirrorsTestProgressDesc(), mirrorCopy.Id, duration1, duration0)
			mu.Lock()
			testResults = append(testResults, results...)
			mu.Unlock()
			pool.JobDone()
		}
	}
	pool.WaitAll()

	_ = checker.pushAllMirrorsTestResults(testResults)
}

func (checker *CDNChecker) init(username string, index string) error {
	checker.CheckTool = CheckTool{
		Conf: configs.NewServerConfig().CdnChecker,
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
	mirrors, err := GetUnpublishedMirrors(configs.NewServerConfig().CdnChecker.Target)
	if err != nil {
		log.Errorf("Get unpublished mirrors found error:%v", err)
		_ = operation.UpdateMirrorStatus(index, constants.STATUS_FAILURE, err.Error())
		return err
	}

	changeFiles, err := GetChangeFiles(*checker.CheckTool.Conf)
	if err != nil {
		log.Errorf("Get change files found error:%v", err)
		_ = operation.UpdateMirrorStatus(index, constants.STATUS_FAILURE, err.Error())
		return err
	}

	if len(changeFiles) == 0 {
		_ = operation.UpdateMirrorStatus(index, constants.STATUS_FINISHED, "")
		return nil
	}

	sort.Strings(changeFiles)

	validateInfoList, err := checker.getValidateInfoList(changeFiles)
	if err != nil {
		log.Errorf("Get validate info list found error:%v", err)
		_ = operation.UpdateMirrorStatus(index, constants.STATUS_FAILURE, err.Error())
		return err
	}

	if optMirror == "" {
		err = checker.prefetchCdnDns(configs.NewServerConfig().CdnChecker.DefaultCdn)
		if err != nil {
			log.Warningf("Fetch CDN DNS found error:%v", err)
			_ = operation.UpdateMirrorStatus(index, constants.STATUS_FAILURE, err.Error())
			return err
		}
		err = operation.UpdateMirrorStatus(index, constants.STATUS_RUNNING, "")
		if err != nil {
			log.Error("Update mirror operation status found error:%#v", err)
		}
		checker.testAllMirrors(mirrors, validateInfoList, username)
	} else {
		var mirror0 *Mirror
		for _, mirror := range mirrors {
			if mirror.Id == optMirror {
				mirror0 = mirror
				break
			}
		}
		if mirror0 == nil {
			_ = operation.UpdateMirrorStatus(index, constants.STATUS_FAILURE, "No such mirror " + optMirror)
			return errors.New("No such mirror name: " + optMirror)
		}
		err = operation.UpdateMirrorStatus(index, constants.STATUS_RUNNING, "")
		if err != nil {
			log.Error("Update mirror operation status found error:%#v", err)
		}
		results := []*TestResult{}
		if mirror0.UrlHttps != "" {
			results = append(results, checker.testMirror(mirror0.Id, fmt.Sprintf("https://%s", mirror0.UrlHttps),
				mirror0.Weight, validateInfoList)...)
		}
		if mirror0.UrlHttp != "" {
			results = append(results, checker.testMirror(mirror0.Id, fmt.Sprintf("http://%s", mirror0.UrlHttp),
				mirror0.Weight, validateInfoList)...)
		}
		//if mirror0.UrlFtp != "" {
		//	results = append(results, checker.testMirror(mirror0.Id, fmt.Sprintf("ftp://%s", mirror0.UrlFtp),
		//		mirror0.Weight, validateInfoList)...)
		//}
		//if mirror0.UrlRsync != "" {
		//	results = append(results, checker.testMirror(mirror0.Id, fmt.Sprintf("rsync://%s", mirror0.UrlRsync),
		//		mirror0.Weight, validateInfoList)...)
		//}
		_ = checker.pushAllMirrorsTestResults(results)
	}
	return nil
}

func (checker *CDNChecker) CheckAllMirrors(username string) string {
	index := uuid.TimeUUID().String()
	err := operation.MirrorOperation{
		Index:         index,
		CreateDate:    time.Now(),
		Username:      username,
		OperationType: constants.SYNC_ALL,
		MirrorId:      "ALL",
		Status:        constants.STATUS_WAITING,
	}.CreateMirrorOperation()
	if err != nil {
		log.Error("Create mirror operation found error:%#v", err)
	}
	go func() {
		err := checker.init(username, index)
		if err != nil {
			err = operation.UpdateMirrorStatus(index, constants.STATUS_FAILURE, err.Error())
			if err != nil {
				log.Error("Update mirror operation status found error:%#v", err)
			}
		} else {
			err = operation.UpdateMirrorStatus(index, constants.STATUS_FINISHED, "")
			if err != nil {
				log.Error("Update mirror operation status found error:%#v", err)
			}
		}
	}()
	return index
}

func (checker *CDNChecker) CheckMirror(name, username string) string {
	optMirror = name
	index := uuid.TimeUUID().String()
	err := operation.MirrorOperation{
		Index:         index,
		CreateDate:    time.Now(),
		Username:      username,
		OperationType: constants.SYNC,
		MirrorId:      name,
		Status:        constants.STATUS_WAITING,
	}.CreateMirrorOperation()
	if err != nil {
		log.Error("Create mirror operation found error:%#v", err)
	}
	go func() {
		err := checker.init(username, index)
		if err != nil {
			err = operation.UpdateMirrorStatus(index, constants.STATUS_FAILURE, err.Error())
			if err != nil {
				log.Error("Update mirror operation status found error:%#v", err)
			}
		} else {
			err = operation.UpdateMirrorStatus(index, constants.STATUS_FINISHED, "")
			if err != nil {
				log.Error("Update mirror operation status found error:%#v", err)
			}
		}
	}()
	return index
}

func (checker *CDNChecker) CheckMirrors(mirrors []mirror.Mirror, username string) string {
	index := uuid.TimeUUID().String()
	err := operation.MirrorOperation{
		Index:         index,
		CreateDate:    time.Now(),
		Username:      username,
		OperationType: constants.SYNC_UPSTREAM,
		Status:        constants.STATUS_WAITING,
		Total: len(mirrors),
	}.CreateMirrorOperation()
	if err != nil {
		log.Error("Create mirror operation found error:%#v", err)
	}
	if len(mirrors) <= 0 {
		_ = operation.UpdateMirrorStatus(index, constants.STATUS_FINISHED, "")
		_ = operation.SyncMirrorFinishOnce(index)
		return index
	}
	go func() {
		for _, mirror := range mirrors {
			optMirror = mirror.Id
			log.Infof("Before init mirror check by upstream")
			err := checker.init(username, index)
			if err != nil && !mirror.IsKey {
				log.Errorf("Sync mirror:[%s] found error:%v", mirror.Id, err)
				_ = operation.SyncMirrorFailedOnce(index)
				break
			}
			_ = mirror.GetMirrorCompletion()
			_ = mirror.GetMirrorCdnCompletion()
			if mirror.IsKey {
				if (mirror.HttpProgress < 1 && mirror.UrlHttp != "") ||
					(mirror.HttpsProgress < 1 && mirror.UrlHttps != "") ||
					(mirror.FtpProgress < 1 && mirror.UrlFtp != "") ||
					(mirror.RsyncProgress < 1 && mirror.UrlRsync != "") {
					_ = operation.SyncMirrorUnfinishOnce(index)
					_ = operation.UpdateMirrorStatus(index, constants.STATUS_FAILURE, "关键镜像" + mirror.Id + "同步未完成")
					break
				}
				for _, comp := range mirror.CdnCompletion {
					if comp.Completion < 1 {
						_ = operation.SyncMirrorUnfinishOnce(index)
						_ = operation.UpdateMirrorStatus(index, constants.STATUS_FAILURE, "关键镜像站" + mirror.Id + "的CDN同步未完成")
						break
					}
				}

			}
			_ = operation.SyncMirrorFinishOnce(index)
		}
		op, _ := operation.GetOperationByIndex(index)
		if (op.Total <= op.Failed + op.Finish) && op.Failed <= 0 && op.Unfinish <= 0 {
			err = operation.UpdateMirrorStatus(index, constants.STATUS_FINISHED, "")
			if err != nil {
				log.Error("Update mirror operation status found error:%#v", err)
			}
		}
	}()
	return index
}

func (checker *CDNChecker) CheckMirrorsByUpstreamWithIndex(upstream, username, index string) (err error) {
	mirrors, err := mirror.GetMirrorsByUpstream(upstream)
	if err != nil {
		return err
	}
	if len(mirrors) <= 0 {
		_ = operation.UpdateMirrorStatus(index, constants.STATUS_FINISHED, "")
		_ = operation.SyncMirrorFinishOnce(index)
		return nil
	}
	go func() {
		for _, mirror := range mirrors {
			optMirror = mirror.Id
			log.Infof("Before init mirror check by upstream")
			err := checker.init(username, index)
			if err != nil && !mirror.IsKey {
				log.Errorf("Sync mirror:[%s] found error:%v", mirror.Id, err)
				_ = operation.SyncMirrorFailedOnce(index)
				break
			}
			_ = mirror.GetMirrorCompletion()
			if mirror.IsKey {
				if (mirror.HttpProgress < 1 && mirror.UrlHttp != "") ||
					(mirror.HttpsProgress < 1 && mirror.UrlHttps != "") ||
					(mirror.FtpProgress < 1 && mirror.UrlFtp != "") ||
					(mirror.RsyncProgress < 1 && mirror.UrlRsync != "") {
					_ = operation.SyncMirrorUnfinishOnce(index)
					_ = operation.UpdateMirrorStatus(index, constants.STATUS_FAILURE, "关键镜像" + mirror.Id + " 同步未完成")
					break
				}

			}
			_ = operation.SyncMirrorFinishOnce(index)
		}
		op, _ := operation.GetOperationByIndex(index)
		if (op.Total <= op.Failed + op.Finish) && op.Failed <= 0 && op.Unfinish <= 0 {
			err = operation.UpdateMirrorStatus(index, constants.STATUS_FINISHED, "")
			if err != nil {
				log.Error("Update mirror operation status found error:%#v", err)
			}
		}
	}()
	return nil
}