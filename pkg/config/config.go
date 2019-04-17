package configs

import (
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"mirrors_status/pkg/log"
)

type InfluxDBConf struct {
	Host     string `yaml:"host"`
<<<<<<< HEAD:pkg/config/config.go
	Port     string `yaml:"port"`
=======
	Port     int `yaml:"port"`
>>>>>>> zhaojuwen/sync-check:pkg/config/config.go
	DBName   string `yaml:"dbName"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type HttpConf struct {
	Port int `yaml:"port"`
	Host string `yaml:"host"`
}

type MySQLConf struct {
	Host     string `yaml:"host"`
	Port     int `yaml:"port"`
	DBName   string `yaml:"dbName"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type CdnCheckerConf struct {
	DefaultCdn string `yaml:"default-cdn"`
	UserAgent string `yaml:"user-agent"`
	ApiSite string `yaml:"api-site"`
	ApiPath string `yaml:"api-path"`
	Target string `yaml:"target"`
	SourceUrl string `yaml:"source-url"`
	SourcePath string `yaml:"source-path"`
}

type MySQLConf struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	DBName   string `yaml:"dbName"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type CdnCheckerConf struct {
	DefaultCdn string `yaml:"default-cdn"`
	UserAgent string `yaml:"user-agent"`
	ApiSite string `yaml:"api-site"`
	ApiPath string `yaml:"api-path"`
	Target string `yaml:"target"`
	SourceUrl string `yaml:"source-url"`
	SourcePath string `yaml:"source-path"`
}

type ServerConf struct {
	InfluxDB *InfluxDBConf `yaml:"influxdb"`
	MySQLDB *MySQLConf `yaml:"mysql"`
	Http     *HttpConf     `yaml:"http"`
	CdnCkecker *CdnCheckerConf `yaml:"cdn-checker"`
}

func (c *ServerConf) ErrHandler(op string, err error) {
	if err != nil {
		log.Fatalf("%s found error: %v", op, err)
	}
}

func (c *ServerConf) GetConfig() *ServerConf {
	log.Info("Loading configs")
	ymlFile, err := ioutil.ReadFile("configs/config.yml")
	c.ErrHandler("openning file", err)

	err = yaml.Unmarshal(ymlFile, c)
	c.ErrHandler("unmarshal yaml", err)

	return c
}
