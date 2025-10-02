package provider

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/Mieluoxxx/Siriusx-API/internal/crypto"
	"github.com/Mieluoxxx/Siriusx-API/internal/models"
)

var (
	// ErrInvalidInput 无效输入
	ErrInvalidInput = errors.New("invalid input")
	// ErrInvalidURL 无效 URL
	ErrInvalidURL = errors.New("invalid URL")
)

// Service 供应商业务逻辑层
type Service struct {
	repo          *Repository
	encryptionKey []byte
}

// NewService 创建 Service 实例
func NewService(repo *Repository) *Service {
	return &Service{
		repo:          repo,
		encryptionKey: nil, // 延迟加载
	}
}

// NewServiceWithEncryption 创建带加密密钥的 Service 实例
func NewServiceWithEncryption(repo *Repository, encryptionKey []byte) *Service {
	return &Service{
		repo:          repo,
		encryptionKey: encryptionKey,
	}
}

// CreateProvider 创建供应商
func (s *Service) CreateProvider(req CreateProviderRequest) (*models.Provider, error) {
	// 验证参数
	if err := s.validateCreateRequest(req); err != nil {
		return nil, err
	}

	// 检查名称是否已存在
	exists, err := s.repo.CheckNameExists(req.Name, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrProviderNameExists
	}

	// 创建 Provider 模型
	provider := &models.Provider{
		Name:         req.Name,
		BaseURL:      req.BaseURL,
		APIKey:       req.APIKey, // 将在保存前加密
		Priority:     50,
		HealthStatus: "unknown",
	}

	// 应用 Enabled（默认值 true）
	if req.Enabled != nil {
		provider.Enabled = *req.Enabled
	} else {
		provider.Enabled = true
	}

	// 应用 Priority（默认值 50）
	if req.Priority != nil {
		provider.Priority = *req.Priority
	}

	// 加密 API Key（如果配置了加密密钥）
	plaintextKey := provider.APIKey // 保存明文用于返回
	if s.encryptionKey != nil {
		encryptedKey, err := crypto.EncryptString(provider.APIKey, s.encryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt API key: %w", err)
		}
		provider.APIKey = encryptedKey
	}

	// 保存到数据库
	if err := s.repo.Create(provider); err != nil {
		return nil, err
	}

	// 返回前恢复明文 API Key（Handler 会负责脱敏）
	provider.APIKey = plaintextKey

	return provider, nil
}

// GetProvider 获取单个供应商
func (s *Service) GetProvider(id uint) (*models.Provider, error) {
	provider, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// 解密 API Key（如果配置了加密密钥）
	if s.encryptionKey != nil && provider.APIKey != "" {
		decryptedKey, err := crypto.DecryptString(provider.APIKey, s.encryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt API key: %w", err)
		}
		provider.APIKey = decryptedKey
	}

	return provider, nil
}

// GetProviderWithDecryptedKey 获取供应商并解密 API Key（内部使用）
// 已废弃：GetProvider 现在默认解密
func (s *Service) GetProviderWithDecryptedKey(id uint) (*models.Provider, error) {
	return s.GetProvider(id)
}

// ListProviders 获取供应商列表（分页）
func (s *Service) ListProviders(page, pageSize int) ([]*models.Provider, int64, error) {
	// 参数验证
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	return s.repo.FindAll(page, pageSize)
}

// UpdateProvider 更新供应商
func (s *Service) UpdateProvider(id uint, req UpdateProviderRequest) (*models.Provider, error) {
	// 验证参数
	if err := s.validateUpdateRequest(req); err != nil {
		return nil, err
	}

	// 查找供应商
	provider, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// 如果要更新名称，检查名称是否已存在
	if req.Name != nil && *req.Name != provider.Name {
		exists, err := s.repo.CheckNameExists(*req.Name, id)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrProviderNameExists
		}
		provider.Name = *req.Name
	}

	// 更新其他字段
	if req.BaseURL != nil {
		provider.BaseURL = *req.BaseURL
	}

	var plaintextKey string // 保存明文用于返回
	if req.APIKey != nil {
		plaintextKey = *req.APIKey
		// 加密 API Key（如果配置了加密密钥）
		if s.encryptionKey != nil {
			encryptedKey, err := crypto.EncryptString(*req.APIKey, s.encryptionKey)
			if err != nil {
				return nil, fmt.Errorf("failed to encrypt API key: %w", err)
			}
			provider.APIKey = encryptedKey
		} else {
			provider.APIKey = *req.APIKey
		}
	}

	if req.Enabled != nil {
		provider.Enabled = *req.Enabled
	}
	if req.Priority != nil {
		provider.Priority = *req.Priority
	}

	// 保存到数据库
	if err := s.repo.Update(provider); err != nil {
		return nil, err
	}

	// 返回前恢复/解密 API Key（Handler 会负责脱敏）
	if req.APIKey != nil {
		provider.APIKey = plaintextKey
	} else if s.encryptionKey != nil && provider.APIKey != "" {
		// 如果没有更新 API Key，则解密现有的
		decryptedKey, err := crypto.DecryptString(provider.APIKey, s.encryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt API key: %w", err)
		}
		provider.APIKey = decryptedKey
	}

	return provider, nil
}

// DeleteProvider 删除供应商（软删除）
func (s *Service) DeleteProvider(id uint) error {
	return s.repo.SoftDelete(id)
}

// UpdateProviderHealthStatus 更新供应商健康状态
func (s *Service) UpdateProviderHealthStatus(id uint, healthStatus string) error {
	return s.repo.UpdateHealthStatus(id, healthStatus)
}

// validateCreateRequest 验证创建请求
func (s *Service) validateCreateRequest(req CreateProviderRequest) error {
	// 名称不能为空
	if strings.TrimSpace(req.Name) == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidInput)
	}

	// Base URL 验证
	if err := s.validateURL(req.BaseURL); err != nil {
		return err
	}

	// API Key 不能为空
	if strings.TrimSpace(req.APIKey) == "" {
		return fmt.Errorf("%w: api_key is required", ErrInvalidInput)
	}

	// Priority 范围验证
	if req.Priority != nil {
		if *req.Priority < 1 || *req.Priority > 100 {
			return fmt.Errorf("%w: priority must be between 1 and 100", ErrInvalidInput)
		}
	}

	return nil
}

// validateUpdateRequest 验证更新请求
func (s *Service) validateUpdateRequest(req UpdateProviderRequest) error {
	// 名称验证
	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		return fmt.Errorf("%w: name cannot be empty", ErrInvalidInput)
	}

	// Base URL 验证
	if req.BaseURL != nil {
		if err := s.validateURL(*req.BaseURL); err != nil {
			return err
		}
	}

	// API Key 验证
	if req.APIKey != nil && strings.TrimSpace(*req.APIKey) == "" {
		return fmt.Errorf("%w: api_key cannot be empty", ErrInvalidInput)
	}

	// Priority 范围验证
	if req.Priority != nil {
		if *req.Priority < 1 || *req.Priority > 100 {
			return fmt.Errorf("%w: priority must be between 1 and 100", ErrInvalidInput)
		}
	}

	return nil
}

// validateURL 验证 URL 格式
func (s *Service) validateURL(urlStr string) error {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}

	// 必须是 HTTP 或 HTTPS
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("%w: URL must be http or https", ErrInvalidURL)
	}

	// 必须有 host
	if parsedURL.Host == "" {
		return fmt.Errorf("%w: URL must have a host", ErrInvalidURL)
	}

	return nil
}
