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
	NoInitialPassword = NewSystemError(1000, "没有初始密码")
)
