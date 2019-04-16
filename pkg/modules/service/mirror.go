package service

import (
	"github.com/influxdata/influxdb/client/v2"
	"mirrors_status/pkg/log"
	"mirrors_status/pkg/modules/db/influxdb"
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

func AddMirror(client *influxdb.Client, mirror model.MirrorsPoint) (err error) {
	err = client.PushMirror(time.Now(), mirror)
	if err != nil {
		log.Errorf("Insert data found error:%v", err)
	}
	return
}

func AddMirrorCdn(client *influxdb.Client, cdn model.MirrorsCdnPoint) (err error) {
	err = client.PushMirrorCdn(time.Now(), cdn)
	if err != nil {
		log.Errorf("Insert data found error:%v", err)
	}
	return
}

func TestApi(client *influxdb.Client, query string) []client.Result {
	data, err := client.QueryDB(query)
	if err != nil {
		log.Errorf("[%s] found error:%v", query, err)
	}
	return data
}