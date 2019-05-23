package jenkins

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	configs "mirrors_status/internal/config"
	"net/http"
	"net/url"
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
	Number int    `json:"number"`
	URL    string `json:"url"`
	Result string `json:"result"`
}

func TriggerBuild(job *configs.JobInfo, params map[string]string) (int, error) {
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

	queueInfo, err := GetQueueInfo(resp.Header.Get("Location"))
	if err != nil {
		return -1, err
	}
	return queueInfo.Number, nil
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
	// Exchange protocol from HTTP to HTTPS due to bugs in Jenkins
	// if match, _ := regexp.MatchString("^https*", queueURL); !match {
	// 	queueURL = "https" + string([]byte(queueURL)[4:])
	// }
	resp, err := http.Get(queueURL + "/api/json")
	if err != nil {
		return nil, err
	}
	if resp.Body == nil {
		return nil, fmt.Errorf("get queue info failure")
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
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