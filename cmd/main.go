package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	configs "mirrors_status/internal/config"
	"mirrors_status/internal/log"
	"mirrors_status/pkg/rest"
	"mirrors_status/pkg/task"

	"strconv"
)

func Init() {
	log.Info("Initializing APP")
	server := configs.NewServerConfig()
	configs.InitDB(*server)
	configs.InitMailClient(server.Mail)
	configs.InitScheme()
	task.Execute()
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
