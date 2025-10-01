package converter

// Claude Types - Claude Messages API 请求和响应类型定义

// ClaudeRequest Claude Messages API 请求
type ClaudeRequest struct {
	Model         string             `json:"model"`
	System        string             `json:"system,omitempty"`
	Messages      []ClaudeMessage    `json:"messages"`
	MaxTokens     int                `json:"max_tokens"`
	Temperature   *float64           `json:"temperature,omitempty"`
	TopP          *float64           `json:"top_p,omitempty"`
	Stream        bool               `json:"stream,omitempty"`
	StopSequences []string           `json:"stop_sequences,omitempty"`
	Tools         []ClaudeTool       `json:"tools,omitempty"`
	ToolChoice    *ClaudeToolChoice  `json:"tool_choice,omitempty"`
}

// ClaudeMessage Claude 消息
type ClaudeMessage struct {
	Role    string               `json:"role"`
	Content []ClaudeContentBlock `json:"content"`
}

// ClaudeContentBlock Claude 内容块
// 支持多种类型: text, image, tool_use, tool_result
type ClaudeContentBlock struct {
	Type string `json:"type"`

	// text 类型
	Text *string `json:"text,omitempty"`

	// image 类型
	Source *ClaudeImageSource `json:"source,omitempty"`

	// tool_use 类型
	ID    *string                `json:"id,omitempty"`
	Name  *string                `json:"name,omitempty"`
	Input map[string]interface{} `json:"input,omitempty"`

	// tool_result 类型
	ToolUseID *string `json:"tool_use_id,omitempty"`
	Content   *string `json:"content,omitempty"` // tool result content
}

// ClaudeImageSource 图片来源
type ClaudeImageSource struct {
	Type      string `json:"type"`       // base64 | url
	MediaType string `json:"media_type"` // image/jpeg, image/png, etc.
	Data      string `json:"data"`       // base64 string or URL
}

// ClaudeTool Claude 工具定义
type ClaudeTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// ClaudeToolChoice Claude 工具选择
type ClaudeToolChoice struct {
	Type string  `json:"type"` // auto | any | tool
	Name *string `json:"name,omitempty"`
}

// ClaudeResponse Claude Messages API 响应
type ClaudeResponse struct {
	ID           string               `json:"id"`
	Type         string               `json:"type"` // "message"
	Role         string               `json:"role"` // "assistant"
	Content      []ClaudeContentBlock `json:"content"`
	Model        string               `json:"model"`
	StopReason   string               `json:"stop_reason,omitempty"`   // "end_turn" | "max_tokens" | "stop_sequence" | "tool_use"
	StopSequence *string              `json:"stop_sequence,omitempty"` // which stop sequence was hit
	Usage        ClaudeUsage          `json:"usage"`
}

// ClaudeUsage Claude token 使用情况
type ClaudeUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}


// OpenAI Types - OpenAI Chat Completions API 请求和响应类型定义

// OpenAIRequest OpenAI Chat Completions API 请求
type OpenAIRequest struct {
	Model       string           `json:"model"`
	Messages    []OpenAIMessage  `json:"messages"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
	Temperature *float64         `json:"temperature,omitempty"`
	TopP        *float64         `json:"top_p,omitempty"`
	Stream      bool             `json:"stream,omitempty"`
	Stop        []string         `json:"stop,omitempty"`
	Tools       []OpenAITool     `json:"tools,omitempty"`
	ToolChoice  interface{}      `json:"tool_choice,omitempty"` // string or object
}

// OpenAIMessage OpenAI 消息
type OpenAIMessage struct {
	Role       string           `json:"role"`
	Content    interface{}      `json:"content,omitempty"` // string or []OpenAIContentBlock
	ToolCalls  []OpenAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"` // for tool role
}

// OpenAIContentBlock OpenAI 内容块
type OpenAIContentBlock struct {
	Type     string          `json:"type"`
	Text     *string         `json:"text,omitempty"`
	ImageURL *OpenAIImageURL `json:"image_url,omitempty"`
}

// OpenAIImageURL 图片 URL
type OpenAIImageURL struct {
	URL string `json:"url"` // data URI or HTTP URL
}

// OpenAIToolCall OpenAI 工具调用
type OpenAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"` // always "function"
	Function OpenAIFunctionCall `json:"function"`
}

// OpenAIFunctionCall OpenAI 函数调用
type OpenAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// OpenAITool OpenAI 工具定义
type OpenAITool struct {
	Type     string            `json:"type"` // always "function"
	Function OpenAIFunctionDef `json:"function"`
}

// OpenAIFunctionDef OpenAI 函数定义
type OpenAIFunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// OpenAIResponse OpenAI Chat Completions API 响应
type OpenAIResponse struct {
	ID      string                `json:"id"`
	Object  string                `json:"object"` // "chat.completion"
	Created int64                 `json:"created"`
	Model   string                `json:"model"`
	Choices []OpenAIChoice        `json:"choices"`
	Usage   OpenAIUsage           `json:"usage"`
}

// OpenAIChoice OpenAI 响应选项
type OpenAIChoice struct {
	Index        int           `json:"index"`
	Message      OpenAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"` // "stop" | "length" | "tool_calls" | "content_filter"
}

// OpenAIUsage OpenAI token 使用情况
type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}


// Helper functions

// StringPtr 返回字符串指针
func StringPtr(s string) *string {
	return &s
}

// Float64Ptr 返回 float64 指针
func Float64Ptr(f float64) *float64 {
	return &f
}
