package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	baseUrl       = "http://packages.deepin.com/deepin/"
	changeListUrl = baseUrl + "changelist/"
)

func getChangeList() ([]string, error) {
	resp, err := http.Get(changeListUrl)
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

type changeMetaInfo struct {
	name string
	t    time.Time
}

type changeMetaInfoSlice []changeMetaInfo

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

func getChangeFiles() ([]string, error) {
	changeList, err := getChangeList()
	if err != nil {
		return nil, err
	}
	var changeMetaInfoList []changeMetaInfo
	for _, name := range changeList {
		tsStr := strings.TrimSuffix(name, ".json")
		ts, err := strconv.ParseInt(tsStr, 10, 64)
		if err != nil {
			continue
		}
		t := time.Unix(ts, 0)
		changeMetaInfoList = append(changeMetaInfoList, changeMetaInfo{
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

	changeFilesMap := make(map[string]struct{})
	var changeFiles []string
	for _, name := range recentlyChanges {
		ci, err := getChangeInfo(name)
		if err != nil {
			log.Println("WARN:", err)
			continue
		}

		for _, a := range ci.Added {
			if !ignoreFile(a.FilePath) {
				changeFilesMap[a.FilePath] = struct{}{}
			}
		}
		for _, d := range ci.Deleted {
			if !ignoreFile(d.FilePath) {
				delete(changeFilesMap, d.FilePath)
			}
		}
	}
	for file := range changeFilesMap {
		changeFiles = append(changeFiles, file)
	}
	return changeFiles, nil
}

func getChangeInfo(name string) (*changeInfo, error) {
	u := changeListUrl + name
	log.Println("getChangeInfo u:", u)
	resp, err := http.Get(u)
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

type fileInfo struct {
	FilePath string `json:"filepath"`
	FileSize string `json:"filesize"`
}

func saveChangeFiles(files []string) error {
	os.MkdirAll("result", 0755)

	filename := "result/change-files.txt"
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	bw := bufio.NewWriter(f)

	for _, file := range files {
		_, err = fmt.Fprintln(bw, file)
		if err != nil {
			return err
		}
	}
	err = bw.Flush()
	if err != nil {
		return err
	}
	return nil
}

//func saveChangeInfo(ci *changeInfo) error {
//	filename := fmt.Sprintf("result/changeInfo-%s.txt", ci.Current)
//	os.MkdirAll("result", 0755)
//
//	f, err := os.Create(filename)
//	if err != nil {
//		return err
//	}
//	defer f.Close()
//
//	bw := bufio.NewWriter(f)
//
//	fmt.Fprintln(bw, "ts:", ci.Current)
//	fmt.Fprintln(bw, "previous ts:", ci.Preview)
//
//	for _, fileInfo := range ci.Added {
//		_, err = fmt.Fprintf(bw, "A %s\n", fileInfo.FilePath)
//		if err != nil {
//			return err
//		}
//	}
//
//	for _, fileInfo := range ci.Deleted {
//		_, err = fmt.Fprintf(bw, "D %s\n", fileInfo.FilePath)
//		if err != nil {
//			return err
//		}
//	}
//
//	err = bw.Flush()
//	if err != nil {
//		return err
//	}
//	return nil
//}
