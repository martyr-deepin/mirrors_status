package mysql

import (
	"mirrors_status/pkg/modules/model"
)

func MigrateTables(client *Client) {
	client.DB.Debug().AutoMigrate(model.MirrorOperation{}, model.Mirror{}, )
}