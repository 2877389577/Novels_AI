package model

import (
	"time"

	"github.com/lib/pq"
	"gorm.io/datatypes"
)

// AIProvider AI提供商表
type AIProvider struct {
	ID int64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID" json:"id"`

	Name string `gorm:"column:name;type:varchar(100);not null;comment:AI名称" json:"name"`

	ProviderType string `gorm:"column:provider_type;type:varchar(50);not null;comment:AI提供商类型" json:"provider_type"`

	BaseURL string `gorm:"column:base_url;not null;type:text;comment:AI基础URL" json:"base_url"`

	APIKeyEncrypted string `gorm:"column:api_key_encrypted;not null;type:text;comment:加密后的API密钥" json:"api_key_encrypted"`

	IsEnabled bool `gorm:"column:is_enabled;not null;comment:是否启用" json:"is_enabled"`

	ConfigJSON datatypes.JSON `gorm:"column:config_json;type:jsonb;not null;default:'{}';comment:配置JSON" json:"config_json"`

	Models           pq.StringArray `gorm:"column:models;type:text;comment:支持的模型列表" json:"models"`
	MaxContextLength int64          `gorm:"column:max_context_length;type:bigint;comment:最大上下文长度" json:"max_context_length"`
	MaxInputTokens   int            `gorm:"column:max_input_tokens;type:int;comment:最大输入令牌数" json:"max_input_tokens"`
	MaxOutputTokens  int            `gorm:"column:max_output_tokens;type:int;comment:最大输出令牌数" json:"max_output_tokens"`

	CreatedAt time.Time `gorm:"column:created_at;not null;default:now();comment:创建时间" json:"created_at"`

	UpdatedAt time.Time `gorm:"column:updated_at;not null;default:now();comment:更新时间" json:"updated_at"`
}

func (AIProvider) TableName() string {
	return "ai_providers"
}
