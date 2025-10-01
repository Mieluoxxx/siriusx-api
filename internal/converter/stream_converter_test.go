package converter

import (
	"bufio"
	"context"
	"io"
	"strings"
	"testing"
)

// TestSSEParser_BasicEvent 测试基础 SSE 事件解析
func TestSSEParser_BasicEvent(t *testing.T) {
	input := "data: {\"test\": \"value\"}\n\n"
	parser := NewSSEParser(strings.NewReader(input))

	data, err := parser.ParseEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data != `{"test": "value"}` {
		t.Errorf("expected %q, got %q", `{"test": "value"}`, data)
	}

	// 下一个应该是 EOF
	_, err = parser.ParseEvent()
	if err != io.EOF {
		t.Errorf("expected EOF, got %v", err)
	}
}

// TestSSEParser_MultipleEvents 测试多个 SSE 事件解析
func TestSSEParser_MultipleEvents(t *testing.T) {
	input := `data: {"event": 1}

data: {"event": 2}

data: {"event": 3}

`
	parser := NewSSEParser(strings.NewReader(input))

	// 第一个事件
	data, err := parser.ParseEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(data, `"event": 1`) {
		t.Errorf("expected event 1, got %q", data)
	}

	// 第二个事件
	data, err = parser.ParseEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(data, `"event": 2`) {
		t.Errorf("expected event 2, got %q", data)
	}

	// 第三个事件
	data, err = parser.ParseEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(data, `"event": 3`) {
		t.Errorf("expected event 3, got %q", data)
	}

	// EOF
	_, err = parser.ParseEvent()
	if err != io.EOF {
		t.Errorf("expected EOF, got %v", err)
	}
}

// TestSSEParser_DoneEvent 测试 [DONE] 事件
func TestSSEParser_DoneEvent(t *testing.T) {
	input := "data: [DONE]\n\n"
	parser := NewSSEParser(strings.NewReader(input))

	data, err := parser.ParseEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data != "[DONE]" {
		t.Errorf("expected [DONE], got %q", data)
	}
}

// TestFormatSSEEvent 测试 SSE 事件格式化
func TestFormatSSEEvent(t *testing.T) {
	data := ClaudeMessageStop{
		Type: "message_stop",
	}

	result, err := FormatSSEEvent("message_stop", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "event: message_stop") {
		t.Errorf("expected event line, got %q", result)
	}
	if !strings.Contains(result, `"type":"message_stop"`) {
		t.Errorf("expected type field, got %q", result)
	}
	if !strings.HasSuffix(result, "\n\n") {
		t.Errorf("expected double newline suffix, got %q", result)
	}
}

// TestConvertStream_BasicText 测试基础文本流式转换
func TestConvertStream_BasicText(t *testing.T) {
	openaiStream := `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1694268190,"model":"gpt-4","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1694268190,"model":"gpt-4","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1694268190,"model":"gpt-4","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1694268190,"model":"gpt-4","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}

data: [DONE]

`

	claudeStream, err := ConvertStream(context.Background(), strings.NewReader(openaiStream))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 读取所有事件
	events := readAllEvents(t, claudeStream)

	// 验证事件顺序
	if len(events) < 7 {
		t.Logf("Events received: %d", len(events))
		for i, event := range events {
			t.Logf("Event %d: %q", i, event)
		}
		t.Fatalf("expected at least 7 events, got %d", len(events))
	}

	// 验证 message_start
	if !strings.Contains(events[0], "event: message_start") {
		t.Errorf("expected message_start event, got %q", events[0])
	}
	if !strings.Contains(events[0], `"type":"message_start"`) {
		t.Errorf("expected message_start type, got %q", events[0])
	}

	// 验证 content_block_start
	if !strings.Contains(events[1], "event: content_block_start") {
		t.Errorf("expected content_block_start event, got %q", events[1])
	}
	if !strings.Contains(events[1], `"index":0`) {
		t.Errorf("expected index 0, got %q", events[1])
	}

	// 验证 content_block_delta (Hello)
	if !strings.Contains(events[2], "event: content_block_delta") {
		t.Errorf("expected content_block_delta event, got %q", events[2])
	}
	if !strings.Contains(events[2], `"text":"Hello"`) {
		t.Errorf("expected Hello text, got %q", events[2])
	}

	// 验证 content_block_delta ( world)
	if !strings.Contains(events[3], "event: content_block_delta") {
		t.Errorf("expected content_block_delta event, got %q", events[3])
	}
	if !strings.Contains(events[3], `"text":" world"`) {
		t.Errorf("expected world text, got %q", events[3])
	}

	// 验证 content_block_stop
	if !strings.Contains(events[4], "event: content_block_stop") {
		t.Errorf("expected content_block_stop event, got %q", events[4])
	}

	// 验证 message_delta (with stop_reason)
	if !strings.Contains(events[5], "event: message_delta") {
		t.Errorf("expected message_delta event, got %q", events[5])
	}
	if !strings.Contains(events[5], `"stop_reason":"end_turn"`) {
		t.Errorf("expected end_turn stop_reason, got %q", events[5])
	}

	// 验证最后有 message_delta 和 message_stop (可能顺序不同)
	hasMessageDelta := false
	hasMessageStop := false
	for i := 5; i < len(events); i++ {
		if strings.Contains(events[i], "event: message_delta") {
			hasMessageDelta = true
		}
		if strings.Contains(events[i], "event: message_stop") {
			hasMessageStop = true
		}
	}

	if !hasMessageDelta {
		t.Error("expected message_delta event")
	}
	if !hasMessageStop {
		t.Error("expected message_stop event")
	}
}

