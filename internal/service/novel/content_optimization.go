package novel

import (
	"Novels_AI/backend/internal/ai/ai_tools"
	"Novels_AI/backend/internal/data/dto"
	"Novels_AI/backend/internal/pkg/common"
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// ContentOptimizationService 负责把小说正文优化 HTTP 请求转换成业务用例参数。
type ContentOptimizationService struct {
	useCase ContentOptimizationUseCase
}

// ContentOptimizationUseCase 描述正文优化 service 依赖的业务能力，便于测试时替换。
type ContentOptimizationUseCase interface {
	OptimizeNovelContent(ctx context.Context, params dto.OptimizeNovelContentRequest) (*ai_tools.NovelContentOptimizeTool, error)
}

// NewContentOptimizationService 创建小说正文优化 HTTP 服务适配器。
func NewContentOptimizationService(useCase ContentOptimizationUseCase) *ContentOptimizationService {
	return &ContentOptimizationService{useCase: useCase}
}

// Optimize 优化小说正文
// @Summary 优化小说正文
// @Description 对用户在编辑器中选中的小说段落进行 AI 优化。请求体只包含用户选中的原文和优化方向；优化方向可为空，表示默认做通用文笔润色。优化方向不为空时，顶层 Agent 会先分析用户意图，再自动选择文笔润色或扩写子 Agent。该接口只返回优化结果，不会修改章节正文或写入数据库。AI 必须通过 novel_content_optimize_tool 返回结构化字段：optimizedContent 为优化或扩写后的正文，approved 表示是否同意处理，rejectReason 表示拒绝原因。
// @Tags novel-content
// @Accept json
// @Produce json
// @Param id path int true "小说 ID，必须是正整数，用于确认本次优化归属的小说存在"
// @Param modelName query string true "大模型名称，例如 doubao-xxx；后端会使用当前启用 AI 提供商的 BaseURL 和 API Key 初始化该模型"
// @Param content body dto.OptimizeNovelContentRequest true "正文优化请求参数。selectedContent 为用户选中的小说原文段落，必填；optimizeDirection 为用户输入的优化方向，可为空；为空时默认文笔润色，包含扩写意图时自动走扩写"
// @Success 200 {object} common.Response{data=ai_tools.NovelContentOptimizeTool} "code = 0 表示请求成功；data.optimizedContent 为优化或扩写后的正文；data.approved 为是否同意处理；data.rejectReason 为拒绝理由"
// @Failure 400 {object} common.Response "请求参数错误，例如小说 ID 不是正整数、modelName 为空或 selectedContent 为空"
// @Failure 500 {object} common.SystemError "系统错误，例如读取提示词失败、API Key 解密失败或上游模型调用失败"
// @Router /novels/{id}/content/optimize [post]
func (service *ContentOptimizationService) Optimize(c *gin.Context) {
	novelID, ok := parseContentOptimizationNovelID(c)
	if !ok {
		return
	}

	modelName := strings.TrimSpace(c.Query("modelName"))
	if modelName == "" {
		_ = c.Error(common.NewHTTPError(http.StatusBadRequest, common.InvalidRequest.Code, "大模型名称不能为空"))
		return
	}

	var request dto.OptimizeNovelContentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		_ = c.Error(common.InvalidRequestWithValidationMessage(request, err))
		return
	}
	request.NovelID = novelID
	request.ModelName = modelName

	result, err := service.useCase.OptimizeNovelContent(c.Request.Context(), request)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: result,
	})
}

func parseContentOptimizationNovelID(c *gin.Context) (int64, bool) {
	novelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || novelID <= 0 {
		_ = c.Error(common.InvalidRequest)
		return 0, false
	}

	return novelID, true
}
