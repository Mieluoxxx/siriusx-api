package handlers

import (
	"errors"
	"math"
	"net/http"
	"strconv"

	"github.com/Mieluoxxx/Siriusx-API/internal/models"
	"github.com/Mieluoxxx/Siriusx-API/internal/provider"
	"github.com/gin-gonic/gin"
)

// ProviderHandler 供应商 HTTP 处理器
type ProviderHandler struct {
	service *provider.Service
}

// NewProviderHandler 创建 ProviderHandler 实例
func NewProviderHandler(service *provider.Service) *ProviderHandler {
	return &ProviderHandler{service: service}
}

// CreateProvider 创建供应商
// @Summary 创建供应商
// @Tags providers
// @Accept json
// @Produce json
// @Param provider body provider.CreateProviderRequest true "供应商信息"
// @Success 201 {object} provider.ProviderResponse
// @Failure 400 {object} provider.ErrorResponse
// @Failure 409 {object} provider.ErrorResponse
// @Router /api/providers [post]
func (h *ProviderHandler) CreateProvider(c *gin.Context) {
	var req provider.CreateProviderRequest

	// 绑定请求体
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, provider.ErrorResponse{
			Error: provider.ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid request parameters",
				Details: err.Error(),
			},
		})
		return
	}

	// 调用 Service 创建供应商
	prov, err := h.service.CreateProvider(req)
	if err != nil {
		if errors.Is(err, provider.ErrProviderNameExists) {
			c.JSON(http.StatusConflict, provider.ErrorResponse{
				Error: provider.ErrorDetail{
					Code:    "NAME_CONFLICT",
					Message: "Provider name already exists",
				},
			})
			return
		}
		if errors.Is(err, provider.ErrInvalidInput) || errors.Is(err, provider.ErrInvalidURL) {
			c.JSON(http.StatusBadRequest, provider.ErrorResponse{
				Error: provider.ErrorDetail{
					Code:    "VALIDATION_ERROR",
					Message: err.Error(),
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, provider.ErrorResponse{
			Error: provider.ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to create provider",
			},
		})
		return
	}

	// 返回响应（API Key 脱敏）
	c.JSON(http.StatusCreated, toProviderResponse(prov))
}

// GetProvider 获取单个供应商
// @Summary 获取单个供应商
// @Tags providers
// @Produce json
// @Param id path int true "供应商 ID"
// @Success 200 {object} provider.ProviderResponse
// @Failure 404 {object} provider.ErrorResponse
// @Router /api/providers/{id} [get]
func (h *ProviderHandler) GetProvider(c *gin.Context) {
	// 解析 ID
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, provider.ErrorResponse{
			Error: provider.ErrorDetail{
				Code:    "INVALID_ID",
				Message: "Invalid provider ID",
			},
		})
		return
	}

	// 查询供应商
	prov, err := h.service.GetProvider(uint(id))
	if err != nil {
		if errors.Is(err, provider.ErrProviderNotFound) {
			c.JSON(http.StatusNotFound, provider.ErrorResponse{
				Error: provider.ErrorDetail{
					Code:    "NOT_FOUND",
					Message: "Provider not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, provider.ErrorResponse{
			Error: provider.ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to get provider",
			},
		})
		return
	}

	// 返回响应
	c.JSON(http.StatusOK, toProviderResponse(prov))
}

// ListProviders 获取供应商列表
// @Summary 获取供应商列表
// @Tags providers
// @Produce json
// @Param page query int false "页码（默认 1）"
// @Param page_size query int false "每页数量（默认 10，最大 100）"
// @Success 200 {object} provider.ProviderListResponse
// @Router /api/providers [get]
func (h *ProviderHandler) ListProviders(c *gin.Context) {
	// 解析分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	// 查询供应商列表
	providers, total, err := h.service.ListProviders(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, provider.ErrorResponse{
			Error: provider.ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to list providers",
			},
		})
		return
	}

	// 构建响应
	data := make([]provider.ProviderResponse, len(providers))
	for i, p := range providers {
		data[i] = toProviderResponse(p)
	}

	// 计算总页数
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	c.JSON(http.StatusOK, provider.ProviderListResponse{
		Data: data,
		Pagination: provider.PaginationMeta{
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: totalPages,
		},
	})
}

