package utils

import (
	"github.com/gin-gonic/gin"
	"mirrors_status/pkg/model"
)

type M map[string]interface{}

func ResponseHelper(m M) gin.H {
	return gin.H{
		"code":  model.SUCCESS,
		"data":  m,
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
