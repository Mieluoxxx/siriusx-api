package token

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/Mieluoxxx/Siriusx-API/internal/models"
)

var (
	// ErrInvalidToken Token 无效
	ErrInvalidToken = errors.New("invalid token")
	// ErrTokenExpired Token 已过期
	ErrTokenExpired = errors.New("token expired")
	// ErrTokenDisabled Token 已禁用
	ErrTokenDisabled = errors.New("token disabled")
	// ErrInvalidExpiresAt 过期时间必须是未来时间
	ErrInvalidExpiresAt = errors.New("expires_at must be in the future")
	// ErrInvalidCustomToken 自定义 Token 格式无效
	ErrInvalidCustomToken = errors.New("custom token must start with 'sk-' and be at least 8 characters")
)

// Service Token 业务逻辑层
type Service struct {
	repo *Repository
}

// NewService 创建 Service 实例
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// GenerateTokenValue 生成唯一的 Token 值
// 格式: sk- + 32 字节 base64 编码 (URLEncoding)
func GenerateTokenValue() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	token := "sk-" + base64.URLEncoding.EncodeToString(bytes)
	return token, nil
}

// ValidateCustomToken 验证自定义 Token 格式
func ValidateCustomToken(token string) error {
	if len(token) < 8 {
		return ErrInvalidCustomToken
	}
	if token[:3] != "sk-" {
		return ErrInvalidCustomToken
	}
	return nil
}

// CreateToken 创建 Token
func (s *Service) CreateToken(name string, expiresAt *time.Time, customToken string) (*models.Token, error) {
	// 验证过期时间
	if expiresAt != nil && expiresAt.Before(time.Now()) {
		return nil, ErrInvalidExpiresAt
	}

	var tokenValue string
	var err error

	// 如果提供了自定义 Token
	if customToken != "" {
		// 验证自定义 Token 格式
		if err := ValidateCustomToken(customToken); err != nil {
			return nil, err
		}

		// 检查自定义 Token 是否已存在
		exists, err := s.repo.CheckValueExists(customToken)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrTokenValueExists
		}

		tokenValue = customToken
	} else {
		// 生成唯一 Token 值
		maxRetries := 5
		for i := 0; i < maxRetries; i++ {
			tokenValue, err = GenerateTokenValue()
			if err != nil {
				return nil, err
			}

			// 检查是否已存在
			exists, err := s.repo.CheckValueExists(tokenValue)
			if err != nil {
				return nil, err
			}
			if !exists {
				break
			}

			// 如果重试次数用完，返回错误
			if i == maxRetries-1 {
				return nil, ErrTokenValueExists
			}
		}
	}

	// 创建 Token 对象
	token := &models.Token{
		Name:      name,
		Token:     tokenValue,
		Enabled:   true,
		ExpiresAt: expiresAt,
	}

	// 保存到数据库
	if err := s.repo.Create(token); err != nil {
		return nil, err
	}

	return token, nil
}

// ListTokens 获取所有 Token 列表
func (s *Service) ListTokens() ([]*models.Token, error) {
	return s.repo.FindAll()
}

// GetToken 根据 ID 获取 Token
func (s *Service) GetToken(id uint) (*models.Token, error) {
	return s.repo.FindByID(id)
}

// DeleteToken 删除 Token
func (s *Service) DeleteToken(id uint) error {
	return s.repo.Delete(id)
}

// ValidateToken 验证 Token (用于认证中间件)
// 检查 Token 是否存在、是否启用、是否过期
func (s *Service) ValidateToken(tokenValue string) (*models.Token, error) {
	// 查找 Token
	token, err := s.repo.FindByValue(tokenValue)
	if err != nil {
		if errors.Is(err, ErrTokenNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, err
	}

	// 检查是否启用
	if !token.Enabled {
		return nil, ErrTokenDisabled
	}

	// 检查是否过期
	if token.ExpiresAt != nil && token.ExpiresAt.Before(time.Now()) {
		return nil, ErrTokenExpired
	}

	return token, nil
}

// MaskToken 脱敏显示 Token
// 格式: sk-****{最后4位}
func MaskToken(token string) string {
	if len(token) < 8 {
		return "****"
	}
	last4 := token[len(token)-4:]
	return "sk-****" + last4
}
