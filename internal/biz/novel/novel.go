package novel

import (
	"context"
	"errors"
	"log/slog"

	"Novels_AI/backend/internal/data"
	"Novels_AI/backend/internal/data/dto"

	"gorm.io/datatypes"
	"gorm.io/gorm"
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
	Update(ctx context.Context, novel *data.Novel) (*data.Novel, error)
	Delete(ctx context.Context, id uint) error
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

// Create 整理小说默认值后创建记录。
func (uc *NovelUseCase) Create(ctx context.Context, params dto.CreateNovelRequest) (*data.Novel, error) {
	novel := &data.Novel{
		Title:        params.Title,
		Intro:        params.Intro,
		NovelOutline: params.NovelOutline,
		AuthorName:   params.AuthorName,
		CoverURL:     params.CoverURL,
		Metadata:     normalizeMetadata(params.Metadata.Value),
	}
	create, err := uc.novelData.Create(ctx, novel)
	if err != nil {
		slog.ErrorContext(ctx, "新增小说失败", "err", err)
		return nil, err
	}
	return create, nil
}

// List 按分页参数查询小说列表。
func (uc *NovelUseCase) List(ctx context.Context, page, pageSize int) (*ListNovelResult, error) {
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

// Update 按请求参数全量保存小说。
func (uc *NovelUseCase) Update(ctx context.Context, params dto.UpdateNovelRequest) (*data.Novel, error) {
	return uc.novelData.Update(ctx, &data.Novel{
		Model:        gorm.Model{ID: params.ID},
		Title:        params.Title,
		Intro:        params.Intro,
		NovelOutline: params.NovelOutline,
		AuthorName:   params.AuthorName,
		CoverURL:     params.CoverURL,
		WordCount:    params.WordCount,
		Metadata:     normalizeMetadata(params.Metadata.Value),
	})
}

func (uc *NovelUseCase) Delete(ctx context.Context, id uint) error {
	return uc.novelData.Delete(ctx, id)
}

func normalizeMetadata(metadata datatypes.JSON) datatypes.JSON {
	if len(metadata) == 0 || string(metadata) == "null" {
		return datatypes.JSON([]byte("{}"))
	}

	return metadata
}
