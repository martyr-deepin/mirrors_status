package service

import (
	"github.com/influxdata/influxdb/client/v2"
	"mirrors_status/pkg/log"
	"mirrors_status/pkg/modules/db/influxdb"
	"mirrors_status/pkg/modules/db/mysql"
	"mirrors_status/pkg/modules/model"
	"time"
)

func GetAllMirrors(client *influxdb.Client) []client.Result {
	res, err := client.QueryDB("select * from mirrors")
	if err != nil {
		log.Errorf("Get mirrors found error:%v", err)
	}
	return res
}

func GetAllMirrorsCdn(client *influxdb.Client) []client.Result {
	res, err := client.QueryDB("select * from mirrors_cdn")
	if err != nil {
		log.Errorf("Get mirrors_cdn found error:%v", err)
	}
	return res
}

func AddMirror(mysqlClient *mysql.Client, influxClient *influxdb.Client, mirror model.MirrorsPoint) (err error) {
	now := time.Now()
	err = influxClient.PushMirror(now, mirror)
	if err != nil {
		log.Errorf("Insert data found error:%v", err)
	}
	CreateOperation(mysqlClient, model.MirrorOperation{
		CreateDate: now,
		OperationType: model.ADD,
		MirrorId: mirror.Name,
	})
	return
}

func AddMirrorCdn(mysqlClient *mysql.Client, client *influxdb.Client, cdn model.MirrorsCdnPoint) (err error) {
	now := time.Now()
	err = client.PushMirrorCdn(now, cdn)
	if err != nil {
		log.Errorf("Insert data found error:%v", err)
	}
	CreateOperation(mysqlClient, model.MirrorOperation{
		CreateDate: now,
		OperationType: model.ADD,
		MirrorId: cdn.MirrorId,
	})
	return
}

func TestApi(client *influxdb.Client, query string) []client.Result {
	data, err := client.QueryDB(query)
	if err != nil {
		log.Errorf("[%s] found error:%v", query, err)
	}
	return data
}