package dto

import "encoding/json"

// CreateNovelRequest 是新增小说接口和业务层共用的请求参数。
type CreateNovelRequest struct {
	// 书名，必填。
	Title string `json:"title" binding:"required" validatormsg:"书名不能为空"`
	// 小说简介。
	Intro string `json:"intro"`
	// 小说大纲，可为空。
	NovelOutline string `json:"novelOutline"`
	// 小说作者。
	AuthorName string `json:"authorName"`
	// 小说封面 URL。
	CoverURL string `json:"coverUrl"`
	// 小说元数据（标签信息）。
	Metadata JSONField `json:"metadata" swaggertype:"object"`
}

// UpdateNovelRequest 是全量更新小说接口和业务层共用的请求参数。
type UpdateNovelRequest struct {
	// 小说 ID，必填。
	ID uint `json:"id" binding:"required" validatormsg:"小说ID不能为空"`
	// 书名，必填。
	Title string `json:"title" binding:"required" validatormsg:"书名不能为空"`
	// 小说简介。
	Intro string `json:"intro"`
	// 小说大纲，可为空。
	NovelOutline string `json:"novelOutline"`
	// 小说作者。
	AuthorName string `json:"authorName"`
	// 小说封面 URL。
	CoverURL string `json:"coverUrl"`
	// 小说字数。
	WordCount int64 `json:"wordCount" binding:"gte=0" validatormsg:"小说字数不能小于0"`
	// 小说元数据（标签信息）。
	Metadata JSONField `json:"metadata" swaggertype:"object"`
}

// SaveNovelOutlineRequest 是小说大纲保存接口和业务层共用的请求参数。
type SaveNovelOutlineRequest struct {
	// 小说 ID 来自路径参数，不从请求体读取。
	NovelID int64 `json:"-"`
	// 小说大纲内容，可为空；传空字符串表示清空大纲。
	NovelOutline string `json:"novelOutline"`
}

// CreateChapterRequest 是新增章节接口和业务层共用的请求参数；章节剧情总结会在章节保存成功后通过事件触发。
type CreateChapterRequest struct {
	// 小说 ID 来自路径参数，不从请求体读取。
	NovelID int64 `json:"-"`
	// 章节编号，必填且大于 0。
	ChapterNo int `json:"chapterNo" binding:"required,gt=0" validatormsg:"章节编号不正确"`
	// 章节标题，必填。
	Title string `json:"title" binding:"required" validatormsg:"章节标题不能为空"`
	// 章节内容，必填。
	Content string `json:"content" binding:"required" validatormsg:"章节内容不能为空"`
	// 章节字数，不能小于 0。
	WordCount int `json:"wordCount" binding:"gte=0" validatormsg:"章节字数不能小于0"`
}

// UpdateChapterRequest 是全量更新章节接口和业务层共用的请求参数；章节剧情总结会在章节保存成功后通过事件重新触发。
type UpdateChapterRequest struct {
	// 小说 ID 来自路径参数，不从请求体读取。
	NovelID int64 `json:"-"`
	// 章节 ID 来自路径参数，不从请求体读取。
	ChapterID uint `json:"-"`
	// 章节编号，必填且大于 0。
	ChapterNo int `json:"chapterNo" binding:"required,gt=0" validatormsg:"章节编号不正确"`
	// 章节标题，必填。
	Title string `json:"title" binding:"required" validatormsg:"章节标题不能为空"`
	// 章节内容，必填。
	Content string `json:"content" binding:"required" validatormsg:"章节内容不能为空"`
	// 章节字数，不能小于 0。
	WordCount int `json:"wordCount" binding:"gte=0" validatormsg:"章节字数不能小于0"`
}

// OptimizeNovelContentRequest 是小说正文优化接口和业务层共用的请求参数。
type OptimizeNovelContentRequest struct {
	// 小说 ID 来自路径参数，不从请求体读取。
	NovelID int64 `json:"-"`
	// 大模型名称来自 query 参数，不占用请求体中的业务字段。
	ModelName string `json:"-"`
	// 用户在编辑器中选中的小说原文段落，必填。
	SelectedContent string `json:"selectedContent" binding:"required" validatormsg:"优化内容不能为空"`
	// 用户输入的优化方向，可为空；为空时业务层默认做文笔优化，非空时交由顶层 Agent 判断润色或扩写。
	OptimizeDirection string `json:"optimizeDirection"`
}

// SaveMindMapRequest 是保存整张 SimpleMindMap 思维导图接口和业务层共用的请求参数。
//
// 推荐前端使用方式：
// 1. 进入思维导图页面时先调用 GET /mind-map，然后把返回的 mindMapData 交给 SimpleMindMap setData。
// 2. 用户在画布里连续编辑节点、备注、概要、关联线、外框、样式或布局时，可以先只维护前端内存状态。
// 3. 用户点击“保存”、离开页面前确认保存，或前端做防抖自动保存时，再提交 mindMap.getData(true) 到本接口。
type SaveMindMapRequest struct {
	// 小说 ID 来自路径参数，不从请求体读取。
	NovelID int64 `json:"-"`
	// SimpleMindMap 的完整 JSON 数据，推荐前端传 getData(true) 的结果；这样主题、布局、视图和插件字段都会一起保存。
	MindMapData json.RawMessage `json:"mindMapData" binding:"required" swaggertype:"object" validatormsg:"思维导图数据不能为空"`
}

