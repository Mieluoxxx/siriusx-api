package mapping

import (
	"time"

	"github.com/Mieluoxxx/Siriusx-API/internal/models"
)

// CreateModelRequest 创建统一模型请求
type CreateModelRequest struct {
	Name        string `json:"name" binding:"required,max=100"`
	DisplayName string `json:"display_name" binding:"omitempty,max=200"`
	Description string `json:"description" binding:"max=500"`
}

// UpdateModelRequest 更新统一模型请求
type UpdateModelRequest struct {
	Name        *string `json:"name" binding:"omitempty,max=100"`
	DisplayName *string `json:"display_name" binding:"omitempty,max=200"`
	Description *string `json:"description" binding:"omitempty,max=500"`
}

// ModelResponse 统一模型响应
type ModelResponse struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
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
		DisplayName: model.DisplayName,
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

// ==================== 映射相关 DTO ====================

// CreateMappingRequest 创建映射请求
type CreateMappingRequest struct {
	UnifiedModelID uint   `json:"-"` // 从URL路径获取，不需要在JSON中提供
	ProviderID     uint   `json:"provider_id" binding:"required"`
	TargetModel    string `json:"target_model" binding:"required,max=100"`
	Weight         int    `json:"weight" binding:"omitempty,min=1,max=100"` // 可选，默认50
	Priority       int    `json:"priority" binding:"omitempty,min=1"`       // 可选，默认1
	Enabled        bool   `json:"enabled"`
}

// UpdateMappingRequest 更新映射请求
type UpdateMappingRequest struct {
	TargetModel *string `json:"target_model" binding:"omitempty,max=100"`
	Weight      *int    `json:"weight" binding:"omitempty,min=1,max=100"`
	Priority    *int    `json:"priority" binding:"omitempty,min=1"`
	Enabled     *bool   `json:"enabled"`
}

// MappingResponse 映射响应
type MappingResponse struct {
	ID             uint                  `json:"id"`
	UnifiedModelID uint                  `json:"unified_model_id"`
	ProviderID     uint                  `json:"provider_id"`
	TargetModel    string                `json:"target_model"`
	Weight         int                   `json:"weight"`
	Priority       int                   `json:"priority"`
	Enabled        bool                  `json:"enabled"`
	Provider       *ProviderInfoResponse `json:"provider,omitempty"`
	UnifiedModel   *ModelInfoResponse    `json:"unified_model,omitempty"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
}

// ProviderInfoResponse 供应商基本信息响应（用于关联查询）
type ProviderInfoResponse struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
	Enabled bool   `json:"enabled"`
}

// ModelInfoResponse 模型基本信息响应（用于关联查询）
type ModelInfoResponse struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ListMappingsResponse 查询映射列表响应
type ListMappingsResponse struct {
	Mappings []*MappingResponse `json:"mappings"`
	Total    int64              `json:"total"`
}

// ToMappingResponse 将映射实体转换为响应对象
func ToMappingResponse(mapping *models.ModelMapping) *MappingResponse {
	response := &MappingResponse{
		ID:             mapping.ID,
		UnifiedModelID: mapping.UnifiedModelID,
		ProviderID:     mapping.ProviderID,
		TargetModel:    mapping.TargetModel,
		Weight:         mapping.Weight,
		Priority:       mapping.Priority,
		Enabled:        mapping.Enabled,
		CreatedAt:      mapping.CreatedAt,
		UpdatedAt:      mapping.UpdatedAt,
	}

	// 如果包含供应商信息
	if mapping.Provider.ID > 0 {
		response.Provider = &ProviderInfoResponse{
			ID:      mapping.Provider.ID,
			Name:    mapping.Provider.Name,
			BaseURL: mapping.Provider.BaseURL,
			Enabled: mapping.Provider.Enabled,
		}
	}

	// 如果包含统一模型信息
	if mapping.UnifiedModel.ID > 0 {
		response.UnifiedModel = &ModelInfoResponse{
			ID:          mapping.UnifiedModel.ID,
			Name:        mapping.UnifiedModel.Name,
			Description: mapping.UnifiedModel.Description,
		}
	}

	return response
}

// ToMappingResponseList 将映射实体列表转换为响应对象列表
func ToMappingResponseList(mappings []*models.ModelMapping) []*MappingResponse {
	responses := make([]*MappingResponse, len(mappings))
	for i, mapping := range mappings {
		responses[i] = ToMappingResponse(mapping)
	}
	return responses
}
