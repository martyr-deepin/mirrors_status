package mirror

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"github.com/influxdata/influxdb/client/v2"
	configs "mirrors_status/internal/config"
	"mirrors_status/internal/log"
	"mirrors_status/pkg/db/client/influxdb"
	"mirrors_status/pkg/db/client/mysql"
	"time"
)

type MirrorsPoint struct {
	Name     string  `json:"name"`
	Progress float64 `json:"progress"`
}

type MirrorsCdnPoint struct {
	MirrorId   string  `json:"mirror_id"`
	NodeIpAddr string  `json:"node_ip_addr"`
	Progress   float64 `json:"progress"`
}

type MirrorResponse struct {
	MirrorsPoint    MirrorsPoint      `json:"mirrors_point"`
	MirrorsCdnPoint []MirrorsCdnPoint `json:"cdns"`
}

type JSON []byte

func (j JSON) Value() (driver.Value, error) {
	if j.IsNull() {
		return nil, nil
	}
	return string(j), nil
}
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	s, ok := value.([]byte)
	if !ok {
		errors.New("Invalid Scan Source")
	}
	*j = append((*j)[0:0], s...)
	return nil
}
func (m JSON) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	return m, nil
}
func (m *JSON) UnmarshalJSON(data []byte) error {
	if m == nil {
		return errors.New("null point exception")
	}
	*m = append((*m)[0:0], data...)
	return nil
}
func (j JSON) IsNull() bool {
	return len(j) == 0 || string(j) == "null"
}
func (j JSON) Equals(j1 JSON) bool {
	return bytes.Equal([]byte(j), []byte(j1))
}

// extension field
type ExtField map[string]interface{}

type MirrorType int

const (
	UnknownServer MirrorType = iota
	IndieServer
	CdnServer
)

type CdnNodeCompletion struct {
	NodeName   string  `json:"nodeName"`
	Completion float64 `json:"completion"`
}

type Mirror struct {
	Mid int    `gorm:"primary_key" json:"index"`
	Id  string `gorm:"type:varchar(64)" json:"id"`

	//Type     MirrorType
	Name     string `gorm:"type:varchar(64)" json:"name"`
	Upstream string `gorm:"type:varchar(64)" json:"upstream"`
	Weight   int    `gorm:"type:int" json:"weight"`
	Location string `gorm:"type:varchar(64)" json:"location"`
	Locale   JSON   `sql:"type:json" json:"locale,omitempty"`
	//LocaleBody ExtField `sql:"-"`

	UrlHttps string `gorm:"type:varchar(64)" json:"url_https"`
	UrlHttp  string `gorm:"type:varchar(64)" json:"url_http"`
	UrlFtp   string `gorm:"type:varchar(64)" json:"url_ftp"`
	UrlRsync string `gorm:"type:varchar(64)" json:"url_rsync"`

	HttpsProgress float64 `sql:"-" json:"https_progress,omitempty"`
	HttpProgress  float64 `sql:"-" json:"http_progress,omitempty"`
	FtpProgress   float64 `sql:"-" json:"ftp_progress,omitempty"`
	RsyncProgress float64 `sql:"-" json:"rsync_progress,omitempty"`
	CdnCompletion []CdnNodeCompletion `sql:"-" json:"cdn_completion,omitempty"`

	Tags  string `gorm:"type:varchar(64)" json:"tags"`
	Extra JSON   `sql:"type:json" json:"extra,omitempty"`
	//ExtraBody ExtField `sql:"-"`
	IsKey bool `gorm:"default:0" json:"is_key"`
}

func (m Mirror) CreateMirror() error {
	return mysql.NewMySQLClient().Table("mirrors").Create(&m).Error
}

func DeleteMirror(index int) error {
	return mysql.NewMySQLClient().Table("mirrors").Delete(&Mirror{}, "`mid` = ?", index).Error
}

func (m Mirror) UpdateMirror() error {
	return mysql.NewMySQLClient().Table("mirrors").Updates(&m, true).Error
}

func GetMirrorsByUpstream(upstream string) (mirrors []Mirror, err error) {
	list := []Mirror{}
	err = mysql.NewMySQLClient().Table("mirrors").Where("upstream = ?", upstream).Scan(&list).Error
	for _, mirror := range list {
		_ = mirror.GetMirrorCompletion()
		_ = mirror.GetMirrorCdnCompletion()
		mirrors = append(mirrors, mirror)
	}
	return
}

