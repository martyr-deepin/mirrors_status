package configs

import (
	"gopkg.in/gomail.v2"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"mirrors_status/internal/log"
)

type InfluxDBConf struct {
	Host     string `yaml:"host"`
	Port     int `yaml:"port"`
	DBName   string `yaml:"dbName"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type HttpConf struct {
	Port int `yaml:"port"`
	Host string `yaml:"host"`
	AllowOrigin []string `yaml:"allow-origin"`
	AdminMail string `yaml:"admin-mail"`
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

type LdapConf struct {
	Server string `yaml:"server"`
	Port int `yaml:"port"`
	BindDn string `yaml:"bind-dn"`
	BindPasswd string `yaml:"bind-passwd"`
	UserSearch string `yaml:"user-search"`
}

type MailConf struct {
	Host string `yaml:"host"`
	Port int `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type JenkinsConf struct {
	Addr string `yaml:"addr"`
	Trigger string `yml:"trigger"`
	Delay int `yaml:"delay"`
	Retry int `yaml:"retry"`
}

type RedisConf struct {
	Host string `yaml:"host"`
	Port int `yaml:"port"`
	DBName int `yaml:"client"`
}

type ServerConf struct {
	InfluxDB *InfluxDBConf `yaml:"influxdb"`
	MySQLDB *MySQLConf `yaml:"mysql"`
	Http     *HttpConf     `yaml:"http"`
	CdnChecker *CdnCheckerConf `yaml:"cdn-checker"`
	Ldap *LdapConf `yaml:"ldap"`
	Mail *MailConf `yaml:"mail"`
	Jenkins *JenkinsConf `yaml:"jenkins"`
	Redis *RedisConf `yaml:"redis"`
}

func ErrHandler(op string, err error) {
	if err != nil {
		log.Fatalf("%s found error: %v", op, err)
	}
}

func NewServerConfig() *ServerConf {
	var serverConf ServerConf
	ymlFile, err := ioutil.ReadFile("configs/config.yml")
	ErrHandler("opening file", err)

	err = yaml.Unmarshal(ymlFile, &serverConf)
	ErrHandler("unmarshal yaml", err)

	return &serverConf
}

var (
	MailDialer *gomail.Dialer
)

func GetMailDialer() *gomail.Dialer {
	return MailDialer
}

func InitMailClient(conf *MailConf) {
	log.Infof("Trying connecting mail server:%s:%d %s", conf.Host, conf.Port, conf.Username)
	MailDialer = &gomail.Dialer{
		Host: conf.Host,
		Port: conf.Port,
		Username: conf.Username,
		Password: conf.Password,
		SSL: true,
	}
}