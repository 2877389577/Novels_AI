package novel

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	novelbiz "Novels_AI/backend/internal/biz/novel"
	"Novels_AI/backend/internal/data/dto"
	"Novels_AI/backend/internal/pkg/common"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

// CharacterRelationService 负责把角色关系图 HTTP 请求转换成业务用例参数。
type CharacterRelationService struct {
	useCase CharacterRelationUseCase
}

// CharacterRelationUseCase 描述角色关系图 service 依赖的业务能力。
type CharacterRelationUseCase interface {
	GetCharacterRelationGraph(ctx context.Context, novelID int64) (*novelbiz.CharacterRelationGraphResult, error)
	SaveCharacterRelationNodeLayouts(ctx context.Context, params dto.SaveCharacterRelationNodeLayoutsRequest) ([]novelbiz.CharacterRelationNode, error)
	ListCharacterRelations(ctx context.Context, novelID int64) (*novelbiz.ListCharacterRelationResult, error)
	CreateCharacterRelation(ctx context.Context, params dto.CreateCharacterRelationRequest) (*novelbiz.CharacterRelation, error)
	GetCharacterRelation(ctx context.Context, novelID int64, relationID int64) (*novelbiz.CharacterRelation, error)
	UpdateCharacterRelation(ctx context.Context, params dto.UpdateCharacterRelationRequest) (*novelbiz.CharacterRelation, error)
	DeleteCharacterRelation(ctx context.Context, novelID int64, relationID int64) error
}

type characterRelationGraphResponse struct {
	// Vue Flow 节点列表，节点 ID 使用角色 ID 的字符串形式。
	Nodes []characterRelationGraphNodeResponse `json:"nodes"`
	// Vue Flow 边列表，边 ID 使用角色关系 ID 的字符串形式。
	Edges []characterRelationGraphEdgeResponse `json:"edges"`
}

type characterRelationGraphNodeResponse struct {
	// Vue Flow 节点 ID，等于角色 ID 的字符串形式。
	ID string `json:"id"`
	// Vue Flow 节点类型，默认 character。
	Type string `json:"type"`
	// Vue Flow 节点位置。
	Position vueFlowPositionResponse `json:"position"`
	// 节点宽度，未保存时为空。
	Width *float64 `json:"width,omitempty"`
	// 节点高度，未保存时为空。
	Height *float64 `json:"height,omitempty"`
	// 是否隐藏节点。
	Hidden bool `json:"hidden"`
	// 是否锁定节点位置。
	Locked bool `json:"locked"`
	// Vue Flow 节点样式。
	Style json.RawMessage `json:"style" swaggertype:"object"`
	// 节点渲染所需角色数据和扩展数据。
	Data characterRelationGraphNodeDataResponse `json:"data"`
}

type vueFlowPositionResponse struct {
	// X 坐标。
	X float64 `json:"x"`
	// Y 坐标。
	Y float64 `json:"y"`
}

type characterRelationGraphNodeDataResponse struct {
	// 节点布局 ID，未保存布局时为空。
	LayoutID *int64 `json:"layoutId"`
	// 小说 ID。
	NovelID int64 `json:"novelId"`
	// 角色 ID。
	CharacterID uint `json:"characterId"`
	// 角色名称。
	Name string `json:"name"`
	// 角色性别。
	Gender string `json:"gender"`
	// 角色形象图 URL。
	AppearanceImgURL string `json:"appearanceImgUrl"`
	// 角色状态：1 在线，2 下线。
	CharacterStatus int16 `json:"characterStatus"`
	// 角色标签。
	CharactersTags []string `json:"charactersTags"`
	// 节点布局状态：1 启用，2 停用。
	NodeStatus int16 `json:"nodeStatus"`
	// 是否锁定节点位置。
	Locked bool `json:"locked"`
	// 节点扩展数据。
	ExtraData json.RawMessage `json:"extraData" swaggertype:"object"`
}

