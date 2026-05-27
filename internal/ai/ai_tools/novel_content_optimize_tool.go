package ai_tools

import (
	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

const (
	novelContentOptimizeToolName          = "novel_content_optimize_tool"
	novelContentOptimizationRouteToolName = "novel_content_optimization_route_tool"
)

// NovelContentOptimizeTool 定义小说正文优化子 Agent 必须返回的结构化结果。
type NovelContentOptimizeTool struct {
	OptimizedContent string `json:"optimizedContent" jsonschema:"required" jsonschema_description:"优化或扩写后的小说正文。"`
	Approved         bool   `json:"approved" jsonschema:"required" jsonschema_description:"是否同意并完成本次处理。任务在当前子 Agent 能力范围内时返回 true；用户要求改剧情、改事实或超出当前任务范围时返回 false。"`
	RejectReason     string `json:"rejectReason,omitempty" jsonschema_description:"拒绝优化的原因。approved 为 false 时必须填写；approved 为 true 时返回空字符串。"`
}

// NovelContentOptimizationRouteTool 定义顶层 Agent 对正文优化任务的分流结果。
type NovelContentOptimizationRouteTool struct {
	AgentName string `json:"agentName" jsonschema:"required" jsonschema_description:"需要调用的子 Agent 名称，只能是 polish 或 expand。"`
	Reason    string `json:"reason,omitempty" jsonschema_description:"选择该子 Agent 的简短原因，便于日志排查；没有必要时可为空。"`
}

// GetNovelContentOptimizeToolSchema 返回小说正文优化 tool schema，用于约束模型输出为前端可直接消费的 JSON 结构。
func GetNovelContentOptimizeToolSchema() (*schema.ToolInfo, error) {
	return utils.GoStruct2ToolInfo[NovelContentOptimizeTool](
		novelContentOptimizeToolName,
		"小说正文优化结果格式化工具。模型必须使用该工具返回处理后的正文、是否同意处理以及拒绝理由。",
	)
}

// GetNovelContentOptimizationRouteToolSchema 返回顶层 Agent 的任务分流 tool schema。
func GetNovelContentOptimizationRouteToolSchema() (*schema.ToolInfo, error) {
	return utils.GoStruct2ToolInfo[NovelContentOptimizationRouteTool](
		novelContentOptimizationRouteToolName,
		"小说正文优化任务分流工具。顶层 Agent 必须使用该工具选择 polish 或 expand 子 Agent，不得直接改写正文。",
	)
}

// ToolOutput2NovelContentOptimizeTool 从模型返回的 content blocks 中解析小说正文优化 tool 调用参数。
func ToolOutput2NovelContentOptimizeTool(blocks []*schema.ContentBlock) (*NovelContentOptimizeTool, bool) {
	for _, block := range blocks {
		if block.Type != schema.ContentBlockTypeFunctionToolCall || block.FunctionToolCall == nil {
			continue
		}
		if block.FunctionToolCall.Name != novelContentOptimizeToolName {
			continue
		}

		var result NovelContentOptimizeTool
		if err := sonic.Unmarshal([]byte(block.FunctionToolCall.Arguments), &result); err != nil {
			continue
		}

		return &result, true
	}

	return nil, false
}

// ToolOutput2NovelContentOptimizationRouteTool 从模型返回的 content blocks 中解析顶层 Agent 的分流结果。
func ToolOutput2NovelContentOptimizationRouteTool(blocks []*schema.ContentBlock) (*NovelContentOptimizationRouteTool, bool) {
	for _, block := range blocks {
		if block.Type != schema.ContentBlockTypeFunctionToolCall || block.FunctionToolCall == nil {
			continue
		}
		if block.FunctionToolCall.Name != novelContentOptimizationRouteToolName {
			continue
		}

		var result NovelContentOptimizationRouteTool
		if err := sonic.Unmarshal([]byte(block.FunctionToolCall.Arguments), &result); err != nil {
			continue
		}

		return &result, true
	}

	return nil, false
}