// TestConvertStream_ToolCalls 测试 tool calls 流式转换
func TestConvertStream_ToolCalls(t *testing.T) {
	openaiStream := `data: {"id":"chatcmpl-456","model":"gpt-4","choices":[{"delta":{"role":"assistant"},"finish_reason":null}]}

data: {"id":"chatcmpl-456","choices":[{"delta":{"content":"Let me check"},"finish_reason":null}]}

data: {"id":"chatcmpl-456","choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_abc","type":"function","function":{"name":"get_weather","arguments":""}}]},"finish_reason":null}]}

data: {"id":"chatcmpl-456","choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"location\""}}]},"finish_reason":null}]}

data: {"id":"chatcmpl-456","choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\":\"SF\"}"}}]},"finish_reason":null}]}

data: {"id":"chatcmpl-456","choices":[{"delta":{},"finish_reason":"tool_calls"}]}

data: [DONE]

`

	claudeStream, err := ConvertStream(context.Background(), strings.NewReader(openaiStream))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 读取所有事件
	events := readAllEvents(t, claudeStream)

	// 验证有文本块和 tool_use 块
	hasTextBlock := false
	hasToolUseBlock := false
	hasInputJSONDelta := false

	for _, event := range events {
		if strings.Contains(event, `"type":"text"`) {
			hasTextBlock = true
		}
		if strings.Contains(event, `"type":"tool_use"`) {
			hasToolUseBlock = true
		}
		if strings.Contains(event, `"type":"input_json_delta"`) {
			hasInputJSONDelta = true
		}
	}

	if !hasTextBlock {
		t.Error("expected text block")
	}
	if !hasToolUseBlock {
		t.Error("expected tool_use block")
	}
	if !hasInputJSONDelta {
		t.Error("expected input_json_delta event")
	}

	// 验证 finish_reason 转换
	found := false
	for _, event := range events {
		if strings.Contains(event, `"stop_reason":"tool_use"`) {
			found = true
			break
		}
	}
	if !found {
		t.Error("finish_reason should be converted to tool_use")
	}
}

