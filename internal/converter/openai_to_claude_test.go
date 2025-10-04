package converter

import (
	"testing"
)

// 测试基础文本响应转换
func TestConvertBasicTextResponse(t *testing.T) {
	resp := &OpenAIResponse{
		ID:      "chatcmpl-123",
		Object:  "chat.completion",
		Created: 1677652288,
		Model:   "gpt-4o",
		Choices: []OpenAIChoice{
			{
				Index: 0,
				Message: OpenAIMessage{
					Role:    "assistant",
					Content: "Hello! How can I help you today?",
				},
				FinishReason: "stop",
			},
		},
		Usage: OpenAIUsage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}

	result, err := ConvertOpenAIToClaude(resp)
	if err != nil {
		t.Fatalf("转换失败: %v", err)
	}

	if result.Type != "message" {
		t.Errorf("Type should be 'message', got %s", result.Type)
	}

	if result.Role != "assistant" {
		t.Errorf("Role should be 'assistant', got %s", result.Role)
	}

	if result.Model != "gpt-4o" {
		t.Errorf("Model mismatch, got %s", result.Model)
	}

	if result.StopReason != "end_turn" {
		t.Errorf("StopReason should be 'end_turn', got %s", result.StopReason)
	}

	if len(result.Content) != 1 {
		t.Fatalf("Should have 1 content block, got %d", len(result.Content))
	}

	if result.Content[0].Type != "text" {
		t.Errorf("Content type should be 'text', got %s", result.Content[0].Type)
	}

	if result.Content[0].Text == nil || *result.Content[0].Text != "Hello! How can I help you today?" {
		t.Error("Content text mismatch")
	}

	if result.Usage.InputTokens != 10 {
		t.Errorf("InputTokens mismatch, got %d", result.Usage.InputTokens)
	}

	if result.Usage.OutputTokens != 20 {
		t.Errorf("OutputTokens mismatch, got %d", result.Usage.OutputTokens)
	}
}

// 测试 finish_reason 映射
func TestConvertFinishReason(t *testing.T) {
	tests := []struct {
		name          string
		finishReason  string
		expectedStop  string
	}{
		{"stop", "stop", "end_turn"},
		{"length", "length", "max_tokens"},
		{"tool_calls", "tool_calls", "tool_use"},
		{"content_filter", "content_filter", "end_turn"},
		{"unknown", "unknown", "end_turn"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertFinishReasonToStopReason(tt.finishReason)
			if result != tt.expectedStop {
				t.Errorf("Expected %s, got %s", tt.expectedStop, result)
			}
		})
	}
}

// 测试 tool_calls 响应转换
func TestConvertToolCallsResponse(t *testing.T) {
	resp := &OpenAIResponse{
		ID:      "chatcmpl-456",
		Object:  "chat.completion",
		Created: 1677652288,
		Model:   "gpt-4o",
		Choices: []OpenAIChoice{
			{
				Index: 0,
				Message: OpenAIMessage{
					Role:    "assistant",
					Content: "Let me check the weather for you.",
					ToolCalls: []OpenAIToolCall{
						{
							ID:   "call_123",
							Type: "function",
							Function: OpenAIFunctionCall{
								Name:      "get_weather",
								Arguments: `{"location":"San Francisco","unit":"celsius"}`,
							},
						},
					},
				},
				FinishReason: "tool_calls",
			},
		},
		Usage: OpenAIUsage{
			PromptTokens:     15,
			CompletionTokens: 25,
			TotalTokens:      40,
		},
	}

	result, err := ConvertOpenAIToClaude(resp)
	if err != nil {
		t.Fatalf("转换失败: %v", err)
	}

	if result.StopReason != "tool_use" {
		t.Errorf("StopReason should be 'tool_use', got %s", result.StopReason)
	}

	if len(result.Content) != 2 {
		t.Fatalf("Should have 2 content blocks, got %d", len(result.Content))
	}

	// 检查文本块
	if result.Content[0].Type != "text" {
		t.Errorf("First block should be 'text', got %s", result.Content[0].Type)
	}

	// 检查 tool_use 块
	if result.Content[1].Type != "tool_use" {
		t.Errorf("Second block should be 'tool_use', got %s", result.Content[1].Type)
	}

	if result.Content[1].ID == nil || *result.Content[1].ID != "call_123" {
		t.Error("ToolUse ID mismatch")
	}

	if result.Content[1].Name == nil || *result.Content[1].Name != "get_weather" {
		t.Error("ToolUse Name mismatch")
	}

	if result.Content[1].Input == nil {
		t.Fatal("ToolUse Input should not be nil")
	}

	if result.Content[1].Input["location"] != "San Francisco" {
		t.Error("ToolUse Input location mismatch")
	}

	if result.Content[1].Input["unit"] != "celsius" {
		t.Error("ToolUse Input unit mismatch")
	}
}

