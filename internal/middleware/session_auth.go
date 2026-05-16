package middleware

import (
	"net/http"

	"Novels_AI/backend/internal/pkg/common"

	"github.com/gin-gonic/gin"
)

// SessionAuth 校验浏览器会话中是否存在登录时写入的 session 标识。
func SessionAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := common.GetSession(c, common.SessionKey); ok {
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, &common.Response{
			Code: http.StatusUnauthorized,
			Msg:  "未登录",
		})
	}
}
