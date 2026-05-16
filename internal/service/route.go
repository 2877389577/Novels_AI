package service

import (
	docs "Novels_AI/backend/docs"
	loginbiz "Novels_AI/backend/internal/biz/login"
	"Novels_AI/backend/internal/data"
	"Novels_AI/backend/internal/middleware"
	"Novels_AI/backend/internal/service/login"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
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
func AddRoute(engine *gin.Engine, requestsPerMinute int, sessionSalt string, db *gorm.DB) {
	docs.SwaggerInfo.BasePath = "/api/v1"

	// 中间件
	engine.Use(middleware.RequestID())
	engine.Use(middleware.RateLimiter(requestsPerMinute))
	engine.Use(middleware.ErrorHandler())
	engine.Use(gin.Recovery())
	engine.Use(sessions.Sessions("go-session", cookie.NewStore([]byte(sessionSalt))))

	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 登陆相关接口
	adminInfoData := data.NewAdminInfoData(db)
	loginUseCase := loginbiz.NewLoginUseCase(adminInfoData)
	loginService := login.NewLoginService(loginUseCase)

	group := engine.Group("/api/v1")
	// 检查初始化密码接口
	group.GET("/login/initial-password", loginService.CheckInitialPassword)
	// 设置初始化密码接口
	group.POST("/login/password", loginService.SetPassword)
	// 登陆接口
	group.POST("/login", loginService.Login)
}
