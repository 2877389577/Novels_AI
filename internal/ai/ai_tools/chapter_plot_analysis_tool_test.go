package ai_tools

import (
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestToolOutput2ChapterPlotAnalysisTool(t *testing.T) {
	// 章节剧情解析必须通过专用 tool 返回，确保后续可以直接持久化结构化 JSON。
	blocks := []*schema.ContentBlock{
		{
			Type: schema.ContentBlockTypeFunctionToolCall,
			FunctionToolCall: &schema.FunctionToolCall{
				Name:      chapterPlotAnalysisToolName,
				Arguments: `{"summary":"陆沉救下宋玉。","key_events":["陆沉救下宋玉"],"characters_involved":["陆沉","宋玉"],"relationship_changes":[{"source":"陆沉","target":"宋玉","change":"初次建立联系"}],"event_analysis":["本章完成主角相遇"],"foreshadowing":["玉佩身份伏笔"],"unresolved_threads":["追兵来源"]}`,
			},
		},
	}

	result, ok := ToolOutput2ChapterPlotAnalysisTool(blocks)
	if !ok {
		t.Fatal("expected chapter plot analysis tool output to be parsed")
	}
	if result.Summary != "陆沉救下宋玉。" {
		t.Fatalf("unexpected summary: %q", result.Summary)
	}
	if len(result.KeyEvents) != 1 || result.KeyEvents[0] != "陆沉救下宋玉" {
		t.Fatalf("unexpected key events: %#v", result.KeyEvents)
	}
	if len(result.RelationshipChanges) != 1 || result.RelationshipChanges[0].Change != "初次建立联系" {
		t.Fatalf("unexpected relationship changes: %#v", result.RelationshipChanges)
	}
}

func TestToolOutput2ChapterPlotAnalysisToolIgnoresOtherTools(t *testing.T) {
	// 解析器只接受章节剧情解析 tool，避免误读其他 AI tool 的输出。
	blocks := []*schema.ContentBlock{
		{
			Type: schema.ContentBlockTypeFunctionToolCall,
			FunctionToolCall: &schema.FunctionToolCall{
				Name:      "other_tool",
				Arguments: `{"summary":"无效"}`,
			},
		},
	}

	if _, ok := ToolOutput2ChapterPlotAnalysisTool(blocks); ok {
		t.Fatal("expected unrelated tool output to be ignored")
	}
}
