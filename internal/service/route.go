package service

import (
	"Novels_AI/backend/internal/middleware"
	"Novels_AI/backend/internal/service/login"

	"github.com/gin-gonic/gin"
)

func AddRoute(engine *gin.Engine) {
	// 中间件
	engine.Use(middleware.ErrorHandler())

	// 登陆相关接口
	loginService := login.NewLoginService()
	engine.POST("/api/v1/login", loginService.Login)
}
