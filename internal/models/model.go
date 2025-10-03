package models

import (
	"time"

	"gorm.io/gorm"
)

// UnifiedModel 统一模型
// 用户自定义的统一模型名称，用于屏蔽不同供应商的命名差异
type UnifiedModel struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"type:varchar(100);not null" json:"name"`
	DisplayName string         `gorm:"type:varchar(200);not null;default:''" json:"display_name"`
	Description string         `gorm:"type:text" json:"description"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"` // 软删除支持
}

// TableName 指定表名
func (UnifiedModel) TableName() string {
	return "unified_models"
}
