package controller

import (
	"github.com/gin-gonic/gin"
	"mirrors_status/internal/config"
	"mirrors_status/internal/log"
	"mirrors_status/pkg/db/service"
	"mirrors_status/pkg/model"
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