package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"mirrors_status/internal/utils"
	cdn_checker "mirrors_status/pkg/business/cdn-checker"
	"mirrors_status/pkg/config"
	"mirrors_status/pkg/log"
	"mirrors_status/pkg/modules/model"
	"mirrors_status/pkg/modules/service"
	"net/http"

	//"mirrors_status/pkg/modules/model"
	//"mirrors_status/pkg/modules/service"
	//"net/http"
	"strconv"

)

type App struct {
	serverConfig *configs.ServerConf
	cdnChecker *cdn_checker.CDNChecker
	mirrorService *service.MirrorService
}

type M map[string]interface{}

func ResponseHelper(m M) gin.H {
	return gin.H{
		"code": utils.SUCCESS,
		"data": m,
		"error": "",
	}
}

func ErrorHelper(err error, statusCode int) gin.H {
	return gin.H{
		"error": err,
		"code": statusCode,
		"data": "",
	}
}

func SetData(key string, val interface{}) M {
	return M{
		key: val,
	}
}

func SuccessResp() M {
	return SetData("status", "success")
}

func Init() (app App) {
	log.Info("Initializing APP")
	var sc configs.ServerConf
	serverConfig := sc.GetConfig()
	app = App{
		serverConfig: serverConfig,
	}

	configs.InitDB(*serverConfig)
	app.cdnChecker = cdn_checker.NewCDNChecker(app.serverConfig.CdnChecker)
	app.mirrorService = service.NewMirrorService(*configs.GetMySQLClient())
	configs.InitScheme()
	return
}

func(app App) Login(c *gin.Context) {

}

func(app App) Logout(c *gin.Context) {

}

func(app App) CheckLoginStatus(c *gin.Context) {

}

func(app App) CreateMirror(c *gin.Context) {
	var mirror model.Mirror
	err := c.BindJSON(&mirror)
	if err != nil {
		log.Errorf("Parse json mirror:%v found error:%v", mirror, err)
		c.JSON(http.StatusBadRequest, ErrorHelper(err, utils.PARAMETER_ERROR))
		return
	}

	err = app.mirrorService.CreateMirror(mirror)
	if err != nil {
		log.Errorf("Create mirror:%v found error:%v", mirror, err)
		c.JSON(http.StatusBadRequest, ErrorHelper(err, utils.CREATE_MIRROR_FAILED))
		return
	}
	c.JSON(http.StatusOK, ResponseHelper(SuccessResp()))
}

func(app App) DeleteMirror(c *gin.Context) {
	pathId := c.Param("id")
	id, err := strconv.Atoi(pathId)
	if err != nil {
		log.Errorf("Parse path param id:%d found error:%v", pathId, err)
		c.JSON(http.StatusBadRequest, ErrorHelper(err, utils.PARAMETER_ERROR))
		return
	}

	err = app.mirrorService.DeleteMirror(id)
	if err != nil {
		log.Errorf("Delete mirror:%d found error:%v", id, err)
		c.JSON(http.StatusBadRequest, ErrorHelper(err, utils.DELETE_MIRROR_FAILED))
		return
	}
	c.JSON(http.StatusOK, ResponseHelper(SuccessResp()))
}

func(app App) UpdateMirror(c *gin.Context) {
	var mirror model.Mirror
	err := c.BindJSON(&mirror)
	if err != nil {
		log.Errorf("Parse json mirror:%v found error:%v", mirror, err)
		c.JSON(http.StatusBadRequest, ErrorHelper(err, utils.PARAMETER_ERROR))
		return
	}

	pathId := c.Param("id")
	id, err := strconv.Atoi(pathId)
	if err != nil {
		log.Errorf("Parse path param id:%d found error:%v", pathId, err)
		c.JSON(http.StatusBadRequest, ErrorHelper(err, utils.PARAMETER_ERROR))
		return
	}

	err = app.mirrorService.UpdateMirror(id, mirror)
	if err != nil {
		log.Errorf("Update mirror:%v found error:%v", mirror, err)
		c.JSON(http.StatusBadRequest, ErrorHelper(err, utils.UPDATE_MIRROR_FAILED))
		return
	}
	c.JSON(http.StatusOK, ResponseHelper(SuccessResp()))
}

func(app App) GetAllMirrors(c *gin.Context) {
	mirrors, err := app.mirrorService.GetAllMirrors()
	if err != nil {
		log.Errorf("Get all mirrors:%v found error:%v", mirrors, err)
		c.JSON(http.StatusBadRequest, ErrorHelper(err, utils.FETCH_DATA_ERROR))
		return
	}
	c.JSON(http.StatusOK, ResponseHelper(SetData("mirrors", mirrors)))
}

