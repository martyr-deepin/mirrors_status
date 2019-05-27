package mirror

import (
	"bytes"
	"database/sql/driver"
	"errors"
	configs "mirrors_status/internal/config"
	"mirrors_status/pkg/db/client/mysql"
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

type Mirror struct {
	Mid int	`gorm:"primary_key" json:"index"`
	Id    string	`gorm:"type:varchar(64)" json:"id"`

	//Type     MirrorType
	Name     string `gorm:"type:varchar(64)" json:"name"`
	Upstream string `gorm:"type:varchar(64)" json:"upstream"`
	Weight   int `gorm:"type:int" json:"weight"`
	Location string `gorm:"type:varchar(64)" json:"location"`
	Locale   JSON `sql:"type:json" json:"locale,omitempty"`
	//LocaleBody ExtField `sql:"-"`

	UrlHttps string `gorm:"type:varchar(64)" json:"url_https"`
	UrlHttp  string `gorm:"type:varchar(64)" json:"url_http"`
	UrlFtp   string `gorm:"type:varchar(64)" json:"url_ftp"`
	UrlRsync string `gorm:"type:varchar(64)" json:"url_rsync"`

	Tags  string  `gorm:"type:varchar(64)" json:"tags"`
	Extra JSON `sql:"type:json" json:"extra,omitempty"`
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
	err = mysql.NewMySQLClient().Table("mirrors").Where("upstream = ?", upstream).Scan(&mirrors).Error
	return
}

func GetMirrorsByIndices(mirrorIndices []int) (mirrors []Mirror, err error) {
	err = mysql.NewMySQLClient().Table("mirrors").Where("`mid` in (?)", mirrorIndices).Scan(&mirrors).Error
	return
}

func GetAllMirrors() (mirrors []Mirror, err error) {
	err = mysql.NewMySQLClient().Table("mirrors").Order("weight").Find(&mirrors).Error
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