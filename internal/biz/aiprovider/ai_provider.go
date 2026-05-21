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
	"Novels_AI/backend/internal/pkg/common"

	"github.com/lib/pq"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	defaultPage     = 1
	defaultPageSize = 10
	maxPageSize     = 100

	defaultAIProviderPriority = 100

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

// AIProviderRepo 描述 AI 提供商业务依赖的数据访问能力。
type AIProviderRepo interface {
	Create(ctx context.Context, provider *data.AIProvider) (*data.AIProvider, error)
	List(ctx context.Context, offset, limit int) ([]data.AIProvider, int64, error)
	FindByID(ctx context.Context, id int64) (*data.AIProvider, error)
	Update(ctx context.Context, id int64, values map[string]any) (*data.AIProvider, error)
	Delete(ctx context.Context, id int64) error
}

// AIProviderDetail 是带明文 API Key 的详情模型，只用于受保护的管理接口响应。
type AIProviderDetail struct {
	Provider data.AIProvider
	APIKey   string
}

type CreateAIProviderParams struct {
	Name         string
	ProviderType string
	BaseURL      string
	APIKey       string
	IsEnabled    *bool
	Priority     *int
	ConfigJSON   datatypes.JSON
	Models       []string
}

type UpdateAIProviderParams struct {
	ID           int64
	Name         *string
	ProviderType *string
	BaseURL      *string
	APIKey       *string
	IsEnabled    *bool
	Priority     *int
	ConfigJSON   *datatypes.JSON
	Models       *[]string
}

type QueryAIProviderModelsParams struct {
	BaseURL string
	APIKey  string
}

