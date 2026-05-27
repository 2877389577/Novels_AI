package novel

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	novelbiz "Novels_AI/backend/internal/biz/novel"
	"Novels_AI/backend/internal/data/dto"
	"Novels_AI/backend/internal/pkg/common"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

type ChapterService struct {
	useCase             ChapterUseCase
	plotAnalysisUseCase ChapterPlotAnalysisUseCase
}

type ChapterUseCase interface {
	CreateChapter(ctx context.Context, params dto.CreateChapterRequest) (*novelbiz.Chapter, error)
	NextChapterNo(ctx context.Context, novelID int64) (int, error)
	ListChapters(ctx context.Context, novelID int64, page, pageSize int) (*novelbiz.ListChapterResult, error)
	GetChapter(ctx context.Context, novelID int64, chapterID uint) (*novelbiz.Chapter, error)
	UpdateChapter(ctx context.Context, params dto.UpdateChapterRequest) (*novelbiz.Chapter, error)
	DeleteChapter(ctx context.Context, novelID int64, chapterID uint) error
}

type ChapterPlotAnalysisUseCase interface {
	GetChapterPlotAnalysis(ctx context.Context, novelID int64, chapterID uint) (*novelbiz.ChapterPlotAnalysis, error)
}

type chapterResponse struct {
	// 章节 ID
	ID uint `json:"id"`
	// 小说 ID
	NovelID int64 `json:"novelId"`
	// 章节编号
	ChapterNo int `json:"chapterNo"`
	// 章节标题
	Title string `json:"title"`
	// 章节内容
	Content string `json:"content"`
	// 章节字数
	WordCount int `json:"wordCount"`
	// 创建时间
	CreatedAt string `json:"createdAt"`
	// 更新时间
	UpdatedAt string `json:"updatedAt"`
}

type chapterListItemResponse struct {
	// 章节 ID
	ID uint `json:"id"`
	// 小说 ID
	NovelID int64 `json:"novelId"`
	// 章节编号
	ChapterNo int `json:"chapterNo"`
	// 章节标题
	Title string `json:"title"`
	// 章节字数
	WordCount int `json:"wordCount"`
	// 创建时间
	CreatedAt string `json:"createdAt"`
	// 更新时间
	UpdatedAt string `json:"updatedAt"`
}

type chapterListResponse struct {
	Items    []chapterListItemResponse `json:"items"`
	Total    int64                     `json:"total"`
	Page     int                       `json:"page"`
	PageSize int                       `json:"pageSize"`
}

type nextChapterNoResponse struct {
	ChapterNo int `json:"chapterNo"`
}

type chapterPlotAnalysisResponse struct {
	// 剧情总结 ID
	ID int64 `json:"id"`
	// 小说 ID
	NovelID int64 `json:"novelId"`
	// 章节 ID
	ChapterID uint `json:"chapterId"`
	// 本章剧情概述
	Summary string `json:"summary"`
	// 本章关键事件列表
	KeyEvents datatypes.JSON `json:"key_events" swaggertype:"array,string"`
	// 本章主要涉及角色列表
	CharactersInvolved datatypes.JSON `json:"characters_involved" swaggertype:"array,string"`
	// 本章人物关系变化列表
	RelationshipChanges datatypes.JSON `json:"relationship_changes" swaggertype:"array,object"`
	// 本章关键事件分析列表
	EventAnalysis datatypes.JSON `json:"event_analysis" swaggertype:"array,string"`
	// 本章伏笔列表
	Foreshadowing datatypes.JSON `json:"foreshadowing" swaggertype:"array,string"`
	// 本章未解决线索列表
	UnresolvedThreads datatypes.JSON `json:"unresolved_threads" swaggertype:"array,string"`
	// 创建时间
	CreatedAt string `json:"createdAt"`
	// 更新时间
	UpdatedAt string `json:"updatedAt"`
}

func NewChapterService(useCase ChapterUseCase, plotAnalysisUseCase ChapterPlotAnalysisUseCase) *ChapterService {
	return &ChapterService{useCase: useCase, plotAnalysisUseCase: plotAnalysisUseCase}
}

// Create 新增章节
// @Summary 新增章节
// @Description 给指定小说新增章节，保存成功后会通过事件触发章节剧情总结，并同步增加小说总字数；剧情总结不在本接口返回
// @Tags chapter
// @Accept json
// @Produce json
// @Param id path int true "小说 ID"
// @Param chapter body dto.CreateChapterRequest true "章节信息"
// @Success 200 {object} common.Response{data=chapterResponse}
// @Failure 400 {object} common.Response "请求参数错误"
// @Failure 401 {object} common.Response "未登录"
// @Failure 404 {object} common.Response "小说不存在"
// @Failure 409 {object} common.Response "章节编号已存在"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /novels/{id}/chapters [post]
func (service *ChapterService) Create(c *gin.Context) {
	novelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || novelID <= 0 {
		_ = c.Error(common.InvalidRequest)
		return
	}

	var request dto.CreateChapterRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		_ = c.Error(common.InvalidRequestWithValidationMessage(request, err))
		return
	}
	request.NovelID = novelID

	chapter, err := service.useCase.CreateChapter(c.Request.Context(), request)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: toChapterResponse(chapter),
	})
}