// CreateMindMapNodeRequest 是新增思维导图节点接口和业务层共用的请求参数。
//
// 这个接口适合“新增节点后立刻落库”的交互；如果前端已经用 SimpleMindMap 在本地完成了新增，
// 也可以不调用本接口，直接在合适时机调用整图保存接口提交最新 getData(true)。
type CreateMindMapNodeRequest struct {
	// 小说 ID 来自路径参数，不从请求体读取。
	NovelID int64 `json:"-"`
	// 父节点 uid，新节点会追加到该节点 children 下。
	ParentUID string `json:"parentUid" binding:"required" validatormsg:"父节点不能为空"`
	// 插入位置；不传或越界时追加到末尾。
	Index *int `json:"index"`
	// SimpleMindMap 节点 JSON，包含 data 和可选 children。
	Node json.RawMessage `json:"node" binding:"required" swaggertype:"object" validatormsg:"节点数据不能为空"`
}

// UpdateMindMapNodeRequest 是更新思维导图节点 data 字段接口和业务层共用的请求参数。
//
// 这个接口只替换指定节点的 data，不触碰 children 子树。适合单节点属性面板保存，
// 例如只改节点文本、备注 note、概要 generalization、关联线 associativeLine*、图片或超链接。
// 如果前端一次性修改了多处结构或布局，仍然建议直接调用整图保存接口。
type UpdateMindMapNodeRequest struct {
	// 小说 ID 来自路径参数，不从请求体读取。
	NovelID int64 `json:"-"`
	// 节点 uid 来自路径参数，不从请求体读取。
	NodeUID string `json:"-"`
	// SimpleMindMap 节点 data 字段，备注、概要、关联线等节点附加数据都保存在这里。
	Data json.RawMessage `json:"data" binding:"required" swaggertype:"object" validatormsg:"节点数据不能为空"`
}

// CreateCharacterRequest 是新增角色接口和业务层共用的请求参数。
type CreateCharacterRequest struct {
	// 小说 ID 来自路径参数，不从请求体读取。
	NovelID int64 `json:"-"`
	// 角色名称，必填。
	Name string `json:"name" binding:"required" validatormsg:"角色名称不能为空"`
	// 角色性别。
	Gender string `json:"gender"`
	// 角色介绍。
	Intro string `json:"intro"`
	// 角色性格。
	Personality string `json:"personality"`
	// 角色外貌。
	Appearance string `json:"appearance"`
	// 角色背景。
	Background string `json:"background"`
	// 角色能力。
	Ability string `json:"ability"`
	// 角色动机。
	Motivation string `json:"motivation"`
	// 角色剧情方向。
	PlotDirection string `json:"plotDirection"`
	// 首次出现章节 ID。
	FirstAppearanceChapterID *int64 `json:"firstAppearanceChapterId"`
	// 角色形象图 URL。
	AppearanceImgURL string `json:"appearanceImgUrl"`
	// 角色状态：1 在线，2 下线。
	Status *int16 `json:"status"`
	// 角色标签。
	CharactersTags []string `json:"charactersTags"`
}

// UpdateCharacterRequest 是全量更新角色接口和业务层共用的请求参数。
type UpdateCharacterRequest struct {
	// 小说 ID 来自路径参数，不从请求体读取。
	NovelID int64 `json:"-"`
	// 角色 ID 来自路径参数，不从请求体读取。
	CharacterID uint `json:"-"`
	// 角色名称，必填。
	Name string `json:"name" binding:"required" validatormsg:"角色名称不能为空"`
	// 角色性别。
	Gender string `json:"gender"`
	// 角色介绍。
	Intro string `json:"intro"`
	// 角色性格。
	Personality string `json:"personality"`
	// 角色外貌。
	Appearance string `json:"appearance"`
	// 角色背景。
	Background string `json:"background"`
	// 角色能力。
	Ability string `json:"ability"`
	// 角色动机。
	Motivation string `json:"motivation"`
	// 角色剧情方向。
	PlotDirection string `json:"plotDirection"`
	// 首次出现章节 ID。
	FirstAppearanceChapterID *int64 `json:"firstAppearanceChapterId"`
	// 角色形象图 URL。
	AppearanceImgURL string `json:"appearanceImgUrl"`
	// 角色状态：1 在线，2 下线。
	Status int16 `json:"status"`
	// 角色标签。
	CharactersTags []string `json:"charactersTags"`
}
