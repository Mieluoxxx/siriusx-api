package handlers

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Mieluoxxx/Siriusx-API/internal/converter"
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

	// 📝 记录接收到的请求
	log.Printf("📥 [ChatCompletions] 收到请求 - 模型: %s, IP: %s", modelName, c.ClientIP())

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

	// 8. 清洗请求参数（移除不兼容的字段）
	h.sanitizeRequest(req, prov.Name)

	// 📝 记录映射选择和转发信息
	log.Printf("🔀 [ChatCompletions] 映射选择 - 统一模型: %s -> 供应商: %s, 目标模型: %s",
		modelName, prov.Name, selectedMapping.TargetModel)

	// 9. 转发请求到供应商
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

	// 📝 记录接收到的请求
	log.Printf("📥 [Messages] 收到请求 - 模型: %s, IP: %s", modelName, c.ClientIP())

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

	// 6.1 规范化 Claude 请求，兼容简化格式
	h.normalizeClaudePayload(req)

	// 6.2 如果上游是 OpenAI 兼容接口，执行 Claude→OpenAI 转换
	if h.shouldConvertToOpenAI(prov, selectedMapping) {
		req["model"] = selectedMapping.TargetModel
		h.sanitizeRequest(req, prov.Name)
		log.Printf("🔁 [Messages] 检测到 OpenAI 上游，执行 Claude→OpenAI 转换 [Provider: %s, Target: %s]", prov.Name, selectedMapping.TargetModel)
		h.forwardClaudeViaOpenAI(c, prov, selectedMapping, req)
		return
	}

	// 7. 替换模型名称
	req["model"] = selectedMapping.TargetModel

	// 8. 清洗请求参数（移除不兼容的字段）
	h.sanitizeRequest(req, prov.Name)

	// 📝 记录映射选择和转发信息
	log.Printf("🔀 [Messages] 映射选择 - 统一模型: %s -> 供应商: %s, 目标模型: %s",
		modelName, prov.Name, selectedMapping.TargetModel)

	// 9. 转发请求到供应商
	h.forwardRequest(c, prov, req, bodyBytes, "/v1/messages")
}

