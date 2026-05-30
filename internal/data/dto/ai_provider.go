package dto

import (
	"bytes"
	"encoding/json"

	"gorm.io/datatypes"
)

// JSONField 承接请求体中的任意 JSON 对象，并额外记录字段是否出现。
type JSONField struct {
	// Set 用于区分更新接口里的“未传 configJson”和“明确传入 configJson”。
	Set bool `json:"-"`
	// Value 保存原始 JSON 内容，业务层会在入库前把空值和 null 统一成默认对象。
	Value datatypes.JSON `json:"-"`
}

// CreateAIProviderRequest 是新增 ai 提供商接口和业务层共用的请求参数。
type CreateAIProviderRequest struct {
	// ai 提供商名称，必填。
	Name string `json:"name" binding:"required" validatormsg:"AI提供商名称不能为空"`
	// ai 提供商类型，必填。
	ProviderType string `json:"providerType" binding:"required" validatormsg:"AI提供商类型不能为空"`
	// ai 提供商基础 URL，必填。
	BaseURL string `json:"baseUrl" binding:"required" validatormsg:"AI提供商基础URL不能为空"`
	// API Key 明文，必填，业务层会在入库前加密。
	APIKey string `json:"apiKey" binding:"required" validatormsg:"AI提供商API密钥不能为空"`
	// 是否启用；不传时由业务层默认启用。
	IsEnabled bool `json:"isEnabled"`
	// ai 提供商额外配置。
	ConfigJSON JSONField `json:"configJson" swaggertype:"object"`
	// 支持的模型列表。
	Models []string `json:"models"`
	// 默认模型，必填；章节剧情总结等自动 AI 任务会使用该模型。
	DefaultModel string `json:"defaultModel" binding:"required" validatormsg:"默认模型不能为空"`
	// 默认生图模型，非必填；需要图片生成能力的业务可优先读取该字段。
	DefaultImageModel string `json:"defaultImageModel"`
	// 最大上下文长度，不传时使用数据库字段零值。
	MaxContextLength int64 `json:"maxContextLength"`
	// 最大输入令牌数，不传时使用数据库字段零值。
	MaxInputTokens int `json:"maxInputTokens"`
	// 最大输出令牌数，不传时使用数据库字段零值。
	MaxOutputTokens int `json:"maxOutputTokens"`
}

// UnmarshalJSON 记录 isEnabled 是否出现，避免显式传 false 被当成未传处理。
func (request *CreateAIProviderRequest) UnmarshalJSON(data []byte) error {
	type createAIProviderRequest CreateAIProviderRequest
	var parsed createAIProviderRequest
	if err := json.Unmarshal(data, &parsed); err != nil {
		return err
	}

	*request = CreateAIProviderRequest(parsed)
	return nil
}

// UpdateAIProviderRequest 是全量更新 ai 提供商接口和业务层共用的请求参数。
type UpdateAIProviderRequest struct {
	// ID 来自路径参数，不从请求体读取。
	ID int64 `json:"-"`
	// ai 提供商名称，必填。
	Name string `json:"name" binding:"required" validatormsg:"AI提供商名称不能为空"`
	// ai 提供商类型，必填。
	ProviderType string `json:"providerType" binding:"required" validatormsg:"AI提供商类型不能为空"`
	// ai 提供商基础 URL，必填。
	BaseURL string `json:"baseUrl" binding:"required" validatormsg:"AI提供商基础URL不能为空"`
	// API Key 明文，必填，业务层会在入库前重新加密。
	APIKey string `json:"apiKey" binding:"required" validatormsg:"AI提供商API密钥不能为空"`
	// 是否启用。
	IsEnabled bool `json:"isEnabled"`
	// ai 提供商额外配置。
	ConfigJSON JSONField `json:"configJson" swaggertype:"object"`
	// 支持的模型列表。
	Models []string `json:"models"`
	// 默认模型，必填；章节剧情总结等自动 AI 任务会使用该模型。
	DefaultModel string `json:"defaultModel" binding:"required" validatormsg:"默认模型不能为空"`
	// 默认生图模型，非必填；为空表示当前提供商暂不配置默认生图模型。
	DefaultImageModel string `json:"defaultImageModel"`
	// 最大上下文长度。
	MaxContextLength int64 `json:"maxContextLength"`
	// 最大输入令牌数。
	MaxInputTokens int `json:"maxInputTokens"`
	// 最大输出令牌数。
	MaxOutputTokens int `json:"maxOutputTokens"`
}

// EnableAIProviderRequest 是一键启用 ai 提供商接口的请求参数。
type EnableAIProviderRequest struct {
	// ID 是需要被设置为启用状态的 ai 提供商主键。
	ID int64 `json:"id" binding:"required,gt=0" validatormsg:"AI提供商ID不能为空"`
}

// QueryAIProviderModelsRequest 是查询上游模型列表接口和业务层共用的请求参数。
type QueryAIProviderModelsRequest struct {
	// ai 提供商基础 URL，后端会按 OpenAI 兼容协议拼接到 /v1/models。
	BaseURL string `json:"baseUrl" binding:"required" validatormsg:"AI提供商基础URL不能为空"`
	// API Key 明文，仅用于本次上游查询，不入库也不返回。
	APIKey string `json:"apiKey" binding:"required" validatormsg:"AI提供商API密钥不能为空"`
}

// UnmarshalJSON 记录 configJson 字段是否出现，便于 PUT 区分“未传”和“传 null”。
func (field *JSONField) UnmarshalJSON(data []byte) error {
	field.Set = true
	if bytes.Equal(data, []byte("null")) {
		field.Value = datatypes.JSON([]byte("{}"))
		return nil
	}

	field.Value = datatypes.JSON(append([]byte(nil), data...))
	return nil
}

func findRawField(data []byte, key string) (json.RawMessage, bool) {
	var rawFields map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawFields); err != nil {
		return nil, false
	}

	rawValue, ok := rawFields[key]
	return rawValue, ok
}
