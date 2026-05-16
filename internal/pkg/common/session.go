package common

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const SessionKey = "novels_ai_session"

func SetSession(c *gin.Context, key, v string) error {
	session := sessions.Default(c)
	session.Options(sessions.Options{
		Path:     "/",
		MaxAge:   0,
		Secure:   false,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Domain:   "",
	})
	session.Set(key, v)
	return session.Save()
}

// GetSession 读取 gin session 中的字符串值，空字符串按未登录处理。
func GetSession(c *gin.Context, key string) (string, bool) {
	value, ok := sessions.Default(c).Get(key).(string)
	if !ok || value == "" {
		return "", false
	}

	return value, true
}
