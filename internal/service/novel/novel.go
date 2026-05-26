package novel

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	novelbiz "Novels_AI/backend/internal/biz/novel"
	"Novels_AI/backend/internal/data/dto"
	"Novels_AI/backend/internal/pkg/common"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const maxNovelPageSize = 100

type NovelService struct {
	useCase NovelUseCase
}

type NovelUseCase interface {
	Create(ctx context.Context, params dto.CreateNovelRequest) (*novelbiz.Novel, error)
	List(ctx context.Context, page, pageSize int) (*novelbiz.ListNovelResult, error)
	Get(ctx context.Context, id uint) (*novelbiz.Novel, error)
	Update(ctx context.Context, params dto.UpdateNovelRequest) (*novelbiz.Novel, error)
	SaveOutline(ctx context.Context, params dto.SaveNovelOutlineRequest) (string, error)
	GetOutline(ctx context.Context, novelID uint) (string, error)
	Delete(ctx context.Context, id uint) error
}

type novelResponse struct {
	// 小说 ID
	ID uint `json:"id"`
	// 书名
	Title string `json:"title"`
	// 小说简介
	Intro string `json:"intro"`
	// 小说大纲
	NovelOutline string `json:"novelOutline"`
	// 小说作者
	AuthorName string `json:"authorName"`
	// 小说封面 URL
	CoverURL string `json:"coverUrl"`
	// 小说字数
	WordCount int64 `json:"wordCount"`
	// 小说元数据（标签信息）
	Metadata json.RawMessage `json:"metadata" swaggertype:"object"`
	// 创建时间
	CreatedAt string `json:"createdAt"`
}

type novelListResponse struct {
	Items    []novelResponse `json:"items"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"pageSize"`
}

type novelOutlineResponse struct {
	// 小说大纲内容；为空字符串表示当前没有大纲。
	NovelOutline string `json:"novelOutline"`
}

func NewNovelService(useCase NovelUseCase) *NovelService {
	return &NovelService{useCase: useCase}
}

// Create 新增小说
// @Summary 新增小说
// @Description 创建小说，仅书名必填
// @Tags novel
// @Accept json
// @Produce json
// @Param novel body dto.CreateNovelRequest true "小说信息"
// @Success 200 {object} common.Response{data=novelResponse}
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /novels [post]
func (service *NovelService) Create(c *gin.Context) {
	var request dto.CreateNovelRequest
	err := c.ShouldBindJSON(&request)
	if err != nil {
		_ = c.Error(common.InvalidRequestWithValidationMessage(request, err))
		return
	}

	novel, err := service.useCase.Create(c.Request.Context(), request)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: toNovelResponse(novel),
	})
}

// List 查询小说列表
// @Summary 查询小说列表
// @Description 分页查询小说列表
// @Tags novel
// @Produce json
// @Param page query int false "页码，默认 1"
// @Param pageSize query int false "每页数量，默认 10，最大 100"
// @Success 200 {object} common.Response{data=novelListResponse}
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /novels [get]
func (service *NovelService) List(c *gin.Context) {
	page, pageSize, ok := bindPagination(c)
	if !ok {
		return
	}

	result, err := service.useCase.List(c.Request.Context(), page, pageSize)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "查询小说列表失败", "err", err)
		_ = c.Error(err)
		return
	}

	items := make([]novelResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toNovelResponse(new(item)))
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: &novelListResponse{
			Items:    items,
			Total:    result.Total,
			Page:     result.Page,
			PageSize: result.PageSize,
		},
	})
}

// Get 查询小说详情
// @Summary 查询小说详情
// @Description 按 ID 查询单本小说
// @Tags novel
// @Produce json
// @Param id path int true "小说 ID"
// @Success 200 {object} common.Response{data=novelResponse}
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /novels/{id} [get]
func (service *NovelService) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		_ = c.Error(common.InvalidRequest)
		return
	}

	novel, err := service.useCase.Get(c.Request.Context(), uint(id))
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: toNovelResponse(novel),
	})
}

// Update 更新小说
// @Summary 更新小说
// @Description 按 ID 全量更新小说
// @Tags novel
// @Accept json
// @Produce json
// @Param novel body dto.UpdateNovelRequest true "小说信息"
// @Success 200 {object} common.Response{data=novelResponse}
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /novels/update [put]
func (service *NovelService) Update(c *gin.Context) {
	var request dto.UpdateNovelRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		_ = c.Error(common.InvalidRequestWithValidationMessage(request, err))
		return
	}

	novel, err := service.useCase.Update(c.Request.Context(), request)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "更新小说失败", "err", err)
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: toNovelResponse(novel),
	})
}

