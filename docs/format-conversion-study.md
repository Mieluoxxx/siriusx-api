# 格式转换核心学习文档

> 基于 claude-code-nexus 的格式转换实现
> 目标：用 Go 重新实现这套转换逻辑

---

## 1. 核心转换规则

### 1.1 请求转换：Claude → OpenAI

#### **字段映射表**

| Claude 字段 | OpenAI 字段 | 转换规则 |
|------------|------------|----------|
| `model` | `model` | 根据模型映射规则转换（如 `claude-3-5-sonnet` → `gpt-4o`） |
| `system` | `messages[0]` | 转为 `{"role": "system", "content": "..."}` 并放在首位 |
| `messages` | `messages` | 逐条转换（见下方详细规则） |
| `max_tokens` | `max_tokens` | 直接映射 |
| `temperature` | `temperature` | 直接映射 |
| `top_p` | `top_p` | 直接映射 |
| `stream` | `stream` | 直接映射 |
| `tools` | `tools` | 转为 `[{"type": "function", "function": {...}}]` |
| `tool_choice` | `tool_choice` | `{"type": "auto"}` → `"auto"`<br>`{"type": "any"}` → `"required"`<br>`{"type": "tool", "name": "x"}` → `{"type": "function", "function": {"name": "x"}}` |
| `stop_sequences` | `stop` | 直接映射数组 |

---

#### **Messages 转换规则**

##### **用户消息（User）**

**Claude 格式**:
```json
{
  "role": "user",
  "content": [
    {"type": "text", "text": "Hello"},
    {
      "type": "image",
      "source": {
        "type": "base64",
        "media_type": "image/jpeg",
        "data": "base64_string"
      }
    }
  ]
}
```

**OpenAI 格式**:
```json
{
  "role": "user",
  "content": [
    {"type": "text", "text": "Hello"},
    {
      "type": "image_url",
      "image_url": {
        "url": "data:image/jpeg;base64,base64_string"
      }
    }
  ]
}
```

**转换逻辑**:
1. `text` 类型 → 直接映射
2. `image` 类型 → 转为 `image_url`，拼接 data URL
3. 如果只有一个文本块 → 简化为字符串 `"content": "Hello"`

##### **助手消息（Assistant）**

**Claude 格式**:
```json
{
  "role": "assistant",
  "content": [
    {"type": "text", "text": "Sure!"},
    {
      "type": "tool_use",
      "id": "toolu_xxx",
      "name": "get_weather",
      "input": {"location": "SF"}
    }
  ]
}
```

**OpenAI 格式**:
```json
{
  "role": "assistant",
  "content": "Sure!",
  "tool_calls": [
    {
      "id": "toolu_xxx",
      "type": "function",
      "function": {
        "name": "get_weather",
        "arguments": "{\"location\":\"SF\"}"
      }
    }
  ]
}
```

**转换逻辑**:
1. 提取所有 `text` 块 → 合并为 `content` 字符串
2. 提取所有 `tool_use` 块 → 转为 `tool_calls` 数组
3. `input` 对象 → JSON 字符串 `arguments`

##### **Tool Result（特殊处理）**

**Claude 格式**:
```json
{
  "role": "user",
  "content": [
    {
      "type": "tool_result",
      "tool_use_id": "toolu_xxx",
      "content": "{\"temperature\": 72}"
    }
  ]
}
```

**OpenAI 格式**:
```json
{
  "role": "tool",
  "tool_call_id": "toolu_xxx",
  "content": "{\"temperature\": 72}"
}
```

**转换逻辑**:
- `tool_result` 必须拆分为独立的消息
- 角色从 `user` 改为 `tool`

---

### 1.2 响应转换：OpenAI → Claude

#### **非流式响应**

**OpenAI 格式**:
```json
{
  "id": "chatcmpl-xxx",
  "object": "chat.completion",
  "model": "gpt-4o",
  "choices": [{
    "index": 0,
    "message": {
      "role": "assistant",
      "content": "Hello!",
      "tool_calls": [{
        "id": "call_xxx",
        "type": "function",
        "function": {
          "name": "get_weather",
          "arguments": "{\"location\":\"SF\"}"
        }
      }]
    },
    "finish_reason": "stop"
  }],
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 20,
    "total_tokens": 30
  }
}
```

**Claude 格式**:
```json
{
  "id": "msg_xxx",
  "type": "message",
  "role": "assistant",
  "content": [
    {"type": "text", "text": "Hello!"},
    {
      "type": "tool_use",
      "id": "call_xxx",
      "name": "get_weather",
      "input": {"location": "SF"}
    }
  ],
  "model": "claude-3-5-sonnet-20240620",
  "stop_reason": "end_turn",
  "stop_sequence": null,
  "usage": {
    "input_tokens": 10,
    "output_tokens": 20
  }
}
```

**转换规则**:

