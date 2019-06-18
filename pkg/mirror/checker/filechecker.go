package checker

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"mirrors_status/internal/log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

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
	log.Infof("Check file for:%s", url0)
	req, err := http.NewRequest(http.MethodGet, url0, nil)
	if err != nil {
		log.Errorf("Check file found error:%v", err)
		return nil, err
	}
	return checkFileReq(filePath, req, allowRetry, client)
}