type ListAIProviderResult struct {
	Items    []data.AIProvider
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

// Create 整理默认值、加密 API Key 后创建 AI 提供商。
func (uc *AIProviderUseCase) Create(ctx context.Context, params CreateAIProviderParams) (*AIProviderDetail, error) {
	name, err := normalizeRequiredString(params.Name, common.AIProviderNameRequired)
	if err != nil {
		return nil, err
	}
	providerType, err := normalizeRequiredString(params.ProviderType, common.AIProviderTypeRequired)
	if err != nil {
		return nil, err
	}
	baseURL, err := normalizeRequiredString(params.BaseURL, common.AIProviderBaseURLRequired)
	if err != nil {
		return nil, err
	}
	apiKey, err := normalizeRequiredString(params.APIKey, common.AIProviderAPIKeyRequired)
	if err != nil {
		return nil, err
	}

	encrypted, err := uc.cipher.Encrypt(apiKey)
	if err != nil {
		slog.ErrorContext(ctx, "加密 AI 提供商 API Key 失败", "err", err)
		return nil, err
	}

	isEnabled := true
	if params.IsEnabled != nil {
		isEnabled = *params.IsEnabled
	}

	priority := defaultAIProviderPriority
	if params.Priority != nil {
		priority = *params.Priority
	}

	provider, err := uc.repo.Create(ctx, &data.AIProvider{
		Name:            name,
		ProviderType:    providerType,
		BaseURL:         baseURL,
		APIKeyEncrypted: encrypted,
		IsEnabled:       isEnabled,
		Priority:        priority,
		ConfigJSON:      normalizeConfigJSON(params.ConfigJSON),
		Models:          normalizeModels(params.Models),
	})
	if err != nil {
		slog.ErrorContext(ctx, "新增 AI 提供商失败", "err", err)
		return nil, err
	}

	return uc.toDetail(ctx, provider)
}

// List 按分页参数查询 AI 提供商列表，列表不解密 API Key。
func (uc *AIProviderUseCase) List(ctx context.Context, page, pageSize int) (*ListAIProviderResult, error) {
	page, pageSize = normalizePagination(page, pageSize)
	offset := (page - 1) * pageSize

	items, total, err := uc.repo.List(ctx, offset, pageSize)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &ListAIProviderResult{Page: page, PageSize: pageSize}, nil
		}
		slog.ErrorContext(ctx, "查询 AI 提供商列表失败", "err", err)
		return nil, err
	}

	return &ListAIProviderResult{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// Get 查询 AI 提供商详情，并把入库密文解密为明文 API Key。
func (uc *AIProviderUseCase) Get(ctx context.Context, id int64) (*AIProviderDetail, error) {
	provider, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return uc.toDetail(ctx, provider)
}

// Update 只更新请求中明确传入的字段，传入 API Key 时会重新加密。
func (uc *AIProviderUseCase) Update(ctx context.Context, params UpdateAIProviderParams) (*AIProviderDetail, error) {
	values := make(map[string]any)
	if params.Name != nil {
		name, err := normalizeRequiredString(*params.Name, common.AIProviderNameRequired)
		if err != nil {
			return nil, err
		}
		values["name"] = name
	}
	if params.ProviderType != nil {
		providerType, err := normalizeRequiredString(*params.ProviderType, common.AIProviderTypeRequired)
		if err != nil {
			return nil, err
		}
		values["provider_type"] = providerType
	}
	if params.BaseURL != nil {
		baseURL, err := normalizeRequiredString(*params.BaseURL, common.AIProviderBaseURLRequired)
		if err != nil {
			return nil, err
		}
		values["base_url"] = baseURL
	}
	if params.APIKey != nil {
		apiKey, err := normalizeRequiredString(*params.APIKey, common.AIProviderAPIKeyRequired)
		if err != nil {
			return nil, err
		}
		encrypted, err := uc.cipher.Encrypt(apiKey)
		if err != nil {
			slog.ErrorContext(ctx, "加密 AI 提供商 API Key 失败", "err", err)
			return nil, err
		}
		values["api_key_encrypted"] = encrypted
	}
	if params.IsEnabled != nil {
		values["is_enabled"] = *params.IsEnabled
	}
	if params.Priority != nil {
		values["priority"] = *params.Priority
	}
	if params.ConfigJSON != nil {
		values["config_json"] = normalizeConfigJSON(*params.ConfigJSON)
	}
	if params.Models != nil {
		values["models"] = normalizeModels(*params.Models)
	}

	provider, err := uc.repo.Update(ctx, params.ID, values)
	if err != nil {
		slog.ErrorContext(ctx, "更新 AI 提供商失败", "err", err)
		return nil, err
	}

	return uc.toDetail(ctx, provider)
}

// Delete 物理删除 AI 提供商。
func (uc *AIProviderUseCase) Delete(ctx context.Context, id int64) error {
	return uc.repo.Delete(ctx, id)
}

// QueryModels 按 OpenAI 兼容的 /v1/models 协议查询提供商支持的模型 ID 列表。
func (uc *AIProviderUseCase) QueryModels(ctx context.Context, params QueryAIProviderModelsParams) ([]string, error) {
	baseURL, err := normalizeRequiredString(params.BaseURL, common.AIProviderBaseURLRequired)
	if err != nil {
		return nil, err
	}
	apiKey, err := normalizeRequiredString(params.APIKey, common.AIProviderAPIKeyRequired)
	if err != nil {
		return nil, err
	}

	modelsURL, err := buildModelsURL(baseURL)
	if err != nil {
		return nil, common.InvalidRequest
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, modelsURL, nil)
	if err != nil {
		return nil, common.InvalidRequest
	}
	request.Header.Set("Authorization", "Bearer "+apiKey)
	request.Header.Set("Accept", "application/json")

	response, err := uc.modelClient.Do(request)
	if err != nil {
		slog.ErrorContext(ctx, "查询 AI 提供商模型列表失败", "err", err)
		return nil, common.AIProviderModelsQueryFailed
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, response.Body)
		slog.ErrorContext(ctx, "AI 提供商模型列表响应状态异常", "status", response.StatusCode)
		return nil, common.AIProviderModelsQueryFailed
	}

	var payload openAIModelsResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		slog.ErrorContext(ctx, "解析 AI 提供商模型列表失败", "err", err)
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
		slog.ErrorContext(ctx, "解密 AI 提供商 API Key 失败", "err", err)
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

func normalizeRequiredString(value string, requiredErr error) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", requiredErr
	}

	return trimmed, nil
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

func normalizePagination(page, pageSize int) (int, int) {
	if page <= 0 {
		page = defaultPage
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	return page, pageSize
}
