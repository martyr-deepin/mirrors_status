package model

import "time"

type MirrorOperation struct {
	Id            int `gorm:"primary_key" json:"id"`
	Index 		  string `json:"index"`
	CreateDate    time.Time `json:"create_date"`
	Username      string `json:"username"`
	OperationType string `json:"operation_type"`
	MirrorId      string `json:"mirror_id"`
	Status        MirrorOperationStatus `json:"status"`
	Msg string	`json:"msg"`
	
	Failed int `json:"failed"`
	Finish int `json:"finish"`
	Total int `json:"total"`
	Retry int `json:"retry"`
}

const (
	ADD    = "ADD"
	DELETE = "DELETE"
	SYNC   = "SYNC"
	SYNC_ALL = "SYNC ALL"
	SYNC_UPSTREAM = "SYNC UPSTREAM"
)