| OpenAI | Claude | 说明 |
|--------|--------|------|
| `choices[0].message.content` | `content[0]` | 转为 `{"type": "text", "text": "..."}` |
| `choices[0].message.tool_calls` | `content[1...]` | 每个 tool_call 转为 `{"type": "tool_use", ...}` |
| `choices[0].finish_reason` | `stop_reason` | `"stop"` → `"end_turn"`<br>`"length"` → `"max_tokens"`<br>`"tool_calls"` → `"tool_use"` |
| `usage.prompt_tokens` | `usage.input_tokens` | 直接映射 |
| `usage.completion_tokens` | `usage.output_tokens` | 直接映射 |

---

#### **流式响应（SSE）- 核心难点**

**关键概念**:
- OpenAI 流式返回 `data: {...delta...}` 格式
- Claude 流式返回 **事件序列**: `ping` → `message_start` → `content_block_start` → `content_block_delta` → ...

**事件流程**:

```
1. [收到第一个 chunk]
   → 发送: event: message_start

2. [delta.content 首次出现]
   → 发送: event: content_block_start (index: 0, type: text)

3. [每个 delta.content]
   → 发送: event: content_block_delta (type: text_delta, text: "...")

4. [delta.tool_calls 首次出现]
   → 发送: event: content_block_start (index: 1, type: tool_use)

5. [每个 delta.tool_calls[i].function.arguments]
   → 发送: event: content_block_delta (type: input_json_delta, partial_json: "...")

6. [收到 [DONE]]
   → 发送: event: content_block_stop (for each block)
   → 发送: event: message_delta (stop_reason, usage)
   → 发送: event: message_stop
```

**状态管理**（关键！）:
```typescript
class StreamConverter {
  private contentBlocks: Array<{
    type: "text" | "tool_use";
    index: number;
    started: boolean;
    content?: any;
  }>;

  private currentToolArgs: Record<string, string>; // 累积工具参数

  // 核心方法
  processOpenAIChunk(chunk) {
    // 1. 检测 delta.content → 文本块
    // 2. 检测 delta.tool_calls → 工具块
    // 3. 维护 index 计数
    // 4. 生成对应的 Claude 事件
  }
}
```

**Go 实现重点**:
```go
type StreamConverter struct {
    MessageID      string
    OriginalModel  string
    ContentBlocks  []ContentBlock
    ToolArgsBuffer map[string]string
}

type ContentBlock struct {
    Type    string // "text" | "tool_use"
    Index   int
    Started bool
    Content interface{}
}

func (sc *StreamConverter) ProcessOpenAIChunk(chunk OpenAIChunk) []SSEEvent {
    events := []SSEEvent{}

    // 处理文本增量
    if chunk.Choices[0].Delta.Content != "" {
        blockIndex := sc.getOrCreateTextBlock()
        events = append(events, sc.handleTextDelta(blockIndex, chunk.Choices[0].Delta.Content)...)
    }

    // 处理工具调用增量
    if len(chunk.Choices[0].Delta.ToolCalls) > 0 {
        for _, toolCall := range chunk.Choices[0].Delta.ToolCalls {
            blockIndex := sc.getOrCreateToolBlock(toolCall.Index, toolCall)
            events = append(events, sc.handleToolDelta(blockIndex, toolCall)...)
        }
    }

    return events
}
```

---

## 2. 关键技术细节

### 2.1 SSE 格式

**Claude SSE 格式**:
```
event: message_start
data: {"type":"message_start","message":{...}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hi"}}

event: message_stop
data: {"type":"message_stop"}
```

**Go 实现**:
```go
type SSEEvent struct {
    Event string
    Data  interface{}
}

func (e SSEEvent) Format() string {
    dataJSON, _ := json.Marshal(e.Data)
    return fmt.Sprintf("event: %s\ndata: %s\n\n", e.Event, dataJSON)
}
```

### 2.2 工具参数累积

OpenAI 流式 `arguments` 是分片发送的：
```
chunk1: {"arguments": "{\"loc"}
chunk2: {"arguments": "ation\":\""}
chunk3: {"arguments": "SF\"}"}
```

需要累积完整字符串，但 Claude 每次都发送 `partial_json` 片段：
```go
// 在 Go 中实现
toolArgsBuffer[toolID] += chunk.Arguments
// 发送增量
event := SSEEvent{
    Event: "content_block_delta",
    Data: map[string]interface{}{
        "delta": map[string]string{
            "type": "input_json_delta",
            "partial_json": chunk.Arguments,
        },
    },
}
```

### 2.3 Index 管理

**规则**:
- 第一个文本块: `index: 0`
- 第一个工具块: `index: 1`（在文本块之后）
- 第二个工具块: `index: 2`

