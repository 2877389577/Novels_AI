package imageprovider

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	imageproviderbiz "Novels_AI/backend/internal/biz/imageprovider"
	"Novels_AI/backend/internal/data/dto"
	"Novels_AI/backend/internal/pkg/common"

	"github.com/gin-gonic/gin"
)

const maxImageAIProviderPageSize = 100

type ImageAIProviderService struct {
	useCase ImageAIProviderUseCase
}

// ImageAIProviderUseCase 描述 service 层依赖的生图 AI 提供商业务能力。
type ImageAIProviderUseCase interface {
	Create(ctx context.Context, params dto.CreateImageAIProviderRequest) (*imageproviderbiz.ImageAIProviderDetail, error)
	List(ctx context.Context, page, pageSize int) (*imageproviderbiz.ListImageAIProviderResult, error)
	Get(ctx context.Context, id int64) (*imageproviderbiz.ImageAIProviderDetail, error)
	Update(ctx context.Context, params dto.UpdateImageAIProviderRequest) (*imageproviderbiz.ImageAIProviderDetail, error)
	Delete(ctx context.Context, id int64) error
	Enable(ctx context.Context, id int64) error
	GenerateImage(ctx context.Context, params dto.GenerateImageRequest) (*imageproviderbiz.GenerateImageResult, error)
}

type imageAIProviderResponse struct {
	// 生图 AI 提供商 ID
	ID int64 `json:"id"`
	// 生图 AI 提供商名称
	Name string `json:"name"`
	// 生图 AI 提供商类型
	ProviderType string `json:"providerType"`
	// 生图 AI 提供商基础 URL
	BaseURL string `json:"baseUrl"`
	// API Key 明文，仅详情类响应返回
	APIKey string `json:"apiKey,omitempty"`
	// 是否启用
	IsEnabled bool `json:"isEnabled"`
	// 生图 AI 提供商额外配置
	ConfigJSON json.RawMessage `json:"configJson" swaggertype:"object"`
	// 支持的生图模型列表
	Models []string `json:"models"`
	// 默认生图模型
	DefaultModel string `json:"defaultModel"`
	// 创建时间
	CreatedAt string `json:"createdAt"`
	// 更新时间
	UpdatedAt string `json:"updatedAt"`
}

type imageAIProviderListResponse struct {
	Items    []imageAIProviderResponse `json:"items"`
	Total    int64                     `json:"total"`
	Page     int                       `json:"page"`
	PageSize int                       `json:"pageSize"`
}

type generateImageResponse struct {
	// S3 临时签名访问 URL
	ImageURL string `json:"imageUrl"`
	// S3 对象 Key
	ImageKey string `json:"imageKey"`
	// 本次实际使用的生图模型
	ModelName string `json:"modelName"`
	// 本次实际使用的生图 AI 提供商 ID
	ProviderID int64 `json:"providerId"`
}

func NewImageAIProviderService(useCase ImageAIProviderUseCase) *ImageAIProviderService {
	return &ImageAIProviderService{useCase: useCase}
}

// Create 新增生图 AI 提供商
// @Summary 新增生图 AI 提供商
// @Description 创建专用图片生成 AI 提供商，API Key 入库前会加密；defaultModel 为生成图片接口未指定模型时使用的默认生图模型
// @Tags image-ai-provider
// @Accept json
// @Produce json
// @Param provider body dto.CreateImageAIProviderRequest true "生图 AI 提供商信息"
// @Success 200 {object} common.Response{data=imageAIProviderResponse}
// @Failure 400 {object} common.Response "请求参数错误"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /image-ai-providers [post]
func (service *ImageAIProviderService) Create(c *gin.Context) {
	var request dto.CreateImageAIProviderRequest
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
		Data: toImageAIProviderDetailResponse(detail),
	})
}

// List 查询生图 AI 提供商列表
// @Summary 查询生图 AI 提供商列表
// @Description 分页查询专用图片生成 AI 提供商列表，列表不返回 API Key
// @Tags image-ai-provider
// @Produce json
// @Param page query int false "页码，默认 1"
// @Param pageSize query int false "每页数量，默认 10，最大 100"
// @Success 200 {object} common.Response{data=imageAIProviderListResponse}
// @Failure 400 {object} common.Response "请求参数错误"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /image-ai-providers [get]
func (service *ImageAIProviderService) List(c *gin.Context) {
	page, pageSize, ok := bindPagination(c)
	if !ok {
		return
	}

	result, err := service.useCase.List(c.Request.Context(), page, pageSize)
	if err != nil {
		_ = c.Error(err)
		return
	}

	items := make([]imageAIProviderResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toImageAIProviderListItemResponse(item))
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: &imageAIProviderListResponse{
			Items:    items,
			Total:    result.Total,
			Page:     result.Page,
			PageSize: result.PageSize,
		},
	})
}