type characterRelationGraphEdgeResponse struct {
	// Vue Flow 边 ID，等于角色关系 ID 的字符串形式。
	ID string `json:"id"`
	// Vue Flow 起始节点 ID，等于起始角色 ID 的字符串形式。
	Source string `json:"source"`
	// Vue Flow 目标节点 ID，等于目标角色 ID 的字符串形式。
	Target string `json:"target"`
	// Vue Flow 边类型，默认 default。
	Type string `json:"type"`
	// 关系展示名称，会展示在线条标签上。
	Label string `json:"label"`
	// 是否动画展示。
	Animated bool `json:"animated"`
	// 起始连接桩 ID。
	SourceHandle string `json:"sourceHandle,omitempty"`
	// 目标连接桩 ID。
	TargetHandle string `json:"targetHandle,omitempty"`
	// Vue Flow 边样式。
	Style json.RawMessage `json:"style" swaggertype:"object"`
	// 关系业务数据和扩展数据。
	Data characterRelationGraphEdgeDataResponse `json:"data"`
}

type characterRelationGraphEdgeDataResponse struct {
	// 角色关系 ID。
	RelationID int64 `json:"relationId"`
	// 小说 ID。
	NovelID int64 `json:"novelId"`
	// 起始角色 ID。
	SourceCharacterID int64 `json:"sourceCharacterId"`
	// 目标角色 ID。
	TargetCharacterID int64 `json:"targetCharacterId"`
	// 关系类型编码。
	RelationType string `json:"relationType"`
	// 关系展示名称。
	RelationLabel string `json:"relationLabel"`
	// 关系说明。
	Description string `json:"description"`
	// 关系方向：1 单向，2 双向或无向。
	Direction int16 `json:"direction"`
	// 排序值。
	SortOrder int `json:"sortOrder"`
	// 关系状态：1 启用，2 停用。
	Status int16 `json:"status"`
	// 关系扩展数据。
	ExtraData json.RawMessage `json:"extraData" swaggertype:"object"`
}

type characterRelationNodeLayoutResponse struct {
	// 节点布局 ID。
	ID int64 `json:"id"`
	// 小说 ID。
	NovelID int64 `json:"novelId"`
	// 角色 ID。
	CharacterID int64 `json:"characterId"`
	// Vue Flow 节点类型。
	NodeType string `json:"type"`
	// X 坐标。
	PositionX float64 `json:"positionX"`
	// Y 坐标。
	PositionY float64 `json:"positionY"`
	// 节点宽度。
	Width *float64 `json:"width"`
	// 节点高度。
	Height *float64 `json:"height"`
	// 是否隐藏。
	Hidden bool `json:"hidden"`
	// 是否锁定。
	Locked bool `json:"locked"`
	// 节点样式。
	Style json.RawMessage `json:"style" swaggertype:"object"`
	// 节点扩展数据。
	ExtraData json.RawMessage `json:"extraData" swaggertype:"object"`
	// 节点状态：1 启用，2 停用。
	Status int16 `json:"status"`
	// 创建时间。
	CreatedAt string `json:"createdAt"`
	// 更新时间。
	UpdatedAt string `json:"updatedAt"`
}

type characterRelationListResponse struct {
	Items []characterRelationResponse `json:"items"`
	Total int64                       `json:"total"`
}

type characterRelationResponse struct {
	// 角色关系 ID。
	ID int64 `json:"id"`
	// 小说 ID。
	NovelID int64 `json:"novelId"`
	// 起始角色 ID。
	SourceCharacterID int64 `json:"sourceCharacterId"`
	// 目标角色 ID。
	TargetCharacterID int64 `json:"targetCharacterId"`
	// 关系类型编码，例如 family、friend、enemy、lover、teacher、other。
	RelationType string `json:"relationType"`
	// 关系展示名称，例如 父女、好友、宿敌、师徒。
	RelationLabel string `json:"relationLabel"`
	// 关系说明。
	Description string `json:"description"`
	// 关系方向：1 单向，2 双向或无向。
	Direction int16 `json:"direction"`
	// Vue Flow 边类型。
	EdgeType string `json:"edgeType"`
	// 是否动画展示。
	Animated bool `json:"animated"`
	// 起始连接桩 ID。
	SourceHandle string `json:"sourceHandle"`
	// 目标连接桩 ID。
	TargetHandle string `json:"targetHandle"`
	// 排序值。
	SortOrder int `json:"sortOrder"`
	// 关系线样式。
	Style json.RawMessage `json:"style" swaggertype:"object"`
	// 关系扩展数据。
	ExtraData json.RawMessage `json:"extraData" swaggertype:"object"`
	// 关系状态：1 启用，2 停用。
	Status int16 `json:"status"`
	// 创建时间。
	CreatedAt string `json:"createdAt"`
	// 更新时间。
	UpdatedAt string `json:"updatedAt"`
}

