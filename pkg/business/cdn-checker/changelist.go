package cdn_checker

import (
	"encoding/json"
	"github.com/PuerkitoBio/goquery"
	"log"
	"math/rand"
	"mirrors_status/pkg/config"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

type changeMetaInfoSlice []ChangeMetaInfo

func (v changeMetaInfoSlice) Len() int {
	return len(v)
}

func (v changeMetaInfoSlice) Less(i, j int) bool {
	t1 := v[i].t
	t2 := v[j].t
	return t1.Sub(t2) < 0
}

func (v changeMetaInfoSlice) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func getChangeInfo(conf configs.CdnCheckerConf, name string) (*ChangeInfo, error) {
	u := conf.SourceUrl + conf.SourcePath + name
	//u := changeListUrl + name
	log.Println("getChangeInfo u:", u)
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	jDec := json.NewDecoder(resp.Body)
	var v ChangeInfo
	err = jDec.Decode(&v)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func getChangeList(conf configs.CdnCheckerConf) ([]string, error) {
	resp, err := http.Get(conf.SourceUrl + conf.SourcePath)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var result []string
	doc.Find("a").Each(func(i int, selection *goquery.Selection) {
		href, ok := selection.Attr("href")
		if !ok {
			return
		}
		if strings.HasSuffix(href, ".json") && href != "current.json" {
			result = append(result, href)
		}
	})
	return result, nil
}

func ignoreFile(filename string) bool {
	if strings.Contains(filename, "__GUARD__") ||
		strings.Contains(filename, "/Sources.diff/") ||
		strings.Contains(filename, "/Packages.diff/") {
		return true
	} else if strings.HasPrefix(filename, "pool/") {
		// in pool dir
		if !strings.HasSuffix(filename, ".deb") {
			return true
		}

		if strings.HasSuffix(filename, "_i386.deb") {
			return true
		}
	}
	return false
}

func randSelectN(in map[string]struct{}, n int) (result []string) {
	total := len(in)

	if total <= n {
		var idx int
		result = make([]string, total)
		for key := range in {
			result[idx] = key
			idx++
		}
		return
	}

	selectedRate := float64(n) / float64(total)
	for key := range in {
		if rand.Float64() <= selectedRate {
			result = append(result, key)
		}
	}
	return
}

func getChangeFiles(conf configs.CdnCheckerConf) ([]string, error) {
	changeList, err := getChangeList(conf)
	if err != nil {
		return nil, err
	}
	var changeMetaInfoList []ChangeMetaInfo
	for _, name := range changeList {
		tsStr := strings.TrimSuffix(name, ".json")
		ts, err := strconv.ParseInt(tsStr, 10, 64)
		if err != nil {
			continue
		}
		t := time.Unix(ts, 0)
		changeMetaInfoList = append(changeMetaInfoList, ChangeMetaInfo{
			name: name,
			t:    t,
		})
	}

	if len(changeMetaInfoList) == 0 {
		return nil, nil
	}

	sort.Sort(changeMetaInfoSlice(changeMetaInfoList))
	maxT := changeMetaInfoList[len(changeMetaInfoList)-1].t

	var recentlyChanges []string
	for i := len(changeMetaInfoList) - 1; i >= 0; i-- {
		t := changeMetaInfoList[i].t
		if maxT.Sub(t) < 10*24*time.Hour {
			recentlyChanges = append(recentlyChanges, changeMetaInfoList[i].name)
		} else {
			break
		}
	}
	// reverse
	for i := len(recentlyChanges)/2 - 1; i >= 0; i-- {
		opp := len(recentlyChanges) - 1 - i
		recentlyChanges[i], recentlyChanges[opp] = recentlyChanges[opp], recentlyChanges[i]
	}

	debChangeFilesMap := make(map[string]struct{})
	nonDebChangeFilesMap := make(map[string]struct{})
	var changeFiles []string
	for _, name := range recentlyChanges {
		ci, err := getChangeInfo(conf, name)
		if err != nil {
			log.Println("WARN:", err)
			continue
		}

		for _, a := range ci.Added {
			if ignoreFile(a.FilePath) {
				continue
			}

			if strings.HasSuffix(a.FilePath, ".deb") {
				debChangeFilesMap[a.FilePath] = struct{}{}
			} else {
				nonDebChangeFilesMap[a.FilePath] = struct{}{}
			}
		}
	}
	// about 300 deb files selected
	changeFiles = randSelectN(debChangeFilesMap, 300)
	for file := range nonDebChangeFilesMap {
		changeFiles = append(changeFiles, file)
	}
	return changeFiles, nil
}
