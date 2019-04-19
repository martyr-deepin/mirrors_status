package cdn_checker

import (
	"crypto/tls"
	"encoding/json"
	"github.com/gorilla/websocket"
	"mirrors_status/pkg/log"
	"mirrors_status/pkg/config"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type CheckTool struct {
	Conf *configs.CdnCheckerConf
}

func NewCheckTool(conf *configs.CdnCheckerConf) CheckTool {
	return CheckTool{
		Conf: conf,
	}
}

func (tool *CheckTool) CheckUser(url1 string, type0 string) (*checkUserResult, error) {
	url0 := tool.Conf.ApiSite + tool.Conf.ApiPath
	postForm := make(url.Values)
	postForm.Add("url", url1)
	postForm.Add("type", type0)
	postForm.Add("isp", "0")

	bodyStr := postForm.Encode()
	body := strings.NewReader(bodyStr)

	req, err := http.NewRequest(http.MethodPost, url0, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Referer", tool.Conf.ApiSite + "/")
	req.Header.Set("User-Agent", tool.Conf.UserAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := clientNormal.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result checkUserResult
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func UnmarshalNewData(data []byte) (*NewData, error) {
	var v NewData
	err := json.Unmarshal(data, &v)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (tool *CheckTool) TestDNS(host string) ([]string, error) {
	checkUserResult, err := tool.CheckUser(host, "dns")
	if err != nil {
		return nil, err
	}

	user := checkUserResult.Data.User
	code := checkUserResult.Data.Code
	ut := checkUserResult.Data.Ut
	log.Infof("User:%s, Code:%s, UT:%d", user, code, ut)

	urlValues := make(url.Values)
	urlValues.Set("user", user)
	urlValues.Set("code", code)
	urlValues.Set("ut", strconv.Itoa(ut))
	url0 := url.URL{
		Scheme:   "wss",
		Host:     "wsapi.17ce.com:8001",
		Path:     "/socket",
		RawQuery: urlValues.Encode(),
	}
	log.Infof("url0:%s", url0.String())

	header := make(http.Header)
	header.Set("User-Agent", tool.Conf.UserAgent)
	header.Set("Origin", tool.Conf.ApiSite)
	websocket.DefaultDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	conn, _, err := websocket.DefaultDialer.Dial(url0.String(), header)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	var speedReq SpeedRequest
	speedReq.Host = host
	speedReq.TestType = "DNS"
	speedReq.TimeOut = 10
	speedReq.Areas = []int{0, 1, 2, 3}
	speedReq.Isps = []int{0, 1, 2, 6, 7, 8, 17, 18, 19, 3, 4}
	speedReq.NodeType = 1
	speedReq.Num = 1
	speedReq.ProIds = []int{12, 49, 79, 80, 180, 183, 184, 188, 189, 190, 192, 193, 194, 195, 196, 221, 227, 235, 236, 238, 241, 243, 250, 346, 349, 350, 351, 353, 354, 355, 356, 357, 239, 352, 3, 5, 8, 18, 27, 42, 43, 46, 47, 51, 56, 85}
	speedReq.Type = 1
	speedReq.Txnid = 1

	// read first msg
	_, _, err = conn.ReadMessage()
	if err != nil {
		return nil, err
	}

	// send DNS test request
	err = conn.WriteJSON(&speedReq)
	if err != nil {
		return nil, err
	}

	var ips []string

	for {
		var resp SpeedResponse
		err = conn.ReadJSON(&resp)
		if err != nil {
			return nil, err
		}
		if resp.Type == "TaskEnd" {
			log.Info("Task end")
			break
		} else if resp.Type == "TaskAccept" {
			log.Info("Task accept")
		} else if resp.Type == "NewData" {
			log.Info("New data")
			newData, err := UnmarshalNewData(resp.Data)
			if err != nil {
				log.Warningf("Unmarshal new data found error:%v", err)
				continue
			}

			if newData.ErrMsg != "" {
				log.Warningf("Error exists in new data:%s", newData.ErrMsg)
				continue
			}

			log.Infof("IP:%s", newData.SrcIP)
			srcIps := strings.Split(newData.SrcIP, ";")
			if len(srcIps) > 0 {
				srcIp := srcIps[0]
				if srcIp == "" {
					continue
				}
				var found bool
				for _, ip := range ips {
					if ip == srcIp {
						found = true
						break
					}
				}
				if !found {
					ips = append(ips, srcIp)
				}
			}

		}

	}

	return ips, nil
}