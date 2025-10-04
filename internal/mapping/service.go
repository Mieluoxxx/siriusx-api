package mapping

import (
	"errors"
	"math"
	"regexp"
	"strings"

	"github.com/Mieluoxxx/Siriusx-API/internal/models"
)

var (
	// ErrInvalidModelName 无效的模型名称
	ErrInvalidModelName = errors.New("模型名称只能包含字母、数字、连字符、下划线、点号和@符号")
	// ErrModelNameEmpty 模型名称为空
	ErrModelNameEmpty = errors.New("模型名称不能为空")
	// ErrModelNameTooLong 模型名称过长
	ErrModelNameTooLong = errors.New("模型名称不能超过100个字符")
	// ErrDisplayNameEmpty 显示名称为空
	ErrDisplayNameEmpty = errors.New("显示名称不能为空")
	// ErrDisplayNameTooLong 显示名称过长
	ErrDisplayNameTooLong = errors.New("显示名称不能超过200个字符")
	// ErrDescriptionTooLong 描述过长
	ErrDescriptionTooLong = errors.New("描述不能超过500个字符")

	// 映射相关错误
	// ErrInvalidWeight 无效的权重
	ErrInvalidWeight = errors.New("权重必须在1-100之间")
	// ErrInvalidPriority 无效的优先级
	ErrInvalidPriority = errors.New("优先级必须大于0")
	// ErrTargetModelEmpty 目标模型为空
	ErrTargetModelEmpty = errors.New("目标模型不能为空")
	// ErrProviderNotFound 供应商不存在
	ErrProviderNotFound = errors.New("供应商不存在")
)

