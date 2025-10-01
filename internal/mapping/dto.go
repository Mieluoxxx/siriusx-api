package mapping

import (
	"time"

	"github.com/Mieluoxxx/Siriusx-API/internal/models"
)

// CreateModelRequest 创建统一模型请求
type CreateModelRequest struct {
	Name        string `json:"name" binding:"required,max=100"`
	Description string `json:"description" binding:"max=500"`
}

// UpdateModelRequest 更新统一模型请求
type UpdateModelRequest struct {
	Name        *string `json:"name" binding:"omitempty,max=100"`
	Description *string `json:"description" binding:"omitempty,max=500"`
}

// ModelResponse 统一模型响应
type ModelResponse struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ListModelsRequest 查询模型列表请求
type ListModelsRequest struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Search   string `form:"search" binding:"omitempty,max=100"`
}

// ListModelsResponse 查询模型列表响应
type ListModelsResponse struct {
	Models     []*ModelResponse `json:"models"`
	Pagination PaginationInfo   `json:"pagination"`
}

// PaginationInfo 分页信息
type PaginationInfo struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// ToModelResponse 将模型实体转换为响应对象
func ToModelResponse(model *models.UnifiedModel) *ModelResponse {
	return &ModelResponse{
		ID:          model.ID,
		Name:        model.Name,
		Description: model.Description,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}
}

// ToModelResponseList 将模型实体列表转换为响应对象列表
func ToModelResponseList(models []*models.UnifiedModel) []*ModelResponse {
	responses := make([]*ModelResponse, len(models))
	for i, model := range models {
		responses[i] = ToModelResponse(model)
	}
	return responses
}