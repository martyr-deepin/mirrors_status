package infrastructure

import (
	"mirrors_status/cmd/log"
	"mirrors_status/cmd/modules/db/influxdb"
)

var (
	influxdbClient *influxdb.Client
)

func GetInfluxdbClient() *influxdb.Client {
	return influxdbClient
}

func InitInfluxdbClient(host, port, dbname, username, password string) error {
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
