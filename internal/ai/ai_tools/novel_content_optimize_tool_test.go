package ai_tools

import (
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestToolOutput2NovelContentOptimizationRouteTool(t *testing.T) {
	// 顶层 Agent 必须通过 route tool 返回子 Agent 名称，这里验证标准 tool call 能被正确解析。
	blocks := []*schema.ContentBlock{
		{
			Type: schema.ContentBlockTypeFunctionToolCall,
			FunctionToolCall: &schema.FunctionToolCall{
				Name:      novelContentOptimizationRouteToolName,
				Arguments: `{"agentName":"expand","reason":"用户要求扩写场景"}`,
			},
		},
	}

	result, ok := ToolOutput2NovelContentOptimizationRouteTool(blocks)
	if !ok {
		t.Fatal("expected route tool output to be parsed")
	}
	if result.AgentName != "expand" {
		t.Fatalf("expected expand agent, got %q", result.AgentName)
	}
	if result.Reason != "用户要求扩写场景" {
		t.Fatalf("unexpected route reason: %q", result.Reason)
	}
}

func TestToolOutput2NovelContentOptimizationRouteToolIgnoresOtherTools(t *testing.T) {
	// 路由解析只接受 route tool，避免把正文优化结果 tool 误判成顶层分流结果。
	blocks := []*schema.ContentBlock{
		{
			Type: schema.ContentBlockTypeFunctionToolCall,
			FunctionToolCall: &schema.FunctionToolCall{
				Name:      novelContentOptimizeToolName,
				Arguments: `{"optimizedContent":"正文","approved":true}`,
			},
		},
	}

	if _, ok := ToolOutput2NovelContentOptimizationRouteTool(blocks); ok {
		t.Fatal("expected non-route tool output to be ignored")
	}
}
