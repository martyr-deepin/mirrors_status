package model

import "time"

type MirrorOperation struct {
	Id int `gorm:"primary_key"`
	CreateDate time.Time
	Username string
	OperationType string
	MirrorNames string
	CDNNodes string
	Duration string
	Operations []OperationData `gorm:"ForeignKey:MirrorOperationId"`
}

type OperationData struct {
	Id int `gorm:"primary_key"`
	MirrorName string
	MirrorOperationId int
}

const (
	ADD = "ADD"
	DELETE = "DELETE"
	UPDATE = "UPDATE"
	QUERY = "QUERY"
	SYNC = "SYNC"
)