package model

import (
	"github.com/lib/pq"
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
