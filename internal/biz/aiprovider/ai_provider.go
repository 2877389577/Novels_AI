package aiprovider

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
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
	defaultModelQueryTimeout = 30 * time.Second
)

type AIProvider = data.AIProvider

// APIKeyCipher 描述业务层需要的 API Key 加解密能力，便于单元测试替换。
type APIKeyCipher interface {
	Encrypt(plainText string) (string, error)
	Decrypt(cipherText string) (string, error)
}

// ModelHTTPClient 描述查询 OpenAI 兼容模型列表时需要的最小 HTTP 能力，方便测试替换。
type ModelHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// AIProviderRepo 描述 ai 提供商业务依赖的数据访问能力。
type AIProviderRepo interface {
	Create(ctx context.Context, provider *data.AIProvider) (*data.AIProvider, error)
	List(ctx context.Context, offset, limit int) ([]*data.AIProvider, int64, error)
	FindByID(ctx context.Context, id int64) (*data.AIProvider, error)
	FindEnabled(ctx context.Context) (*data.AIProvider, error)
	Update(ctx context.Context, provider *data.AIProvider) (*data.AIProvider, error)
	Delete(ctx context.Context, id int64) error
	Enable(ctx context.Context, id int64) error
}

// AIProviderDetail 是带明文 API Key 的详情模型，只用于受保护的管理接口响应。
type AIProviderDetail struct {
	Provider data.AIProvider
	APIKey   string
}

type ListAIProviderResult struct {
	Items    []*data.AIProvider
	Total    int64
	Page     int
	PageSize int
}

type AIProviderUseCase struct {
	repo        AIProviderRepo
	cipher      APIKeyCipher
	modelClient ModelHTTPClient
}

func NewAIProviderUseCase(repo AIProviderRepo, cipher APIKeyCipher) *AIProviderUseCase {
	return &AIProviderUseCase{
		repo:        repo,
		cipher:      cipher,
		modelClient: &http.Client{Timeout: defaultModelQueryTimeout},
	}
}

// Create 整理默认值、加密 API Key 后创建 ai 提供商。
func (uc *AIProviderUseCase) Create(ctx context.Context, params dto.CreateAIProviderRequest) (*AIProviderDetail, error) {
	isEnabled := params.IsEnabled
	if isEnabled {
		if err := uc.ensureEnabledProviderUnique(ctx, 0); err != nil {
			slog.ErrorContext(ctx, "确保 ai 提供商启用唯一性失败", "err", err)
			return nil, err
		}
	}

	encrypted, err := uc.cipher.Encrypt(params.APIKey)
	if err != nil {
		slog.ErrorContext(ctx, "加密 ai 提供商 API Key 失败", "err", err)
		return nil, err
	}

	provider, err := uc.repo.Create(ctx, &data.AIProvider{
		Name:             params.Name,
		ProviderType:     params.ProviderType,
		BaseURL:          params.BaseURL,
		APIKeyEncrypted:  encrypted,
		IsEnabled:        isEnabled,
		ConfigJSON:       normalizeConfigJSON(params.ConfigJSON.Value),
		Models:           normalizeModels(params.Models),
		DefaultModel:     strings.TrimSpace(params.DefaultModel),
		MaxContextLength: params.MaxContextLength,
		MaxInputTokens:   params.MaxInputTokens,
		MaxOutputTokens:  params.MaxOutputTokens,
	})
	if err != nil {
		slog.ErrorContext(ctx, "新增 ai 提供商失败", "err", err)
		return nil, err
	}

	return uc.toDetail(ctx, provider)
}

