package cdn_checker

import (
	"encoding/json"
	"time"
)

type ChangeInfo struct {
	Preview string     `json:"preview"` // date type timestamp
	Current string     `json:"current"` // date type timestamp
	Size    uint64     `json:"size"`
	Deleted []FileInfo `json:"deleted"`
	Added   []FileInfo `json:"added"`
}

type Mirror struct {
	Id       string                       `json:"id"`
	Weight   int                          `json:"weight"`
	Name     string                       `json:"name"`
	UrlHttp  string                       `json:"urlHttp"`
	UrlHttps string                       `json:"urlHttps"`
	UrlFtp   string                       `json:"urlFtp"`
	Country  string                       `json:"country"`
	Locale   map[string]map[string]string `json:"locale"`
}

type FileValidateInfo struct {
	FilePath string
	MD5Sum   []byte
	Size     int
	ModTime  string
	URL      string
}

type TestRecord struct {
	Standard *FileValidateInfo
	Result   *FileValidateInfo
	Equal    bool
	Err      error
}

type TestResult struct {
	Name           string `json:"name"`
	UrlPrefix      string `json:"url_prefix"`
	CdnNodeAddress string `json:"cdn_node_address"`
	Records        []TestRecord `json:"records"`
	Percent        float64 `json:"percent"`
	NumErrs        int `json:"num_errs"`
}

type ChangeMetaInfo struct {
	name string
	t    time.Time
}

type FileInfo struct {
	FilePath string `json:"filepath"`
	FileSize string `json:"filesize"`
}

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