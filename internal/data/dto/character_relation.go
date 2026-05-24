package dto

// SaveCharacterRelationNodeLayoutsRequest 是批量保存角色关系图节点布局的请求参数。
type SaveCharacterRelationNodeLayoutsRequest struct {
	// 小说 ID 来自路径参数，不从请求体读取。
	NovelID int64 `json:"-"`
	// 需要保存布局的节点列表，至少传入一个节点。
	Nodes []CharacterRelationNodeLayoutRequest `json:"nodes" binding:"required,min=1,dive" validatormsg:"节点布局不能为空"`
}

// CharacterRelationNodeLayoutRequest 描述单个角色节点在 Vue Flow 画布上的展示状态。
type CharacterRelationNodeLayoutRequest struct {
	// 角色 ID，必须属于当前小说。
	CharacterID int64 `json:"characterId" binding:"required,gt=0" validatormsg:"角色ID不能为空"`
	// Vue Flow 节点类型，不传时默认 character。
	NodeType string `json:"type"`
	// 节点在画布上的 X 坐标。
	PositionX float64 `json:"positionX"`
	// 节点在画布上的 Y 坐标。
	PositionY float64 `json:"positionY"`
	// 节点宽度，前端需要记忆尺寸时传入。
	Width *float64 `json:"width"`
	// 节点高度，前端需要记忆尺寸时传入。
	Height *float64 `json:"height"`
	// 是否隐藏该节点。
	Hidden bool `json:"hidden"`
	// 是否锁定该节点位置。
	Locked bool `json:"locked"`
	// 节点展示样式，保存 Vue Flow 需要的样式扩展。
	Style JSONField `json:"style" swaggertype:"object"`
	// 节点扩展数据，用于保存前端未来新增的非固定字段。
	ExtraData JSONField `json:"extraData" swaggertype:"object"`
	// 节点状态：1 启用，2 停用。不传时默认 1。
	Status *int16 `json:"status" binding:"omitempty,oneof=1 2" validatormsg:"节点状态不正确"`
}

// CreateCharacterRelationRequest 是新增角色关系接口和业务层共用的请求参数。
type CreateCharacterRelationRequest struct {
	// 小说 ID 来自路径参数，不从请求体读取。
	NovelID int64 `json:"-"`
	// 起始角色 ID，对应 Vue Flow edge.source。
	SourceCharacterID int64 `json:"sourceCharacterId" binding:"required,gt=0" validatormsg:"起始角色ID不能为空"`
	// 目标角色 ID，对应 Vue Flow edge.target。
	TargetCharacterID int64 `json:"targetCharacterId" binding:"required,gt=0" validatormsg:"目标角色ID不能为空"`
	// 关系类型编码，例如 family、friend、enemy、lover、teacher、other。
	RelationType string `json:"relationType" binding:"required" validatormsg:"关系类型不能为空"`
	// 关系展示名称，例如 父女、好友、宿敌、师徒。
	RelationLabel string `json:"relationLabel" binding:"required" validatormsg:"关系名称不能为空"`
	// 关系说明，用于展示两个角色关系的背景或补充信息。
	Description string `json:"description"`
	// 关系方向：1 单向，2 双向或无向。不传时默认 2。
	Direction *int16 `json:"direction" binding:"omitempty,oneof=1 2" validatormsg:"关系方向不正确"`
	// Vue Flow 边类型，不传时默认 default。
	EdgeType string `json:"edgeType"`
	// 是否用动画线展示关系。
	Animated bool `json:"animated"`
	// Vue Flow 起始连接桩 ID。
	SourceHandle string `json:"sourceHandle"`
	// Vue Flow 目标连接桩 ID。
	TargetHandle string `json:"targetHandle"`
	// 排序值，前端可用于控制多条关系线的展示顺序。
	SortOrder int `json:"sortOrder"`
	// 关系线展示样式，保存 Vue Flow 需要的样式扩展。
	Style JSONField `json:"style" swaggertype:"object"`
	// 关系扩展数据，用于保存前端未来新增的非固定字段。
	ExtraData JSONField `json:"extraData" swaggertype:"object"`
	// 关系状态：1 启用，2 停用。不传时默认 1。
	Status *int16 `json:"status" binding:"omitempty,oneof=1 2" validatormsg:"关系状态不正确"`
}

// UpdateCharacterRelationRequest 是全量更新角色关系接口和业务层共用的请求参数。
type UpdateCharacterRelationRequest struct {
	// 小说 ID 来自路径参数，不从请求体读取。
	NovelID int64 `json:"-"`
	// 角色关系 ID 来自路径参数，不从请求体读取。
	RelationID int64 `json:"-"`
	// 起始角色 ID，对应 Vue Flow edge.source。
	SourceCharacterID int64 `json:"sourceCharacterId" binding:"required,gt=0" validatormsg:"起始角色ID不能为空"`
	// 目标角色 ID，对应 Vue Flow edge.target。
	TargetCharacterID int64 `json:"targetCharacterId" binding:"required,gt=0" validatormsg:"目标角色ID不能为空"`
	// 关系类型编码，例如 family、friend、enemy、lover、teacher、other。
	RelationType string `json:"relationType" binding:"required" validatormsg:"关系类型不能为空"`
	// 关系展示名称，例如 父女、好友、宿敌、师徒。
	RelationLabel string `json:"relationLabel" binding:"required" validatormsg:"关系名称不能为空"`
	// 关系说明，用于展示两个角色关系的背景或补充信息。
	Description string `json:"description"`
	// 关系方向：1 单向，2 双向或无向。
	Direction int16 `json:"direction" binding:"required,oneof=1 2" validatormsg:"关系方向不正确"`
	// Vue Flow 边类型，不传时默认 default。
	EdgeType string `json:"edgeType"`
	// 是否用动画线展示关系。
	Animated bool `json:"animated"`
	// Vue Flow 起始连接桩 ID。
	SourceHandle string `json:"sourceHandle"`
	// Vue Flow 目标连接桩 ID。
	TargetHandle string `json:"targetHandle"`
	// 排序值，前端可用于控制多条关系线的展示顺序。
	SortOrder int `json:"sortOrder"`
	// 关系线展示样式，保存 Vue Flow 需要的样式扩展。
	Style JSONField `json:"style" swaggertype:"object"`
	// 关系扩展数据，用于保存前端未来新增的非固定字段。
	ExtraData JSONField `json:"extraData" swaggertype:"object"`
	// 关系状态：1 启用，2 停用。
	Status int16 `json:"status" binding:"required,oneof=1 2" validatormsg:"关系状态不正确"`
}
