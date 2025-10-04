package converter

import (
	"fmt"
	"reflect"
	"strings"
)

// ========================
// 公共转换函数
// ========================
// 遵循 KISS 和 DRY 原则，提取重复的转换逻辑

// ConvertIDOpenAIToClaude 转换 OpenAI ID 为 Claude ID
// OpenAI: chatcmpl-xxx → Claude: msg_xxx
func ConvertIDOpenAIToClaude(openaiID string) string {
	if openaiID == "" {
		return "msg_unknown"
	}
	// 保留原始 ID，添加 msg_ 前缀
	if id, found := strings.CutPrefix(openaiID, "chatcmpl-"); found {
		return "msg_" + id
	}
	return "msg_" + openaiID
}

// ConvertIDClaudeToOpenAI 转换 Claude ID 为 OpenAI ID
// Claude: msg_xxx → OpenAI: chatcmpl-xxx
func ConvertIDClaudeToOpenAI(claudeID string) string {
	if claudeID == "" {
		return "chatcmpl-unknown"
	}
	// 移除 msg_ 前缀，添加 chatcmpl- 前缀
	if id, found := strings.CutPrefix(claudeID, "msg_"); found {
		return "chatcmpl-" + id
	}
	return "chatcmpl-" + claudeID
}

// ========================
// 停止原因转换
// ========================

// ConvertFinishReasonToStopReason 转换 OpenAI finish_reason 为 Claude stop_reason
func ConvertFinishReasonToStopReason(finishReason string) string {
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

// ConvertStopReasonToFinishReason 转换 Claude stop_reason 为 OpenAI finish_reason
func ConvertStopReasonToFinishReason(stopReason string) string {
	switch stopReason {
	case "end_turn":
		return "stop"
	case "max_tokens":
		return "length"
	case "tool_use":
		return "tool_calls"
	case "stop_sequence":
		return "stop"
	default:
		return "stop"
	}
}

// ========================
// 文本内容提取
// ========================

// ExtractTextFromContent 从 any 类型的 content 中提取文本
// 支持 string 或 []any 类型
func ExtractTextFromContent(content any) string {
	switch v := content.(type) {
	case string:
		return v
	case []any:
		// 提取所有文本块并合并
		var texts []string
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				if m["type"] == "text" {
					if text, ok := m["text"].(string); ok {
						texts = append(texts, text)
					}
				}
			}
		}
		return strings.Join(texts, "")
	default:
		return ""
	}
}

// ExtractTextFromClaudeContent 从 Claude ContentBlock 数组中提取所有文本
func ExtractTextFromClaudeContent(content []ClaudeContentBlock) string {
	var texts []string
	for _, block := range content {
		if block.Type == "text" && block.Text != nil {
			texts = append(texts, *block.Text)
		}
	}
	return strings.Join(texts, "")
}

// ========================
// 错误处理工具
// ========================

// ConversionError 转换错误类型
type ConversionError struct {
	Stage   string // 转换阶段：request, response, streaming
	Message string
	Err     error
}

func (e *ConversionError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("转换失败 [%s]: %s: %v", e.Stage, e.Message, e.Err)
	}
	return fmt.Sprintf("转换失败 [%s]: %s", e.Stage, e.Message)
}

func (e *ConversionError) Unwrap() error {
	return e.Err
}

// NewConversionError 创建转换错误
func NewConversionError(stage, message string, err error) error {
	return &ConversionError{
		Stage:   stage,
		Message: message,
		Err:     err,
	}
}

// ========================
// 验证工具
// ========================

// ValidateNonEmpty 验证字符串非空
func ValidateNonEmpty(value, fieldName string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s 不能为空", fieldName)
	}
	return nil
}

// ValidateNonNil 验证指针非空
func ValidateNonNil(value any, fieldName string) error {
	if value == nil {
		return fmt.Errorf("%s 不能为 nil", fieldName)
	}

	// 使用反射检查是否是 nil 指针
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return fmt.Errorf("%s 不能为 nil", fieldName)
	}

	return nil
}

// ========================
// 常量定义
// ========================

const (
	// OpenAI 相关常量
	OpenAIObjectChatCompletion      = "chat.completion"
	OpenAIObjectChatCompletionChunk = "chat.completion.chunk"
	OpenAIToolTypeFunction          = "function"

	// Claude 相关常量
	ClaudeTypeMessage = "message"
	ClaudeRoleAssistant = "assistant"
	ClaudeRoleUser = "user"

	// Content Block Types
	ContentTypeText       = "text"
	ContentTypeImage      = "image"
	ContentTypeToolUse    = "tool_use"
	ContentTypeToolResult = "tool_result"

	// Claude Stream Event Types
	EventTypeMessageStart       = "message_start"
	EventTypeContentBlockStart  = "content_block_start"
	EventTypeContentBlockDelta  = "content_block_delta"
	EventTypeContentBlockStop   = "content_block_stop"
	EventTypeMessageDelta       = "message_delta"
	EventTypeMessageStop        = "message_stop"

	// Delta Types
	DeltaTypeTextDelta      = "text_delta"
	DeltaTypeInputJSONDelta = "input_json_delta"
)