// MessagesCountTokens 计算 Claude 请求的 token 用量（本地估算）
func (h *ProxyHandler) MessagesCountTokens(c *gin.Context) {
	var req map[string]interface{}
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.respondClaudeError(c, http.StatusBadRequest, "invalid_request_error", "无法读取请求体")
		return
	}

	if len(bodyBytes) == 0 {
		h.respondClaudeError(c, http.StatusBadRequest, "invalid_request_error", "请求体不能为空")
		return
	}

	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		h.respondClaudeError(c, http.StatusBadRequest, "invalid_request_error", "无效的 JSON 格式")
		return
	}

	h.normalizeClaudePayload(req)
	inputTokens := calculateInputTokens(req)

	response := gin.H{
		"type": "message",
		"usage": gin.H{
			"input_tokens":                inputTokens,
			"output_tokens":               0,
			"cache_creation_input_tokens": 0,
			"cache_read_input_tokens":     0,
		},
	}

	c.JSON(http.StatusOK, response)
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

	// 📝 记录转发详情
	log.Printf("➡️  [转发] 目标URL: %s, 请求体大小: %d bytes", targetURL, len(newBody))

	// 创建新请求
	proxyReq, err := http.NewRequest("POST", targetURL, bytes.NewBuffer(newBody))
	if err != nil {
		log.Printf("❌ [转发失败] 创建请求失败: %v", err)
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
		log.Printf("❌ [转发失败] Provider: %s, 错误: %v", prov.Name, err)

		c.JSON(http.StatusBadGateway, gin.H{
			"error": fmt.Sprintf("请求供应商失败: %v", err),
		})
		return
	}
	defer resp.Body.Close()

	// 检测是否是流式响应 (通过 Content-Type 判断)
	contentType := resp.Header.Get("Content-Type")
	isStreamResponse := strings.Contains(contentType, "text/event-stream") ||
		strings.Contains(contentType, "stream")

	if isStreamResponse {
		log.Printf("🌊 [流式响应] 检测到流式响应 (Content-Type: %s)，开始流式转发...", contentType)

		// 复制响应头
		for key, values := range resp.Header {
			for _, value := range values {
				c.Header(key, value)
			}
		}

		// 设置响应状态
		c.Status(resp.StatusCode)

		// 流式转发响应体
		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			log.Printf("❌ [流式转发失败] ResponseWriter 不支持流式传输")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "不支持流式传输",
			})
			return
		}

		// 边读边写，实现真正的流式转发
		buffer := make([]byte, 4096)
		totalBytes := 0
		for {
			n, readErr := resp.Body.Read(buffer)
			if n > 0 {
				totalBytes += n
				if _, writeErr := c.Writer.Write(buffer[:n]); writeErr != nil {
					log.Printf("❌ [流式转发] 写入失败: %v", writeErr)
					return
				}
				flusher.Flush() // 立即刷新，确保客户端能实时接收
			}
			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				log.Printf("❌ [流式转发] 读取失败: %v", readErr)
				return
			}
		}

		log.Printf("✅ [完成] 流式响应转发完成，共 %d bytes", totalBytes)
		return
	}

	// 非流式响应：先读取原始响应体
	rawRespBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("❌ [响应失败] 读取响应体失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "读取响应失败",
		})
		return
	}

	// 检查是否是 gzip 压缩 (通过魔术字节检测)
	var respBody []byte
	isGzipped := len(rawRespBody) >= 2 && rawRespBody[0] == 0x1f && rawRespBody[1] == 0x8b

	if isGzipped || resp.Header.Get("Content-Encoding") == "gzip" {
		log.Printf("🗜️  [响应] 检测到 gzip 压缩响应，进行解压... (来源: %s)",
			func() string {
				if isGzipped && resp.Header.Get("Content-Encoding") == "gzip" {
					return "Header+MagicBytes"
				} else if isGzipped {
					return "MagicBytes"
				}
				return "Header"
			}())

		gzipReader, err := gzip.NewReader(bytes.NewReader(rawRespBody))
		if err != nil {
			log.Printf("❌ [响应失败] gzip 解压失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "gzip 解压失败",
			})
			return
		}
		defer gzipReader.Close()

		respBody, err = io.ReadAll(gzipReader)
		if err != nil {
			log.Printf("❌ [响应失败] 读取解压后的响应失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "读取解压后的响应失败",
			})
			return
		}
		log.Printf("✅ [响应] gzip 解压成功，解压前: %d bytes, 解压后: %d bytes",
			len(rawRespBody), len(respBody))
	} else {
		respBody = rawRespBody
	}

	// 📝 记录响应状态和大小
	log.Printf("⬅️  [响应] Provider: %s, 状态码: %d, 响应体大小: %d bytes",
		prov.Name, resp.StatusCode, len(respBody))

	// 解析响应获取token信息
	var respData map[string]interface{}
	promptTokens := 0
	completionTokens := 0
	totalTokens := 0

	if err := json.Unmarshal(respBody, &respData); err == nil {
		// 检查是否是错误响应
		if errorData, ok := respData["error"]; ok {
			log.Printf("⚠️  [错误响应] Provider返回错误: %+v", errorData)
		}

		log.Printf("🔍 解析响应成功，查找usage字段...")

		// 尝试从响应中获取usage信息
		usageFound := false
		if usage, ok := respData["usage"].(map[string]interface{}); ok {
			log.Printf("✅ 找到usage字段: %+v", usage)
			usageFound = true

			if pt, ok := usage["prompt_tokens"].(float64); ok {
				promptTokens = int(pt)
				log.Printf("✅ prompt_tokens: %d", promptTokens)
			} else if pt, ok := usage["input_tokens"].(float64); ok {
				promptTokens = int(pt)
				log.Printf("✅ input_tokens: %d", promptTokens)
			}

			if ct, ok := usage["completion_tokens"].(float64); ok {
				completionTokens = int(ct)
				log.Printf("✅ completion_tokens: %d", completionTokens)
			} else if ct, ok := usage["output_tokens"].(float64); ok {
				completionTokens = int(ct)
				log.Printf("✅ output_tokens: %d", completionTokens)
			}

			if tt, ok := usage["total_tokens"].(float64); ok {
				totalTokens = int(tt)
			} else {
				totalTokens = promptTokens + completionTokens
			}
			log.Printf("📊 从响应获取的token统计: prompt=%d, completion=%d, total=%d", promptTokens, completionTokens, totalTokens)
		} else {
			log.Printf("❌ 未找到usage字段，响应体keys: %v", getKeys(respData))
		}

		// 如果usage中的token为0或未找到usage，使用估算方法
		if !usageFound || (promptTokens == 0 && completionTokens == 0) {
			log.Printf("⚠️  响应中token为0或未找到usage，使用估算方法...")

			// 估算输入tokens
			estimatedPromptTokens := calculateInputTokens(req)

			// 估算输出tokens
			responseText := extractResponseText(respData)
			estimatedCompletionTokens := estimateTokens(responseText)

			// 如果原始值为0，使用估算值
			if promptTokens == 0 {
				promptTokens = estimatedPromptTokens
			}
			if completionTokens == 0 {
				completionTokens = estimatedCompletionTokens
			}
			totalTokens = promptTokens + completionTokens

			log.Printf("🔢 使用估算的token统计: prompt=%d, completion=%d, total=%d", promptTokens, completionTokens, totalTokens)
		}
	} else {
		// JSON 解析失败，打印响应体前 200 个字符用于调试
		preview := string(respBody)
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		log.Printf("❌ JSON解析失败: %v", err)
		log.Printf("📄 响应体预览 (前200字符): %s", preview)
		log.Printf("⚠️  使用估算方法...")
		// JSON解析失败，完全使用估算
		promptTokens = calculateInputTokens(req)
		// 无法从响应获取文本，设为0
		completionTokens = 0
		totalTokens = promptTokens
	}

	// 复制响应头
	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}

	// 返回响应
	c.Status(resp.StatusCode)
	c.Writer.Write(respBody)

	// 📝 记录最终响应状态
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf("✅ [完成] 状态: %d, Tokens: prompt=%d + completion=%d = %d",
			resp.StatusCode, promptTokens, completionTokens, totalTokens)
	} else {
		log.Printf("❌ [完成] 状态: %d (错误响应)", resp.StatusCode)
	}
}

