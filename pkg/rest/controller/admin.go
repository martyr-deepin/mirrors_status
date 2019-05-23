package controller

import (
	"github.com/gin-gonic/gin"
	"mirrors_status/internal/config"
	"mirrors_status/internal/log"
	"mirrors_status/pkg/db/service"
	"mirrors_status/pkg/mirror/checker"
	"mirrors_status/pkg/model"
	"mirrors_status/pkg/task/jenkins"
	"mirrors_status/pkg/utils"
	"net/http"
	"strconv"
)

func CreateMirror(c *gin.Context) {
	var mirror model.Mirror
	err := c.BindJSON(&mirror)
	if err != nil {
		log.Errorf("Parse json mirror:%v found error:%v", mirror, err)
		c.JSON(http.StatusBadRequest, utils.ErrorHelper(err, utils.PARAMETER_ERROR))
		return
	}

	err = service.NewMirrorService(*configs.GetMySQLClient()).CreateMirror(mirror)
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

	err = service.NewMirrorService(*configs.GetMySQLClient()).DeleteMirror(id)
	if err != nil {
		log.Errorf("Delete mirror:%d found error:%v", id, err)
		c.JSON(http.StatusBadRequest, utils.ErrorHelper(err, utils.DELETE_MIRROR_FAILED))
		return
	}
	c.JSON(http.StatusOK, utils.ResponseHelper(utils.SuccessResp()))
}

func UpdateMirror(c *gin.Context) {
	var mirror model.Mirror
	err := c.BindJSON(&mirror)
	if err != nil {
		log.Errorf("Parse json mirror:%v found error:%v", mirror, err)
		c.JSON(http.StatusBadRequest, utils.ErrorHelper(err, utils.PARAMETER_ERROR))
		return
	}

	pathId := c.Param("id")
	id, err := strconv.Atoi(pathId)
	if err != nil {
		log.Errorf("Parse path param id:%d found error:%v", pathId, err)
		c.JSON(http.StatusBadRequest, utils.ErrorHelper(err, utils.PARAMETER_ERROR))
		return
	}

	err = service.NewMirrorService(*configs.GetMySQLClient()).UpdateMirror(id, mirror)
	if err != nil {
		log.Errorf("Update mirror:%v found error:%v", mirror, err)
		c.JSON(http.StatusBadRequest, utils.ErrorHelper(err, utils.UPDATE_MIRROR_FAILED))
		return
	}
	c.JSON(http.StatusOK, utils.ResponseHelper(utils.SuccessResp()))
}

func CreateTask(c *gin.Context) {
	var task model.Task
	err := c.BindJSON(&task)
	log.Infof("Creating task:%#v", task)
	if err != nil {
		log.Errorf("Parse json mirror:%v found error:%v", task, err)
		c.JSON(http.StatusBadRequest, utils.ErrorHelper(err, utils.PARAMETER_ERROR))
		return
	}
	mirrorService := service.NewMirrorService(*configs.GetMySQLClient())
	taskService := service.NewTaskService(configs.GetMySQLClient())
	mirrors, err := mirrorService.GetMirrorsByUpstream(task.Upstream)
	if err != nil {
		log.Errorf("Get mirrors by upstream in [%#v] found error:%v", mirrors, err)
		c.JSON(http.StatusBadRequest, utils.ErrorHelper(err, utils.PARAMETER_ERROR))
		return
	}
	id, err := taskService.CreateTask(task)
	if err != nil {
		log.Errorf("Create task [%#v] found error:%v", task, err)
		c.JSON(http.StatusBadRequest, utils.ErrorHelper(err, utils.CREATE_TASK_FAILED))
		return
	}
	jobs := jenkins.GetMirrorJobsByUpstream(task.Upstream)
	for _, job := range jobs {
		ciTask := model.CITask{
			TaskId: id,
			JobUrl: job.URL,
			Name: job.Name,
			Description: job.Description,
			Token: job.Token,
		}
		err := taskService.CreateCiTask(ciTask)
		if err != nil {
			log.Errorf("Create ci task [%#v] found error:%v", ciTask, err)
			c.JSON(http.StatusBadRequest, utils.ErrorHelper(err, utils.CREATE_CITASK_FAILED))
			return
		}
	}
	c.JSON(http.StatusOK, utils.ResponseHelper(utils.SuccessResp()))
}

type TaskDetailResp struct {
	Task model.Task `json:"task"`
	CITasks []model.CITask `json:"ci_tasks"`
	MirrorOperation model.MirrorOperation `json:"mirror_operation,omitempty"`
}

func GetTaskById(c *gin.Context) {
	pathId := c.Param("id")
	id, err := strconv.Atoi(pathId)
	if err != nil {
		log.Errorf("Parse path param id:%d found error:%v", pathId, err)
		c.JSON(http.StatusBadRequest, utils.ErrorHelper(err, utils.PARAMETER_ERROR))
		return
	}

	taskService := service.NewTaskService(configs.GetMySQLClient())
	task, err := taskService.GetTaskById(id)
	if err != nil {
		log.Errorf("Get task by id:[%d] found error:%#v", id, err)
		c.JSON(http.StatusNoContent, utils.ErrorHelper(err, utils.FETCH_DATA_ERROR))
		return
	}
	var mirrorOperation model.MirrorOperation
	if task.MirrorOperationIndex != "" {
		mirrorOperation, err = service.GetOperationByIndex(task.MirrorOperationIndex)
	}
	ciTasks, err := taskService.GetCiTasksById(id)
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
	taskService := service.NewTaskService(configs.GetMySQLClient())
	var openTasks OpenTasksResp
	tasks, err := taskService.GetOpenTasks()
	if err != nil {
		log.Errorf("Get open tasks found error:%#v", err)
		c.JSON(http.StatusNoContent, utils.ErrorHelper(err, utils.FETCH_DATA_ERROR))
		return
	}
	for _, task := range tasks {
		var mirrorOperation model.MirrorOperation
		if task.MirrorOperationIndex != "" {
			mirrorOperation, err = service.GetOperationByIndex(task.MirrorOperationIndex)
		}
		ciTasks, err := taskService.GetCiTasksById(task.Id)
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
	mirrors, err := service.NewMirrorService(*configs.GetMySQLClient()).GetMirrorsByUpstream(upstream)
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
	mirrorService := service.NewMirrorService(*configs.GetMySQLClient())
	mirrors, err := mirrorService.MultiGetByIndices(mirrorsReq.Mirrors)
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
	taskService := service.NewTaskService(configs.GetMySQLClient())
	err = taskService.CloseTask(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.ErrorHelper(err, utils.UPDATE_TASK_STATUS_FAILED))
		return
	}
	c.JSON(http.StatusOK, utils.ResponseHelper(utils.SuccessResp()))
}