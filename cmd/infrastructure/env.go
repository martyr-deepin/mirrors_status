package infrastructure

import (
	"mirrors_status/pkg/business/cdn-checker"
	configs "mirrors_status/pkg/config"
	"mirrors_status/pkg/log"
	"mirrors_status/pkg/modules/db/influxdb"
	"mirrors_status/pkg/modules/db/mysql"
)

var (
	influxdbClient *influxdb.Client
	mysqlClient *mysql.Client
	cdnChecker *cdn_checker.CDNChecker
)

func GetInfluxdbClient() *influxdb.Client {
	return influxdbClient
}

func GetMySQLClient() *mysql.Client {
	return mysqlClient
}

func GetCdnChecker() *cdn_checker.CDNChecker {
	return cdnChecker
}

<<<<<<< HEAD
func InitInfluxdbClient(host, port, dbname, username, password string) error {
=======
func InitInfluxdbClient(host string, port int, dbname, username, password string) error {
>>>>>>> zhaojuwen/sync-check
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

<<<<<<< HEAD
func InitMySQLClient(host, port, dbname, username, password string) error {
=======
func InitMySQLClient(host string, port int, dbname, username, password string) error {
>>>>>>> zhaojuwen/sync-check
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

func InitDB(config configs.ServerConf) {
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

func InitCDNCkecker(conf configs.CdnCheckerConf) {
	log.Info("Initializing CDN checker")
	checkTool := cdn_checker.CheckTool{
		Conf: &conf,
	}
	cdnChecker = &cdn_checker.CDNChecker{
		InfluxClient: influxdbClient,
		CheckTool: checkTool,
	}
}