// forwardClaudeViaOpenAI 将 Claude Messages 请求转换为 OpenAI Chat Completions 请求再转发
func (h *ProxyHandler) forwardClaudeViaOpenAI(c *gin.Context, prov *models.Provider, mapping *models.ModelMapping, req map[string]interface{}) {
	payloadBytes, err := json.Marshal(req)
	if err != nil {
		log.Printf("❌ [转换失败] 无法序列化 Claude 请求: %v", err)
		h.respondClaudeError(c, http.StatusInternalServerError, "api_error", "生成上游请求失败")
		return
	}

	var claudeReq converter.ClaudeRequest
	if err := json.Unmarshal(payloadBytes, &claudeReq); err != nil {
		preview := string(payloadBytes)
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		log.Printf("❌ [解析失败] Claude 请求无法解析: %v", err)
		log.Printf("📄 Claude 请求预览 (前200字符): %s", preview)
		h.respondClaudeError(c, http.StatusBadRequest, "invalid_request_error", "请求格式不符合 Claude Messages 规范")
		return
	}

	claudeReq.Model = mapping.TargetModel

	openaiReq, err := converter.ConvertClaudeToOpenAI(&claudeReq)
	if err != nil {
		log.Printf("❌ [转换失败] Claude→OpenAI: %v", err)
		h.respondClaudeError(c, http.StatusBadRequest, "invalid_request_error", "Claude 请求转换 OpenAI 格式失败")
		return
	}

	openaiReq.Model = mapping.TargetModel

	openaiBody, err := json.Marshal(openaiReq)
	if err != nil {
		log.Printf("❌ [序列化失败] OpenAI 请求: %v", err)
		h.respondClaudeError(c, http.StatusInternalServerError, "api_error", "生成上游请求失败")
		return
	}

	targetURL := strings.TrimSuffix(prov.BaseURL, "/") + "/v1/chat/completions"
	log.Printf("➡️  [转发] Claude→OpenAI 目标URL: %s, 请求体大小: %d bytes", targetURL, len(openaiBody))

	proxyReq, err := http.NewRequest("POST", targetURL, bytes.NewBuffer(openaiBody))
	if err != nil {
		log.Printf("❌ [转发失败] 创建 OpenAI 请求失败: %v", err)
		h.respondClaudeError(c, http.StatusInternalServerError, "api_error", "创建代理请求失败")
		return
	}

	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("Authorization", "Bearer "+prov.APIKey)
	if openaiReq.Stream {
		proxyReq.Header.Set("Accept", "text/event-stream")
	}

	for key, values := range c.Request.Header {
		if key == "Host" || key == "Authorization" {
			continue
		}
		lowerKey := strings.ToLower(key)
		if strings.HasPrefix(lowerKey, "anthropic") {
			continue
		}
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(proxyReq)
	if err != nil {
		log.Printf("❌ [转发失败] Provider: %s, 错误: %v", prov.Name, err)
		h.respondClaudeError(c, http.StatusBadGateway, "api_error", fmt.Sprintf("请求供应商失败: %v", err))
		return
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	isStreamResponse := strings.Contains(strings.ToLower(contentType), "text/event-stream")

	if isStreamResponse {
		convertedReader, err := converter.ConvertStream(c.Request.Context(), resp.Body)
		if err != nil {
			log.Printf("❌ [流式转换失败] Provider: %s, 错误: %v", prov.Name, err)
			h.respondClaudeError(c, http.StatusBadGateway, "api_error", "上游流式响应转换失败")
			return
		}

		for key, values := range resp.Header {
			lowerKey := strings.ToLower(key)
			if lowerKey == "content-type" {
				c.Header(key, "text/event-stream; charset=utf-8")
				continue
			}
			for _, value := range values {
				c.Header(key, value)
			}
		}

		c.Status(resp.StatusCode)

		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			log.Printf("❌ [流式转发失败] ResponseWriter 不支持流式传输")
			h.respondClaudeError(c, http.StatusInternalServerError, "api_error", "不支持流式传输")
			return
		}

		buffer := make([]byte, 4096)
		totalBytes := 0
		for {
			n, readErr := convertedReader.Read(buffer)
			if n > 0 {
				totalBytes += n
				if _, writeErr := c.Writer.Write(buffer[:n]); writeErr != nil {
					log.Printf("❌ [流式转发] 写入失败: %v", writeErr)
					return
				}
				flusher.Flush()
			}
			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				log.Printf("❌ [流式转发] 读取失败: %v", readErr)
				return
			}
		}

		log.Printf("✅ [完成] Claude 流式响应转换完成，共 %d bytes", totalBytes)
		return
	}

	rawRespBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("❌ [响应失败] 读取 OpenAI 响应体失败: %v", err)
		h.respondClaudeError(c, http.StatusInternalServerError, "api_error", "读取上游响应失败")
		return
	}

	respBody, wasGzip, err := decompressIfNeeded(rawRespBody, resp.Header)
	if err != nil {
		log.Printf("❌ [解压失败] OpenAI 响应: %v", err)
		h.respondClaudeError(c, http.StatusInternalServerError, "api_error", "解压上游响应失败")
		return
	}
	if wasGzip {
		log.Printf("🗜️  [响应] OpenAI 响应已解压缩 (Provider: %s)", prov.Name)
	}

	if resp.StatusCode >= 400 {
		var openaiErr struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
			} `json:"error"`
		}
		if err := json.Unmarshal(respBody, &openaiErr); err == nil && openaiErr.Error.Message != "" {
			h.respondClaudeError(c, resp.StatusCode, "api_error", openaiErr.Error.Message)
			return
		}

		preview := string(respBody)
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		log.Printf("❌ [OpenAI 错误响应] 状态: %d, 内容: %s", resp.StatusCode, preview)
		h.respondClaudeError(c, resp.StatusCode, "api_error", "上游返回错误响应")
		return
	}

	var openaiResp converter.OpenAIResponse
	if err := json.Unmarshal(respBody, &openaiResp); err != nil {
		log.Printf("❌ [解析失败] OpenAI 响应: %v", err)
		preview := string(respBody)
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		log.Printf("📄 OpenAI 响应体预览 (前200字符): %s", preview)
		h.respondClaudeError(c, http.StatusInternalServerError, "api_error", "解析上游响应失败")
		return
	}

	claudeResp, err := converter.ConvertOpenAIToClaude(&openaiResp)
	if err != nil {
		log.Printf("❌ [转换失败] OpenAI→Claude: %v", err)
		h.respondClaudeError(c, http.StatusInternalServerError, "api_error", "上游响应转换 Claude 格式失败")
		return
	}

	respBytes, err := json.Marshal(claudeResp)
	if err != nil {
		log.Printf("❌ [序列化失败] Claude 响应: %v", err)
		h.respondClaudeError(c, http.StatusInternalServerError, "api_error", "序列化响应失败")
		return
	}

	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}
	c.Header("Content-Type", "application/json")

	c.Status(resp.StatusCode)
	if _, err := c.Writer.Write(respBytes); err != nil {
		log.Printf("❌ [响应写入失败] Claude 响应: %v", err)
		return
	}

	log.Printf("✅ [完成] Claude 非流式响应转换成功，状态: %d", resp.StatusCode)
}

