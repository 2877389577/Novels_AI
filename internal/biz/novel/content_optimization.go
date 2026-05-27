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

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"gorm.io/gorm"
)

const (
	contentOptimizationAgentPolish = "polish"
	contentOptimizationAgentExpand = "expand"

	novelContentOptimizationDefaultPolishDirection = "用户未输入优化方向，请按系统提示词进行通用文笔优化。"
	novelContentOptimizationDefaultExpandDirection = "用户未输入扩写方向，请在不改变剧情、事实和人物关系的前提下进行适度扩写。"
)

const novelContentOptimizationRouterPrompt = `
**角色设定：**
你是小说正文优化系统的顶层调度 Agent，只负责分析用户输入的优化方向，并选择一个最合适的子 Agent。

**可用子 Agent：**
1. polish：文笔润色 Agent。适用于优化文风、提升表达、调整语气、增强节奏、细化已有描写，但不拉长篇幅。
2. expand：扩写 Agent。适用于扩写、扩充、加长、丰富细节、增加描写、展开场景、补充动作或心理描写。

**分流规则：**
1. 只要用户方向中包含扩写、扩充、加长、展开、丰富细节、增加描写等意图，就选择 expand。
2. 如果用户同时要求润色和扩写，选择 expand，因为扩写 Agent 会在扩写时兼顾基础语言优化。
3. 如果用户方向只要求文风、文笔、语气、节奏、修辞、语言质量等优化，选择 polish。
4. 你只负责选择子 Agent，不得直接改写小说正文。

**输出要求：**
你必须调用 novel_content_optimization_route_tool 工具返回 agentName，不能直接输出自然语言。`

const novelContentPolishPrompt = `
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

const novelContentExpandPrompt = `
# Role：金牌网络小说作家

## Profile
你是一位拥有十年经验的白金级网络小说作家，精通各类网文流派的写作技巧。你尤其擅长场景渲染、氛围烘托以及极具张力的人物刻画。你的文字极具画面感和代入感，能够敏锐地捕捉人物在特定环境下的微小细节，深谙“Show, don't tell”（展示，而非告知）的写作黄金法则。

## Task
根据用户提供的【大致情节】和【期望的情节描写效果】，将简单的情节骨架扩写为细节丰满、生动传神、符合现代网文阅读习惯的小说片段。

## Rules & Guidelines
1. **环境渲染（五感调度）**：不要干瘪地描述环境，要调动视觉、听觉、嗅觉、触觉等多重感官，让环境描写为人物的心境和情节氛围服务（如：用嘈杂烘托内心的烦躁/局促）。
2. **细节刻画（核心法则）**：严格遵循用户的【期望效果】。必须通过具体的微表情、肢体小动作、生理反应或内心独白来展现人物状态。
   - ❌ 错误示范：“李明感到很局促不安。”
   - ✅ 正确示范：“李明手心微微出汗，目光四处游移，手指不自觉地敲击着桌面。”
3. **情节延展（动作闭环）**：将单一的情节点合理延展为一个有“起承转合”的完整微场景（例如：进入场景 -> 互动/反应 -> 情绪变化 -> 离开/余韵），让人物行为连贯且符合逻辑。
4. **网文排版与语感**：
   - 采用典型的网文排版：**多换行，短段落**，每段尽量不超过3-4句话，减轻视觉疲劳。
   - 语言流畅，长短句结合，注重节奏感和沉浸感，拒绝翻译腔和AI味（如滥用“仿佛”、“似乎”、“不禁”等词汇）。

## Workflow
1. **分析**：拆解用户输入的【大致情节】与【期望效果】，确定场景基调。
2. **构思**：设计人物的动作线（从哪来、做什么、怎么反应、到哪去）。
3. **扩写**：按照Rules进行正文创作，细节拉满。