// NewCharacterRelationService 创建角色关系图 HTTP 服务适配器。
func NewCharacterRelationService(useCase CharacterRelationUseCase) *CharacterRelationService {
	return &CharacterRelationService{useCase: useCase}
}

// GetGraph 查询角色关系图
// @Summary 查询角色关系图
// @Description 查询指定小说下的角色关系图数据，返回 Vue Flow 可直接使用的 nodes 和 edges。nodes 中每个节点代表一个角色，id 为角色 ID 的字符串形式；edges 中每条边代表一条启用的角色关系，source 和 target 分别对应起始角色和目标角色。
// @Tags character-relation
// @Produce json
// @Param id path int true "小说 ID"
// @Success 200 {object} common.Response{data=characterRelationGraphResponse} "code = 0 表示查询成功；data.nodes 为角色节点列表；data.edges 为角色关系连线列表"
// @Failure 400 {object} common.Response "请求参数错误，例如小说 ID 不是正整数"
// @Failure 401 {object} common.Response "未登录"
// @Failure 404 {object} common.Response "小说不存在"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /novels/{id}/character-relation-graph [get]
func (service *CharacterRelationService) GetGraph(c *gin.Context) {
	novelID, ok := parseCharacterNovelID(c)
	if !ok {
		return
	}

	result, err := service.useCase.GetCharacterRelationGraph(c.Request.Context(), novelID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: toCharacterRelationGraphResponse(result),
	})
}

// SaveNodeLayouts 保存角色关系图节点布局
// @Summary 保存角色关系图节点布局
// @Description 批量保存指定小说下角色节点在 Vue Flow 画布中的位置、尺寸、隐藏状态、锁定状态和样式。后端按 novelId + characterId 更新已有布局，不存在则新增；不会删除本次未提交的其他节点布局。
// @Tags character-relation
// @Accept json
// @Produce json
// @Param id path int true "小说 ID"
// @Param layout body dto.SaveCharacterRelationNodeLayoutsRequest true "节点布局列表。nodes[].characterId 为角色 ID；nodes[].positionX 和 nodes[].positionY 为画布坐标；nodes[].style 和 nodes[].extraData 可保存前端扩展 JSON"
// @Success 200 {object} common.Response{data=[]characterRelationNodeLayoutResponse} "code = 0 表示保存成功；data 返回本次保存后的节点布局"
// @Failure 400 {object} common.Response "请求参数错误，例如节点列表为空、角色 ID 为空或节点状态不是 1/2"
// @Failure 401 {object} common.Response "未登录"
// @Failure 404 {object} common.Response "小说或角色不存在"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /novels/{id}/character-relation-graph/nodes/layout [put]
func (service *CharacterRelationService) SaveNodeLayouts(c *gin.Context) {
	novelID, ok := parseCharacterNovelID(c)
	if !ok {
		return
	}

	var request dto.SaveCharacterRelationNodeLayoutsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		_ = c.Error(common.InvalidRequestWithValidationMessage(request, err))
		return
	}
	request.NovelID = novelID

	nodes, err := service.useCase.SaveCharacterRelationNodeLayouts(c.Request.Context(), request)
	if err != nil {
		_ = c.Error(err)
		return
	}

	items := make([]characterRelationNodeLayoutResponse, 0, len(nodes))
	for _, node := range nodes {
		items = append(items, toCharacterRelationNodeLayoutResponse(new(node)))
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: items,
	})
}

