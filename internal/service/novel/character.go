package novel

import (
	"context"
	"net/http"
	"strconv"

	novelbiz "Novels_AI/backend/internal/biz/novel"
	"Novels_AI/backend/internal/pkg/common"

	"github.com/gin-gonic/gin"
)

// CharacterService 负责把角色相关 HTTP 请求转换成业务用例参数，并统一写出响应。
type CharacterService struct {
	useCase CharacterUseCase
}

// CharacterUseCase 描述角色 service 依赖的业务能力，便于测试时替换为轻量实现。
type CharacterUseCase interface {
	CreateCharacter(ctx context.Context, params novelbiz.CreateCharacterParams) (*novelbiz.Character, error)
	ListCharacters(ctx context.Context, novelID int64, page, pageSize int) (*novelbiz.ListCharacterResult, error)
	GetCharacter(ctx context.Context, novelID int64, characterID uint) (*novelbiz.Character, error)
	UpdateCharacter(ctx context.Context, params novelbiz.UpdateCharacterParams) (*novelbiz.Character, error)
	DeleteCharacter(ctx context.Context, novelID int64, characterID uint) error
}

type createCharacterRequest struct {
	// 角色名称，必填
	Name string `json:"name" binding:"required"`
	// 角色性别
	Gender string `json:"gender"`
	// 角色介绍
	Intro string `json:"intro"`
	// 角色性格
	Personality string `json:"personality"`
	// 角色外貌
	Appearance string `json:"appearance"`
	// 角色背景
	Background string `json:"background"`
	// 角色能力
	Ability string `json:"ability"`
	// 角色动机
	Motivation string `json:"motivation"`
	// 角色剧情方向
	PlotDirection string `json:"plotDirection"`
	// 首次出现章节 ID
	FirstAppearanceChapterID *int64 `json:"firstAppearanceChapterId"`
	// 角色形象图 URL
	AppearanceImgURL string `json:"appearanceImgUrl"`
	// 角色状态：1 在线，2 下线
	Status *int16 `json:"status"`
	// 角色标签
	CharactersTags []string `json:"charactersTags"`
}

type updateCharacterRequest struct {
	// 角色名称
	Name *string `json:"name"`
	// 角色性别
	Gender *string `json:"gender"`
	// 角色介绍
	Intro *string `json:"intro"`
	// 角色性格
	Personality *string `json:"personality"`
	// 角色外貌
	Appearance *string `json:"appearance"`
	// 角色背景
	Background *string `json:"background"`
	// 角色能力
	Ability *string `json:"ability"`
	// 角色动机
	Motivation *string `json:"motivation"`
	// 角色剧情方向
	PlotDirection *string `json:"plotDirection"`
	// 首次出现章节 ID
	FirstAppearanceChapterID *int64 `json:"firstAppearanceChapterId"`
	// 角色形象图 URL
	AppearanceImgURL *string `json:"appearanceImgUrl"`
	// 角色状态：1 在线，2 下线
	Status *int16 `json:"status"`
	// 角色标签；使用指针用于区分“未传”和“传空数组”
	CharactersTags *[]string `json:"charactersTags"`
}

type characterResponse struct {
	// 角色 ID
	ID uint `json:"id"`
	// 小说 ID
	NovelID int64 `json:"novelId"`
	// 角色名称
	Name string `json:"name"`
	// 角色性别
	Gender string `json:"gender"`
	// 角色介绍
	Intro string `json:"intro"`
	// 角色性格
	Personality string `json:"personality"`
	// 角色外貌
	Appearance string `json:"appearance"`
	// 角色背景
	Background string `json:"background"`
	// 角色能力
	Ability string `json:"ability"`
	// 角色动机
	Motivation string `json:"motivation"`
	// 角色剧情方向
	PlotDirection string `json:"plotDirection"`
	// 首次出现章节 ID
	FirstAppearanceChapterID *int64 `json:"firstAppearanceChapterId"`
	// 角色形象图 URL
	AppearanceImgURL string `json:"appearanceImgUrl"`
	// 角色状态：1 在线，2 下线
	Status int16 `json:"status"`
	// 角色标签
	CharactersTags []string `json:"charactersTags"`
	// 创建时间
	CreatedAt string `json:"createdAt"`
	// 更新时间
	UpdatedAt string `json:"updatedAt"`
}

type characterListItemResponse struct {
	// 角色名称
	Name string `json:"name"`
	// 角色性别
	Gender string `json:"gender"`
	// 角色形象图 URL
	AppearanceImgURL string `json:"appearanceImgUrl"`
	// 角色状态：1 在线，2 下线
	Status int16 `json:"status"`
	// 角色标签
	CharactersTags []string `json:"charactersTags"`
}

