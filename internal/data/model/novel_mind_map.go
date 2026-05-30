package model

import (
	"time"

	"gorm.io/datatypes"
)

// NovelMindMap 保存单本小说的 SimpleMindMap 原始数据。
type NovelMindMap struct {
	ID int64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID" json:"id"`

	NovelID int64 `gorm:"column:novel_id;not null;uniqueIndex:uk_novel_mind_maps_novel_id;comment:小说ID" json:"novel_id"`

	MindMapData datatypes.JSON `gorm:"column:mind_map_data;type:jsonb;not null;default:'{}';comment:SimpleMindMap完整JSON数据" json:"mind_map_data"`

	CreatedAt time.Time `gorm:"column:created_at;not null;default:now();comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;default:now();comment:更新时间" json:"updated_at"`
}

func (NovelMindMap) TableName() string {
	return "novel_mind_maps"
}
