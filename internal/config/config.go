package configs

import (
	"gopkg.in/gomail.v2"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"mirrors_status/internal/log"
	"mirrors_status/pkg/db/client/influxdb"
	"mirrors_status/pkg/db/client/mysql"
	"mirrors_status/pkg/db/client/redis"
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
	AllowOrigin string `yaml:"allow-origin"`
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
	Server string `yml:"server"`
	Port int `yml:"port"`
	BindDn string `yml:"bind_dn"`
	BindPasswd string `yml:"bind_passwd"`
	UserSearch string `yml:"user_search"`
	GroupSearch string `yml:"group_search"`
}

type MailConf struct {
	Host string `yml:"host"`
	Port int `yml:"port"`
	Username string `yml:"username"`
	Password string `yml:"password"`
}

type JenkinsConf struct {
	Addr string `yml:"addr"`
	Trigger string `yml:"trigger"`
	Delay int `yml:"delay"`
	Retry int `yml:"retry"`
}

type RedisConf struct {
	Host string `yml:"host"`
	Port int `yml:"port"`
	Username string `yml:"username"`
	Password string `yml:"password"`
	DBName int `yml:"client"`
}

type ServerConf struct {
	InfluxDB *InfluxDBConf `yaml:"influxdb"`
	MySQLDB *MySQLConf `yaml:"mysql"`
	Http     *HttpConf     `yaml:"http"`
	CdnChecker *CdnCheckerConf `yaml:"cdn-checker"`
	Ldap *LdapConf `yml:"ldap"`
	Mail *MailConf `yml:"mail"`
	Jenkins *JenkinsConf `yml:"jenkins"`
	Redis *RedisConf `yml:"redis"`
}

func ErrHandler(op string, err error) {
	if err != nil {
		log.Fatalf("%s found error: %v", op, err)
	}
}

func NewServerConfig() *ServerConf {
	var serverConf ServerConf
	log.Info("Loading server configs")
	ymlFile, err := ioutil.ReadFile("configs/config.yml")
	ErrHandler("opening file", err)

	err = yaml.Unmarshal(ymlFile, &serverConf)
	ErrHandler("unmarshal yaml", err)

	return &serverConf
}

var (
	InfluxdbClient *influxdb.Client
	MysqlClient *mysql.Client
	RedisClient *redis.Client
	MailDialer *gomail.Dialer
)

func GetInfluxdbClient() *influxdb.Client {
	return InfluxdbClient
}

func GetMySQLClient() *mysql.Client {
	return MysqlClient
}

func GetMailDialer() *gomail.Dialer {
	return MailDialer
}

func GetRedisClient() *redis.Client {
	return RedisClient
}

func InitInfluxdbClient(host string, port int, dbname, username, password string) error {
	log.Infof("trying connecting influxdb:%s:%d %s %s %s", host, port, dbname, username, password)
	InfluxdbClient = &influxdb.Client{
		Host:     host,
		Port:     port,
		DbName:   dbname,
		Username: username,
		Password: password,
	}
	return InfluxdbClient.NewInfluxClient()
}

func InitMySQLClient(host string, port int, dbname, username, password string) error {
	log.Infof("trying connecting MySQL:%s:%d %s %s %s", host, port, dbname, username, password)
	MysqlClient = &mysql.Client{
		Host:     host,
		Port:     port,
		DbName:   dbname,
		Username: username,
		Password: password,
	}
	return MysqlClient.NewMySQLClient()
}

func InitRedisClient(host string, port int, username, password string, db int) error {
	log.Infof("trying connecting Redis:%s:%d", host, port)
	RedisClient = &redis.Client{
		Host: host,
		Port: port,
		Username: username,
		Password: password,
		DBName: db,
	}
	return RedisClient.NewRedisClient()
}

func InitDB(config ServerConf) {
	host := config.InfluxDB.Host
	port := config.InfluxDB.Port
	dbName := config.InfluxDB.DBName
	username := config.InfluxDB.Username
	password := config.InfluxDB.Password

	err := InitInfluxdbClient(host, port, dbName, username, password)
	if err != nil {
		log.Errorf("Connecting influxdb:%v found error:%v", config.InfluxDB, err)
	}

	host = config.MySQLDB.Host
	port = config.MySQLDB.Port
	dbName = config.MySQLDB.DBName
	username = config.MySQLDB.Username
	password = config.MySQLDB.Password

	err = InitMySQLClient(host, port, dbName, username, password)
	if err != nil {
		log.Errorf("Connecting MySQL:%v found error:%v", config.MySQLDB, err)
	}

	host = config.Redis.Host
	port = config.Redis.Port
	username = config.Redis.Username
	password = config.Redis.Password
	db := config.Redis.DBName
	err = InitRedisClient(host, port, username, password, db)
	if err != nil {
		log.Errorf("Connecting Redis:%v found error:%v", config.Redis, err)
	}
}

func InitScheme() {
	mysql.MigrateTables(MysqlClient)
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