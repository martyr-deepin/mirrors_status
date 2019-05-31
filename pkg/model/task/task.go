package task

import (
	"errors"
	"github.com/jinzhu/gorm"
	configs "mirrors_status/internal/config"
	"mirrors_status/internal/log"
	"mirrors_status/pkg/db/client/mysql"
	checker2 "mirrors_status/pkg/mirror/checker"
	"mirrors_status/pkg/model/constants"
	"mirrors_status/pkg/model/mirror"
	"mirrors_status/pkg/model/operation"
	"mirrors_status/pkg/task/jenkins"
	"mirrors_status/pkg/utils"
	"time"
)

type Task struct {
	Id                   int                   `gorm:"primary_key" json:"id"`
	Creator              string                `gorm:"type:varchar(32)" json:"creator"`
	CreateAt             time.Time             `gorm:"default:now()" json:"create_at"`
	Upstream             string                `gorm:"type:varchar(64)" json:"upstream"`
	IsOpen               bool                  `gorm:"default:true" json:"is_open"`
	ContactMail          string                `gorm:"type:varchar(64)" json:"contact_mail"`
	MirrorOperationIndex string                `gorm:"type:varchar(64)" json:"mirror_operation_index"`
	MirrorSyncFinished   bool                  `gorm:"default:false" json:"mirror_sync_finished"`
	Status               constants.MirrorOperationStatus `gorm:"default:0" json:"status"`

	MirrorOperationStart  time.Time `gorm:"default:null" json:"mirror_operation_start"`
	MirrorOperationStatus constants.MirrorOperationStatus       `gorm:"type:tinyint(1)" json:"mirror_operation_status"`
	Progress              int       `gorm:"default:0" json:"progress"`
}

