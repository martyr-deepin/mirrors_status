package main

import (
	"github.com/gin-gonic/gin"
	"mirrors_status/cmd/config"
	"mirrors_status/cmd/infrastructure"
	"mirrors_status/cmd/log"
	"mirrors_status/cmd/modules/db/influxdb"
	"mirrors_status/cmd/modules/model"
	"time"
)

type App struct {
	serverConfig *configs.ServerConf
	client *influxdb.Client
}

func Init() (app App) {
	log.Info("Initializing APP")
	var sc configs.ServerConf
	serverConfig := sc.GetConfig()
	app = App{
		serverConfig: serverConfig,
	}

	InitInfluxDB(*serverConfig)
	app.client = infrastructure.GetInfluxdbClient()
	return
}

func InitInfluxDB(config configs.ServerConf) {
	host := config.InfluxDB.Host
	port := config.InfluxDB.Port
	dbName := config.InfluxDB.DBName
	username := config.InfluxDB.Username
	password := config.InfluxDB.Password
	err := infrastructure.InitInfluxdbClient(host, port, dbName, username, password)
	if err != nil {
		log.Errorf("Err connecting influxdb:%v", config.InfluxDB)
	}
}

func(app App) GetAllMirrors(c *gin.Context) {
	data, _ := app.client.QueryDB("select * from mirrors")
	c.JSON(200, gin.H{
		"res": data,
	})
}

func(app App) GetAllMirrorsCdn(c *gin.Context) {
	data, _ := app.client.QueryDB("select * from mirrors_cdn")
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
	err = app.client.PushMirror(time.Now(), reqMirror)
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
	err = app.client.PushMirrorCdn(time.Now(), reqMirrorCdn)
	if err != nil {
		log.Errorf("Insert data found error:%v", err)
	}
	c.JSON(200, gin.H{
		"res": err,
	})
}

func (app App) TestApi(c *gin.Context) {
	data, _ := app.client.QueryDB(c.PostForm("query"))
	c.JSON(200, gin.H{
		"res": data,
	})
}

func main() {
	app := Init()
	r := gin.Default()

	r.GET("/mirrors", app.GetAllMirrors)
	r.GET("/mirrors_cdn", app.GetAllMirrorsCdn)

	r.POST("/mirrors", app.AddMirror)
	r.POST("/mirrors_cdn", app.AddMirrorCdn)

	r.POST("/test", app.TestApi)
	r.Run(":" + app.serverConfig.Http.Port)
}
