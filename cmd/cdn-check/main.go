package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/ivpusic/grpool"
)

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

	for _, added := range changeInfo.Added {
		vi, err := checkFile(added)
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

type cdnTestResult struct {
	cdnAddress string
	cdnName    string
	records    []cdnTestRecord
	percent    float64
	numErrs    int
}

func (tr cdnTestResult) save() error {
	err := makeResultDir()
	if err != nil {
		return err
	}

	filename := filepath.Join("result",
		fmt.Sprintf("%s %s.txt", tr.cdnAddress, tr.cdnName))
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	bw := bufio.NewWriter(f)

	fmt.Fprintln(bw, "ip:", tr.cdnAddress)
	fmt.Fprintln(bw, "name:", tr.cdnName)
	fmt.Fprintf(bw, "percent: %.3f%%\n", tr.percent)

	// err
	fmt.Fprintln(bw, "\n# Error:")
	for _, record := range tr.records {
		if record.err == nil {
			continue
		}
		fmt.Fprintln(bw, "file path:", record.standard.FilePath)
		fmt.Fprintln(bw, "standard url:", record.standard.URL)
		fmt.Fprintln(bw, "err:", record.err)
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

type cdnTestResultSlice []*cdnTestResult

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
		log.Printf(summaryFmt, testResult.cdnAddress,
			testResult.cdnName, testResult.percent, testResult.numErrs)
		fmt.Fprintf(bw, summaryFmt, testResult.cdnAddress,
			testResult.cdnName, testResult.percent, testResult.numErrs)
	}

	err = bw.Flush()
	if err != nil {
		return err
	}

	return nil
}

type cdnTestRecord struct {
	standard *FileValidateInfo
	result   *FileValidateInfo
	equal    bool
	err      error
}

func testOneCdn(cdnAddress, cdnName string, validateInfoList []*FileValidateInfo) *cdnTestResult {
	pool := grpool.NewPool(3, 1)
	defer pool.Release()
	var mu sync.Mutex
	records := make([]cdnTestRecord, 0, len(validateInfoList))
	var good int
	var numErrs int

	pool.WaitCount(len(validateInfoList))
	for _, validateInfo := range validateInfoList {
		vi := validateInfo
		pool.JobQueue <- func() {
			validateInfo1, err := checkFileCdn(fileInfo{
				FilePath: vi.FilePath,
			}, cdnAddress)

			var record cdnTestRecord
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

	r := &cdnTestResult{
		cdnAddress: cdnAddress,
		cdnName:    cdnName,
		records:    records,
		percent:    percent,
		numErrs:    numErrs,
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

func main() {
	log.SetFlags(log.Lshortfile)

	http.DefaultClient.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   360 * time.Second,
			KeepAlive: 360 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	http.DefaultClient.Timeout = 10 * time.Minute
	changeInfo, err := getChangeInfo()
	if err != nil {
		log.Fatal(err)
	}

	validateInfoList, err := getValidateInfoList(changeInfo)
	if err != nil {
		log.Fatal(err)
	}

	cdnAddresses := map[string]string{
		"36.110.211.9":   "中国北京  电信",
		"52.0.26.226":    "美国弗吉尼亚州阿什本  amazon.com",
		"221.130.199.56": "中国上海  移动",
		"42.236.10.34":   "中国河南郑州  联通",
		"1.192.192.70":   "中国河南郑州  电信",
		"60.28.115.30":   "中国天津  联通",
	}

	pool := grpool.NewPool(len(cdnAddresses), 1)
	defer pool.Release()

	var testResults cdnTestResultSlice
	var testResultsMu sync.Mutex

	pool.WaitCount(len(cdnAddresses))
	for cdnAddress, cdnName := range cdnAddresses {
		cdnAddressCopy := cdnAddress
		cdnNameCopy := cdnName
		pool.JobQueue <- func() {
			testResult := testOneCdn(cdnAddressCopy, cdnNameCopy, validateInfoList)
			testResultsMu.Lock()
			testResults = append(testResults, testResult)
			testResultsMu.Unlock()
			pool.JobDone()
		}
	}
	pool.WaitAll()

	sort.Sort(testResults)
	err = testResults.save()
	if err != nil {
		log.Println("WARN:", err)
	}
}

func checkFile(fileInfo fileInfo) (*FileValidateInfo, error) {
	//url0 := "http://pools.corp.deepin.com/deepin/" + fileInfo.FilePath
	//url0 := "http://server-04/deepin/" + fileInfo.FilePath
	url0 := "http://packages.deepin.com/deepin/" + fileInfo.FilePath
	log.Println("checkFile", url0)

	req, err := http.NewRequest(http.MethodGet, url0, nil)
	if err != nil {
		log.Println("WARN:", err)
		return nil, err
	}
	return checkFileReq(fileInfo.FilePath, req)
}

func checkFileCdn(fileInfo fileInfo, cdnIp string) (*FileValidateInfo, error) {
	url0 := "http://" + cdnIp + "/deepin/" + fileInfo.FilePath
	log.Println("checkFileCdn:", url0)
	req, err := http.NewRequest(http.MethodGet, url0, nil)
	if err != nil {
		log.Println("WARN:", err)
		return nil, err
	}
	req.Host = "cdn.packages.deepin.com"
	vi, err := checkFileReq(fileInfo.FilePath, req)
	//if err != nil {
	//	log.Printf("checkFileReq error %#v\n", err)
	//}
	return vi, err
}

func parseContentRange(str string) (posBegin, posEnd, total int, err error) {
	_, err = fmt.Sscanf(str, "bytes %d-%d/%d", &posBegin, &posEnd, &total)
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

func checkFileReq(filePath string, req *http.Request) (*FileValidateInfo, error) {
	size := 4 * 1024
	// 第一次请求
	req.Header.Set("Range", "bytes=0-"+strconv.Itoa(size-1))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("WARN:", err)
		return nil, err
	}

	defer resp.Body.Close()

	modTime := resp.Header.Get("Last-Modified")
	contentRange := resp.Header.Get("Content-Range")
	posStart, postEnd, total, err := parseContentRange(contentRange)
	if err != nil {
		log.Println("WARN:", err)
		return nil, err
	}

	if posStart != 0 {
		log.Println("WARN: posStart != 0")
		return nil, errors.New("posStart != 0")
	}

	buf := make([]byte, size)
	n, err := io.ReadFull(resp.Body, buf)
	if err == io.ErrUnexpectedEOF {
		if n != postEnd+1 {
			log.Println("WARN: unexpectedEOF")
			return nil, err
		}
	} else if err != nil {
		log.Println("WARN:", err)
		return nil, err
	}

	md5hash := md5.New()
	_, err = md5hash.Write(buf[:n])
	if err != nil {
		log.Println("WARN:", err)
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
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		log.Println("WARN:", err)
		return nil, err
	}

	defer resp.Body.Close()

	modTime2 := resp.Header.Get("Last-Modified")
	if modTime != modTime2 {
		return nil, errors.New("mod time changed")
	}

	contentRange = resp.Header.Get("Content-Range")
	posStart2, postEnd2, total2, err2 := parseContentRange(contentRange)
	if err2 != nil {
		log.Println("WARN:", err2)
		return nil, err2
	}
	if posStart2 != secondPosBegin {
		log.Println("WARN: 2rd req posStart not match")
		return nil, errors.New("2rd req posStart not match")
	}
	if postEnd2 != total-1 {
		log.Println("WARN: 2rd req posEnd not match")
		return nil, errors.New("2rd req posEnd not match")
	}
	if total != total2 {
		log.Println("WARN: total not match")
		return nil, errors.New("total not match")
	}

	buf2Size := total - 1 - secondPosBegin + 1
	if n+buf2Size > total {
		panic("assert failed: n + buf2size <= total")
	}

	buf2 := buf[:buf2Size]

	_, err = io.ReadFull(resp.Body, buf2)
	if err != nil {
		log.Println("WARN:", err)
		return nil, err
	}

	_, err = md5hash.Write(buf2)
	if err != nil {
		log.Println("WARN:", err)
		return nil, err
	}

	vi.MD5Sum = md5hash.Sum(nil)
	return vi, nil
}
