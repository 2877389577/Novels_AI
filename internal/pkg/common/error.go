package common

type SystemError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (err *SystemError) Error() string {
	return err.Msg
}

func NewSystemError(code int, msg string) *SystemError {
	return &SystemError{
		Code: code,
		Msg:  msg,
	}
}

var (
	NoInitialPassword  = NewSystemError(1000, "没有初始密码")
	PasswordAlreadySet = NewSystemError(1001, "密码已设置")
	PasswordIncorrect  = NewSystemError(1002, "密码错误")
)

var (
	InvalidRequest = NewSystemError(2000, "请求参数错误")
)
