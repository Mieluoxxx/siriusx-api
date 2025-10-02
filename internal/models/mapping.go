package models

import "time"

// ModelMapping 模型映射
// 将统一模型映射到具体供应商的模型，支持权重和优先级配置
type ModelMapping struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UnifiedModelID uint      `gorm:"not null;index" json:"unified_model_id"`
	ProviderID     uint      `gorm:"not null;index" json:"provider_id"`
	TargetModel    string    `gorm:"type:varchar(100);not null" json:"target_model"`
	Weight         int       `gorm:"not null;default:50;check:weight >= 1 AND weight <= 100" json:"weight"` // 1-100，用于负载均衡
	Priority       int       `gorm:"not null;check:priority >= 1" json:"priority"`                          // 1, 2, 3...，数字越小优先级越高
	Enabled        bool      `gorm:"not null;default:true" json:"enabled"`                                  // 是否启用
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	// 关联关系
	UnifiedModel UnifiedModel `gorm:"foreignKey:UnifiedModelID;constraint:OnDelete:CASCADE" json:"unified_model,omitempty"`
	Provider     Provider     `gorm:"foreignKey:ProviderID;constraint:OnDelete:CASCADE" json:"provider,omitempty"`
}

// TableName 指定表名
func (ModelMapping) TableName() string {
	return "model_mappings"
}