// SaveOutline 保存小说大纲
// @Summary 保存小说大纲
// @Description 给指定小说保存大纲内容。该接口同时承担新增、修改和删除大纲的能力：novelOutline 传入非空字符串表示保存或覆盖大纲，传入空字符串表示清空大纲；接口只更新小说大纲字段，不会修改小说标题、简介、作者、封面、字数和元数据。
// @Tags novel
// @Accept json
// @Produce json
// @Param id path int true "小说 ID，必须是正整数"
// @Param outline body dto.SaveNovelOutlineRequest true "小说大纲保存参数。novelOutline 为大纲内容，可为空；为空字符串表示清空大纲"
// @Success 200 {object} common.Response{data=novelOutlineResponse} "code = 0 表示保存成功；data.novelOutline 为保存后的大纲内容"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /novels/{id}/outline [post]
func (service *NovelService) SaveOutline(c *gin.Context) {
	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || novelID == 0 {
		_ = c.Error(common.InvalidRequest)
		return
	}

	var request dto.SaveNovelOutlineRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		_ = c.Error(common.InvalidRequestWithValidationMessage(request, err))
		return
	}
	request.NovelID = int64(novelID)

	outline, err := service.useCase.SaveOutline(c.Request.Context(), request)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "保存小说大纲失败", "err", err)
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: novelOutlineResponse{NovelOutline: outline},
	})
}

// GetOutline 查询小说大纲
// @Summary 查询小说大纲
// @Description 查询指定小说的大纲内容。该接口只返回 novelOutline 字段，不返回小说标题、简介、作者、封面、字数和元数据等详情字段。
// @Tags novel
// @Produce json
// @Param id path int true "小说 ID，必须是正整数"
// @Success 200 {object} common.Response{data=novelOutlineResponse} "code = 0 表示查询成功；data.novelOutline 为大纲内容，空字符串表示当前没有大纲"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /novels/{id}/outline [get]
func (service *NovelService) GetOutline(c *gin.Context) {
	novelID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || novelID == 0 {
		_ = c.Error(common.InvalidRequest)
		return
	}

	outline, err := service.useCase.GetOutline(c.Request.Context(), uint(novelID))
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "查询小说大纲失败", "err", err)
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: novelOutlineResponse{NovelOutline: outline},
	})
}

// Delete 删除小说
// @Summary 删除小说
// @Description 按 ID 软删除小说
// @Tags novel
// @Produce json
// @Param id path int true "小说 ID"
// @Success 200 {object} common.Response
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /novels/{id} [delete]
func (service *NovelService) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		_ = c.Error(common.InvalidRequest)
		return
	}

	if err := service.useCase.Delete(c.Request.Context(), uint(id)); err != nil {
		slog.ErrorContext(c.Request.Context(), "删除小说失败", "err", err)
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
	})
}

func bindPagination(c *gin.Context) (int, int, bool) {
	page, ok := bindPositiveQuery(c, "page", 1)
	if !ok {
		return 0, 0, false
	}

	pageSize, ok := bindPositiveQuery(c, "pageSize", 10)
	if !ok {
		return 0, 0, false
	}
	if pageSize > maxNovelPageSize {
		pageSize = maxNovelPageSize
	}

	return page, pageSize, true
}

func bindPositiveQuery(c *gin.Context, key string, defaultValue int) (int, bool) {
	rawValue := c.Query(key)
	if rawValue == "" {
		return defaultValue, true
	}

	value, err := strconv.Atoi(rawValue)
	if err != nil || value <= 0 {
		c.JSON(http.StatusBadRequest, &common.Response{
			Code: http.StatusBadRequest,
			Msg:  "请求参数错误",
		})
		return 0, false
	}

	return value, true
}

func toNovelResponse(novel *novelbiz.Novel) novelResponse {
	metadata := json.RawMessage(novel.Metadata)
	if len(metadata) == 0 {
		metadata = json.RawMessage(`{}`)
	}

	return novelResponse{
		ID:           novel.ID,
		Title:        novel.Title,
		Intro:        novel.Intro,
		NovelOutline: novel.NovelOutline,
		AuthorName:   novel.AuthorName,
		CoverURL:     novel.CoverURL,
		WordCount:    novel.WordCount,
		Metadata:     metadata,
		CreatedAt:    novel.CreatedAt.Format("2006-01-02T15:04:05"),
	}
}
