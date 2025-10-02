package events

import (
	"testing"

	"github.com/Mieluoxxx/Siriusx-API/internal/db"
	"github.com/Mieluoxxx/Siriusx-API/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(database)
	require.NoError(t, err)

	return database
}

func TestEventService_LogEvent(t *testing.T) {
	database := setupTestDB(t)
	service := NewService(database)

	// 测试记录事件
	err := service.LogInfo(models.EventTypeHealthCheck, "健康检查成功", map[string]interface{}{
		"provider_id": 1,
		"status":      "healthy",
	})
	require.NoError(t, err)

	// 验证事件已保存
	var count int64
	database.Model(&models.SystemEvent{}).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestEventService_GetRecentEvents(t *testing.T) {
	database := setupTestDB(t)
	service := NewService(database)

	// 插入多个事件
	for i := 0; i < 15; i++ {
		service.LogInfo(models.EventTypeHealthCheck, "测试事件", nil)
	}

	// 获取最近 10 条
	events, err := service.GetRecentEvents(10)
	require.NoError(t, err)
	assert.Equal(t, 10, len(events))
}

func TestEventService_GetEventsByType(t *testing.T) {
	database := setupTestDB(t)
	service := NewService(database)

	// 插入不同类型的事件
	service.LogInfo(models.EventTypeHealthCheck, "健康检查1", nil)
	service.LogInfo(models.EventTypeHealthCheck, "健康检查2", nil)
	service.LogWarning(models.EventTypeFailover, "故障转移", nil)

	// 按类型查询
	events, err := service.GetEventsByType(models.EventTypeHealthCheck, 10)
	require.NoError(t, err)
	assert.Equal(t, 2, len(events))

	for _, evt := range events {
		assert.Equal(t, models.EventTypeHealthCheck, evt.Type)
	}
}

func TestEventService_GetEventsByLevel(t *testing.T) {
	database := setupTestDB(t)
	service := NewService(database)

	// 插入不同级别的事件
	service.LogInfo(models.EventTypeHealthCheck, "信息事件", nil)
	service.LogWarning(models.EventTypeFailover, "警告事件", nil)
	service.LogError(models.EventTypeProviderError, "错误事件", nil)

	// 按级别查询
	errorEvents, err := service.GetEventsByLevel(models.EventLevelError, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, len(errorEvents))
	assert.Equal(t, models.EventLevelError, errorEvents[0].Level)
}

func TestEventService_CleanupOldEvents(t *testing.T) {
	database := setupTestDB(t)
	service := NewService(database)

	// 插入事件
	for i := 0; i < 5; i++ {
		service.LogInfo(models.EventTypeHealthCheck, "测试事件", nil)
	}

	// 清理旧事件（保留最近 0 天，即全部清理）
	deleted, err := service.CleanupOldEvents(0)
	require.NoError(t, err)
	assert.Equal(t, int64(5), deleted)

	// 验证已清理
	var count int64
	database.Model(&models.SystemEvent{}).Count(&count)
	assert.Equal(t, int64(0), count)
}
