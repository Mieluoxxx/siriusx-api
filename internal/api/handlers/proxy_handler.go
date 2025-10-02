package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/Mieluoxxx/Siriusx-API/internal/mapping"
	"github.com/Mieluoxxx/Siriusx-API/internal/models"
	"github.com/Mieluoxxx/Siriusx-API/internal/provider"
	"github.com/gin-gonic/gin"
)

// ProxyHandler 代理请求处理器
type ProxyHandler struct {
	providerService *provider.Service
	mappingService  *mapping.Service
}

// NewProxyHandler 创建代理处理器
func NewProxyHandler(providerService *provider.Service, mappingService *mapping.Service) *ProxyHandler {
	return &ProxyHandler{
		providerService: providerService,
		mappingService:  mappingService,
	}
}

// ChatCompletionRequest OpenAI 聊天完成请求
type ChatCompletionRequest struct {
	Model    string      `json:"model" binding:"required"`
	Messages interface{} `json:"messages" binding:"required"`
	Stream   bool        `json:"stream"`
	// 其他字段保持原样传递
}

// ChatCompletions 处理聊天完成请求
func (h *ProxyHandler) ChatCompletions(c *gin.Context) {
	// 1. 解析请求体
	var req map[string]interface{}
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法读取请求体"})
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 JSON 格式"})
		return
	}

	// 2. 获取模型名称
	modelName, ok := req["model"].(string)
	if !ok || modelName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 model 参数"})
		return
	}

	// 3. 查找统一模型
	unifiedModel, err := h.mappingService.GetModelByName(modelName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("未找到模型: %s", modelName),
		})
		return
	}

	// 4. 获取该模型的所有映射
	mappings, err := h.mappingService.GetMappingsByModelID(unifiedModel.ID)
	if err != nil || len(mappings) == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("模型 %s 没有可用的映射", modelName),
		})
		return
	}

	// 5. 选择一个可用的映射（负载均衡）
	selectedMapping := h.selectMapping(mappings)
	if selectedMapping == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "没有可用的供应商",
		})
		return
	}

	// 6. 获取供应商信息
	prov, err := h.providerService.GetProvider(selectedMapping.ProviderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取供应商信息失败",
		})
		return
	}

	// 7. 替换模型名称
	req["model"] = selectedMapping.TargetModel

	// 8. 转发请求到供应商
	h.forwardRequest(c, prov, req, bodyBytes, "/v1/chat/completions")
}

// Messages 处理 Claude Messages API 请求
func (h *ProxyHandler) Messages(c *gin.Context) {
	// 1. 解析请求体
	var req map[string]interface{}
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.respondClaudeError(c, http.StatusBadRequest, "invalid_request_error", "无法读取请求体")
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		h.respondClaudeError(c, http.StatusBadRequest, "invalid_request_error", "无效的 JSON 格式")
		return
	}

	// 2. 获取模型名称
	modelName, ok := req["model"].(string)
	if !ok || modelName == "" {
		h.respondClaudeError(c, http.StatusBadRequest, "invalid_request_error", "缺少 model 参数")
		return
	}

	// 3. 查找统一模型
	unifiedModel, err := h.mappingService.GetModelByName(modelName)
	if err != nil {
		h.respondClaudeError(c, http.StatusNotFound, "not_found_error", fmt.Sprintf("未找到模型: %s", modelName))
		return
	}

	// 4. 获取该模型的所有映射
	mappings, err := h.mappingService.GetMappingsByModelID(unifiedModel.ID)
	if err != nil || len(mappings) == 0 {
		h.respondClaudeError(c, http.StatusNotFound, "not_found_error", fmt.Sprintf("模型 %s 没有可用的映射", modelName))
		return
	}

	// 5. 选择一个可用的映射（负载均衡）
	selectedMapping := h.selectMapping(mappings)
	if selectedMapping == nil {
		h.respondClaudeError(c, http.StatusServiceUnavailable, "overloaded_error", "没有可用的供应商")
		return
	}

	// 6. 获取供应商信息
	prov, err := h.providerService.GetProvider(selectedMapping.ProviderID)
	if err != nil {
		h.respondClaudeError(c, http.StatusInternalServerError, "api_error", "获取供应商信息失败")
		return
	}

	// 7. 替换模型名称
	req["model"] = selectedMapping.TargetModel

	// 8. 转发请求到供应商
	h.forwardRequest(c, prov, req, bodyBytes, "/v1/messages")
}

