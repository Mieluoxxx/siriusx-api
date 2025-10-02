package events

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Mieluoxxx/Siriusx-API/internal/models"
	"gorm.io/gorm"
)

// Service 事件日志服务
type Service struct {
	db *gorm.DB
}

// NewService 创建事件日志服务实例
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// LogEvent 记录事件
func (s *Service) LogEvent(eventType, message, level string, metadata map[string]interface{}) error {
	// 序列化元数据为 JSON
	var metadataJSON string
	if metadata != nil {
		data, err := json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("序列化元数据失败: %w", err)
		}
		metadataJSON = string(data)
	}

	// 创建事件记录
	event := &models.SystemEvent{
		Type:      eventType,
		Message:   message,
		Level:     level,
		Metadata:  metadataJSON,
		CreatedAt: time.Now(),
	}

	if err := s.db.Create(event).Error; err != nil {
		return fmt.Errorf("保存事件失败: %w", err)
	}

	return nil
}

// LogInfo 记录信息级别事件
func (s *Service) LogInfo(eventType, message string, metadata map[string]interface{}) error {
	return s.LogEvent(eventType, message, models.EventLevelInfo, metadata)
}

// LogWarning 记录警告级别事件
func (s *Service) LogWarning(eventType, message string, metadata map[string]interface{}) error {
	return s.LogEvent(eventType, message, models.EventLevelWarning, metadata)
}

// LogError 记录错误级别事件
func (s *Service) LogError(eventType, message string, metadata map[string]interface{}) error {
	return s.LogEvent(eventType, message, models.EventLevelError, metadata)
}

// GetRecentEvents 获取最近的事件
func (s *Service) GetRecentEvents(limit int) ([]models.SystemEvent, error) {
	var events []models.SystemEvent

	err := s.db.Order("created_at DESC").Limit(limit).Find(&events).Error
	if err != nil {
		return nil, fmt.Errorf("查询事件失败: %w", err)
	}

	return events, nil
}

// GetEventsByType 按类型获取事件
func (s *Service) GetEventsByType(eventType string, limit int) ([]models.SystemEvent, error) {
	var events []models.SystemEvent

	err := s.db.Where("type = ?", eventType).
		Order("created_at DESC").
		Limit(limit).
		Find(&events).Error

	if err != nil {
		return nil, fmt.Errorf("查询事件失败: %w", err)
	}

	return events, nil
}

// GetEventsByLevel 按级别获取事件
func (s *Service) GetEventsByLevel(level string, limit int) ([]models.SystemEvent, error) {
	var events []models.SystemEvent

	err := s.db.Where("level = ?", level).
		Order("created_at DESC").
		Limit(limit).
		Find(&events).Error

	if err != nil {
		return nil, fmt.Errorf("查询事件失败: %w", err)
	}

	return events, nil
}

// CleanupOldEvents 清理旧事件（保留最近N天）
func (s *Service) CleanupOldEvents(days int) (int64, error) {
	cutoffTime := time.Now().AddDate(0, 0, -days)

	result := s.db.Where("created_at < ?", cutoffTime).Delete(&models.SystemEvent{})
	if result.Error != nil {
		return 0, fmt.Errorf("清理旧事件失败: %w", result.Error)
	}

	return result.RowsAffected, nil
}
