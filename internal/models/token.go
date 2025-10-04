package models

import (
	"time"

	"gorm.io/gorm"
)

// Token API 令牌
// 用于验证客户端访问权限
type Token struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"type:varchar(100);not null" json:"name"`
	Token     string         `gorm:"type:varchar(100);not null;uniqueIndex" json:"token"`
	Enabled   bool           `gorm:"default:true;not null" json:"enabled"`
	ExpiresAt *time.Time     `json:"expires_at,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"` // 软删除支持
}

// TableName 指定表名
func (Token) TableName() string {
	return "tokens"
}
