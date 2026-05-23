package novel

import (
	"context"
	"errors"
	"log/slog"

	"Novels_AI/backend/internal/data"
	"Novels_AI/backend/internal/data/dto"
	"Novels_AI/backend/internal/pkg/common"

	"gorm.io/gorm"
)

type Chapter = data.Chapter

type ChapterUseCase struct {
	novelData   NovelRepo
	chapterData ChapterRepo
}

type ChapterRepo interface {
	Create(ctx context.Context, chapter *data.Chapter, wordDelta int64) (*data.Chapter, error)
	List(ctx context.Context, novelID int64, offset, limit int) ([]data.Chapter, int64, error)
	FindByID(ctx context.Context, novelID int64, chapterID uint) (*data.Chapter, error)
	ChapterNoExists(ctx context.Context, novelID int64, chapterNo int, excludeID uint) (bool, error)
	MaxChapterNo(ctx context.Context, novelID int64) (int, error)
	Update(ctx context.Context, chapter *data.Chapter, wordDelta int64) (*data.Chapter, error)
	Delete(ctx context.Context, novelID int64, chapterID uint, wordDelta int64) error
}

type ListChapterResult struct {
	Items    []data.Chapter
	Total    int64
	Page     int
	PageSize int
}

func NewChapterUseCase(novelData NovelRepo, chapterData ChapterRepo) *ChapterUseCase {
	return &ChapterUseCase{
		novelData:   novelData,
		chapterData: chapterData,
	}
}

// CreateChapter 新增章节，并把章节字数同步累加到小说总字数。
func (uc *ChapterUseCase) CreateChapter(ctx context.Context, params dto.CreateChapterRequest) (*data.Chapter, error) {
	if err := uc.ensureNovelExists(ctx, params.NovelID); err != nil {
		return nil, err
	}

	exists, err := uc.chapterData.ChapterNoExists(ctx, params.NovelID, params.ChapterNo, 0)
	if err != nil {
		slog.ErrorContext(ctx, "检查章节编号失败", "err", err)
		return nil, err
	}
	if exists {
		return nil, common.ChapterNoExists
	}

	chapter := &data.Chapter{
		NovelID:   params.NovelID,
		ChapterNo: params.ChapterNo,
		Title:     params.Title,
		Content:   params.Content,
		WordCount: params.WordCount,
	}
	return uc.chapterData.Create(ctx, chapter, int64(params.WordCount))
}

// NextChapterNo 返回单本小说的下一章编号，即当前最大章节编号加一。
func (uc *ChapterUseCase) NextChapterNo(ctx context.Context, novelID int64) (int, error) {
	if err := uc.ensureNovelExists(ctx, novelID); err != nil {
		return 0, err
	}

	maxChapterNo, err := uc.chapterData.MaxChapterNo(ctx, novelID)
	if err != nil {
		slog.ErrorContext(ctx, "查询最大章节编号失败", "err", err)
		return 0, err
	}

	return maxChapterNo + 1, nil
}

// ListChapters 按分页参数查询单本小说下的章节列表。
func (uc *ChapterUseCase) ListChapters(ctx context.Context, novelID int64, page, pageSize int) (*ListChapterResult, error) {
	if err := uc.ensureNovelExists(ctx, novelID); err != nil {
		return nil, err
	}

	offset := (page - 1) * pageSize

	items, total, err := uc.chapterData.List(ctx, novelID, offset, pageSize)
	if err != nil {
		slog.ErrorContext(ctx, "查询章节列表失败", "err", err)
		return nil, err
	}

	return &ListChapterResult{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (uc *ChapterUseCase) GetChapter(ctx context.Context, novelID int64, chapterID uint) (*data.Chapter, error) {
	return uc.chapterData.FindByID(ctx, novelID, chapterID)
}

// UpdateChapter 全量更新章节；如果章节字数变化，则按差值维护小说总字数。
func (uc *ChapterUseCase) UpdateChapter(ctx context.Context, params dto.UpdateChapterRequest) (*data.Chapter, error) {
	oldChapter, err := uc.chapterData.FindByID(ctx, params.NovelID, params.ChapterID)
	if err != nil {
		return nil, err
	}

	if params.ChapterNo != oldChapter.ChapterNo {
		exists, err := uc.chapterData.ChapterNoExists(ctx, params.NovelID, params.ChapterNo, params.ChapterID)
		if err != nil {
			slog.ErrorContext(ctx, "检查章节编号失败", "err", err)
			return nil, err
		}
		if exists {
			return nil, common.ChapterNoExists
		}
	}

	chapter := *oldChapter
	chapter.ChapterNo = params.ChapterNo
	chapter.Title = params.Title
	chapter.Content = params.Content
	chapter.WordCount = params.WordCount
	wordDelta := int64(params.WordCount - oldChapter.WordCount)

	return uc.chapterData.Update(ctx, &chapter, wordDelta)
}

// DeleteChapter 软删除章节，并按章节原字数扣减小说总字数。
func (uc *ChapterUseCase) DeleteChapter(ctx context.Context, novelID int64, chapterID uint) error {
	chapter, err := uc.chapterData.FindByID(ctx, novelID, chapterID)
	if err != nil {
		return err
	}

	return uc.chapterData.Delete(ctx, novelID, chapterID, -int64(chapter.WordCount))
}

func (uc *ChapterUseCase) ensureNovelExists(ctx context.Context, novelID int64) error {
	_, err := uc.novelData.FindByID(ctx, uint(novelID))
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, ErrNovelNotFound) {
		return ErrNovelNotFound
	}

	slog.ErrorContext(ctx, "查询小说失败", "err", err)
	return err
}
