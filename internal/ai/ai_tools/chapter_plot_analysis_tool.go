package ai_tools

import (
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

const chapterPlotAnalysisToolName = "chapter_plot_analysis_tool"

// ChapterRelationshipChangeTool 描述本章中两个角色关系的变化，字段名保持和前端剧情解析 JSON 一致。
type ChapterRelationshipChangeTool struct {
	Source string `json:"source" jsonschema:"required" jsonschema_description:"关系变化的来源角色名称。"`
	Target string `json:"target" jsonschema:"required" jsonschema_description:"关系变化的目标角色名称。"`
	Change string `json:"change" jsonschema:"required" jsonschema_description:"本章中两人关系发生的具体变化。"`
}

// ChapterPlotAnalysisTool 定义章节剧情解析 AI 必须返回的结构化结果。
type ChapterPlotAnalysisTool struct {
	Summary             string                          `json:"summary" jsonschema:"required" jsonschema_description:"本章剧情概述，用一到两句话说明本章核心发展。"`
	KeyEvents           []string                        `json:"key_events" jsonschema:"required" jsonschema_description:"本章关键事件列表，按发生顺序输出。"`
	CharactersInvolved  []string                        `json:"characters_involved" jsonschema:"required" jsonschema_description:"本章主要出现或被重点提及的角色名称列表。"`
	RelationshipChanges []ChapterRelationshipChangeTool `json:"relationship_changes" jsonschema:"required" jsonschema_description:"本章人物关系变化列表，没有变化时返回空数组。"`
	EventAnalysis       []string                        `json:"event_analysis" jsonschema:"required" jsonschema_description:"对本章关键事件的作用、影响和剧情意义的分析。"`
	Foreshadowing       []string                        `json:"foreshadowing" jsonschema:"required" jsonschema_description:"本章埋下或推进的伏笔列表，没有时返回空数组。"`
	UnresolvedThreads   []string                        `json:"unresolved_threads" jsonschema:"required" jsonschema_description:"本章留下的未解决线索或悬念列表，没有时返回空数组。"`
}

// GetChapterPlotAnalysisToolSchema 返回章节剧情解析 tool schema，用于约束模型输出为可持久化的 JSON 结构。
func GetChapterPlotAnalysisToolSchema() (*schema.ToolInfo, error) {
	return utils.GoStruct2ToolInfo[ChapterPlotAnalysisTool](
		chapterPlotAnalysisToolName,
		"章节剧情解析结果格式化工具。模型必须使用该工具返回剧情概述、关键事件、主要角色、人物关系变化、事件分析、伏笔和未解决线索。",
	)
}

// ParseChapterPlotAnalysisToolOutput 从模型返回的 content blocks 中解析章节剧情解析 tool 调用参数。
// 返回 error 而不是静默 false，方便异步任务在日志中记录模型未按约定返回时的具体原因。
func ParseChapterPlotAnalysisToolOutput(blocks []*schema.ContentBlock) (*ChapterPlotAnalysisTool, error) {
	for _, block := range blocks {
		if block.Type != schema.ContentBlockTypeFunctionToolCall || block.FunctionToolCall == nil {
			continue
		}
		if block.FunctionToolCall.Name != chapterPlotAnalysisToolName {
			continue
		}

		var result ChapterPlotAnalysisTool
		if err := sonic.Unmarshal([]byte(block.FunctionToolCall.Arguments), &result); err != nil {
			return nil, fmt.Errorf("解析 %s 参数失败: %w", chapterPlotAnalysisToolName, err)
		}

		if strings.TrimSpace(result.Summary) == "" {
			return nil, fmt.Errorf("%s 返回的 summary 为空", chapterPlotAnalysisToolName)
		}

		return &result, nil
	}

	return nil, fmt.Errorf("AI 响应未调用 %s", chapterPlotAnalysisToolName)
}

// ToolOutput2ChapterPlotAnalysisTool 保留旧的 bool 返回形式，便于已有调用方继续按“是否解析成功”判断。
func ToolOutput2ChapterPlotAnalysisTool(blocks []*schema.ContentBlock) (*ChapterPlotAnalysisTool, bool) {
	result, err := ParseChapterPlotAnalysisToolOutput(blocks)
	return result, err == nil
}
