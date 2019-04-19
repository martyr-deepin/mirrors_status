package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	cdn_checker "mirrors_status/pkg/business/cdn-checker"
	"mirrors_status/pkg/config"
	"mirrors_status/pkg/log"
	"mirrors_status/pkg/modules/db/influxdb"
	"mirrors_status/pkg/modules/db/mysql"
	"mirrors_status/pkg/modules/model"
	"mirrors_status/pkg/modules/service"
	"strconv"
)

type App struct {
	serverConfig *configs.ServerConf
	influxClient *influxdb.Client
	mysqlClient *mysql.Client
	cdnChecker *cdn_checker.CDNChecker
}

func Init() (app App) {
	log.Info("Initializing APP")
	var sc configs.ServerConf
	serverConfig := sc.GetConfig()
	app = App{
		serverConfig: serverConfig,
	}

	configs.InitDB(*serverConfig)
	app.influxClient = configs.GetInfluxdbClient()
	app.mysqlClient = configs.GetMySQLClient()
	app.cdnChecker = cdn_checker.NewCDNChecker(app.serverConfig.CdnChecker)
	configs.InitScheme()
	return
}

func(app App) GetAllMirrors(c *gin.Context) {
	data := service.GetAllMirrors(app.influxClient)
	c.JSON(200, gin.H{
		"res": data,
	})
}

func(app App) GetAllMirrorsCdn(c *gin.Context) {
	data := service.GetAllMirrorsCdn(app.influxClient)
	c.JSON(200, gin.H{
		"res": data,
	})
}

func (app App) AddMirror(c *gin.Context) {
	var reqMirror model.MirrorsPoint
	err := c.ShouldBindJSON(&reqMirror)
	if err != nil {
		log.Errorf("Bind json found error:%v", err)
	}
	err = service.AddMirror(app.mysqlClient, app.influxClient, reqMirror)
	if err != nil {
		log.Errorf("Insert data found error:%v", err)
	}
	c.JSON(200, gin.H{
		"res": err,
	})
}

func (app App) AddMirrorCdn(c *gin.Context) {
	var reqMirrorCdn model.MirrorsCdnPoint
	err := c.ShouldBindJSON(&reqMirrorCdn)
	if err != nil {
		log.Errorf("Bind json found error:%v", err)
	}
	err = service.AddMirrorCdn(app.mysqlClient, app.influxClient, reqMirrorCdn)
	if err != nil {
		log.Errorf("Insert data found error:%v", err)
	}
	c.JSON(200, gin.H{
		"res": err,
	})
}

func (app App) TestApi(c *gin.Context) {
	query := c.PostForm("query")
	data := service.TestApi(app.influxClient, query)
	c.JSON(200, gin.H{
		"res": data,
	})
}

func (app App) SyncAllMirrors(c *gin.Context) {
	username := c.Param("username")
	log.Infof("User:%s trying sync all mirrors")
	err := app.cdnChecker.CheckAllMirrors(app.serverConfig.CdnChecker, username)
	if err != nil {
		log.Errorf("Sync all mirror found error:%v", err)
		c.JSON(400, gin.H{
			"res": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"res": "success",
	})
}

func (app App) SyncMirror(c *gin.Context) {
	mirrorName := c.Param("mirror")
	username := c.Param("username")

	log.Infof("Username:%s, Mirror ID:%s", username, mirrorName)
	err := app.cdnChecker.CheckMirror(mirrorName, app.serverConfig.CdnChecker, username)
	if err != nil {
		log.Errorf("Sync mirror found error:%v", err)
		c.JSON(400, gin.H{
			"res": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"res": "success",
	})
}

func (app App) OperationHistory(c *gin.Context) {
	data := service.GetOperationsByDateDesc(app.mysqlClient)
	c.JSON(200, gin.H{
		"history": data,
	})
}

func (app App) OperationHistoryByMirror(c *gin.Context) {
	mirror := c.Param("mirror")
	data := service.GetOperationsByMirror(app.mysqlClient, mirror)
	c.JSON(200, gin.H{
		"history": data,
	})
}

func main() {
	app := Init()
	r := gin.Default()
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins=[]string{app.serverConfig.Http.AllowOrigin}
	r.Use(cors.New(corsConfig))

	//r.GET("/mirrors", app.GetAllMirrors)
	//r.GET("/mirrors_cdn", app.GetAllMirrorsCdn)
	//
	//r.POST("/mirrors", app.AddMirror)
	//r.POST("/mirrors_cdn", app.AddMirrorCdn)
	//
	//r.POST("/test", app.TestApi)

	r.GET("/check/:username", app.SyncAllMirrors)

	r.GET("/check/:username/:mirror", app.SyncMirror)

	r.GET("/history", app.OperationHistory)

	r.GET("/history/:mirror", app.OperationHistoryByMirror)

	r.Run(":" + strconv.Itoa(app.serverConfig.Http.Port))
}
