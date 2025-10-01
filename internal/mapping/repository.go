package mapping

import (
	"errors"
	"fmt"

	"github.com/Mieluoxxx/Siriusx-API/internal/models"
	"gorm.io/gorm"
)

var (
	// ErrModelNotFound 模型不存在
	ErrModelNotFound = errors.New("model not found")
	// ErrModelNameExists 模型名称已存在
	ErrModelNameExists = errors.New("model name already exists")
)

// Repository 统一模型数据访问层
type Repository struct {
	db *gorm.DB
}

// NewRepository 创建 Repository 实例
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create 创建统一模型
func (r *Repository) Create(model *models.UnifiedModel) error {
	// 使用 Select 明确指定要保存的字段
	return r.db.Select("Name", "Description").Create(model).Error
}

// FindByID 根据 ID 查找模型
func (r *Repository) FindByID(id uint) (*models.UnifiedModel, error) {
	var model models.UnifiedModel
	err := r.db.First(&model, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrModelNotFound
		}
		return nil, err
	}
	return &model, nil
}

// FindByName 根据名称查找模型
func (r *Repository) FindByName(name string) (*models.UnifiedModel, error) {
	var model models.UnifiedModel
	err := r.db.Where("name = ?", name).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrModelNotFound
		}
		return nil, err
	}
	return &model, nil
}

// FindAll 查询所有模型（支持分页和搜索）
func (r *Repository) FindAll(page, pageSize int, search string) ([]*models.UnifiedModel, int64, error) {
	var modelList []*models.UnifiedModel
	var total int64

	query := r.db.Model(&models.UnifiedModel{})

	// 搜索功能：按名称模糊匹配
	if search != "" {
		searchPattern := fmt.Sprintf("%%%s%%", search)
		query = query.Where("name LIKE ?", searchPattern)
	}

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&modelList).Error
	if err != nil {
		return nil, 0, err
	}

	return modelList, total, nil
}

// Update 更新模型
func (r *Repository) Update(model *models.UnifiedModel) error {
	// 使用 Select 明确指定要更新的字段
	return r.db.Select("Name", "Description").Save(model).Error
}

// Delete 删除模型（软删除）
func (r *Repository) Delete(id uint) error {
	result := r.db.Delete(&models.UnifiedModel{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrModelNotFound
	}
	return nil
}

// CheckNameExists 检查模型名称是否已存在（排除指定 ID）
func (r *Repository) CheckNameExists(name string, excludeID uint) (bool, error) {
	var count int64
	query := r.db.Model(&models.UnifiedModel{}).Where("name = ?", name)

	// 如果提供了 excludeID，则排除该记录
	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}

	err := query.Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}