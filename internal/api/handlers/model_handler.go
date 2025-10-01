package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Mieluoxxx/Siriusx-API/internal/mapping"
	"github.com/gin-gonic/gin"
)

// ModelHandler 统一模型 HTTP 处理器
type ModelHandler struct {
	service *mapping.Service
}

// NewModelHandler 创建 ModelHandler 实例
func NewModelHandler(service *mapping.Service) *ModelHandler {
	return &ModelHandler{service: service}
}

// ErrorResponse 错误响应格式
type ErrorResponse struct {
	Error string `json:"error"`
}

// CreateModel 创建统一模型
// @Summary 创建统一模型
// @Tags models
// @Accept json
// @Produce json
// @Param model body mapping.CreateModelRequest true "模型信息"
// @Success 201 {object} mapping.ModelResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Router /api/models [post]
func (h *ModelHandler) CreateModel(c *gin.Context) {
	var req mapping.CreateModelRequest

	// 绑定请求体
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "参数验证失败: " + err.Error(),
		})
		return
	}

	// 调用 Service 创建模型
	model, err := h.service.CreateModel(req)
	if err != nil {
		status := h.getErrorStatus(err)
		c.JSON(status, ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, model)
}

// ListModels 查询模型列表
// @Summary 查询模型列表
// @Tags models
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param search query string false "搜索关键词"
// @Success 200 {object} mapping.ListModelsResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/models [get]
func (h *ModelHandler) ListModels(c *gin.Context) {
	var req mapping.ListModelsRequest

	// 绑定查询参数
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "参数验证失败: " + err.Error(),
		})
		return
	}

	// 调用 Service 查询模型列表
	response, err := h.service.ListModels(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetModel 根据 ID 获取模型
// @Summary 获取单个模型
// @Tags models
// @Accept json
// @Produce json
// @Param id path int true "模型ID"
// @Success 200 {object} mapping.ModelResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/models/{id} [get]
func (h *ModelHandler) GetModel(c *gin.Context) {
	// 解析 ID 参数
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "无效的模型ID",
		})
		return
	}

	// 调用 Service 获取模型
	model, err := h.service.GetModel(uint(id))
	if err != nil {
		status := h.getErrorStatus(err)
		c.JSON(status, ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model)
}

// UpdateModel 更新模型
// @Summary 更新模型
// @Tags models
// @Accept json
// @Produce json
// @Param id path int true "模型ID"
// @Param model body mapping.UpdateModelRequest true "更新的模型信息"
// @Success 200 {object} mapping.ModelResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Router /api/models/{id} [put]
func (h *ModelHandler) UpdateModel(c *gin.Context) {
	// 解析 ID 参数
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "无效的模型ID",
		})
		return
	}

	var req mapping.UpdateModelRequest

	// 绑定请求体
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "参数验证失败: " + err.Error(),
		})
		return
	}

	// 调用 Service 更新模型
	model, err := h.service.UpdateModel(uint(id), req)
	if err != nil {
		status := h.getErrorStatus(err)
		c.JSON(status, ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model)
}

// DeleteModel 删除模型
// @Summary 删除模型
// @Tags models
// @Accept json
// @Produce json
// @Param id path int true "模型ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/models/{id} [delete]
func (h *ModelHandler) DeleteModel(c *gin.Context) {
	// 解析 ID 参数
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "无效的模型ID",
		})
		return
	}

	// 调用 Service 删除模型
	err = h.service.DeleteModel(uint(id))
	if err != nil {
		status := h.getErrorStatus(err)
		c.JSON(status, ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// getErrorStatus 根据错误类型返回适当的 HTTP 状态码
func (h *ModelHandler) getErrorStatus(err error) int {
	switch {
	case errors.Is(err, mapping.ErrModelNotFound):
		return http.StatusNotFound
	case errors.Is(err, mapping.ErrModelNameExists):
		return http.StatusConflict
	case errors.Is(err, mapping.ErrInvalidModelName),
		errors.Is(err, mapping.ErrModelNameEmpty),
		errors.Is(err, mapping.ErrModelNameTooLong),
		errors.Is(err, mapping.ErrDescriptionTooLong):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}