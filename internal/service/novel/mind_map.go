package novel

import (
	"context"
	"net/http"
	"strconv"

	novelbiz "Novels_AI/backend/internal/biz/novel"
	"Novels_AI/backend/internal/data/dto"
	"Novels_AI/backend/internal/pkg/common"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

type MindMapService struct {
	useCase MindMapUseCase
}

type MindMapUseCase interface {
	GetMindMap(ctx context.Context, novelID int64) (*novelbiz.NovelMindMap, error)
	SaveMindMap(ctx context.Context, params dto.SaveMindMapRequest) (*novelbiz.NovelMindMap, error)
	GetNode(ctx context.Context, novelID int64, nodeUID string) (datatypes.JSON, error)
	CreateNode(ctx context.Context, params dto.CreateMindMapNodeRequest) (datatypes.JSON, error)
	UpdateNode(ctx context.Context, params dto.UpdateMindMapNodeRequest) (datatypes.JSON, error)
	DeleteNode(ctx context.Context, novelID int64, nodeUID string) error
}

type mindMapResponse struct {
	// 思维导图 ID
	ID int64 `json:"id"`
	// 小说 ID
	NovelID int64 `json:"novelId"`
	// SimpleMindMap 完整 JSON 数据，可直接传给 setData。
	MindMapData datatypes.JSON `json:"mindMapData" swaggertype:"object"`
	// 创建时间
	CreatedAt string `json:"createdAt"`
	// 更新时间
	UpdatedAt string `json:"updatedAt"`
}

type mindMapNodeResponse struct {
	// SimpleMindMap 单个节点 JSON。
	Node datatypes.JSON `json:"node" swaggertype:"object"`
}

func NewMindMapService(useCase MindMapUseCase) *MindMapService {
	return &MindMapService{useCase: useCase}
}

// Get 查询小说思维导图。
//
// 前端打开思维导图页面时调用本接口。拿到 mindMapData 后可以直接交给 SimpleMindMap：
// mindMap.setData(response.data.mindMapData)。如果这本小说从未保存过思维导图，后端会先创建一个默认根节点。
// @Summary 查询小说思维导图
// @Description 查询指定小说的 SimpleMindMap 完整 JSON 数据；首次查询时会初始化默认根节点，前端拿到 mindMapData 后可直接 setData
// @Tags mind-map
// @Produce json
// @Param id path int true "小说 ID"
// @Success 200 {object} common.Response{data=mindMapResponse}
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /novels/{id}/mind-map [get]
func (service *MindMapService) Get(c *gin.Context) {
	novelID, ok := bindMindMapNovelID(c)
	if !ok {
		return
	}

	mindMap, err := service.useCase.GetMindMap(c.Request.Context(), novelID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: toMindMapResponse(mindMap),
	})
}

// Save 保存小说思维导图。
//
// 前端推荐在用户点击“保存”、离开页面前确认保存、或监听 SimpleMindMap 变更事件后做防抖自动保存时调用。
// 请求体里的 mindMapData 建议使用 mindMap.getData(true)，这样节点树、备注、概要、关联线、外框、主题、布局和视图都会保留。
// @Summary 保存小说思维导图
// @Description 保存 SimpleMindMap 完整 JSON 数据，推荐前端在保存按钮、离开页面前或防抖自动保存时提交 getData(true) 的结果
// @Tags mind-map
// @Accept json
// @Produce json
// @Param id path int true "小说 ID"
// @Param mindMap body dto.SaveMindMapRequest true "思维导图数据"
// @Success 200 {object} common.Response{data=mindMapResponse}
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /novels/{id}/mind-map [put]
func (service *MindMapService) Save(c *gin.Context) {
	novelID, ok := bindMindMapNovelID(c)
	if !ok {
		return
	}

	var request dto.SaveMindMapRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		_ = c.Error(common.InvalidRequestWithValidationMessage(request, err))
		return
	}
	request.NovelID = novelID

	mindMap, err := service.useCase.SaveMindMap(c.Request.Context(), request)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: toMindMapResponse(mindMap),
	})
}

// GetNode 查询思维导图节点。
//
// 前端一般在需要重新加载某个节点后端最新状态时使用，例如打开节点属性面板前做一次刷新。
// 如果前端已经持有整张 SimpleMindMap 数据，通常不需要频繁调用这个接口。
// @Summary 查询思维导图节点
// @Description 按 SimpleMindMap 节点 uid 查询单个节点 JSON；已有整图数据时前端通常可以直接本地查找
// @Tags mind-map
// @Produce json
// @Param id path int true "小说 ID"
// @Param nodeUid path string true "节点 uid"
// @Success 200 {object} common.Response{data=mindMapNodeResponse}
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /novels/{id}/mind-map/nodes/{nodeUid} [get]
func (service *MindMapService) GetNode(c *gin.Context) {
	novelID, nodeUID, ok := bindMindMapNodePath(c)
	if !ok {
		return
	}

	node, err := service.useCase.GetNode(c.Request.Context(), novelID, nodeUID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: mindMapNodeResponse{Node: node},
	})
}

