package novel

import (
	"context"
	"errors"
	"log/slog"

	"Novels_AI/backend/internal/data"
	"Novels_AI/backend/internal/data/dto"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Character = data.Character

// CharacterUseCase 承载角色资料的业务规则，并复用小说仓储校验归属小说是否存在。
type CharacterUseCase struct {
	novelData     NovelRepo
	characterData CharacterRepo
}

// CharacterRepo 定义角色业务层需要的数据访问能力，便于 service 和 data 保持解耦。
type CharacterRepo interface {
	Create(ctx context.Context, character *data.Character) (*data.Character, error)
	List(ctx context.Context, novelID int64, offset, limit int) ([]data.Character, int64, error)
	FindByID(ctx context.Context, novelID int64, characterID uint) (*data.Character, error)
	Update(ctx context.Context, character *data.Character) (*data.Character, error)
	Delete(ctx context.Context, novelID int64, characterID uint) error
}

// ListCharacterResult 包含角色列表和分页信息，列表项字段由 data 层查询列控制。
type ListCharacterResult struct {
	Items    []data.Character
	Total    int64
	Page     int
	PageSize int
}

// NewCharacterUseCase 创建角色业务用例，并注入小说仓储用于归属校验。
func NewCharacterUseCase(novelData NovelRepo, characterData CharacterRepo) *CharacterUseCase {
	return &CharacterUseCase{
		novelData:     novelData,
		characterData: characterData,
	}
}

// CreateCharacter 新增角色，并确认角色归属的小说存在。
func (uc *CharacterUseCase) CreateCharacter(ctx context.Context, params dto.CreateCharacterRequest) (*data.Character, error) {
	if err := uc.ensureNovelExists(ctx, params.NovelID); err != nil {
		return nil, err
	}

	status := int16(1)
	if params.Status != nil {
		status = *params.Status
	}

	character := &data.Character{
		NovelID:                  params.NovelID,
		Name:                     params.Name,
		Gender:                   params.Gender,
		Intro:                    params.Intro,
		Personality:              params.Personality,
		Appearance:               params.Appearance,
		Background:               params.Background,
		Ability:                  params.Ability,
		Motivation:               params.Motivation,
		PlotDirection:            params.PlotDirection,
		FirstAppearanceChapterID: params.FirstAppearanceChapterID,
		AppearanceImgURL:         params.AppearanceImgURL,
		Status:                   status,
		CharactersTags:           pq.StringArray(params.CharactersTags),
	}
	return uc.characterData.Create(ctx, character)
}

// ListCharacters 按分页参数查询单本小说下的角色列表，返回字段由 data 层控制为列表摘要。
func (uc *CharacterUseCase) ListCharacters(ctx context.Context, novelID int64, page, pageSize int) (*ListCharacterResult, error) {
	if err := uc.ensureNovelExists(ctx, novelID); err != nil {
		return nil, err
	}

	offset := (page - 1) * pageSize

	items, total, err := uc.characterData.List(ctx, novelID, offset, pageSize)
	if err != nil {
		slog.ErrorContext(ctx, "查询角色列表失败", "err", err)
		return nil, err
	}

	return &ListCharacterResult{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// GetCharacter 查询指定小说下的角色详情。
func (uc *CharacterUseCase) GetCharacter(ctx context.Context, novelID int64, characterID uint) (*data.Character, error) {
	return uc.characterData.FindByID(ctx, novelID, characterID)
}

// UpdateCharacter 按请求参数全量保存角色资料。
func (uc *CharacterUseCase) UpdateCharacter(ctx context.Context, params dto.UpdateCharacterRequest) (*data.Character, error) {
	return uc.characterData.Update(ctx, &data.Character{
		Model:                    gorm.Model{ID: params.CharacterID},
		NovelID:                  params.NovelID,
		Name:                     params.Name,
		Gender:                   params.Gender,
		Intro:                    params.Intro,
		Personality:              params.Personality,
		Appearance:               params.Appearance,
		Background:               params.Background,
		Ability:                  params.Ability,
		Motivation:               params.Motivation,
		PlotDirection:            params.PlotDirection,
		FirstAppearanceChapterID: params.FirstAppearanceChapterID,
		AppearanceImgURL:         params.AppearanceImgURL,
		Status:                   params.Status,
		CharactersTags:           pq.StringArray(params.CharactersTags),
	})
}

// DeleteCharacter 删除指定小说下的角色。
func (uc *CharacterUseCase) DeleteCharacter(ctx context.Context, novelID int64, characterID uint) error {
	return uc.characterData.Delete(ctx, novelID, characterID)
}

// ensureNovelExists 确认角色归属的小说存在，避免给不存在的小说维护角色资料。
func (uc *CharacterUseCase) ensureNovelExists(ctx context.Context, novelID int64) error {
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
