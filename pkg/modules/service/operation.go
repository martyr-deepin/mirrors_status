package service

import (
	"mirrors_status/pkg/log"
	"mirrors_status/pkg/modules/db/mysql"
	"mirrors_status/pkg/modules/model"
)

func CreateOperation(client *mysql.Client, operation model.MirrorOperation) {
	log.Infof("Inserting operation:%v", operation)
	client.DB.Create(&operation)
}

func GetOperationsByUsername(client *mysql.Client, username string) []model.MirrorOperation {
	operations := []model.MirrorOperation{}
	client.DB.Where("username=?", username).Find(&operations)
	return operations
}

type MirrorOperations struct {
	Operations []*model.MirrorOperation
}

func GetOperationsByDateDesc(client *mysql.Client) []model.MirrorOperation {
	var operations []model.MirrorOperation
	client.DB.Raw("select * from mirror_operations order by create_date desc").Scan(&operations)
	return operations
}

func DelOperationByUsername(client *mysql.Client, username string) {
	client.DB.Debug().Table("mirror_operations").Where("username=?", username).Delete(&model.MirrorOperation{})
}