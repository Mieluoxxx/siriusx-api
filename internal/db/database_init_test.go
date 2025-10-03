package db

import (
	"testing"
	"time"

	"github.com/Mieluoxxx/Siriusx-API/internal/config"
	"github.com/Mieluoxxx/Siriusx-API/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitDefaultData(t *testing.T) {
	// 创建测试数据库配置
	cfg := &config.DatabaseConfig{
		Path:            ":memory:",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
	}

	// 初始化数据库
	db, err := InitDatabase(cfg)
	require.NoError(t, err)

	// 执行迁移（包含默认数据初始化）
	err = AutoMigrate(db)
	require.NoError(t, err)

	// 验证创建了3个默认模型
	var count int64
	db.Model(&models.UnifiedModel{}).Count(&count)
	assert.Equal(t, int64(3), count, "应该创建3个默认模型")

	// 验证每个默认模型
	expectedModels := []struct {
		name        string
		displayName string
		description string
	}{
		{
			name:        "claude-3-5-haiku-20241022",
			displayName: "claude-3-5-haiku-20241022",
			description: "ClaudeCode默认haiku模型",
		},
		{
			name:        "claude-sonnet-4-5-20250929",
			displayName: "claude-sonnet-4-5-20250929",
			description: "ClaudeCode默认sonnet模型",
		},
		{
			name:        "claude-opus-4-1-20250805",
			displayName: "claude-opus-4-1-20250805",
			description: "ClaudeCode默认opus模型",
		},
	}

	for _, expected := range expectedModels {
		var model models.UnifiedModel
		err = db.Where("name = ?", expected.name).First(&model).Error
		require.NoError(t, err, "应该找到模型: %s", expected.name)

		assert.Equal(t, expected.name, model.Name)
		assert.Equal(t, expected.displayName, model.DisplayName)
		assert.Equal(t, expected.description, model.Description)

		t.Logf("✅ 验证默认模型: %s", model.Name)
	}
}

func TestInitDefaultData_SkipIfDataExists(t *testing.T) {
	// 创建测试数据库配置
	cfg := &config.DatabaseConfig{
		Path:            ":memory:",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
	}

	// 初始化数据库
	db, err := InitDatabase(cfg)
	require.NoError(t, err)

	// 执行迁移
	err = AutoMigrate(db)
	require.NoError(t, err)

	// 统计模型数量
	var count1 int64
	db.Model(&models.UnifiedModel{}).Count(&count1)
	assert.Equal(t, int64(3), count1, "应该有3个默认模型")

	// 再次执行初始化（不应该重复创建）
	err = initDefaultData(db)
	require.NoError(t, err)

	// 验证模型数量没有增加
	var count2 int64
	db.Model(&models.UnifiedModel{}).Count(&count2)
	assert.Equal(t, int64(3), count2, "模型数量不应该增加")

	t.Log("✅ 已有数据时正确跳过初始化")
}
