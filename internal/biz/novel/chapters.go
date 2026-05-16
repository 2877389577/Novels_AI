package novel

import "gorm.io/gorm"

type Chapter struct {
	gorm.Model
	NovelID   int64  `gorm:"column:novel_id;not null;uniqueIndex:uk_novel_chapter_no" json:"novelId"`     // 小说ID
	Title     string `gorm:"column:title;type:varchar(255);not null" json:"title"`                        // 章节标题
	Content   string `gorm:"column:content;type:text;not null" json:"content"`                            // 章节内容
	ChapterNo int    `gorm:"column:chapter_no;not null;uniqueIndex:uk_novel_chapter_no" json:"chapterNo"` // 章节序号
	WordCount int    `gorm:"column:word_count;not null;default:0" json:"wordCount"`                       // 章节字数
}

func (Chapter) TableName() string {
	return "chapters"
}
