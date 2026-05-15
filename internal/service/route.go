package service

import (
	"Novels_AI/backend/internal/middleware"
	"Novels_AI/backend/internal/service/login"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           AI小说系统接口文档
// @version         1.0
// @description     AI小说系统接口文档
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api/v1

// @externalDocs.description  OpenAPI
// @externalDocs.url          https://swagger.io/resources/open-api/
func AddRoute(engine *gin.Engine) {

	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 中间件
	engine.Use(middleware.ErrorHandler())

	// 登陆相关接口
	loginService := login.NewLoginService()

	group := engine.Group("/api/v1")
	// 登陆接口
	group.POST("/login", loginService.Login)
}
