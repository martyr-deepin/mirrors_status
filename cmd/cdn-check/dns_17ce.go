package main

import (
	"crypto/tls"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
)

const userAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/68.0.3440.84 Safari/537.36"

type checkUserResult struct {
	Rt   bool `json:"rt"`
	Data struct {
		Error string `json:"error"`
		Code  string `json:"code"`
		URL   string `json:"url"`
		User  string `json:"user"`
		Ut    int    `json:"ut"`

		Fullips []struct {
			City     string `json:"city"`
			Fullname string `json:"fullname"`
			Isp      string `json:"isp"`
			Ispid    string `json:"ispid"`
			Link     string `json:"link"`
			Linkname string `json:"linkname"`
			Name     string `json:"name"`
			NodeType string `json:"node_type"`
			ProID    string `json:"pro_id"`
			Province string `json:"province"`
			Py       string `json:"py"`
			Sid      string `json:"sid"`
		} `json:"fullips"`

		// TODO
		//Nodelink []interface{} `json:"nodelink"`

	} `json:"data"`
}

const refer = site + "/"
const site = "https://www.17ce.com"
const apiPathCheckUser = "/site/checkuser"

// 获取的 code 应该只对 websocket 接口有效
func checkUser(url1 string, type0 string) (*checkUserResult, error) {
	url0 := site + apiPathCheckUser
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
	req.Header.Set("Referer", refer)
	req.Header.Set("User-Agent", userAgent)
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

type SpeedRequest struct {
	Txnid             int `json:"txnid"`
	NodeType          int `json:"nodetype"`
	Num               int `json:"num"`
	Url               string
	TestType          string
	Host              string
	TimeOut           int
	Request           string
	NoCache           bool
	Speed             int
	Cookie            string
	Trace             bool
	Referer           string
	UserAgent         string
	FollowLocation    int
	GetMD5            bool
	GetResponseHeader bool
	MaxDown           int
	AutoDecompress    bool
	Type              int   `json:"type"`
	Isps              []int `json:"isps"`
	ProIds            []int `json:"pro_ids"`
	//CityIds           []int `json:"city_ids"`
	// 虽然文档上有 city_ids 字段，但是加上这个字段会报错 area value invalid
	Areas []int `json:"areas"`
	//Proxies           string
	SnapShot  bool
	PostField string `json:"postfield"`
	PingCount int
	PingSize  int
}

type SpeedResponse struct {
	Rt    int             `json:"rt"`
	Error string          `json:"error"`
	Txnid int             `json:"txnid"`
	Type  string          `json:"type"`
	Data  json.RawMessage `json:"data"`
}

func testDNS(host string) ([]string, error) {
	checkUserResult, err := checkUser(host, "dns")
	if err != nil {
		return nil, err
	}

	user := checkUserResult.Data.User
	log.Println("user:", user)
	code := checkUserResult.Data.Code
	log.Println("code:", code)
	ut := checkUserResult.Data.Ut
	log.Println("ut:", ut)

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
	log.Println("url0:", url0.String())

	header := make(http.Header)
	header.Set("User-Agent", userAgent)
	header.Set("Origin", "https://www.17ce.com")
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
			log.Println("task end")
			break
		} else if resp.Type == "TaskAccept" {
			log.Println("task accept")
		} else if resp.Type == "NewData" {
			log.Println("new data")
			newData, err := unmarshalNewData(resp.Data)
			if err != nil {
				log.Println("WARN:", err)
				continue
			}

			if newData.ErrMsg != "" {
				log.Println("WARN newData.ErrMsg:", newData.ErrMsg)
				continue
			}

			log.Println(newData.SrcIP)
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

type NewData struct {
	TaskId   string
	NodeID   int
	ErrMsg   string
	NsLookup float64
	SrcIP    string
	NodeInfo struct {
		Ip     string `json:"ip"`
		Area   string `json:"area"`
		Isp    string `json:"isp"`
		ProId  string `json:"pro_id"`
		CityId string `json:"city_id"`
	}
	SrcIP0 struct {
		SrcIp     string `json:"srcip"`
		SrcIpFrom string `json:"srcip_from"`
	} `json:"srcip"`
}

func unmarshalNewData(data []byte) (*NewData, error) {
	var v NewData
	err := json.Unmarshal(data, &v)
	if err != nil {
		return nil, err
	}
	return &v, nil
}