// shouldConvertToOpenAI 判断是否需要将 Claude 请求转换为 OpenAI 兼容请求
func (h *ProxyHandler) shouldConvertToOpenAI(prov *models.Provider, mapping *models.ModelMapping) bool {
	target := strings.ToLower(mapping.TargetModel)
	if strings.Contains(target, "claude") {
		return false
	}

	baseURL := strings.ToLower(prov.BaseURL)
	if strings.Contains(baseURL, "anthropic") {
		return false
	}

	return true
}

// decompressIfNeeded 如果响应是 gzip 压缩则解压缩
func decompressIfNeeded(raw []byte, header http.Header) ([]byte, bool, error) {
	isGzipped := len(raw) >= 2 && raw[0] == 0x1f && raw[1] == 0x8b
	if isGzipped || strings.EqualFold(header.Get("Content-Encoding"), "gzip") {
		reader, err := gzip.NewReader(bytes.NewReader(raw))
		if err != nil {
			return nil, false, err
		}
		decompressed, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			return nil, false, err
		}
		return decompressed, true, nil
	}

	return raw, false, nil
}

// normalizeClaudePayload 兼容简化版 Claude 消息格式
func (h *ProxyHandler) normalizeClaudePayload(req map[string]interface{}) {
	// 规范 messages[].content
	if messages, ok := req["messages"].([]interface{}); ok {
		for idx, rawMsg := range messages {
			msgMap, ok := rawMsg.(map[string]interface{})
			if !ok {
				continue
			}

			// content 为字符串时转换成标准 text block
			if contentStr, ok := msgMap["content"].(string); ok {
				msgMap["content"] = []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": contentStr,
					},
				}
				messages[idx] = msgMap
				continue
			}

			// content 为 map（单个 block）时，转为数组
			if contentMap, ok := msgMap["content"].(map[string]interface{}); ok {
				msgMap["content"] = []interface{}{contentMap}
				messages[idx] = msgMap
				continue
			}
		}
		req["messages"] = messages
	}

	// 规范 system 字段
	if systemVal, exists := req["system"]; exists {
		switch val := systemVal.(type) {
		case []interface{}:
			var parts []string
			for _, item := range val {
				switch v := item.(type) {
				case string:
					parts = append(parts, v)
				case map[string]interface{}:
					if text, ok := v["text"].(string); ok {
						parts = append(parts, text)
					}
				}
			}
			req["system"] = strings.Join(parts, "\n")
		case map[string]interface{}:
			if text, ok := val["text"].(string); ok {
				req["system"] = text
			}
		}
	}
}