// ModelNamePattern 模型名称正则表达式（字母、数字、连字符、下划线、点号、@符号）
var ModelNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_.\-@]+$`)

// Service 统一模型业务逻辑层
type Service struct {
	repo *Repository
}

// NewService 创建 Service 实例
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// CreateModel 创建统一模型
func (s *Service) CreateModel(req CreateModelRequest) (*ModelResponse, error) {
	// 验证输入参数
	if err := s.validateModelName(req.Name); err != nil {
		return nil, err
	}

	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		displayName = strings.TrimSpace(req.Name)
	}

	if err := s.validateDisplayName(displayName); err != nil {
		return nil, err
	}

	if err := s.validateDescription(req.Description); err != nil {
		return nil, err
	}

	// 检查名称是否已存在
	exists, err := s.repo.CheckNameExists(req.Name, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrModelNameExists
	}

	// 创建模型实体
	model := &models.UnifiedModel{
		Name:        strings.TrimSpace(req.Name),
		DisplayName: displayName,
		Description: strings.TrimSpace(req.Description),
	}

	// 保存到数据库
	if err := s.repo.Create(model); err != nil {
		return nil, err
	}

	return ToModelResponse(model), nil
}

// GetModel 根据 ID 获取模型
func (s *Service) GetModel(id uint) (*ModelResponse, error) {
	model, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	return ToModelResponse(model), nil
}

// ListModels 查询模型列表（支持分页和搜索）
func (s *Service) ListModels(req ListModelsRequest) (*ListModelsResponse, error) {
	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	// 清理搜索关键词
	search := strings.TrimSpace(req.Search)

	// 查询数据
	models, total, err := s.repo.FindAll(req.Page, req.PageSize, search)
	if err != nil {
		return nil, err
	}

	// 计算总页数
	totalPages := int(math.Ceil(float64(total) / float64(req.PageSize)))

	// 构建响应
	response := &ListModelsResponse{
		Models: ToModelResponseList(models),
		Pagination: PaginationInfo{
			Page:       req.Page,
			PageSize:   req.PageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}

	return response, nil
}

// UpdateModel 更新模型
func (s *Service) UpdateModel(id uint, req UpdateModelRequest) (*ModelResponse, error) {
	// 查找现有模型
	model, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// 更新字段
	updated := false

	if req.Name != nil {
		newName := strings.TrimSpace(*req.Name)
		if err := s.validateModelName(newName); err != nil {
			return nil, err
		}

		// 检查新名称是否与其他模型冲突
		if newName != model.Name {
			exists, err := s.repo.CheckNameExists(newName, model.ID)
			if err != nil {
				return nil, err
			}
			if exists {
				return nil, ErrModelNameExists
			}
			model.Name = newName
			updated = true
		}
	}

	if req.DisplayName != nil {
		newDisplayName := strings.TrimSpace(*req.DisplayName)
		if newDisplayName == "" {
			newDisplayName = strings.TrimSpace(model.Name)
		}
		if err := s.validateDisplayName(newDisplayName); err != nil {
			return nil, err
		}
		if newDisplayName != model.DisplayName {
			model.DisplayName = newDisplayName
			updated = true
		}
	}

	if req.Description != nil {
		newDesc := strings.TrimSpace(*req.Description)
		if err := s.validateDescription(newDesc); err != nil {
			return nil, err
		}
		if newDesc != model.Description {
			model.Description = newDesc
			updated = true
		}
	}

	// 如果有更新，保存到数据库
	if updated {
		if err := s.repo.Update(model); err != nil {
			return nil, err
		}
	}

	return ToModelResponse(model), nil
}

// DeleteModel 删除模型
func (s *Service) DeleteModel(id uint) error {
	// 检查模型是否存在
	_, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}

	// 删除模型
	return s.repo.Delete(id)
}

// validateModelName 验证模型名称
func (s *Service) validateModelName(name string) error {
	name = strings.TrimSpace(name)

	if name == "" {
		return ErrModelNameEmpty
	}

	if len(name) > 100 {
		return ErrModelNameTooLong
	}

	if !ModelNamePattern.MatchString(name) {
		return ErrInvalidModelName
	}

	return nil
}

// validateDisplayName 验证显示名称
func (s *Service) validateDisplayName(displayName string) error {
	displayName = strings.TrimSpace(displayName)

	if displayName == "" {
		return ErrDisplayNameEmpty
	}

	if len(displayName) > 200 {
		return ErrDisplayNameTooLong
	}

	return nil
}

// validateDescription 验证描述
func (s *Service) validateDescription(description string) error {
	if len(description) > 500 {
		return ErrDescriptionTooLong
	}
	return nil
}

// ==================== 映射相关方法 ====================

// CreateMapping 创建模型映射
func (s *Service) CreateMapping(req CreateMappingRequest) (*MappingResponse, error) {
	// 设置默认值（仅当为0时）
	if req.Weight == 0 {
		req.Weight = 50 // 默认权重 50
	}
	if req.Priority == 0 {
		req.Priority = 1 // 默认优先级 1
	}

	// 验证输入参数（负数会被拒绝）
	if err := s.validateMappingRequest(req); err != nil {
		return nil, err
	}

	// 检查统一模型是否存在
	_, err := s.repo.FindByID(req.UnifiedModelID)
	if err != nil {
		if errors.Is(err, ErrModelNotFound) {
			return nil, ErrModelNotFound
		}
		return nil, err
	}

	// 检查映射是否已存在
	exists, err := s.repo.CheckMappingExists(req.UnifiedModelID, req.ProviderID, req.TargetModel, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrMappingExists
	}

	// 注意：允许相同的优先级，不再检查优先级冲突

	// 创建映射实体
	mapping := &models.ModelMapping{
		UnifiedModelID: req.UnifiedModelID,
		ProviderID:     req.ProviderID,
		TargetModel:    strings.TrimSpace(req.TargetModel),
		Weight:         req.Weight,
		Priority:       req.Priority,
		Enabled:        req.Enabled,
	}

	// 保存到数据库
	if err := s.repo.CreateMapping(mapping); err != nil {
		return nil, err
	}

	// 查询完整的映射信息（包含关联数据）
	fullMapping, err := s.repo.FindMappingByID(mapping.ID)
	if err != nil {
		return nil, err
	}

	return ToMappingResponse(fullMapping), nil
}

// ListMappings 查询模型的所有映射
func (s *Service) ListMappings(modelID uint, includeProvider bool) (*ListMappingsResponse, error) {
	// 检查统一模型是否存在
	_, err := s.repo.FindByID(modelID)
	if err != nil {
		return nil, err
	}

	// 查询映射列表
	mappings, err := s.repo.FindMappingsByModelIDWithAll(modelID, includeProvider)
	if err != nil {
		return nil, err
	}

	return &ListMappingsResponse{
		Mappings: ToMappingResponseList(mappings),
		Total:    int64(len(mappings)),
	}, nil
}

// GetMapping 根据 ID 获取映射
func (s *Service) GetMapping(id uint) (*MappingResponse, error) {
	mapping, err := s.repo.FindMappingByID(id)
	if err != nil {
		return nil, err
	}

	return ToMappingResponse(mapping), nil
}

// UpdateMapping 更新映射
func (s *Service) UpdateMapping(id uint, req UpdateMappingRequest) (*MappingResponse, error) {
	// 查找现有映射
	mapping, err := s.repo.FindMappingByID(id)
	if err != nil {
		return nil, err
	}

	// 更新字段
	updated := false

	if req.TargetModel != nil {
		newTargetModel := strings.TrimSpace(*req.TargetModel)
		if newTargetModel == "" {
			return nil, ErrTargetModelEmpty
		}
		if newTargetModel != mapping.TargetModel {
			// 检查新映射是否已存在
			exists, err := s.repo.CheckMappingExists(mapping.UnifiedModelID, mapping.ProviderID, newTargetModel, mapping.ID)
			if err != nil {
				return nil, err
			}
			if exists {
				return nil, ErrMappingExists
			}
			mapping.TargetModel = newTargetModel
			updated = true
		}
	}

	if req.Weight != nil {
		if *req.Weight < 1 || *req.Weight > 100 {
			return nil, ErrInvalidWeight
		}
		if *req.Weight != mapping.Weight {
			mapping.Weight = *req.Weight
			updated = true
		}
	}

	if req.Priority != nil {
		if *req.Priority < 1 {
			return nil, ErrInvalidPriority
		}
		if *req.Priority != mapping.Priority {
			// 注意：允许相同的优先级，不再检查优先级冲突
			mapping.Priority = *req.Priority
			updated = true
		}
	}

	if req.Enabled != nil {
		if *req.Enabled != mapping.Enabled {
			mapping.Enabled = *req.Enabled
			updated = true
		}
	}

	// 如果有更新，保存到数据库
	if updated {
		if err := s.repo.UpdateMapping(mapping); err != nil {
			return nil, err
		}
	}

	return ToMappingResponse(mapping), nil
}

// DeleteMapping 删除映射
func (s *Service) DeleteMapping(id uint) error {
	// 检查映射是否存在
	_, err := s.repo.FindMappingByID(id)
	if err != nil {
		return err
	}

	// 删除映射
	return s.repo.DeleteMapping(id)
}

// validateMappingRequest 验证映射请求
func (s *Service) validateMappingRequest(req CreateMappingRequest) error {
	if req.UnifiedModelID == 0 {
		return ErrModelNotFound
	}

	if req.ProviderID == 0 {
		return ErrProviderNotFound
	}

	if strings.TrimSpace(req.TargetModel) == "" {
		return ErrTargetModelEmpty
	}

	if req.Weight < 1 || req.Weight > 100 {
		return ErrInvalidWeight
	}

	if req.Priority < 1 {
		return ErrInvalidPriority
	}

	return nil
}

// GetModelByName 根据名称获取统一模型
func (s *Service) GetModelByName(name string) (*models.UnifiedModel, error) {
	return s.repo.FindByName(name)
}

// GetMappingsByModelID 获取模型的所有映射
func (s *Service) GetMappingsByModelID(modelID uint) ([]*models.ModelMapping, error) {
	return s.repo.FindMappingsByModelIDWithAll(modelID, true)
}
