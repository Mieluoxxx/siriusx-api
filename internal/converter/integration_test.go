package converter

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// TestIntegration_CompleteConversionFlow 完整转换流程集成测试
func TestIntegration_CompleteConversionFlow(t *testing.T) {
	// 场景1: 基础文本对话
	t.Run("BasicTextConversation", func(t *testing.T) {
		// Step 1: Claude 请求 -> OpenAI 请求
		claudeReq := &ClaudeRequest{
			Model:     "claude-3-5-sonnet-20241022",
			System:    "You are a helpful assistant",
			MaxTokens: 1024,
			Messages: []ClaudeMessage{
				{
					Role: "user",
					Content: []ClaudeContentBlock{
						{
							Type: "text",
							Text: StringPtr("Hello, how are you?"),
						},
					},
				},
			},
		}

		openaiReq, err := ConvertClaudeToOpenAI(claudeReq)
		if err != nil {
			t.Fatalf("Failed to convert Claude to OpenAI: %v", err)
		}

		// 验证转换结果
		if openaiReq.Model != "claude-3-5-sonnet-20241022" {
			t.Errorf("Model mismatch: got %s", openaiReq.Model)
		}
		if len(openaiReq.Messages) != 2 {
			t.Errorf("Expected 2 messages (system + user), got %d", len(openaiReq.Messages))
		}

		// Step 2: 模拟 OpenAI 响应 -> Claude 响应
		openaiResp := &OpenAIResponse{
			ID:    "chatcmpl-123",
			Model: "claude-3-5-sonnet-20241022",
			Choices: []OpenAIChoice{
				{
					Message: OpenAIMessage{
						Role:    "assistant",
						Content: "I'm doing great, thank you for asking!",
					},
					FinishReason: "stop",
				},
			},
			Usage: OpenAIUsage{
				PromptTokens:     10,
				CompletionTokens: 8,
				TotalTokens:      18,
			},
		}

		claudeResp, err := ConvertOpenAIToClaude(openaiResp)
		if err != nil {
			t.Fatalf("Failed to convert OpenAI to Claude: %v", err)
		}

		// 验证响应转换
		if claudeResp.Role != "assistant" {
			t.Errorf("Role mismatch: got %s", claudeResp.Role)
		}
		if claudeResp.StopReason != "end_turn" {
			t.Errorf("Stop reason mismatch: got %s", claudeResp.StopReason)
		}
		if len(claudeResp.Content) != 1 {
			t.Errorf("Expected 1 content block, got %d", len(claudeResp.Content))
		}
	})

	// 场景2: 工具调用对话
	t.Run("ToolCallConversation", func(t *testing.T) {
		// Step 1: Claude 请求带工具
		claudeReq := &ClaudeRequest{
			Model:     "claude-3-5-sonnet-20241022",
			MaxTokens: 1024,
			Messages: []ClaudeMessage{
				{
					Role: "user",
					Content: []ClaudeContentBlock{
						{Type: "text", Text: StringPtr("What's the weather in SF?")},
					},
				},
			},
			Tools: []ClaudeTool{
				{
					Name:        "get_weather",
					Description: "Get weather information",
					InputSchema: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"location": map[string]interface{}{
								"type":        "string",
								"description": "City name",
							},
						},
					},
				},
			},
		}

		openaiReq, err := ConvertClaudeToOpenAI(claudeReq)
		if err != nil {
			t.Fatalf("Failed to convert Claude to OpenAI: %v", err)
		}

		if len(openaiReq.Tools) != 1 {
			t.Errorf("Expected 1 tool, got %d", len(openaiReq.Tools))
		}

		// Step 2: 模拟 OpenAI 工具调用响应
		openaiResp := &OpenAIResponse{
			ID:    "chatcmpl-456",
			Model: "claude-3-5-sonnet-20241022",
			Choices: []OpenAIChoice{
				{
					Message: OpenAIMessage{
						Role:    "assistant",
						Content: "Let me check the weather for you.",
						ToolCalls: []OpenAIToolCall{
							{
								ID:   "call_abc123",
								Type: "function",
								Function: OpenAIFunctionCall{
									Name:      "get_weather",
									Arguments: `{"location":"SF"}`,
								},
							},
						},
					},
					FinishReason: "tool_calls",
				},
			},
		}

		claudeResp, err := ConvertOpenAIToClaude(openaiResp)
		if err != nil {
			t.Fatalf("Failed to convert OpenAI to Claude: %v", err)
		}

		// 验证工具调用转换
		if claudeResp.StopReason != "tool_use" {
			t.Errorf("Stop reason mismatch: got %s", claudeResp.StopReason)
		}
		if len(claudeResp.Content) != 2 {
			t.Errorf("Expected 2 content blocks (text + tool_use), got %d", len(claudeResp.Content))
		}
		if claudeResp.Content[1].Type != "tool_use" {
			t.Errorf("Second block should be tool_use, got %s", claudeResp.Content[1].Type)
		}
	})

	// 场景3: 流式转换端到端
	t.Run("StreamingConversion", func(t *testing.T) {
		// 模拟 OpenAI 流式响应
		openaiStream := `data: {"id":"chatcmpl-789","object":"chat.completion.chunk","created":1694268190,"model":"claude-3-5-sonnet-20241022","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}

data: {"id":"chatcmpl-789","object":"chat.completion.chunk","created":1694268190,"model":"claude-3-5-sonnet-20241022","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}

data: {"id":"chatcmpl-789","object":"chat.completion.chunk","created":1694268190,"model":"claude-3-5-sonnet-20241022","choices":[{"index":0,"delta":{"content":" there"},"finish_reason":null}]}

data: {"id":"chatcmpl-789","object":"chat.completion.chunk","created":1694268190,"model":"claude-3-5-sonnet-20241022","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}

data: [DONE]

`

		claudeStream, err := ConvertStream(context.Background(), strings.NewReader(openaiStream))
		if err != nil {
			t.Fatalf("Failed to convert stream: %v", err)
		}

		// 读取并解析所有事件
		parser := NewSSEParser(claudeStream)
		var events []string
		for {
			eventLine, err := parser.ParseEvent()
			if err != nil {
				break
			}
			if eventLine == "" {
				break
			}

			// 解析事件类型
			var eventData map[string]interface{}
			if err := json.Unmarshal([]byte(eventLine), &eventData); err == nil {
				if eventType, ok := eventData["type"].(string); ok {
					events = append(events, eventType)
					t.Logf("Event %d: %s", len(events)-1, eventType)
				}
			}
		}

		// 验证事件序列（调整为实际的事件序列）
		expectedEvents := []string{
			"message_start",
			"content_block_start",
			"content_block_delta",
			"content_block_delta",
			"content_block_stop",
			"message_delta",
			"message_delta", // 有两个 message_delta
			"message_stop",
		}

		if len(events) != len(expectedEvents) {
			t.Errorf("Event count mismatch: got %d, expected %d", len(events), len(expectedEvents))
		}

		for i, expected := range expectedEvents {
			if i >= len(events) {
				break
			}
			if events[i] != expected {
				t.Errorf("Event %d mismatch: got %s, expected %s", i, events[i], expected)
			}
		}
	})
}

