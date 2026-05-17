package data

import (
	"context"

	"Novels_AI/backend/internal/data/model"
	"Novels_AI/backend/internal/pkg/common"

	"gorm.io/gorm"
)

// ErrNovelNotFound 复用公共业务错误，保证 service 通过 gin.Context.Error 交给中间件后仍能返回 404。
var ErrNovelNotFound = common.NovelNotFound

type Novel = model.Novel

type NovelData struct {
	db *gorm.DB
}

func NewNovelData(db *gorm.DB) *NovelData {
	return &NovelData{db: db}
}

// Create 保存小说基础信息，默认值由 biz 层在入库前整理。
func (n *NovelData) Create(ctx context.Context, novel *Novel) (*Novel, error) {
	err := n.db.WithContext(ctx).Create(novel).Error
	if err != nil {
		return nil, err
	}

	return novel, nil
}

// List 按 ID 倒序分页读取未软删除的小说。
func (n *NovelData) List(ctx context.Context, offset, limit int) ([]Novel, int64, error) {
	var total int64
	err := n.db.WithContext(ctx).Model(&model.Novel{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	var novels []Novel
	err = n.db.WithContext(ctx).
		Order("id DESC").
		Offset(offset).
		Limit(limit).
		Find(&novels).Error
	if err != nil {
		return nil, 0, err
	}

	return novels, total, nil
}

// FindByID 读取单本小说，GORM 会自动过滤已软删除记录。
func (n *NovelData) FindByID(ctx context.Context, id uint) (*Novel, error) {
	var novel Novel
	err := n.db.WithContext(ctx).First(&novel, id).Error
	if err != nil {
		return nil, err
	}

	return &novel, nil
}

// Update 按字段局部更新小说，空 map 时只校验并返回当前记录。
func (n *NovelData) Update(ctx context.Context, id uint, values map[string]any) (*Novel, error) {
	if len(values) == 0 {
		return n.FindByID(ctx, id)
	}

	result := n.db.WithContext(ctx).
		Model(&model.Novel{}).
		Where("id = ?", id).
		Updates(values)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, ErrNovelNotFound
	}

	return n.FindByID(ctx, id)
}

// Delete 使用 GORM 软删除，保留历史数据。
func (n *NovelData) Delete(ctx context.Context, id uint) error {
	return n.db.WithContext(ctx).Delete(&model.Novel{}, id).Error

}
