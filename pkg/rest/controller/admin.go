package controller

import (
	"github.com/gin-gonic/gin"
	"mirrors_status/internal/config"
	"mirrors_status/internal/log"
	"mirrors_status/pkg/db/redis"
	"mirrors_status/pkg/mirror/checker"
	"mirrors_status/pkg/model/archive"
	"mirrors_status/pkg/model/constants"
	mirror2 "mirrors_status/pkg/model/mirror"
	"mirrors_status/pkg/model/operation"
	task2 "mirrors_status/pkg/model/task"
	"mirrors_status/pkg/task/jenkins"
	"mirrors_status/pkg/utils"
	"net/http"
	"strconv"
)

func CreateMirror(c *gin.Context) {
	var mirror mirror2.Mirror
	err := c.BindJSON(&mirror)
	if err != nil {
		log.Errorf("Parse json mirror:%v found error:%v", mirror, err)
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.PARAMETER_ERROR))
		return
	}

	err = mirror.CreateMirror()
	if err != nil {
		log.Errorf("Create mirror:%v found error:%v", mirror, err)
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.CREATE_MIRROR_FAILED))
		return
	}
	c.JSON(http.StatusOK, utils.ResponseHelper(utils.SuccessResp()))
}

func DeleteMirror(c *gin.Context) {
	pathId := c.Param("id")
	id, err := strconv.Atoi(pathId)
	if err != nil {
		log.Errorf("Parse path param id:%d found error:%v", pathId, err)
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.PARAMETER_ERROR))
		return
	}

	err = mirror2.DeleteMirror(id)
	if err != nil {
		log.Errorf("Delete mirror:%d found error:%v", id, err)
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.DELETE_MIRROR_FAILED))
		return
	}
	c.JSON(http.StatusOK, utils.ResponseHelper(utils.SuccessResp()))
}

func UpdateMirror(c *gin.Context) {
	var mirror mirror2.Mirror
	err := c.BindJSON(&mirror)
	if err != nil {
		log.Errorf("Parse json mirror:%v found error:%v", mirror, err)
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.PARAMETER_ERROR))
		return
	}

	err = mirror.UpdateMirror()
	if err != nil {
		log.Errorf("Update mirror:%v found error:%v", mirror, err)
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.UPDATE_MIRROR_FAILED))
		return
	}
	c.JSON(http.StatusOK, utils.ResponseHelper(utils.SuccessResp()))
}

func CreateTask(c *gin.Context) {
	var task task2.Task
	err := c.BindJSON(&task)
	log.Infof("Creating task:%#v", task)
	if err != nil {
		log.Errorf("Parse json mirror:%v found error:%v", task, err)
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.PARAMETER_ERROR))
		return
	}
	mirrors, err := mirror2.GetMirrorsByUpstream(task.Upstream)
	if err != nil {
		log.Errorf("Get mirrors by upstream in [%#v] found error:%v", mirrors, err)
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.PARAMETER_ERROR))
		return
	}
	t, err := task.CreateTask()
	if err != nil {
		log.Errorf("Create task [%#v] found error:%v", task, err)
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.CREATE_TASK_FAILED))
		return
	}
	_ = archive.ArchiveTask(t.Id)
	var jobs configs.JobInfoList
	if task.Type == task2.PublishType {
		jobs = jenkins.GetPublishJobsByUpstream(task.Upstream)
	} else {
		jobs = jenkins.GetMirrorJobsByUpstream(task.Upstream)
	}
	for _, job := range jobs {
		err = task2.CITask{
			TaskId: t.Id,
			JobUrl: job.URL,
			Name: job.Name,
			Description: job.Description,
			Token: job.Token,
		}.CreateCiTask()
		if err != nil {
			log.Errorf("Create ci task found error:%v", err)
			c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.CREATE_CITASK_FAILED))
			return
		}
	}
	go t.Handle(func(upstream string) {
		log.Infof("Start executing task:[%s]", upstream)
	})
	c.JSON(http.StatusOK, utils.ResponseHelper(utils.SuccessResp()))
}

type TaskDetailResp struct {
	Task            task2.Task                `json:"task"`
	CITasks         []task2.CITask            `json:"ci_tasks"`
	MirrorOperation operation.MirrorOperation `json:"mirror_operation,omitempty"`
}

func GetTaskById(c *gin.Context) {
	pathId := c.Param("id")
	id, err := strconv.Atoi(pathId)
	if err != nil {
		log.Errorf("Parse path param id:%d found error:%v", pathId, err)
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.PARAMETER_ERROR))
		return
	}

	task, err := task2.GetTaskById(id)
	if err != nil {
		log.Errorf("Get task by id:[%d] found error:%#v", id, err)
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.FETCH_DATA_ERROR))
		return
	}
	var mirrorOperation operation.MirrorOperation
	if task.MirrorOperationIndex != "" {
		mirrorOperation, err = operation.GetOperationByIndex(task.MirrorOperationIndex)
	}
	ciTasks, err := task2.GetCiTasksById(id)
	if err != nil {
		log.Errorf("Get ci tasks by id:[%d] found error:%#v", id, err)
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.FETCH_DATA_ERROR))
		return
	}
	for _, ciTask := range ciTasks {
		ciTask.Token = ""
	}
	c.JSON(http.StatusOK, utils.ResponseHelper(utils.SetData("task", TaskDetailResp{
		Task: task,
		CITasks: ciTasks,
		MirrorOperation: mirrorOperation,
	})))
}

