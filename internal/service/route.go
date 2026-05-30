package service

import (
	docs "Novels_AI/backend/docs"
	aiproviderbiz "Novels_AI/backend/internal/biz/aiprovider"
	aitaskconfigbiz "Novels_AI/backend/internal/biz/aitaskconfig"
	imageproviderbiz "Novels_AI/backend/internal/biz/imageprovider"
	loginbiz "Novels_AI/backend/internal/biz/login"
	novelbiz "Novels_AI/backend/internal/biz/novel"
	uploadbiz "Novels_AI/backend/internal/biz/upload"
	"Novels_AI/backend/internal/data"
	"Novels_AI/backend/internal/event"
	"Novels_AI/backend/internal/middleware"
	aiproviderservice "Novels_AI/backend/internal/service/aiprovider"
	aitaskconfigservice "Novels_AI/backend/internal/service/aitaskconfig"
	imageproviderservice "Novels_AI/backend/internal/service/imageprovider"
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
	mindMapData := data.NewNovelMindMapData(db)
	mindMapUseCase := novelbiz.NewMindMapUseCase(novelData, mindMapData)
	mindMapService := novelservice.NewMindMapService(mindMapUseCase)
	// 章节相关接口
	chapterData := data.NewChapterData(db)
	chapterPlotAnalysisData := data.NewChapterPlotAnalysisData(db)
	eventBus := event.NewBus()

	// ai 提供商相关接口
	aiProviderData := data.NewAIProviderData(db)
	aiProviderUseCase := aiproviderbiz.NewAIProviderUseCase(aiProviderData, apiKeyCipher)
	aiProviderService := aiproviderservice.NewAIProviderService(aiProviderUseCase)
	aiTaskConfigData := data.NewAITaskConfigData(db)
	aiTaskConfigUseCase := aitaskconfigbiz.NewAITaskConfigUseCase(aiTaskConfigData)
	aiTaskConfigService := aitaskconfigservice.NewAITaskConfigService(aiTaskConfigUseCase)
	s3UploadData := data.NewS3UploadData(uploadConfig)
	imageAIProviderData := data.NewImageAIProviderData(db)
	imageAIProviderUseCase := imageproviderbiz.NewImageAIProviderUseCase(imageAIProviderData, apiKeyCipher, s3UploadData)
	imageAIProviderService := imageproviderservice.NewImageAIProviderService(imageAIProviderUseCase)
	contentOptimizationUseCase := novelbiz.NewContentOptimizationUseCase(novelData, aiProviderData, apiKeyCipher)
	contentOptimizationService := novelservice.NewContentOptimizationService(contentOptimizationUseCase)
	chapterPlotAnalysisUseCase := novelbiz.NewChapterPlotAnalysisUseCase(chapterPlotAnalysisData, aiTaskConfigUseCase, aiProviderData, apiKeyCipher)
	novelbiz.RegisterChapterPlotAnalysisEventHandlers(eventBus, chapterPlotAnalysisUseCase)
	chapterUseCase := novelbiz.NewChapterUseCase(novelData, chapterData, eventBus)
	chapterService := novelservice.NewChapterService(chapterUseCase, chapterPlotAnalysisUseCase)

	// 角色相关接口
	characterData := data.NewCharacterData(db)
	characterUseCase := novelbiz.NewCharacterUseCase(novelData, characterData, aiProviderData, chapterData, apiKeyCipher)
	characterService := novelservice.NewCharacterService(characterUseCase)
	characterRelationData := data.NewCharacterRelationData(db)
	characterRelationUseCase := novelbiz.NewCharacterRelationUseCase(novelData, characterData, characterRelationData)
	characterRelationService := novelservice.NewCharacterRelationService(characterRelationUseCase)

	// 上传相关接口
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
	novelGroup.POST("/:id/outline", novelService.SaveOutline)
	novelGroup.GET("/:id/outline", novelService.GetOutline)
	novelGroup.GET("/:id/mind-map", mindMapService.Get)
	novelGroup.PUT("/:id/mind-map", mindMapService.Save)
	novelGroup.POST("/:id/mind-map/nodes", mindMapService.CreateNode)
	novelGroup.GET("/:id/mind-map/nodes/:nodeUid", mindMapService.GetNode)
	novelGroup.PUT("/:id/mind-map/nodes/:nodeUid", mindMapService.UpdateNode)
	novelGroup.DELETE("/:id/mind-map/nodes/:nodeUid", mindMapService.DeleteNode)
	novelGroup.POST("/:id/content/optimize", contentOptimizationService.Optimize)
	novelGroup.POST("/:id/chapters", chapterService.Create)
	novelGroup.GET("/:id/chapters", chapterService.List)
	novelGroup.GET("/:id/chapters/:chapterId/plot-analysis", chapterService.GetPlotAnalysis)
	novelGroup.GET("/:id/chapters/:chapterId", chapterService.Get)
	novelGroup.PUT("/:id/chapters/:chapterId", chapterService.Update)
	novelGroup.DELETE("/:id/chapters/:chapterId", chapterService.Delete)
	// 按章节内容生成角色卡片，路径中的 id 表示章节 ID。
	novelGroup.GET("/chapters/:id/characters/generate-card", characterService.GenerateCharacterCard)
	novelGroup.POST("/:id/characters", characterService.Create)
	novelGroup.GET("/:id/characters", characterService.List)
	novelGroup.GET("/:id/characters/:characterId", characterService.Get)
	novelGroup.PUT("/:id/characters/:characterId", characterService.Update)
	novelGroup.DELETE("/:id/characters/:characterId", characterService.Delete)
	novelGroup.GET("/:id/character-relation-graph", characterRelationService.GetGraph)
	novelGroup.PUT("/:id/character-relation-graph/nodes/layout", characterRelationService.SaveNodeLayouts)
	novelGroup.GET("/:id/character-relations", characterRelationService.List)
	novelGroup.POST("/:id/character-relations", characterRelationService.Create)
	novelGroup.GET("/:id/character-relations/:relationId", characterRelationService.Get)
	novelGroup.PUT("/:id/character-relations/:relationId", characterRelationService.Update)
	novelGroup.DELETE("/:id/character-relations/:relationId", characterRelationService.Delete)

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

	// 主动执行 AI 任务配置接口需要登录会话，避免匿名用户开关后台自动 AI 功能。
	aiTaskConfigGroup := group.Group("/ai-task-configs", middleware.SessionAuth())
	aiTaskConfigGroup.GET("", aiTaskConfigService.List)
	aiTaskConfigGroup.PUT("/:taskCode", aiTaskConfigService.Update)

	// 生图 AI 提供商接口需要登录会话，避免匿名用户读取或维护生图密钥。
	imageAIProviderGroup := group.Group("/image-ai-providers", middleware.SessionAuth())
	imageAIProviderGroup.POST("", imageAIProviderService.Create)
	imageAIProviderGroup.POST("/enable", imageAIProviderService.Enable)
	imageAIProviderGroup.GET("", imageAIProviderService.List)
	imageAIProviderGroup.GET("/:id", imageAIProviderService.Get)
	imageAIProviderGroup.PUT("/:id", imageAIProviderService.Update)
	imageAIProviderGroup.DELETE("/:id", imageAIProviderService.Delete)

	// 图片生成接口会使用当前启用的生图 AI 提供商，并把生成结果上传到对象存储。
	imageGroup := group.Group("/images", middleware.SessionAuth())
	imageGroup.POST("/generate", imageAIProviderService.GenerateImage)

	// 上传接口同样需要登录会话，避免匿名用户写入对象存储。
	group.POST("/upload", middleware.SessionAuth(), uploadService.Upload)
}
