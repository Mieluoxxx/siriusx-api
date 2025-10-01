package converter

import (
	"encoding/json"
	"fmt"
)

// FormatSSEEvent 格式化 Claude SSE 事件
// 生成标准 SSE 格式: event: xxx\ndata: {...}\n\n
func FormatSSEEvent(eventType string, data interface{}) (string, error) {
	// 序列化数据为 JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal event data: %w", err)
	}

	// 生成 SSE 格式
	// event: <eventType>
	// data: <jsonData>
	// <empty line>
	return fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, string(jsonData)), nil
}
