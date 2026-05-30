package aitaskconfig

import (
	"context"

	"Novels_AI/backend/internal/data"
	"Novels_AI/backend/internal/data/dto"
)

const (
	// TaskCodeChapterPlotAnalysis 是自动章节剧情总结任务的稳定编码。
	TaskCodeChapterPlotAnalysis = data.AITaskCodeChapterPlotAnalysis
)

type AITaskConfig = data.AITaskConfig

// AITaskConfigRepo 描述主动执行 AI 任务配置依赖的数据访问能力。
type AITaskConfigRepo interface {
	List(ctx context.Context) ([]*data.AITaskConfig, error)
	FindByTaskCode(ctx context.Context, taskCode string) (*data.AITaskConfig, error)
	UpdateEnabled(ctx context.Context, taskCode string, isEnabled bool) (*data.AITaskConfig, error)
}

// AITaskConfigUseCase 负责后台主动执行 AI 任务配置的查询和开关更新。
type AITaskConfigUseCase struct {
	repo AITaskConfigRepo
}

func NewAITaskConfigUseCase(repo AITaskConfigRepo) *AITaskConfigUseCase {
	return &AITaskConfigUseCase{repo: repo}
}

func (uc *AITaskConfigUseCase) List(ctx context.Context) ([]*data.AITaskConfig, error) {
	return uc.repo.List(ctx)
}

func (uc *AITaskConfigUseCase) Update(ctx context.Context, params dto.UpdateAITaskConfigRequest) (*data.AITaskConfig, error) {
	return uc.repo.UpdateEnabled(ctx, params.TaskCode, *params.IsEnabled)
}

// IsEnabled 返回指定主动 AI 任务是否启用；找不到配置时由仓储返回统一业务错误。
func (uc *AITaskConfigUseCase) IsEnabled(ctx context.Context, taskCode string) (bool, error) {
	config, err := uc.repo.FindByTaskCode(ctx, taskCode)
	if err != nil {
		return false, err
	}

	return config.IsEnabled, nil
}