type CITask struct {
	Id          int       `gorm:"primary_key" json:"id"`
	TaskId      int       `json:"taskId"`
	JenkinsUrl  string    `json:"jenkinsUrl"`
	Result      string    `json:"result"`
	CreateAt    time.Time `gorm:"default:now()" json:"create_at"`
	JobUrl      string    `json:"jobUrl"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Token       string    `json:"token"`
	Retry       int       `gorm:"default:0" json:"retry"`
}

func (t Task) CreateTask() (Task, error) {
	openTasks, err := GetOpenTasks()
	if err != nil {
		return Task{}, err
	}
	for _, openTask := range openTasks {
		if openTask.Upstream == t.Upstream {
			return Task{}, errors.New("task already exists")
		}
	}
	err = mysql.NewMySQLClient().Table("tasks").Create(&t).Error
	log.Infof("%#v", t)
	return t, err
}

func CloseTask(id int) error {
	return mysql.NewMySQLClient().Table("tasks").Where("id = ?", id).Update("is_open", false).Error
}

func UpdateMirrorOperationIndex(id int, index string) error {
	return mysql.NewMySQLClient().Table("tasks").Where("id = ?", id).Update("mirror_operation_index", index).Error
}

func UpdateMirrorOperationStart(id int) error {
	return mysql.NewMySQLClient().Table("tasks").Where("id = ?", id).Update("mirror_operation_start", time.Now()).Error
}

func UpdateMirrorOperationStatus(id int, status constants.MirrorOperationStatus) error {
	return mysql.NewMySQLClient().Table("tasks").Where("id = ?", id).Update("mirror_operation_status", status).Error
}

func TaskProceed(id int) error {
	return mysql.NewMySQLClient().Table("tasks").Where("id = ?", id).Update("progress", gorm.Expr("progress + ?", 1)).Error
}

func (c CITask) CreateCiTask() error {
	return mysql.NewMySQLClient().Table("ci_tasks").Create(&c).Error
}

func GetTaskById(id int) (task Task, err error) {
	err = mysql.NewMySQLClient().Table("tasks").Where("id = ?", id).Scan(&task).Error
	return
}

func GetCiTaskById(id int) (task CITask, err error) {
	err = mysql.NewMySQLClient().Table("ci_tasks").Where("id = ?", id).Scan(&task).Error
	return
}

func GetCiTasksById(id int) (tasks []CITask, err error) {
	err = mysql.NewMySQLClient().Table("ci_tasks").Where("task_id = ?", id).Scan(&tasks).Error
	return
}

func GetOpenTasks() (tasks []Task, err error) {
	err = mysql.NewMySQLClient().Table("tasks").Where("is_open = ?", true).Scan(&tasks).Error
	return
}

func UpdateCiTaskJenkinsUrl(id int, url string) error {
	return mysql.NewMySQLClient().Table("ci_tasks").Where("id = ?", id).Update("jenkins_url", url).Error
}

func UpdateCiTaskResult(id int, result string) error {
	return mysql.NewMySQLClient().Table("ci_tasks").Where("id = ?", id).Update("result", result).Error
}

func CiTaskRetry(id int) error {
	return mysql.NewMySQLClient().Table("tasks").Where("id = ?", id).Update("retry", gorm.Expr("retry + ?", 1)).Error
}

func UpdateTaskStatus(id int, status constants.MirrorOperationStatus) error {
	return mysql.NewMySQLClient().Table("tasks").Where("id = ?", id).Update("status", status).Error
}

func (t Task) Handle(delTask func(string)) {
	log.Infof("Starting executing task:[%#v]", t)
	if t.Status == constants.STATUS_FAILURE || t.Status == constants.STATUS_FINISHED {
		delTask(t.Upstream)
		return
	}
	_ = UpdateTaskStatus(t.Id, constants.STATUS_RUNNING)
	ciTasks, err := GetCiTasksById(t.Id)
	if err != nil {
		log.Error("Get CI tasks by task id:[%d] found error:%#v", t.Id, err)
		mailAdmin(err)
		return
	}
	log.Infof("Length of tasks:%d", 2 * len(ciTasks))
	if t.Progress >= 2 * len(ciTasks) {
		_ = CloseTask(t.Id)
		delTask(t.Upstream)
		_ = UpdateTaskStatus(t.Id, constants.STATUS_FINISHED)
		return
	}
	for _, ciTask := range ciTasks {
		ciTask.Handle(t.Id, t.Upstream, t.ContactMail, t.Creator)
	}
}

func (c CITask) Handle(id int, upstream, contact, creator string) {
	log.Infof("Starting executing CI task:[%s]", c.Description)
	if c.Result != "" {
		return
	}
	jobInfo := &configs.JobInfo{
		Name:  c.Name,
		URL:   c.JobUrl,
		Token: c.Token,
	}
	params := make(map[string]string)
	params["UPSTREAM"] = upstream
	params["MIRRORS_API"] = c.JobUrl
	abort := make(chan bool)
	queueId, err := jenkins.TriggerBuild(jobInfo, params, abort)
	if err != nil {
		_ = utils.SendMail("warning", "<h3>Jenkins task trigger failed. Error messages are as follow:</h3><br><p>"+err.Error()+"</p><hr><p>Sent from <strong>Mirrors Management System</strong><p>", contact)
		_ = UpdateTaskStatus(id, constants.STATUS_FAILURE)
		_ = UpdateCiTaskResult(c.Id, "FAILURE")
		return
	}
	for {
		ct, _ := GetCiTaskById(c.Id)
		if ct.Result != "" {
			break
		}
		buildInfo, err := jenkins.GetBuildInfo(jobInfo, queueId)
		if err != nil {
			_ = utils.SendMail("warning", "<h3>Jenkins task build info fetch got error. Error messages are as follow:</h3><br><p>"+err.Error()+"</p><hr><p>Sent from <strong>Mirrors Management System</strong><p>", contact)
			_ = UpdateTaskStatus(id, constants.STATUS_FAILURE)
			_ = UpdateCiTaskResult(c.Id, "FAILURE")
			break
		}
		err = UpdateCiTaskJenkinsUrl(c.Id, buildInfo.URL)
		if err != nil {
			log.Errorf("Update ci task jenkins url found error:%#v", err)
			break
		}
		err = UpdateCiTaskResult(c.Id, buildInfo.Result)
		if err != nil {
			log.Errorf("Update ci task result url found error:%#v", err)
			break
		}
		if buildInfo.Result == "SUCCESS" {
			c.Result = "SUCCESS"
			err = TaskProceed(c.TaskId)
			if err != nil {
				log.Errorf("Update task progress url found error:%#v", err)
				break
			}
			err = UpdateMirrorOperationStatus(c.TaskId, constants.STATUS_WAITING)
			if err != nil {
				log.Errorf("Update mirror operation status url found error:%#v", err)
				break
			}
			delay := configs.NewServerConfig().Jenkins.Delay
			time.Sleep(time.Duration(delay) * time.Minute)
			HandleMirrorOperation(id, upstream, contact, creator)
		} else {
			time.Sleep(time.Second * 10)
			continue
		}
	}
}

func HandleMirrorOperation(id int, upstream, contact, creator string) {
	log.Infof("Starting executing mirror checking:[%s]", upstream)
	checker := checker2.NewCDNChecker()
	mirrors, err := mirror.GetMirrorsByUpstream(upstream)
	if err != nil {
		log.Errorf("Get mirrors by upstream:[%s] found error:%#v", upstream, err)
		mailContact(err, "get mirrors by upstream", contact)
		return
	}
	index := checker.CheckMirrors(mirrors, creator)
	err = UpdateMirrorOperationIndex(id, index)
	if err != nil {
		log.Errorf("Update mirror operation index:[%s] in task id:[%d] found error:%#v", index, id, err)
		mailContact(err, "Update mirror operation", contact)
		return
	}
	for {
		mirrorOperation, err := operation.GetOperationByIndex(index)
		if err != nil {
			if err != nil {
				log.Errorf("Get mirror operation by index:[%s] found error:%#v", index, err)
				mailContact(err, "Get mirror operation", contact)
				return
			}
		}
		if mirrorOperation.Status == constants.STATUS_FINISHED {
			err = UpdateMirrorOperationStatus(id, constants.STATUS_FINISHED)
			if err != nil {
				log.Errorf("Update mirror operation status by id:[%d] found error:%#v", id, err)
				mailContact(err, "Update mirror operation status", contact)
				return
			}
			err := TaskProceed(id)
			if err != nil {
				log.Errorf("Task proceed in id:[%d] found error:%#v", id, err)
				mailContact(err, "Task proceed", contact)
				return
			}
			return
		} else if mirrorOperation.Status == constants.STATUS_FAILURE {
			err = UpdateMirrorOperationStatus(id, constants.STATUS_FAILURE)
			if err != nil {
				log.Errorf("Update mirror operation status by id:[%d] found error:%#v", id, err)
				mailContact(err, "Update mirror operation status", contact)
				return
			}
		} else {
			time.Sleep(time.Second * 10)
			continue
		}
	}
}

func mailAdmin(err error) {
	_ = utils.SendMail("Alert", "<h3>System error. Error messages are as follow:</h3><br><p>"+err.Error()+"</p><hr><p>Sent from <strong>Mirrors Management System</strong><p>", configs.NewServerConfig().Http.AdminMail)
}

func mailContact(err error, op, contact string) {
	_ = utils.SendMail("Alert", "<h3>"+op+" found error. Error messages are as follow:</h3><br><p>"+err.Error()+"</p><hr><p>Sent from <strong>Mirrors Management System</strong><p>", contact)
}
