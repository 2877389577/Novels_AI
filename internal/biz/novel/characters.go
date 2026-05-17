package novel

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"Novels_AI/backend/internal/data"
	"Novels_AI/backend/internal/pkg/common"

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
	Update(ctx context.Context, novelID int64, characterID uint, values map[string]any) (*data.Character, error)
	Delete(ctx context.Context, novelID int64, characterID uint) error
}

// CreateCharacterParams 是新增角色时 service 层传入的完整业务参数。
type CreateCharacterParams struct {
	NovelID                  int64
	Name                     string
	Gender                   string
	Intro                    string
	Personality              string
	Appearance               string
	Background               string
	Ability                  string
	Motivation               string
	PlotDirection            string
	FirstAppearanceChapterID *int64
	AppearanceImgURL         string
	Status                   *int16
	CharactersTags           []string
}

// UpdateCharacterParams 使用指针字段区分“未传”和“传入零值”，支持局部更新。
type UpdateCharacterParams struct {
	NovelID                  int64
	CharacterID              uint
	Name                     *string
	Gender                   *string
	Intro                    *string
	Personality              *string
	Appearance               *string
	Background               *string
	Ability                  *string
	Motivation               *string
	PlotDirection            *string
	FirstAppearanceChapterID *int64
	AppearanceImgURL         *string
	Status                   *int16
	CharactersTags           *[]string
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

// CreateCharacter 校验角色基础信息后新增角色，并确认角色归属的小说存在。
func (uc *CharacterUseCase) CreateCharacter(ctx context.Context, params CreateCharacterParams) (*data.Character, error) {
	name, err := normalizeCharacterName(params.Name)
	if err != nil {
		return nil, err
	}
	if err := uc.ensureNovelExists(ctx, params.NovelID); err != nil {
		return nil, err
	}

	status := int16(1)
	if params.Status != nil {
		status = *params.Status
	}

	character := &data.Character{
		NovelID:                  params.NovelID,
		Name:                     name,
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

	page, pageSize = normalizePagination(page, pageSize)
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

// UpdateCharacter 按请求中出现的字段局部更新角色资料。
func (uc *CharacterUseCase) UpdateCharacter(ctx context.Context, params UpdateCharacterParams) (*data.Character, error) {
	values := make(map[string]any)
	if params.Name != nil {
		name, err := normalizeCharacterName(*params.Name)
		if err != nil {
			return nil, err
		}
		values["name"] = name
	}
	if params.Gender != nil {
		values["gender"] = *params.Gender
	}
	if params.Intro != nil {
		values["intro"] = *params.Intro
	}
	if params.Personality != nil {
		values["personality"] = *params.Personality
	}
	if params.Appearance != nil {
		values["appearance"] = *params.Appearance
	}
	if params.Background != nil {
		values["background"] = *params.Background
	}
	if params.Ability != nil {
		values["ability"] = *params.Ability
	}
	if params.Motivation != nil {
		values["motivation"] = *params.Motivation
	}
	if params.PlotDirection != nil {
		values["plot_direction"] = *params.PlotDirection
	}
	if params.FirstAppearanceChapterID != nil {
		values["first_appearance_chapter_id"] = *params.FirstAppearanceChapterID
	}
	if params.AppearanceImgURL != nil {
		values["appearance_img_url"] = *params.AppearanceImgURL
	}
	if params.Status != nil {
		values["status"] = *params.Status
	}
	if params.CharactersTags != nil {
		values["characters_tags"] = pq.StringArray(*params.CharactersTags)
	}

	return uc.characterData.Update(ctx, params.NovelID, params.CharacterID, values)
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

// normalizeCharacterName 统一修剪角色名，并把空角色名转换成公共业务错误。
func normalizeCharacterName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", common.CharacterNameRequired
	}

	return name, nil
}
