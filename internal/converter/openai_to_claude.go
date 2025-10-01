package converter

import (
	"encoding/json"
	"fmt"
)

// ConvertOpenAIToClaude 将 OpenAI Chat Completions API 响应转换为 Claude Messages API 响应
func ConvertOpenAIToClaude(resp *OpenAIResponse) (*ClaudeResponse, error) {
	if resp == nil {
		return nil, fmt.Errorf("响应不能为空")
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("响应中没有choices")
	}

	// 取第一个 choice
	choice := resp.Choices[0]

	// 创建 Claude 响应
	claudeResp := &ClaudeResponse{
		ID:    convertID(resp.ID),
		Type:  "message",
		Role:  "assistant",
		Model: resp.Model,
		Usage: ClaudeUsage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		},
	}

	// 转换 stop_reason
	claudeResp.StopReason = convertFinishReason(choice.FinishReason)

	// 转换 content
	content, err := convertResponseContent(choice.Message)
	if err != nil {
		return nil, fmt.Errorf("转换内容失败: %w", err)
	}
	claudeResp.Content = content

	return claudeResp, nil
}

// convertID 转换响应 ID (OpenAI chatcmpl-xxx → Claude msg_xxx)
func convertID(openaiID string) string {
	// 简单映射，实际项目中可能需要保留原 ID 或生成新 ID
	if openaiID == "" {
		return "msg_unknown"
	}
	return "msg_" + openaiID
}

// convertFinishReason 转换结束原因
func convertFinishReason(finishReason string) string {
	switch finishReason {
	case "stop":
		return "end_turn"
	case "length":
		return "max_tokens"
	case "tool_calls":
		return "tool_use"
	case "content_filter":
		return "end_turn" // 或者自定义为 "content_filter"
	default:
		return "end_turn"
	}
}

// convertResponseContent 转换响应内容
func convertResponseContent(msg OpenAIMessage) ([]ClaudeContentBlock, error) {
	var content []ClaudeContentBlock

	// 处理文本内容
	if msg.Content != nil {
		textContent := extractTextContent(msg.Content)
		if textContent != "" {
			content = append(content, ClaudeContentBlock{
				Type: "text",
				Text: &textContent,
			})
		}
	}

	// 处理 tool_calls
	if len(msg.ToolCalls) > 0 {
		for _, toolCall := range msg.ToolCalls {
			block, err := convertToolCallToToolUse(toolCall)
			if err != nil {
				return nil, fmt.Errorf("转换 tool_call 失败: %w", err)
			}
			content = append(content, block)
		}
	}

	// 如果没有任何内容，添加一个空文本块
	if len(content) == 0 {
		emptyText := ""
		content = append(content, ClaudeContentBlock{
			Type: "text",
			Text: &emptyText,
		})
	}

	return content, nil
}

// extractTextContent 提取文本内容
func extractTextContent(content interface{}) string {
	switch v := content.(type) {
	case string:
		return v
	case []interface{}:
		// 如果是数组，提取所有文本块
		var texts []string
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				if m["type"] == "text" {
					if text, ok := m["text"].(string); ok {
						texts = append(texts, text)
					}
				}
			}
		}
		if len(texts) > 0 {
			result := ""
			for _, t := range texts {
				result += t
			}
			return result
		}
	}
	return ""
}

// convertToolCallToToolUse 转换 tool_call 为 tool_use
func convertToolCallToToolUse(toolCall OpenAIToolCall) (ClaudeContentBlock, error) {
	// 解析 arguments JSON 字符串为 map
	var input map[string]interface{}
	if toolCall.Function.Arguments != "" {
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &input); err != nil {
			return ClaudeContentBlock{}, fmt.Errorf("解析 arguments 失败: %w", err)
		}
	} else {
		input = make(map[string]interface{})
	}

	return ClaudeContentBlock{
		Type:  "tool_use",
		ID:    &toolCall.ID,
		Name:  &toolCall.Function.Name,
		Input: input,
	}, nil
}
