package model

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Novel struct {
	gorm.Model
	Title      string         `gorm:"column:title;type:varchar(255);not null" json:"title"`             // 小说书名
	Intro      string         `gorm:"column:intro;type:text;not null" json:"intro"`                     // 小说简介
	AuthorName string         `gorm:"column:author_name;type:varchar(255)" json:"authorName"`           // 小说作者
	CoverURL   string         `gorm:"column:cover_url;type:text" json:"coverUrl"`                       // 小说封面URL
	WordCount  int64          `gorm:"column:word_count;not null;default:0" json:"wordCount"`            // 小说总字数
	Metadata   datatypes.JSON `gorm:"column:metadata;type:jsonb;not null;default:'{}'" json:"metadata"` // 小说元数据，标签等
}

func (Novel) TableName() string {
	return "novels"
}
