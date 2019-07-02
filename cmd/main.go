package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	configs "mirrors_status/internal/config"
	"mirrors_status/internal/log"
	"mirrors_status/pkg/db/influxdb"
	"mirrors_status/pkg/db/mysql"
	"mirrors_status/pkg/db/redis"
	"mirrors_status/pkg/model/archive"
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

	err := mysql.NewMySQLClient().Debug().AutoMigrate(operation.MirrorOperation{}, mirror.Mirror{}, task.Task{}, task.CITask{}, archive.Archive{}).Error
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

	if configs.NewServerConfig().Http.AllowOrigin != nil {
		corsConfig := cors.DefaultConfig()
		corsConfig.AllowOrigins = configs.NewServerConfig().Http.AllowOrigin
		corsConfig.AllowCredentials = true
		r.Use(cors.New(corsConfig))
	} else {
		r.Use(cors.New(cors.Config{
			AllowAllOrigins: true,
			AllowHeaders: []string{"Authorization", "Content-Length", "X-CSRF-Token", "Accept", "Origin", "Host", "Connection", "Accept-Encoding", "Accept-Language,DNT", "X-CustomHeader", "Keep-Alive", "User-Agent", "X-Requested-With", "If-Modified-Since", "Cache-Control", "Content-Type", "Pragma", "Timestamp", "timestamp"},
			AllowMethods: []string{"POST", "GET", "OPTIONS", "PUT", "DELETE", "PATCH"},
			ExposeHeaders: []string{"Content-Length", "Access-Control-Allow-Origin", "Access-Control-Allow-Headers", "Content-Type"},
		}))
	}

	rest.InitGuestController(r)
	rest.InitAdminController(r)

	r.Run(":" + strconv.Itoa(configs.NewServerConfig().Http.Port))
}
