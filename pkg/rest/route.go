package rest

import (
	"github.com/gin-gonic/gin"
	"mirrors_status/pkg/rest/controller"
)

func InitGuestController(engine *gin.Engine) {
	r := engine.Group("/api/v1")

	r.GET("/mirrors", controller.GetAllMirrors)
	r.GET("/mirrors/:upstream", controller.GetMirrorsByUpstream)
	r.GET("/upstreams", controller.GetAllUpstreams)
}

func InitAdminController(engine *gin.Engine) {
	r := engine.Group("/api/v1/admin")

	r.POST("/mirrors", controller.CreateMirror)
	r.DELETE("/mirrors/:id", controller.DeleteMirror)
	r.PUT("/mirrors", controller.UpdateMirror)
	r.POST("/tasks", controller.CreateTask)
	r.GET("/tasks/:id", controller.GetTaskById)
	r.GET("/tasks", controller.GetIOpenTasks)
	r.GET("/check", controller.CheckAllMirrors)
	r.GET("/check/:upstream", controller.CheckMirrorsByUpstream)
	r.POST("/check", controller.CheckMirrors)
	r.DELETE("/tasks/:id", controller.AbortTask)
	r.PATCH("/tasks/:id/:status", controller.UpdateTaskStatus)
	r.GET("/archives/:id", controller.GetArchiveByTaskId)
	r.GET("/archives", controller.GetAllArchives)
}