// UpdateProvider 更新供应商
// @Summary 更新供应商
// @Tags providers
// @Accept json
// @Produce json
// @Param id path int true "供应商 ID"
// @Param provider body provider.UpdateProviderRequest true "更新信息"
// @Success 200 {object} provider.ProviderResponse
// @Failure 400 {object} provider.ErrorResponse
// @Failure 404 {object} provider.ErrorResponse
// @Failure 409 {object} provider.ErrorResponse
// @Router /api/providers/{id} [put]
func (h *ProviderHandler) UpdateProvider(c *gin.Context) {
	// 解析 ID
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, provider.ErrorResponse{
			Error: provider.ErrorDetail{
				Code:    "INVALID_ID",
				Message: "Invalid provider ID",
			},
		})
		return
	}

	var req provider.UpdateProviderRequest

	// 绑定请求体
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, provider.ErrorResponse{
			Error: provider.ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid request parameters",
				Details: err.Error(),
			},
		})
		return
	}

	// 调用 Service 更新供应商
	prov, err := h.service.UpdateProvider(uint(id), req)
	if err != nil {
		if errors.Is(err, provider.ErrProviderNotFound) {
			c.JSON(http.StatusNotFound, provider.ErrorResponse{
				Error: provider.ErrorDetail{
					Code:    "NOT_FOUND",
					Message: "Provider not found",
				},
			})
			return
		}
		if errors.Is(err, provider.ErrProviderNameExists) {
			c.JSON(http.StatusConflict, provider.ErrorResponse{
				Error: provider.ErrorDetail{
					Code:    "NAME_CONFLICT",
					Message: "Provider name already exists",
				},
			})
			return
		}
		if errors.Is(err, provider.ErrInvalidInput) || errors.Is(err, provider.ErrInvalidURL) {
			c.JSON(http.StatusBadRequest, provider.ErrorResponse{
				Error: provider.ErrorDetail{
					Code:    "VALIDATION_ERROR",
					Message: err.Error(),
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, provider.ErrorResponse{
			Error: provider.ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to update provider",
			},
		})
		return
	}

	// 返回响应
	c.JSON(http.StatusOK, toProviderResponse(prov))
}

// DeleteProvider 删除供应商（软删除）
// @Summary 删除供应商
// @Tags providers
// @Param id path int true "供应商 ID"
// @Success 204 "No Content"
// @Failure 404 {object} provider.ErrorResponse
// @Router /api/providers/{id} [delete]
func (h *ProviderHandler) DeleteProvider(c *gin.Context) {
	// 解析 ID
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, provider.ErrorResponse{
			Error: provider.ErrorDetail{
				Code:    "INVALID_ID",
				Message: "Invalid provider ID",
			},
		})
		return
	}

	// 调用 Service 删除供应商
	if err := h.service.DeleteProvider(uint(id)); err != nil {
		if errors.Is(err, provider.ErrProviderNotFound) {
			c.JSON(http.StatusNotFound, provider.ErrorResponse{
				Error: provider.ErrorDetail{
					Code:    "NOT_FOUND",
					Message: "Provider not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, provider.ErrorResponse{
			Error: provider.ErrorDetail{
				Code:    "INTERNAL_ERROR",
				Message: "Failed to delete provider",
			},
		})
		return
	}

	// 返回 204 No Content
	c.Status(http.StatusNoContent)
}

// toProviderResponse 将 Provider 模型转换为响应（API Key 脱敏）
func toProviderResponse(p *models.Provider) provider.ProviderResponse {
	return provider.ProviderResponse{
		ID:           p.ID,
		Name:         p.Name,
		BaseURL:      p.BaseURL,
		APIKey:       provider.MaskAPIKey(p.APIKey),
		Enabled:      p.Enabled,
		Priority:     p.Priority,
		HealthStatus: p.HealthStatus,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
	}
}