type OpenTasksResp []TaskDetailResp

func GetIOpenTasks(c *gin.Context) {
	var openTasks OpenTasksResp
	tasks, err := task2.GetOpenTasks()
	if err != nil {
		log.Errorf("Get open tasks found error:%#v", err)
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.FETCH_DATA_ERROR))
		return
	}
	for _, task := range tasks {
		var mirrorOperation operation.MirrorOperation
		if task.MirrorOperationIndex != "" {
			mirrorOperation, err = operation.GetOperationByIndex(task.MirrorOperationIndex)
		}
		ciTasks, err := task2.GetCiTasksById(task.Id)
		if err != nil {
			log.Errorf("Get ci tasks by id:[%d] found error:%#v", task.Id, err)
			c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.FETCH_DATA_ERROR))
			return
		}
		for _, ciTask := range ciTasks {
			ciTask.Token = ""
		}
		openTasks = append(openTasks, TaskDetailResp{
			Task: task,
			CITasks: ciTasks,
			MirrorOperation: mirrorOperation,
		})
	}
	c.JSON(http.StatusOK, utils.ResponseHelper(utils.SetData("tasks", openTasks)))
}

func CheckAllMirrors(c *gin.Context) {
	username, err := c.Cookie("username")
	if err != nil {
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.PARSE_COOKIE_ERROR))
		return
	}
	checker := checker.NewCDNChecker()
	index := checker.CheckAllMirrors(username)
	log.Infof("Check all mirror tasks established index:[%s]", index)
	c.JSON(http.StatusOK, utils.ResponseHelper(utils.SuccessResp()))
}

func CheckMirrorsByUpstream(c *gin.Context) {
	username, err := c.Cookie("username")
	if err != nil {
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.PARSE_COOKIE_ERROR))
		return
	}
	upstream := c.Param("upstream")
	mirrors, err := mirror2.GetMirrorsByUpstream(upstream)
	if err != nil {
		log.Errorf("Get mirrors by upstream:%s found error:%v", upstream, err)
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.FETCH_DATA_ERROR))
		return
	}
	checker := checker.NewCDNChecker()
	index := checker.CheckMirrors(mirrors, username)
	log.Infof("Check all mirror tasks established index:[%s]", index)
	c.JSON(http.StatusOK, utils.ResponseHelper(utils.SuccessResp()))
}

type MirrorsReq struct {
	Mirrors []int `json:"mirrors"`
}

func CheckMirrors(c *gin.Context) {
	username, err := c.Cookie("username")
	if err != nil {
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.PARSE_COOKIE_ERROR))
		return
	}
	var mirrorsReq MirrorsReq
	err = c.BindJSON(&mirrorsReq)
	if err != nil {
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.PARAMETER_ERROR))
		return
	}
	mirrors, err := mirror2.GetMirrorsByIndices(mirrorsReq.Mirrors)
	if err != nil {
		log.Errorf("Get mirrors by indices:%#v found error:%v", mirrorsReq, err)
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.FETCH_DATA_ERROR))
		return
	}
	checker := checker.NewCDNChecker()
	index := checker.CheckMirrors(mirrors, username)
	log.Infof("Multi check mirror by indices:%#v established index:%s", mirrorsReq.Mirrors, index)
	c.JSON(http.StatusOK, utils.ResponseHelper(utils.SuccessResp()))
}

func AbortTask(c *gin.Context) {
	pathId := c.Param("id")
	id, err := strconv.Atoi(pathId)
	if err != nil {
		log.Errorf("Parse path param id:%d found error:%v", pathId, err)
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.PARAMETER_ERROR))
		return
	}
	err = task2.AbortTask(id)
	if err != nil {
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.UPDATE_TASK_STATUS_FAILED))
		return
	}
	c.JSON(http.StatusOK, utils.ResponseHelper(utils.SuccessResp()))
}

func UpdateTaskStatus(c *gin.Context) {
	pathId := c.Param("id")
	id, err := strconv.Atoi(pathId)
	if err != nil {
		log.Errorf("Parse path param id:%d found error:%v", pathId, err)
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.PARAMETER_ERROR))
		return
	}
	pathStatus := c.Param("status")
	status, err := strconv.Atoi(pathStatus)
	if err != nil {
		log.Errorf("Parse path param id:%d found error:%v", pathId, err)
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.PARAMETER_ERROR))
		return
	}
	err = task2.UpdateTaskStatus(id, constants.MirrorOperationStatus(status))
	if err != nil {
		log.Errorf("Update task status by task id:%d found error:%v", pathId, err)
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.UPDATE_TASK_STATUS_FAILED))
		return
	}
	err = task2.CloseTask(id)
	if err != nil {
		log.Errorf("Update task status by task id:%d found error:%v", pathId, err)
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.UPDATE_TASK_STATUS_FAILED))
		return
	}
	c.JSON(http.StatusOK, utils.ResponseHelper(utils.SuccessResp()))
}