**Go 实现**:
```go
func (sc *StreamConverter) getOrCreateTextBlock() int {
    for _, block := range sc.ContentBlocks {
        if block.Type == "text" {
            return block.Index
        }
    }
    // 创建新文本块
    newIndex := 0
    sc.ContentBlocks = append(sc.ContentBlocks, ContentBlock{
        Type: "text",
        Index: newIndex,
        Started: false,
    })
    return newIndex
}

func (sc *StreamConverter) getOrCreateToolBlock(toolIndex int, toolCall ToolCall) int {
    // 计算块索引：文本块数量 + 工具索引
    textBlockCount := 0
    for _, block := range sc.ContentBlocks {
        if block.Type == "text" {
            textBlockCount++
        }
    }
    blockIndex := textBlockCount + toolIndex

    // 检查是否已存在
    for _, block := range sc.ContentBlocks {
        if block.Index == blockIndex {
            return blockIndex
        }
    }

    // 创建新工具块
    sc.ContentBlocks = append(sc.ContentBlocks, ContentBlock{
        Type: "tool_use",
        Index: blockIndex,
        Started: false,
        Content: map[string]interface{}{
            "id": toolCall.ID,
            "name": toolCall.Function.Name,
        },
    })
    return blockIndex
}
```

---

## 3. Go 实现骨架

### 3.1 项目结构

```
internal/
├── converter/
│   ├── claude_to_openai.go      # 请求转换
│   ├── openai_to_claude.go      # 响应转换
│   ├── stream_converter.go      # 流式转换
│   └── types.go                 # 类型定义
├── models/
│   ├── claude.go                # Claude API 结构体
│   └── openai.go                # OpenAI API 结构体
└── utils/
    └── sse.go                   # SSE 工具
```

### 3.2 核心接口

```go
package converter

// 请求转换
func ConvertClaudeToOpenAI(claudeReq ClaudeRequest, targetModel string) OpenAIRequest

// 非流式响应转换
func ConvertOpenAIToClaude(openAIResp OpenAIResponse, originalModel string) ClaudeResponse

// 流式转换器
type StreamConverter struct {
    MessageID      string
    OriginalModel  string
    ContentBlocks  []ContentBlock
    ToolArgsBuffer map[string]string
    TotalInputTokens  int
    TotalOutputTokens int
}

func NewStreamConverter(originalModel string) *StreamConverter
func (sc *StreamConverter) GenerateInitialEvents() []SSEEvent
func (sc *StreamConverter) ProcessOpenAIChunk(chunk OpenAIChunk) []SSEEvent
func (sc *StreamConverter) GenerateFinishEvents(finishReason string) []SSEEvent
```

---

## 4. 测试用例

### 4.1 基础文本转换

**输入（Claude）**:
```json
{
  "model": "claude-3-5-sonnet-20240620",
  "max_tokens": 1024,
  "messages": [
    {"role": "user", "content": "Hello"}
  ]
}
```

**期望（OpenAI）**:
```json
{
  "model": "gpt-4o",
  "max_tokens": 1024,
  "messages": [
    {"role": "user", "content": "Hello"}
  ]
}
```

### 4.2 工具调用转换

**输入（Claude）**:
```json
{
  "model": "claude-3-5-sonnet-20240620",
  "messages": [...],
  "tools": [{
    "name": "get_weather",
    "description": "Get weather",
    "input_schema": {
      "type": "object",
      "properties": {
        "location": {"type": "string"}
      }
    }
  }],
  "tool_choice": {"type": "auto"}
}
```

**期望（OpenAI）**:
```json
{
  "model": "gpt-4o",
  "messages": [...],
  "tools": [{
    "type": "function",
    "function": {
      "name": "get_weather",
      "description": "Get weather",
      "parameters": {
        "type": "object",
        "properties": {
          "location": {"type": "string"}
        }
      }
    }
  }],
  "tool_choice": "auto"
}
```

### 4.3 流式响应转换

**输入（OpenAI SSE）**:
```
data: {"choices":[{"delta":{"content":"Hello"}}]}

data: {"choices":[{"delta":{"content":" world"}}]}

data: {"choices":[{"finish_reason":"stop"}]}

data: [DONE]
```

**期望（Claude SSE）**:
```
event: message_start
data: {"type":"message_start","message":{...}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" world"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"}}

event: message_stop
data: {"type":"message_stop"}
```

---

## 5. 实现优先级

### MVP（必须）
- [x] 基础请求转换（文本消息）
- [x] 基础响应转换（文本消息）
- [x] 流式响应转换（纯文本）
- [x] System prompt 处理

### V1.1（重要）
- [ ] 工具调用转换（请求）
- [ ] 工具调用转换（响应）
- [ ] 流式工具调用

### V1.2（增强）
- [ ] 图片消息转换
- [ ] Tool result 转换
- [ ] 多模态内容

---

**参考文件**:
- [claudeConverter.ts](../ref_proj/claude-code-nexus/src/utils/claudeConverter.ts)
- [claude.ts](../ref_proj/claude-code-nexus/src/routes/claude.ts)
- [REQUIREMENTS.md](../ref_proj/claude-code-nexus/REQUIREMENTS.md)
