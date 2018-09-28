package main

import (
	"encoding/json"
	"fmt"
	"github.com/influxdata/influxdb/client/v2"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"
)

func loadOne(p string) ([]OldResult, error) {
	t, err := parseTimeByName(p)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var vs []OldResult
	err = json.NewDecoder(f).Decode(&vs)
	if err != nil {
		return nil, err
	}
	var ret []OldResult
	for _, v := range vs {
		v.CheckTime = t
		ret = append(ret, v)
	}
	return ret, nil
}

func loadAll(dir string) []OldResult {
	fs, err := ioutil.ReadDir(dir)
	if err != nil {
		fmt.Println("Can't scan:", err)
		return nil
	}
	var ret []OldResult
	for _, info := range fs {
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".json") {
			continue
		}
		vs, err := loadOne(path.Join(dir, info.Name()))
		if err != nil {
			fmt.Println("E:", err)
			continue
		}
		ret = append(ret, vs...)
	}
	return ret
}

func parseTimeByName(raw string) (time.Time, error) {
	raw = path.Base(raw)
	layout := ("2006-01-02_15:04:05")
	var v string
	_, err := fmt.Sscanf(raw, "result_cn_%s", &v)
	if err == nil {
		return time.Parse(layout, strings.TrimRight(v, ".json"))
	}
	_, err = fmt.Sscanf(raw, "result_other_%s", &v)
	if err == nil {
		return time.Parse(layout, strings.TrimRight(v, ".json"))
	}
	return time.Now(), fmt.Errorf("Unknown name format:%q %q -> %v", raw, v, err)
}

func PushMirrorStatus(c *InfluxClient, vs []OldResult) error {
	var data []*client.Point
	for _, v := range vs {
		p, err := client.NewPoint(
			"mirrors",
			map[string]string{
				"name": v.Name,
			},
			map[string]interface{}{
				"progress": v.Progress,
				"latency":  v.Latency,
			},
			v.CheckTime)
		if err != nil {
			panic(err)
		}

		data = append(data, p)
	}
	return c.Write(data...)
}
