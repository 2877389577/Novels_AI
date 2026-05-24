package novel

import (
	"Novels_AI/backend/internal/ai"
	"Novels_AI/backend/internal/ai/ai_tools"
	"Novels_AI/backend/internal/biz/aiprovider"
	"Novels_AI/backend/internal/data/dto"
	"Novels_AI/backend/internal/pkg/common"
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/cloudwego/eino/schema"
	"gorm.io/gorm"
)

const novelContentOptimizePromptPath = `
**角色设定：**
你是一位资深的中文文学编辑，专精于在不改动故事内核的前提下，对文字进行精准且富有文学感的润色。

任务说明：
用户将提供一段小说原文，并可能附带一个明确的“优化方向”。你需要对原文进行纯文笔润色，可以且应当结合用户的方向，但必须严格遵守以下所有铁律。

**铁律（优先级最高，无条件执行）：**
1. 剧情零变动：不改动任何情节、因果、事件顺序与故事走向。
2. 事实信息锁定：保留所有人名、地名、时间、环境细节、物品、动作事实等客观描述。
3. 心理与情感基调保留：原文已有的人物心理和情感，可优化其表达方式，但不得改变情绪性质、程度及产生原因。
4. 对话处理：对话可优化语气与节奏，但不得改变原意、信息量及人物性格。
5. 篇幅控制：润色后段落长度与原文基本一致，避免明显膨胀或缩减。
6. 只输出结果：只返回润色后段落，不加任何前言、标注或注释。
7. 如果用户要求你修改剧情内容，你必须拒绝，并说明拒绝原因是你不能修改剧情内容，只做文笔优化。
8. 优化的内容除了必须的段落换行符外，不要添加多余的换行符。

**用户方向处理规则：**
- 如果用户给出了优化方向（例如：“让文风更冷峻”“强化动作的打击感”“用更细腻的笔触描写心理”），请在润色时将该方向作为主要风格指引，调整选词、句式、修辞和描写详略。此过程仍须完全遵守上述铁律。
- 如果用户未给出方向，则进行通用文笔优化，提升语言的生动性、精准度与文学美感。
- 如果用户给的优化方向比较模糊，你可以根据下述的通用方法和用户方向处理规则，进行调整。

**通用润色方法（在铁律和用户方向框架下运用）：**
- 词语替换：用更精准、更具画面感或符合风格的词汇替换笼统平淡的用词。
- 句式变化：运用长短句、整散句结合等手段增强节奏与呼吸感。
- 描写强化：优化人物外貌、动作、神态、心理及场景描写，使其更细腻可感，但不额外添加信息。
- 修辞运用：适度使用比喻、拟人等修辞，以不改变原意为底线。
- 删繁去赘：去除啰嗦重复的表达，但绝不丢失信息点。

请严格按照以上所有要求执行润色。`

// ContentOptimizationUseCase 承载小说正文润色的业务流程，只返回 AI 优化结果，不修改章节正文。
type ContentOptimizationUseCase struct {
	novelData    NovelRepo
	aiProvider   aiprovider.AIProviderRepo
	apiKeyCipher aiprovider.APIKeyCipher
}

// NewContentOptimizationUseCase 创建小说正文优化业务用例。
func NewContentOptimizationUseCase(novelData NovelRepo, aiProvider aiprovider.AIProviderRepo, apiKeyCipher aiprovider.APIKeyCipher) *ContentOptimizationUseCase {
	return &ContentOptimizationUseCase{
		novelData:    novelData,
		aiProvider:   aiProvider,
		apiKeyCipher: apiKeyCipher,
	}
}

