package model

import "time"

type MirrorOperation struct {
	Id int `gorm:"primary_key"`
	CreateDate time.Time
	Username string
	OperationType string
	MirrorId string
}

const (
	ADD = "ADD"
	DELETE = "DELETE"
	UPDATE = "UPDATE"
	QUERY = "QUERY"
	SYNC = "SYNC"
)