// TestConvertStream_MultipleToolCalls 测试多个 tool calls
func TestConvertStream_MultipleToolCalls(t *testing.T) {
	openaiStream := `data: {"id":"chatcmpl-789","choices":[{"delta":{"role":"assistant"},"finish_reason":null}]}

data: {"id":"chatcmpl-789","choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"tool1","arguments":""}}]},"finish_reason":null}]}

data: {"id":"chatcmpl-789","choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{}"}}]},"finish_reason":null}]}

data: {"id":"chatcmpl-789","choices":[{"delta":{"tool_calls":[{"index":1,"id":"call_2","type":"function","function":{"name":"tool2","arguments":""}}]},"finish_reason":null}]}

data: {"id":"chatcmpl-789","choices":[{"delta":{"tool_calls":[{"index":1,"function":{"arguments":"{}"}}]},"finish_reason":null}]}

data: {"id":"chatcmpl-789","choices":[{"delta":{},"finish_reason":"tool_calls"}]}

data: [DONE]

`

	claudeStream, err := ConvertStream(context.Background(), strings.NewReader(openaiStream))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 读取所有事件
	events := readAllEvents(t, claudeStream)

	// 应该有两个 content_block_start (两个 tool calls)
	startCount := 0
	for _, event := range events {
		if strings.Contains(event, "event: content_block_start") {
			startCount++
		}
	}
	if startCount != 2 {
		t.Errorf("expected 2 content_block_start events, got %d", startCount)
	}
}

// TestConvertStream_EmptyStream 测试空流
func TestConvertStream_EmptyStream(t *testing.T) {
	openaiStream := ``

	claudeStream, err := ConvertStream(context.Background(), strings.NewReader(openaiStream))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 读取所有事件
	events := readAllEvents(t, claudeStream)

	// 空流应该至少有 message_delta 和 message_stop
	if len(events) < 2 {
		t.Errorf("expected at least 2 events, got %d", len(events))
	}
}

// TestConvertStream_ContextCancellation 测试上下文取消
func TestConvertStream_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// 创建长流
	openaiStream := strings.Repeat(`data: {"choices":[{"delta":{"content":"a"},"finish_reason":null}]}`+"\n\n", 100)

	claudeStream, err := ConvertStream(ctx, strings.NewReader(openaiStream))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 立即取消
	cancel()

	// 尝试读取
	buf := make([]byte, 1024)
	_, err = claudeStream.Read(buf)

	// 应该返回错误
	if err == nil {
		t.Error("expected error on cancelled context")
	}
}

// TestEventOrder 测试事件顺序
func TestEventOrder(t *testing.T) {
	openaiStream := `data: {"id":"test","choices":[{"delta":{"role":"assistant"},"finish_reason":null}]}

data: {"id":"test","choices":[{"delta":{"content":"Hi"},"finish_reason":null}]}

data: {"id":"test","choices":[{"delta":{},"finish_reason":"stop"}]}

data: [DONE]

`

	claudeStream, err := ConvertStream(context.Background(), strings.NewReader(openaiStream))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events := readAllEvents(t, claudeStream)

	// 提取事件类型
	eventTypes := extractEventTypes(events)

	// 期望的顺序 (可能有两个 message_delta)
	expectedOrder := []string{
		"message_start",
		"content_block_start",
		"content_block_delta",
		"content_block_stop",
		"message_delta",
	}

	// 验证前面的事件顺序
	for i, expected := range expectedOrder {
		if i >= len(eventTypes) {
			t.Errorf("expected at least %d events, got %d", len(expectedOrder), len(eventTypes))
			break
		}
		if eventTypes[i] != expected {
			t.Errorf("event %d: expected %q, got %q", i, expected, eventTypes[i])
		}
	}

	// 验证最后一个事件是 message_stop
	if len(eventTypes) > 0 {
		lastEvent := eventTypes[len(eventTypes)-1]
		if lastEvent != "message_stop" {
			t.Errorf("last event should be message_stop, got %q", lastEvent)
		}
	}
}

// Helper: 读取所有事件
func readAllEvents(t *testing.T, stream io.Reader) []string {
	var events []string
	scanner := bufio.NewScanner(stream)

	var currentEvent strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			// 事件结束
			if currentEvent.Len() > 0 {
				events = append(events, currentEvent.String())
				currentEvent.Reset()
			}
		} else {
			currentEvent.WriteString(line + "\n")
		}
	}

	// 最后一个事件
	if currentEvent.Len() > 0 {
		events = append(events, currentEvent.String())
	}

	return events
}

// Helper: 提取事件类型
func extractEventTypes(events []string) []string {
	var types []string
	for _, event := range events {
		lines := strings.Split(event, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "event: ") {
				eventType := strings.TrimPrefix(line, "event: ")
				types = append(types, eventType)
				break
			}
		}
	}
	return types
}
