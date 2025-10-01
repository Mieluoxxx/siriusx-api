package converter

import (
	"encoding/json"
	"testing"
)

// 测试基础字段转换
func TestConvertBasicFields(t *testing.T) {
	temp := 0.7
	topP := 0.9
	req := &ClaudeRequest{
		Model:         "claude-3-5-sonnet",
		MaxTokens:     1024,
		Temperature:   &temp,
		TopP:          &topP,
		Stream:        true,
		StopSequences: []string{"stop1", "stop2"},
		Messages: []ClaudeMessage{
			{
				Role:    "user",
				Content: []ClaudeContentBlock{{Type: "text", Text: StringPtr("Hello")}},
			},
		},
	}

	result, err := ConvertClaudeToOpenAI(req)
	if err != nil {
		t.Fatalf("转换失败: %v", err)
	}

	if result.Model != "claude-3-5-sonnet" {
		t.Errorf("Model 不匹配: got %s, want claude-3-5-sonnet", result.Model)
	}

	if result.MaxTokens != 1024 {
		t.Errorf("MaxTokens 不匹配: got %d, want 1024", result.MaxTokens)
	}

	if result.Temperature == nil || *result.Temperature != 0.7 {
		t.Error("Temperature 不匹配")
	}

	if result.TopP == nil || *result.TopP != 0.9 {
		t.Error("TopP 不匹配")
	}

	if !result.Stream {
		t.Error("Stream 应该为 true")
	}

	if len(result.Stop) != 2 || result.Stop[0] != "stop1" {
		t.Error("Stop sequences 不匹配")
	}
}

// 测试 System 消息转换
func TestConvertSystemMessage(t *testing.T) {
	req := &ClaudeRequest{
		Model:   "claude-3-5-sonnet",
		System:  "You are a helpful assistant",
		MaxTokens: 1024,
		Messages: []ClaudeMessage{
			{
				Role:    "user",
				Content: []ClaudeContentBlock{{Type: "text", Text: StringPtr("Hello")}},
			},
		},
	}

	result, err := ConvertClaudeToOpenAI(req)
	if err != nil {
		t.Fatalf("转换失败: %v", err)
	}

	if len(result.Messages) < 2 {
		t.Fatal("消息数量不足")
	}

	if result.Messages[0].Role != "system" {
		t.Errorf("第一条消息角色应该是 system, got %s", result.Messages[0].Role)
	}

	if result.Messages[0].Content != "You are a helpful assistant" {
		t.Error("System 消息内容不匹配")
	}

	if result.Messages[1].Role != "user" {
		t.Error("第二条消息应该是用户消息")
	}
}

// 测试用户文本消息转换
func TestConvertUserTextMessage(t *testing.T) {
	req := &ClaudeRequest{
		Model:   "claude-3-5-sonnet",
		MaxTokens: 1024,
		Messages: []ClaudeMessage{
			{
				Role: "user",
				Content: []ClaudeContentBlock{
					{Type: "text", Text: StringPtr("Hello, how are you?")},
				},
			},
		},
	}

	result, err := ConvertClaudeToOpenAI(req)
	if err != nil {
		t.Fatalf("转换失败: %v", err)
	}

	if len(result.Messages) != 1 {
		t.Fatalf("消息数量错误: got %d, want 1", len(result.Messages))
	}

	msg := result.Messages[0]
	if msg.Role != "user" {
		t.Errorf("角色错误: got %s, want user", msg.Role)
	}

	// 单个文本块应该简化为字符串
	if str, ok := msg.Content.(string); !ok {
		t.Error("Content 应该是字符串类型")
	} else if str != "Hello, how are you?" {
		t.Errorf("Content 不匹配: got %s", str)
	}
}

// 测试用户图片消息转换
func TestConvertUserImageMessage(t *testing.T) {
	req := &ClaudeRequest{
		Model:   "claude-3-5-sonnet",
		MaxTokens: 1024,
		Messages: []ClaudeMessage{
			{
				Role: "user",
				Content: []ClaudeContentBlock{
					{Type: "text", Text: StringPtr("What's in this image?")},
					{
						Type: "image",
						Source: &ClaudeImageSource{
							Type:      "base64",
							MediaType: "image/jpeg",
							Data:      "abc123",
						},
					},
				},
			},
		},
	}

	result, err := ConvertClaudeToOpenAI(req)
	if err != nil {
		t.Fatalf("转换失败: %v", err)
	}

	msg := result.Messages[0]

	// 多个内容块应该是数组
	blocks, ok := msg.Content.([]OpenAIContentBlock)
	if !ok {
		t.Fatal("Content 应该是 []OpenAIContentBlock 类型")
	}

	if len(blocks) != 2 {
		t.Fatalf("Content blocks 数量错误: got %d, want 2", len(blocks))
	}

	// 检查文本块
	if blocks[0].Type != "text" || blocks[0].Text == nil {
		t.Error("第一个块应该是文本")
	}

	// 检查图片块
	if blocks[1].Type != "image_url" {
		t.Error("第二个块应该是 image_url")
	}

	if blocks[1].ImageURL == nil {
		t.Fatal("ImageURL 不能为 nil")
	}

	expectedURL := "data:image/jpeg;base64,abc123"
	if blocks[1].ImageURL.URL != expectedURL {
		t.Errorf("图片 URL 不匹配: got %s, want %s", blocks[1].ImageURL.URL, expectedURL)
	}
}

