package mysql

import (
	"mirrors_status/pkg/model"
)

func MigrateTables(client *Client) {
	client.DB.Debug().AutoMigrate(model.MirrorOperation{}, model.Mirror{}, model.Task{}, model.CITask{})
}