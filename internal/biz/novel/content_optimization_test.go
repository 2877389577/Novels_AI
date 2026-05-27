package novel

import (
	"Novels_AI/backend/internal/data/dto"
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestNewContentOptimizationAgentRegistryRegistersDefaultAgents(t *testing.T) {
	// 默认注册表是后续扩展 Agent 的入口，当前必须包含文笔润色和扩写两个子 Agent。
	registry := newContentOptimizationAgentRegistry()

	if _, ok := registry.Get(contentOptimizationAgentPolish); !ok {
		t.Fatalf("expected %s agent to be registered", contentOptimizationAgentPolish)
	}
	if _, ok := registry.Get(contentOptimizationAgentExpand); !ok {
		t.Fatalf("expected %s agent to be registered", contentOptimizationAgentExpand)
	}
}

func TestRouteContentOptimizationAgentDefaultsToPolishWhenDirectionEmpty(t *testing.T) {
	// 空优化方向不能触发额外模型分流，应直接走文笔润色 Agent，降低默认优化链路的成本。
	uc := &ContentOptimizationUseCase{
		agentRegistry: newContentOptimizationAgentRegistry(),
	}

	agentName, err := uc.routeContentOptimizationAgent(context.Background(), nil, dto.OptimizeNovelContentRequest{
		OptimizeDirection: "   ",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agentName != contentOptimizationAgentPolish {
		t.Fatalf("expected default agent %q, got %q", contentOptimizationAgentPolish, agentName)
	}
}

func TestBuildNovelContentOptimizationAgentMessagesUsesPolishDefaultDirection(t *testing.T) {
	// 文笔润色 Agent 在用户未填写方向时，需要收到明确的默认润色方向。
	messages := buildNovelContentOptimizationAgentMessages("system prompt", dto.OptimizeNovelContentRequest{
		SelectedContent: "夜色落下。",
	}, novelContentOptimizationDefaultPolishDirection)

	if got := agenticMessageText(messages[1]); !strings.Contains(got, novelContentOptimizationDefaultPolishDirection) {
		t.Fatalf("expected default polish direction in message, got %q", got)
	}
	if got := agenticMessageText(messages[2]); got != "小说原文：\n夜色落下。" {
		t.Fatalf("unexpected selected content message: %q", got)
	}
}

func TestBuildNovelContentOptimizationAgentMessagesUsesExpandPromptAndDirection(t *testing.T) {
	// 扩写 Agent 必须使用扩写提示词，并保留用户输入的扩写方向传给模型。
	systemPrompt, err := loadNovelContentOptimizationAgentSystemPrompt(novelContentExpandPrompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	messages := buildNovelContentOptimizationAgentMessages(systemPrompt, dto.OptimizeNovelContentRequest{
		SelectedContent:   "门外传来脚步声。",
		OptimizeDirection: "扩写这段动作和氛围",
	}, novelContentOptimizationDefaultExpandDirection)

	if got := agenticMessageText(messages[0]); !strings.Contains(got, "novel_content_optimize_tool") {
		t.Fatalf("expected expand system prompt, got %q", got)
	}
	if got := agenticMessageText(messages[1]); got != "优化方向：\n扩写这段动作和氛围" {
		t.Fatalf("unexpected direction message: %q", got)
	}
}

func agenticMessageText(message *schema.AgenticMessage) string {
	// 当前业务只构造纯文本 AgenticMessage，测试里集中读取首个文本 block 以减少重复断言代码。
	if len(message.ContentBlocks) == 0 || message.ContentBlocks[0].UserInputText == nil {
		return ""
	}

	return message.ContentBlocks[0].UserInputText.Text
}
