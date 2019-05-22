package rest

import (
	"github.com/gin-gonic/gin"
	"mirrors_status/pkg/rest/controller"
)

func InitAdminController(engine *gin.Engine) {
	r := engine.Group("/admin")

	r.POST("/mirrors", controller.CreateMirror)
	r.DELETE("/mirrors/:id", controller.DeleteMirror)
	r.PUT("/mirrors/:id", controller.UpdateMirror)
}

func InitGuestController(engine *gin.Engine) {
	r := engine.Group("/")
	r.GET("/mirrors", controller.GetAllMirrors)
	r.GET("/mirrors/:upstream", controller.GetMirrorsByUpstream)
	r.GET("/upstream", controller.GetAllUpstreams)
}