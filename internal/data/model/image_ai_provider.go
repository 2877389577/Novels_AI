package model

import (
	"time"

	"github.com/lib/pq"
	"gorm.io/datatypes"
)

// ImageAIProvider 保存专用图片生成 AI 提供商配置。
type ImageAIProvider struct {
	ID int64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID" json:"id"`

	Name string `gorm:"column:name;type:varchar(100);not null;comment:生图AI提供商名称" json:"name"`

	ProviderType string `gorm:"column:provider_type;type:varchar(50);not null;default:'openai_compatible';comment:生图AI提供商类型，首版按OpenAI兼容协议调用" json:"provider_type"`

	BaseURL string `gorm:"column:base_url;type:text;not null;comment:生图AI提供商基础URL" json:"base_url"`

	APIKeyEncrypted string `gorm:"column:api_key_encrypted;type:text;not null;comment:加密后的API密钥" json:"api_key_encrypted"`

	IsEnabled bool `gorm:"column:is_enabled;not null;default:false;uniqueIndex:uk_image_ai_providers_enabled,where:is_enabled = true;comment:是否启用，全局只能有一个启用中的生图AI提供商" json:"is_enabled"`

	ConfigJSON datatypes.JSON `gorm:"column:config_json;type:jsonb;not null;default:'{}';comment:生图AI提供商扩展配置JSON" json:"config_json"`

	Models       pq.StringArray `gorm:"column:models;type:text[];not null;default:ARRAY[]::text[];comment:支持的生图模型列表" json:"models"`
	DefaultModel string         `gorm:"column:default_model;type:varchar(255);not null;default:'';comment:默认生图模型，请求未指定模型时使用" json:"default_model"`

	CreatedAt time.Time `gorm:"column:created_at;not null;default:now();comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;default:now();comment:更新时间" json:"updated_at"`
}

func (ImageAIProvider) TableName() string {
	return "image_ai_providers"
}