// NextChapterNo 查询下一章编号
// @Summary 查询下一章编号
// @Description 返回指定小说当前最大章节编号加一
// @Tags chapter
// @Produce json
// @Param id path int true "小说 ID"
// @Success 200 {object} common.Response{data=nextChapterNoResponse}
// @Failure 400 {object} common.Response "请求参数错误"
// @Failure 401 {object} common.Response "未登录"
// @Failure 404 {object} common.Response "小说不存在"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /novels/{id}/chapters/next-no [get]
func (service *ChapterService) NextChapterNo(c *gin.Context) {
	novelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || novelID <= 0 {
		_ = c.Error(common.InvalidRequest)
		return
	}

	chapterNo, err := service.useCase.NextChapterNo(c.Request.Context(), novelID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: nextChapterNoResponse{ChapterNo: chapterNo},
	})
}

// List 查询章节列表
// @Summary 查询章节列表
// @Description 分页查询指定小说的章节列表，列表不返回章节正文
// @Tags chapter
// @Produce json
// @Param id path int true "小说 ID"
// @Param page query int false "页码，默认 1"
// @Param pageSize query int false "每页数量，默认 10，最大 100"
// @Success 200 {object} common.Response{data=chapterListResponse}
// @Failure 400 {object} common.Response "请求参数错误"
// @Failure 401 {object} common.Response "未登录"
// @Failure 404 {object} common.Response "小说不存在"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /novels/{id}/chapters [get]
func (service *ChapterService) List(c *gin.Context) {
	novelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || novelID <= 0 {
		_ = c.Error(common.InvalidRequest)
		return
	}

	page := 1
	if c.Query("page") != "" {
		page, err = strconv.Atoi(c.Query("page"))
		if err != nil || page <= 0 {
			_ = c.Error(common.InvalidRequest)
			return
		}
	}

	pageSize := 10
	if c.Query("pageSize") != "" {
		pageSize, err = strconv.Atoi(c.Query("pageSize"))
		if err != nil || pageSize <= 0 {
			_ = c.Error(common.InvalidRequest)
			return
		}
	}
	if pageSize > maxNovelPageSize {
		pageSize = maxNovelPageSize
	}

	result, err := service.useCase.ListChapters(c.Request.Context(), novelID, page, pageSize)
	if err != nil {
		_ = c.Error(err)
		return
	}

	items := make([]chapterListItemResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toChapterListItemResponse(new(item)))
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: &chapterListResponse{
			Items:    items,
			Total:    result.Total,
			Page:     result.Page,
			PageSize: result.PageSize,
		},
	})
}

// Get 查询章节详情
// @Summary 查询章节详情
// @Description 查询指定小说下的章节详情，返回完整正文；章节剧情总结请调用独立查询接口
// @Tags chapter
// @Produce json
// @Param id path int true "小说 ID"
// @Param chapterId path int true "章节 ID"
// @Success 200 {object} common.Response{data=chapterResponse}
// @Failure 400 {object} common.Response "请求参数错误"
// @Failure 401 {object} common.Response "未登录"
// @Failure 404 {object} common.Response "章节不存在"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /novels/{id}/chapters/{chapterId} [get]
func (service *ChapterService) Get(c *gin.Context) {
	// Gin 不允许同一层级同时注册静态 next-no 和参数 :chapterId，这里保留对外 URL 并在统一入口分派。
	if c.Param("chapterId") == "next-no" {
		service.NextChapterNo(c)
		return
	}

	novelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || novelID <= 0 {
		_ = c.Error(common.InvalidRequest)
		return
	}
	chapterID, err := strconv.ParseUint(c.Param("chapterId"), 10, 64)
	if err != nil || chapterID == 0 {
		_ = c.Error(common.InvalidRequest)
		return
	}

	chapter, err := service.useCase.GetChapter(c.Request.Context(), novelID, uint(chapterID))
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: toChapterResponse(chapter),
	})
}

// GetPlotAnalysis 查询章节剧情总结
// @Summary 查询章节剧情总结
// @Description 查询指定小说章节已经持久化的 AI 剧情总结。章节保存或修改后会通过事件触发总结生成；如果 AI 尚未成功生成总结，本接口返回章节剧情总结不存在
// @Tags chapter
// @Produce json
// @Param id path int true "小说 ID"
// @Param chapterId path int true "章节 ID"
// @Success 200 {object} common.Response{data=chapterPlotAnalysisResponse} "code=0 表示查询成功；code=3008 表示 AI 尚未成功生成总结"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /novels/{id}/chapters/{chapterId}/plot-analysis [get]
func (service *ChapterService) GetPlotAnalysis(c *gin.Context) {
	novelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || novelID <= 0 {
		_ = c.Error(common.InvalidRequest)
		return
	}
	chapterID, err := strconv.ParseUint(c.Param("chapterId"), 10, 64)
	if err != nil || chapterID == 0 {
		_ = c.Error(common.InvalidRequest)
		return
	}

	analysis, err := service.plotAnalysisUseCase.GetChapterPlotAnalysis(c.Request.Context(), novelID, uint(chapterID))
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: toChapterPlotAnalysisResponse(analysis),
	})
}

