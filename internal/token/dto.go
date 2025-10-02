package token

import (
	"time"

	"github.com/Mieluoxxx/Siriusx-API/internal/models"
)

// CreateTokenRequest 创建 Token 请求
type CreateTokenRequest struct {
	Name      string     `json:"name" binding:"required,max=100"`
	ExpiresAt *time.Time `json:"expires_at"`
}

// TokenDTO Token 数据传输对象
type TokenDTO struct {
	ID           uint       `json:"id"`
	Name         string     `json:"name"`
	Token        string     `json:"token,omitempty"`   // 仅在创建时返回
	TokenDisplay string     `json:"token_display"`     // 脱敏显示
	Enabled      bool       `json:"enabled"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// ToTokenDTO 将 Token 模型转换为 DTO（包含完整 Token）
func ToTokenDTO(token *models.Token, showFullToken bool) *TokenDTO {
	dto := &TokenDTO{
		ID:           token.ID,
		Name:         token.Name,
		TokenDisplay: MaskToken(token.Token),
		Enabled:      token.Enabled,
		ExpiresAt:    token.ExpiresAt,
		CreatedAt:    token.CreatedAt,
		UpdatedAt:    token.UpdatedAt,
	}

	// 仅在需要时显示完整 Token
	if showFullToken {
		dto.Token = token.Token
	}

	return dto
}
