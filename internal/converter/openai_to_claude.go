package converter

import (
	"encoding/json"
	"fmt"
)

// ConvertOpenAIToClaude 将 OpenAI Chat Completions API 响应转换为 Claude Messages API 响应
func ConvertOpenAIToClaude(resp *OpenAIResponse) (*ClaudeResponse, error) {
	// 验证输入
	if err := ValidateNonNil(resp, "OpenAI响应"); err != nil {
		return nil, NewConversionError("response", "验证失败", err)
	}

	if len(resp.Choices) == 0 {
		return nil, NewConversionError("response", "响应中没有choices", nil)
	}

	// 取第一个 choice
	choice := resp.Choices[0]

	// 创建 Claude 响应
	claudeResp := &ClaudeResponse{
		ID:    ConvertIDOpenAIToClaude(resp.ID),
		Type:  ClaudeTypeMessage,
		Role:  ClaudeRoleAssistant,
		Model: resp.Model,
		Usage: ClaudeUsage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		},
	}

	// 转换 stop_reason
	claudeResp.StopReason = ConvertFinishReasonToStopReason(choice.FinishReason)

	// 转换 content
	content, err := convertResponseContent(choice.Message)
	if err != nil {
		return nil, NewConversionError("response", "转换内容失败", err)
	}
	claudeResp.Content = content

	return claudeResp, nil
}

// convertResponseContent 转换响应内容
func convertResponseContent(msg OpenAIMessage) ([]ClaudeContentBlock, error) {
	var content []ClaudeContentBlock

	// 处理文本内容
	if msg.Content != nil {
		textContent := ExtractTextFromContent(msg.Content)
		if textContent != "" {
			content = append(content, ClaudeContentBlock{
				Type: ContentTypeText,
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
			Type: ContentTypeText,
			Text: &emptyText,
		})
	}

	return content, nil
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
		Type:  ContentTypeToolUse,
		ID:    &toolCall.ID,
		Name:  &toolCall.Function.Name,
		Input: input,
	}, nil
}
