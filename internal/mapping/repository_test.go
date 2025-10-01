package mapping

import (
	"testing"

	"github.com/Mieluoxxx/Siriusx-API/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
	// 直接创建内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// 手动迁移只需要的模型
	err = db.AutoMigrate(&models.UnifiedModel{})
	require.NoError(t, err)

	return db
}

func TestRepository_Create(t *testing.T) {
	database := setupTestDB(t)
	repo := NewRepository(database)

	model := &models.UnifiedModel{
		Name:        "test-model",
		Description: "Test model description",
	}

	err := repo.Create(model)
	assert.NoError(t, err)
	assert.NotZero(t, model.ID)
	assert.NotZero(t, model.CreatedAt)
	assert.NotZero(t, model.UpdatedAt)
}

func TestRepository_FindByID(t *testing.T) {
	database := setupTestDB(t)
	repo := NewRepository(database)

	// 创建测试模型
	model := &models.UnifiedModel{
		Name:        "test-model",
		Description: "Test model description",
	}
	err := repo.Create(model)
	require.NoError(t, err)

	// 测试成功查找
	found, err := repo.FindByID(model.ID)
	assert.NoError(t, err)
	assert.Equal(t, model.Name, found.Name)
	assert.Equal(t, model.Description, found.Description)

	// 测试找不到的情况
	_, err = repo.FindByID(9999)
	assert.ErrorIs(t, err, ErrModelNotFound)
}

func TestRepository_FindByName(t *testing.T) {
	database := setupTestDB(t)
	repo := NewRepository(database)

	// 创建测试模型
	model := &models.UnifiedModel{
		Name:        "test-model",
		Description: "Test model description",
	}
	err := repo.Create(model)
	require.NoError(t, err)

	// 测试成功查找
	found, err := repo.FindByName("test-model")
	assert.NoError(t, err)
	assert.Equal(t, model.ID, found.ID)
	assert.Equal(t, model.Description, found.Description)

	// 测试找不到的情况
	_, err = repo.FindByName("non-existent")
	assert.ErrorIs(t, err, ErrModelNotFound)
}

func TestRepository_FindAll(t *testing.T) {
	database := setupTestDB(t)
	repo := NewRepository(database)

	// 创建测试数据
	models := []*models.UnifiedModel{
		{Name: "claude-sonnet-4", Description: "Claude Sonnet 4"},
		{Name: "gpt-4o", Description: "GPT-4o"},
		{Name: "claude-haiku", Description: "Claude Haiku"},
	}

	for _, model := range models {
		err := repo.Create(model)
		require.NoError(t, err)
	}

	// 测试基本分页查询
	result, total, err := repo.FindAll(1, 2, "")
	assert.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, result, 2)

	// 测试搜索功能
	result, total, err = repo.FindAll(1, 10, "claude")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, result, 2)

	// 测试第二页
	result, total, err = repo.FindAll(2, 2, "")
	assert.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, result, 1)
}

func TestRepository_Update(t *testing.T) {
	database := setupTestDB(t)
	repo := NewRepository(database)

	// 创建测试模型
	model := &models.UnifiedModel{
		Name:        "test-model",
		Description: "Test model description",
	}
	err := repo.Create(model)
	require.NoError(t, err)

	// 更新模型
	model.Name = "updated-model"
	model.Description = "Updated description"

	err = repo.Update(model)
	assert.NoError(t, err)

	// 验证更新
	found, err := repo.FindByID(model.ID)
	require.NoError(t, err)
	assert.Equal(t, "updated-model", found.Name)
	assert.Equal(t, "Updated description", found.Description)
}

func TestRepository_Delete(t *testing.T) {
	database := setupTestDB(t)
	repo := NewRepository(database)

	// 创建测试模型
	model := &models.UnifiedModel{
		Name:        "test-model",
		Description: "Test model description",
	}
	err := repo.Create(model)
	require.NoError(t, err)

	// 删除模型
	err = repo.Delete(model.ID)
	assert.NoError(t, err)

	// 验证删除
	_, err = repo.FindByID(model.ID)
	assert.ErrorIs(t, err, ErrModelNotFound)

	// 测试删除不存在的模型
	err = repo.Delete(9999)
	assert.ErrorIs(t, err, ErrModelNotFound)
}

func TestRepository_CheckNameExists(t *testing.T) {
	database := setupTestDB(t)
	repo := NewRepository(database)

	// 创建测试模型
	model := &models.UnifiedModel{
		Name:        "test-model",
		Description: "Test model description",
	}
	err := repo.Create(model)
	require.NoError(t, err)

	// 测试名称存在
	exists, err := repo.CheckNameExists("test-model", 0)
	assert.NoError(t, err)
	assert.True(t, exists)

	// 测试名称不存在
	exists, err = repo.CheckNameExists("non-existent", 0)
	assert.NoError(t, err)
	assert.False(t, exists)

	// 测试排除当前模型
	exists, err = repo.CheckNameExists("test-model", model.ID)
	assert.NoError(t, err)
	assert.False(t, exists)

	// 创建另一个模型
	model2 := &models.UnifiedModel{
		Name:        "test-model-2",
		Description: "Test model 2",
	}
	err = repo.Create(model2)
	require.NoError(t, err)

	// 测试排除其他模型时名称冲突
	exists, err = repo.CheckNameExists("test-model", model2.ID)
	assert.NoError(t, err)
	assert.True(t, exists)
}