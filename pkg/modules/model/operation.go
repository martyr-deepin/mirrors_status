package model

import "time"

type MirrorOperation struct {
	Id int `gorm:"primary_key"`
	CreateDate time.Time
	Username string
	OperationType string
<<<<<<< HEAD
=======
	MirrorNames string
	CDNNodes string
>>>>>>> zhaojuwen/sync-check
	Operations []OperationData `gorm:"ForeignKey:MirrorOperationId"`
}

type OperationData struct {
	Id int `gorm:"primary_key"`
	MirrorName string
	MirrorOperationId int
}

