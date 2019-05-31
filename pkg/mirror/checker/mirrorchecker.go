package checker

import (
	"fmt"
	"github.com/ivpusic/grpool"
	"mirrors_status/internal/log"
	"sync"
)

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