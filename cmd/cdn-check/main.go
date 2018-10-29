package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"crypto/tls"
	"encoding/gob"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
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

const currentJsonUrl = "http://packages.deepin.com/deepin/changelist/current.json"

type changeInfo struct {
	Preview string     `json:"preview"` // date type timestamp
	Current string     `json:"current"` // date type timestamp
	Size    uint64     `json:"size"`
	Deleted []fileInfo `json:"deleted"`
	Added   []fileInfo `json:"added"`
}

type fileInfo struct {
	FilePath string `json:"filepath"`
	FileSize string `json:"filesize"`
}

func getChangeInfo() (*changeInfo, error) {
	resp, err := http.Get(currentJsonUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	jDec := json.NewDecoder(resp.Body)
	var v changeInfo
	err = jDec.Decode(&v)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func saveValidateInfoListGob(l []*FileValidateInfo, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	bw := bufio.NewWriter(f)
	enc := gob.NewEncoder(bw)
	err = enc.Encode(l)
	if err != nil {
		return err
	}

	err = bw.Flush()
	if err != nil {
		return err
	}
	return nil
}

func loadValidateInfoListGob(filename string) ([]*FileValidateInfo, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	br := bufio.NewReader(f)
	dec := gob.NewDecoder(br)
	var result []*FileValidateInfo
	err = dec.Decode(&result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func getValidateInfoList(changeInfo *changeInfo) ([]*FileValidateInfo, error) {
	log.Println("current:", changeInfo.Current)
	filename := changeInfo.Current + ".gob"

	validateInfoList, err := loadValidateInfoListGob(filename)
	if err == nil {
		return validateInfoList, nil
	}

	client := getHttpClient(9999)
	for _, added := range changeInfo.Added {
		vi, err := checkFile("http://packages.deepin.com/deepin/", added.FilePath, true, client)
		if err != nil {
			continue
		}
		validateInfoList = append(validateInfoList, vi)
	}

	err = saveValidateInfoListGob(validateInfoList, filename)
	if err != nil {
		return nil, err
	}

	return validateInfoList, nil
}

type testResult struct {
	name           string
	urlPrefix      string
	cdnNodeAddress string
	cdnNodeName    string
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
		filename = fmt.Sprintf("%s-%s-%s.txt", tr.name, tr.cdnNodeAddress,
			tr.cdnNodeName)
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
		fmt.Fprintln(bw, "cdn node name:", tr.cdnNodeName)
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

func (v cdnTestResultSlice) save() error {
	if len(v) == 0 {
		return nil
	}

	err := makeResultDir()
	if err != nil {
		return err
	}

	f, err := os.Create("result/summary")
	if err != nil {
		return err
	}
	defer f.Close()
	bw := bufio.NewWriter(f)

	for _, testResult := range v {
		const summaryFmt = "%s (%s) %.3f%% %d\n"
		log.Printf(summaryFmt, testResult.cdnNodeAddress,
			testResult.cdnNodeName, testResult.percent, testResult.numErrs)
		fmt.Fprintf(bw, summaryFmt, testResult.cdnNodeAddress,
			testResult.cdnNodeName, testResult.percent, testResult.numErrs)
	}

	err = bw.Flush()
	if err != nil {
		return err
	}

	return nil
}

type testRecord struct {
	standard *FileValidateInfo
	result   *FileValidateInfo
	equal    bool
	err      error
}

func testOne(mirrorId string, urlPrefix string, mirrorWeight int,
	validateInfoList []*FileValidateInfo) *testResult {
	log.Printf("start test mirror %q, urlPrefix: %q, weight %d\n",
		mirrorId, urlPrefix, mirrorWeight)

	if urlPrefix == "" {
		return &testResult{
			name: mirrorId,
		}
	}

	pool := grpool.NewPool(6, 1)
	defer pool.Release()
	var mu sync.Mutex
	records := make([]testRecord, 0, len(validateInfoList))
	var good int
	var numErrs int

	client := getHttpClient(mirrorWeight)

	pool.WaitCount(len(validateInfoList))
	for _, validateInfo := range validateInfoList {
		vi := validateInfo
		pool.JobQueue <- func() {
			validateInfo1, err := checkFile(urlPrefix, vi.FilePath, mirrorWeight >= 0, client)

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

func testOneCdn(cdnAddress, cdnName string, validateInfoList []*FileValidateInfo) *testResult {
	pool := grpool.NewPool(3, 1)
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
			}, cdnAddress, client)

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
		cdnNodeAddress: cdnAddress,
		cdnNodeName:    cdnName,
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

	changeInfo, err := getChangeInfo()
	if err != nil {
		log.Fatal(err)
	}

	validateInfoList, err := getValidateInfoList(changeInfo)
	if err != nil {
		log.Fatal(err)
	}

	if optMirror == "" {
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

		testOne(mirror0.Id, mirror0.getUrlPrefix(), mirror0.Weight, validateInfoList)
	}

	//cdnAddresses := map[string]string{
	//	"52.0.26.226":    "美国弗吉尼亚州阿什本  amazon.com",
	//	"221.130.199.56": "中国上海  移动",
	//	"36.110.211.9":   "中国北京  电信",
	//	"1.192.192.70":   "中国河南郑州  电信",
	//	"42.236.10.34":   "中国河南郑州  联通",
	//}
	//
	//pool := grpool.NewPool(len(cdnAddresses), 1)
	//defer pool.Release()
	//
	//var testResults cdnTestResultSlice
	//var testResultsMu sync.Mutex
	//
	//pool.WaitCount(len(cdnAddresses))
	//for cdnAddress, cdnName := range cdnAddresses {
	//	cdnAddressCopy := cdnAddress
	//	cdnNameCopy := cdnName
	//	pool.JobQueue <- func() {
	//		testResult := testOneCdn(cdnAddressCopy, cdnNameCopy, validateInfoList)
	//		testResultsMu.Lock()
	//		testResults = append(testResults, testResult)
	//		testResultsMu.Unlock()
	//		pool.JobDone()
	//	}
	//}
	//pool.WaitAll()
	//
	//sort.Sort(testResults)
	//err = testResults.save()
	//if err != nil {
	//	log.Println("WARN:", err)
	//}
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

	t0 := time.Now()
	finishCount := 0
	var testResults []*testResult
	var mu sync.Mutex

	for _, mirror := range mirrors0 {
		mirrorCopy := mirror
		pool.JobQueue <- func() {
			t1 := time.Now()
			testResult := testOne(mirrorCopy.Id, mirrorCopy.getUrlPrefix(),
				mirrorCopy.Weight, validateInfoList)
			duration0 := time.Since(t0)
			duration1 := time.Since(t1)

			mu.Lock()
			finishCount++
			log.Printf("[%d/%d] finish test for mirror %q, takes %v,"+
				" since the beginning of the test %v",
				finishCount, len(mirrors0), mirrorCopy.Id, duration1, duration0)
			testResults = append(testResults, testResult)
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
	var items []dbTestResultItem
	for _, testResult := range testResults {
		if testResult.urlPrefix == "" {
			continue
		}

		items = append(items, dbTestResultItem{
			Name:     testResult.urlPrefix,
			Progress: testResult.percent / 100.0,
		})
	}
	err = pushMirrorStatus(client, items, time.Now())
	if err != nil {
		log.Fatal(err)
	}
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

func checkFileReq(filePath string, req *http.Request, allowRetry bool,
	client *http.Client) (vi *FileValidateInfo, err error) {
	retryDelay := func() {
		time.Sleep(3 * time.Second)
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
			}

			errMsg := err.Error()
			for _, msg := range allowRetryErrMessages {
				if strings.Contains(errMsg, msg) {
					retryDelay()
					continue loop0
				}
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
