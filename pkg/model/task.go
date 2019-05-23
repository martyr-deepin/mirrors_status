package model

import (
	"time"
)

type Task struct {
	Id                   int                 `gorm:"primary_key" json:"id"`
	Creator              string              `gorm:"type:varchar(32)" json:"creator"`
	CreateAt             time.Time           `gorm:"default:now()" json:"create_at"`
	Upstream             string              `gorm:"type:varchar(64)" json:"upstream"`
	IsOpen               bool                `gorm:"default:true" json:"is_open"`
	ContactMail          string              `gorm:"type:varchar(64)" json:"contact_mail"`
	MirrorOperationIndex string              `gorm:"type:varchar(64)" json:"mirror_operation_index"`
	MirrorSyncFinished   bool                `gorm:"default:false" json:"mirror_sync_finished"`
	Status               MirrorOperationStatus `gorm:"default:0" json:"status"`

	MirrorOperationStart time.Time `gorm:"default:null" json:"mirror_operation_start"`
	MirrorOperationStatus int `gorm:"type:tinyint(1)" json:"mirror_operation_status"`
	Progress int `gorm:"default:0" json:"progress"`
}

type MirrorOperationStatus int

const (
	STATUS_FAILURE MirrorOperationStatus = -1
	STATUS_WAITING MirrorOperationStatus = iota
	STATUS_RUNNING MirrorOperationStatus = STATUS_WAITING + 1
	STATUS_FINISHED MirrorOperationStatus = STATUS_RUNNING + 1
)

type CITask struct {
	Id          int    `gorm:"primary_key" json:"id"`
	TaskId      int    `json:"taskId"`
	JenkinsUrl  string `json:"jenkinsUrl"`
	Result      string `json:"result"`
	CreateAt    time.Time `gorm:"default:now()" json:"create_at"`
	JobUrl      string `json:"jobUrl"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Token 		string `json:"token"`
	Retry 		int	   `gorm:"default:0" json:"retry"`
}
