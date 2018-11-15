package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
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

	"github.com/davecgh/go-spew/spew"
	"github.com/ivpusic/grpool"
)

var optMirror string
var optNoTestHidden bool
var optDevEnv bool
var optInfluxdbAddr string

var maxNumOfRetries int

func init() {
	flag.StringVar(&optMirror, "mirror", "", "")
	flag.BoolVar(&optNoTestHidden, "no-hidden", false, "")
	flag.BoolVar(&optDevEnv, "dev-env", false, "")
	flag.StringVar(&optInfluxdbAddr, "influxdb-addr",
		"http://influxdb.trend.deepin.io:10086", "")
}

type changeInfo struct {
	Preview string     `json:"preview"` // date type timestamp
	Current string     `json:"current"` // date type timestamp
	Size    uint64     `json:"size"`
	Deleted []fileInfo `json:"deleted"`
	Added   []fileInfo `json:"added"`
}

func getValidateInfoList(files []string) ([]*FileValidateInfo, error) {
	var validateInfoList []*FileValidateInfo
	var mu sync.Mutex
	client := getHttpClient(9999)
	pool := grpool.NewPool(3, 1)
	defer pool.Release()
	pool.WaitCount(len(files))

	for _, file := range files {
		fileCopy := file
		pool.JobQueue <- func() {
			defer pool.JobDone()

			vi, err := checkFile(baseUrl, fileCopy, true, client)
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

type testResult struct {
	name           string
	urlPrefix      string
	cdnNodeAddress string
	records        []testRecord
	percent        float64
	numErrs        int
}

func (tr *testResult) save() error {
	err := makeResultDir()
	if err != nil {
		return err
	}

	var filename string
	if tr.cdnNodeAddress == "" {
		filename = tr.name + ".txt"
	} else {
		filename = fmt.Sprintf("%s-%s.txt", tr.name, tr.cdnNodeAddress)
	}
	filename = filepath.Join("result", filename)
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	bw := bufio.NewWriter(f)

	fmt.Fprintln(bw, "name:", tr.name)
	fmt.Fprintln(bw, "urlPrefix:", tr.urlPrefix)

	if tr.cdnNodeAddress != "" {
		fmt.Fprintln(bw, "cdn node address:", tr.cdnNodeAddress)
	}
	fmt.Fprintf(bw, "percent: %.3f%%\n", tr.percent)
	if tr.percent == 100 {
		fmt.Fprintln(bw, "sync completed")
	}

	// err
	for _, record := range tr.records {
		if record.err != nil {
			fmt.Fprintln(bw, "has error")
			break
		}
	}
	fmt.Fprintln(bw, "\n# Error:")
	for _, record := range tr.records {
		if record.err == nil {
			continue
		}
		fmt.Fprintln(bw, "file path:", record.standard.FilePath)
		fmt.Fprintln(bw, "standard url:", record.standard.URL)
		fmt.Fprintln(bw, "err:", record.err)
		fmt.Fprintln(bw, "errDump:", spew.Sdump(record.err))
		fmt.Fprintln(bw)
	}

	// not equal
	fmt.Fprintln(bw, "\n# Not Equal:")
	for _, record := range tr.records {
		if record.result == nil || record.equal {
			continue
		}

		// result is not nil and not equal
		fmt.Fprintln(bw, "file path:", record.standard.FilePath)
		fmt.Fprintln(bw, "standard url:", record.standard.URL)
		fmt.Fprintln(bw, "url:", record.result.URL)

		fmt.Fprintln(bw, "standard size:", record.standard.Size)
		fmt.Fprintln(bw, "size:", record.result.Size)

		fmt.Fprintf(bw, "standard md5sum: %x\n", record.standard.MD5Sum)
		fmt.Fprintf(bw, "md5sum: %x\n", record.result.MD5Sum)

		fmt.Fprintln(bw, "standard mod time:", record.standard.ModTime)
		fmt.Fprintln(bw, "mod time:", record.result.ModTime)

		fmt.Fprintln(bw)
	}

	// equal
	fmt.Fprintln(bw, "\n# Equal:")
	for _, record := range tr.records {
		if !record.equal {
			continue
		}

		fmt.Fprintln(bw, "file path:", record.standard.FilePath)
		fmt.Fprintln(bw, "standard url:", record.standard.URL)
		fmt.Fprintln(bw, "url:", record.result.URL)

		fmt.Fprintln(bw, "size:", record.result.Size)
		fmt.Fprintf(bw, "md5sum: %x\n", record.result.MD5Sum)

		fmt.Fprintln(bw, "standard mod time:", record.standard.ModTime)
		fmt.Fprintln(bw, "mod time:", record.result.ModTime)

		fmt.Fprintln(bw)
	}

	err = bw.Flush()
	if err != nil {
		return err
	}

	return nil
}

type cdnTestResultSlice []*testResult

func (v cdnTestResultSlice) Len() int {
	return len(v)
}

func (v cdnTestResultSlice) Less(i, j int) bool {
	return v[i].percent < v[j].percent
}

func (v cdnTestResultSlice) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

type testRecord struct {
	standard *FileValidateInfo
	result   *FileValidateInfo
	equal    bool
	err      error
}

func testMirrorCommon(mirrorId, urlPrefix string, mirrorWeight int,
	validateInfoList []*FileValidateInfo) *testResult {
	if urlPrefix == "" {
		return &testResult{
			name: mirrorId,
		}
	}

	pool := grpool.NewPool(6, 1)
	defer pool.Release()
	var mu sync.Mutex
	numTotal := len(validateInfoList)
	records := make([]testRecord, 0, numTotal)
	var good int
	var numErrs int
	var numCompleted int

	client := getHttpClient(mirrorWeight)

	pool.WaitCount(numTotal)

	for _, validateInfo := range validateInfoList {
		vi := validateInfo
		pool.JobQueue <- func() {
			validateInfo1, err := checkFile(urlPrefix, vi.FilePath, mirrorWeight >= 0, client)

			var record testRecord
			record.standard = vi
			mu.Lock()
			numCompleted++
			log.Printf("%s %s [%d/%d]\n", getMirrorsTestProgressDesc(),
				mirrorId, numCompleted, numTotal)
			if err != nil {
				numErrs++
				log.Println("WARN:", err)
				record.err = err

			} else {
				record.result = validateInfo1
				if vi.equal(validateInfo1) {
					good++
					record.equal = true
				}
			}
			records = append(records, record)

			mu.Unlock()
			pool.JobDone()
		}
	}
	pool.WaitAll()
	percent := float64(good) / float64(len(validateInfoList)) * 100.0

	r := &testResult{
		name:      mirrorId,
		urlPrefix: urlPrefix,
		records:   records,
		percent:   percent,
		numErrs:   numErrs,
	}

	err := r.save()
	if err != nil {
		log.Println("WARN:", err)
	}

	return r
}

var dnsCache = make(map[string][]string)

func prefetchCdnDns(host string) error {
	ips, ok := dnsCache[host]
	if ok {
		return nil
	}

	var err error
	ips, err = testDNS(host)
	if err != nil {
		return err
	}

	dnsCache[host] = ips
	return nil
}

func getCdnDns(host string) []string {
	ips, ok := dnsCache[host]
	if !ok {
		if host == "cdn.packages.deepin.com" {
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

func testMirrorCdn(mirrorId, urlPrefix string,
	validateInfoList []*FileValidateInfo) []*testResult {
	u, err := url.Parse(urlPrefix)
	if err != nil {
		panic(err)
	}

	ips := getCdnDns(u.Hostname())
	log.Printf("testMirrorCdn mirrorId: %s, ips: %v\n", mirrorId, ips)

	if len(ips) == 0 {
		return []*testResult{
			{
				name: mirrorId,
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
			testResult := testCdnNode(mirrorId, urlPrefix, cdnAddressCopy, validateInfoList)
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

func testMirror(mirrorId string, urlPrefix string, mirrorWeight int,
	validateInfoList []*FileValidateInfo) []*testResult {
	log.Printf("start test mirror %q, urlPrefix: %q, weight %d\n",
		mirrorId, urlPrefix, mirrorWeight)

	if mirrorId == "default" {
		// is cdn
		return testMirrorCdn(mirrorId, urlPrefix, validateInfoList)
	}
	r := testMirrorCommon(mirrorId, urlPrefix, mirrorWeight, validateInfoList)
	return []*testResult{r}
}

func testCdnNode(mirrorId, urlPrefix, cdnNodeAddress string, validateInfoList []*FileValidateInfo) *testResult {
	pool := grpool.NewPool(6, 1)
	defer pool.Release()
	var mu sync.Mutex
	records := make([]testRecord, 0, len(validateInfoList))
	var good int
	var numErrs int

	client := getHttpClient(1000)

	pool.WaitCount(len(validateInfoList))
	for _, validateInfo := range validateInfoList {
		vi := validateInfo
		pool.JobQueue <- func() {
			validateInfo1, err := checkFileCdn(fileInfo{
				FilePath: vi.FilePath,
			}, cdnNodeAddress, client)

			var record testRecord
			record.standard = vi
			mu.Lock()
			if err != nil {
				numErrs++
				log.Println("WARN:", err)
				record.err = err

			} else {
				record.result = validateInfo1
				if vi.equal(validateInfo1) {
					good++
					record.equal = true
				}
			}
			records = append(records, record)

			mu.Unlock()
			pool.JobDone()
		}
	}

	pool.WaitAll()
	percent := float64(good) / float64(len(validateInfoList)) * 100.0

	r := &testResult{
		name:           mirrorId,
		urlPrefix:      urlPrefix,
		cdnNodeAddress: cdnNodeAddress,
		records:        records,
		percent:        percent,
		numErrs:        numErrs,
	}

	err := r.save()
	if err != nil {
		log.Println("WARN:", err)
	}

	return r
}

func makeResultDir() error {
	err := os.Mkdir("result", 0755)
	if err != nil && !os.IsExist(err) {
		return err
	}
	return nil
}

var clientNormal *http.Client
var clientHidden *http.Client

func getHttpClient(mirrorWeight int) *http.Client {
	if mirrorWeight >= 0 {
		return clientNormal
	}
	return clientHidden
}

func main() {
	flag.Parse()
	log.SetFlags(log.Lshortfile)

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

	mirrorsUrl := "http://server-12:8900/v1/mirrors"
	mirrors, err := getUnpublishedMirrors(mirrorsUrl)
	if err != nil {
		log.Fatal(err)
	}

	changeFiles, err := getChangeFiles()
	if err != nil {
		log.Fatal(err)
	}

	if len(changeFiles) == 0 {
		return
	}

	sort.Strings(changeFiles)
	saveChangeFiles(changeFiles)

	validateInfoList, err := getValidateInfoList(changeFiles)
	if err != nil {
		log.Fatal(err)
	}

	rand.Seed(time.Now().UnixNano())

	if optMirror == "" {
		err = prefetchCdnDns("cdn.packages.deepin.com")
		if err != nil {
			log.Println("WARN:", err)
		}
		testAllMirrors(mirrors, validateInfoList)
	} else {
		var mirror0 *mirror
		for _, mirror := range mirrors {
			if mirror.Id == optMirror {
				mirror0 = mirror
				break
			}
		}
		if mirror0 == nil {
			log.Fatal("not found mirror " + optMirror)
		}

		testMirror(mirror0.Id, mirror0.getUrlPrefix(), mirror0.Weight, validateInfoList)
	}

}

var numMirrorsTotal int
var numMirrorsFinished int
var numMirrorsMu sync.Mutex

func getMirrorsTestProgressDesc() string {
	numMirrorsMu.Lock()
	str := fmt.Sprintf("[%d/%d]", numMirrorsFinished, numMirrorsTotal)
	numMirrorsMu.Unlock()
	return str
}

func testMirrorFinish() {
	numMirrorsMu.Lock()
	numMirrorsFinished++
	numMirrorsMu.Unlock()
}

func testAllMirrors(mirrors0 mirrors, validateInfoList []*FileValidateInfo) {
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
	var testResults []*testResult
	var mu sync.Mutex

	for _, mirror := range mirrors0 {
		mirrorCopy := mirror
		pool.JobQueue <- func() {
			t1 := time.Now()
			testResult := testMirror(mirrorCopy.Id, mirrorCopy.getUrlPrefix(),
				mirrorCopy.Weight, validateInfoList)
			testMirrorFinish()
			duration0 := time.Since(t0)
			duration1 := time.Since(t1)

			log.Printf("%s finish test for mirror %q, takes %v,"+
				" since the beginning of the test %v",
				getMirrorsTestProgressDesc(), mirrorCopy.Id, duration1, duration0)
			mu.Lock()
			testResults = append(testResults, testResult...)
			mu.Unlock()
			pool.JobDone()
		}
	}
	pool.WaitAll()

	pushAllMirrorsTestResults(testResults)
}

func pushAllMirrorsTestResults(testResults []*testResult) {
	dbUser := os.Getenv("INFLUX_USER")
	if dbUser == "" {
		log.Fatal("no set env INFLUX_USER")
	}

	dbPassword := os.Getenv("INFLUX_PASSWD")
	if dbPassword == "" {
		log.Fatal("no set env INFLUX_PASSWD")
	}

	dbName := "mirror_status"
	client, err := NewInfluxClient(optInfluxdbAddr, dbUser, dbPassword, dbName)
	if err != nil {
		log.Fatal(err)
	}
	var mirrorsPoints []mirrorsPoint
	var mirrorsCdnPoints []mirrorsCdnPoint

	var mirrorsPointsAppendedMap = make(map[string]struct{})
	for _, testResult := range testResults {
		if testResult.cdnNodeAddress == "" {
			if testResult.urlPrefix != "" {
				mirrorsPoints = append(mirrorsPoints, mirrorsPoint{
					Name:     testResult.urlPrefix,
					Progress: testResult.percent / 100.0,
				})
			}
		} else {
			if _, ok := mirrorsPointsAppendedMap[testResult.name]; !ok {
				mirrorsPoints = append(mirrorsPoints, mirrorsPoint{
					Name:     testResult.urlPrefix,
					Progress: testResult.percent / 100.0,
				})
				mirrorsPointsAppendedMap[testResult.name] = struct{}{}
			}

			mirrorsCdnPoints = append(mirrorsCdnPoints, mirrorsCdnPoint{
				MirrorId:   testResult.name,
				NodeIpAddr: testResult.cdnNodeAddress,
				Progress:   testResult.percent / 100.0,
			})
		}
	}
	now := time.Now()
	err = pushToMirrors(client, mirrorsPoints, now)
	if err != nil {
		log.Fatal(err)
	}

	err = pushToMirrorsCdn(client, mirrorsCdnPoints, now)
}

func checkFile(urlPrefix string, filePath string, allowRetry bool,
	client *http.Client) (*FileValidateInfo, error) {
	if !strings.HasSuffix(urlPrefix, "/") {
		urlPrefix += "/"
	}

	url0 := urlPrefix + filePath
	log.Println("checkFile:", url0)
	req, err := http.NewRequest(http.MethodGet, url0, nil)
	if err != nil {
		log.Println("WARN:", err)
		return nil, err
	}
	return checkFileReq(filePath, req, allowRetry, client)
}

func checkFileCdn(fileInfo fileInfo, cdnIp string, client *http.Client) (*FileValidateInfo, error) {
	url0 := "http://" + cdnIp + "/deepin/" + fileInfo.FilePath
	log.Println("checkFileCdn:", url0)
	req, err := http.NewRequest(http.MethodGet, url0, nil)
	if err != nil {
		log.Println("WARN:", err)
		return nil, err
	}
	req.Host = "cdn.packages.deepin.com"
	vi, err := checkFileReq(fileInfo.FilePath, req, true, client)
	return vi, err
}

func parseContentRange(str string) (posBegin, posEnd, total int, err error) {
	_, err = fmt.Sscanf(str, "bytes %d-%d/%d", &posBegin, &posEnd, &total)
	if err != nil {
		err = fmt.Errorf("parseContentRange: %q error %s", str, err.Error())
	}
	return
}

type FileValidateInfo struct {
	FilePath string
	MD5Sum   []byte
	Size     int
	ModTime  string
	URL      string
}

func (vi *FileValidateInfo) equal(other *FileValidateInfo) bool {
	return vi.FilePath == other.FilePath &&
		vi.Size == other.Size &&
		bytes.Equal(vi.MD5Sum, other.MD5Sum)
}

var regErrLookupTimeout = regexp.MustCompile("lookup.*on.*read udp.*i/o timeout")

var dialTcpTimeoutMap = make(map[string]int)
var dialTcpTimeoutMapMu sync.Mutex

var regDialTcpTimeout = regexp.MustCompile(`dial tcp (\S+): i/o timeout`)

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
			log.Println("retry", i, req.URL)
		}

		vi, err = checkFileReq0(filePath, req, client)

		if err != nil {
			log.Printf("WARN: url: %s, err: %v\n", req.URL, err)
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
			log.Println("retry success", i, req.URL)
		}
		return
	}
	log.Println("maximum retry times exceeded", req.URL)
	return
}

func checkFileReq0(filePath string, req *http.Request, client *http.Client) (*FileValidateInfo, error) {
	size := 4 * 1024
	// 第一次请求
	req.Header.Set("Range", "bytes=0-"+strconv.Itoa(size-1))
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
	posStart, postEnd, total, err := parseContentRange(contentRange)
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
	posStart2, postEnd2, total2, err2 := parseContentRange(contentRange)
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
