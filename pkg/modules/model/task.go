package model

import "time"

type Task struct {
	Id       int       `sql:",pk" json:"id"`
	Creator  string    `sql:",notnull" json:"creator"`
	Time     time.Time `sql:"default:now()" json:"time"`
	Upstream string    `sql:",notnull" json:"upstream"`
	IsOpen   bool      `sql:",notnull" json:"isOpen"`
	CITasks     []CITask `json:"ciTasks"`
	ContactMail string   `sql:",notnull" json:"contaciMail"`
	MirrorOperationIndex string `json:"mirror_operation_index"`
	MirrorOperation MirrorOperation `sql:"-" json:"mirror_operation"`
	MirrorSyncFinished bool `sql:"default:false" json:"mirror_sync_finished"`
	Status string `json:"status"`

	MirrorOperationStart time.Time `json:"mirror_operation_start"`
	MirrorOperationStatus string `json:"mirror_operation_status"`
}

const (
	MirrorOperationStatus_WAITING = "WAITING"
	MirrorOperationStatus_RUNNING = "RUNNING"
	MirrorOperationStatus_FINISHED = "FINISHED"
)

type Archive struct {
	tableName struct{} `sql:"archive"`

	Id       int              `sql:",pk" json:"id"`
	Approver string           `sql:",notnull" json:"approver"`
	Time     time.Time        `sql:"default:now()" json:"time"`
	Upstream string           `sql:",notnull" json:"upstream"`
	Mirrors  []MirrorSnapshot `sql:",notnull" json:"mirrors"`

	Task string `sql:",notnull" json:"tasks"`
}

type MirrorSnapshot struct {
	Mirror     string   `json:"mirror"`
	Completion *float64 `json:"completion"`

	CdnNodeCompletion []CdnNodeCompletion `json:"cdnNodeCompletion"`
}

type CdnNodeCompletion struct {
	NodeName   string  `json:"nodeName"`
	Completion float64 `json:"completion"`
}

type Extension struct {
	tableName struct{} `sql:"extension"`

	Index  int `sql:",pk"`
	Mirror string
	IsKey  bool `sql:"default:false" json:"isKey"`

	// Newly added
	LastUpdate time.Time `sql:"default:now()" json:"time"`
}

type CITask struct {
	Id          int    `json:"id"`
	TaskId      int    `json:"taskId"`
	JenkinsUrl  string `json:"jenkinsUrl"`
	Result      string `json:"result"`
	Time        string `sql:"default:now()" json:"time"`
	JobUrl      string `json:"jobUrl"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Token       string `json:"extension"`
}
