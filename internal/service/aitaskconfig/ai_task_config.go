package aitaskconfig

import (
	"context"
	"net/http"
	"strings"

	aitaskconfigbiz "Novels_AI/backend/internal/biz/aitaskconfig"
	"Novels_AI/backend/internal/data/dto"
	"Novels_AI/backend/internal/pkg/common"

	"github.com/gin-gonic/gin"
)

type AITaskConfigService struct {
	useCase AITaskConfigUseCase
}

// AITaskConfigUseCase 描述 service 层依赖的主动执行 AI 任务配置能力。
type AITaskConfigUseCase interface {
	List(ctx context.Context) ([]*aitaskconfigbiz.AITaskConfig, error)
	Update(ctx context.Context, params dto.UpdateAITaskConfigRequest) (*aitaskconfigbiz.AITaskConfig, error)
}

type aiTaskConfigResponse struct {
	// 配置 ID
	ID int64 `json:"id"`
	// 任务编码
	TaskCode string `json:"taskCode"`
	// 任务名称
	TaskName string `json:"taskName"`
	// 任务说明
	Description string `json:"description"`
	// 是否启用
	IsEnabled bool `json:"isEnabled"`
	// 创建时间
	CreatedAt string `json:"createdAt"`
	// 更新时间
	UpdatedAt string `json:"updatedAt"`
}

type aiTaskConfigListResponse struct {
	Items []aiTaskConfigResponse `json:"items"`
}

func NewAITaskConfigService(useCase AITaskConfigUseCase) *AITaskConfigService {
	return &AITaskConfigService{useCase: useCase}
}

// List 查询主动执行 AI 任务配置
// @Summary 查询主动执行 AI 任务配置
// @Description 查询所有后台自动触发的 AI 任务配置，后台可据此展示开关状态
// @Tags ai-task-config
// @Produce json
// @Success 200 {object} common.Response{data=aiTaskConfigListResponse}
// @Failure 401 {object} common.Response "未登录"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /ai-task-configs [get]
func (service *AITaskConfigService) List(c *gin.Context) {
	configs, err := service.useCase.List(c.Request.Context())
	if err != nil {
		_ = c.Error(err)
		return
	}

	items := make([]aiTaskConfigResponse, 0, len(configs))
	for _, item := range configs {
		items = append(items, toAITaskConfigResponse(item))
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: aiTaskConfigListResponse{Items: items},
	})
}

// Update 更新主动执行 AI 任务开关
// @Summary 更新主动执行 AI 任务开关
// @Description 按任务编码更新后台自动触发 AI 任务的启用状态
// @Tags ai-task-config
// @Accept json
// @Produce json
// @Param taskCode path string true "任务编码，例如 chapter_plot_analysis"
// @Param config body dto.UpdateAITaskConfigRequest true "任务开关配置"
// @Success 200 {object} common.Response{data=aiTaskConfigResponse}
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /ai-task-configs/{taskCode} [put]
func (service *AITaskConfigService) Update(c *gin.Context) {
	taskCode := strings.TrimSpace(c.Param("taskCode"))
	if taskCode == "" {
		_ = c.Error(common.InvalidRequest)
		return
	}

	var request dto.UpdateAITaskConfigRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		_ = c.Error(common.InvalidRequestWithValidationMessage(request, err))
		return
	}
	request.TaskCode = taskCode

	config, err := service.useCase.Update(c.Request.Context(), request)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: toAITaskConfigResponse(config),
	})
}

func toAITaskConfigResponse(config *aitaskconfigbiz.AITaskConfig) aiTaskConfigResponse {
	return aiTaskConfigResponse{
		ID:          config.ID,
		TaskCode:    config.TaskCode,
		TaskName:    config.TaskName,
		Description: config.Description,
		IsEnabled:   config.IsEnabled,
		CreatedAt:   config.CreatedAt.Format("2006-01-02T15:04:05"),
		UpdatedAt:   config.UpdatedAt.Format("2006-01-02T15:04:05"),
	}
}
