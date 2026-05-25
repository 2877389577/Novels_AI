package dto

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

// CreateChapterRequest 是新增章节接口和业务层共用的请求参数。
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

// UpdateChapterRequest 是全量更新章节接口和业务层共用的请求参数。
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
	// 用户输入的优化方向，可为空；为空时业务层按系统提示词做通用文笔优化。
	OptimizeDirection string `json:"optimizeDirection"`
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
