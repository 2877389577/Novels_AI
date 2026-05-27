package model

import (
	"time"

	"gorm.io/datatypes"
)

// ChapterPlotAnalysis 保存单章剧情总结的结构化结果，和章节正文分表存储，避免章节详情接口被 AI 结果耦合。
type ChapterPlotAnalysis struct {
	ID int64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID" json:"id"`

	NovelID int64 `gorm:"column:novel_id;not null;index;comment:小说ID" json:"novel_id"`

	ChapterID uint `gorm:"column:chapter_id;not null;uniqueIndex:uk_chapter_plot_analyses_chapter_id;comment:章节ID" json:"chapter_id"`

	Summary             string         `gorm:"column:summary;type:text;not null;default:'';comment:本章剧情概述" json:"summary"`
	KeyEvents           datatypes.JSON `gorm:"column:key_events;type:jsonb;not null;default:'[]';comment:关键事件列表" json:"key_events"`
	CharactersInvolved  datatypes.JSON `gorm:"column:characters_involved;type:jsonb;not null;default:'[]';comment:涉及角色列表" json:"characters_involved"`
	RelationshipChanges datatypes.JSON `gorm:"column:relationship_changes;type:jsonb;not null;default:'[]';comment:人物关系变化" json:"relationship_changes"`
	EventAnalysis       datatypes.JSON `gorm:"column:event_analysis;type:jsonb;not null;default:'[]';comment:事件分析" json:"event_analysis"`
	Foreshadowing       datatypes.JSON `gorm:"column:foreshadowing;type:jsonb;not null;default:'[]';comment:伏笔列表" json:"foreshadowing"`
	UnresolvedThreads   datatypes.JSON `gorm:"column:unresolved_threads;type:jsonb;not null;default:'[]';comment:未解决线索" json:"unresolved_threads"`

	CreatedAt time.Time `gorm:"column:created_at;not null;default:now();comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;default:now();comment:更新时间" json:"updated_at"`
}

func (ChapterPlotAnalysis) TableName() string {
	return "chapter_plot_analyses"
}
