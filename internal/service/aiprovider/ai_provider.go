package aiprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	aiproviderbiz "Novels_AI/backend/internal/biz/aiprovider"
	"Novels_AI/backend/internal/pkg/common"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

type AIProviderService struct {
	useCase AIProviderUseCase
}

// AIProviderUseCase 描述 service 层依赖的 AI 提供商业务能力。
type AIProviderUseCase interface {
	Create(ctx context.Context, params aiproviderbiz.CreateAIProviderParams) (*aiproviderbiz.AIProviderDetail, error)
	List(ctx context.Context, page, pageSize int) (*aiproviderbiz.ListAIProviderResult, error)
	Get(ctx context.Context, id int64) (*aiproviderbiz.AIProviderDetail, error)
	Update(ctx context.Context, params aiproviderbiz.UpdateAIProviderParams) (*aiproviderbiz.AIProviderDetail, error)
	Delete(ctx context.Context, id int64) error
	QueryModels(ctx context.Context, params aiproviderbiz.QueryAIProviderModelsParams) ([]string, error)
}

type createAIProviderRequest struct {
	// AI 提供商名称，必填
	Name string `json:"name" binding:"required"`
	// AI 提供商类型，必填
	ProviderType string `json:"providerType" binding:"required"`
	// AI 提供商基础 URL，必填
	BaseURL string `json:"baseUrl" binding:"required"`
	// API Key 明文，必填，入库前会加密
	APIKey string `json:"apiKey" binding:"required"`
	// 是否启用；不传时默认 true
	IsEnabled *bool `json:"isEnabled"`
	// 优先级；不传时默认 100
	Priority *int `json:"priority"`
	// AI 提供商额外配置
	ConfigJSON rawJSONField `json:"configJson" swaggertype:"object"`
	// 支持的模型列表
	Models []string `json:"models"`
}

type updateAIProviderRequest struct {
	// AI 提供商名称
	Name *string `json:"name"`
	// AI 提供商类型
	ProviderType *string `json:"providerType"`
	// AI 提供商基础 URL
	BaseURL *string `json:"baseUrl"`
	// API Key 明文；传入时重新加密，不传则保留原值
	APIKey *string `json:"apiKey"`
	// 是否启用
	IsEnabled *bool `json:"isEnabled"`
	// 优先级
	Priority *int `json:"priority"`
	// AI 提供商额外配置
	ConfigJSON rawJSONField `json:"configJson" swaggertype:"object"`
	// 支持的模型列表；使用指针区分“未传”和“传空数组”
	Models *[]string `json:"models"`
}

type queryAIProviderModelsRequest struct {
	// AI 提供商基础 URL，后端会按 OpenAI 兼容协议拼接到 /v1/models。
	BaseURL string `json:"baseUrl" binding:"required"`
	// API Key 明文，仅用于本次上游查询，不入库也不返回。
	APIKey string `json:"apiKey" binding:"required"`
}

type rawJSONField struct {
	Set   bool
	Value datatypes.JSON
}

