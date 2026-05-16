package novel

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"Novels_AI/backend/internal/data"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	defaultPage     = 1
	defaultPageSize = 10
	maxPageSize     = 100
)

var (
	ErrTitleRequired = errors.New("书名不能为空")
	ErrNovelNotFound = data.ErrNovelNotFound
)

type Novel = data.Novel

type NovelUseCase struct {
	novelData NovelRepo
}

type NovelRepo interface {
	Create(ctx context.Context, novel *data.Novel) (*data.Novel, error)
	List(ctx context.Context, offset, limit int) ([]data.Novel, int64, error)
	FindByID(ctx context.Context, id uint) (*data.Novel, error)
	Update(ctx context.Context, id uint, values map[string]any) (*data.Novel, error)
	Delete(ctx context.Context, id uint) error
}

type CreateNovelParams struct {
	Title      string
	Intro      string
	AuthorName string
	CoverURL   string
	WordCount  int64
	Metadata   datatypes.JSON
}

type UpdateNovelParams struct {
	Title      *string
	Intro      *string
	AuthorName *string
	CoverURL   *string
	WordCount  *int64
	Metadata   *datatypes.JSON
}

type ListNovelResult struct {
	Items    []data.Novel
	Total    int64
	Page     int
	PageSize int
}

func NewNovelUseCase(novelData NovelRepo) *NovelUseCase {
	return &NovelUseCase{novelData: novelData}
}

// Create 整理小说默认值后创建记录，目前仅书名是必填项。
func (uc *NovelUseCase) Create(ctx context.Context, params CreateNovelParams) (*data.Novel, error) {
	title := strings.TrimSpace(params.Title)
	if title == "" {
		return nil, ErrTitleRequired
	}

	novel := &data.Novel{
		Title:      title,
		Intro:      params.Intro,
		AuthorName: params.AuthorName,
		CoverURL:   params.CoverURL,
		WordCount:  params.WordCount,
		Metadata:   normalizeMetadata(params.Metadata),
	}
	create, err := uc.novelData.Create(ctx, novel)
	if err != nil {
		slog.ErrorContext(ctx, "新增小说失败", "err", err)
		return nil, err
	}
	return create, nil
}

// List 按分页参数查询小说列表，并把缺省和上限逻辑集中在业务层。
func (uc *NovelUseCase) List(ctx context.Context, page, pageSize int) (*ListNovelResult, error) {
	page, pageSize = normalizePagination(page, pageSize)
	offset := (page - 1) * pageSize

	items, total, err := uc.novelData.List(ctx, offset, pageSize)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return new(ListNovelResult), nil
		}
		slog.ErrorContext(ctx, "查询小说列表失败", "err", err)
		return nil, err
	}

	return &ListNovelResult{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (uc *NovelUseCase) Get(ctx context.Context, id uint) (*data.Novel, error) {
	byID, err := uc.novelData.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		slog.ErrorContext(ctx, "查询小说失败", "err", err)
		return nil, err
	}
	return byID, nil
}

// Update 只更新请求中明确传入的字段，未传字段保持原值。
func (uc *NovelUseCase) Update(ctx context.Context, id uint, params UpdateNovelParams) (*data.Novel, error) {
	values := make(map[string]any)
	if params.Title != nil {
		title := strings.TrimSpace(*params.Title)
		if title == "" {
			return nil, ErrTitleRequired
		}
		values["title"] = title
	}
	if params.Intro != nil {
		values["intro"] = *params.Intro
	}
	if params.AuthorName != nil {
		values["author_name"] = *params.AuthorName
	}
	if params.CoverURL != nil {
		values["cover_url"] = *params.CoverURL
	}
	if params.WordCount != nil {
		values["word_count"] = *params.WordCount
	}
	if params.Metadata != nil {
		values["metadata"] = normalizeMetadata(*params.Metadata)
	}

	return uc.novelData.Update(ctx, id, values)
}

func (uc *NovelUseCase) Delete(ctx context.Context, id uint) error {
	return uc.novelData.Delete(ctx, id)
}

func normalizePagination(page, pageSize int) (int, int) {
	if page <= 0 {
		page = defaultPage
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	return page, pageSize
}

func normalizeMetadata(metadata datatypes.JSON) datatypes.JSON {
	if len(metadata) == 0 || string(metadata) == "null" {
		return datatypes.JSON([]byte("{}"))
	}

	return metadata
}