// 测试助手文本消息转换
func TestConvertAssistantTextMessage(t *testing.T) {
	req := &ClaudeRequest{
		Model:   "claude-3-5-sonnet",
		MaxTokens: 1024,
		Messages: []ClaudeMessage{
			{
				Role: "assistant",
				Content: []ClaudeContentBlock{
					{Type: "text", Text: StringPtr("I'm doing well, thank you!")},
				},
			},
		},
	}

	result, err := ConvertClaudeToOpenAI(req)
	if err != nil {
		t.Fatalf("转换失败: %v", err)
	}

	msg := result.Messages[0]
	if msg.Role != "assistant" {
		t.Errorf("角色错误: got %s, want assistant", msg.Role)
	}

	if str, ok := msg.Content.(string); !ok {
		t.Error("Content 应该是字符串类型")
	} else if str != "I'm doing well, thank you!" {
		t.Errorf("Content 不匹配: got %s", str)
	}
}

// 测试助手工具调用转换
func TestConvertAssistantToolUseMessage(t *testing.T) {
	input := map[string]interface{}{
		"location": "San Francisco",
		"unit":     "celsius",
	}

	req := &ClaudeRequest{
		Model:   "claude-3-5-sonnet",
		MaxTokens: 1024,
		Messages: []ClaudeMessage{
			{
				Role: "assistant",
				Content: []ClaudeContentBlock{
					{Type: "text", Text: StringPtr("Let me check the weather")},
					{
						Type:  "tool_use",
						ID:    StringPtr("toolu_123"),
						Name:  StringPtr("get_weather"),
						Input: input,
					},
				},
			},
		},
	}

	result, err := ConvertClaudeToOpenAI(req)
	if err != nil {
		t.Fatalf("转换失败: %v", err)
	}

	msg := result.Messages[0]
	if msg.Role != "assistant" {
		t.Error("角色应该是 assistant")
	}

	// 检查文本内容
	if str, ok := msg.Content.(string); !ok {
		t.Error("Content 应该是字符串")
	} else if str != "Let me check the weather" {
		t.Errorf("Content 不匹配: got %s", str)
	}

	// 检查 tool_calls
	if len(msg.ToolCalls) != 1 {
		t.Fatalf("ToolCalls 数量错误: got %d, want 1", len(msg.ToolCalls))
	}

	toolCall := msg.ToolCalls[0]
	if toolCall.ID != "toolu_123" {
		t.Errorf("ToolCall ID 不匹配: got %s", toolCall.ID)
	}

	if toolCall.Type != "function" {
		t.Error("ToolCall Type 应该是 function")
	}

	if toolCall.Function.Name != "get_weather" {
		t.Errorf("Function Name 不匹配: got %s", toolCall.Function.Name)
	}

	// 检查 arguments 是否为有效的 JSON
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		t.Fatalf("Arguments 不是有效的 JSON: %v", err)
	}

	if args["location"] != "San Francisco" {
		t.Error("Arguments location 不匹配")
	}
}

// 测试 tool_result 消息转换
func TestConvertToolResultMessage(t *testing.T) {
	req := &ClaudeRequest{
		Model:   "claude-3-5-sonnet",
		MaxTokens: 1024,
		Messages: []ClaudeMessage{
			{
				Role: "user",
				Content: []ClaudeContentBlock{
					{
						Type:      "tool_result",
						ToolUseID: StringPtr("toolu_123"),
						Content:   StringPtr(`{"temperature": 72, "unit": "fahrenheit"}`),
					},
				},
			},
		},
	}

	result, err := ConvertClaudeToOpenAI(req)
	if err != nil {
		t.Fatalf("转换失败: %v", err)
	}

	if len(result.Messages) != 1 {
		t.Fatalf("消息数量错误: got %d, want 1", len(result.Messages))
	}

	msg := result.Messages[0]
	if msg.Role != "tool" {
		t.Errorf("角色应该是 tool, got %s", msg.Role)
	}

	if msg.ToolCallID != "toolu_123" {
		t.Errorf("ToolCallID 不匹配: got %s", msg.ToolCallID)
	}

	if str, ok := msg.Content.(string); !ok {
		t.Error("Content 应该是字符串")
	} else if str != `{"temperature": 72, "unit": "fahrenheit"}` {
		t.Errorf("Content 不匹配: got %s", str)
	}
}