// 测试多个 tool_calls
func TestConvertMultipleToolCalls(t *testing.T) {
	resp := &OpenAIResponse{
		ID:      "chatcmpl-789",
		Object:  "chat.completion",
		Created: 1677652288,
		Model:   "gpt-4o",
		Choices: []OpenAIChoice{
			{
				Index: 0,
				Message: OpenAIMessage{
					Role:    "assistant",
					Content: "I'll check both locations.",
					ToolCalls: []OpenAIToolCall{
						{
							ID:   "call_1",
							Type: "function",
							Function: OpenAIFunctionCall{
								Name:      "get_weather",
								Arguments: `{"location":"SF"}`,
							},
						},
						{
							ID:   "call_2",
							Type: "function",
							Function: OpenAIFunctionCall{
								Name:      "get_weather",
								Arguments: `{"location":"NYC"}`,
							},
						},
					},
				},
				FinishReason: "tool_calls",
			},
		},
		Usage: OpenAIUsage{
			PromptTokens:     20,
			CompletionTokens: 30,
			TotalTokens:      50,
		},
	}

	result, err := ConvertOpenAIToClaude(resp)
	if err != nil {
		t.Fatalf("转换失败: %v", err)
	}

	// 应该有 1 个文本块 + 2 个 tool_use 块
	if len(result.Content) != 3 {
		t.Fatalf("Should have 3 content blocks, got %d", len(result.Content))
	}

	if result.Content[0].Type != "text" {
		t.Error("First block should be text")
	}

	if result.Content[1].Type != "tool_use" || *result.Content[1].ID != "call_1" {
		t.Error("Second block should be tool_use with ID call_1")
	}

	if result.Content[2].Type != "tool_use" || *result.Content[2].ID != "call_2" {
		t.Error("Third block should be tool_use with ID call_2")
	}
}

// 测试空响应
func TestConvertNilResponse(t *testing.T) {
	_, err := ConvertOpenAIToClaude(nil)
	if err == nil {
		t.Error("Should return error for nil response")
	}
}

// 测试空 choices
func TestConvertEmptyChoices(t *testing.T) {
	resp := &OpenAIResponse{
		ID:      "chatcmpl-empty",
		Object:  "chat.completion",
		Created: 1677652288,
		Model:   "gpt-4o",
		Choices: []OpenAIChoice{},
		Usage: OpenAIUsage{
			PromptTokens:     10,
			CompletionTokens: 0,
			TotalTokens:      10,
		},
	}

	_, err := ConvertOpenAIToClaude(resp)
	if err == nil {
		t.Error("Should return error for empty choices")
	}
}

// 测试空内容
func TestConvertEmptyContent(t *testing.T) {
	resp := &OpenAIResponse{
		ID:      "chatcmpl-empty-content",
		Object:  "chat.completion",
		Created: 1677652288,
		Model:   "gpt-4o",
		Choices: []OpenAIChoice{
			{
				Index: 0,
				Message: OpenAIMessage{
					Role:    "assistant",
					Content: nil,
				},
				FinishReason: "stop",
			},
		},
		Usage: OpenAIUsage{
			PromptTokens:     10,
			CompletionTokens: 0,
			TotalTokens:      10,
		},
	}

	result, err := ConvertOpenAIToClaude(resp)
	if err != nil {
		t.Fatalf("转换失败: %v", err)
	}

	// 应该有一个空文本块
	if len(result.Content) != 1 {
		t.Fatalf("Should have 1 content block, got %d", len(result.Content))
	}

	if result.Content[0].Type != "text" {
		t.Error("Should have text type")
	}

	if result.Content[0].Text == nil || *result.Content[0].Text != "" {
		t.Error("Should have empty text")
	}
}

// 性能基准测试
func BenchmarkConvertOpenAIToClaude(b *testing.B) {
	resp := &OpenAIResponse{
		ID:      "chatcmpl-bench",
		Object:  "chat.completion",
		Created: 1677652288,
		Model:   "gpt-4o",
		Choices: []OpenAIChoice{
			{
				Index: 0,
				Message: OpenAIMessage{
					Role:    "assistant",
					Content: "Hello! How can I help you?",
				},
				FinishReason: "stop",
			},
		},
		Usage: OpenAIUsage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ConvertOpenAIToClaude(resp)
	}
}
