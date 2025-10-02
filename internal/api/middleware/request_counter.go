package middleware

import (
	"github.com/Mieluoxxx/Siriusx-API/internal/stats"
	"github.com/gin-gonic/gin"
)

// RequestCounterMiddleware 请求计数中间件
// 统计所有通过的请求
func RequestCounterMiddleware(counter *stats.RequestCounter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 增加请求计数
		counter.Increment()

		// 继续处理请求
		c.Next()
	}
}
