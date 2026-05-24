package common

import (
	"errors"
	"net/http"
	"reflect"

	"github.com/go-playground/validator/v10"
)

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

// InvalidRequestWithValidationMessage 从结构体校验错误中读取字段 validatormsg 标签，读取失败时回退到默认请求参数错误。
func InvalidRequestWithValidationMessage(request any, err error) *SystemError {
	msg := ValidationErrorMessage(request, err)
	if msg == "" {
		return InvalidRequest
	}

	return NewHTTPError(http.StatusBadRequest, InvalidRequest.Code, msg)
}

// ValidationErrorMessage 返回第一个校验失败字段上的 validatormsg 标签内容。
func ValidationErrorMessage(request any, err error) string {
	var fields validator.ValidationErrors
	if !errors.As(err, &fields) || len(fields) == 0 {
		return ""
	}

	requestType := reflect.TypeOf(request)
	if requestType == nil {
		return ""
	}
	if requestType.Kind() == reflect.Pointer {
		requestType = requestType.Elem()
	}
	if requestType.Kind() != reflect.Struct {
		return ""
	}

	structField, ok := requestType.FieldByName(fields[0].StructField())
	if !ok {
		return ""
	}

	return structField.Tag.Get("validatormsg")
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
	// CharacterNameRequired 表示角色名缺失，角色名是角色资料中唯一强制要求的业务字段。
	CharacterNameRequired = NewHTTPError(http.StatusBadRequest, 5000, "角色名称不能为空")
	// CharacterNotFound 表示当前小说下没有找到指定角色，避免跨小说误操作其他角色。
	CharacterNotFound = NewHTTPError(http.StatusNotFound, 5001, "角色不存在")
	// CharacterRelationNotFound 表示当前小说下没有找到指定角色关系。
	CharacterRelationNotFound = NewHTTPError(http.StatusNotFound, 5002, "角色关系不存在")
	// CharacterRelationExists 表示同一小说、同一组角色和同一关系类型下已经存在有效关系。
	CharacterRelationExists = NewHTTPError(http.StatusConflict, 5003, "角色关系已存在")
	// CharacterRelationSelfNotAllowed 表示不能把一个角色关联到自己。
	CharacterRelationSelfNotAllowed = NewHTTPError(http.StatusBadRequest, 5004, "不能创建角色和自己的关系")
)

var (
	// AIProviderNameRequired 表示 ai 提供商名称缺失。
	AIProviderNameRequired = NewHTTPError(http.StatusBadRequest, 6000, "AI提供商名称不能为空")
	// AIProviderTypeRequired 表示 ai 提供商类型缺失。
	AIProviderTypeRequired = NewHTTPError(http.StatusBadRequest, 6001, "AI提供商类型不能为空")
	// AIProviderBaseURLRequired 表示 ai 提供商基础 URL 缺失。
	AIProviderBaseURLRequired = NewHTTPError(http.StatusBadRequest, 6002, "AI提供商基础URL不能为空")
	// AIProviderAPIKeyRequired 表示 ai 提供商 API Key 缺失。
	AIProviderAPIKeyRequired = NewHTTPError(http.StatusBadRequest, 6003, "AI提供商API密钥不能为空")
	// AIProviderNotFound 表示没有找到指定 ai 提供商。
	AIProviderNotFound = NewHTTPError(http.StatusNotFound, 6004, "AI提供商不存在")
	// AIProviderAPIKeyDecryptFailed 表示已入库密钥无法按当前配置解密。
	AIProviderAPIKeyDecryptFailed = NewHTTPError(http.StatusInternalServerError, 6005, "AI提供商API密钥解密失败")
	// AIProviderModelsQueryFailed 表示按 OpenAI 兼容协议查询模型列表失败。
	AIProviderModelsQueryFailed = NewHTTPError(http.StatusBadGateway, 6006, "AI提供商模型查询失败")
	// AIProviderEnabledConflict 表示当前系统已经存在启用的 ai 提供商，业务层应阻止再启用其他供应商。
	AIProviderEnabledConflict = NewHTTPError(http.StatusOK, 6007, "已有启用的AI提供商")
)

var (
	InvalidRequest = NewHTTPError(http.StatusBadRequest, 2000, "请求参数错误")
)