// List 按分页参数查询 ai 提供商列表，列表不解密 API Key。
func (uc *AIProviderUseCase) List(ctx context.Context, page, pageSize int) (*ListAIProviderResult, error) {
	offset := (page - 1) * pageSize

	items, total, err := uc.repo.List(ctx, offset, pageSize)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &ListAIProviderResult{Page: page, PageSize: pageSize}, nil
		}
		slog.ErrorContext(ctx, "查询 ai 提供商列表失败", "err", err)
		return nil, err
	}

	for _, v := range items {
		decrypt, err := uc.cipher.Decrypt(v.APIKeyEncrypted)
		if err != nil {
			slog.ErrorContext(ctx, "解密 ai 提供商 API Key 失败", "err", err)
		}
		v.APIKeyEncrypted = decrypt
	}

	return &ListAIProviderResult{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// Get 查询 ai 提供商详情，并把入库密文解密为明文 API Key。
func (uc *AIProviderUseCase) Get(ctx context.Context, id int64) (*AIProviderDetail, error) {
	provider, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return uc.toDetail(ctx, provider)
}

// Update 按请求参数全量保存 ai 提供商，API Key 会在入库前重新加密。
func (uc *AIProviderUseCase) Update(ctx context.Context, params dto.UpdateAIProviderRequest) (*AIProviderDetail, error) {
	if params.IsEnabled {
		if err := uc.ensureEnabledProviderUnique(ctx, params.ID); err != nil {
			return nil, err
		}
	}

	encrypted, err := uc.cipher.Encrypt(params.APIKey)
	if err != nil {
		slog.ErrorContext(ctx, "加密 ai 提供商 API Key 失败", "err", err)
		return nil, err
	}

	provider, err := uc.repo.Update(ctx, &data.AIProvider{
		ID:               params.ID,
		Name:             params.Name,
		ProviderType:     params.ProviderType,
		BaseURL:          params.BaseURL,
		APIKeyEncrypted:  encrypted,
		IsEnabled:        params.IsEnabled,
		ConfigJSON:       normalizeConfigJSON(params.ConfigJSON.Value),
		Models:           normalizeModels(params.Models),
		DefaultModel:     strings.TrimSpace(params.DefaultModel),
		MaxContextLength: params.MaxContextLength,
		MaxInputTokens:   params.MaxInputTokens,
		MaxOutputTokens:  params.MaxOutputTokens,
	})
	if err != nil {
		slog.ErrorContext(ctx, "更新 ai 提供商失败", "err", err)
		return nil, err
	}

	return uc.toDetail(ctx, provider)
}

// Delete 物理删除 ai 提供商。
func (uc *AIProviderUseCase) Delete(ctx context.Context, id int64) error {
	return uc.repo.Delete(ctx, id)
}

// Enable 一键启用指定 ai 提供商，具体切换过程由数据层在事务里完成。
func (uc *AIProviderUseCase) Enable(ctx context.Context, id int64) error {
	if err := uc.repo.Enable(ctx, id); err != nil {
		slog.ErrorContext(ctx, "一键启用 ai 提供商失败", "id", id, "err", err)
		return err
	}

	return nil
}

// ListEnabledModels 返回当前启用 ai 提供商保存在数据库中的模型列表，不请求上游接口。
func (uc *AIProviderUseCase) ListEnabledModels(ctx context.Context) ([]string, error) {
	provider, err := uc.repo.FindEnabled(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "查询启用中的 ai 提供商模型列表失败", "err", err)
		return nil, err
	}
	if provider == nil {
		return []string{}, nil
	}

	models := make([]string, 0, len(provider.Models))
	models = append(models, provider.Models...)
	return models, nil
}

// ensureEnabledProviderUnique 保证全局只有一个启用中的 ai 提供商；currentID 为当前更新记录，新增时传 0。
func (uc *AIProviderUseCase) ensureEnabledProviderUnique(ctx context.Context, currentID int64) error {
	enabledProvider, err := uc.repo.FindEnabled(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "查询启用中的 ai 提供商失败", "err", err)
		return err
	}
	if enabledProvider == nil {
		return nil
	}
	if enabledProvider.ID == currentID {
		return nil
	}

	return common.AIProviderEnabledConflict
}

// QueryModels 按 OpenAI 兼容的 /v1/models 协议查询提供商支持的模型 ID 列表。
func (uc *AIProviderUseCase) QueryModels(ctx context.Context, params dto.QueryAIProviderModelsRequest) ([]string, error) {
	modelsURL, err := buildModelsURL(params.BaseURL)
	if err != nil {
		return nil, common.InvalidRequest
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, modelsURL, nil)
	if err != nil {
		return nil, common.InvalidRequest
	}
	request.Header.Set("Authorization", "Bearer "+params.APIKey)
	request.Header.Set("Accept", "application/json")

	response, err := uc.modelClient.Do(request)
	if err != nil {
		slog.ErrorContext(ctx, "查询 ai 提供商模型列表失败", "err", err)
		return nil, common.AIProviderModelsQueryFailed
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, response.Body)
		slog.ErrorContext(ctx, "ai 提供商模型列表响应状态异常", "status", response.StatusCode)
		return nil, common.AIProviderModelsQueryFailed
	}

	var payload openAIModelsResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		slog.ErrorContext(ctx, "解析 ai 提供商模型列表失败", "err", err)
		return nil, common.AIProviderModelsQueryFailed
	}

	models := make([]string, 0, len(payload.Data))
	for _, item := range payload.Data {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		models = append(models, id)
	}
	if len(models) == 0 {
		return nil, common.AIProviderModelsQueryFailed
	}

	return models, nil
}

func (uc *AIProviderUseCase) toDetail(ctx context.Context, provider *data.AIProvider) (*AIProviderDetail, error) {
	apiKey, err := uc.cipher.Decrypt(provider.APIKeyEncrypted)
	if err != nil {
		slog.ErrorContext(ctx, "解密 ai 提供商 API Key 失败", "err", err)
		return nil, common.AIProviderAPIKeyDecryptFailed
	}

	return &AIProviderDetail{
		Provider: *provider,
		APIKey:   apiKey,
	}, nil
}

type openAIModelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

func buildModelsURL(baseURL string) (string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("invalid ai provider base url")
	}

	parsed.Path = strings.TrimRight(parsed.Path, "/")
	if !strings.HasSuffix(parsed.Path, "/v1") {
		parsed.Path += "/v1"
	}
	parsed.Path += "/models"
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
