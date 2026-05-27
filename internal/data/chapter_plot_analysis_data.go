package data

import (
	"context"
	"errors"

	"Novels_AI/backend/internal/data/model"
	"Novels_AI/backend/internal/pkg/common"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ChapterPlotAnalysis = model.ChapterPlotAnalysis

// ChapterPlotAnalysisData 负责章节剧情总结表的持久化读写。
type ChapterPlotAnalysisData struct {
	db *gorm.DB
}

func NewChapterPlotAnalysisData(db *gorm.DB) *ChapterPlotAnalysisData {
	return &ChapterPlotAnalysisData{db: db}
}

// UpsertByChapterID 按 chapter_id 保存最新剧情总结，章节被修改后会覆盖旧总结。
func (d *ChapterPlotAnalysisData) UpsertByChapterID(ctx context.Context, analysis *ChapterPlotAnalysis) (*ChapterPlotAnalysis, error) {
	if err := d.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "chapter_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"novel_id",
				"summary",
				"key_events",
				"characters_involved",
				"relationship_changes",
				"event_analysis",
				"foreshadowing",
				"unresolved_threads",
				"updated_at",
			}),
		}).
		Create(analysis).Error; err != nil {
		return nil, err
	}

	return d.FindByChapterID(ctx, analysis.NovelID, analysis.ChapterID)
}

// FindByChapterID 查询指定小说和章节的剧情总结，避免跨小说误读其他章节的总结。
func (d *ChapterPlotAnalysisData) FindByChapterID(ctx context.Context, novelID int64, chapterID uint) (*ChapterPlotAnalysis, error) {
	var analysis ChapterPlotAnalysis
	err := d.db.WithContext(ctx).
		Where("novel_id = ? AND chapter_id = ?", novelID, chapterID).
		First(&analysis).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.ChapterPlotAnalysisNotFound
		}
		return nil, err
	}

	return &analysis, nil
}
