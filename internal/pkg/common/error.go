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
	// ChapterPlotAnalysisModelNotFound 表示当前启用的 AI 提供商没有配置可用于章节剧情解析的模型。
	ChapterPlotAnalysisModelNotFound = NewHTTPError(http.StatusBadGateway, 3006, "未配置章节剧情解析模型")
	// ChapterPlotAnalysisNoResult 表示模型没有按要求调用章节剧情解析 tool 返回结构化结果。
	ChapterPlotAnalysisNoResult = NewHTTPError(http.StatusBadGateway, 3007, "AI未返回章节剧情解析结果")
	// ChapterPlotAnalysisNotFound 表示指定章节还没有生成可查询的剧情总结；异步生成未完成是正常业务状态，HTTP 层返回 200。
	ChapterPlotAnalysisNotFound = NewHTTPError(http.StatusOK, 3008, "章节剧情总结不存在")
)

var (
	// NovelContentOptimizeNoResult 表示模型没有按要求调用正文优化 tool 返回结构化结果。
	NovelContentOptimizeNoResult = NewHTTPError(http.StatusBadGateway, 7000, "AI未返回内容优化结果")
	// NovelContentOptimizePromptReadFailed 表示服务端无法读取正文优化系统提示词文件。
	NovelContentOptimizePromptReadFailed = NewHTTPError(http.StatusInternalServerError, 7001, "读取小说内容优化提示词失败")
	// NovelContentOptimizeRouteNoResult 表示顶层 Agent 没有按要求返回子 Agent 分流结果。
	NovelContentOptimizeRouteNoResult = NewHTTPError(http.StatusBadGateway, 7002, "AI未返回内容优化任务分析结果")
)

var (
	// MindMapNotFound 表示指定小说还没有保存过思维导图数据。
	MindMapNotFound = NewHTTPError(http.StatusNotFound, 8000, "思维导图不存在")
	// MindMapNodeNotFound 表示思维导图中没有找到指定节点。
	MindMapNodeNotFound = NewHTTPError(http.StatusNotFound, 8001, "思维导图节点不存在")
	// MindMapNodeExists 表示思维导图中已经存在相同 uid 的节点。
	MindMapNodeExists = NewHTTPError(http.StatusConflict, 8002, "思维导图节点已存在")
	// MindMapRootDeleteNotAllowed 表示不能删除思维导图根节点。
	MindMapRootDeleteNotAllowed = NewHTTPError(http.StatusBadRequest, 8003, "不能删除思维导图根节点")
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
	// AITaskConfigNotFound 表示没有找到指定主动执行 AI 任务配置。
	AITaskConfigNotFound = NewHTTPError(http.StatusNotFound, 10000, "AI任务配置不存在")
)

var (
	// ImageAIProviderNotFound 表示没有找到指定生图 AI 提供商。
	ImageAIProviderNotFound = NewHTTPError(http.StatusNotFound, 9000, "生图AI提供商不存在")
	// ImageAIProviderNotEnabled 表示当前系统没有启用的生图 AI 提供商，图片生成不可执行。
	ImageAIProviderNotEnabled = NewHTTPError(http.StatusNotFound, 9001, "未启用生图AI提供商")
	// ImageAIProviderDefaultModelNotConfigured 表示启用的生图 AI 提供商没有配置默认生图模型。
	ImageAIProviderDefaultModelNotConfigured = NewHTTPError(http.StatusBadGateway, 9002, "未配置默认生图模型")
	// ImageAIProviderAPIKeyDecryptFailed 表示已入库生图密钥无法按当前配置解密。
	ImageAIProviderAPIKeyDecryptFailed = NewHTTPError(http.StatusInternalServerError, 9003, "生图AI提供商API密钥解密失败")
	// ImageGenerationFailed 表示调用上游生图接口或下载上游图片失败。
	ImageGenerationFailed = NewHTTPError(http.StatusBadGateway, 9004, "图片生成失败")
	// ImageGenerationNoImage 表示上游生图接口没有返回可处理的图片 URL 或 base64。
	ImageGenerationNoImage = NewHTTPError(http.StatusBadGateway, 9005, "AI未返回图片")
	// ImageGenerationUploadFailed 表示图片已生成但写入对象存储失败。
	ImageGenerationUploadFailed = NewHTTPError(http.StatusInternalServerError, 9006, "AI生成图片上传失败")
	// ImageAIProviderEnabledConflict 表示当前系统已经存在启用的生图 AI 提供商，业务层应阻止再启用其他供应商。
	ImageAIProviderEnabledConflict = NewHTTPError(http.StatusOK, 9007, "已有启用的生图AI提供商")
)

var (
	InvalidRequest = NewHTTPError(http.StatusBadRequest, 2000, "请求参数错误")
)
