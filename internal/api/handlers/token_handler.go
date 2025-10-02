package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Mieluoxxx/Siriusx-API/internal/token"
	"github.com/gin-gonic/gin"
)

// TokenHandler Token HTTP 处理器
type TokenHandler struct {
	service *token.Service
}

// NewTokenHandler 创建 TokenHandler 实例
func NewTokenHandler(service *token.Service) *TokenHandler {
	return &TokenHandler{service: service}
}

// CreateToken 创建 Token
// @Summary 创建 Token
// @Tags tokens
// @Accept json
// @Produce json
// @Param token body token.CreateTokenRequest true "Token 信息"
// @Success 201 {object} token.TokenDTO
// @Failure 400 {object} ErrorResponse
// @Router /api/tokens [post]
func (h *TokenHandler) CreateToken(c *gin.Context) {
	var req token.CreateTokenRequest

	// 绑定请求体
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Invalid request parameters",
				"details": err.Error(),
			},
		})
		return
	}

	// 调用 Service 创建 Token
	tok, err := h.service.CreateToken(req.Name, req.ExpiresAt, req.CustomToken)
	if err != nil {
		h.handleTokenError(c, err)
		return
	}

	// 返回响应（包含完整 Token，仅此一次）
	dto := token.ToTokenDTO(tok, true)
	c.JSON(http.StatusCreated, dto)
}

// ListTokens 获取 Token 列表
// @Summary 获取 Token 列表
// @Tags tokens
// @Produce json
// @Success 200 {array} token.TokenDTO
// @Router /api/tokens [get]
func (h *TokenHandler) ListTokens(c *gin.Context) {
	// 调用 Service 获取 Token 列表
	tokens, err := h.service.ListTokens()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to retrieve tokens",
			},
		})
		return
	}

	// 转换为 DTO（脱敏显示）
	dtos := make([]*token.TokenDTO, len(tokens))
	for i, tok := range tokens {
		dtos[i] = token.ToTokenDTO(tok, false) // false: 不显示完整 Token
	}

	c.JSON(http.StatusOK, dtos)
}

// GetToken 获取单个 Token（包含完整 Token 值）
// @Summary 获取单个 Token 详情
// @Tags tokens
// @Produce json
// @Param id path int true "Token ID"
// @Success 200 {object} token.TokenDTO
// @Failure 404 {object} ErrorResponse
// @Router /api/tokens/{id} [get]
func (h *TokenHandler) GetToken(c *gin.Context) {
	// 解析 ID
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_ID",
				"message": "Invalid token ID",
			},
		})
		return
	}

	// 调用 Service 获取 Token
	tok, err := h.service.GetToken(uint(id))
	if err != nil {
		h.handleTokenError(c, err)
		return
	}

	// 返回完整 Token
	dto := token.ToTokenDTO(tok, true)
	c.JSON(http.StatusOK, dto)
}

// DeleteToken 删除 Token
// @Summary 删除 Token
// @Tags tokens
// @Param id path int true "Token ID"
// @Success 204 "No Content"
// @Failure 404 {object} ErrorResponse
// @Router /api/tokens/{id} [delete]
func (h *TokenHandler) DeleteToken(c *gin.Context) {
	// 解析 ID
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_ID",
				"message": "Invalid token ID",
			},
		})
		return
	}

	// 调用 Service 删除 Token
	if err := h.service.DeleteToken(uint(id)); err != nil {
		h.handleTokenError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// handleTokenError 处理 Token 相关错误
func (h *TokenHandler) handleTokenError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, token.ErrTokenNotFound):
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "TOKEN_NOT_FOUND",
				"message": "Token not found",
			},
		})
	case errors.Is(err, token.ErrInvalidExpiresAt):
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_EXPIRES_AT",
				"message": "Expiration time must be in the future",
			},
		})
	case errors.Is(err, token.ErrTokenValueExists):
		c.JSON(http.StatusConflict, gin.H{
			"error": gin.H{
				"code":    "TOKEN_CONFLICT",
				"message": "Token already exists",
			},
		})
	case errors.Is(err, token.ErrInvalidCustomToken):
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_CUSTOM_TOKEN",
				"message": "Custom token must start with 'sk-' and be at least 8 characters",
			},
		})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Internal server error",
			},
		})
	}
}
