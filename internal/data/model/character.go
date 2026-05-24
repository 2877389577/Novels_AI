package model

import (
	"time"

	"github.com/lib/pq"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Character struct {
	gorm.Model
	NovelID                  int64          `gorm:"column:novel_id;not null;index" json:"novelId"`                       // 小说ID
	Name                     string         `gorm:"column:name;type:varchar(255);not null" json:"name"`                  // 角色名称
	Gender                   string         `gorm:"column:gender;type:varchar(50)" json:"gender"`                        // 角色性别
	Intro                    string         `gorm:"column:intro;type:text" json:"intro"`                                 // 角色介绍
	Personality              string         `gorm:"column:personality;type:text" json:"personality"`                     // 角色性格
	Appearance               string         `gorm:"column:appearance;type:text" json:"appearance"`                       // 角色外貌
	Background               string         `gorm:"column:background;type:text" json:"background"`                       // 角色背景
	Ability                  string         `gorm:"column:ability;type:text" json:"ability"`                             // 角色能力
	Motivation               string         `gorm:"column:motivation;type:text" json:"motivation"`                       // 角色动机
	PlotDirection            string         `gorm:"column:plot_direction;type:text" json:"plotDirection"`                // 角色剧情方向
	FirstAppearanceChapterID *int64         `gorm:"column:first_appearance_chapter_id" json:"firstAppearanceChapterId"`  // 首次出现章节ID
	AppearanceImgURL         string         `gorm:"column:appearance_img_url;type:varchar(255)" json:"appearanceImgUrl"` // 角色形象图URL
	Status                   int16          `gorm:"column:status;type:smallint;not null;default:1" json:"status"`        // 角色状态：1在线 2下线
	CharactersTags           pq.StringArray `gorm:"column:characters_tags;type:varchar(50)[]" json:"charactersTags"`     // 角色标签
}

func (Character) TableName() string {
	return "characters"
}

// CharacterRelationNode 保存角色在关系图画布上的节点布局，不承载角色本身的业务资料。
type CharacterRelationNode struct {
	ID          int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`                             // 节点布局ID
	NovelID     int64          `gorm:"column:novel_id;not null;index" json:"novelId"`                            // 小说ID
	CharacterID int64          `gorm:"column:character_id;not null;index" json:"characterId"`                    // 角色ID
	NodeType    string         `gorm:"column:node_type;type:varchar(50);not null;default:character" json:"type"` // Vue Flow 节点类型
	PositionX   float64        `gorm:"column:position_x;not null;default:0" json:"positionX"`                    // 节点X坐标
	PositionY   float64        `gorm:"column:position_y;not null;default:0" json:"positionY"`                    // 节点Y坐标
	Width       *float64       `gorm:"column:width" json:"width"`                                                // 节点宽度
	Height      *float64       `gorm:"column:height" json:"height"`                                              // 节点高度
	Hidden      bool           `gorm:"column:hidden;not null;default:false" json:"hidden"`                       // 是否隐藏
	Locked      bool           `gorm:"column:locked;not null;default:false" json:"locked"`                       // 是否锁定
	Style       datatypes.JSON `gorm:"column:style;type:jsonb;not null;default:'{}'" json:"style"`               // 前端样式配置
	ExtraData   datatypes.JSON `gorm:"column:extra_data;type:jsonb;not null;default:'{}'" json:"extraData"`      // 节点扩展数据
	Status      int16          `gorm:"column:status;type:smallint;not null;default:1" json:"status"`             // 状态：1启用 2停用
	CreatedAt   time.Time      `gorm:"column:created_at;not null;default:now()" json:"createdAt"`                // 创建时间
	UpdatedAt   time.Time      `gorm:"column:updated_at;not null;default:now()" json:"updatedAt"`                // 更新时间
}

func (CharacterRelationNode) TableName() string {
	return "character_relation_nodes"
}

// CharacterRelation 保存两个角色之间的业务关系，对应 Vue Flow 中的一条边。
type CharacterRelation struct {
	ID                int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`                               // 关系ID
	NovelID           int64          `gorm:"column:novel_id;not null;index" json:"novelId"`                              // 小说ID
	SourceCharacterID int64          `gorm:"column:source_character_id;not null;index" json:"sourceCharacterId"`         // 起始角色ID
	TargetCharacterID int64          `gorm:"column:target_character_id;not null;index" json:"targetCharacterId"`         // 目标角色ID
	RelationType      string         `gorm:"column:relation_type;type:varchar(50);not null" json:"relationType"`         // 关系类型编码
	RelationLabel     string         `gorm:"column:relation_label;type:varchar(100);not null" json:"relationLabel"`      // 关系展示名称
	Description       string         `gorm:"column:description;type:text" json:"description"`                            // 关系说明
	Direction         int16          `gorm:"column:direction;type:smallint;not null;default:2" json:"direction"`         // 关系方向：1单向 2双向或无向
	EdgeType          string         `gorm:"column:edge_type;type:varchar(50);not null;default:default" json:"edgeType"` // Vue Flow 边类型
	Animated          bool           `gorm:"column:animated;not null;default:false" json:"animated"`                     // 是否动画展示
	SourceHandle      string         `gorm:"column:source_handle;type:varchar(50)" json:"sourceHandle"`                  // 起始连接桩ID
	TargetHandle      string         `gorm:"column:target_handle;type:varchar(50)" json:"targetHandle"`                  // 目标连接桩ID
	SortOrder         int            `gorm:"column:sort_order;not null;default:0" json:"sortOrder"`                      // 排序值
	Style             datatypes.JSON `gorm:"column:style;type:jsonb;not null;default:'{}'" json:"style"`                 // 关系线样式配置
	ExtraData         datatypes.JSON `gorm:"column:extra_data;type:jsonb;not null;default:'{}'" json:"extraData"`        // 关系扩展数据
	Status            int16          `gorm:"column:status;type:smallint;not null;default:1" json:"status"`               // 状态：1启用 2停用
	CreatedAt         time.Time      `gorm:"column:created_at;not null;default:now()" json:"createdAt"`                  // 创建时间
	UpdatedAt         time.Time      `gorm:"column:updated_at;not null;default:now()" json:"updatedAt"`                  // 更新时间
}

func (CharacterRelation) TableName() string {
	return "character_relations"
}
