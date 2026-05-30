package dto

// CreateImageAIProviderRequest 是新增生图 AI 提供商接口和业务层共用的请求参数。
type CreateImageAIProviderRequest struct {
	// 生图 AI 提供商名称，必填。
	Name string `json:"name" binding:"required" validatormsg:"生图AI提供商名称不能为空"`
	// 生图 AI 提供商类型，首版按 OpenAI 兼容协议调用，必填。
	ProviderType string `json:"providerType" binding:"required" validatormsg:"生图AI提供商类型不能为空"`
	// 生图 AI 提供商基础 URL，必填，业务层会拼接 /v1/images/generations。
	BaseURL string `json:"baseUrl" binding:"required" validatormsg:"生图AI提供商基础URL不能为空"`
	// API Key 明文，必填，业务层会在入库前加密。
	APIKey string `json:"apiKey" binding:"required" validatormsg:"生图AI提供商API密钥不能为空"`
	// 是否启用；启用时业务层会确保全局只有一个生图 AI 提供商处于启用状态。
	IsEnabled bool `json:"isEnabled"`
	// 生图 AI 提供商额外配置，预留给不同厂商的扩展参数。
	ConfigJSON JSONField `json:"configJson" swaggertype:"object"`
	// 支持的生图模型列表。
	Models []string `json:"models"`
	// 默认生图模型，必填；生成图片接口没有传 modelName 时会使用该模型。
	DefaultModel string `json:"defaultModel" binding:"required" validatormsg:"默认生图模型不能为空"`
}

// UpdateImageAIProviderRequest 是全量更新生图 AI 提供商接口和业务层共用的请求参数。
type UpdateImageAIProviderRequest struct {
	// ID 来自路径参数，不从请求体读取。
	ID int64 `json:"-"`
	// 生图 AI 提供商名称，必填。
	Name string `json:"name" binding:"required" validatormsg:"生图AI提供商名称不能为空"`
	// 生图 AI 提供商类型，首版按 OpenAI 兼容协议调用，必填。
	ProviderType string `json:"providerType" binding:"required" validatormsg:"生图AI提供商类型不能为空"`
	// 生图 AI 提供商基础 URL，必填，业务层会拼接 /v1/images/generations。
	BaseURL string `json:"baseUrl" binding:"required" validatormsg:"生图AI提供商基础URL不能为空"`
	// API Key 明文，必填，业务层会在入库前重新加密。
	APIKey string `json:"apiKey" binding:"required" validatormsg:"生图AI提供商API密钥不能为空"`
	// 是否启用；启用时业务层会确保全局只有一个生图 AI 提供商处于启用状态。
	IsEnabled bool `json:"isEnabled"`
	// 生图 AI 提供商额外配置，预留给不同厂商的扩展参数。
	ConfigJSON JSONField `json:"configJson" swaggertype:"object"`
	// 支持的生图模型列表。
	Models []string `json:"models"`
	// 默认生图模型，必填；生成图片接口没有传 modelName 时会使用该模型。
	DefaultModel string `json:"defaultModel" binding:"required" validatormsg:"默认生图模型不能为空"`
}

// EnableImageAIProviderRequest 是一键启用生图 AI 提供商接口的请求参数。
type EnableImageAIProviderRequest struct {
	// ID 是需要被设置为启用状态的生图 AI 提供商主键。
	ID int64 `json:"id" binding:"required,gt=0" validatormsg:"生图AI提供商ID不能为空"`
}

// GenerateImageRequest 是图片生成接口和业务层共用的请求参数。
type GenerateImageRequest struct {
	// Prompt 是图片生成提示词，必填。
	Prompt string `json:"prompt" binding:"required" validatormsg:"图片提示词不能为空"`
	// ModelName 是本次指定的生图模型，可为空；为空时使用启用提供商的 defaultModel。
	ModelName string `json:"modelName"`
}
