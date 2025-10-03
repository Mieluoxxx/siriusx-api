package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Mieluoxxx/Siriusx-API/internal/mapping"
	"github.com/gin-gonic/gin"
)

// MappingHandler 映射处理器
type MappingHandler struct {
	service *mapping.Service
}

// NewMappingHandler 创建映射处理器实例
func NewMappingHandler(service *mapping.Service) *MappingHandler {
	return &MappingHandler{service: service}
}

// CreateMapping 创建映射
// POST /api/models/:id/mappings
func (h *MappingHandler) CreateMapping(c *gin.Context) {
	// 解析路径参数
	modelIDStr := c.Param("id")
	modelID, err := strconv.ParseUint(modelIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的模型ID"})
		return
	}

	// 解析请求体
	var req mapping.CreateMappingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	// 确保 UnifiedModelID 与路径参数一致
	req.UnifiedModelID = uint(modelID)

	// 调用服务层
	response, err := h.service.CreateMapping(req)
	if err != nil {
		c.JSON(h.handleMappingError(err), ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, response)
}

// ListMappings 查询模型的所有映射
// GET /api/models/:id/mappings
func (h *MappingHandler) ListMappings(c *gin.Context) {
	// 解析路径参数
	modelIDStr := c.Param("id")
	modelID, err := strconv.ParseUint(modelIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的模型ID"})
		return
	}

	// 解析查询参数
	includeProvider := c.Query("include_provider") == "true"

	// 调用服务层
	response, err := h.service.ListMappings(uint(modelID), includeProvider)
	if err != nil {
		c.JSON(h.handleMappingError(err), ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetMapping 根据 ID 获取映射
// GET /api/mappings/:id
func (h *MappingHandler) GetMapping(c *gin.Context) {
	// 解析路径参数
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的映射ID"})
		return
	}

	// 调用服务层
	response, err := h.service.GetMapping(uint(id))
	if err != nil {
		c.JSON(h.handleMappingError(err), ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// UpdateMapping 更新映射
// PUT /api/mappings/:id
func (h *MappingHandler) UpdateMapping(c *gin.Context) {
	// 解析路径参数
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的映射ID"})
		return
	}

	// 解析请求体
	var req mapping.UpdateMappingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	// 调用服务层
	response, err := h.service.UpdateMapping(uint(id), req)
	if err != nil {
		c.JSON(h.handleMappingError(err), ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// DeleteMapping 删除映射
// DELETE /api/mappings/:id
func (h *MappingHandler) DeleteMapping(c *gin.Context) {
	// 解析路径参数
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的映射ID"})
		return
	}

	// 调用服务层
	err = h.service.DeleteMapping(uint(id))
	if err != nil {
		c.JSON(h.handleMappingError(err), ErrorResponse{Error: err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// handleMappingError 处理映射相关的错误并返回对应的 HTTP 状态码
func (h *MappingHandler) handleMappingError(err error) int {
	switch {
	case errors.Is(err, mapping.ErrModelNotFound):
		return http.StatusNotFound
	case errors.Is(err, mapping.ErrMappingNotFound):
		return http.StatusNotFound
	case errors.Is(err, mapping.ErrMappingExists):
		return http.StatusConflict
	case errors.Is(err, mapping.ErrInvalidWeight):
		return http.StatusBadRequest
	case errors.Is(err, mapping.ErrInvalidPriority):
		return http.StatusBadRequest
	case errors.Is(err, mapping.ErrTargetModelEmpty):
		return http.StatusBadRequest
	case errors.Is(err, mapping.ErrProviderNotFound):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}