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
	ErrInvalidModelName = errors.New("模型名称只能包含字母、数字、连字符和下划线")
	// ErrModelNameEmpty 模型名称为空
	ErrModelNameEmpty = errors.New("模型名称不能为空")
	// ErrModelNameTooLong 模型名称过长
	ErrModelNameTooLong = errors.New("模型名称不能超过100个字符")
	// ErrDescriptionTooLong 描述过长
	ErrDescriptionTooLong = errors.New("描述不能超过500个字符")
)

// ModelNamePattern 模型名称正则表达式（字母、数字、连字符、下划线）
var ModelNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

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

// validateDescription 验证描述
func (s *Service) validateDescription(description string) error {
	if len(description) > 500 {
		return ErrDescriptionTooLong
	}
	return nil
}