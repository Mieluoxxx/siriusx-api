package converter

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ConvertClaudeToOpenAI 将 Claude Messages API 请求转换为 OpenAI Chat Completions API 请求
func ConvertClaudeToOpenAI(req *ClaudeRequest) (*OpenAIRequest, error) {
	// 验证输入
	if err := ValidateNonNil(req, "Claude请求"); err != nil {
		return nil, NewConversionError("request", "验证失败", err)
	}

	// 创建 OpenAI 请求
	openaiReq := &OpenAIRequest{
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stream:      req.Stream,
		Stop:        req.StopSequences,
	}

	// 转换 messages
	messages, err := convertMessages(req.Messages, req.System)
	if err != nil {
		return nil, NewConversionError("request", "转换消息失败", err)
	}
	openaiReq.Messages = messages

	// 转换 tools
	if len(req.Tools) > 0 {
		openaiReq.Tools = convertTools(req.Tools)
	}

	// 转换 tool_choice
	if req.ToolChoice != nil {
		openaiReq.ToolChoice = convertToolChoice(req.ToolChoice)
	}

	return openaiReq, nil
}

// convertMessages 转换消息数组
func convertMessages(claudeMessages []ClaudeMessage, system string) ([]OpenAIMessage, error) {
	var messages []OpenAIMessage

	// 如果有 system 参数，添加为第一条消息
	if system != "" {
		messages = append(messages, OpenAIMessage{
			Role:    "system",
			Content: system,
		})
	}

	// 转换每条消息
	for _, msg := range claudeMessages {
		converted, err := convertSingleMessage(msg)
		if err != nil {
			return nil, fmt.Errorf("转换消息失败: %w", err)
		}
		messages = append(messages, converted...)
	}

	return messages, nil
}

// convertSingleMessage 转换单条消息
func convertSingleMessage(msg ClaudeMessage) ([]OpenAIMessage, error) {
	// 检查是否有 tool_result
	hasToolResult := false
	for _, block := range msg.Content {
		if block.Type == ContentTypeToolResult {
			hasToolResult = true
			break
		}
	}

	// 如果有 tool_result，需要特殊处理
	if hasToolResult {
		return convertToolResultMessage(msg)
	}

	// 普通消息转换
	switch msg.Role {
	case ClaudeRoleUser:
		return []OpenAIMessage{convertUserMessage(msg)}, nil
	case ClaudeRoleAssistant:
		return []OpenAIMessage{convertAssistantMessage(msg)}, nil
	default:
		return nil, fmt.Errorf("不支持的角色: %s", msg.Role)
	}
}

// convertUserMessage 转换用户消息
func convertUserMessage(msg ClaudeMessage) OpenAIMessage {
	// 检查是否只有一个文本块
	if len(msg.Content) == 1 && msg.Content[0].Type == ContentTypeText && msg.Content[0].Text != nil {
		return OpenAIMessage{
			Role:    ClaudeRoleUser,
			Content: *msg.Content[0].Text,
		}
	}

	// 多个内容块，需要转换为数组
	var contentBlocks []OpenAIContentBlock
	for _, block := range msg.Content {
		switch block.Type {
		case ContentTypeText:
			if block.Text != nil {
				contentBlocks = append(contentBlocks, OpenAIContentBlock{
					Type: ContentTypeText,
					Text: block.Text,
				})
			}
		case ContentTypeImage:
			if block.Source != nil {
				// 转换图片为 data URI
				dataURI := fmt.Sprintf("data:%s;base64,%s",
					block.Source.MediaType,
					block.Source.Data)
				contentBlocks = append(contentBlocks, OpenAIContentBlock{
					Type: "image_url",
					ImageURL: &OpenAIImageURL{
						URL: dataURI,
					},
				})
			}
		}
	}

	return OpenAIMessage{
		Role:    ClaudeRoleUser,
		Content: contentBlocks,
	}
}

// convertAssistantMessage 转换助手消息
func convertAssistantMessage(msg ClaudeMessage) OpenAIMessage {
	var textParts []string
	var toolCalls []OpenAIToolCall

	// 分离文本和工具调用
	for _, block := range msg.Content {
		switch block.Type {
		case ContentTypeText:
			if block.Text != nil {
				textParts = append(textParts, *block.Text)
			}
		case ContentTypeToolUse:
			if block.ID != nil && block.Name != nil {
				// 序列化 input 为 JSON 字符串
				args, _ := json.Marshal(block.Input)
				toolCalls = append(toolCalls, OpenAIToolCall{
					ID:   *block.ID,
					Type: OpenAIToolTypeFunction,
					Function: OpenAIFunctionCall{
						Name:      *block.Name,
						Arguments: string(args),
					},
				})
			}
		}
	}

	// 合并文本内容
	content := strings.Join(textParts, "")

	return OpenAIMessage{
		Role:      ClaudeRoleAssistant,
		Content:   content,
		ToolCalls: toolCalls,
	}
}

// convertToolResultMessage 转换 tool_result 消息
func convertToolResultMessage(msg ClaudeMessage) ([]OpenAIMessage, error) {
	var messages []OpenAIMessage

	// 每个 tool_result 转换为独立的消息
	for _, block := range msg.Content {
		if block.Type == ContentTypeToolResult {
			if block.ToolUseID == nil {
				return nil, fmt.Errorf("tool_result 缺少 tool_use_id")
			}

			content := ""
			if block.Content != nil {
				content = *block.Content
			}

			messages = append(messages, OpenAIMessage{
				Role:       "tool",
				Content:    content,
				ToolCallID: *block.ToolUseID,
			})
		}
	}

	return messages, nil
}

// convertTools 转换工具定义
func convertTools(claudeTools []ClaudeTool) []OpenAITool {
	var openaiTools []OpenAITool

	for _, tool := range claudeTools {
		openaiTools = append(openaiTools, OpenAITool{
			Type: OpenAIToolTypeFunction,
			Function: OpenAIFunctionDef{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.InputSchema,
			},
		})
	}

	return openaiTools
}

// convertToolChoice 转换 tool_choice
func convertToolChoice(choice *ClaudeToolChoice) any {
	if choice == nil {
		return nil
	}

	switch choice.Type {
	case "auto":
		return "auto"
	case "any":
		return "required"
	case "tool":
		if choice.Name != nil {
			return map[string]any{
				"type": OpenAIToolTypeFunction,
				"function": map[string]string{
					"name": *choice.Name,
				},
			}
		}
	}

	return "auto"
}
