package dto

// PasswordRequest 是登录和初始化密码接口共用的请求参数。
type PasswordRequest struct {
	// 管理员密码，必填。
	Password string `json:"password" binding:"required" validatormsg:"密码不能为空"`
}
