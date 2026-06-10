package model

import (
	"gorm.io/gorm"
)

type Chapter struct {
	gorm.Model
	NovelID   int64  `gorm:"column:novel_id;not null" json:"novelId"`               // 小说ID；未删除章节的编号唯一性由启动迁移创建的部分唯一索引保证。
	Title     string `gorm:"column:title;type:varchar(255);not null" json:"title"`  // 章节标题
	Content   string `gorm:"column:content;type:text;not null" json:"content"`      // 章节内容
	ChapterNo int    `gorm:"column:chapter_no;not null" json:"chapterNo"`           // 章节序号；软删除后允许同一本小说重新使用原章节编号。
	WordCount int    `gorm:"column:word_count;not null;default:0" json:"wordCount"` // 章节字数
}

func (Chapter) TableName() string {
	return "chapters"
}
