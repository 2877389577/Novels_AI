package novel

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"Novels_AI/backend/internal/ai"
	"Novels_AI/backend/internal/ai/ai_tools"
	"Novels_AI/backend/internal/biz/aiprovider"
	"Novels_AI/backend/internal/data"
	"Novels_AI/backend/internal/event"
	"Novels_AI/backend/internal/pkg/common"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino/schema"
	"gorm.io/datatypes"
)

const chapterSavedEventName = "chapter.saved"

type ChapterPlotAnalysis = data.ChapterPlotAnalysis

// ChapterSavedEvent 是章节保存成功后发布给剧情总结模块的最小事件载荷。
type ChapterSavedEvent struct {
	NovelID   int64
	ChapterID uint
	Title     string
	Content   string
}

type ChapterPlotAnalysisRepo interface {
	UpsertByChapterID(ctx context.Context, analysis *data.ChapterPlotAnalysis) (*data.ChapterPlotAnalysis, error)
	FindByChapterID(ctx context.Context, novelID int64, chapterID uint) (*data.ChapterPlotAnalysis, error)
}

type chapterPlotAnalyzer interface {
	Analyze(ctx context.Context, title, content string) (*ai_tools.ChapterPlotAnalysisTool, error)
}

// ChapterPlotAnalysisUseCase 负责响应章节保存事件、调用 AI 总结剧情并提供独立查询能力。
type ChapterPlotAnalysisUseCase struct {
	analysisData ChapterPlotAnalysisRepo
	analyzer     chapterPlotAnalyzer
}

func NewChapterPlotAnalysisUseCase(analysisData ChapterPlotAnalysisRepo, aiProvider aiprovider.AIProviderRepo, apiKeyCipher aiprovider.APIKeyCipher) *ChapterPlotAnalysisUseCase {
	return &ChapterPlotAnalysisUseCase{
		analysisData: analysisData,
		analyzer: &aiChapterPlotAnalyzer{
			aiProvider:   aiProvider,
			apiKeyCipher: apiKeyCipher,
		},
	}
}

// RegisterChapterPlotAnalysisEventHandlers 把剧情总结处理器挂到事件总线，后续新增订阅者时可在这里集中注册。
func RegisterChapterPlotAnalysisEventHandlers(bus *event.Bus, useCase *ChapterPlotAnalysisUseCase) {
	if bus == nil || useCase == nil {
		slog.Warn("章节剧情总结事件处理器注册失败", "eventName", chapterSavedEventName, "busNil", bus == nil, "useCaseNil", useCase == nil)
		return
	}

	slog.Info("注册章节剧情总结事件处理器", "eventName", chapterSavedEventName)

	bus.Subscribe(chapterSavedEventName, func(ctx context.Context, evt event.Event) error {
		payload, ok := evt.Payload.(ChapterSavedEvent)
		if !ok {
			err := errors.New("章节保存事件载荷类型错误")
			slog.ErrorContext(ctx, "章节剧情总结事件载荷类型错误", "eventName", evt.Name, "payloadType", fmt.Sprintf("%T", evt.Payload), "err", err)
			return err
		}

		// AI 章节剧情总结耗时较长，必须脱离保存章节的 HTTP 请求异步执行，避免用户等待模型生成。
		asyncCtx := context.WithoutCancel(ctx)
		go func(payload ChapterSavedEvent, evt event.Event) {
			start := time.Now()
			slog.InfoContext(asyncCtx, "开始异步生成章节剧情总结", "eventName", evt.Name, "novelId", payload.NovelID, "chapterId", payload.ChapterID)
			// 异步任务不能把 panic 传播到 HTTP 请求链路，这里统一记录章节信息和耗时后结束 goroutine。
			defer func() {
				if recovered := recover(); recovered != nil {
					slog.ErrorContext(asyncCtx, "异步生成章节剧情总结 panic", "eventName", evt.Name, "novelId", payload.NovelID, "chapterId", payload.ChapterID, "duration", time.Since(start).String(), "panic", recovered)
				}
			}()

			if err := useCase.HandleChapterSaved(asyncCtx, evt); err != nil {
				slog.ErrorContext(asyncCtx, "异步生成章节剧情总结失败", "eventName", evt.Name, "novelId", payload.NovelID, "chapterId", payload.ChapterID, "duration", time.Since(start).String(), "err", err)
				return
			}

			slog.InfoContext(asyncCtx, "异步生成章节剧情总结完成", "eventName", evt.Name, "novelId", payload.NovelID, "chapterId", payload.ChapterID, "duration", time.Since(start).String())
		}(payload, evt)

		return nil
	})
}

