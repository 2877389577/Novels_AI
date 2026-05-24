package ai_tools

import (
	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

const novelContentOptimizeToolName = "novel_content_optimize_tool"

// NovelContentOptimizeTool 定义小说正文优化接口要求 AI 通过 tool 返回的结构化结果。
type NovelContentOptimizeTool struct {
	OptimizedContent string `json:"optimizedContent" jsonschema:"required" jsonschema_description:"优化后的小说正文。"`
	Approved         bool   `json:"approved" jsonschema:"required" jsonschema_description:"是否同意并完成本次优化。只做文笔优化时返回 true；用户要求改剧情、改事实或超出文笔优化范围时返回 false。"`
	RejectReason     string `json:"rejectReason,omitempty" jsonschema_description:"拒绝优化的原因。approved 为 false 时必须填写；approved 为 true 时返回空字符串。"`
}

// GetNovelContentOptimizeToolSchema 返回小说正文优化 tool schema，用于约束模型输出为前端可直接消费的 JSON 结构。
func GetNovelContentOptimizeToolSchema() (*schema.ToolInfo, error) {
	return utils.GoStruct2ToolInfo[NovelContentOptimizeTool](
		novelContentOptimizeToolName,
		"小说正文优化结果格式化工具。模型必须使用该工具返回优化后的正文、是否同意优化以及拒绝理由。",
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
