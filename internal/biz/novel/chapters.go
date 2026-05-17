package novel

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"Novels_AI/backend/internal/data"
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
	Update(ctx context.Context, novelID int64, chapterID uint, values map[string]any, wordDelta int64) (*data.Chapter, error)
	Delete(ctx context.Context, novelID int64, chapterID uint, wordDelta int64) error
}

type CreateChapterParams struct {
	NovelID   int64
	ChapterNo int
	Title     string
	Content   string
	WordCount int
}

type UpdateChapterParams struct {
	NovelID   int64
	ChapterID uint
	ChapterNo *int
	Title     *string
	Content   *string
	WordCount *int
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

// CreateChapter 校验章节信息后新增章节，并把章节字数同步累加到小说总字数。
func (uc *ChapterUseCase) CreateChapter(ctx context.Context, params CreateChapterParams) (*data.Chapter, error) {
	title, content, err := normalizeChapterText(params.Title, params.Content)
	if err != nil {
		return nil, err
	}
	if err := validateChapterNo(params.ChapterNo); err != nil {
		return nil, err
	}
	if err := validateChapterWordCount(params.WordCount); err != nil {
		return nil, err
	}
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
		Title:     title,
		Content:   content,
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

	page, pageSize = normalizePagination(page, pageSize)
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

// UpdateChapter 局部更新章节；如果章节字数变化，则按差值维护小说总字数。
func (uc *ChapterUseCase) UpdateChapter(ctx context.Context, params UpdateChapterParams) (*data.Chapter, error) {
	oldChapter, err := uc.chapterData.FindByID(ctx, params.NovelID, params.ChapterID)
	if err != nil {
		return nil, err
	}

	values := make(map[string]any)
	if params.ChapterNo != nil {
		if err := validateChapterNo(*params.ChapterNo); err != nil {
			return nil, err
		}
		if *params.ChapterNo != oldChapter.ChapterNo {
			exists, err := uc.chapterData.ChapterNoExists(ctx, params.NovelID, *params.ChapterNo, params.ChapterID)
			if err != nil {
				slog.ErrorContext(ctx, "检查章节编号失败", "err", err)
				return nil, err
			}
			if exists {
				return nil, common.ChapterNoExists
			}
			values["chapter_no"] = *params.ChapterNo
		}
	}
	if params.Title != nil {
		title := strings.TrimSpace(*params.Title)
		if title == "" {
			return nil, common.ChapterTitleRequired
		}
		values["title"] = title
	}
	if params.Content != nil {
		content := strings.TrimSpace(*params.Content)
		if content == "" {
			return nil, common.ChapterContentRequired
		}
		values["content"] = content
	}

	var wordDelta int64
	if params.WordCount != nil {
		if err := validateChapterWordCount(*params.WordCount); err != nil {
			return nil, err
		}
		if *params.WordCount != oldChapter.WordCount {
			values["word_count"] = *params.WordCount
			wordDelta = int64(*params.WordCount - oldChapter.WordCount)
		}
	}

	return uc.chapterData.Update(ctx, params.NovelID, params.ChapterID, values, wordDelta)
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

func normalizeChapterText(title, content string) (string, string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", "", common.ChapterTitleRequired
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return "", "", common.ChapterContentRequired
	}

	return title, content, nil
}

func validateChapterNo(chapterNo int) error {
	if chapterNo <= 0 {
		return common.ChapterNoInvalid
	}

	return nil
}

func validateChapterWordCount(wordCount int) error {
	if wordCount < 0 {
		return common.ChapterWordCountInvalid
	}

	return nil
}