// sanitizeRequest 清洗请求参数，移除不兼容的字段
func (h *ProxyHandler) sanitizeRequest(req map[string]interface{}, providerName string) {
	// 针对智谱 GLM 等对参数格式要求严格的 API
	// 移除可能导致错误的 Claude 特有参数

	// 1. 移除 tools 参数（如果为空或格式不正确）
	if tools, ok := req["tools"].([]interface{}); ok {
		// 检查 tools 是否为空或包含无效数据
		if len(tools) == 0 {
			delete(req, "tools")
			log.Printf("🧹 已移除空的 tools 参数 [Provider: %s]", providerName)
		} else {
			// 检查第一个 tool 的 type 字段
			if tool, ok := tools[0].(map[string]interface{}); ok {
				if toolType, exists := tool["type"]; !exists || toolType == "" {
					delete(req, "tools")
					log.Printf("🧹 已移除无效的 tools 参数 [Provider: %s]", providerName)
				}
			}
		}
	}

	// 2. 移除其他 Claude 特有的参数
	claudeSpecificParams := []string{
		"anthropic_version",
		"metadata",
	}

	for _, param := range claudeSpecificParams {
		if _, exists := req[param]; exists {
			delete(req, param)
			log.Printf("🧹 已移除 Claude 特有参数: %s [Provider: %s]", param, providerName)
		}
	}
}