// HandleChapterSaved 处理章节保存事件；注册到事件总线时会异步调用，直接调用时返回处理错误便于单测覆盖。
func (uc *ChapterPlotAnalysisUseCase) HandleChapterSaved(ctx context.Context, evt event.Event) error {
	payload, ok := evt.Payload.(ChapterSavedEvent)
	if !ok {
		return errors.New("章节保存事件载荷类型错误")
	}

	slog.InfoContext(ctx, "开始调用 AI 生成章节剧情总结", "novelId", payload.NovelID, "chapterId", payload.ChapterID)
	result, err := uc.analyzer.Analyze(ctx, payload.Title, payload.Content)
	if err != nil {
		slog.ErrorContext(ctx, "AI 生成章节剧情总结失败", "novelId", payload.NovelID, "chapterId", payload.ChapterID, "err", err)
		return err
	}
	slog.InfoContext(ctx, "AI 章节剧情总结生成完成", "novelId", payload.NovelID, "chapterId", payload.ChapterID)

	analysis, err := chapterPlotAnalysisFromTool(payload, result)
	if err != nil {
		slog.ErrorContext(ctx, "转换章节剧情总结结果失败", "novelId", payload.NovelID, "chapterId", payload.ChapterID, "err", err)
		return err
	}

	_, err = uc.analysisData.UpsertByChapterID(ctx, analysis)
	if err != nil {
		slog.ErrorContext(ctx, "保存章节剧情总结失败", "novelId", payload.NovelID, "chapterId", payload.ChapterID, "err", err)
	}
	if err == nil {
		slog.InfoContext(ctx, "章节剧情总结已保存", "novelId", payload.NovelID, "chapterId", payload.ChapterID)
	}
	return err
}

func (uc *ChapterPlotAnalysisUseCase) GetChapterPlotAnalysis(ctx context.Context, novelID int64, chapterID uint) (*data.ChapterPlotAnalysis, error) {
	return uc.analysisData.FindByChapterID(ctx, novelID, chapterID)
}

type aiChapterPlotAnalyzer struct {
	aiProvider   aiprovider.AIProviderRepo
	apiKeyCipher aiprovider.APIKeyCipher
}

// Analyze 使用当前启用 AI Provider 的默认模型生成章节剧情总结。
func (a *aiChapterPlotAnalyzer) Analyze(ctx context.Context, title, content string) (*ai_tools.ChapterPlotAnalysisTool, error) {
	aiProvider, err := a.aiProvider.FindEnabled(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "查询启用中的 AI 提供商失败", "err", err)
		return nil, err
	}
	if aiProvider == nil {
		return nil, common.AIProviderNotFound
	}

	modelName, ok := enabledAIProviderDefaultModel(aiProvider)
	if !ok {
		return nil, common.ChapterPlotAnalysisModelNotFound
	}

	apiKey, err := a.apiKeyCipher.Decrypt(aiProvider.APIKeyEncrypted)
	if err != nil {
		slog.ErrorContext(ctx, "解密 AI Provider API Key 失败", "err", err)
		return nil, common.AIProviderAPIKeyDecryptFailed
	}

	chatModel, err := ai.NewChatModel(ctx, aiProvider.BaseURL, modelName, apiKey)
	if err != nil {
		slog.ErrorContext(ctx, "创建章节剧情解析模型失败", "err", err)
		return nil, err
	}

	toolSchema, err := ai_tools.GetChapterPlotAnalysisToolSchema()
	if err != nil {
		slog.ErrorContext(ctx, "获取章节剧情解析工具失败", "err", err)
		return nil, err
	}

	chatModelWithTools, err := chatModel.WithTools([]*schema.ToolInfo{toolSchema})
	if err != nil {
		slog.ErrorContext(ctx, "AI 模型添加章节剧情解析工具失败", "err", err)
		return nil, err
	}

	generate, err := chatModelWithTools.Generate(ctx, buildChapterPlotAnalysisMessages(title, content))
	if err != nil {
		slog.ErrorContext(ctx, "AI 分析章节剧情失败", "err", err)
		return nil, err
	}

	result, ok := ai_tools.ToolOutput2ChapterPlotAnalysisTool(generate.ContentBlocks)
	if !ok {
		return nil, common.ChapterPlotAnalysisNoResult
	}

	return result, nil
}

