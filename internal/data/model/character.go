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

// CharacterRelationship 角色关系表，用于描述同一部小说中不同角色之间的关系，可用于前端人物关系图展示
type CharacterRelationship struct {
	ID int64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID" json:"id"`

	NovelID int64 `gorm:"column:novel_id;not null;comment:小说ID，表示该人物关系属于哪一部小说" json:"novel_id"`

	SourceCharacterID int64 `gorm:"column:source_character_id;not null;comment:关系起点角色ID，例如“张三喜欢李四”中的张三" json:"source_character_id"`

	TargetCharacterID int64 `gorm:"column:target_character_id;not null;comment:关系目标角色ID，例如“张三喜欢李四”中的李四" json:"target_character_id"`

	RelationType string `gorm:"column:relation_type;type:varchar(100);not null;comment:关系类型，例如：朋友、敌人、恋人、亲人、师徒、上下级、合作、竞争、暗恋等" json:"relation_type"`

	Description string `gorm:"column:description;type:text;comment:关系描述，用于详细说明两个角色之间的关系背景、变化过程或特殊说明" json:"description"`

	Direction string `gorm:"column:direction;type:varchar(50);not null;default:directed;comment:关系方向。directed表示有方向关系，例如A暗恋B；undirected表示无方向关系，例如A和B是朋友" json:"direction"`

	Strength int `gorm:"column:strength;not null;default:1;comment:关系强度，数值越大表示关系越强烈。例如1表示弱关系，5表示强关系，10表示核心关系" json:"strength"`

	Status string `gorm:"column:status;type:varchar(50);not null;default:active;comment:关系状态。active表示当前有效，inactive表示已失效，hidden表示隐藏，deleted表示逻辑删除" json:"status"`

	SourceChapterID *int64 `gorm:"column:source_chapter_id;comment:关系来源章节ID，表示这个关系最早或主要在哪个章节中出现" json:"source_chapter_id"`

	CreatedBy string `gorm:"column:created_by;type:varchar(50);not null;default:manual;comment:创建来源。manual表示用户手动创建，ai表示AI生成，imported表示外部导入" json:"created_by"`

	Metadata datatypes.JSON `gorm:"column:metadata;type:jsonb;not null;default:'{}';comment:扩展字段，用于存储前端关系图坐标、样式、AI分析依据、额外标签等非固定结构数据" json:"metadata"`

	CreatedAt time.Time `gorm:"column:created_at;not null;default:now();comment:创建时间" json:"created_at"`

	UpdatedAt time.Time `gorm:"column:updated_at;not null;default:now();comment:更新时间" json:"updated_at"`
}

func (CharacterRelationship) TableName() string {
	return "character_relationships"
}
