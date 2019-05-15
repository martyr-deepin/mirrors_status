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
	Msg string	`json:"msg"`
	
	Failed int `json:"failed"`
	Finish int `json:"finish"`
	Total int `json:"total"`
}

const (
	ADD    = "ADD"
	DELETE = "DELETE"
	SYNC   = "SYNC"
	SYNC_ALL = "SYNC ALL"
	SYNC_UPSTREAM = "SYNC UPSTREAM"
	UNCHECK = "UNCHECK"
	CHECKING = "CHECKING"
	FAILURE = "FAILURE"
	SUCCESS = "SUCCESS"
)
