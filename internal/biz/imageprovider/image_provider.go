package imageprovider

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"Novels_AI/backend/internal/data"
	"Novels_AI/backend/internal/data/dto"
	"Novels_AI/backend/internal/pkg/common"

	"github.com/lib/pq"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	defaultImageGenerationTimeout = 120 * time.Second
)

type ImageAIProvider = data.ImageAIProvider

// APIKeyCipher 描述业务层需要的 API Key 加解密能力，便于单元测试替换。
type APIKeyCipher interface {
	Encrypt(plainText string) (string, error)
	Decrypt(cipherText string) (string, error)
}

// ImageAIProviderRepo 描述生图 AI 提供商业务依赖的数据访问能力。
type ImageAIProviderRepo interface {
	Create(ctx context.Context, provider *data.ImageAIProvider) (*data.ImageAIProvider, error)
	List(ctx context.Context, offset, limit int) ([]*data.ImageAIProvider, int64, error)
	FindByID(ctx context.Context, id int64) (*data.ImageAIProvider, error)
	FindEnabled(ctx context.Context) (*data.ImageAIProvider, error)
	Update(ctx context.Context, provider *data.ImageAIProvider) (*data.ImageAIProvider, error)
	Delete(ctx context.Context, id int64) error
	Enable(ctx context.Context, id int64) error
}

// UploadRepo 描述图片生成后写入对象存储所需的最小上传能力。
type UploadRepo interface {
	Upload(ctx context.Context, object data.UploadObject) (*data.UploadedObject, error)
}

// ImageGenerationHTTPClient 描述调用上游生图接口和下载图片 URL 时需要的 HTTP 能力。
type ImageGenerationHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// ImageAIProviderDetail 是带明文 API Key 的详情模型，只用于受保护的管理接口响应。
type ImageAIProviderDetail struct {
	Provider data.ImageAIProvider
	APIKey   string
}

type ListImageAIProviderResult struct {
	Items    []*data.ImageAIProvider
	Total    int64
	Page     int
	PageSize int
}

type GenerateImageResult struct {
	ImageURL   string
	ImageKey   string
	ModelName  string
	ProviderID int64
}

type ImageAIProviderUseCase struct {
	repo       ImageAIProviderRepo
	cipher     APIKeyCipher
	uploadRepo UploadRepo
	httpClient ImageGenerationHTTPClient
}

func NewImageAIProviderUseCase(repo ImageAIProviderRepo, cipher APIKeyCipher, uploadRepo UploadRepo) *ImageAIProviderUseCase {
	return &ImageAIProviderUseCase{
		repo:       repo,
		cipher:     cipher,
		uploadRepo: uploadRepo,
		httpClient: &http.Client{Timeout: defaultImageGenerationTimeout},
	}
}

// Create 整理默认值、加密 API Key 后创建生图 AI 提供商。
func (uc *ImageAIProviderUseCase) Create(ctx context.Context, params dto.CreateImageAIProviderRequest) (*ImageAIProviderDetail, error) {
	isEnabled := params.IsEnabled
	if isEnabled {
		if err := uc.ensureEnabledProviderUnique(ctx, 0); err != nil {
			slog.ErrorContext(ctx, "确保生图 AI 提供商启用唯一性失败", "err", err)
			return nil, err
		}
	}

	encrypted, err := uc.cipher.Encrypt(params.APIKey)
	if err != nil {
		slog.ErrorContext(ctx, "加密生图 AI 提供商 API Key 失败", "err", err)
		return nil, err
	}

	provider, err := uc.repo.Create(ctx, &data.ImageAIProvider{
		Name:            strings.TrimSpace(params.Name),
		ProviderType:    strings.TrimSpace(params.ProviderType),
		BaseURL:         strings.TrimSpace(params.BaseURL),
		APIKeyEncrypted: encrypted,
		IsEnabled:       isEnabled,
		ConfigJSON:      normalizeConfigJSON(params.ConfigJSON.Value),
		Models:          normalizeModels(params.Models),
		DefaultModel:    strings.TrimSpace(params.DefaultModel),
	})
	if err != nil {
		slog.ErrorContext(ctx, "新增生图 AI 提供商失败", "err", err)
		return nil, err
	}

	return uc.toDetail(ctx, provider)
}

