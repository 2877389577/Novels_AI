package common

import "net/http"

type SystemError struct {
	// Code 是业务错误码，统一写入响应体，方便前端按业务场景处理。
	Code int `json:"code"`
	// Msg 是展示给调用方的错误提示，保持和业务错误定义在同一处维护。
	Msg string `json:"msg"`
	// HTTPStatus 只用于中间件选择 HTTP 响应状态，不暴露到 JSON 响应体。
	HTTPStatus int `json:"-"`
}

func (err *SystemError) Error() string {
	return err.Msg
}

// StatusCode 返回错误对应的 HTTP 状态码，避免 service 层再聚合业务错误和响应状态。
func (err *SystemError) StatusCode() int {
	if err.HTTPStatus == 0 {
		return http.StatusInternalServerError
	}

	return err.HTTPStatus
}

func NewSystemError(code int, msg string) *SystemError {
	return NewHTTPError(http.StatusInternalServerError, code, msg)
}

// NewHTTPError 创建带 HTTP 状态码的业务错误，Code 和 Msg 仍然作为统一响应体返回给前端。
func NewHTTPError(statusCode, code int, msg string) *SystemError {
	return &SystemError{
		Code:       code,
		Msg:        msg,
		HTTPStatus: statusCode,
	}
}

var (
	NoInitialPassword  = NewSystemError(1000, "没有初始密码")
	PasswordAlreadySet = NewSystemError(1001, "密码已设置")
	PasswordIncorrect  = NewSystemError(1002, "密码错误")
)

var (
	NovelNotFound           = NewHTTPError(http.StatusNotFound, 4000, "小说不存在")
	ChapterTitleRequired    = NewHTTPError(http.StatusBadRequest, 3000, "章节标题不能为空")
	ChapterContentRequired  = NewHTTPError(http.StatusBadRequest, 3001, "章节内容不能为空")
	ChapterNoInvalid        = NewHTTPError(http.StatusBadRequest, 3002, "章节编号不正确")
	ChapterWordCountInvalid = NewHTTPError(http.StatusBadRequest, 3003, "章节字数不能小于0")
	ChapterNoExists         = NewHTTPError(http.StatusConflict, 3004, "章节编号已存在")
	ChapterNotFound         = NewHTTPError(http.StatusNotFound, 3005, "章节不存在")
)

var (
	InvalidRequest = NewHTTPError(http.StatusBadRequest, 2000, "请求参数错误")
)
