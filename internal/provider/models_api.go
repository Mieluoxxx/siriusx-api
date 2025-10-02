package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ModelInfo 模型信息
type ModelInfo struct {
	ID     string `json:"id"`
	Object string `json:"object"`
}

// AvailableModelsResponse 可用模型响应
type AvailableModelsResponse struct {
	ProviderID   uint        `json:"provider_id"`
	ProviderName string      `json:"provider_name"`
	Models       []ModelInfo `json:"models"`
	Total        int         `json:"total"`
	FetchedAt    time.Time   `json:"fetched_at"`
}

// GetAvailableModels 获取供应商的可用模型列表
func (s *Service) GetAvailableModels(id uint) (*AvailableModelsResponse, error) {
	// 获取供应商信息
	provider, err := s.GetProvider(id)
	if err != nil {
		return nil, err
	}

	// 构建请求 URL
	// 标准化 baseURL,移除末尾斜杠以避免双斜杠问题
	modelsURL := strings.TrimRight(provider.BaseURL, "/") + "/v1/models"

	// 创建HTTP客户端
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", modelsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置认证头
	req.Header.Set("Authorization", "Bearer "+provider.APIKey)
	req.Header.Set("User-Agent", "Siriusx-API/1.0")

	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("获取模型列表失败: HTTP %d", resp.StatusCode)
	}

	// 解析响应
	var result struct {
		Data []ModelInfo `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 构建响应
	return &AvailableModelsResponse{
		ProviderID:   provider.ID,
		ProviderName: provider.Name,
		Models:       result.Data,
		Total:        len(result.Data),
		FetchedAt:    time.Now(),
	}, nil
}
