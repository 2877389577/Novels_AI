package data

import (
	"context"
	"errors"

	"Novels_AI/backend/internal/data/model"
	"Novels_AI/backend/internal/pkg/common"

	"gorm.io/gorm"
)

const (
	// AITaskCodeChapterPlotAnalysis 是自动章节剧情总结任务的稳定编码。
	AITaskCodeChapterPlotAnalysis = model.AITaskCodeChapterPlotAnalysis
	// AITaskNameChapterPlotAnalysis 是自动章节剧情总结任务在后台配置页展示的名称。
	AITaskNameChapterPlotAnalysis = model.AITaskNameChapterPlotAnalysis
	// AITaskDescriptionChapterPlotAnalysis 说明自动章节剧情总结的触发场景。
	AITaskDescriptionChapterPlotAnalysis = model.AITaskDescriptionChapterPlotAnalysis
)

type AITaskConfig = model.AITaskConfig

// AITaskConfigData 负责主动执行 AI 任务配置表的持久化读写。
type AITaskConfigData struct {
	db *gorm.DB
}

// NewAITaskConfigData 创建主动执行 AI 任务配置数据访问对象。
func NewAITaskConfigData(db *gorm.DB) *AITaskConfigData {
	return &AITaskConfigData{db: db}
}

// List 读取全部主动执行 AI 任务配置，按任务编码排序保证后台展示稳定。
func (d *AITaskConfigData) List(ctx context.Context) ([]*AITaskConfig, error) {
	var configs []*AITaskConfig
	if err := d.db.WithContext(ctx).
		Order("task_code ASC").
		Find(&configs).Error; err != nil {
		return nil, err
	}

	return configs, nil
}

// FindByTaskCode 按任务编码读取配置，找不到时返回统一业务错误。
func (d *AITaskConfigData) FindByTaskCode(ctx context.Context, taskCode string) (*AITaskConfig, error) {
	var config AITaskConfig
	err := d.db.WithContext(ctx).
		Where("task_code = ?", taskCode).
		First(&config).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.AITaskConfigNotFound
		}
		return nil, err
	}

	return &config, nil
}

// UpdateEnabled 使用 GORM Save 全量保存任务配置，避免手动拼装更新字段。
func (d *AITaskConfigData) UpdateEnabled(ctx context.Context, taskCode string, isEnabled bool) (*AITaskConfig, error) {
	config, err := d.FindByTaskCode(ctx, taskCode)
	if err != nil {
		return nil, err
	}

	config.IsEnabled = isEnabled
	if err := d.db.WithContext(ctx).Save(config).Error; err != nil {
		return nil, err
	}

	return d.FindByTaskCode(ctx, taskCode)
}
