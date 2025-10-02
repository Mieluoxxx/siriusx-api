package token

import (
	"errors"

	"github.com/Mieluoxxx/Siriusx-API/internal/models"
	"gorm.io/gorm"
)

var (
	// ErrTokenNotFound Token 不存在
	ErrTokenNotFound = errors.New("token not found")
	// ErrTokenValueExists Token 值已存在
	ErrTokenValueExists = errors.New("token value already exists")
)

// Repository Token 数据访问层
type Repository struct {
	db *gorm.DB
}

// NewRepository 创建 Repository 实例
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create 创建 Token
func (r *Repository) Create(token *models.Token) error {
	// 使用 Select 明确指定要保存的字段，包括零值字段
	return r.db.Select("Name", "Token", "Enabled", "ExpiresAt").Create(token).Error
}

// FindByID 根据 ID 查找 Token
func (r *Repository) FindByID(id uint) (*models.Token, error) {
	var token models.Token
	err := r.db.First(&token, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTokenNotFound
		}
		return nil, err
	}
	return &token, nil
}

// FindByValue 根据 Token 值查找 Token
func (r *Repository) FindByValue(tokenValue string) (*models.Token, error) {
	var token models.Token
	err := r.db.Where("token = ?", tokenValue).First(&token).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTokenNotFound
		}
		return nil, err
	}
	return &token, nil
}

// FindAll 查找所有 Token
func (r *Repository) FindAll() ([]*models.Token, error) {
	var tokens []*models.Token
	err := r.db.Order("created_at DESC").Find(&tokens).Error
	if err != nil {
		return nil, err
	}
	return tokens, nil
}

// Delete 删除 Token
func (r *Repository) Delete(id uint) error {
	result := r.db.Delete(&models.Token{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrTokenNotFound
	}
	return nil
}

// CheckValueExists 检查 Token 值是否存在
func (r *Repository) CheckValueExists(tokenValue string) (bool, error) {
	var count int64
	err := r.db.Model(&models.Token{}).Where("token = ?", tokenValue).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
