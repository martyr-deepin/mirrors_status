package model

import "time"

type MirrorOperation struct {
	Id            int `gorm:"primary_key" json:"id"`
	Index 		  string `json:"index"`
	CreateDate    time.Time `json:"create_date"`
	Username      string `json:"username"`
	OperationType string `json:"operation_type"`
	MirrorId      string `json:"mirror_id"`
	Status        string `json:"status"`
	Msg string
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
