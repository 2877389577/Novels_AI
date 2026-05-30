package data

import (
	"context"
	"errors"

	"Novels_AI/backend/internal/data/model"
	"Novels_AI/backend/internal/pkg/common"

	"gorm.io/gorm"
)

type ImageAIProvider = model.ImageAIProvider

// ImageAIProviderData 负责专用生图 AI 提供商表的持久化读写。
type ImageAIProviderData struct {
	db *gorm.DB
}

// NewImageAIProviderData 创建生图 AI 提供商数据访问对象。
func NewImageAIProviderData(db *gorm.DB) *ImageAIProviderData {
	return &ImageAIProviderData{db: db}
}

// Create 保存生图 AI 提供商配置，API Key 在进入数据层前已经完成加密。
func (d *ImageAIProviderData) Create(ctx context.Context, provider *ImageAIProvider) (*ImageAIProvider, error) {
	if err := d.db.WithContext(ctx).
		// GORM 遇到带 default tag 的 bool 零值时可能让数据库默认值生效，这里显式选择字段以保证 false 能写入。
		Select("Name", "ProviderType", "BaseURL", "APIKeyEncrypted", "IsEnabled", "ConfigJSON", "Models", "DefaultModel").
		Create(provider).Error; err != nil {
		return nil, err
	}

	return provider, nil
}

// List 按 ID 倒序分页读取生图 AI 提供商列表。
func (d *ImageAIProviderData) List(ctx context.Context, offset, limit int) ([]*ImageAIProvider, int64, error) {
	var total int64
	err := d.db.WithContext(ctx).
		Model(&model.ImageAIProvider{}).
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	var providers []*ImageAIProvider
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

// FindByID 按主键读取生图 AI 提供商详情，找不到时返回统一业务错误。
func (d *ImageAIProviderData) FindByID(ctx context.Context, id int64) (*ImageAIProvider, error) {
	var provider ImageAIProvider
	err := d.db.WithContext(ctx).First(&provider, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.ImageAIProviderNotFound
		}
		return nil, err
	}

	return &provider, nil
}

// FindEnabled 查询当前启用中的生图 AI 提供商；没有启用记录时返回 nil。
func (d *ImageAIProviderData) FindEnabled(ctx context.Context) (*ImageAIProvider, error) {
	var provider ImageAIProvider
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

// Update 使用 GORM Save 全量保存生图 AI 提供商字段，并保留创建时间。
func (d *ImageAIProviderData) Update(ctx context.Context, provider *ImageAIProvider) (*ImageAIProvider, error) {
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

// Delete 物理删除生图 AI 提供商记录，当前表没有 deleted_at 字段。
func (d *ImageAIProviderData) Delete(ctx context.Context, id int64) error {
	result := d.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&model.ImageAIProvider{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return common.ImageAIProviderNotFound
	}

	return nil
}

// Enable 在一个事务中完成生图 AI 提供商启用状态切换，保证全局只有一个启用记录。
func (d *ImageAIProviderData) Enable(ctx context.Context, id int64) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 先关闭所有已启用的生图提供商，避免事务提交后出现两个启用记录。
		if err := tx.Model(&model.ImageAIProvider{}).
			Where("is_enabled = ?", true).
			Update("is_enabled", false).Error; err != nil {
			return err
		}

		// 再启用目标提供商；如果目标不存在，RowsAffected 为 0，事务会回滚上面的关闭操作。
		result := tx.Model(&model.ImageAIProvider{}).
			Where("id = ?", id).
			Update("is_enabled", true)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return common.ImageAIProviderNotFound
		}

		return nil
	})
}