type characterListResponse struct {
	Items    []characterListItemResponse `json:"items"`
	Total    int64                       `json:"total"`
	Page     int                         `json:"page"`
	PageSize int                         `json:"pageSize"`
}

// NewCharacterService 创建角色 HTTP 服务适配器。
func NewCharacterService(useCase CharacterUseCase) *CharacterService {
	return &CharacterService{useCase: useCase}
}

// Create 新增角色
// @Summary 新增角色
// @Description 给指定小说新增角色资料
// @Tags character
// @Accept json
// @Produce json
// @Param id path int true "小说 ID"
// @Param character body createCharacterRequest true "角色信息"
// @Success 200 {object} common.Response{data=characterResponse}
// @Failure 400 {object} common.Response "请求参数错误"
// @Failure 401 {object} common.Response "未登录"
// @Failure 404 {object} common.Response "小说不存在"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /novels/{id}/characters [post]
func (service *CharacterService) Create(c *gin.Context) {
	novelID, ok := parseCharacterNovelID(c)
	if !ok {
		return
	}

	var request createCharacterRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		_ = c.Error(common.InvalidRequest)
		return
	}

	character, err := service.useCase.CreateCharacter(c.Request.Context(), novelbiz.CreateCharacterParams{
		NovelID:                  novelID,
		Name:                     request.Name,
		Gender:                   request.Gender,
		Intro:                    request.Intro,
		Personality:              request.Personality,
		Appearance:               request.Appearance,
		Background:               request.Background,
		Ability:                  request.Ability,
		Motivation:               request.Motivation,
		PlotDirection:            request.PlotDirection,
		FirstAppearanceChapterID: request.FirstAppearanceChapterID,
		AppearanceImgURL:         request.AppearanceImgURL,
		Status:                   request.Status,
		CharactersTags:           request.CharactersTags,
	})
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: toCharacterResponse(character),
	})
}

// List 查询角色列表
// @Summary 查询角色列表
// @Description 分页查询指定小说的角色列表，列表只返回角色名、性别、图像 URL、状态和标签
// @Tags character
// @Produce json
// @Param id path int true "小说 ID"
// @Param page query int false "页码，默认 1"
// @Param pageSize query int false "每页数量，默认 10，最大 100"
// @Success 200 {object} common.Response{data=characterListResponse}
// @Failure 400 {object} common.Response "请求参数错误"
// @Failure 401 {object} common.Response "未登录"
// @Failure 404 {object} common.Response "小说不存在"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /novels/{id}/characters [get]
func (service *CharacterService) List(c *gin.Context) {
	novelID, ok := parseCharacterNovelID(c)
	if !ok {
		return
	}

	page, pageSize, ok := bindCharacterPagination(c)
	if !ok {
		return
	}

	result, err := service.useCase.ListCharacters(c.Request.Context(), novelID, page, pageSize)
	if err != nil {
		_ = c.Error(err)
		return
	}

	items := make([]characterListItemResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toCharacterListItemResponse(new(item)))
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: &characterListResponse{
			Items:    items,
			Total:    result.Total,
			Page:     result.Page,
			PageSize: result.PageSize,
		},
	})
}

// Get 查询角色详情
// @Summary 查询角色详情
// @Description 查询指定小说下的角色完整资料
// @Tags character
// @Produce json
// @Param id path int true "小说 ID"
// @Param characterId path int true "角色 ID"
// @Success 200 {object} common.Response{data=characterResponse}
// @Failure 400 {object} common.Response "请求参数错误"
// @Failure 401 {object} common.Response "未登录"
// @Failure 404 {object} common.Response "角色不存在"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /novels/{id}/characters/{characterId} [get]
func (service *CharacterService) Get(c *gin.Context) {
	novelID, characterID, ok := parseCharacterIDs(c)
	if !ok {
		return
	}

	character, err := service.useCase.GetCharacter(c.Request.Context(), novelID, characterID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: toCharacterResponse(character),
	})
}