// List 按分页参数查询生图 AI 提供商列表，列表不解密 API Key。
func (uc *ImageAIProviderUseCase) List(ctx context.Context, page, pageSize int) (*ListImageAIProviderResult, error) {
	offset := (page - 1) * pageSize

	items, total, err := uc.repo.List(ctx, offset, pageSize)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &ListImageAIProviderResult{Page: page, PageSize: pageSize}, nil
		}
		slog.ErrorContext(ctx, "查询生图 AI 提供商列表失败", "err", err)
		return nil, err
	}

	return &ListImageAIProviderResult{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// Get 查询生图 AI 提供商详情，并把入库密文解密为明文 API Key。
func (uc *ImageAIProviderUseCase) Get(ctx context.Context, id int64) (*ImageAIProviderDetail, error) {
	provider, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return uc.toDetail(ctx, provider)
}

// Update 按请求参数全量保存生图 AI 提供商，API Key 会在入库前重新加密。
func (uc *ImageAIProviderUseCase) Update(ctx context.Context, params dto.UpdateImageAIProviderRequest) (*ImageAIProviderDetail, error) {
	if params.IsEnabled {
		if err := uc.ensureEnabledProviderUnique(ctx, params.ID); err != nil {
			return nil, err
		}
	}

	encrypted, err := uc.cipher.Encrypt(params.APIKey)
	if err != nil {
		slog.ErrorContext(ctx, "加密生图 AI 提供商 API Key 失败", "err", err)
		return nil, err
	}

	provider, err := uc.repo.Update(ctx, &data.ImageAIProvider{
		ID:              params.ID,
		Name:            strings.TrimSpace(params.Name),
		ProviderType:    strings.TrimSpace(params.ProviderType),
		BaseURL:         strings.TrimSpace(params.BaseURL),
		APIKeyEncrypted: encrypted,
		IsEnabled:       params.IsEnabled,
		ConfigJSON:      normalizeConfigJSON(params.ConfigJSON.Value),
		Models:          normalizeModels(params.Models),
		DefaultModel:    strings.TrimSpace(params.DefaultModel),
	})
	if err != nil {
		slog.ErrorContext(ctx, "更新生图 AI 提供商失败", "err", err)
		return nil, err
	}

	return uc.toDetail(ctx, provider)
}

// Delete 物理删除生图 AI 提供商。
func (uc *ImageAIProviderUseCase) Delete(ctx context.Context, id int64) error {
	return uc.repo.Delete(ctx, id)
}

// Enable 一键启用指定生图 AI 提供商，具体切换过程由数据层在事务里完成。
func (uc *ImageAIProviderUseCase) Enable(ctx context.Context, id int64) error {
	if err := uc.repo.Enable(ctx, id); err != nil {
		slog.ErrorContext(ctx, "一键启用生图 AI 提供商失败", "id", id, "err", err)
		return err
	}

	return nil
}

// GenerateImage 使用当前启用的生图 AI 提供商生成图片，并把图片结果上传到对象存储。
func (uc *ImageAIProviderUseCase) GenerateImage(ctx context.Context, params dto.GenerateImageRequest) (*GenerateImageResult, error) {
	provider, err := uc.repo.FindEnabled(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "查询启用中的生图 AI 提供商失败", "err", err)
		return nil, err
	}
	if provider == nil {
		return nil, common.ImageAIProviderNotEnabled
	}

	modelName := strings.TrimSpace(params.ModelName)
	if modelName == "" {
		modelName = strings.TrimSpace(provider.DefaultModel)
	}
	if modelName == "" {
		return nil, common.ImageAIProviderDefaultModelNotConfigured
	}

	apiKey, err := uc.cipher.Decrypt(provider.APIKeyEncrypted)
	if err != nil {
		slog.ErrorContext(ctx, "解密生图 AI 提供商 API Key 失败", "providerID", provider.ID, "err", err)
		return nil, common.ImageAIProviderAPIKeyDecryptFailed
	}

	generated, err := uc.generateImage(ctx, provider.BaseURL, apiKey, modelName, strings.TrimSpace(params.Prompt))
	if err != nil {
		return nil, err
	}

	uploaded, err := uc.uploadGeneratedImage(ctx, generated)
	if err != nil {
		slog.ErrorContext(ctx, "上传 AI 生成图片失败", "providerID", provider.ID, "modelName", modelName, "err", err)
		return nil, common.ImageGenerationUploadFailed
	}

	return &GenerateImageResult{
		ImageURL:   uploaded.URL,
		ImageKey:   uploaded.Key,
		ModelName:  modelName,
		ProviderID: provider.ID,
	}, nil
}

// ensureEnabledProviderUnique 保证全局只有一个启用中的生图 AI 提供商；currentID 为当前更新记录，新增时传 0。
func (uc *ImageAIProviderUseCase) ensureEnabledProviderUnique(ctx context.Context, currentID int64) error {
	enabledProvider, err := uc.repo.FindEnabled(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "查询启用中的生图 AI 提供商失败", "err", err)
		return err
	}
	if enabledProvider == nil {
		return nil
	}
	if enabledProvider.ID == currentID {
		return nil
	}

	return common.ImageAIProviderEnabledConflict
}

func (uc *ImageAIProviderUseCase) toDetail(ctx context.Context, provider *data.ImageAIProvider) (*ImageAIProviderDetail, error) {
	apiKey, err := uc.cipher.Decrypt(provider.APIKeyEncrypted)
	if err != nil {
		slog.ErrorContext(ctx, "解密生图 AI 提供商 API Key 失败", "providerID", provider.ID, "err", err)
		return nil, common.ImageAIProviderAPIKeyDecryptFailed
	}

	return &ImageAIProviderDetail{
		Provider: *provider,
		APIKey:   apiKey,
	}, nil
}

type generatedImage struct {
	bytes       []byte
	contentType string
	fileName    string
}

type openAIImageGenerationRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	N      int    `json:"n"`
}