func GetMirrorsByIndices(mirrorIndices []int) (mirrors []Mirror, err error) {
	list := []Mirror{}
	err = mysql.NewMySQLClient().Table("mirrors").Where("`mid` in (?)", mirrorIndices).Scan(&list).Error
	for _, mirror := range list {
		_ = mirror.GetMirrorCompletion()
		_ = mirror.GetMirrorCdnCompletion()
		mirrors = append(mirrors, mirror)
	}
	return
}

func GetAllMirrors() (mirrors []Mirror, err error) {
	list := []Mirror{}
	err = mysql.NewMySQLClient().Table("mirrors").Order("weight").Find(&list).Error
	for _, mirror := range list {
		_ = mirror.GetMirrorCompletion()
		_ = mirror.GetMirrorCdnCompletion()
		mirrors = append(mirrors, mirror)
	}
	return
}

func GetMirrorUpstreams() (upstreamList configs.RepositoryInfoList) {
	jenkinsConfig := configs.NewJenkinsConfig()
	upstreamList = jenkinsConfig.Repositories
	for _, upstream := range upstreamList {
		for _, job := range upstream.Jobs {
			job.Token = ""
		}
	}
	return upstreamList
}

func interface2Float(data interface{}) (float64, error) {
	if v, ok := data.(json.Number); ok {
		progress, err := v.Float64()
		if err != nil {
			return -1, err
		}
		return progress, nil
	}
	return -1, errors.New("parameter error")
}

func (m *Mirror) GetMirrorCompletion() (err error) {
	if m.UrlHttps != "" {
		data, err := influxdb.LatestData("mirrors", "progress", "", map[string]interface{}{"name": m.UrlHttps}, "")
		if err != nil {
			return err
		}
		if len(data[0]) < 2 {
			return errors.New("no https progress record")
		}
		m.HttpsProgress, err = interface2Float(data[0][1])
		if err != nil {
			return err
		}
	}
	if m.UrlHttp != "" {
		data, err := influxdb.LatestData("mirrors", "progress", "", map[string]interface{}{"name": m.UrlHttp}, "")
		if err != nil {
			return err
		}
		if len(data[0]) < 2 {
			return errors.New("no http progress record")
		}
		m.HttpProgress, err = interface2Float(data[0][1])
		if err != nil {
			return err
		}
	}
	if m.UrlFtp != "" {
		data, err := influxdb.LatestData("mirrors", "progress", "", map[string]interface{}{"name": m.UrlFtp}, "")
		if err != nil {
			return err
		}
		if len(data[0]) < 2 {
			return errors.New("no ftp progress record")
		}
		m.FtpProgress, err = interface2Float(data[0][1])
		if err != nil {
			return err
		}
	}
	if m.UrlRsync != "" {
		data, err := influxdb.LatestData("mirrors", "progress", "", map[string]interface{}{"name": m.UrlRsync}, "")
		if err != nil {
			return err
		}
		if len(data[0]) < 2 {
			return errors.New("no rsync progress record")
		}
		m.RsyncProgress, err = interface2Float(data[0][1])
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Mirror) GetMirrorCdnCompletion() (err error) {
	data, err := influxdb.LatestData("mirrors_cdn", "progress", "node_ip_addr", map[string]interface{}{"mirror_id": m.Id}, "node_ip_addr")
	log.Info(data)
	return nil
}

func (m *MirrorsPoint) PushMirror() error {
	var cPoints []*client.Point
	p, err := client.NewPoint(
		"mirrors",
		map[string]string{
			"name": m.Name,
		},
		map[string]interface{}{
			"progress": m.Progress,
			"latency":  0,
		},
		time.Now())
	if err != nil {
		return err
	}
	log.Infof("Pushing mirror:%v", m)
	cPoints = append(cPoints, p)
	return influxdb.Write(cPoints...)
}

func PushMirrors(points []MirrorsPoint) error {
	for _, p := range points {
		err := p.PushMirror()
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *MirrorsCdnPoint) PushMirrorCdn() error {
	var cPoints []*client.Point
	p, err := client.NewPoint(
		"mirrors_cdn",
		map[string]string{
			"mirror_id":    m.MirrorId,
			"node_ip_addr": m.NodeIpAddr,
		},
		map[string]interface{}{
			"progress": m.Progress,
		},
		time.Now())
	if err != nil {
		return err
	}
	cPoints = append(cPoints, p)
	return influxdb.Write(cPoints...)
}

func PushMirrorsCdn(points []MirrorsCdnPoint) error {
	for _, p := range points {
		err := p.PushMirrorCdn()
		if err != nil {
			return err
		}
	}
	return nil
}