package data

import (
	"context"
	"errors"

	"Novels_AI/backend/internal/data/model"
	"Novels_AI/backend/internal/pkg/common"

	"gorm.io/gorm"
)

type NovelMindMap = model.NovelMindMap

// NovelMindMapData 负责小说思维导图数据的持久化读写。
type NovelMindMapData struct {
	db *gorm.DB
}

func NewNovelMindMapData(db *gorm.DB) *NovelMindMapData {
	return &NovelMindMapData{db: db}
}

// FindByNovelID 查询单本小说当前保存的思维导图。
func (d *NovelMindMapData) FindByNovelID(ctx context.Context, novelID int64) (*NovelMindMap, error) {
	var mindMap NovelMindMap
	err := d.db.WithContext(ctx).
		Where("novel_id = ?", novelID).
		First(&mindMap).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.MindMapNotFound
		}
		return nil, err
	}

	return &mindMap, nil
}

// Save 使用完整模型保存思维导图；ID 为空时新增，已有 ID 时全量更新。
func (d *NovelMindMapData) Save(ctx context.Context, mindMap *NovelMindMap) (*NovelMindMap, error) {
	if err := d.db.WithContext(ctx).Save(mindMap).Error; err != nil {
		return nil, err
	}

	return d.FindByNovelID(ctx, mindMap.NovelID)
}
