package models

import "time"

// SystemEvent 系统事件日志
// 用于记录系统重要事件，如故障转移、配置变更、健康检查等
type SystemEvent struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Type      string    `gorm:"type:varchar(50);not null;index" json:"type"` // failover, config_change, health_check, etc.
	Message   string    `gorm:"type:text;not null" json:"message"`
	Level     string    `gorm:"type:varchar(20);not null;default:'info'" json:"level"` // info, warning, error
	Metadata  string    `gorm:"type:json" json:"metadata,omitempty"`                   // 额外的元数据（JSON 格式）
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

// TableName 指定表名
func (SystemEvent) TableName() string {
	return "system_events"
}

// EventType 事件类型常量
const (
	EventTypeFailover      = "failover"       // 故障转移
	EventTypeConfigChange  = "config_change"  // 配置变更
	EventTypeHealthCheck   = "health_check"   // 健康检查
	EventTypeProviderAdded = "provider_added" // 供应商添加
	EventTypeProviderError = "provider_error" // 供应商错误
)

// EventLevel 事件级别常量
const (
	EventLevelInfo    = "info"
	EventLevelWarning = "warning"
	EventLevelError   = "error"
)
