package provider

import (
	"errors"

	"github.com/Mieluoxxx/Siriusx-API/internal/models"
	"gorm.io/gorm"
)

var (
	// ErrProviderNotFound 供应商不存在
	ErrProviderNotFound = errors.New("provider not found")
	// ErrProviderNameExists 供应商名称已存在
	ErrProviderNameExists = errors.New("provider name already exists")
)

// Repository 供应商数据访问层
type Repository struct {
	db *gorm.DB
}

// NewRepository 创建 Repository 实例
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create 创建供应商
func (r *Repository) Create(provider *models.Provider) error {
	// 使用 Select 明确指定要保存的字段，包括零值字段
	return r.db.Select("Name", "BaseURL", "APIKey", "TestModel", "Enabled", "HealthStatus").Create(provider).Error
}

// FindByID 根据 ID 查找供应商
func (r *Repository) FindByID(id uint) (*models.Provider, error) {
	var provider models.Provider
	err := r.db.First(&provider, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProviderNotFound
		}
		return nil, err
	}
	return &provider, nil
}

// FindByName 根据名称查找供应商
func (r *Repository) FindByName(name string) (*models.Provider, error) {
	var provider models.Provider
	err := r.db.Where("name = ?", name).First(&provider).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProviderNotFound
		}
		return nil, err
	}
	return &provider, nil
}

// FindAll 查找所有供应商（分页）
func (r *Repository) FindAll(page, pageSize int) ([]*models.Provider, int64, error) {
	var providers []*models.Provider
	var total int64

	// 计算总数
	if err := r.db.Model(&models.Provider{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := r.db.Offset(offset).Limit(pageSize).Find(&providers).Error
	if err != nil {
		return nil, 0, err
	}

	return providers, total, nil
}

// Update 更新供应商
func (r *Repository) Update(provider *models.Provider) error {
	return r.db.Save(provider).Error
}

// UpdateHealthStatus 仅更新健康状态
func (r *Repository) UpdateHealthStatus(id uint, healthStatus string) error {
	return r.db.Model(&models.Provider{}).Where("id = ?", id).Update("health_status", healthStatus).Error
}

// Delete 删除供应商（硬删除）
func (r *Repository) Delete(id uint) error {
	result := r.db.Delete(&models.Provider{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrProviderNotFound
	}
	return nil
}

// CheckNameExists 检查名称是否存在（排除指定 ID）
func (r *Repository) CheckNameExists(name string, excludeID uint) (bool, error) {
	var count int64
	query := r.db.Model(&models.Provider{}).Where("name = ?", name)
	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}
	err := query.Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