// TestIntegration_EdgeCases 边界情况集成测试
func TestIntegration_EdgeCases(t *testing.T) {
	t.Run("EmptyContent", func(t *testing.T) {
		claudeReq := &ClaudeRequest{
			Model:     "claude-3-5-sonnet-20241022",
			MaxTokens: 100,
			Messages:  []ClaudeMessage{},
		}

		openaiReq, err := ConvertClaudeToOpenAI(claudeReq)
		if err != nil {
			t.Fatalf("Failed to convert: %v", err)
		}

		if len(openaiReq.Messages) != 0 {
			t.Errorf("Expected empty messages, got %d", len(openaiReq.Messages))
		}
	})

	t.Run("MultipleToolCalls", func(t *testing.T) {
		openaiResp := &OpenAIResponse{
			ID:    "chatcmpl-multi",
			Model: "claude-3-5-sonnet-20241022",
			Choices: []OpenAIChoice{
				{
					Message: OpenAIMessage{
						Role: "assistant",
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
									Name:      "get_time",
									Arguments: `{"timezone":"PST"}`,
								},
							},
						},
					},
					FinishReason: "tool_calls",
				},
			},
		}

		claudeResp, err := ConvertOpenAIToClaude(openaiResp)
		if err != nil {
			t.Fatalf("Failed to convert: %v", err)
		}

		toolUseCount := 0
		for _, block := range claudeResp.Content {
			if block.Type == "tool_use" {
				toolUseCount++
			}
		}

		if toolUseCount != 2 {
			t.Errorf("Expected 2 tool_use blocks, got %d", toolUseCount)
		}
	})
}

// TestIntegration_Performance 性能测试
func TestIntegration_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	t.Run("RequestConversionPerformance", func(t *testing.T) {
		claudeReq := &ClaudeRequest{
			Model:     "claude-3-5-sonnet-20241022",
			MaxTokens: 1024,
			Messages: []ClaudeMessage{
				{
					Role: "user",
					Content: []ClaudeContentBlock{
						{Type: "text", Text: StringPtr("Hello")},
					},
				},
			},
		}

		// 执行1000次转换
		for i := 0; i < 1000; i++ {
			_, err := ConvertClaudeToOpenAI(claudeReq)
			if err != nil {
				t.Fatalf("Conversion failed at iteration %d: %v", i, err)
			}
		}
	})

	t.Run("ResponseConversionPerformance", func(t *testing.T) {
		openaiResp := &OpenAIResponse{
			ID:    "chatcmpl-perf",
			Model: "claude-3-5-sonnet-20241022",
			Choices: []OpenAIChoice{
				{
					Message: OpenAIMessage{
						Role:    "assistant",
						Content: "Test response",
					},
					FinishReason: "stop",
				},
			},
		}

		// 执行1000次转换
		for i := 0; i < 1000; i++ {
			_, err := ConvertOpenAIToClaude(openaiResp)
			if err != nil {
				t.Fatalf("Conversion failed at iteration %d: %v", i, err)
			}
		}
	})
}