// Get 查询生图 AI 提供商详情
// @Summary 查询生图 AI 提供商详情
// @Description 按 ID 查询专用图片生成 AI 提供商详情，返回解密后的 API Key 供编辑页回显
// @Tags image-ai-provider
// @Produce json
// @Param id path int true "生图 AI 提供商 ID"
// @Success 200 {object} common.Response{data=imageAIProviderResponse}
// @Failure 400 {object} common.Response "请求参数错误"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /image-ai-providers/{id} [get]
func (service *ImageAIProviderService) Get(c *gin.Context) {
	id, ok := parseImageAIProviderID(c)
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
		Data: toImageAIProviderDetailResponse(detail),
	})
}

// Update 更新生图 AI 提供商
// @Summary 更新生图 AI 提供商
// @Description 按 ID 全量更新专用图片生成 AI 提供商，API Key 会重新加密
// @Tags image-ai-provider
// @Accept json
// @Produce json
// @Param id path int true "生图 AI 提供商 ID"
// @Param provider body dto.UpdateImageAIProviderRequest true "生图 AI 提供商信息"
// @Success 200 {object} common.Response{data=imageAIProviderResponse}
// @Failure 400 {object} common.Response "请求参数错误"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /image-ai-providers/{id} [put]
func (service *ImageAIProviderService) Update(c *gin.Context) {
	id, ok := parseImageAIProviderID(c)
	if !ok {
		return
	}

	var request dto.UpdateImageAIProviderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		slog.ErrorContext(c.Request.Context(), "更新生图 AI 提供商请求参数错误", "err", err)
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
		Data: toImageAIProviderDetailResponse(detail),
	})
}

// Delete 删除生图 AI 提供商
// @Summary 删除生图 AI 提供商
// @Description 按 ID 物理删除专用图片生成 AI 提供商
// @Tags image-ai-provider
// @Produce json
// @Param id path int true "生图 AI 提供商 ID"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response "请求参数错误"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /image-ai-providers/{id} [delete]
func (service *ImageAIProviderService) Delete(c *gin.Context) {
	id, ok := parseImageAIProviderID(c)
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

// Enable 一键启用生图 AI 提供商
// @Summary 一键启用生图 AI 提供商
// @Description 将当前已启用的生图 AI 提供商全部设为未启用，再启用请求中指定的生图 AI 提供商
// @Tags image-ai-provider
// @Accept json
// @Produce json
// @Param provider body dto.EnableImageAIProviderRequest true "需要启用的生图 AI 提供商 ID"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response "请求参数错误"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /image-ai-providers/enable [post]
func (service *ImageAIProviderService) Enable(c *gin.Context) {
	var request dto.EnableImageAIProviderRequest
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

// GenerateImage 生成图片
// @Summary 生成图片
// @Description 使用当前已启用的专用生图 AI 提供商生成图片，并上传到 S3 兼容对象存储。请求未传 modelName 时使用已启用提供商的 defaultModel
// @Tags image-generation
// @Accept json
// @Produce json
// @Param image body dto.GenerateImageRequest true "图片生成请求参数。prompt 为图片提示词，必填；modelName 可为空"
// @Success 200 {object} common.Response{data=generateImageResponse}
// @Failure 400 {object} common.Response "请求参数错误"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /images/generate [post]
func (service *ImageAIProviderService) GenerateImage(c *gin.Context) {
	var request dto.GenerateImageRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		_ = c.Error(common.InvalidRequestWithValidationMessage(request, err))
		return
	}

	result, err := service.useCase.GenerateImage(c.Request.Context(), request)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: &generateImageResponse{
			ImageURL:   result.ImageURL,
			ImageKey:   result.ImageKey,
			ModelName:  result.ModelName,
			ProviderID: result.ProviderID,
		},
	})
}

func parseImageAIProviderID(c *gin.Context) (int64, bool) {
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
	if pageSize > maxImageAIProviderPageSize {
		pageSize = maxImageAIProviderPageSize
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

func toImageAIProviderDetailResponse(detail *imageproviderbiz.ImageAIProviderDetail) imageAIProviderResponse {
	response := toImageAIProviderListItemResponse(&detail.Provider)
	response.APIKey = detail.APIKey

	return response
}

func toImageAIProviderListItemResponse(provider *imageproviderbiz.ImageAIProvider) imageAIProviderResponse {
	configJSON := json.RawMessage(provider.ConfigJSON)
	if len(configJSON) == 0 {
		configJSON = json.RawMessage(`{}`)
	}

	return imageAIProviderResponse{
		ID:           provider.ID,
		Name:         provider.Name,
		ProviderType: provider.ProviderType,
		BaseURL:      provider.BaseURL,
		IsEnabled:    provider.IsEnabled,
		ConfigJSON:   configJSON,
		Models:       []string(provider.Models),
		DefaultModel: provider.DefaultModel,
		CreatedAt:    provider.CreatedAt.Format("2006-01-02T15:04:05"),
		UpdatedAt:    provider.UpdatedAt.Format("2006-01-02T15:04:05"),
	}
}
