package configs

import (
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"mirrors_status/pkg/log"
	"mirrors_status/pkg/modules/db/influxdb"
	"mirrors_status/pkg/modules/db/mysql"
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

type ServerConf struct {
	InfluxDB *InfluxDBConf `yaml:"influxdb"`
	MySQLDB *MySQLConf `yaml:"mysql"`
	Http     *HttpConf     `yaml:"http"`
	CdnChecker *CdnCheckerConf `yaml:"cdn-checker"`
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

var (
	influxdbClient *influxdb.Client
	mysqlClient *mysql.Client
)

func GetInfluxdbClient() *influxdb.Client {
	return influxdbClient
}

func GetMySQLClient() *mysql.Client {
	return mysqlClient
}

func InitInfluxdbClient(host string, port int, dbname, username, password string) error {
	log.Infof("trying connecting influxdb:%s:%s %s %s %s", host, port, dbname, username, password)
	influxdbClient = &influxdb.Client{
		Host:     host,
		Port:     port,
		DbName:   dbname,
		Username: username,
		Password: password,
	}
	return influxdbClient.NewInfluxClient()
}

func InitMySQLClient(host string, port int, dbname, username, password string) error {
	log.Infof("trying connecting MySQL:%s:%s %s %s %s", host, port, dbname, username, password)
	mysqlClient = &mysql.Client{
		Host:     host,
		Port:     port,
		DbName:   dbname,
		Username: username,
		Password: password,
	}
	return mysqlClient.NewMySQLClient()
}

func InitDB(config ServerConf) {
	host := config.InfluxDB.Host
	port := config.InfluxDB.Port
	dbName := config.InfluxDB.DBName
	username := config.InfluxDB.Username
	password := config.InfluxDB.Password

	err := InitInfluxdbClient(host, port, dbName, username, password)
	if err != nil {
		log.Errorf("Err connecting influxdb:%v", config.InfluxDB)
	}

	host = config.MySQLDB.Host
	port = config.MySQLDB.Port
	dbName = config.MySQLDB.DBName
	username = config.MySQLDB.Username
	password = config.MySQLDB.Password

	err = InitMySQLClient(host, port, dbName, username, password)
	if err != nil {
		log.Errorf("Err connecting MySQL:%v", config.MySQLDB)
	}
}

func InitScheme() {
	mysql.MigrateTables(mysqlClient)
}