// OptimizeNovelContent 使用当前启用的 AI 提供商和指定模型，对用户选中的小说正文做文笔优化。
func (uc *ContentOptimizationUseCase) OptimizeNovelContent(ctx context.Context, params dto.OptimizeNovelContentRequest) (*ai_tools.NovelContentOptimizeTool, error) {
	if err := uc.ensureNovelExists(ctx, params.NovelID); err != nil {
		return nil, err
	}

	aiProvider, err := uc.aiProvider.FindEnabled(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "查询启用中的 AI 提供商失败", "err", err)
		return nil, err
	}
	if aiProvider == nil {
		return nil, common.AIProviderNotFound
	}

	apiKey, err := uc.apiKeyCipher.Decrypt(aiProvider.APIKeyEncrypted)
	if err != nil {
		slog.ErrorContext(ctx, "解密 AI Provider API Key 失败", "err", err)
		return nil, common.AIProviderAPIKeyDecryptFailed
	}

	model, err := ai.NewChatModel(ctx, aiProvider.BaseURL, params.ModelName, apiKey)
	if err != nil {
		slog.ErrorContext(ctx, "创建小说正文优化模型失败", "err", err)
		return nil, err
	}

	toolSchema, err := ai_tools.GetNovelContentOptimizeToolSchema()
	if err != nil {
		slog.ErrorContext(ctx, "获取小说正文优化工具失败", "err", err)
		return nil, err
	}

	chatModelWithTools, err := model.WithTools([]*schema.ToolInfo{toolSchema})
	if err != nil {
		slog.ErrorContext(ctx, "AI 模型添加小说正文优化工具失败", "err", err)
		return nil, err
	}

	systemPrompt, err := loadNovelContentOptimizeSystemPrompt()
	if err != nil {
		slog.ErrorContext(ctx, "读取小说正文优化提示词失败", "err", err)
		return nil, common.NovelContentOptimizePromptReadFailed
	}

	generate, err := chatModelWithTools.Generate(ctx, buildNovelContentOptimizeMessages(systemPrompt, params))
	if err != nil {
		slog.ErrorContext(ctx, "AI 优化小说正文失败", "err", err)
		return nil, err
	}

	result, ok := ai_tools.ToolOutput2NovelContentOptimizeTool(generate.ContentBlocks)
	if !ok {
		return nil, common.NovelContentOptimizeNoResult
	}

	return result, nil
}

// ensureNovelExists 确认正文优化归属的小说存在，避免对不存在的小说发起 AI 优化流程。
func (uc *ContentOptimizationUseCase) ensureNovelExists(ctx context.Context, novelID int64) error {
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

// loadNovelContentOptimizeSystemPrompt 读取项目根目录下的正文优化提示词，并追加 tool 输出约束。
func loadNovelContentOptimizeSystemPrompt() (string, error) {
	promptBuilder := strings.Builder{}
	promptBuilder.WriteString(novelContentOptimizePromptPath)
	promptBuilder.WriteString("\n\n结构化输出要求：你必须调用 novel_content_optimize_tool 工具返回结果，不要直接输出自然语言正文。")
	promptBuilder.WriteString("如果可以执行文笔优化，请把润色后的正文写入 optimizedContent，approved 填 true，rejectReason 留空。")
	promptBuilder.WriteString("如果用户要求修改剧情、事实、人物关系或其他超出文笔优化范围的内容，请把 approved 填 false，optimizedContent 留空，并在 rejectReason 中说明原因。")

	return promptBuilder.String(), nil
}

// buildNovelContentOptimizeMessages 把系统提示词、优化方向和小说原文拆成清晰的对话消息。
func buildNovelContentOptimizeMessages(systemPrompt string, params dto.OptimizeNovelContentRequest) []*schema.AgenticMessage {
	direction := strings.TrimSpace(params.OptimizeDirection)
	if direction == "" {
		direction = "用户未输入优化方向，请按系统提示词进行通用文笔优化。"
	}

	return []*schema.AgenticMessage{
		schema.SystemAgenticMessage(systemPrompt),
		schema.UserAgenticMessage("优化方向：\n" + direction),
		schema.UserAgenticMessage("小说原文：\n" + params.SelectedContent),
	}
}
