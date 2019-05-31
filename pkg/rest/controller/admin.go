package controller

import (
	"github.com/gin-gonic/gin"
	configs "mirrors_status/internal/config"
	"mirrors_status/internal/log"
	"mirrors_status/pkg/mirror/checker"
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
		c.JSON(http.StatusBadRequest, utils.ErrorHelper(err, utils.PARAMETER_ERROR))
		return
	}

	err = mirror.CreateMirror()
	if err != nil {
		log.Errorf("Create mirror:%v found error:%v", mirror, err)
		c.JSON(http.StatusBadRequest, utils.ErrorHelper(err, utils.CREATE_MIRROR_FAILED))
		return
	}
	c.JSON(http.StatusOK, utils.ResponseHelper(utils.SuccessResp()))
}

func DeleteMirror(c *gin.Context) {
	pathId := c.Param("id")
	id, err := strconv.Atoi(pathId)
	if err != nil {
		log.Errorf("Parse path param id:%d found error:%v", pathId, err)
		c.JSON(http.StatusBadRequest, utils.ErrorHelper(err, utils.PARAMETER_ERROR))
		return
	}

	err = mirror2.DeleteMirror(id)
	if err != nil {
		log.Errorf("Delete mirror:%d found error:%v", id, err)
		c.JSON(http.StatusBadRequest, utils.ErrorHelper(err, utils.DELETE_MIRROR_FAILED))
		return
	}
	c.JSON(http.StatusOK, utils.ResponseHelper(utils.SuccessResp()))
}

func UpdateMirror(c *gin.Context) {
	var mirror mirror2.Mirror
	err := c.BindJSON(&mirror)
	if err != nil {
		log.Errorf("Parse json mirror:%v found error:%v", mirror, err)
		c.JSON(http.StatusBadRequest, utils.ErrorHelper(err, utils.PARAMETER_ERROR))
		return
	}

	err = mirror.UpdateMirror()
	if err != nil {
		log.Errorf("Update mirror:%v found error:%v", mirror, err)
		c.JSON(http.StatusBadRequest, utils.ErrorHelper(err, utils.UPDATE_MIRROR_FAILED))
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
		c.JSON(http.StatusBadRequest, utils.ErrorHelper(err, utils.PARAMETER_ERROR))
		return
	}
	mirrors, err := mirror2.GetMirrorsByUpstream(task.Upstream)
	if err != nil {
		log.Errorf("Get mirrors by upstream in [%#v] found error:%v", mirrors, err)
		c.JSON(http.StatusBadRequest, utils.ErrorHelper(err, utils.PARAMETER_ERROR))
		return
	}
	t, err := task.CreateTask()
	if err != nil {
		log.Errorf("Create task [%#v] found error:%v", task, err)
		c.JSON(http.StatusBadRequest, utils.ErrorHelper(err, utils.CREATE_TASK_FAILED))
		return
	}
	jobs := jenkins.GetMirrorJobsByUpstream(task.Upstream)
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
			c.JSON(http.StatusBadRequest, utils.ErrorHelper(err, utils.CREATE_CITASK_FAILED))
			return
		}
	}
	//go task.Handle()
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
		c.JSON(http.StatusBadRequest, utils.ErrorHelper(err, utils.PARAMETER_ERROR))
		return
	}

	task, err := task2.GetTaskById(id)
	if err != nil {
		log.Errorf("Get task by id:[%d] found error:%#v", id, err)
		c.JSON(http.StatusNoContent, utils.ErrorHelper(err, utils.FETCH_DATA_ERROR))
		return
	}
	var mirrorOperation operation.MirrorOperation
	if task.MirrorOperationIndex != "" {
		mirrorOperation, err = operation.GetOperationByIndex(task.MirrorOperationIndex)
	}
	ciTasks, err := task2.GetCiTasksById(id)
	if err != nil {
		log.Errorf("Get ci tasks by id:[%d] found error:%#v", id, err)
		c.JSON(http.StatusNoContent, utils.ErrorHelper(err, utils.FETCH_DATA_ERROR))
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
		c.JSON(http.StatusNoContent, utils.ErrorHelper(err, utils.FETCH_DATA_ERROR))
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
			c.JSON(http.StatusNoContent, utils.ErrorHelper(err, utils.FETCH_DATA_ERROR))
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
		c.JSON(http.StatusBadRequest, utils.ErrorHelper(err, utils.PARSE_COOKIE_ERROR))
		return
	}
	checker := checker.NewCDNChecker(configs.NewServerConfig().CdnChecker)
	index := checker.CheckAllMirrors(username)
	log.Infof("Check all mirror tasks established index:[%s]", index)
	c.JSON(http.StatusOK, utils.ResponseHelper(utils.SuccessResp()))
}

func CheckMirrorsByUpstream(c *gin.Context) {
	username, err := c.Cookie("username")
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.ErrorHelper(err, utils.PARSE_COOKIE_ERROR))
		return
	}
	upstream := c.Param("upstream")
	mirrors, err := mirror2.GetMirrorsByUpstream(upstream)
	if err != nil {
		log.Errorf("Get mirrors by upstream:%s found error:%v", upstream, err)
		c.JSON(http.StatusBadRequest, utils.ErrorHelper(err, utils.FETCH_DATA_ERROR))
		return
	}
	checker := checker.NewCDNChecker(configs.NewServerConfig().CdnChecker)
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
		c.JSON(http.StatusBadRequest, utils.ErrorHelper(err, utils.PARSE_COOKIE_ERROR))
		return
	}
	var mirrorsReq MirrorsReq
	err = c.BindJSON(&mirrorsReq)
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.ErrorHelper(err, utils.PARAMETER_ERROR))
		return
	}
	mirrors, err := mirror2.GetMirrorsByIndices(mirrorsReq.Mirrors)
	if err != nil {
		log.Errorf("Get mirrors by indices:%#v found error:%v", mirrorsReq, err)
		c.JSON(http.StatusBadRequest, utils.ErrorHelper(err, utils.FETCH_DATA_ERROR))
		return
	}
	checker := checker.NewCDNChecker(configs.NewServerConfig().CdnChecker)
	index := checker.CheckMirrors(mirrors, username)
	log.Infof("Multi check mirror by indices:%#v established index:%s", mirrorsReq.Mirrors, index)
	c.JSON(http.StatusOK, utils.ResponseHelper(utils.SuccessResp()))
}

func AbortTask(c *gin.Context) {
	pathId := c.Param("id")
	id, err := strconv.Atoi(pathId)
	if err != nil {
		log.Errorf("Parse path param id:%d found error:%v", pathId, err)
		c.JSON(http.StatusBadRequest, utils.ErrorHelper(err, utils.PARAMETER_ERROR))
		return
	}
	err = task2.CloseTask(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.ErrorHelper(err, utils.UPDATE_TASK_STATUS_FAILED))
		return
	}
	c.JSON(http.StatusOK, utils.ResponseHelper(utils.SuccessResp()))
}