// CreateNode 新增思维导图节点。
//
// 当前端希望“新增节点后立即保存到数据库”时调用本接口。另一种更简单的方式是：
// 前端先用 SimpleMindMap 在本地新增节点，等用户点击保存时调用 Save 一次性提交整图。
// @Summary 新增思维导图节点
// @Description 在指定父节点下新增 SimpleMindMap 节点；适合新增后立即落库，不强制前端每次新增都调用
// @Tags mind-map
// @Accept json
// @Produce json
// @Param id path int true "小说 ID"
// @Param node body dto.CreateMindMapNodeRequest true "节点数据"
// @Success 200 {object} common.Response{data=mindMapNodeResponse}
// @Failure 400 {object} common.Response "请求参数错误"
// @Failure 401 {object} common.Response "未登录"
// @Failure 404 {object} common.Response "小说或父节点不存在"
// @Failure 409 {object} common.Response "节点已存在"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /novels/{id}/mind-map/nodes [post]
func (service *MindMapService) CreateNode(c *gin.Context) {
	novelID, ok := bindMindMapNovelID(c)
	if !ok {
		return
	}

	var request dto.CreateMindMapNodeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		_ = c.Error(common.InvalidRequestWithValidationMessage(request, err))
		return
	}
	request.NovelID = novelID

	node, err := service.useCase.CreateNode(c.Request.Context(), request)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: mindMapNodeResponse{Node: node},
	})
}

// UpdateNode 修改思维导图节点。
//
// 当前端只保存单个节点属性时调用本接口，例如节点文本、备注 note、概要 generalization、
// 关联线 associativeLine*、图片、标签、超链接或自定义位置。它只替换 data 字段，不会改 children 子树。
// @Summary 修改思维导图节点
// @Description 更新指定 SimpleMindMap 节点的 data 字段，保留 children 子树；多节点或布局变更建议直接保存整图
// @Tags mind-map
// @Accept json
// @Produce json
// @Param id path int true "小说 ID"
// @Param nodeUid path string true "节点 uid"
// @Param node body dto.UpdateMindMapNodeRequest true "节点 data 数据"
// @Success 200 {object} common.Response{data=mindMapNodeResponse}
// @Failure 400 {object} common.Response "请求参数错误"
// @Failure 401 {object} common.Response "未登录"
// @Failure 404 {object} common.Response "小说或节点不存在"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /novels/{id}/mind-map/nodes/{nodeUid} [put]
func (service *MindMapService) UpdateNode(c *gin.Context) {
	novelID, nodeUID, ok := bindMindMapNodePath(c)
	if !ok {
		return
	}

	var request dto.UpdateMindMapNodeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		_ = c.Error(common.InvalidRequestWithValidationMessage(request, err))
		return
	}
	request.NovelID = novelID
	request.NodeUID = nodeUID

	node, err := service.useCase.UpdateNode(c.Request.Context(), request)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: mindMapNodeResponse{Node: node},
	})
}

// DeleteNode 删除思维导图节点。
//
// 当前端希望“删除节点后立即保存到数据库”时调用本接口。后端会删除该节点及所有后代节点；
// 如果前端已经在 SimpleMindMap 本地删除并稍后调用 Save 保存整图，也可以不调用本接口。
// @Summary 删除思维导图节点
// @Description 删除指定 SimpleMindMap 节点及其所有后代节点；删除父节点会连同子树一起删除
// @Tags mind-map
// @Produce json
// @Param id path int true "小说 ID"
// @Param nodeUid path string true "节点 uid"
// @Success 200 {object} common.Response
// @Failure 400 {object} common.Response "请求参数错误"
// @Failure 401 {object} common.Response "未登录"
// @Failure 404 {object} common.Response "小说或节点不存在"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /novels/{id}/mind-map/nodes/{nodeUid} [delete]
func (service *MindMapService) DeleteNode(c *gin.Context) {
	novelID, nodeUID, ok := bindMindMapNodePath(c)
	if !ok {
		return
	}

	if err := service.useCase.DeleteNode(c.Request.Context(), novelID, nodeUID); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
	})
}

func bindMindMapNovelID(c *gin.Context) (int64, bool) {
	novelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || novelID <= 0 {
		_ = c.Error(common.InvalidRequest)
		return 0, false
	}

	return novelID, true
}

func bindMindMapNodePath(c *gin.Context) (int64, string, bool) {
	novelID, ok := bindMindMapNovelID(c)
	if !ok {
		return 0, "", false
	}

	nodeUID := c.Param("nodeUid")
	if nodeUID == "" {
		_ = c.Error(common.InvalidRequest)
		return 0, "", false
	}

	return novelID, nodeUID, true
}

func toMindMapResponse(mindMap *novelbiz.NovelMindMap) mindMapResponse {
	return mindMapResponse{
		ID:          mindMap.ID,
		NovelID:     mindMap.NovelID,
		MindMapData: mindMap.MindMapData,
		CreatedAt:   mindMap.CreatedAt.Format("2006-01-02T15:04:05"),
		UpdatedAt:   mindMap.UpdatedAt.Format("2006-01-02T15:04:05"),
	}
}
