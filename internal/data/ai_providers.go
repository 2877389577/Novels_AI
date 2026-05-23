package data

import (
	"context"
	"errors"

	"Novels_AI/backend/internal/data/model"
	"Novels_AI/backend/internal/pkg/common"

	"gorm.io/gorm"
)

type AIProvider = model.AIProvider

// AIProviderData 负责 AI 提供商表的持久化读写。
type AIProviderData struct {
	db *gorm.DB
}

// NewAIProviderData 创建 AI 提供商数据访问对象。
func NewAIProviderData(db *gorm.DB) *AIProviderData {
	return &AIProviderData{db: db}
}

// Create 保存 AI 提供商配置，API Key 在进入数据层前已经完成加密。
func (d *AIProviderData) Create(ctx context.Context, provider *AIProvider) (*AIProvider, error) {
	if err := d.db.WithContext(ctx).
		// GORM 遇到带 default tag 的 bool 零值时可能让数据库默认值生效，这里显式选择字段以保证 false 能写入。
		Select("Name", "ProviderType", "BaseURL", "APIKeyEncrypted", "IsEnabled", "ConfigJSON", "Models", "MaxContextLength", "MaxInputTokens", "MaxOutputTokens").
		Create(provider).Error; err != nil {
		return nil, err
	}

	return provider, nil
}

// List 按 ID 倒序分页读取 AI 提供商列表，避免依赖已删除的旧排序字段。
func (d *AIProviderData) List(ctx context.Context, offset, limit int) ([]*AIProvider, int64, error) {
	var total int64
	err := d.db.WithContext(ctx).
		Model(&model.AIProvider{}).
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	var providers []*AIProvider
	err = d.db.WithContext(ctx).
		Order("id DESC").
		Offset(offset).
		Limit(limit).
		Find(&providers).Error
	if err != nil {
		return nil, 0, err
	}

	return providers, total, nil
}

// FindByID 按主键读取 AI 提供商详情，找不到时返回统一业务错误。
func (d *AIProviderData) FindByID(ctx context.Context, id int64) (*AIProvider, error) {
	var provider AIProvider
	err := d.db.WithContext(ctx).First(&provider, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.AIProviderNotFound
		}
		return nil, err
	}

	return &provider, nil
}

// FindEnabled 查询当前启用中的 AI 提供商；没有启用记录时返回 nil，交由业务层决定是否允许后续写入。
func (d *AIProviderData) FindEnabled(ctx context.Context) (*AIProvider, error) {
	var provider AIProvider
	err := d.db.WithContext(ctx).
		Where("is_enabled = ?", true).
		First(&provider).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &provider, nil
}

// Update 使用 GORM Save 全量保存 AI 提供商字段，并保留创建时间。
func (d *AIProviderData) Update(ctx context.Context, provider *AIProvider) (*AIProvider, error) {
	current, err := d.FindByID(ctx, provider.ID)
	if err != nil {
		return nil, err
	}

	provider.CreatedAt = current.CreatedAt
	if err := d.db.WithContext(ctx).Save(provider).Error; err != nil {
		return nil, err
	}

	return d.FindByID(ctx, provider.ID)
}

// Delete 物理删除 AI 提供商记录，当前表没有 deleted_at 字段。
func (d *AIProviderData) Delete(ctx context.Context, id int64) error {
	result := d.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&model.AIProvider{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return common.AIProviderNotFound
	}

	return nil
}
