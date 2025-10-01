package converter

import (
	"context"
	"encoding/json"
	"io"
	"strings"
)

// StreamConverter 流式响应转换器
type StreamConverter struct {
	// 消息元数据
	messageID    string
	model        string
	role         string
	created      int64

	// 状态管理
	currentIndex    int                    // 当前 content block 索引
	messageStarted  bool                   // 是否已发送 message_start
	blockStarted    bool                   // 当前块是否已开始
	currentBlockType string                // 当前块类型 ("text" | "tool_use")

	// 累积状态
	textBuffer      strings.Builder        // 文本缓冲区
	toolCallsBuffer map[int]*ToolCallState // tool calls 缓冲区

	// 统计
	inputTokens  int
	outputTokens int
}

// ToolCallState tool call 累积状态
type ToolCallState struct {
	ID        string
	Name      string
	Arguments strings.Builder
}

// NewStreamConverter 创建流式转换器
func NewStreamConverter() *StreamConverter {
	return &StreamConverter{
		toolCallsBuffer: make(map[int]*ToolCallState),
		currentIndex:    -1, // 初始值为 -1，第一个块时会设为 0
	}
}

// ConvertStream 转换 OpenAI 流式响应为 Claude 流式响应
func ConvertStream(ctx context.Context, openaiStream io.Reader) (io.Reader, error) {
	// 创建管道用于零拷贝传输
	pipeReader, pipeWriter := io.Pipe()

	// 在 goroutine 中处理流式转换
	go func() {
		defer pipeWriter.Close()

		converter := NewStreamConverter()
		parser := NewSSEParser(openaiStream)

		for {
			// 检查上下文取消
			select {
			case <-ctx.Done():
				pipeWriter.CloseWithError(ctx.Err())
				return
			default:
			}

			// 解析下一个事件
			eventData, err := parser.ParseEvent()
			if err == io.EOF {
				// 流结束，发送 message_stop
				if converter.blockStarted {
					// 关闭当前块
					if event, err := converter.emitContentBlockStop(); err == nil {
						pipeWriter.Write([]byte(event))
					}
				}
				if event, err := converter.emitMessageDelta(""); err == nil {
					pipeWriter.Write([]byte(event))
				}
				if event, err := converter.emitMessageStop(); err == nil {
					pipeWriter.Write([]byte(event))
				}
				return
			}
			if err != nil {
				pipeWriter.CloseWithError(err)
				return
			}

			// 处理 [DONE] 事件
			if eventData == "[DONE]" {
				// 关闭当前块
				if converter.blockStarted {
					if event, err := converter.emitContentBlockStop(); err == nil {
						pipeWriter.Write([]byte(event))
					}
				}
				// 发送 message_delta 和 message_stop
				if event, err := converter.emitMessageDelta(""); err == nil {
					pipeWriter.Write([]byte(event))
				}
				if event, err := converter.emitMessageStop(); err == nil {
					pipeWriter.Write([]byte(event))
				}
				return
			}

			// 解析 JSON chunk
			var chunk OpenAIStreamChunk
			if err := json.Unmarshal([]byte(eventData), &chunk); err != nil {
				// 解析错误，跳过
				continue
			}

			// 处理 chunk
			events, err := converter.processChunk(&chunk)
			if err != nil {
				pipeWriter.CloseWithError(err)
				return
			}

			// 写入事件到管道
			for _, event := range events {
				if _, err := pipeWriter.Write([]byte(event)); err != nil {
					return
				}
			}
		}
	}()

	return pipeReader, nil
}