// List 查询角色关系列表
// @Summary 查询角色关系列表
// @Description 查询指定小说下所有角色关系，包含启用和停用状态。该接口用于关系管理列表，关系图展示接口只返回 status = 1 的启用关系。
// @Tags character-relation
// @Produce json
// @Param id path int true "小说 ID"
// @Success 200 {object} common.Response{data=characterRelationListResponse} "code = 0 表示查询成功；data.items 为关系列表；data.total 为关系总数"
// @Failure 400 {object} common.Response "请求参数错误，例如小说 ID 不是正整数"
// @Failure 401 {object} common.Response "未登录"
// @Failure 404 {object} common.Response "小说不存在"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /novels/{id}/character-relations [get]
func (service *CharacterRelationService) List(c *gin.Context) {
	novelID, ok := parseCharacterNovelID(c)
	if !ok {
		return
	}

	result, err := service.useCase.ListCharacterRelations(c.Request.Context(), novelID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	items := make([]characterRelationResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toCharacterRelationResponse(new(item)))
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: characterRelationListResponse{
			Items: items,
			Total: result.Total,
		},
	})
}

// Create 新增角色关系
// @Summary 新增角色关系
// @Description 给指定小说新增一条角色关系。sourceCharacterId 和 targetCharacterId 必须都属于当前小说，且不能相同；同一小说、同一组角色、同一 relationType 只能存在一条未删除关系。
// @Tags character-relation
// @Accept json
// @Produce json
// @Param id path int true "小说 ID"
// @Param relation body dto.CreateCharacterRelationRequest true "角色关系信息。relationType 是关系类型编码；relationLabel 是前端展示名称；direction 为 1 单向、2 双向或无向；style 和 extraData 可保存前端扩展 JSON"
// @Success 200 {object} common.Response{data=characterRelationResponse} "code = 0 表示新增成功；data 为新增后的角色关系"
// @Failure 400 {object} common.Response "请求参数错误，例如角色 ID 为空、关系类型为空、关系名称为空、方向或状态不正确、角色关联到自己"
// @Failure 401 {object} common.Response "未登录"
// @Failure 404 {object} common.Response "小说或角色不存在"
// @Failure 409 {object} common.Response "角色关系已存在"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /novels/{id}/character-relations [post]
func (service *CharacterRelationService) Create(c *gin.Context) {
	novelID, ok := parseCharacterNovelID(c)
	if !ok {
		return
	}

	var request dto.CreateCharacterRelationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		_ = c.Error(common.InvalidRequestWithValidationMessage(request, err))
		return
	}
	request.NovelID = novelID

	relation, err := service.useCase.CreateCharacterRelation(c.Request.Context(), request)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: toCharacterRelationResponse(relation),
	})
}

// Get 查询角色关系详情
// @Summary 查询角色关系详情
// @Description 查询指定小说下的单条角色关系详情，返回关系类型、展示名称、方向、样式和扩展数据等完整信息。
// @Tags character-relation
// @Produce json
// @Param id path int true "小说 ID"
// @Param relationId path int true "角色关系 ID"
// @Success 200 {object} common.Response{data=characterRelationResponse} "code = 0 表示查询成功；data 为角色关系详情"
// @Failure 400 {object} common.Response "请求参数错误，例如小说 ID 或关系 ID 不是正整数"
// @Failure 401 {object} common.Response "未登录"
// @Failure 404 {object} common.Response "小说或角色关系不存在"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /novels/{id}/character-relations/{relationId} [get]
func (service *CharacterRelationService) Get(c *gin.Context) {
	novelID, relationID, ok := parseCharacterRelationIDs(c)
	if !ok {
		return
	}

	relation, err := service.useCase.GetCharacterRelation(c.Request.Context(), novelID, relationID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: toCharacterRelationResponse(relation),
	})
}

// Update 修改角色关系
// @Summary 修改角色关系
// @Description 全量更新指定小说下的角色关系。请求体需要传入完整关系信息；sourceCharacterId 和 targetCharacterId 必须都属于当前小说，且不能相同。
// @Tags character-relation
// @Accept json
// @Produce json
// @Param id path int true "小说 ID"
// @Param relationId path int true "角色关系 ID"
// @Param relation body dto.UpdateCharacterRelationRequest true "角色关系信息。direction 必填且只能为 1 或 2；status 必填且只能为 1 或 2；style 和 extraData 会全量覆盖原值"
// @Success 200 {object} common.Response{data=characterRelationResponse} "code = 0 表示更新成功；data 为更新后的角色关系"
// @Failure 400 {object} common.Response "请求参数错误，例如角色 ID 为空、关系类型为空、方向或状态不正确、角色关联到自己"
// @Failure 401 {object} common.Response "未登录"
// @Failure 404 {object} common.Response "小说、角色或角色关系不存在"
// @Failure 409 {object} common.Response "角色关系已存在"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /novels/{id}/character-relations/{relationId} [put]
func (service *CharacterRelationService) Update(c *gin.Context) {
	novelID, relationID, ok := parseCharacterRelationIDs(c)
	if !ok {
		return
	}

	var request dto.UpdateCharacterRelationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		_ = c.Error(common.InvalidRequestWithValidationMessage(request, err))
		return
	}
	request.NovelID = novelID
	request.RelationID = relationID

	relation, err := service.useCase.UpdateCharacterRelation(c.Request.Context(), request)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: toCharacterRelationResponse(relation),
	})
}

