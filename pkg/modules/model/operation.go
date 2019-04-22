package model

import "time"

type MirrorOperation struct {
	Id            int `gorm:"primary_key"`
	Index 		  string
	CreateDate    time.Time
	Username      string
	OperationType string
	MirrorId      string
	Status        string
}

const (
	ADD    = "ADD"
	DELETE = "DELETE"
	UPDATE = "UPDATE"
	QUERY  = "QUERY"
	SYNC   = "SYNC"
	SYNC_ALL = "SYNC ALL"
	UNCHECK = "UNCHECK"
	CHECKING = "CHECKING"
	FAILURE = "FAILURE"
	SUCCESS = "SUCCESS"
)
