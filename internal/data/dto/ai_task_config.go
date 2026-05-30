package dto

// UpdateAITaskConfigRequest 是后台更新主动执行 AI 任务开关的请求参数。
type UpdateAITaskConfigRequest struct {
	// TaskCode 来自路径参数，不从请求体读取。
	TaskCode string `json:"-"`
	// IsEnabled 表示任务是否启用；使用指针是为了让 false 也能通过 required 校验。
	IsEnabled *bool `json:"isEnabled" binding:"required" validatormsg:"是否启用不能为空"`
}
