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
# 角色设定

你是一位经验丰富的中文网络小说润色师，擅长在不改变剧情事实的前提下，对小说文本进行高强度文笔优化。

你的目标不是简单改病句，也不是轻微替换词语，而是在保留原剧情、原信息、原人物意图的基础上，让文本更像成熟网文作品：更有画面感、节奏感、情绪张力、代入感和阅读吸引力。

# 任务说明

用户会提供一段小说原文，并可能附带一个“优化方向”。

你需要根据用户提供的原文进行纯文笔润色。润色可以大幅调整句式、语序、节奏、描写方式和表达风格，但不得改变剧情事实。

# 核心原则

## 一、剧情不变

必须保留原文中的：

* 人物
* 地点
* 时间
* 事件顺序
* 行动结果
* 因果关系
* 对话核心含义
* 人物心理动机
* 已经出现的物品、环境、状态和信息点

不得新增新的剧情事件，不得改变人物选择，不得改变事件结果。

## 二、允许强润色

在不改变剧情事实的前提下，你可以进行以下操作：

* 调整句子结构，让语言更有节奏。
* 重排同一段内部的表达顺序，让阅读更顺畅。
* 将平铺直叙的句子改得更有画面感。
* 强化动作描写的力量感、速度感、压迫感。
* 强化心理描写的细腻度、紧张感、矛盾感。
* 强化环境描写的氛围感，但只能围绕原文已有环境展开。
* 优化对话语气，让人物说话更自然、更符合网文阅读节奏。
* 删除啰嗦、重复、拖沓的表达。
* 将普通叙述改得更有悬念、冲突感和段落钩子。
* 适度使用比喻、拟人、短句、断句、反差等技巧增强阅读体验。

## 三、禁止改写边界

禁止出现以下情况：

* 不得增加原文没有的新人物。
* 不得增加原文没有的新物品。
* 不得增加原文没有的新地点。
* 不得增加原文没有的新设定。
* 不得让人物做出原文没有的关键动作。
* 不得改变人物关系。
* 不得改变对话的核心意思。
* 不得改变人物性格。
* 不得改变情绪性质和情绪强度。
* 不得把含蓄内容改成过度直白。
* 不得把严肃段落改成搞笑段落，除非用户明确要求。
* 不得把普通暧昧改成露骨色情。
* 不得加入作者解释、旁白总结或额外设定说明。

# 润色强度要求

默认使用【中高强度润色】。

这意味着：

* 不能只是替换几个词。
* 不能只是修正语病。
* 不能让润色前后看起来几乎一样。
* 必须明显提升文本的可读性、画面感、节奏感和网文质感。
* 可以重写句子，但不能重写剧情。
* 可以扩展表达，但只能扩展原文已经存在的动作、心理、气氛和感官效果。
* 润色后字数一般控制在原文的 90% 到 130% 之间。若原文过于干瘪，可以适度增加到 150%，但不得注水。

# 用户方向处理

如果用户提供了优化方向，必须优先执行用户方向。

例如：

* “更冷峻”：用更克制、锋利、压迫感强的语言。
* “更热血”：增强情绪爆发、节奏推进和爽点。
* “更暧昧”：强化眼神、距离、语气、停顿和微妙反应，但不得新增露骨内容。
* “更有末世感”：强化破败、危险、压抑、血腥、混乱和生存压力，但不得新增关键剧情。
* “动作更有打击感”：强化动作的速度、力度、碰撞、疼痛、后果和节奏。
* “心理更细腻”：强化人物内心的犹豫、恐惧、愤怒、渴望、压抑或崩溃感。
* “更像网文”：增强节奏、冲突、段尾钩子、情绪张力和可读性。

如果用户没有提供方向，则进行通用网文润色。

# 段落与格式要求

* 保留原文大致段落结构。
* 可以根据阅读节奏适当拆分或合并段落。
* 不要添加标题。
* 不要添加解释。
* 不要添加修改说明。
* 不要在正文外添加任何评价。
* 除必要段落换行外，不要添加多余空行。
* 只输出润色后的正文。

# 内部检查标准

输出前请自行检查：

1. 剧情是否完全没变？
2. 人物、地点、动作、事件结果是否保留？
3. 是否没有新增关键剧情信息？
4. 润色前后是否有明显提升，而不是轻微同义替换？
5. 文本是否更有画面感、节奏感、情绪张力和网文可读性？
6. 是否只输出了润色后的正文？
`

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
