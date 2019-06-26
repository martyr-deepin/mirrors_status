package checker

import (
	"encoding/json"
	"fmt"
	"mirrors_status/internal/log"
	"net/http"
)

type mirrors []*Mirror

type UnpublishedMirrors struct {
	Error   string  `json:"error"`
	Mirrors mirrors `json:"mirrors"`
}

func (v mirrors) Len() int {
	return len(v)
}

func (v mirrors) Less(i, j int) bool {
	return v[i].Weight > v[j].Weight
}

func (v mirrors) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func GetUnpublishedMirrors(url string) (mirrors, error) {
	log.Infof("Fetching mirrors via:%s", url)
	rep, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer rep.Body.Close()

	d := json.NewDecoder(rep.Body)
	var v UnpublishedMirrors
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

