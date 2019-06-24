package middleware

import (
	"github.com/gin-gonic/gin"
	"mirrors_status/pkg/db/redis"
	"net/http"
	"time"
)

func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, err := c.Cookie("username")
		if err != nil {
			c.Abort()
			c.String(http.StatusUnauthorized, "username cookie expired")
			return
		}
		session_id, err := c.Cookie("sessionId")
		if err != nil {
			c.Abort()
			c.String(http.StatusUnauthorized, "cookie sessionId expired")
			return
		}
		redisSession, err := redis.Get(username + "-session-id")
		if err != nil {
			c.Abort()
			c.String(http.StatusUnauthorized, "cookie sessionId illegal")
			return
		}
		if session_id == redisSession {
			_ = redis.Set(username + "-session-id", session_id, time.Hour * 1)
			c.Next()
			return
		}
		c.Abort()
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "Set cookie failed",
		})
	}
}