func(app App) GetMirrorsByUpstream(c *gin.Context) {
	upstream := c.Param("upstream")
	mirrors, err := app.mirrorService.GetMirrorsByUpstream(upstream)
	if err != nil {
		log.Errorf("Get mirrors by upstream:%s found error:%v", upstream, err)
		c.JSON(http.StatusBadRequest, ErrorHelper(err, utils.FETCH_DATA_ERROR))
		return
	}
	c.JSON(http.StatusOK, ResponseHelper(SetData("mirrors", mirrors)))
}

func(app App) GetAllUpstreams(c *gin.Context) {

}

func(app App) PublishUpstream(c *gin.Context) {

}

func(app App) CreateTask(c *gin.Context) {

}

func(app App) AbortTask(c *gin.Context) {

}

func(app App) GetOpenTasks(c *gin.Context) {

}

func(app App) GetTaskById(c *gin.Context) {

}

func(app App) UpdateTask(c *gin.Context) {

}

func(app App) GetArchives(c *gin.Context) {

}

func(app App) GetArchiveById(c *gin.Context) {

}

func(app App) MailGroup(c *gin.Context) {

}

//func(app App) GetAllMirrors(c *gin.Context) {
//	data := service.GetAllMirrors()
//	c.JSON(200, gin.H{
//		"res": data,
//	})
//}

//func(app App) GetAllMirrorsCdn(c *gin.Context) {
//	data := service.GetAllMirrorsCdn()
//	c.JSON(200, gin.H{
//		"res": data,
//	})
//}
//
//func (app App) AddMirror(c *gin.Context) {
//	var reqMirror model.MirrorsPoint
//	err := c.ShouldBindJSON(&reqMirror)
//	if err != nil {
//		log.Errorf("Bind json found error:%v", err)
//	}
//	err = service.AddMirror(reqMirror)
//	if err != nil {
//		log.Errorf("Insert data found error:%v", err)
//	}
//	c.JSON(200, gin.H{
//		"res": err,
//	})
//}
//
//func (app App) AddMirrorCdn(c *gin.Context) {
//	var reqMirrorCdn model.MirrorsCdnPoint
//	err := c.ShouldBindJSON(&reqMirrorCdn)
//	if err != nil {
//		log.Errorf("Bind json found error:%v", err)
//	}
//	err = service.AddMirrorCdn(reqMirrorCdn)
//	if err != nil {
//		log.Errorf("Insert data found error:%v", err)
//	}
//	c.JSON(200, gin.H{
//		"res": err,
//	})
//}
//
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
	app := Init()
	r := gin.Default()
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins=[]string{app.serverConfig.Http.AllowOrigin}
	r.Use(cors.New(corsConfig))

	guestRouter := r.Group("/")
	{
		guestRouter.POST("/session", app.Login)
		guestRouter.DELETE("/session", app.Logout)
		guestRouter.GET("/session", app.CheckLoginStatus)
		guestRouter.GET("/mirrors", app.GetAllMirrors)
		guestRouter.GET("/mirrors/:upstream", app.GetMirrorsByUpstream)
		guestRouter.GET("/upstreams", app.GetAllUpstreams)
	}

	adminRouter := r.Group("/admin")
	{
		adminRouter.POST("/mirrors", app.CreateMirror)
		adminRouter.DELETE("/mirrors/:id", app.DeleteMirror)
		adminRouter.PUT("/mirrors/:id", app.UpdateMirror)
		adminRouter.POST("/publish/:upstream", app.PublishUpstream)
		adminRouter.POST("/tasks", app.CreateTask)
		adminRouter.DELETE("/tasks/:id", app.AbortTask)
		adminRouter.GET("/tasks", app.GetOpenTasks)
		adminRouter.GET("tasks/:id", app.GetTaskById)
		adminRouter.PATCH("/tasks/:id", app.UpdateTask)
		adminRouter.GET("/archives", app.GetArchives)
		adminRouter.GET("/archives/:id", app.GetArchiveById)
		adminRouter.POST("/mail", app.MailGroup)
	}

	//r.GET("/check/:username", app.SyncAllMirrors)
	//r.GET("/check/:username/:mirror", app.SyncMirror)
	//r.POST("/check/:username", app.CheckMirrorsByUpstream)
	//r.GET("/history", app.OperationHistory)
	//r.GET("/history/:mirror", app.OperationHistoryByMirror)
	//r.GET("/operation/:index", app.GetOperationByIndex)

	r.Run(":" + strconv.Itoa(app.serverConfig.Http.Port))
}