type openAIImageGenerationResponse struct {
	Data []struct {
		URL     string `json:"url"`
		B64JSON string `json:"b64_json"`
	} `json:"data"`
}

func (uc *ImageAIProviderUseCase) generateImage(ctx context.Context, baseURL, apiKey, modelName, prompt string) (*generatedImage, error) {
	endpoint, err := buildImageGenerationsURL(baseURL)
	if err != nil {
		return nil, common.InvalidRequest
	}

	body := openAIImageGenerationRequest{
		Model:  modelName,
		Prompt: prompt,
		N:      1,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, common.InvalidRequest
	}
	request.Header.Set("Authorization", "Bearer "+apiKey)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	response, err := uc.httpClient.Do(request)
	if err != nil {
		slog.ErrorContext(ctx, "调用生图 AI 提供商生成图片失败", "err", err)
		return nil, common.ImageGenerationFailed
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, response.Body)
		slog.ErrorContext(ctx, "生图 AI 提供商生成图片响应状态异常", "status", response.StatusCode)
		return nil, common.ImageGenerationFailed
	}

	var result openAIImageGenerationResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		slog.ErrorContext(ctx, "解析生图 AI 提供商响应失败", "err", err)
		return nil, common.ImageGenerationFailed
	}
	if len(result.Data) == 0 {
		return nil, common.ImageGenerationNoImage
	}

	item := result.Data[0]
	if strings.TrimSpace(item.B64JSON) != "" {
		return decodeBase64GeneratedImage(item.B64JSON)
	}
	if strings.TrimSpace(item.URL) != "" {
		return uc.downloadGeneratedImage(ctx, item.URL)
	}

	return nil, common.ImageGenerationNoImage
}

func (uc *ImageAIProviderUseCase) downloadGeneratedImage(ctx context.Context, imageURL string) (*generatedImage, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, common.ImageGenerationFailed
	}

	response, err := uc.httpClient.Do(request)
	if err != nil {
		slog.ErrorContext(ctx, "下载 AI 生成图片失败", "err", err)
		return nil, common.ImageGenerationFailed
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, response.Body)
		slog.ErrorContext(ctx, "下载 AI 生成图片响应状态异常", "status", response.StatusCode)
		return nil, common.ImageGenerationFailed
	}

	imageBytes, err := io.ReadAll(response.Body)
	if err != nil {
		slog.ErrorContext(ctx, "读取 AI 生成图片失败", "err", err)
		return nil, common.ImageGenerationFailed
	}
	if len(imageBytes) == 0 {
		return nil, common.ImageGenerationNoImage
	}

	contentType := normalizeImageContentType(response.Header.Get("Content-Type"), imageBytes)
	return &generatedImage{
		bytes:       imageBytes,
		contentType: contentType,
		fileName:    "generated-image" + imageExtension(contentType),
	}, nil
}

func (uc *ImageAIProviderUseCase) uploadGeneratedImage(ctx context.Context, image *generatedImage) (*data.UploadedObject, error) {
	return uc.uploadRepo.Upload(ctx, data.UploadObject{
		FileName:    image.fileName,
		ContentType: image.contentType,
		Size:        int64(len(image.bytes)),
		Body:        bytes.NewReader(image.bytes),
	})
}

func decodeBase64GeneratedImage(rawValue string) (*generatedImage, error) {
	value := strings.TrimSpace(rawValue)
	if comma := strings.Index(value, ","); comma >= 0 && strings.Contains(value[:comma], "base64") {
		value = value[comma+1:]
	}

	imageBytes, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, common.ImageGenerationFailed
	}
	if len(imageBytes) == 0 {
		return nil, common.ImageGenerationNoImage
	}

	return &generatedImage{
		bytes:       imageBytes,
		contentType: "image/png",
		fileName:    "generated-image.png",
	}, nil
}

func buildImageGenerationsURL(baseURL string) (string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("invalid image ai provider base url")
	}

	parsed.Path = strings.TrimRight(parsed.Path, "/")
	if !strings.HasSuffix(parsed.Path, "/v1") {
		parsed.Path += "/v1"
	}
	parsed.Path += "/images/generations"
	parsed.RawQuery = ""
	parsed.Fragment = ""

	return parsed.String(), nil
}

func normalizeConfigJSON(config datatypes.JSON) datatypes.JSON {
	if len(config) == 0 || string(config) == "null" {
		return datatypes.JSON([]byte("{}"))
	}

	return config
}

func normalizeModels(models []string) pq.StringArray {
	normalized := make([]string, 0, len(models))
	for _, model := range models {
		trimmed := strings.TrimSpace(model)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}

	return pq.StringArray(normalized)
}

func normalizeImageContentType(contentType string, imageBytes []byte) string {
	contentType = strings.TrimSpace(strings.Split(contentType, ";")[0])
	if strings.HasPrefix(contentType, "image/") {
		return contentType
	}

	return http.DetectContentType(imageBytes)
}

func imageExtension(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/png":
		return ".png"
	default:
		extensions, err := mime.ExtensionsByType(contentType)
		if err == nil && len(extensions) > 0 {
			return extensions[0]
		}
		return ".png"
	}
}