type aiProviderResponse struct {
	// AI 提供商 ID
	ID int64 `json:"id"`
	// AI 提供商名称
	Name string `json:"name"`
	// AI 提供商类型
	ProviderType string `json:"providerType"`
	// AI 提供商基础 URL
	BaseURL string `json:"baseUrl"`
	// API Key 明文，仅详情类响应返回
	APIKey string `json:"apiKey,omitempty"`
	// 是否启用
	IsEnabled bool `json:"isEnabled"`
	// 优先级
	Priority int `json:"priority"`
	// AI 提供商额外配置
	ConfigJSON json.RawMessage `json:"configJson" swaggertype:"object"`
	// 支持的模型列表
	Models []string `json:"models"`
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

// UnmarshalJSON 记录 configJson 字段是否出现，便于 PUT 区分“未传”和“传 null”。
func (r *rawJSONField) UnmarshalJSON(data []byte) error {
	r.Set = true
	if bytes.Equal(data, []byte("null")) {
		r.Value = datatypes.JSON([]byte("{}"))
		return nil
	}

	r.Value = datatypes.JSON(append([]byte(nil), data...))
	return nil
}

// Create 新增 AI 提供商
// @Summary 新增 AI 提供商
// @Description 创建 AI 提供商，API Key 入库前会加密
// @Tags ai-provider
// @Accept json
// @Produce json
// @Param provider body createAIProviderRequest true "AI 提供商信息"
// @Success 200 {object} common.Response{data=aiProviderResponse}
// @Failure 400 {object} common.Response "请求参数错误"
// @Failure 401 {object} common.Response "未登录"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /ai-providers [post]
func (service *AIProviderService) Create(c *gin.Context) {
	var request createAIProviderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		_ = c.Error(common.InvalidRequest)
		return
	}

	detail, err := service.useCase.Create(c.Request.Context(), aiproviderbiz.CreateAIProviderParams{
		Name:         request.Name,
		ProviderType: request.ProviderType,
		BaseURL:      request.BaseURL,
		APIKey:       request.APIKey,
		IsEnabled:    request.IsEnabled,
		Priority:     request.Priority,
		ConfigJSON:   request.ConfigJSON.Value,
		Models:       request.Models,
	})
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

// List 查询 AI 提供商列表
// @Summary 查询 AI 提供商列表
// @Description 分页查询 AI 提供商列表，列表不返回 API Key
// @Tags ai-provider
// @Produce json
// @Param page query int false "页码，默认 1"
// @Param pageSize query int false "每页数量，默认 10，最大 100"
// @Success 200 {object} common.Response{data=aiProviderListResponse}
// @Failure 400 {object} common.Response "请求参数错误"
// @Failure 401 {object} common.Response "未登录"
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
		items = append(items, toAIProviderListItemResponse(new(item)))
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

// QueryModels 查询 AI 提供商模型列表
// @Summary 查询 AI 提供商模型列表
// @Description 使用前端传入的 baseUrl 和 apiKey，按 OpenAI 兼容 /v1/models 协议查询模型 ID 列表；不会写入数据库
// @Tags ai-provider
// @Accept json
// @Produce json
// @Param provider body queryAIProviderModelsRequest true "AI 提供商连接信息"
// @Success 200 {object} common.Response{data=aiProviderModelsResponse}
// @Failure 400 {object} common.Response "请求参数错误"
// @Failure 401 {object} common.Response "未登录"
// @Failure 502 {object} common.Response "AI 提供商模型查询失败"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /ai-providers/models/query [post]
func (service *AIProviderService) QueryModels(c *gin.Context) {
	var request queryAIProviderModelsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		_ = c.Error(common.InvalidRequest)
		return
	}

	models, err := service.useCase.QueryModels(c.Request.Context(), aiproviderbiz.QueryAIProviderModelsParams{
		BaseURL: request.BaseURL,
		APIKey:  request.APIKey,
	})
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

// Get 查询 AI 提供商详情
// @Summary 查询 AI 提供商详情
// @Description 按 ID 查询 AI 提供商详情，返回解密后的 API Key
// @Tags ai-provider
// @Produce json
// @Param id path int true "AI 提供商 ID"
// @Success 200 {object} common.Response{data=aiProviderResponse}
// @Failure 400 {object} common.Response "请求参数错误"
// @Failure 401 {object} common.Response "未登录"
// @Failure 404 {object} common.Response "AI 提供商不存在"
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

// Update 更新 AI 提供商
// @Summary 更新 AI 提供商
// @Description 按 ID 局部更新 AI 提供商，传入 API Key 时重新加密
// @Tags ai-provider
// @Accept json
// @Produce json
// @Param id path int true "AI 提供商 ID"
// @Param provider body updateAIProviderRequest true "AI 提供商信息"
// @Success 200 {object} common.Response{data=aiProviderResponse}
// @Failure 400 {object} common.Response "请求参数错误"
// @Failure 401 {object} common.Response "未登录"
// @Failure 404 {object} common.Response "AI 提供商不存在"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /ai-providers/{id} [put]
func (service *AIProviderService) Update(c *gin.Context) {
	id, ok := parseAIProviderID(c)
	if !ok {
		return
	}

	var request updateAIProviderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		_ = c.Error(common.InvalidRequest)
		return
	}

	params := aiproviderbiz.UpdateAIProviderParams{
		ID:           id,
		Name:         request.Name,
		ProviderType: request.ProviderType,
		BaseURL:      request.BaseURL,
		APIKey:       request.APIKey,
		IsEnabled:    request.IsEnabled,
		Priority:     request.Priority,
		Models:       request.Models,
	}
	if request.ConfigJSON.Set {
		params.ConfigJSON = new(request.ConfigJSON.Value)
	}

	detail, err := service.useCase.Update(c.Request.Context(), params)
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

// Delete 删除 AI 提供商
// @Summary 删除 AI 提供商
// @Description 按 ID 物理删除 AI 提供商
// @Tags ai-provider
// @Produce json
// @Param id path int true "AI 提供商 ID"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response "请求参数错误"
// @Failure 401 {object} common.Response "未登录"
// @Failure 404 {object} common.Response "AI 提供商不存在"
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
		ID:           provider.ID,
		Name:         provider.Name,
		ProviderType: provider.ProviderType,
		BaseURL:      provider.BaseURL,
		IsEnabled:    provider.IsEnabled,
		Priority:     provider.Priority,
		ConfigJSON:   configJSON,
		Models:       []string(provider.Models),
		CreatedAt:    provider.CreatedAt.Format("2006-01-02T15:04:05"),
		UpdatedAt:    provider.UpdatedAt.Format("2006-01-02T15:04:05"),
	}
}
