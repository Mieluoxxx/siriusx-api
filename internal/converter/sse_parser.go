package converter

import (
	"bufio"
	"bytes"
	"io"
	"strings"
)

// SSEParser SSE (Server-Sent Events) 事件解析器
type SSEParser struct {
	scanner *bufio.Scanner
}

// NewSSEParser 创建 SSE 解析器
func NewSSEParser(r io.Reader) *SSEParser {
	scanner := bufio.NewScanner(r)
	// 自定义分隔函数：以双换行为分隔符
	scanner.Split(splitSSEEvent)
	return &SSEParser{
		scanner: scanner,
	}
}

// ParseEvent 解析下一个 SSE 事件
// 返回事件数据，或 io.EOF 表示流结束
func (p *SSEParser) ParseEvent() (string, error) {
	if !p.scanner.Scan() {
		if err := p.scanner.Err(); err != nil {
			return "", err
		}
		return "", io.EOF
	}

	eventText := p.scanner.Text()
	if eventText == "" {
		// 空事件，继续读取下一个
		return p.ParseEvent()
	}

	// 解析事件内容
	lines := strings.Split(eventText, "\n")
	var data strings.Builder

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 处理 data: 行
		if strings.HasPrefix(line, "data: ") {
			content := strings.TrimPrefix(line, "data: ")
			data.WriteString(content)
		}
		// 忽略 event:, id:, retry: 等字段（Claude API 不需要）
	}

	return data.String(), nil
}

// splitSSEEvent 自定义分隔函数，以双换行为事件分隔符
func splitSSEEvent(data []byte, atEOF bool) (advance int, token []byte, err error) {
	// 查找双换行
	delimiter := []byte("\n\n")
	if i := bytes.Index(data, delimiter); i >= 0 {
		// 找到分隔符，返回事件数据
		return i + len(delimiter), data[0:i], nil
	}

	// 如果到达 EOF
	if atEOF {
		if len(data) > 0 {
			// 返回剩余数据
			return len(data), data, nil
		}
		return 0, nil, nil
	}

	// 请求更多数据
	return 0, nil, nil
}