// enabledAIProviderDefaultModel 读取启用 Provider 的默认模型，空值表示自动剧情总结不可执行。
func enabledAIProviderDefaultModel(provider *data.AIProvider) (string, bool) {
	if provider == nil {
		return "", false
	}

	modelName := strings.TrimSpace(provider.DefaultModel)
	if modelName == "" {
		return "", false
	}

	return modelName, true
}

func chapterPlotAnalysisFromTool(payload ChapterSavedEvent, result *ai_tools.ChapterPlotAnalysisTool) (*data.ChapterPlotAnalysis, error) {
	if result == nil {
		return nil, common.ChapterPlotAnalysisNoResult
	}

	keyEvents, err := marshalJSONArray(result.KeyEvents)
	if err != nil {
		return nil, err
	}
	charactersInvolved, err := marshalJSONArray(result.CharactersInvolved)
	if err != nil {
		return nil, err
	}
	relationshipChanges, err := marshalJSONArray(result.RelationshipChanges)
	if err != nil {
		return nil, err
	}
	eventAnalysis, err := marshalJSONArray(result.EventAnalysis)
	if err != nil {
		return nil, err
	}
	foreshadowing, err := marshalJSONArray(result.Foreshadowing)
	if err != nil {
		return nil, err
	}
	unresolvedThreads, err := marshalJSONArray(result.UnresolvedThreads)
	if err != nil {
		return nil, err
	}

	return &data.ChapterPlotAnalysis{
		NovelID:             payload.NovelID,
		ChapterID:           payload.ChapterID,
		Summary:             result.Summary,
		KeyEvents:           keyEvents,
		CharactersInvolved:  charactersInvolved,
		RelationshipChanges: relationshipChanges,
		EventAnalysis:       eventAnalysis,
		Foreshadowing:       foreshadowing,
		UnresolvedThreads:   unresolvedThreads,
	}, nil
}

// marshalJSONArray 把 tool 的切片字段转换成 jsonb 数组，nil 统一保存为空数组。
func marshalJSONArray[T any](values []T) (datatypes.JSON, error) {
	if values == nil {
		values = []T{}
	}

	raw, err := sonic.Marshal(values)
	if err != nil {
		slog.Error("序列化章节剧情总结数组字段失败", "err", err)
		return nil, err
	}

	return datatypes.JSON(raw), nil
}

// buildChapterPlotAnalysisMessages 构造独立章节剧情解析 AI 的输入，不复用正文优化多层 Agent。
func buildChapterPlotAnalysisMessages(title, content string) []*schema.AgenticMessage {
	return []*schema.AgenticMessage{
		schema.SystemAgenticMessage(`你是专业的中文小说剧情分析师。你的任务是阅读用户提供的单章标题和正文，提取本章剧情概述、关键事件、主要角色、人物关系变化、事件分析、伏笔追踪和未解决线索。

要求：
1. 只分析本章正文中已经出现或被明确暗示的信息，不要编造不存在的剧情。
2. 关键事件需要按本章发生顺序输出。
3. 主要角色只包含本章出现、行动、被重点提及或推动剧情的人物。
4. 人物关系变化只记录本章中新建立、恶化、缓和、暴露或发生关键转折的关系；没有变化时返回空数组。
5. 事件分析需要说明关键事件对剧情推进、人物选择或冲突升级的作用。
6. 伏笔追踪和未解决线索可以为空数组，但不能省略字段。
7. 必须调用 chapter_plot_analysis_tool 工具返回结构化结果，不要直接输出自然语言。`),
		schema.UserAgenticMessage("章节标题：\n" + title + "\n\n章节正文：\n" + content),
	}
}
