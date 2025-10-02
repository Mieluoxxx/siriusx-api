package handlers

import (
	"net/http"

	"github.com/Mieluoxxx/Siriusx-API/internal/events"
	"github.com/Mieluoxxx/Siriusx-API/internal/stats"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// StatsHandler 统计信息处理器
type StatsHandler struct {
	db             *gorm.DB
	requestCounter *stats.RequestCounter
	eventService   *events.Service
}

// NewStatsHandler 创建统计处理器
func NewStatsHandler(db *gorm.DB, requestCounter *stats.RequestCounter, eventService *events.Service) *StatsHandler {
	return &StatsHandler{
		db:             db,
		requestCounter: requestCounter,
		eventService:   eventService,
	}
}

// SystemStats 系统统计信息响应
type SystemStats struct {
	Providers     ProviderStats     `json:"providers"`
	Requests      RequestStats      `json:"requests"`
	RecentEvents  []Event           `json:"recent_events"`
}

// ProviderStats 供应商统计
type ProviderStats struct {
	Total     int `json:"total"`
	Healthy   int `json:"healthy"`
	Unhealthy int `json:"unhealthy"`
}

// RequestStats 请求统计
type RequestStats struct {
	Total      int64   `json:"total"`
	CurrentQPS float64 `json:"current_qps"`
}

// Event 事件日志
type Event struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Message   string `json:"message"`
}

// GetStats 获取系统统计信息
// @Summary 获取系统统计信息
// @Description 获取系统概览统计数据，包括供应商状态、请求统计、QPS 等
// @Tags Stats
// @Produce json
// @Success 200 {object} SystemStats
// @Router /api/stats [get]
func (h *StatsHandler) GetStats(c *gin.Context) {
	// 查询供应商统计
	var totalProviders int64
	var healthyProviders int64

	h.db.Table("providers").Count(&totalProviders)
	h.db.Table("providers").Where("health_status = ?", "healthy").Count(&healthyProviders)

	// 获取请求统计
	requestStats := h.requestCounter.GetStats()

	// 获取最近事件（最多 10 条）
	recentEventsData, err := h.eventService.GetRecentEvents(10)
	recentEvents := make([]Event, 0, len(recentEventsData))

	if err == nil {
		for _, evt := range recentEventsData {
			recentEvents = append(recentEvents, Event{
				Timestamp: evt.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				Type:      evt.Type,
				Message:   evt.Message,
			})
		}
	}

	stats := SystemStats{
		Providers: ProviderStats{
			Total:     int(totalProviders),
			Healthy:   int(healthyProviders),
			Unhealthy: int(totalProviders - healthyProviders),
		},
		Requests: RequestStats{
			Total:      requestStats.Total,
			CurrentQPS: requestStats.CurrentQPS,
		},
		RecentEvents: recentEvents,
	}

	c.JSON(http.StatusOK, stats)
}
