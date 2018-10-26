package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type unpublishedMirrors struct {
	Error   string  `json:"error"`
	Mirrors mirrors `json:"mirrors"`
}

type mirror struct {
	Id       string                       `json:"id"`
	Weight   int                          `json:"weight"`
	Name     string                       `json:"name"`
	UrlHttp  string                       `json:"urlHttp"`
	UrlHttps string                       `json:"urlHttps"`
	UrlFtp   string                       `json:"urlFtp"`
	Country  string                       `json:"country"`
	Locale   map[string]map[string]string `json:"locale"`
}

func (m *mirror) getUrlPrefix() (result string) {
	if m.UrlHttp != "" {
		result = "http://" + m.UrlHttp
	}
	if m.UrlHttps != "" {
		result = "https://" + m.UrlHttps
	}
	return
}

type mirrors []*mirror

// implement sort.Interface interface

func (v mirrors) Len() int {
	return len(v)
}

func (v mirrors) Less(i, j int) bool {
	return v[i].Weight > v[j].Weight
}

func (v mirrors) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func getUnpublishedMirrors(url string) (mirrors, error) {
	log.Println("mirrors api url:", url)

	rep, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer rep.Body.Close()

	d := json.NewDecoder(rep.Body)
	var v unpublishedMirrors
	err = d.Decode(&v)
	if err != nil {
		return nil, err
	}

	if rep.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("callApiMirrors: fetch %q is not ok, status: %q",
			url, rep.Status)
	}

	return v.Mirrors, nil
}
