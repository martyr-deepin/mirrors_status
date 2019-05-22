package service

import (
	"mirrors_status/pkg/modules/db/mysql"
	"mirrors_status/pkg/modules/model"
)

//func GetAllMirrors() []client.Result {
//	res, err := configs.GetInfluxdbClient().QueryDB("select * from mirrors")
//	if err != nil {
//		log.Errorf("Get mirrors found error:%v", err)
//	}
//	return res
//}
//
//func GetAllMirrorsCdn() []client.Result {
//	res, err := configs.GetInfluxdbClient().QueryDB("select * from mirrors_cdn")
//	if err != nil {
//		log.Errorf("Get mirrors_cdn found error:%v", err)
//	}
//	return res
//}
//
//func AddMirror(mirror model.MirrorsPoint) (err error) {
//	now := time.Now()
//	err = configs.GetInfluxdbClient().PushMirror(now, mirror)
//	if err != nil {
//		log.Errorf("Insert data found error:%v", err)
//	}
//	CreateOperation(model.MirrorOperation{
//		CreateDate: now,
//		OperationType: model.ADD,
//		MirrorId: mirror.Name,
//	})
//	return
//}
//
//func AddMirrorCdn(cdn model.MirrorsCdnPoint) (err error) {
//	now := time.Now()
//	err = configs.GetInfluxdbClient().PushMirrorCdn(now, cdn)
//	if err != nil {
//		log.Errorf("Insert data found error:%v", err)
//	}
//	CreateOperation(model.MirrorOperation{
//		CreateDate: now,
//		OperationType: model.ADD,
//		MirrorId: cdn.MirrorId,
//	})
//	return
//}


type MirrorService struct {
	client *mysql.Client
}

func NewMirrorService(client mysql.Client) *MirrorService {
	return &MirrorService{ client: &client }
}

func (m *MirrorService) CreateMirror(mirror model.Mirror) error {
	return m.client.DB.Table("mirrors").Create(&mirror).Error
}

func (m *MirrorService) DeleteMirror(id int) error {
	return m.client.DB.Table("mirrors").Delete(model.Mirror{}, "`index` = ?", id).Error
}

func (m *MirrorService) UpdateMirror(id int, mirror model.Mirror) error {
	return m.client.DB.Table("mirrors").Where("`index` = ?", id).Updates(&mirror, true).Error
}

func (m *MirrorService) GetAllMirrors() (mirrors []model.Mirror, err error) {
	err = m.client.DB.Table("mirrors").Find(&mirrors).Order("weight DESC").Error
	return
}

func (m *MirrorService) GetMirrorsByUpstream(upstream string) (mirrors []model.Mirror, err error) {
	err = m.client.DB.Table("mirrors").Where("upstream = ?", upstream).Scan(&mirrors).Error
	return
}