// Update 修改章节
// @Summary 修改章节
// @Description 全量更新章节，保存成功后会通过事件触发章节剧情总结，并按章节字数差值同步小说总字数；剧情总结不在本接口返回
// @Tags chapter
// @Accept json
// @Produce json
// @Param id path int true "小说 ID"
// @Param chapterId path int true "章节 ID"
// @Param chapter body dto.UpdateChapterRequest true "章节信息"
// @Success 200 {object} common.Response{data=chapterResponse}
// @Failure 400 {object} common.Response "请求参数错误"
// @Failure 401 {object} common.Response "未登录"
// @Failure 404 {object} common.Response "章节不存在"
// @Failure 409 {object} common.Response "章节编号已存在"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /novels/{id}/chapters/{chapterId} [put]
func (service *ChapterService) Update(c *gin.Context) {
	novelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || novelID <= 0 {
		_ = c.Error(common.NewSystemError(2000, "小说 ID 不能为空"))
		return
	}
	chapterID, err := strconv.ParseUint(c.Param("chapterId"), 10, 64)
	if err != nil || chapterID == 0 {
		_ = c.Error(common.NewSystemError(2000, "章节 ID 不能为空"))
		return
	}

	var request dto.UpdateChapterRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		slog.ErrorContext(c.Request.Context(), "更新章节请求参数错误", "err", err)
		_ = c.Error(common.InvalidRequestWithValidationMessage(request, err))
		return
	}
	request.NovelID = novelID
	request.ChapterID = uint(chapterID)

	chapter, err := service.useCase.UpdateChapter(c.Request.Context(), request)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: toChapterResponse(chapter),
	})
}

// Delete 删除章节
// @Summary 删除章节
// @Description 软删除章节，并扣减小说总字数
// @Tags chapter
// @Produce json
// @Param id path int true "小说 ID"
// @Param chapterId path int true "章节 ID"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response "请求参数错误"
// @Failure 401 {object} common.Response "未登录"
// @Failure 404 {object} common.Response "章节不存在"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /novels/{id}/chapters/{chapterId} [delete]
func (service *ChapterService) Delete(c *gin.Context) {
	novelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || novelID <= 0 {
		_ = c.Error(common.InvalidRequest)
		return
	}
	chapterID, err := strconv.ParseUint(c.Param("chapterId"), 10, 64)
	if err != nil || chapterID == 0 {
		_ = c.Error(common.InvalidRequest)
		return
	}

	if err := service.useCase.DeleteChapter(c.Request.Context(), novelID, uint(chapterID)); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
	})
}

func toChapterResponse(chapter *novelbiz.Chapter) chapterResponse {
	return chapterResponse{
		ID:        chapter.ID,
		NovelID:   chapter.NovelID,
		ChapterNo: chapter.ChapterNo,
		Title:     chapter.Title,
		Content:   chapter.Content,
		WordCount: chapter.WordCount,
		CreatedAt: chapter.CreatedAt.Format("2006-01-02T15:04:05"),
		UpdatedAt: chapter.UpdatedAt.Format("2006-01-02T15:04:05"),
	}
}

func toChapterListItemResponse(chapter *novelbiz.Chapter) chapterListItemResponse {
	return chapterListItemResponse{
		ID:        chapter.ID,
		NovelID:   chapter.NovelID,
		ChapterNo: chapter.ChapterNo,
		Title:     chapter.Title,
		WordCount: chapter.WordCount,
		CreatedAt: chapter.CreatedAt.Format("2006-01-02T15:04:05"),
		UpdatedAt: chapter.UpdatedAt.Format("2006-01-02T15:04:05"),
	}
}

func toChapterPlotAnalysisResponse(analysis *novelbiz.ChapterPlotAnalysis) chapterPlotAnalysisResponse {
	return chapterPlotAnalysisResponse{
		ID:                  analysis.ID,
		NovelID:             analysis.NovelID,
		ChapterID:           analysis.ChapterID,
		Summary:             analysis.Summary,
		KeyEvents:           analysis.KeyEvents,
		CharactersInvolved:  analysis.CharactersInvolved,
		RelationshipChanges: analysis.RelationshipChanges,
		EventAnalysis:       analysis.EventAnalysis,
		Foreshadowing:       analysis.Foreshadowing,
		UnresolvedThreads:   analysis.UnresolvedThreads,
		CreatedAt:           analysis.CreatedAt.Format("2006-01-02T15:04:05"),
		UpdatedAt:           analysis.UpdatedAt.Format("2006-01-02T15:04:05"),
	}
}