// getKeys 获取map的所有keys（用于调试）
func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// estimateTokens 估算文本的token数量
// 基于经验公式：英文约4字符=1token，中文约1.5字符=1token
func estimateTokens(text string) int {
	if text == "" {
		return 0
	}

	charCount := utf8.RuneCountInString(text)
	// 统计中文字符数
	chineseCount := 0
	for _, r := range text {
		if r >= 0x4e00 && r <= 0x9fa5 {
			chineseCount++
		}
	}

	// 中文字符按1.5字符=1token计算，其他按4字符=1token计算
	englishChars := charCount - chineseCount
	tokens := (chineseCount*2 + englishChars) / 3

	// 至少返回1个token（如果有内容的话）
	if tokens == 0 && charCount > 0 {
		return 1
	}

	return tokens
}

// calculateInputTokens 计算输入token数量
func calculateInputTokens(req map[string]interface{}) int {
	totalTokens := 0

	// 计算messages的tokens
	if messages, ok := req["messages"].([]interface{}); ok {
		for _, msg := range messages {
			if msgMap, ok := msg.(map[string]interface{}); ok {
				if content, ok := msgMap["content"].(string); ok {
					totalTokens += estimateTokens(content)
				} else if contentArray, ok := msgMap["content"].([]interface{}); ok {
					// 处理复杂content（数组格式）
					for _, contentPart := range contentArray {
						if partMap, ok := contentPart.(map[string]interface{}); ok {
							if text, ok := partMap["text"].(string); ok {
								totalTokens += estimateTokens(text)
							}
						}
					}
				}
			}
		}
	}

	// 计算system的tokens
	if system, ok := req["system"].(string); ok {
		totalTokens += estimateTokens(system)
	}

	return totalTokens
}

// extractResponseText 从响应中提取文本内容
func extractResponseText(respData map[string]interface{}) string {
	var text strings.Builder

	// OpenAI格式: choices[0].message.content
	if choices, ok := respData["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					return content
				}
			}
		}
	}

	// Claude格式: content[].text
	if content, ok := respData["content"].([]interface{}); ok {
		for _, item := range content {
			if contentMap, ok := item.(map[string]interface{}); ok {
				if contentText, ok := contentMap["text"].(string); ok {
					text.WriteString(contentText)
				}
			}
		}
		return text.String()
	}

	return ""
}
