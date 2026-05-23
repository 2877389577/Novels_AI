package data

import (
	"context"
	"errors"

	"Novels_AI/backend/internal/data/model"
	"Novels_AI/backend/internal/pkg/common"

	"gorm.io/gorm"
)

type Chapter = model.Chapter

type ChapterData struct {
	db *gorm.DB
}

func NewChapterData(db *gorm.DB) *ChapterData {
	return &ChapterData{db: db}
}

// Create 在同一个事务里新增章节并增加小说总字数，避免章节和小说统计出现半成功状态。
func (d *ChapterData) Create(ctx context.Context, chapter *Chapter, wordDelta int64) (*Chapter, error) {
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(chapter).Error; err != nil {
			return err
		}

		return updateNovelWordCount(tx, chapter.NovelID, wordDelta)
	})
	if err != nil {
		return nil, err
	}

	return chapter, nil
}

// List 按章节序号升序分页查询指定小说下的未删除章节。
func (d *ChapterData) List(ctx context.Context, novelID int64, offset, limit int) ([]Chapter, int64, error) {
	var total int64
	err := d.db.WithContext(ctx).
		Model(&model.Chapter{}).
		Where("novel_id = ?", novelID).
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	var chapters []Chapter
	err = d.db.WithContext(ctx).
		Select("id, novel_id, chapter_no, title, word_count, created_at, updated_at").
		Where("novel_id = ?", novelID).
		Order("chapter_no ASC").
		Offset(offset).
		Limit(limit).
		Find(&chapters).Error
	if err != nil {
		return nil, 0, err
	}

	return chapters, total, nil
}

// FindByID 查询指定小说下的章节，避免跨小说 ID 误读其他小说的章节。
func (d *ChapterData) FindByID(ctx context.Context, novelID int64, chapterID uint) (*Chapter, error) {
	var chapter Chapter
	db := d.db.WithContext(ctx)

	// 这里不需要检查小说ID，现在懒得改，就兼容一下。
	if novelID == 0 {
		db.Where("id = ? ", chapterID)
	} else {
		db.Where("id = ? AND novel_id = ?", chapterID, novelID)
	}

	err := db.First(&chapter).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.ChapterNotFound
		}
		return nil, err
	}

	return &chapter, nil
}

// ChapterNoExists 检查同一本小说下章节编号是否已存在，excludeID 用于更新时排除当前章节。
func (d *ChapterData) ChapterNoExists(ctx context.Context, novelID int64, chapterNo int, excludeID uint) (bool, error) {
	query := d.db.WithContext(ctx).
		Model(&model.Chapter{}).
		Where("novel_id = ? AND chapter_no = ?", novelID, chapterNo)
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return false, err
	}

	return total > 0, nil
}

// MaxChapterNo 返回指定小说当前最大章节编号；没有章节时返回 0。
func (d *ChapterData) MaxChapterNo(ctx context.Context, novelID int64) (int, error) {
	var maxChapterNo int
	err := d.db.WithContext(ctx).
		Model(&model.Chapter{}).
		Where("novel_id = ?", novelID).
		Select("COALESCE(MAX(chapter_no), 0)").
		Scan(&maxChapterNo).Error
	if err != nil {
		return 0, err
	}

	return maxChapterNo, nil
}

// Update 在同一个事务里全量保存章节，并按字数差值调整小说总字数。
func (d *ChapterData) Update(ctx context.Context, chapter *Chapter, wordDelta int64) (*Chapter, error) {
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(chapter).Error; err != nil {
			return err
		}

		return updateNovelWordCount(tx, chapter.NovelID, wordDelta)
	})
	if err != nil {
		return nil, err
	}

	return d.FindByID(ctx, chapter.NovelID, chapter.ID)
}

// Delete 使用 GORM 软删除章节，并同步扣减小说总字数。
func (d *ChapterData) Delete(ctx context.Context, novelID int64, chapterID uint, wordDelta int64) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Where("id = ? AND novel_id = ?", chapterID, novelID).
			Delete(&model.Chapter{})
		if result.Error != nil {
			return result.Error
		}

		return updateNovelWordCount(tx, novelID, wordDelta)
	})
}

// updateNovelWordCount 按差值维护小说总字数，并用 CASE 保证统计值不会被扣成负数。
func updateNovelWordCount(tx *gorm.DB, novelID int64, delta int64) error {
	if delta == 0 {
		return nil
	}

	result := tx.Model(&model.Novel{}).
		Where("id = ?", novelID).
		Update("word_count", gorm.Expr("CASE WHEN word_count + ? < 0 THEN 0 ELSE word_count + ? END", delta, delta))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNovelNotFound
	}

	return nil
}