// 测试工具定义转换
func TestConvertTools(t *testing.T) {
	inputSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"location": map[string]interface{}{
				"type":        "string",
				"description": "The city name",
			},
		},
		"required": []string{"location"},
	}

	req := &ClaudeRequest{
		Model:   "claude-3-5-sonnet",
		MaxTokens: 1024,
		Messages: []ClaudeMessage{
			{Role: "user", Content: []ClaudeContentBlock{{Type: "text", Text: StringPtr("What's the weather?")}}},
		},
		Tools: []ClaudeTool{
			{
				Name:        "get_weather",
				Description: "Get the current weather",
				InputSchema: inputSchema,
			},
		},
	}

	result, err := ConvertClaudeToOpenAI(req)
	if err != nil {
		t.Fatalf("转换失败: %v", err)
	}

	if len(result.Tools) != 1 {
		t.Fatalf("Tools 数量错误: got %d, want 1", len(result.Tools))
	}

	tool := result.Tools[0]
	if tool.Type != "function" {
		t.Error("Tool Type 应该是 function")
	}

	if tool.Function.Name != "get_weather" {
		t.Errorf("Function Name 不匹配: got %s", tool.Function.Name)
	}

	if tool.Function.Description != "Get the current weather" {
		t.Error("Description 不匹配")
	}

	if tool.Function.Parameters == nil {
		t.Fatal("Parameters 不能为 nil")
	}
}

// 测试 tool_choice 转换
func TestConvertToolChoice(t *testing.T) {
	tests := []struct {
		name     string
		choice   *ClaudeToolChoice
		expected interface{}
	}{
		{
			name:     "auto",
			choice:   &ClaudeToolChoice{Type: "auto"},
			expected: "auto",
		},
		{
			name:     "any",
			choice:   &ClaudeToolChoice{Type: "any"},
			expected: "required",
		},
		{
			name:   "specific tool",
			choice: &ClaudeToolChoice{Type: "tool", Name: StringPtr("get_weather")},
			expected: map[string]interface{}{
				"type": "function",
				"function": map[string]string{
					"name": "get_weather",
				},
			},
		},
		{
			name:     "nil",
			choice:   nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToolChoice(tt.choice)

			if tt.expected == nil {
				if result != nil {
					t.Error("应该返回 nil")
				}
				return
			}

			// 对于字符串类型
			if expectedStr, ok := tt.expected.(string); ok {
				if resultStr, ok := result.(string); !ok {
					t.Error("结果应该是字符串")
				} else if resultStr != expectedStr {
					t.Errorf("结果不匹配: got %s, want %s", resultStr, expectedStr)
				}
				return
			}

			// 对于 map 类型，使用 JSON 比较
			expectedJSON, _ := json.Marshal(tt.expected)
			resultJSON, _ := json.Marshal(result)
			if string(expectedJSON) != string(resultJSON) {
				t.Errorf("结果不匹配: got %s, want %s", resultJSON, expectedJSON)
			}
		})
	}
}

// 测试空请求
func TestConvertNilRequest(t *testing.T) {
	_, err := ConvertClaudeToOpenAI(nil)
	if err == nil {
		t.Error("空请求应该返回错误")
	}
}

// 测试空消息数组
func TestConvertEmptyMessages(t *testing.T) {
	req := &ClaudeRequest{
		Model:     "claude-3-5-sonnet",
		MaxTokens: 1024,
		Messages:  []ClaudeMessage{},
	}

	result, err := ConvertClaudeToOpenAI(req)
	if err != nil {
		t.Fatalf("转换失败: %v", err)
	}

	if len(result.Messages) != 0 {
		t.Error("消息数组应该为空")
	}
}

// 性能基准测试
func BenchmarkConvertClaudeToOpenAI(b *testing.B) {
	req := &ClaudeRequest{
		Model:     "claude-3-5-sonnet",
		MaxTokens: 1024,
		Messages: []ClaudeMessage{
			{
				Role: "user",
				Content: []ClaudeContentBlock{
					{Type: "text", Text: StringPtr("Hello, how are you?")},
				},
			},
			{
				Role: "assistant",
				Content: []ClaudeContentBlock{
					{Type: "text", Text: StringPtr("I'm doing well, thank you!")},
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ConvertClaudeToOpenAI(req)
	}
}
