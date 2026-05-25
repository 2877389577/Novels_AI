package novel

import (
	"Novels_AI/backend/internal/ai"
	"Novels_AI/backend/internal/ai/ai_tools"
	"Novels_AI/backend/internal/biz/aiprovider"
	"context"
	"errors"
	"log/slog"
	"strings"

	"Novels_AI/backend/internal/data"
	"Novels_AI/backend/internal/data/dto"

	"github.com/cloudwego/eino/schema"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Character = data.Character

// CharacterUseCase 承载角色资料的业务规则，并复用小说仓储校验归属小说是否存在。
type CharacterUseCase struct {
	novelData     NovelRepo
	characterData CharacterRepo
	aiProvider    aiprovider.AIProviderRepo
	chapterData   ChapterRepo
	aPIKeyCipher  aiprovider.APIKeyCipher
}

// CharacterRepo 定义角色业务层需要的数据访问能力，便于 service 和 data 保持解耦。
type CharacterRepo interface {
	Create(ctx context.Context, character *data.Character) (*data.Character, error)
	List(ctx context.Context, novelID int64, offset, limit int) ([]data.Character, int64, error)
	ListDedupCharacters(ctx context.Context, novelID int64) ([]data.Character, error)
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
func NewCharacterUseCase(novelData NovelRepo, characterData CharacterRepo, aiProvider aiprovider.AIProviderRepo, chapterData ChapterRepo, apiKeyCipher aiprovider.APIKeyCipher) *CharacterUseCase {
	return &CharacterUseCase{
		novelData:     novelData,
		characterData: characterData,
		aiProvider:    aiProvider,
		chapterData:   chapterData,
		aPIKeyCipher:  apiKeyCipher,
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

// GenerateCharacterCard 使用 AI 生成角色卡片。
func (uc *CharacterUseCase) GenerateCharacterCard(ctx context.Context, chapterID int64, modelName string) ([]*ai_tools.CharacterCardTool, error) {
	aiProvider, err := uc.aiProvider.FindEnabled(ctx)
	if err != nil {
		return nil, err
	}
	if aiProvider == nil {
		return nil, nil
	}

	// 先按章节 ID 查询章节内容和小说 ID，后续用小说 ID 过滤已存在的角色。
	chapter, err := uc.chapterData.FindByID(ctx, 0, uint(chapterID))
	if err != nil {
		slog.ErrorContext(ctx, "查询章节失败", "err", err)
		return nil, err
	}

	apikey, err := uc.aPIKeyCipher.Decrypt(aiProvider.APIKeyEncrypted)
	if err != nil {
		slog.ErrorContext(ctx, "解密 API Key 失败", "err", err)
		return nil, err
	}

	model, err := ai.NewChatModel(ctx, aiProvider.BaseURL, modelName, apikey)
	if err != nil {
		slog.ErrorContext(ctx, "创建模型失败", "err", err)
		return nil, err
	}

	// 获取对应的tool，用来格式化模型输出
	toolSchema, err := ai_tools.GetCharacterCardToolSchema()
	if err != nil {
		slog.ErrorContext(ctx, "获取角色卡片工具失败", "err", err)
		return nil, err
	}

	chatModelWithTools, err := model.WithTools([]*schema.ToolInfo{toolSchema})
	if err != nil {
		slog.ErrorContext(ctx, "AI模型添加工具失败", "err", err)
		return nil, err
	}

	content := strings.Builder{}
	content.WriteString("小说章节内容:\n")
	content.WriteString(chapter.Content)

	// 提示词和用户输入
	historyMsg := []*schema.AgenticMessage{
		schema.SystemAgenticMessage(`你是一个专业的小说角色分析师，你的任务是根据用户提供的小说章节内容，分析出小说中出现的角色信息，并将信息格式化为 'character_card_tool' 工具的入参，而不是直接输出文本。
注意：如果用户提供的章节内容中没有角色信息，不要调用工具生成信息。`),
		schema.UserAgenticMessage(content.String()),
	}

	generate, err := chatModelWithTools.Generate(ctx, historyMsg)
	if err != nil {
		slog.ErrorContext(ctx, "AI生成角色卡片失败", "err", err)
		return nil, err
	}
	// 解析模型输出
	cs := ai_tools.ToolOutput2CharacterCardTool(generate.ContentBlocks)
	filtered, err := uc.filterDuplicateCharacterCards(ctx, chapter.NovelID, cs)
	if err != nil {
		return nil, err
	}

	return filtered, nil
}

// filterDuplicateCharacterCards 按“角色名 + 性别”过滤数据库已有角色和本次 AI 输出中的重复角色。
func (uc *CharacterUseCase) filterDuplicateCharacterCards(ctx context.Context, novelID int64, cards []*ai_tools.CharacterCardTool) ([]*ai_tools.CharacterCardTool, error) {
	existingCharacters, err := uc.characterData.ListDedupCharacters(ctx, novelID)
	if err != nil {
		slog.ErrorContext(ctx, "查询已有角色失败", "err", err)
		return nil, err
	}

	seen := make(map[string]struct{}, len(existingCharacters)+len(cards))
	for _, character := range existingCharacters {
		key := characterDedupKey(character.Name, character.Gender)
		if key == "" {
			continue
		}
		seen[key] = struct{}{}
	}

	filtered := make([]*ai_tools.CharacterCardTool, 0, len(cards))
	for _, card := range cards {
		if card == nil {
			continue
		}

		key := characterDedupKey(card.Name, card.Gender)
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}

		seen[key] = struct{}{}
		filtered = append(filtered, card)
	}

	return filtered, nil
}

// characterDedupKey 生成角色去重键，只清理前后空白，不做同义名、繁简或大小写转换。
func characterDedupKey(name, gender string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}

	return name + "\x00" + strings.TrimSpace(gender)
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
