package models

import "time"

// Token API 令牌
// 用于验证客户端访问权限
type Token struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	Name      string     `gorm:"type:varchar(100);not null" json:"name"`
	Token     string     `gorm:"type:varchar(100);uniqueIndex;not null" json:"token"`
	Enabled   bool       `gorm:"default:true;not null" json:"enabled"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// TableName 指定表名
func (Token) TableName() string {
	return "tokens"
}
