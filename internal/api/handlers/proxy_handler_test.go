package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Mieluoxxx/Siriusx-API/internal/converter"
	"github.com/Mieluoxxx/Siriusx-API/internal/models"
	"github.com/gin-gonic/gin"
)

func TestShouldConvertToOpenAI(t *testing.T) {
	handler := &ProxyHandler{}

	claudeProvider := &models.Provider{BaseURL: "https://api.anthropic.com"}
	claudeMapping := &models.ModelMapping{TargetModel: "claude-3-5-sonnet"}

	if handler.shouldConvertToOpenAI(claudeProvider, claudeMapping) {
		t.Fatalf("expected anthropic upstream to skip conversion")
	}

	openAIProvider := &models.Provider{BaseURL: "https://newapi.ixio.cc"}
	openAIMapping := &models.ModelMapping{TargetModel: "glm-4.6"}

	if !handler.shouldConvertToOpenAI(openAIProvider, openAIMapping) {
		t.Fatalf("expected non-claude upstream to require conversion")
	}
}

func TestForwardClaudeViaOpenAI(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var capturedBody []byte
	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read upstream request body: %v", err)
		}
		capturedBody = body

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
            "id": "chatcmpl-123",
            "object": "chat.completion",
            "created": 123,
            "model": "glm-4.6",
            "choices": [{
                "index": 0,
                "message": {"role": "assistant", "content": "pong"},
                "finish_reason": "stop"
            }],
            "usage": {
                "prompt_tokens": 5,
                "completion_tokens": 3,
                "total_tokens": 8
            }
        }`))
	}))
	defer server.Close()

	handler := &ProxyHandler{}
	provider := &models.Provider{
		Name:    "ixio",
		BaseURL: server.URL,
		APIKey:  "sk-test",
	}
	mapping := &models.ModelMapping{TargetModel: "glm-4.6"}

	raw := []byte(`{
        "model": "claude-sonnet-4-5-20250929",
        "messages": [{
            "role": "user",
            "content": [{"type": "text", "text": "ping"}]
        }]
    }`)

	var req map[string]interface{}
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatalf("failed to prepare test payload: %v", err)
	}

	req["model"] = "claude-sonnet-4-5-20250929"
	handler.sanitizeRequest(req, provider.Name)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/messages", bytes.NewBuffer(raw))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.forwardClaudeViaOpenAI(c, provider, mapping, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	if capturedPath != "/v1/chat/completions" {
		t.Fatalf("expected upstream to hit /v1/chat/completions, got %s", capturedPath)
	}

	var openaiReq converter.OpenAIRequest
	if err := json.Unmarshal(capturedBody, &openaiReq); err != nil {
		t.Fatalf("failed to decode upstream request: %v", err)
	}
	if openaiReq.Model != "glm-4.6" {
		t.Fatalf("expected upstream model glm-4.6, got %s", openaiReq.Model)
	}

	var claudeResp converter.ClaudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&claudeResp); err != nil {
		t.Fatalf("failed to decode downstream response: %v", err)
	}

	if len(claudeResp.Content) == 0 || claudeResp.Content[0].Text == nil {
		t.Fatalf("expected Claude text content in response")
	}
	if *claudeResp.Content[0].Text != "pong" {
		t.Fatalf("expected converted text 'pong', got %s", *claudeResp.Content[0].Text)
	}
}

func TestNormalizeClaudePayload(t *testing.T) {
	handler := &ProxyHandler{}
	req := map[string]interface{}{
		"messages": []interface{}{
			map[string]interface{}{
				"role":    "user",
				"content": "hello world",
			},
		},
		"system": []interface{}{
			"alpha",
			map[string]interface{}{"text": "beta"},
		},
	}

	handler.normalizeClaudePayload(req)

	msgs, ok := req["messages"].([]interface{})
	if !ok || len(msgs) != 1 {
		t.Fatalf("normalized messages not preserved")
	}

	msgMap, ok := msgs[0].(map[string]interface{})
	if !ok {
		t.Fatalf("message payload type mismatch")
	}

	content, ok := msgMap["content"].([]interface{})
	if !ok || len(content) != 1 {
		t.Fatalf("content should be wrapped into array")
	}

	block, ok := content[0].(map[string]interface{})
	if !ok || block["type"] != "text" || block["text"] != "hello world" {
		t.Fatalf("content block mismatch: %+v", block)
	}

	system, ok := req["system"].(string)
	if !ok || system != "alpha\nbeta" {
		t.Fatalf("system field normalization failed: %v", req["system"])
	}
}

func TestMessagesCountTokens(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &ProxyHandler{}

	requestBody := []byte(`{
        "model": "claude-sonnet-4-5-20250929",
        "messages": [{
            "role": "user",
            "content": "hello world"
        }],
        "system": ["remember"],
        "metadata": {"client": "test"}
    }`)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/messages/count_tokens", bytes.NewBuffer(requestBody))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.MessagesCountTokens(c)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	var payload map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	usage, ok := payload["usage"].(map[string]interface{})
	if !ok {
		t.Fatalf("usage field missing in response")
	}

	inputTokens, ok := usage["input_tokens"].(float64)
	if !ok || inputTokens == 0 {
		t.Fatalf("input_tokens should be positive, got %v", usage["input_tokens"])
	}

	if usage["output_tokens"].(float64) != 0 {
		t.Fatalf("output_tokens should be 0, got %v", usage["output_tokens"])
	}
}
