package jenkins

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	configs "mirrors_status/internal/config"
	"mirrors_status/internal/log"
	"net/http"
	"net/url"
	"time"
)

type BuildInfo struct {
	Building        bool   `json:"building"`
	DisplayName     string `json:"displayName"` // '#'+id
	FullDisplayName string `json:"fullDisplayName"`
	Duration        int    `json:"duration"`
	ID              string `json:"id"`
	QueueID         int64  `json:"queueId"`
	Result          string `json:"result"`
	Timestamp       int64  `json:"timestamp"`
	URL             string `json:"url"`
}

type QueueInfo struct {
	Executable struct {
		Number int    `json:"number"`
		URL    string `json:"url"`
		//Result string `json:"result"`
	} `json:"executable"`
}

func TriggerBuild(job *configs.JobInfo, params map[string]string, abort <-chan bool) (int, error) {
	values := url.Values{}
	values.Set("token", job.Token)
	for k, v := range params {
		values.Set(k, v)
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/build",
		job.URL), bytes.NewBufferString(values.Encode()))
	if err != nil {
		return -1, err
	}
	// must set 'Content-type' to 'application/x-www-form-urlencoded'
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return -1, err
	}
	var data []byte
	if resp.Body != nil {
		defer resp.Body.Close()
		data, _ = ioutil.ReadAll(resp.Body)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return -1, fmt.Errorf("Info: %s, reason: %s",
			resp.Status, string(data))
	}
	var buildID int
	var timer = time.NewTimer(time.Second * 3)
	for {
		select {
		case <-timer.C:
			queueInfo, err := GetQueueInfo(resp.Header.Get("Location"))
			log.Infof("%v", resp.Header)
			if err != nil {
				return -1, err
			}
			log.Infof("Number %d", queueInfo.Executable.Number)
			if queueInfo.Executable.Number != 0 {
				buildID = queueInfo.Executable.Number
				return buildID, nil
			}
			timer.Reset(time.Second * 10)
		case <-abort:
			return  buildID, nil
		}

	}
	//return buildID, nil
}

func LastBuildInfo(job *configs.JobInfo) (*BuildInfo, error) {
	return GetBuildInfo(job, -1)
}

func GetBuildInfo(job *configs.JobInfo, id int) (*BuildInfo, error) {
	idStr := fmt.Sprint(id)
	if id == -1 {
		idStr = "lastBuild"
	}

	resp, err := http.Get(fmt.Sprintf("%s/%s/api/json", job.URL, idStr))
	log.Info(job.URL, idStr)
	if err != nil {
		return nil, err
	}
	if resp.Body == nil {
		return nil, fmt.Errorf("get build info failure")
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("get build info failure, reason: %s", string(data))
	}

	var info BuildInfo
	err = json.Unmarshal(data, &info)
	if err != nil {
		fmt.Println("Failed to get build info:", string(data))
		return nil, err
	}

	return &info, nil
}

func GetQueueInfo(queueURL string) (*QueueInfo, error) {
	u, _ := url.Parse(queueURL)
	log.Infof(u.Scheme)
	if u.Scheme == "http" {
		u.Scheme = "https"
		queueURL = u.String()
	}
	log.Info(queueURL)
	resp, err := http.Get(queueURL + "/api/json")
	if err != nil {
		log.Infof("%#v", err)
		return nil, err
	}
	if resp.Body == nil {
		return nil, fmt.Errorf("get queue info failure")
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	log.Infof("Data:%s", string(data))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("get queue info failure, reason: %s", string(data))
	}

	var info QueueInfo
	err = json.Unmarshal(data, &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func GetMirrorJobsByUpstream(upstream string) configs.JobInfoList {
	jenkinsConf := configs.NewJenkinsConfig()
	for _, repo := range jenkinsConf.Repositories {
		if repo.Name == upstream {
			return repo.Jobs
		}
	}
	return nil
}
