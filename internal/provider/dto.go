package provider

import (
	"time"

	"github.com/Mieluoxxx/Siriusx-API/internal/models"
)

// CreateProviderRequest 创建供应商请求
type CreateProviderRequest struct {
	Name     string `json:"name" binding:"required"`
	BaseURL  string `json:"base_url" binding:"required,url"`
	APIKey   string `json:"api_key" binding:"required"`
	Enabled  *bool  `json:"enabled"`
	Priority *int   `json:"priority" binding:"omitempty,min=1,max=100"`
}

// UpdateProviderRequest 更新供应商请求
type UpdateProviderRequest struct {
	Name     *string `json:"name"`
	BaseURL  *string `json:"base_url" binding:"omitempty,url"`
	APIKey   *string `json:"api_key"`
	Enabled  *bool   `json:"enabled"`
	Priority *int    `json:"priority" binding:"omitempty,min=1,max=100"`
}

// ProviderResponse 供应商响应（API Key 脱敏）
type ProviderResponse struct {
	ID           uint      `json:"id"`
	Name         string    `json:"name"`
	BaseURL      string    `json:"base_url"`
	APIKey       string    `json:"api_key"` // 脱敏显示
	Enabled      bool      `json:"enabled"`
	Priority     int       `json:"priority"`
	HealthStatus string    `json:"health_status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ProviderListResponse 供应商列表响应（带分页）
type ProviderListResponse struct {
	Data       []ProviderResponse `json:"data"`
	Pagination PaginationMeta     `json:"pagination"`
}

// PaginationMeta 分页元数据
type PaginationMeta struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail 错误详情
type ErrorDetail struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// MaskAPIKey API Key 脱敏
// 格式: sk-****last4
func MaskAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return "****"
	}
	return apiKey[:3] + "****" + apiKey[len(apiKey)-4:]
}

// MaskEncryptedAPIKey 加密后的 API Key 脱敏显示
// 由于加密后是 Base64 字符串，无法直接脱敏
// 因此显示固定的脱敏格式
func MaskEncryptedAPIKey() string {
	return "[encrypted]****"
}

// ToProviderResponse 转换为响应（API Key 脱敏）
func ToProviderResponse(provider *models.Provider, maskKey bool) *ProviderResponse {
	resp := &ProviderResponse{
		ID:           provider.ID,
		Name:         provider.Name,
		BaseURL:      provider.BaseURL,
		Enabled:      provider.Enabled,
		Priority:     provider.Priority,
		HealthStatus: provider.HealthStatus,
		CreatedAt:    provider.CreatedAt,
		UpdatedAt:    provider.UpdatedAt,
	}

	// API Key 脱敏
	if maskKey {
		resp.APIKey = MaskEncryptedAPIKey()
	} else {
		resp.APIKey = provider.APIKey
	}

	return resp
}

// ToProviderResponseWithDecryption 转换为响应（解密并脱敏 API Key）
func ToProviderResponseWithDecryption(provider *models.Provider, decryptedKey string) *ProviderResponse {
	resp := &ProviderResponse{
		ID:           provider.ID,
		Name:         provider.Name,
		BaseURL:      provider.BaseURL,
		Enabled:      provider.Enabled,
		Priority:     provider.Priority,
		HealthStatus: provider.HealthStatus,
		CreatedAt:    provider.CreatedAt,
		UpdatedAt:    provider.UpdatedAt,
		APIKey:       MaskAPIKey(decryptedKey), // 脱敏明文 API Key
	}

	return resp
}
