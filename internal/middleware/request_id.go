package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"

	"Novels_AI/backend/internal/pkg/common"

	"github.com/gin-gonic/gin"
)

const (
	// RequestIDKey 是日志字段、gin.Context 字段统一使用的请求 ID 名称。
	RequestIDKey = "request_id"
	// RequestIDHeader 把请求 ID 回写给客户端，方便客户端和服务端日志互相对齐。
	RequestIDHeader = "X-Request-Id"
)

type requestIDContextKey struct{}
type requestLoggerContextKey struct{}

// RequestID 为每个 HTTP 请求生成一个全局唯一 ID，并注入到响应头、gin.Context 和 request.Context。
//
// 后续处理链如果使用 slog.InfoContext(c.Request.Context(), ...) 打印日志，
// 配合 NewRequestIDLogHandler 会自动输出 request_id 字段；如果需要无 context 的 logger，
// 可以通过 RequestLogger(c) 取到已经绑定 request_id 的 logger。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID, err := newRequestID()
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, &common.Response{
				Code: http.StatusInternalServerError,
				Msg:  "生成请求ID失败",
			})
			return
		}

		ctx := WithRequestID(c.Request.Context(), requestID)
		ctx = context.WithValue(ctx, requestLoggerContextKey{}, slog.Default().With(slog.String(RequestIDKey, requestID)))
		c.Request = c.Request.WithContext(ctx)
		c.Set(RequestIDKey, requestID)
		c.Header(RequestIDHeader, requestID)
		c.Next()
	}
}

// WithRequestID 把请求 ID 写入 context，供 slog handler 和业务代码读取。
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey{}, requestID)
}

// RequestIDFromContext 从 context 中读取当前请求 ID。
func RequestIDFromContext(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(requestIDContextKey{}).(string)
	return requestID, ok && requestID != ""
}

// RequestLogger 返回绑定了当前 request_id 的 logger，适合仍然使用 logger.Info(...) 的代码。
func RequestLogger(c *gin.Context) *slog.Logger {
	logger, ok := c.Request.Context().Value(requestLoggerContextKey{}).(*slog.Logger)
	if ok {
		return logger
	}

	return slog.Default()
}

// NewRequestIDLogHandler 包装 slog.Handler，让所有携带请求 context 的日志自动追加 request_id。
func NewRequestIDLogHandler(handler slog.Handler) slog.Handler {
	return &requestIDLogHandler{handler: handler}
}

type requestIDLogHandler struct {
	handler slog.Handler
}

func (h *requestIDLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *requestIDLogHandler) Handle(ctx context.Context, record slog.Record) error {
	requestID, ok := RequestIDFromContext(ctx)
	if ok && !recordHasAttr(record, RequestIDKey) {
		record.AddAttrs(slog.String(RequestIDKey, requestID))
	}

	return h.handler.Handle(ctx, record)
}

func (h *requestIDLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &requestIDLogHandler{handler: h.handler.WithAttrs(attrs)}
}

func (h *requestIDLogHandler) WithGroup(name string) slog.Handler {
	return &requestIDLogHandler{handler: h.handler.WithGroup(name)}
}

// recordHasAttr 避免调用方已经显式传入 request_id 时重复输出同名字段。
func recordHasAttr(record slog.Record, key string) bool {
	found := false
	record.Attrs(func(attr slog.Attr) bool {
		if attr.Key == key {
			found = true
			return false
		}

		return true
	})

	return found
}

// newRequestID 使用 128 位安全随机数生成请求 ID，不引入额外依赖。
func newRequestID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes[:]), nil
}
