package service

import (
	"github.com/jinzhu/gorm"
	"mirrors_status/pkg/db/client/mysql"
	"mirrors_status/pkg/model"
	"time"
)

type TaskService struct {
	client *mysql.Client
}

func NewTaskService(client *mysql.Client) *TaskService {
	return &TaskService{
		client: client,
	}
}

func (t *TaskService) CreateTask(task model.Task) (id int, err error ){
	err = t.client.DB.Table("tasks").Create(&task).Error
	return task.Id, err
}

func (t *TaskService) CloseTask(id int) error {
	return t.client.DB.Table("tasks").Where("id = ?", id).Update("is_open", false).Error
}

func (t *TaskService) UpdateMirrorOperationIndex(id int, index string) error  {
	return t.client.DB.Table("tasks").Where("id = ?", id).Update("mirror_operation_index", index).Error
}

func (t *TaskService) UpdateMirrorOperationStart(id int) error {
	return t.client.DB.Table("tasks").Where("id = ?", id).Update("mirror_operation_start", time.Now()).Error
}

func (t *TaskService) UpdateMirrorOperationStatus(id int, status model.MirrorOperationStatus) error {
	return t.client.DB.Table("tasks").Where("id = ?", id).Update("mirror_operation_status", status).Error
}

func (t *TaskService) TaskProceed(id int) error {
	return t.client.DB.Table("tasks").Where("id = ?", id).Update("progress", gorm.Expr("progress + ?", 1)).Error
}

func (t *TaskService) CreateCiTask(task model.CITask) error {
	return t.client.DB.Table("ci_tasks").Create(&task).Error
}

func (t *TaskService) GetTaskById(id int) (task model.Task, err error) {
	err = t.client.DB.Table("tasks").Where("id = ?", id).Scan(&task).Error
	return
}

func (t *TaskService) GetCiTasksById(id int) (tasks []model.CITask, err error) {
	err = t.client.DB.Table("ci_tasks").Where("task_id = ?", id).Scan(&tasks).Error
	return
}

func (t *TaskService) GetOpenTasks() (tasks []model.Task, err error) {
	err = t.client.DB.Table("tasks").Where("is_open = ?", true).Scan(&tasks).Error
	return
}

func (t *TaskService) UpdateCiTaskJenkinsUrl(id int, url string) error {
	return t.client.DB.Table("ci_tasks").Where("id = ?", id).Update("jenkins_url", url).Error
}

func (t *TaskService) UpdateCiTaskResult(id int, result string) error {
	return t.client.DB.Table("ci_tasks").Where("id = ?", id).Update("result", result).Error
}

func (t *TaskService) CiTaskRetry(id int) error {
	return t.client.DB.Table("tasks").Where("id = ?", id).Update("retry", gorm.Expr("retry + ?", 1)).Error
}

func (t *TaskService) UpdateTaskStatus(id int, status model.MirrorOperationStatus) error {
	return t.client.DB.Table("tasks").Where("id = ?", id).Update("status", status).Error
}