func GetArchiveByTaskId(c *gin.Context) {
	pathId := c.Param("id")
	id, err := strconv.Atoi(pathId)
	if err != nil {
		log.Errorf("Parse path param id:%d found error:%v", pathId, err)
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.PARAMETER_ERROR))
		return
	}
	archive, err := archive.GetArchiveByTaskId(id)
	if err != nil {
		log.Errorf("Get archive by id:[%d] found error:%#v", id, err)
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.FETCH_DATA_ERROR))
		return
	}
	c.JSON(http.StatusOK, utils.ResponseHelper(utils.SetData("archive", archive)))
}

func GetAllArchives(c *gin.Context) {
	page := c.Query("page")
	limit := c.Query("limit")
	if len(page) == 0 && len(limit) == 0 {
		GetArchivesCount(c)
		return
	}
	if len(page) != 0 && len(limit) != 0 {
		archivePage, err1 := strconv.Atoi(page)
		archiveLimit, err2 := strconv.Atoi(limit)
		if err1 != nil || err2 != nil {
			c.JSON(http.StatusOK, utils.ErrorHelper(nil, utils.PARAMETER_ERROR))
			return
		}
		GetPagedArchives(c, archivePage, archiveLimit)
		return
	}
	archives, err := archive.GetAllArchives()
	if err != nil {
		log.Errorf("Get archives found error:%#v", err)
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.FETCH_DATA_ERROR))
		return
	}
	c.JSON(http.StatusOK, utils.ResponseHelper(utils.SetData("archives", archives)))
}

func GetArchivesCount(c *gin.Context) {
	count, err := archive.GetArchivesCount()
	if err != nil {
		log.Errorf("Get archive page count found error:%#v", err)
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.FETCH_DATA_ERROR))
		return
	}
	c.JSON(http.StatusOK, utils.ResponseHelper(utils.SetData("count", count)))
}

func GetPagedArchives(c *gin.Context, page, size int) {
	archives, err := archive.GetPagedArchives(page, size)
	if err != nil {
		log.Errorf("Get archives found error:%#v", err)
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.FETCH_DATA_ERROR))
		return
	}
	c.JSON(http.StatusOK, utils.ResponseHelper(utils.SetData("archives", archives)))
}

func Logout(c *gin.Context) {
	username, err := c.Cookie("username")
	if err != nil {
		log.Errorf("Get cookie username found error:%#v", err)
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.GET_COOKIE_ERROR))
		return
	}
	sessionId, err := c.Cookie("sessionId")
	if err != nil {
		log.Errorf("Get cookie sessionId found error:%#v", err)
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.GET_COOKIE_ERROR))
		return
	}
	redisSessionId, err := redis.Get(username + "-session-id")
	if err != nil {
		log.Errorf("Get redis sessionId found error:%#v", err)
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.FETCH_DATA_ERROR))
		return
	}
	if sessionId != redisSessionId {
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.USER_NOT_LOGIN))
		return
	}
	err = redis.Del(username + "-session-id")
	if err != nil {
		log.Errorf("Del redis sessionId found error:%#v", err)
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.FETCH_DATA_ERROR))
		return
	}
	c.JSON(http.StatusOK, utils.ResponseHelper(utils.SuccessResp()))
}

func GetLoginStatus(c *gin.Context) {
	username, err := c.Cookie("username")
	if err != nil {
		log.Errorf("Get cookie username found error:%#v", err)
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.GET_COOKIE_ERROR))
		return
	}
	sessionId, err := c.Cookie("sessionId")
	if err != nil {
		log.Errorf("Get cookie sessionId found error:%#v", err)
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.GET_COOKIE_ERROR))
		return
	}
	redisSessionId, err := redis.Get(username + "-session-id")
	if err != nil {
		log.Errorf("Get redis sessionId found error:%#v", err)
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.FETCH_DATA_ERROR))
		return
	}
	if sessionId != redisSessionId {
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.USER_NOT_LOGIN))
		return
	}
	c.JSON(http.StatusOK, utils.ResponseHelper(utils.SetData("username", username)))
}

type MailReq struct {
	Mails []string `json:"mails"`
	Subject string `json:"subject"`
	Content string `json:"content"`
}

func SendMail(c *gin.Context) {
	var mailReq MailReq
	err := c.BindJSON(&mailReq)
	if err != nil {
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.PARAMETER_ERROR))
		return
	}
	err = utils.SendMail(mailReq.Subject, mailReq.Content, mailReq.Mails...)
	if err != nil {
		c.JSON(http.StatusOK, utils.ErrorHelper(err, utils.UNKNOWN_ERROR))
		return
	}
	c.JSON(http.StatusOK, utils.ResponseHelper(utils.SuccessResp()))
}