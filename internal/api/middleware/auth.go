package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Mieluoxxx/Siriusx-API/internal/token"
	"github.com/gin-gonic/gin"
)

// TokenAuthMiddleware Token 验证中间件
// 用于验证 API 请求中的 Bearer Token
func TokenAuthMiddleware(tokenService *token.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 提取 Authorization 头
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "MISSING_AUTH_HEADER",
					"message": "Missing authorization header",
				},
			})
			c.Abort()
			return
		}

		// 2. 解析 Bearer Token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" || strings.TrimSpace(parts[1]) == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "INVALID_AUTH_FORMAT",
					"message": "Invalid authorization format. Expected: Bearer <token>",
				},
			})
			c.Abort()
			return
		}

		tokenValue := parts[1]

		// 3. 验证 Token
		tok, err := tokenService.ValidateToken(tokenValue)
		if err != nil {
			handleAuthError(c, err)
			c.Abort()
			return
		}

		// 4. 将 Token 信息存入 Context
		c.Set("token_id", tok.ID)
		c.Set("token", tok)

		c.Next()
	}
}

// handleAuthError 处理认证错误
func handleAuthError(c *gin.Context, err error) {
	var code, message string

	switch {
	case errors.Is(err, token.ErrInvalidToken):
		code = "INVALID_TOKEN"
		message = "Invalid token"
	case errors.Is(err, token.ErrTokenDisabled):
		code = "TOKEN_DISABLED"
		message = "Token disabled"
	case errors.Is(err, token.ErrTokenExpired):
		code = "TOKEN_EXPIRED"
		message = "Token expired"
	default:
		code = "AUTH_ERROR"
		message = "Authentication failed"
	}

	c.JSON(http.StatusUnauthorized, gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}