// Update 修改角色
// @Summary 修改角色
// @Description 局部更新指定小说下的角色资料
// @Tags character
// @Accept json
// @Produce json
// @Param id path int true "小说 ID"
// @Param characterId path int true "角色 ID"
// @Param character body updateCharacterRequest true "角色信息"
// @Success 200 {object} common.Response{data=characterResponse}
// @Failure 400 {object} common.Response "请求参数错误"
// @Failure 401 {object} common.Response "未登录"
// @Failure 404 {object} common.Response "角色不存在"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /novels/{id}/characters/{characterId} [put]
func (service *CharacterService) Update(c *gin.Context) {
	novelID, characterID, ok := parseCharacterIDs(c)
	if !ok {
		return
	}

	var request updateCharacterRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		_ = c.Error(common.InvalidRequest)
		return
	}

	character, err := service.useCase.UpdateCharacter(c.Request.Context(), novelbiz.UpdateCharacterParams{
		NovelID:                  novelID,
		CharacterID:              characterID,
		Name:                     request.Name,
		Gender:                   request.Gender,
		Intro:                    request.Intro,
		Personality:              request.Personality,
		Appearance:               request.Appearance,
		Background:               request.Background,
		Ability:                  request.Ability,
		Motivation:               request.Motivation,
		PlotDirection:            request.PlotDirection,
		FirstAppearanceChapterID: request.FirstAppearanceChapterID,
		AppearanceImgURL:         request.AppearanceImgURL,
		Status:                   request.Status,
		CharactersTags:           request.CharactersTags,
	})
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: toCharacterResponse(character),
	})
}

// Delete 删除角色
// @Summary 删除角色
// @Description 软删除指定小说下的角色
// @Tags character
// @Produce json
// @Param id path int true "小说 ID"
// @Param characterId path int true "角色 ID"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response "请求参数错误"
// @Failure 401 {object} common.Response "未登录"
// @Failure 404 {object} common.Response "角色不存在"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /novels/{id}/characters/{characterId} [delete]
func (service *CharacterService) Delete(c *gin.Context) {
	novelID, characterID, ok := parseCharacterIDs(c)
	if !ok {
		return
	}

	if err := service.useCase.DeleteCharacter(c.Request.Context(), novelID, characterID); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
	})
}

// parseCharacterNovelID 解析小说路径参数，失败时交给统一错误中间件返回请求参数错误。
func parseCharacterNovelID(c *gin.Context) (int64, bool) {
	novelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || novelID <= 0 {
		_ = c.Error(common.InvalidRequest)
		return 0, false
	}

	return novelID, true
}

// parseCharacterIDs 同时解析小说 ID 和角色 ID，保证详情、更新、删除接口的参数校验一致。
func parseCharacterIDs(c *gin.Context) (int64, uint, bool) {
	novelID, ok := parseCharacterNovelID(c)
	if !ok {
		return 0, 0, false
	}

	characterID, err := strconv.ParseUint(c.Param("characterId"), 10, 64)
	if err != nil || characterID == 0 {
		_ = c.Error(common.InvalidRequest)
		return 0, 0, false
	}

	return novelID, uint(characterID), true
}

// bindCharacterPagination 解析分页查询参数，默认值与小说、章节列表保持一致。
func bindCharacterPagination(c *gin.Context) (int, int, bool) {
	page := 1
	if c.Query("page") != "" {
		value, err := strconv.Atoi(c.Query("page"))
		if err != nil || value <= 0 {
			_ = c.Error(common.InvalidRequest)
			return 0, 0, false
		}
		page = value
	}

	pageSize := 10
	if c.Query("pageSize") != "" {
		value, err := strconv.Atoi(c.Query("pageSize"))
		if err != nil || value <= 0 {
			_ = c.Error(common.InvalidRequest)
			return 0, 0, false
		}
		pageSize = value
	}

	return page, pageSize, true
}

// toCharacterResponse 将业务角色模型转换成详情响应，详情接口返回完整角色资料。
func toCharacterResponse(character *novelbiz.Character) characterResponse {
	return characterResponse{
		ID:                       character.ID,
		NovelID:                  character.NovelID,
		Name:                     character.Name,
		Gender:                   character.Gender,
		Intro:                    character.Intro,
		Personality:              character.Personality,
		Appearance:               character.Appearance,
		Background:               character.Background,
		Ability:                  character.Ability,
		Motivation:               character.Motivation,
		PlotDirection:            character.PlotDirection,
		FirstAppearanceChapterID: character.FirstAppearanceChapterID,
		AppearanceImgURL:         character.AppearanceImgURL,
		Status:                   character.Status,
		CharactersTags:           []string(character.CharactersTags),
		CreatedAt:                character.CreatedAt.Format("2006-01-02T15:04:05"),
		UpdatedAt:                character.UpdatedAt.Format("2006-01-02T15:04:05"),
	}
}

// toCharacterListItemResponse 将业务角色模型转换成列表摘要，只保留用户要求的五类角色信息。
func toCharacterListItemResponse(character *novelbiz.Character) characterListItemResponse {
	return characterListItemResponse{
		Name:             character.Name,
		Gender:           character.Gender,
		AppearanceImgURL: character.AppearanceImgURL,
		Status:           character.Status,
		CharactersTags:   []string(character.CharactersTags),
	}
}
