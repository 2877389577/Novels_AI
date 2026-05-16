package middleware

import (
	"net/http"
	"sync"
	"time"

	"Novels_AI/backend/internal/pkg/common"

	"github.com/gin-gonic/gin"
)

type minuteRateLimiter struct {
	// mu 保护窗口时间和请求计数，避免并发请求同时更新导致限速失准。
	mu sync.Mutex
	// limit 是一个时间窗口内允许通过的最大请求数。
	limit int
	// windowStart 记录当前分钟窗口的开始时间。
	windowStart time.Time
	// count 记录当前窗口内已经放行的请求数量。
	count int
}

// RateLimiter 创建全局固定窗口限速中间件，requestsPerMinute 表示每分钟允许的请求数。
//
// 当配置值小于等于 0 时不启用限速，便于本地调试或临时关闭限流。
func RateLimiter(requestsPerMinute int) gin.HandlerFunc {
	limiter := &minuteRateLimiter{limit: requestsPerMinute}

	return func(c *gin.Context) {
		if limiter.allow() {
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusTooManyRequests, &common.Response{
			Code: http.StatusTooManyRequests,
			Msg:  "请求过于频繁，请稍后再试",
		})
	}
}

func (l *minuteRateLimiter) allow() bool {
	if l.limit <= 0 {
		return true
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	if l.windowStart.IsZero() || now.Sub(l.windowStart) >= time.Minute {
		l.windowStart = now
		l.count = 0
	}

	if l.count >= l.limit {
		return false
	}

	l.count++
	return true
}
