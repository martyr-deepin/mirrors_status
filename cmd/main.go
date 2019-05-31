package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	configs "mirrors_status/internal/config"
	"mirrors_status/internal/log"
	"mirrors_status/pkg/db/client/influxdb"
	"mirrors_status/pkg/db/client/mysql"
	"mirrors_status/pkg/db/client/redis"
	"mirrors_status/pkg/model/mirror"
	"mirrors_status/pkg/model/operation"
	"mirrors_status/pkg/model/task"
	"mirrors_status/pkg/rest"
	tasks "mirrors_status/pkg/task"
	"strconv"
)

func Init() {
	log.Info("Initializing APP")
	server := configs.NewServerConfig()

	mysql.InitMySQLClient()
	redis.InitRedisClient()
	influxdb.InitInfluxClient()

	err := mysql.NewMySQLClient().Debug().AutoMigrate(operation.MirrorOperation{}, mirror.Mirror{}, task.Task{}, task.CITask{}).Error
	if err != nil {
		panic(err)
	}
	configs.InitMailClient(server.Mail)

	tasks.NewTaskManager()
	return
}

func main() {
	Init()
	r := gin.Default()
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins=[]string{configs.NewServerConfig().Http.AllowOrigin}
	r.Use(cors.New(corsConfig))

	rest.InitGuestController(r)
	rest.InitAdminController(r)

	//r.POST("/check/:username", app.CheckMirrorsByUpstream)
	//r.GET("/history", app.OperationHistory)
	//r.GET("/history/:mirror", app.OperationHistoryByMirror)
	//r.GET("/operation/:index", app.GetOperationByIndex)

	r.Run(":" + strconv.Itoa(configs.NewServerConfig().Http.Port))
}
