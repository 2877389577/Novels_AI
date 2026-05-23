package service

import (
	docs "Novels_AI/backend/docs"
	aiproviderbiz "Novels_AI/backend/internal/biz/aiprovider"
	loginbiz "Novels_AI/backend/internal/biz/login"
	novelbiz "Novels_AI/backend/internal/biz/novel"
	uploadbiz "Novels_AI/backend/internal/biz/upload"
	"Novels_AI/backend/internal/data"
	"Novels_AI/backend/internal/middleware"
	aiproviderservice "Novels_AI/backend/internal/service/aiprovider"
	"Novels_AI/backend/internal/service/login"
	novelservice "Novels_AI/backend/internal/service/novel"
	uploadservice "Novels_AI/backend/internal/service/upload"

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
func AddRoute(engine *gin.Engine, requestsPerMinute int, sessionSalt string, uploadConfig data.S3UploadConfig, apiKeyCipher aiproviderbiz.APIKeyCipher, db *gorm.DB) {
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

	// 小说相关接口
	novelData := data.NewNovelData(db)
	novelUseCase := novelbiz.NewNovelUseCase(novelData)
	novelService := novelservice.NewNovelService(novelUseCase)
	// 章节相关接口
	chapterData := data.NewChapterData(db)
	chapterUseCase := novelbiz.NewChapterUseCase(novelData, chapterData)
	chapterService := novelservice.NewChapterService(chapterUseCase)
	// 角色相关接口
	characterData := data.NewCharacterData(db)
	characterUseCase := novelbiz.NewCharacterUseCase(novelData, characterData)
	characterService := novelservice.NewCharacterService(characterUseCase)

	// ai 提供商相关接口
	aiProviderData := data.NewAIProviderData(db)
	aiProviderUseCase := aiproviderbiz.NewAIProviderUseCase(aiProviderData, apiKeyCipher)
	aiProviderService := aiproviderservice.NewAIProviderService(aiProviderUseCase)

	// 上传相关接口
	s3UploadData := data.NewS3UploadData(uploadConfig)
	uploadUseCase := uploadbiz.NewUploadUseCase(s3UploadData)
	uploadService := uploadservice.NewUploadService(uploadUseCase)

	group := engine.Group("/api/v1")
	// 检查初始化密码接口
	group.GET("/login/initial-password", loginService.CheckInitialPassword)
	// 设置初始化密码接口
	group.POST("/login/password", loginService.SetPassword)
	// 登陆接口
	group.POST("/login", loginService.Login)

	// 小说相关接口需要登录会话，避免未登录用户直接维护小说数据。
	novelGroup := group.Group("/novels", middleware.SessionAuth())
	novelGroup.POST("", novelService.Create)
	novelGroup.GET("", novelService.List)
	novelGroup.GET("/:id", novelService.Get)
	novelGroup.PUT("/update", novelService.Update)
	novelGroup.DELETE("/:id", novelService.Delete)
	novelGroup.POST("/:id/chapters", chapterService.Create)
	novelGroup.GET("/:id/chapters", chapterService.List)
	novelGroup.GET("/:id/chapters/:chapterId", chapterService.Get)
	novelGroup.PUT("/:id/chapters/:chapterId", chapterService.Update)
	novelGroup.DELETE("/:id/chapters/:chapterId", chapterService.Delete)
	novelGroup.POST("/:id/characters", characterService.Create)
	novelGroup.GET("/:id/characters", characterService.List)
	novelGroup.GET("/:id/characters/:characterId", characterService.Get)
	novelGroup.PUT("/:id/characters/:characterId", characterService.Update)
	novelGroup.DELETE("/:id/characters/:characterId", characterService.Delete)

	// ai 提供商接口需要登录会话，避免匿名用户读取或维护模型密钥。
	aiProviderGroup := group.Group("/ai-providers", middleware.SessionAuth())
	aiProviderGroup.POST("", aiProviderService.Create)
	aiProviderGroup.POST("/models/query", aiProviderService.QueryModels)
	aiProviderGroup.POST("/enable", aiProviderService.Enable)
	aiProviderGroup.GET("", aiProviderService.List)
	aiProviderGroup.GET("/models", aiProviderService.ListEnabledModels)
	aiProviderGroup.GET("/:id", aiProviderService.Get)
	aiProviderGroup.PUT("/:id", aiProviderService.Update)
	aiProviderGroup.DELETE("/:id", aiProviderService.Delete)

	// 上传接口同样需要登录会话，避免匿名用户写入对象存储。
	group.POST("/upload", middleware.SessionAuth(), uploadService.Upload)
}
