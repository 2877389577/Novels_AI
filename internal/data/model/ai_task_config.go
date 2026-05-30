package model

import "time"

const (
	// AITaskCodeChapterPlotAnalysis 是自动章节剧情总结任务的稳定编码。
	AITaskCodeChapterPlotAnalysis = "chapter_plot_analysis"
	// AITaskNameChapterPlotAnalysis 是自动章节剧情总结任务在后台配置页展示的名称。
	AITaskNameChapterPlotAnalysis = "自动章节剧情总结"
	// AITaskDescriptionChapterPlotAnalysis 说明自动章节剧情总结的触发场景。
	AITaskDescriptionChapterPlotAnalysis = "章节保存或修改后异步调用 AI 生成章节剧情总结"
)

// AITaskConfig 保存主动执行 AI 任务的启用配置。
type AITaskConfig struct {
	ID int64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID" json:"id"`

	TaskCode string `gorm:"column:task_code;type:varchar(100);not null;uniqueIndex:uk_ai_task_configs_task_code;comment:任务编码，代码中使用的稳定标识" json:"taskCode"`

	TaskName string `gorm:"column:task_name;type:varchar(100);not null;comment:任务名称" json:"taskName"`

	Description string `gorm:"column:description;type:text;not null;default:'';comment:任务说明" json:"description"`

	IsEnabled bool `gorm:"column:is_enabled;not null;default:false;comment:是否启用" json:"isEnabled"`

	CreatedAt time.Time `gorm:"column:created_at;not null;default:now();comment:创建时间" json:"createdAt"`

	UpdatedAt time.Time `gorm:"column:updated_at;not null;default:now();comment:更新时间" json:"updatedAt"`
}

func (AITaskConfig) TableName() string {
	return "ai_task_configs"
}
