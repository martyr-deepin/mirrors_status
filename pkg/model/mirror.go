package model

import (
	"bytes"
	"database/sql/driver"
	"errors"
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
	Index int	`gorm:"primary_key" json:"index"`
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
	IsKey bool `gorm:"default:false" json:"is_key"`
}