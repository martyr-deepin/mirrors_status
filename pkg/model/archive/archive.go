package archive

import (
	"encoding/json"
	"mirrors_status/internal/log"
	"mirrors_status/pkg/db/mysql"
	"mirrors_status/pkg/model/mirror"
	"mirrors_status/pkg/model/operation"
	"mirrors_status/pkg/model/task"
	"time"
)

type Archive struct {
	Id                int                       `gorm:"primary_key" json:"id"`
	TaskId            int                       `json:"task_id"`
	MirrorSnapshot    mirror.JSON               `sql:"type:json" json:"mirror_snapshot,omitempty"`
	TaskSnapshot      task.Task                 `sql:"-" json:"task_snapshot,omitempty"`
	CITaskSnapshot    []task.CITask             `sql:"-" json:"ci_task_snapshot"`
	OperationSnapshot operation.MirrorOperation `sql:"-" json:"operation_snapshot"`
	CreateAt          time.Time                 `gorm:"default:now()" json:"create_at"`
}

func (a Archive) CreateArchive() error {
	return mysql.NewMySQLClient().Table("archives").Create(&a).Error
}

func GetArchiveByTaskId(id int) (archive Archive, err error) {
	t, err := task.GetTaskById(id)
	if err != nil {
		log.Errorf("Get task by id:[%d] found error:%#v", id, err)
		return Archive{}, err
	}
	err = mysql.NewMySQLClient().Table("archives").Where("task_id = ?", id).Scan(&archive).Error
	if err != nil {
		log.Errorf("Get archive by task id:[%d] found error:%#v", id, err)
		return Archive{}, err
	}
	op, err := operation.GetOperationByIndex(t.MirrorOperationIndex)
	if err != nil {
		return Archive{}, err
	}
	ciTasks, err := task.GetCiTasksById(id)
	if err != nil {
		return Archive{}, err
	}
	return Archive{
		MirrorSnapshot: archive.MirrorSnapshot,
		TaskSnapshot:   t,
		OperationSnapshot: op,
		CITaskSnapshot: ciTasks,
	}, nil
}

func ArchiveTask(id int) (err error) {
	t, err := task.GetTaskById(id)
	if err != nil {
		return err
	}
	mirrors, err := mirror.GetMirrorsByUpstream(t.Upstream)
	if err != nil {
		return err
	}
	mirrorsSnapshot, err := json.Marshal(mirrors)
	if err != nil {
		return err
	}
	archive := Archive{
		TaskId: id,
		MirrorSnapshot: mirrorsSnapshot,
	}
	return archive.CreateArchive()
}

func GetAllArchives() (archives []Archive, err error) {
	list := []Archive{}
	err = mysql.NewMySQLClient().Table("archives").Scan(&list).Error
	if err != nil {
		return nil, err
	}
	for _, a := range list {
		archive, err := GetArchiveByTaskId(a.TaskId)
		if err != nil {
			return nil, err
		}
		archives = append(archives, archive)
	}
	return archives, nil
}