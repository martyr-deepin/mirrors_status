package service

import (
	"github.com/influxdata/influxdb/client/v2"
	"mirrors_status/pkg/config"
	"mirrors_status/pkg/log"
	"mirrors_status/pkg/modules/model"
	"time"
)

func GetAllMirrors() []client.Result {
	res, err := configs.GetInfluxdbClient().QueryDB("select * from mirrors")
	if err != nil {
		log.Errorf("Get mirrors found error:%v", err)
	}
	return res
}

func GetAllMirrorsCdn() []client.Result {
	res, err := configs.GetInfluxdbClient().QueryDB("select * from mirrors_cdn")
	if err != nil {
		log.Errorf("Get mirrors_cdn found error:%v", err)
	}
	return res
}

func AddMirror(mirror model.MirrorsPoint) (err error) {
	now := time.Now()
	err = configs.GetInfluxdbClient().PushMirror(now, mirror)
	if err != nil {
		log.Errorf("Insert data found error:%v", err)
	}
	CreateOperation(model.MirrorOperation{
		CreateDate: now,
		OperationType: model.ADD,
		MirrorId: mirror.Name,
	})
	return
}

func AddMirrorCdn(cdn model.MirrorsCdnPoint) (err error) {
	now := time.Now()
	err = configs.GetInfluxdbClient().PushMirrorCdn(now, cdn)
	if err != nil {
		log.Errorf("Insert data found error:%v", err)
	}
	CreateOperation(model.MirrorOperation{
		CreateDate: now,
		OperationType: model.ADD,
		MirrorId: cdn.MirrorId,
	})
	return
}

func TestApi(query string) []client.Result {
	data, err := configs.GetInfluxdbClient().QueryDB(query)
	if err != nil {
		log.Errorf("[%s] found error:%v", query, err)
	}
	return data
}