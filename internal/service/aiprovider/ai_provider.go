package aiprovider

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	aiproviderbiz "Novels_AI/backend/internal/biz/aiprovider"
	"Novels_AI/backend/internal/data/dto"
	"Novels_AI/backend/internal/pkg/common"

	"github.com/gin-gonic/gin"
)

const maxAIProviderPageSize = 100

type AIProviderService struct {
	useCase AIProviderUseCase
}

// AIProviderUseCase 描述 service 层依赖的 ai 提供商业务能力。
type AIProviderUseCase interface {
	Create(ctx context.Context, params dto.CreateAIProviderRequest) (*aiproviderbiz.AIProviderDetail, error)
	List(ctx context.Context, page, pageSize int) (*aiproviderbiz.ListAIProviderResult, error)
	Get(ctx context.Context, id int64) (*aiproviderbiz.AIProviderDetail, error)
	Update(ctx context.Context, params dto.UpdateAIProviderRequest) (*aiproviderbiz.AIProviderDetail, error)
	Delete(ctx context.Context, id int64) error
	Enable(ctx context.Context, id int64) error
	ListEnabledModels(ctx context.Context) ([]string, error)
	QueryModels(ctx context.Context, params dto.QueryAIProviderModelsRequest) ([]string, error)
}

type aiProviderResponse struct {
	// ai 提供商 ID
	ID int64 `json:"id"`
	// ai 提供商名称
	Name string `json:"name"`
	// ai 提供商类型
	ProviderType string `json:"providerType"`
	// ai 提供商基础 URL
	BaseURL string `json:"baseUrl"`
	// API Key 明文，仅详情类响应返回
	APIKey string `json:"apiKey,omitempty"`
	// 是否启用
	IsEnabled bool `json:"isEnabled"`
	// ai 提供商额外配置
	ConfigJSON json.RawMessage `json:"configJson" swaggertype:"object"`
	// 支持的模型列表
	Models []string `json:"models"`
	// 默认模型
	DefaultModel string `json:"defaultModel"`
	// 默认生图模型
	DefaultImageModel string `json:"defaultImageModel"`
	// 最大上下文长度
	MaxContextLength int64 `json:"maxContextLength"`
	// 最大输入令牌数
	MaxInputTokens int `json:"maxInputTokens"`
	// 最大输出令牌数
	MaxOutputTokens int `json:"maxOutputTokens"`
	// 创建时间
	CreatedAt string `json:"createdAt"`
	// 更新时间
	UpdatedAt string `json:"updatedAt"`
}

type aiProviderListResponse struct {
	Items    []aiProviderResponse `json:"items"`
	Total    int64                `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"pageSize"`
}

type aiProviderModelsResponse struct {
	// 支持的模型 ID 列表
	Models []string `json:"models"`
}

func NewAIProviderService(useCase AIProviderUseCase) *AIProviderService {
	return &AIProviderService{useCase: useCase}
}

// Create 新增 ai 提供商
// @Summary 新增 ai 提供商
// @Description 创建 ai 提供商，API Key 入库前会加密
// @Tags ai-provider
// @Accept json
// @Produce json
// @Param provider body dto.CreateAIProviderRequest true "ai 提供商信息"
// @Success 200 {object} common.Response{data=aiProviderResponse}
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /ai-providers [post]
func (service *AIProviderService) Create(c *gin.Context) {
	var request dto.CreateAIProviderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		_ = c.Error(common.InvalidRequestWithValidationMessage(request, err))
		return
	}

	detail, err := service.useCase.Create(c.Request.Context(), request)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: toAIProviderDetailResponse(detail),
	})
}

// List 查询 ai 提供商列表
// @Summary 查询 ai 提供商列表
// @Description 分页查询 ai 提供商列表，列表不返回 API Key
// @Tags ai-provider
// @Produce json
// @Param page query int false "页码，默认 1"
// @Param pageSize query int false "每页数量，默认 10，最大 100"
// @Success 200 {object} common.Response{data=aiProviderListResponse}
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /ai-providers [get]
func (service *AIProviderService) List(c *gin.Context) {
	page, pageSize, ok := bindPagination(c)
	if !ok {
		return
	}

	result, err := service.useCase.List(c.Request.Context(), page, pageSize)
	if err != nil {
		_ = c.Error(err)
		return
	}

	items := make([]aiProviderResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toAIProviderListItemResponse(item))
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: &aiProviderListResponse{
			Items:    items,
			Total:    result.Total,
			Page:     result.Page,
			PageSize: result.PageSize,
		},
	})
}

// QueryModels 查询 ai 提供商模型列表
// @Summary 查询 ai 提供商模型列表
// @Description 使用前端传入的 baseUrl 和 apiKey，按 OpenAI 兼容 /v1/models 协议查询模型 ID 列表；不会写入数据库
// @Tags ai-provider
// @Accept json
// @Produce json
// @Param provider body dto.QueryAIProviderModelsRequest true "ai 提供商连接信息"
// @Success 200 {object} common.Response{data=aiProviderModelsResponse}
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /ai-providers/models/query [post]
func (service *AIProviderService) QueryModels(c *gin.Context) {
	var request dto.QueryAIProviderModelsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		_ = c.Error(common.InvalidRequestWithValidationMessage(request, err))
		return
	}

	models, err := service.useCase.QueryModels(c.Request.Context(), request)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: aiProviderModelsResponse{Models: models},
	})
}

