package middleware

import (
	"errors"
	"github.com/gin-gonic/gin"
	"mirrors_status/pkg/db/redis"
	"mirrors_status/pkg/utils"
	"net/http"
)

func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, err := c.Cookie("username")
		if err != nil {
			c.Abort()
			c.JSON(http.StatusOK, utils.ErrorHelper(errors.New("parse cookie username error"), utils.GET_COOKIE_ERROR))
			return
		}
		session_id, err := c.Cookie("sessionId")
		if err != nil {
			c.Abort()
			c.JSON(http.StatusOK, utils.ErrorHelper(errors.New("parse cookie session id error"), utils.GET_COOKIE_ERROR))
			return
		}
		redisSession, err := redis.Get(username + "-session-id")
		if err != nil {
			c.Abort()
			c.JSON(http.StatusOK, utils.ErrorHelper(errors.New("illegal session"), utils.GET_COOKIE_ERROR))
			return
		}
		if session_id == redisSession {
			//_ = redis.Set(username + "-session-id", session_id, time.Hour * 1)
			c.Next()
			return
		}
		c.Abort()
		c.JSON(http.StatusOK, utils.ErrorHelper(errors.New("session expired"), utils.GET_COOKIE_ERROR))
	}
}