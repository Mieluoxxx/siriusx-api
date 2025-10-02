package stats

import (
	"testing"
	"time"
)

func TestRequestCounter_Increment(t *testing.T) {
	counter := NewRequestCounter(1 * time.Second)

	// 测试基本计数
	for i := 0; i < 10; i++ {
		counter.Increment()
	}

	total := counter.GetTotal()
	if total != 10 {
		t.Errorf("Expected total 10, got %d", total)
	}
}

func TestRequestCounter_QPS(t *testing.T) {
	counter := NewRequestCounter(2 * time.Second)

	// 模拟快速请求
	for i := 0; i < 100; i++ {
		counter.Increment()
	}

	qps := counter.GetQPS()
	if qps <= 0 {
		t.Errorf("Expected QPS > 0, got %f", qps)
	}

	t.Logf("QPS: %.2f", qps)
}

func TestRequestCounter_WindowRotation(t *testing.T) {
	counter := NewRequestCounter(1 * time.Second)

	// 第一批请求
	for i := 0; i < 10; i++ {
		counter.Increment()
	}

	firstQPS := counter.GetQPS()
	t.Logf("First QPS: %.2f", firstQPS)

	// 等待窗口滚动
	time.Sleep(1500 * time.Millisecond)

	// 第二批请求
	for i := 0; i < 20; i++ {
		counter.Increment()
	}

	secondQPS := counter.GetQPS()
	t.Logf("Second QPS: %.2f", secondQPS)

	// 总请求数应该是 30
	total := counter.GetTotal()
	if total != 30 {
		t.Errorf("Expected total 30, got %d", total)
	}
}

func TestRequestCounter_GetStats(t *testing.T) {
	counter := NewRequestCounter(1 * time.Second)

	for i := 0; i < 50; i++ {
		counter.Increment()
	}

	stats := counter.GetStats()

	if stats.Total != 50 {
		t.Errorf("Expected total 50, got %d", stats.Total)
	}

	if stats.CurrentQPS <= 0 {
		t.Errorf("Expected QPS > 0, got %f", stats.CurrentQPS)
	}

	t.Logf("Stats: Total=%d, QPS=%.2f", stats.Total, stats.CurrentQPS)
}