// respondClaudeError 返回 Claude API 格式的错误响应
func (h *ProxyHandler) respondClaudeError(c *gin.Context, status int, errorType, message string) {
	c.JSON(status, gin.H{
		"type": "error",
		"error": gin.H{
			"type":    errorType,
			"message": message,
		},
	})
}

// selectMapping 选择一个映射（基于权重的负载均衡）
func (h *ProxyHandler) selectMapping(mappings []*models.ModelMapping) *models.ModelMapping {
	// 过滤启用的且供应商健康的映射
	var available []*models.ModelMapping
	var totalWeight int

	for _, m := range mappings {
		if !m.Enabled {
			continue
		}

		// 检查供应商健康状态
		prov, err := h.providerService.GetProvider(m.ProviderID)
		if err != nil || !prov.Enabled || prov.HealthStatus != "healthy" {
			continue
		}

		available = append(available, m)
		totalWeight += m.Weight
	}

	if len(available) == 0 {
		return nil
	}

	// 基于权重随机选择
	if totalWeight == 0 {
		// 如果所有权重都是0，随机选择
		return available[rand.Intn(len(available))]
	}

	// 加权随机
	r := rand.Intn(totalWeight)
	sum := 0
	for _, m := range available {
		sum += m.Weight
		if r < sum {
			return m
		}
	}

	return available[0]
}

// forwardRequest 转发请求到供应商
func (h *ProxyHandler) forwardRequest(c *gin.Context, prov *models.Provider, req map[string]interface{}, originalBody []byte, endpoint string) {
	// 重新序列化请求体
	newBody, err := json.Marshal(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "序列化请求失败",
		})
		return
	}

	// 构建目标 URL
	targetURL := strings.TrimSuffix(prov.BaseURL, "/") + endpoint

	// 创建新请求
	proxyReq, err := http.NewRequest("POST", targetURL, bytes.NewBuffer(newBody))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "创建代理请求失败",
		})
		return
	}

	// 设置基本请求头
	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("Authorization", "Bearer "+prov.APIKey)

	// 针对 Claude Messages API 设置特殊请求头
	if endpoint == "/v1/messages" {
		// 传递 anthropic-version 头（如果客户端提供了的话）
		if version := c.GetHeader("anthropic-version"); version != "" {
			proxyReq.Header.Set("anthropic-version", version)
		} else {
			// 使用默认版本
			proxyReq.Header.Set("anthropic-version", "2023-06-01")
		}

		// 传递 anthropic-beta 头（如果客户端提供了的话）
		if beta := c.GetHeader("anthropic-beta"); beta != "" {
			proxyReq.Header.Set("anthropic-beta", beta)
		}
	}

	// 复制其他相关请求头
	for key, values := range c.Request.Header {
		if key != "Host" && key != "Authorization" &&
		   key != "Anthropic-Version" && key != "Anthropic-Beta" {
			for _, value := range values {
				proxyReq.Header.Add(key, value)
			}
		}
	}

	// 发送请求
	client := &http.Client{
		Timeout: 120 * time.Second,
	}

	resp, err := client.Do(proxyReq)
	if err != nil {
		log.Printf("转发请求失败 [Provider: %s]: %v", prov.Name, err)
		c.JSON(http.StatusBadGateway, gin.H{
			"error": fmt.Sprintf("请求供应商失败: %v", err),
		})
		return
	}
	defer resp.Body.Close()

	// 复制响应头
	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}

	// 返回响应
	c.Status(resp.StatusCode)
	io.Copy(c.Writer, resp.Body)

	log.Printf("✅ 请求成功 [Model: %s -> Provider: %s/%s] Status: %d",
		req["model"], prov.Name, req["model"], resp.StatusCode)
}
