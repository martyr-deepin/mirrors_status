package controller

import (
	"github.com/gin-gonic/gin"
	"mirrors_status/internal/config"
	"mirrors_status/internal/log"
	"mirrors_status/pkg/db/service"
	"mirrors_status/pkg/utils"
	"net/http"
)

func GetAllMirrors(ctx *gin.Context) {
	mirrors, err := service.NewMirrorService(*configs.GetMySQLClient()).GetAllMirrors()
	if err != nil {
		log.Errorf("Get all mirrors found error:%#v", err)
		ctx.JSON(http.StatusNoContent, utils.ErrorHelper(err, utils.FETCH_DATA_ERROR))
		return
	}
	ctx.JSON(http.StatusOK, utils.ResponseHelper(utils.SetData("mirrors", mirrors)))
}

func GetMirrorsByUpstream(c *gin.Context) {
	upstream := c.Param("upstream")
	mirrors, err := service.NewMirrorService(*configs.GetMySQLClient()).GetMirrorsByUpstream(upstream)
	if err != nil {
		log.Errorf("Get mirrors by upstream:%s found error:%v", upstream, err)
		c.JSON(http.StatusBadRequest, utils.ErrorHelper(err, utils.FETCH_DATA_ERROR))
		return
	}
	c.JSON(http.StatusOK, utils.ResponseHelper(utils.SetData("mirrors", mirrors)))
}

func GetAllUpstreams(c *gin.Context) {
	upstreams := service.NewMirrorService(*configs.GetMySQLClient()).GetMirrorUpstreams()
	c.JSON(http.StatusOK, utils.ResponseHelper(utils.SetData("upstreams", upstreams)))
}