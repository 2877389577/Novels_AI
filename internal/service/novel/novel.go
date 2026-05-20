package novel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	novelbiz "Novels_AI/backend/internal/biz/novel"
	"Novels_AI/backend/internal/pkg/common"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type NovelService struct {
	useCase NovelUseCase
}

type NovelUseCase interface {
	Create(ctx context.Context, params novelbiz.CreateNovelParams) (*novelbiz.Novel, error)
	List(ctx context.Context, page, pageSize int) (*novelbiz.ListNovelResult, error)
	Get(ctx context.Context, id uint) (*novelbiz.Novel, error)
	Update(ctx context.Context, id uint, params novelbiz.UpdateNovelParams) (*novelbiz.Novel, error)
	Delete(ctx context.Context, id uint) error
}

type createNovelRequest struct {
	// 书名，必填
	Title string `json:"title" binding:"required"`
	// 小说简介
	Intro string `json:"intro"`
	// 小说作者
	AuthorName string `json:"authorName"`
	// 小说封面 URL
	CoverURL string `json:"coverUrl"`
	// 小说元数据（标签信息）
	Metadata rawJSONField `json:"metadata" swaggertype:"object"`
}

type updateNovelRequest struct {
	// 小说 ID，必填
	ID uint `json:"id" binding:"required"`
	// 书名
	Title *string `json:"title"`
	// 小说简介
	Intro *string `json:"intro"`
	// 小说作者
	AuthorName *string `json:"authorName"`
	// 小说封面 URL
	CoverURL *string `json:"coverUrl"`
	// 小说元数据（标签信息）
	Metadata rawJSONField `json:"metadata" swaggertype:"object"`
}

type rawJSONField struct {
	Set   bool
	Value datatypes.JSON
}

type novelResponse struct {
	// 小说 ID
	ID uint `json:"id"`
	// 书名
	Title string `json:"title"`
	// 小说简介
	Intro string `json:"intro"`
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

func NewNovelService(useCase NovelUseCase) *NovelService {
	return &NovelService{useCase: useCase}
}

// UnmarshalJSON 记录 metadata 字段是否出现，便于 PUT 区分“未传”和“传 null”。
func (r *rawJSONField) UnmarshalJSON(data []byte) error {
	r.Set = true
	if bytes.Equal(data, []byte("null")) {
		r.Value = datatypes.JSON([]byte("{}"))
		return nil
	}

	r.Value = datatypes.JSON(append([]byte(nil), data...))
	return nil
}

// Create 新增小说
// @Summary 新增小说
// @Description 创建小说，仅书名必填
// @Tags novel
// @Accept json
// @Produce json
// @Param novel body createNovelRequest true "小说信息"
// @Success 200 {object} common.Response{data=novelResponse}
// @Failure 400 {object} common.Response "请求参数错误"
// @Failure 401 {object} common.Response "未登录"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /novels [post]
func (service *NovelService) Create(c *gin.Context) {
	var request createNovelRequest
	err := c.ShouldBindJSON(&request)
	if err != nil {
		_ = c.Error(common.InvalidRequest)
		return
	}

	novel, err := service.useCase.Create(c.Request.Context(), novelbiz.CreateNovelParams{
		Title:      request.Title,
		Intro:      request.Intro,
		AuthorName: request.AuthorName,
		CoverURL:   request.CoverURL,
		Metadata:   request.Metadata.Value,
	})
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
// @Failure 400 {object} common.Response "请求参数错误"
// @Failure 401 {object} common.Response "未登录"
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
// @Failure 400 {object} common.Response "请求参数错误"
// @Failure 401 {object} common.Response "未登录"
// @Failure 404 {object} common.Response "小说不存在"
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
// @Description 按 ID 局部更新小说
// @Tags novel
// @Accept json
// @Produce json
// @Param novel body updateNovelRequest true "小说信息"
// @Success 200 {object} common.Response{data=novelResponse}
// @Failure 400 {object} common.Response "请求参数错误"
// @Failure 401 {object} common.Response "未登录"
// @Failure 404 {object} common.Response "小说不存在"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /novels/update [put]
func (service *NovelService) Update(c *gin.Context) {
	var request updateNovelRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		_ = c.Error(common.InvalidRequest)
		return
	}

	params := novelbiz.UpdateNovelParams{
		Title:      request.Title,
		Intro:      request.Intro,
		AuthorName: request.AuthorName,
		CoverURL:   request.CoverURL,
	}
	if request.Metadata.Set {
		params.Metadata = new(request.Metadata.Value)
	}

	novel, err := service.useCase.Update(c.Request.Context(), request.ID, params)
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

// Delete 删除小说
// @Summary 删除小说
// @Description 按 ID 软删除小说
// @Tags novel
// @Produce json
// @Param id path int true "小说 ID"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response "请求参数错误"
// @Failure 401 {object} common.Response "未登录"
// @Failure 404 {object} common.Response "小说不存在"
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
		ID:         novel.ID,
		Title:      novel.Title,
		Intro:      novel.Intro,
		AuthorName: novel.AuthorName,
		CoverURL:   novel.CoverURL,
		WordCount:  novel.WordCount,
		Metadata:   metadata,
		CreatedAt:  novel.CreatedAt.Format("2006-01-02T15:04:05"),
	}
}