// Delete 删除角色关系
// @Summary 删除角色关系
// @Description 物理删除指定小说下的角色关系。删除后该关系记录会从数据库移除，不会再出现在角色关系图 edges 中。
// @Tags character-relation
// @Produce json
// @Param id path int true "小说 ID"
// @Param relationId path int true "角色关系 ID"
// @Success 200 {object} common.Response "code = 0 表示删除成功"
// @Failure 400 {object} common.Response "请求参数错误，例如小说 ID 或关系 ID 不是正整数"
// @Failure 401 {object} common.Response "未登录"
// @Failure 404 {object} common.Response "小说或角色关系不存在"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /novels/{id}/character-relations/{relationId} [delete]
func (service *CharacterRelationService) Delete(c *gin.Context) {
	novelID, relationID, ok := parseCharacterRelationIDs(c)
	if !ok {
		return
	}

	if err := service.useCase.DeleteCharacterRelation(c.Request.Context(), novelID, relationID); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
	})
}

// parseCharacterRelationIDs 同时解析小说 ID 和角色关系 ID。
func parseCharacterRelationIDs(c *gin.Context) (int64, int64, bool) {
	novelID, ok := parseCharacterNovelID(c)
	if !ok {
		return 0, 0, false
	}

	relationID, err := strconv.ParseInt(c.Param("relationId"), 10, 64)
	if err != nil || relationID <= 0 {
		_ = c.Error(common.InvalidRequest)
		return 0, 0, false
	}

	return novelID, relationID, true
}

// toCharacterRelationGraphResponse 将业务数据组装成 Vue Flow 的 nodes 和 edges。
func toCharacterRelationGraphResponse(result *novelbiz.CharacterRelationGraphResult) characterRelationGraphResponse {
	layoutByCharacterID := make(map[int64]novelbiz.CharacterRelationNode, len(result.Nodes))
	for _, node := range result.Nodes {
		layoutByCharacterID[node.CharacterID] = node
	}

	nodes := make([]characterRelationGraphNodeResponse, 0, len(result.Characters))
	for _, character := range result.Characters {
		nodes = append(nodes, toCharacterRelationGraphNodeResponse(new(character), layoutByCharacterID[int64(character.ID)]))
	}

	edges := make([]characterRelationGraphEdgeResponse, 0, len(result.Relations))
	for _, relation := range result.Relations {
		edges = append(edges, toCharacterRelationGraphEdgeResponse(new(relation)))
	}

	return characterRelationGraphResponse{
		Nodes: nodes,
		Edges: edges,
	}
}

// toCharacterRelationGraphNodeResponse 将角色和可选布局转换成 Vue Flow 节点。
func toCharacterRelationGraphNodeResponse(character *novelbiz.Character, layout novelbiz.CharacterRelationNode) characterRelationGraphNodeResponse {
	nodeType := layout.NodeType
	if nodeType == "" {
		nodeType = "character"
	}

	nodeStatus := layout.Status
	if nodeStatus == 0 {
		nodeStatus = 1
	}

	var layoutID *int64
	if layout.ID > 0 {
		id := layout.ID
		layoutID = &id
	}

	return characterRelationGraphNodeResponse{
		ID:   strconv.FormatUint(uint64(character.ID), 10),
		Type: nodeType,
		Position: vueFlowPositionResponse{
			X: layout.PositionX,
			Y: layout.PositionY,
		},
		Width:  layout.Width,
		Height: layout.Height,
		Hidden: layout.Hidden,
		Locked: layout.Locked,
		Style:  rawJSON(layout.Style),
		Data: characterRelationGraphNodeDataResponse{
			LayoutID:         layoutID,
			NovelID:          character.NovelID,
			CharacterID:      character.ID,
			Name:             character.Name,
			Gender:           character.Gender,
			AppearanceImgURL: character.AppearanceImgURL,
			CharacterStatus:  character.Status,
			CharactersTags:   []string(character.CharactersTags),
			NodeStatus:       nodeStatus,
			Locked:           layout.Locked,
			ExtraData:        rawJSON(layout.ExtraData),
		},
	}
}

