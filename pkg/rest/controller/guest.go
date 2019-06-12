package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/influxdata/influxdb/uuid"
	"mirrors_status/internal/log"
	"mirrors_status/pkg/db/redis"
	"mirrors_status/pkg/ldap"
	"mirrors_status/pkg/model/mirror"
	"mirrors_status/pkg/utils"
	"net/http"
	"time"
)

func GetAllMirrors(ctx *gin.Context) {
	mirrors, err := mirror.GetAllMirrors()
	if err != nil {
		log.Errorf("Get all mirrors found error:%#v", err)
		ctx.JSON(http.StatusNoContent, utils.ErrorHelper(err, utils.FETCH_DATA_ERROR))
		return
	}
	ctx.JSON(http.StatusOK, utils.ResponseHelper(utils.SetData("mirrors", mirrors)))
}

func GetMirrorsByUpstream(c *gin.Context) {
	upstream := c.Param("upstream")
	mirrors, err := mirror.GetMirrorsByUpstream(upstream)
	if err != nil {
		log.Errorf("Get mirrors by upstream:%s found error:%v", upstream, err)
		c.JSON(http.StatusBadRequest, utils.ErrorHelper(err, utils.FETCH_DATA_ERROR))
		return
	}
	c.JSON(http.StatusOK, utils.ResponseHelper(utils.SetData("mirrors", mirrors)))
}

func GetAllUpstreams(c *gin.Context) {
	upstreams := mirror.GetMirrorUpstreams()
	c.JSON(http.StatusOK, utils.ResponseHelper(utils.SetData("upstreams", upstreams)))
}

type LoginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func Login(c *gin.Context) {
	var loginReq LoginReq
	err := c.BindJSON(&loginReq)
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.ErrorHelper(err, utils.PARAMETER_ERROR))
		return
	}
	clt, err := ldap.NewLdapClient()
	if err != nil {
		c.JSON(http.StatusInternalServerError, utils.ErrorHelper(err, utils.INTERNAL_LDAP_ERROR))
		return
	}
	err = clt.CheckUserPassword(loginReq.Username, loginReq.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, utils.ErrorHelper(err, utils.LOGIN_FAILED))
		return
	}
	sessionId := uuid.FromTime(time.Now()).String()
	_ = redis.Set(loginReq.Username + "-session-id", sessionId, time.Hour* 12)
	c.SetCookie("username", loginReq.Username, 3600, "/", "", false, false)
	c.SetCookie("sessionId", sessionId, 3600, "/", "", false, false)
	c.JSON(http.StatusOK, utils.ResponseHelper(utils.SuccessResp()))
}