package service

import (
	"mirrors_status/pkg/modules/db/mysql"
	"mirrors_status/pkg/modules/model"
)

func CreateOperation(client *mysql.Client, operation model.MirrorOperation) {
	client.DB.Create(operation)
}

func GetOperationsByUsername(client *mysql.Client, username string) []model.MirrorOperation {
	operations := []model.MirrorOperation{}
	client.DB.Where("username=?", username).Find(&operations)
	return operations
}

func GetOperationsByDateDesc(client *mysql.Client) []model.MirrorOperation {
	operations := []model.MirrorOperation{}
	client.DB.Exec("select * from mirror_operations order by create_date desc", operations)
	return operations
}

func DelOperationByUsername(client *mysql.Client, username string) {
	client.DB.Debug().Table("mirror_operations").Where("username=?", username).Delete(&model.MirrorOperation{})
}