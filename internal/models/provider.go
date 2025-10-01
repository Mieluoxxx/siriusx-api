package models

import "time"

// Provider 供应商模型
// 用于存储 AI 服务供应商的配置信息
type Provider struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"name"`
	BaseURL      string    `gorm:"type:varchar(255);not null" json:"base_url"`
	APIKey       string    `gorm:"type:text;not null" json:"api_key"` // 加密存储
	Enabled      bool      `gorm:"not null" json:"enabled"`
	Priority     int       `gorm:"not null" json:"priority"`
	HealthStatus string    `gorm:"type:varchar(20);default:'unknown'" json:"health_status"` // healthy/unhealthy/unknown
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TableName 指定表名
func (Provider) TableName() string {
	return "providers"
}
