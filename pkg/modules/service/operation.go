package service

import (
	"github.com/jinzhu/gorm"
	"mirrors_status/pkg/config"
	"mirrors_status/pkg/log"
	"mirrors_status/pkg/modules/model"
)

func CreateOperation(operation model.MirrorOperation) {
	log.Infof("Inserting operation:%v", operation)
	configs.GetMySQLClient().DB.Create(&operation)
}

func GetOperationsByUsername(username string) []model.MirrorOperation {
	operations := []model.MirrorOperation{}
	configs.GetMySQLClient().DB.Where("username=?", username).Find(&operations)
	return operations
}

func GetOperationsByMirror(mirror string) []model.MirrorOperation {
	operations := []model.MirrorOperation{}
	configs.GetMySQLClient().DB.Where("mirror_id=?", mirror).Find(&operations)
	return operations
}

type MirrorOperations struct {
	Operations []*model.MirrorOperation
}

func GetOperationsByDateDesc() []model.MirrorOperation {
	var operations []model.MirrorOperation
	configs.GetMySQLClient().DB.Raw("select * from mirror_operations order by create_date desc").Scan(&operations)
	return operations
}

func DelOperationByUsername(username string) {
	configs.GetMySQLClient().DB.Debug().Table("mirror_operations").Where("username=?", username).Delete(&model.MirrorOperation{})
}

func UpdateMirrorStatus(index string, status string, msg string) {
	var operation model.MirrorOperation
	configs.GetMySQLClient().DB.Table("mirror_operations").Where("`index` = ?", index).Find(&operation)
	configs.GetMySQLClient().DB.Table("mirror_operations").Model(&operation).Update("status", status).Update("msg", msg)
}

func GetOperationByIndex(index string) (op model.MirrorOperation, err error) {
	err = configs.GetMySQLClient().DB.Table("mirror_operations").Where("`index` = ?", index).Find(&op).Error
	return
}

func SyncMirrorFinishOnce(index string) (err error) {
	err = configs.GetMySQLClient().DB.Table("mirror_operations").Where("`index` = ?", index).Update("finish", gorm.Expr("finish + ?", 1)).Error
	return
}

func SyncMirrorFailedOnce(index string) (err error) {
	err = configs.GetMySQLClient().DB.Table("mirror_operations").Where("`index` = ?", index).Update("failed", gorm.Expr("failed + ?", 1)).Error
	return
}

func UpdateMirrorTaskMsg(index, msg string) (err error) {
	err = configs.GetMySQLClient().DB.Table("mirror_operations").Where("`index` = ?", index).Update("msg", msg).Error
	return
}