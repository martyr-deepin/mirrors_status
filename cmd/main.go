package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	configs "mirrors_status/internal/config"
	"mirrors_status/internal/log"
	"mirrors_status/pkg/rest"
	//"mirrors_status/pkg/client/model"
	//"mirrors_status/pkg/client/service"
	//"net/http"
	"strconv"
)

func Init() {
	log.Info("Initializing APP")
	configs.InitDB(configs.NewServerConfig())
	configs.InitScheme()
	return
}

//func (app App) SyncAllMirrors(c *gin.Context) {
//	username := c.Param("username")
//	log.Infof("User:%s trying sync all mirrors")
//	index := app.cdnChecker.CheckAllMirrors(app.serverConfig.CdnChecker, username)
//	c.JSON(http.StatusAccepted, gin.H{
//		"index": index,
//	})
//}
//
//func (app App) SyncMirror(c *gin.Context) {
//	mirrorName := c.Param("mirror")
//	username := c.Param("username")
//
//	log.Infof("Username:%s, Mirror ID:%s", username, mirrorName)
//	index := app.cdnChecker.CheckMirror(mirrorName, app.serverConfig.CdnChecker, username)
//	c.JSON(http.StatusAccepted, gin.H{
//		"index": index,
//	})
//}
//
//func (app App) OperationHistory(c *gin.Context) {
//	data := service.GetOperationsByDateDesc()
//	c.JSON(http.StatusOK, gin.H{
//		"history": data,
//	})
//}
//
//func (app App) OperationHistoryByMirror(c *gin.Context) {
//	mirror := c.Param("mirror")
//	data := service.GetOperationsByMirror(mirror)
//	c.JSON(http.StatusOK, gin.H{
//		"history": data,
//	})
//}
//
//func (app App) GetOperationByIndex(c *gin.Context) {
//	index := c.Param("index")
//	log.Info(index)
//	data, err := service.GetOperationByIndex(index)
//	if err != nil {
//		log.Infof("%#v", err)
//		c.JSON(http.StatusNoContent, gin.H{
//			"msg": "get operation data found error",
//		})
//		return
//	}
//	c.JSON(http.StatusOK, gin.H{
//		"operation": data,
//	})
//}
//
//func (app App) CheckMirrorsByUpstream(c *gin.Context) {
//	username := c.Param("username")
//	var mirrors cdn_checker.MirrorsReq
//	err := c.BindJSON(&mirrors)
//	if err != nil {
//		c.JSON(http.StatusBadRequest, gin.H{"msg": err.Error()})
//		return
//	}
//	index := app.cdnChecker.CheckMirrorsByUpstream(mirrors, app.serverConfig.CdnChecker, username)
//	c.JSON(http.StatusAccepted, gin.H{
//		"index": index,
//	})
//}

func main() {
	Init()
	r := gin.Default()
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins=[]string{configs.NewServerConfig().Http.AllowOrigin}
	r.Use(cors.New(corsConfig))

	rest.InitGuestController(r)
	rest.InitAdminController(r)

	//r.GET("/check/:username", app.SyncAllMirrors)
	//r.GET("/check/:username/:mirror", app.SyncMirror)
	//r.POST("/check/:username", app.CheckMirrorsByUpstream)
	//r.GET("/history", app.OperationHistory)
	//r.GET("/history/:mirror", app.OperationHistoryByMirror)
	//r.GET("/operation/:index", app.GetOperationByIndex)

	r.Run(":" + strconv.Itoa(configs.NewServerConfig().Http.Port))
}