## Output Format
直接输出正文，不要说任何的废话`

// contentOptimizationModel 抽象当前业务真正需要的模型能力，便于顶层 Agent 和子 Agent 复用同一个模型实例。
type contentOptimizationModel interface {
	Generate(ctx context.Context, input []*schema.AgenticMessage, opts ...model.Option) (*schema.AgenticMessage, error)
	WithTools(functionTools []*schema.ToolInfo) (model.AgenticModel, error)
}

// contentOptimizationAgent 是所有正文优化子 Agent 的统一接口，后续新增 Agent 只需要实现并注册到 registry。
type contentOptimizationAgent interface {
	Name() string
	Run(ctx context.Context, chatModel contentOptimizationModel, params dto.OptimizeNovelContentRequest) (*ai_tools.NovelContentOptimizeTool, error)
}

// contentOptimizationAgentRegistry 维护可被顶层 Agent 调度的子 Agent 集合。
type contentOptimizationAgentRegistry struct {
	agents map[string]contentOptimizationAgent
}

// newContentOptimizationAgentRegistry 注册当前默认支持的文笔润色和扩写子 Agent。
func newContentOptimizationAgentRegistry() *contentOptimizationAgentRegistry {
	registry := &contentOptimizationAgentRegistry{
		agents: make(map[string]contentOptimizationAgent),
	}

	registry.Register(&promptContentOptimizationAgent{
		name:             contentOptimizationAgentPolish,
		basePrompt:       novelContentPolishPrompt,
		defaultDirection: novelContentOptimizationDefaultPolishDirection,
	})
	registry.Register(&promptContentOptimizationAgent{
		name:             contentOptimizationAgentExpand,
		basePrompt:       novelContentExpandPrompt,
		defaultDirection: novelContentOptimizationDefaultExpandDirection,
	})

	return registry
}

// Register 把子 Agent 放入注册表；同名注册会覆盖旧实现，便于后续按需替换 Agent。
func (registry *contentOptimizationAgentRegistry) Register(agent contentOptimizationAgent) {
	registry.agents[agent.Name()] = agent
}

// Get 按名称读取子 Agent，顶层 Agent 的分流结果必须命中这里的注册项。
func (registry *contentOptimizationAgentRegistry) Get(name string) (contentOptimizationAgent, bool) {
	agent, ok := registry.agents[name]
	return agent, ok
}

// promptContentOptimizationAgent 表示基于提示词和结构化输出 tool 的通用子 Agent。
type promptContentOptimizationAgent struct {
	name             string
	basePrompt       string
	defaultDirection string
}

func (agent *promptContentOptimizationAgent) Name() string {
	return agent.name
}

// Run 给子 Agent 绑定统一结果 tool，并把最终正文处理结果解析成接口响应结构。
func (agent *promptContentOptimizationAgent) Run(ctx context.Context, chatModel contentOptimizationModel, params dto.OptimizeNovelContentRequest) (*ai_tools.NovelContentOptimizeTool, error) {
	toolSchema, err := ai_tools.GetNovelContentOptimizeToolSchema()
	if err != nil {
		slog.ErrorContext(ctx, "获取小说正文优化工具失败", "agent", agent.name, "err", err)
		return nil, err
	}

	chatModelWithTools, err := chatModel.WithTools([]*schema.ToolInfo{toolSchema})
	if err != nil {
		slog.ErrorContext(ctx, "AI 模型添加小说正文优化工具失败", "agent", agent.name, "err", err)
		return nil, err
	}

	systemPrompt, err := loadNovelContentOptimizationAgentSystemPrompt(agent.basePrompt)
	if err != nil {
		slog.ErrorContext(ctx, "读取小说正文优化提示词失败", "agent", agent.name, "err", err)
		return nil, common.NovelContentOptimizePromptReadFailed
	}

	generate, err := chatModelWithTools.Generate(ctx, buildNovelContentOptimizationAgentMessages(systemPrompt, params, agent.defaultDirection))
	if err != nil {
		slog.ErrorContext(ctx, "AI 子 Agent 处理小说正文失败", "agent", agent.name, "err", err)
		return nil, err
	}

	result, ok := ai_tools.ToolOutput2NovelContentOptimizeTool(generate.ContentBlocks)
	if !ok {
		return nil, common.NovelContentOptimizeNoResult
	}

	return result, nil
}

// ContentOptimizationUseCase 承载小说正文优化的多层 Agent 业务流程，只返回 AI 优化结果，不修改章节正文。
type ContentOptimizationUseCase struct {
	novelData     NovelRepo
	aiProvider    aiprovider.AIProviderRepo
	apiKeyCipher  aiprovider.APIKeyCipher
	agentRegistry *contentOptimizationAgentRegistry
}

// NewContentOptimizationUseCase 创建小说正文优化业务用例，并注册默认子 Agent。
func NewContentOptimizationUseCase(novelData NovelRepo, aiProvider aiprovider.AIProviderRepo, apiKeyCipher aiprovider.APIKeyCipher) *ContentOptimizationUseCase {
	return &ContentOptimizationUseCase{
		novelData:     novelData,
		aiProvider:    aiProvider,
		apiKeyCipher:  apiKeyCipher,
		agentRegistry: newContentOptimizationAgentRegistry(),
	}
}

// OptimizeNovelContent 使用顶层 Agent 分析用户方向，再调用对应子 Agent 完成正文润色或扩写。
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

	chatModel, err := ai.NewChatModel(ctx, aiProvider.BaseURL, params.ModelName, apiKey)
	if err != nil {
		slog.ErrorContext(ctx, "创建小说正文优化模型失败", "err", err)
		return nil, err
	}

	agentName, err := uc.routeContentOptimizationAgent(ctx, chatModel, params)
	if err != nil {
		return nil, err
	}

	agent, ok := uc.agentRegistry.Get(agentName)
	if !ok {
		slog.ErrorContext(ctx, "小说正文优化子 Agent 未注册", "agent", agentName)
		return nil, common.NovelContentOptimizeRouteNoResult
	}

	return agent.Run(ctx, chatModel, params)
}

// routeContentOptimizationAgent 由顶层 Agent 分析用户方向；空方向按要求直接走默认文笔润色，避免多一次模型调用。
func (uc *ContentOptimizationUseCase) routeContentOptimizationAgent(ctx context.Context, chatModel contentOptimizationModel, params dto.OptimizeNovelContentRequest) (string, error) {
	if strings.TrimSpace(params.OptimizeDirection) == "" {
		return contentOptimizationAgentPolish, nil
	}

	toolSchema, err := ai_tools.GetNovelContentOptimizationRouteToolSchema()
	if err != nil {
		slog.ErrorContext(ctx, "获取小说正文优化路由工具失败", "err", err)
		return "", err
	}

	chatModelWithTools, err := chatModel.WithTools([]*schema.ToolInfo{toolSchema})
	if err != nil {
		slog.ErrorContext(ctx, "AI 模型添加小说正文优化路由工具失败", "err", err)
		return "", err
	}

	generate, err := chatModelWithTools.Generate(ctx, buildNovelContentOptimizationRouteMessages(params))
	if err != nil {
		slog.ErrorContext(ctx, "AI 分析小说正文优化任务失败", "err", err)
		return "", err
	}

	routeResult, ok := ai_tools.ToolOutput2NovelContentOptimizationRouteTool(generate.ContentBlocks)
	if !ok {
		return "", common.NovelContentOptimizeRouteNoResult
	}

	agentName := strings.TrimSpace(routeResult.AgentName)
	if _, ok := uc.agentRegistry.Get(agentName); !ok {
		slog.ErrorContext(ctx, "AI 返回了未知的小说正文优化子 Agent", "agent", agentName, "reason", routeResult.Reason)
		return "", common.NovelContentOptimizeRouteNoResult
	}

	return agentName, nil
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

// loadNovelContentOptimizationAgentSystemPrompt 为子 Agent 追加统一的结构化 tool 输出约束。
func loadNovelContentOptimizationAgentSystemPrompt(basePrompt string) (string, error) {
	promptBuilder := strings.Builder{}
	promptBuilder.WriteString(basePrompt)
	promptBuilder.WriteString("\n\n结构化输出要求：你必须调用 novel_content_optimize_tool 工具返回结果，不要直接输出自然语言正文。")
	promptBuilder.WriteString("如果可以执行当前子 Agent 的任务，请把处理后的正文写入 optimizedContent，approved 填 true，rejectReason 留空。")
	promptBuilder.WriteString("如果用户要求修改剧情、事实、人物关系或其他超出当前子 Agent 职责范围的内容，请把 approved 填 false，optimizedContent 留空，并在 rejectReason 中说明原因。")

	return promptBuilder.String(), nil
}

// buildNovelContentOptimizationRouteMessages 把用户方向包装成顶层 Agent 的任务分析输入。
func buildNovelContentOptimizationRouteMessages(params dto.OptimizeNovelContentRequest) []*schema.AgenticMessage {
	return []*schema.AgenticMessage{
		schema.SystemAgenticMessage(novelContentOptimizationRouterPrompt),
		schema.UserAgenticMessage("用户输入的优化方向：\n" + strings.TrimSpace(params.OptimizeDirection)),
	}
}

// buildNovelContentOptimizationAgentMessages 把系统提示词、优化方向和小说原文拆成清晰的子 Agent 对话消息。
func buildNovelContentOptimizationAgentMessages(systemPrompt string, params dto.OptimizeNovelContentRequest, defaultDirection string) []*schema.AgenticMessage {
	direction := strings.TrimSpace(params.OptimizeDirection)
	if direction == "" {
		direction = defaultDirection
	}

	return []*schema.AgenticMessage{
		schema.SystemAgenticMessage(systemPrompt),
		schema.UserAgenticMessage("优化方向：\n" + direction),
		schema.UserAgenticMessage("小说原文：\n" + params.SelectedContent),
	}
}