// processChunk 处理单个 OpenAI chunk，返回 Claude 事件列表
func (c *StreamConverter) processChunk(chunk *OpenAIStreamChunk) ([]string, error) {
	events := []string{}

	// 更新元数据
	if chunk.ID != "" {
		c.messageID = chunk.ID
	}
	if chunk.Model != "" {
		c.model = chunk.Model
	}
	if chunk.Created != 0 {
		c.created = chunk.Created
	}

	// 处理第一个 choice
	if len(chunk.Choices) == 0 {
		return events, nil
	}

	choice := chunk.Choices[0]
	delta := choice.Delta

	// 处理 role (第一个 chunk)
	if delta.Role != "" {
		c.role = delta.Role
	}

	// 发送 message_start (如果还没发送)
	if !c.messageStarted {
		event, err := c.emitMessageStart()
		if err != nil {
			return nil, err
		}
		events = append(events, event)
		c.messageStarted = true
	}

	// 处理文本内容
	if delta.Content != "" {
		// 如果当前块未开始，先开始文本块
		if !c.blockStarted {
			c.currentIndex++
			c.currentBlockType = "text"
			event, err := c.emitContentBlockStart("text")
			if err != nil {
				return nil, err
			}
			events = append(events, event)
			c.blockStarted = true
		}

		// 发送文本增量
		c.textBuffer.WriteString(delta.Content)
		event, err := c.emitTextDelta(delta.Content)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	// 处理 tool calls
	if len(delta.ToolCalls) > 0 {
		for _, toolCall := range delta.ToolCalls {
			index := toolCall.Index

			// 获取或创建 tool call 状态
			state, exists := c.toolCallsBuffer[index]
			if !exists {
				// 新的 tool call
				// 如果有文本块在进行，先关闭
				if c.blockStarted && c.currentBlockType == "text" {
					event, err := c.emitContentBlockStop()
					if err != nil {
						return nil, err
					}
					events = append(events, event)
					c.blockStarted = false
				}

				// 创建新状态
				state = &ToolCallState{
					ID:   toolCall.ID,
					Name: toolCall.Function.Name,
				}
				c.toolCallsBuffer[index] = state

				// 开始新的 tool_use 块
				c.currentIndex++
				c.currentBlockType = "tool_use"
				event, err := c.emitToolUseStart(state.ID, state.Name)
				if err != nil {
					return nil, err
				}
				events = append(events, event)
				c.blockStarted = true
			}

			// 累积参数
			if toolCall.Function != nil && toolCall.Function.Arguments != "" {
				state.Arguments.WriteString(toolCall.Function.Arguments)

				// 发送参数增量
				event, err := c.emitToolUseDelta(toolCall.Function.Arguments)
				if err != nil {
					return nil, err
				}
				events = append(events, event)
			}
		}
	}

	// 处理 finish_reason
	if choice.FinishReason != nil && *choice.FinishReason != "" {
		// 关闭当前块
		if c.blockStarted {
			event, err := c.emitContentBlockStop()
			if err != nil {
				return nil, err
			}
			events = append(events, event)
			c.blockStarted = false
		}

		// 转换 finish_reason
		stopReason := convertFinishReason(*choice.FinishReason)

		// 发送 message_delta
		event, err := c.emitMessageDelta(stopReason)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, nil
}

// emitMessageStart 发送 message_start 事件
func (c *StreamConverter) emitMessageStart() (string, error) {
	data := ClaudeMessageStart{
		Type: "message_start",
		Message: ClaudeMessageMetadata{
			ID:         c.messageID,
			Type:       "message",
			Role:       "assistant",
			Content:    []ClaudeContentBlock{},
			Model:      c.model,
			StopReason: nil,
			Usage: ClaudeUsage{
				InputTokens:  c.inputTokens,
				OutputTokens: 0,
			},
		},
	}

	return FormatSSEEvent("message_start", data)
}

// emitContentBlockStart 发送 content_block_start 事件
func (c *StreamConverter) emitContentBlockStart(blockType string) (string, error) {
	var contentBlock ClaudeContentBlock
	if blockType == "text" {
		contentBlock = ClaudeContentBlock{
			Type: "text",
			Text: StringPtr(""),
		}
	}

	data := ClaudeContentBlockStart{
		Type:         "content_block_start",
		Index:        c.currentIndex,
		ContentBlock: contentBlock,
	}

	return FormatSSEEvent("content_block_start", data)
}

// emitToolUseStart 发送 tool_use content_block_start 事件
func (c *StreamConverter) emitToolUseStart(id, name string) (string, error) {
	contentBlock := ClaudeContentBlock{
		Type:  "tool_use",
		ID:    StringPtr(id),
		Name:  StringPtr(name),
		Input: make(map[string]interface{}),
	}

	data := ClaudeContentBlockStart{
		Type:         "content_block_start",
		Index:        c.currentIndex,
		ContentBlock: contentBlock,
	}

	return FormatSSEEvent("content_block_start", data)
}

// emitTextDelta 发送 text_delta 事件
func (c *StreamConverter) emitTextDelta(text string) (string, error) {
	data := ClaudeContentBlockDelta{
		Type:  "content_block_delta",
		Index: c.currentIndex,
		Delta: ClaudeDelta{
			Type: "text_delta",
			Text: text,
		},
	}

	return FormatSSEEvent("content_block_delta", data)
}

// emitToolUseDelta 发送 input_json_delta 事件
func (c *StreamConverter) emitToolUseDelta(partialJSON string) (string, error) {
	data := ClaudeContentBlockDelta{
		Type:  "content_block_delta",
		Index: c.currentIndex,
		Delta: ClaudeDelta{
			Type:        "input_json_delta",
			PartialJSON: partialJSON,
		},
	}

	return FormatSSEEvent("content_block_delta", data)
}

// emitContentBlockStop 发送 content_block_stop 事件
func (c *StreamConverter) emitContentBlockStop() (string, error) {
	data := ClaudeContentBlockStop{
		Type:  "content_block_stop",
		Index: c.currentIndex,
	}

	return FormatSSEEvent("content_block_stop", data)
}

// emitMessageDelta 发送 message_delta 事件
func (c *StreamConverter) emitMessageDelta(stopReason string) (string, error) {
	var stopReasonPtr *string
	if stopReason != "" {
		stopReasonPtr = &stopReason
	}

	data := ClaudeMessageDelta{
		Type: "message_delta",
		Delta: ClaudeMessageDeltaData{
			StopReason: stopReasonPtr,
		},
		Usage: &ClaudeUsage{
			InputTokens:  c.inputTokens,
			OutputTokens: c.outputTokens,
		},
	}

	return FormatSSEEvent("message_delta", data)
}

// emitMessageStop 发送 message_stop 事件
func (c *StreamConverter) emitMessageStop() (string, error) {
	data := ClaudeMessageStop{
		Type: "message_stop",
	}

	return FormatSSEEvent("message_stop", data)
}