// ListEnabledModels 查询已启用 ai 提供商保存的模型列表
// @Summary 查询已启用 ai 提供商保存的模型列表
// @Description 查询数据库中当前已启用 ai 提供商保存的模型列表，不请求上游 OpenAI 兼容 /v1/models 接口
// @Tags ai-provider
// @Produce json
// @Success 200 {object} common.Response{data=aiProviderModelsResponse}
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /ai-providers/models [get]
func (service *AIProviderService) ListEnabledModels(c *gin.Context) {
	models, err := service.useCase.ListEnabledModels(c.Request.Context())
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: aiProviderModelsResponse{Models: models},
	})
}

// Enable 一键启用 ai 提供商
// @Summary 一键启用 ai 提供商
// @Description 将当前已启用的 ai 提供商全部设为未启用，再启用请求中指定的 ai 提供商
// @Tags ai-provider
// @Accept json
// @Produce json
// @Param provider body dto.EnableAIProviderRequest true "需要启用的 ai 提供商 ID"
// @Success 200 {object} common.Response
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /ai-providers/enable [post]
func (service *AIProviderService) Enable(c *gin.Context) {
	var request dto.EnableAIProviderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		_ = c.Error(common.InvalidRequestWithValidationMessage(request, err))
		return
	}

	if err := service.useCase.Enable(c.Request.Context(), request.ID); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
	})
}

// Get 查询 ai 提供商详情
// @Summary 查询 ai 提供商详情
// @Description 按 ID 查询 ai 提供商详情，返回解密后的 API Key
// @Tags ai-provider
// @Produce json
// @Param id path int true "ai 提供商 ID"
// @Success 200 {object} common.Response{data=aiProviderResponse}
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /ai-providers/{id} [get]
func (service *AIProviderService) Get(c *gin.Context) {
	id, ok := parseAIProviderID(c)
	if !ok {
		return
	}

	detail, err := service.useCase.Get(c.Request.Context(), id)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: toAIProviderDetailResponse(detail),
	})
}

// Update 更新 ai 提供商
// @Summary 更新 ai 提供商
// @Description 按 ID 全量更新 ai 提供商，API Key 会重新加密
// @Tags ai-provider
// @Accept json
// @Produce json
// @Param id path int true "ai 提供商 ID"
// @Param provider body dto.UpdateAIProviderRequest true "ai 提供商信息"
// @Success 200 {object} common.Response{data=aiProviderResponse}
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /ai-providers/{id} [put]
func (service *AIProviderService) Update(c *gin.Context) {
	id, ok := parseAIProviderID(c)
	if !ok {
		return
	}

	var request dto.UpdateAIProviderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		slog.ErrorContext(c.Request.Context(), "更新 ai 提供商请求参数错误", "err", err)
		_ = c.Error(common.InvalidRequestWithValidationMessage(request, err))
		return
	}
	request.ID = id

	detail, err := service.useCase.Update(c.Request.Context(), request)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: toAIProviderDetailResponse(detail),
	})
}

// Delete 删除 ai 提供商
// @Summary 删除 ai 提供商
// @Description 按 ID 物理删除 ai 提供商
// @Tags ai-provider
// @Produce json
// @Param id path int true "ai 提供商 ID"
// @Success 200 {object} common.Response
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /ai-providers/{id} [delete]
func (service *AIProviderService) Delete(c *gin.Context) {
	id, ok := parseAIProviderID(c)
	if !ok {
		return
	}

	if err := service.useCase.Delete(c.Request.Context(), id); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
	})
}

func parseAIProviderID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		_ = c.Error(common.InvalidRequest)
		return 0, false
	}

	return id, true
}

func bindPagination(c *gin.Context) (int, int, bool) {
	page, ok := bindPositiveQuery(c, "page", 1)
	if !ok {
		return 0, 0, false
	}

	pageSize, ok := bindPositiveQuery(c, "pageSize", 10)
	if !ok {
		return 0, 0, false
	}
	if pageSize > maxAIProviderPageSize {
		pageSize = maxAIProviderPageSize
	}

	return page, pageSize, true
}

func bindPositiveQuery(c *gin.Context, key string, defaultValue int) (int, bool) {
	rawValue := c.Query(key)
	if rawValue == "" {
		return defaultValue, true
	}

	value, err := strconv.Atoi(rawValue)
	if err != nil || value <= 0 {
		_ = c.Error(common.InvalidRequest)
		return 0, false
	}

	return value, true
}

func toAIProviderDetailResponse(detail *aiproviderbiz.AIProviderDetail) aiProviderResponse {
	response := toAIProviderListItemResponse(&detail.Provider)
	response.APIKey = detail.APIKey

	return response
}

func toAIProviderListItemResponse(provider *aiproviderbiz.AIProvider) aiProviderResponse {
	configJSON := json.RawMessage(provider.ConfigJSON)
	if len(configJSON) == 0 {
		configJSON = json.RawMessage(`{}`)
	}

	return aiProviderResponse{
		ID:                provider.ID,
		Name:              provider.Name,
		ProviderType:      provider.ProviderType,
		APIKey:            provider.APIKeyEncrypted,
		BaseURL:           provider.BaseURL,
		IsEnabled:         provider.IsEnabled,
		ConfigJSON:        configJSON,
		Models:            []string(provider.Models),
		DefaultModel:      provider.DefaultModel,
		DefaultImageModel: provider.DefaultImageModel,
		MaxContextLength:  provider.MaxContextLength,
		MaxInputTokens:    provider.MaxInputTokens,
		MaxOutputTokens:   provider.MaxOutputTokens,
		CreatedAt:         provider.CreatedAt.Format("2006-01-02T15:04:05"),
		UpdatedAt:         provider.UpdatedAt.Format("2006-01-02T15:04:05"),
	}
}
