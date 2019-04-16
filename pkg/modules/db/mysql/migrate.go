package mysql

import (
	"mirrors_status/pkg/modules/model"
)

func MigrateTables(client *Client) {
	client.DB.Debug().AutoMigrate(model.MirrorOperation{}, model.OperationData{})
	client.DB.Model(model.OperationData{}).AddForeignKey("mirror_operation_id", "mirror_operations(id)", "CASCADE", "CASCADE")
}