// toCharacterRelationGraphEdgeResponse 将角色关系转换成 Vue Flow 边。
func toCharacterRelationGraphEdgeResponse(relation *novelbiz.CharacterRelation) characterRelationGraphEdgeResponse {
	return characterRelationGraphEdgeResponse{
		ID:           strconv.FormatInt(relation.ID, 10),
		Source:       strconv.FormatInt(relation.SourceCharacterID, 10),
		Target:       strconv.FormatInt(relation.TargetCharacterID, 10),
		Type:         relation.EdgeType,
		Label:        relation.RelationLabel,
		Animated:     relation.Animated,
		SourceHandle: relation.SourceHandle,
		TargetHandle: relation.TargetHandle,
		Style:        rawJSON(relation.Style),
		Data: characterRelationGraphEdgeDataResponse{
			RelationID:        relation.ID,
			NovelID:           relation.NovelID,
			SourceCharacterID: relation.SourceCharacterID,
			TargetCharacterID: relation.TargetCharacterID,
			RelationType:      relation.RelationType,
			RelationLabel:     relation.RelationLabel,
			Description:       relation.Description,
			Direction:         relation.Direction,
			SortOrder:         relation.SortOrder,
			Status:            relation.Status,
			ExtraData:         rawJSON(relation.ExtraData),
		},
	}
}

// toCharacterRelationNodeLayoutResponse 将节点布局模型转换成接口响应。
func toCharacterRelationNodeLayoutResponse(node *novelbiz.CharacterRelationNode) characterRelationNodeLayoutResponse {
	return characterRelationNodeLayoutResponse{
		ID:          node.ID,
		NovelID:     node.NovelID,
		CharacterID: node.CharacterID,
		NodeType:    node.NodeType,
		PositionX:   node.PositionX,
		PositionY:   node.PositionY,
		Width:       node.Width,
		Height:      node.Height,
		Hidden:      node.Hidden,
		Locked:      node.Locked,
		Style:       rawJSON(node.Style),
		ExtraData:   rawJSON(node.ExtraData),
		Status:      node.Status,
		CreatedAt:   node.CreatedAt.Format("2006-01-02T15:04:05"),
		UpdatedAt:   node.UpdatedAt.Format("2006-01-02T15:04:05"),
	}
}

// toCharacterRelationResponse 将角色关系模型转换成管理接口响应。
func toCharacterRelationResponse(relation *novelbiz.CharacterRelation) characterRelationResponse {
	return characterRelationResponse{
		ID:                relation.ID,
		NovelID:           relation.NovelID,
		SourceCharacterID: relation.SourceCharacterID,
		TargetCharacterID: relation.TargetCharacterID,
		RelationType:      relation.RelationType,
		RelationLabel:     relation.RelationLabel,
		Description:       relation.Description,
		Direction:         relation.Direction,
		EdgeType:          relation.EdgeType,
		Animated:          relation.Animated,
		SourceHandle:      relation.SourceHandle,
		TargetHandle:      relation.TargetHandle,
		SortOrder:         relation.SortOrder,
		Style:             rawJSON(relation.Style),
		ExtraData:         rawJSON(relation.ExtraData),
		Status:            relation.Status,
		CreatedAt:         relation.CreatedAt.Format("2006-01-02T15:04:05"),
		UpdatedAt:         relation.UpdatedAt.Format("2006-01-02T15:04:05"),
	}
}

// rawJSON 保证 jsonb 空值返回空对象，前端无需额外处理 null。
func rawJSON(value datatypes.JSON) json.RawMessage {
	raw := json.RawMessage(value)
	if len(raw) == 0 || string(raw) == "null" {
		return json.RawMessage(`{}`)
	}

	return raw
}
