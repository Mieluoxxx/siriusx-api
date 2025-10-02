package mapping

import (
	"context"
	"testing"
	"time"

	"github.com/Mieluoxxx/Siriusx-API/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestingT 通用测试接口
type TestingT interface {
	Errorf(format string, args ...interface{})
	FailNow()
	Helper()
}

func setupTestRouter(t TestingT) (*DefaultRouter, *Repository) {
	// 创建测试数据库
	database := setupTestDB(&testing.T{})
	repo := NewRepository(database)

	// 创建路由器
	config := DefaultRouterConfig()
	config.Cache.TTL = 100 * time.Millisecond // 短TTL用于测试
	router := NewRouter(repo, config)

	return router, repo
}

// 为基准测试创建单独的设置函数
func setupTestRouterForBench(b *testing.B) (*DefaultRouter, *Repository) {
	// 创建测试数据库 - 直接调用不依赖类型转换
	database := setupTestDB(b)
	repo := NewRepository(database)

	// 创建路由器
	config := DefaultRouterConfig()
	config.Cache.TTL = 100 * time.Millisecond // 短TTL用于测试
	router := NewRouter(repo, config)

	return router, repo
}

func createTestModelAndProvidersForRouter(t TestingT, repo *Repository) (*models.UnifiedModel, []*models.Provider) {
	// 创建测试统一模型
	model := &models.UnifiedModel{
		Name:        "claude-sonnet-4",
		Description: "Claude Sonnet 4",
	}
	err := repo.Create(model)
	if err != nil {
		t.Errorf("Failed to create test model: %v", err)
		t.FailNow()
	}

	// 创建测试供应商
	providers := []*models.Provider{
		{
			Name:         "provider-1",
			BaseURL:      "https://api.provider1.com",
			APIKey:       "key1",
			Enabled:      true,
			HealthStatus: "healthy",
		},
		{
			Name:         "provider-2",
			BaseURL:      "https://api.provider2.com",
			APIKey:       "key2",
			Enabled:      true,
			HealthStatus: "healthy",
		},
		{
			Name:         "provider-3",
			BaseURL:      "https://api.provider3.com",
			APIKey:       "key3",
			Enabled:      false, // 禁用的供应商
			HealthStatus: "healthy",
		},
	}

	for _, provider := range providers {
		err := repo.db.Create(provider).Error
		if err != nil {
			t.Errorf("Failed to create test provider: %v", err)
			t.FailNow()
		}
	}

	return model, providers
}

func createTestMappings(t TestingT, repo *Repository, model *models.UnifiedModel, providers []*models.Provider) {
	mappings := []*models.ModelMapping{
		{
			UnifiedModelID: model.ID,
			ProviderID:     providers[0].ID,
			TargetModel:    "gpt-4o-provider1",
			Weight:         70,
			Priority:       1,
			Enabled:        true,
		},
		{
			UnifiedModelID: model.ID,
			ProviderID:     providers[1].ID,
			TargetModel:    "gpt-4o-provider2",
			Weight:         30,
			Priority:       2,
			Enabled:        true,
		},
		{
			UnifiedModelID: model.ID,
			ProviderID:     providers[2].ID,
			TargetModel:    "gpt-4o-provider3",
			Weight:         50,
			Priority:       3,
			Enabled:        false, // 禁用的映射
		},
	}

	for _, mapping := range mappings {
		err := repo.CreateMapping(mapping)
		if err != nil {
			t.Errorf("Failed to create test mapping: %v", err)
			t.FailNow()
		}
	}
}

func TestRouter_ResolveModel_Success(t *testing.T) {
	router, repo := setupTestRouter(t)
	defer router.Close()

	model, providers := createTestModelAndProvidersForRouter(t, repo)
	createTestMappings(t, repo, model, providers)

	// 解析模型
	ctx := context.Background()
	mappings, err := router.ResolveModel(ctx, "claude-sonnet-4")

	assert.NoError(t, err)
	assert.Len(t, mappings, 2) // 只有2个启用的映射

	// 验证排序（按优先级升序）
	assert.Equal(t, 1, mappings[0].Priority)
	assert.Equal(t, 2, mappings[1].Priority)

	// 验证供应商信息
	assert.NotNil(t, mappings[0].Provider)
	assert.Equal(t, "provider-1", mappings[0].Provider.Name)
	assert.Equal(t, "gpt-4o-provider1", mappings[0].TargetModel)

	// 验证权重
	assert.Equal(t, 70, mappings[0].Weight)
	assert.Equal(t, 30, mappings[1].Weight)
}

func TestRouter_ResolveModel_Cache(t *testing.T) {
	router, repo := setupTestRouter(t)
	defer router.Close()

	model, providers := createTestModelAndProvidersForRouter(t, repo)
	createTestMappings(t, repo, model, providers)

	ctx := context.Background()

	// 第一次解析
	start := time.Now()
	mappings1, err := router.ResolveModel(ctx, "claude-sonnet-4")
	duration1 := time.Since(start)
	assert.NoError(t, err)
	assert.Len(t, mappings1, 2)

	// 第二次解析（应该命中缓存）
	start = time.Now()
	mappings2, err := router.ResolveModel(ctx, "claude-sonnet-4")
	duration2 := time.Since(start)
	assert.NoError(t, err)
	assert.Len(t, mappings2, 2)

	// 缓存命中应该更快
	assert.Less(t, duration2, duration1)

	// 验证数据一致性
	assert.Equal(t, mappings1[0].ID, mappings2[0].ID)
	assert.Equal(t, mappings1[0].TargetModel, mappings2[0].TargetModel)

	// 验证缓存统计
	stats := router.GetCacheStats()
	assert.Equal(t, int64(1), stats.HitCount)
	assert.Equal(t, int64(1), stats.MissCount)
	assert.InDelta(t, 0.5, stats.HitRate, 0.01)
}

func TestRouter_ResolveModel_ModelNotFound(t *testing.T) {
	router, _ := setupTestRouter(t)
	defer router.Close()

	ctx := context.Background()
	mappings, err := router.ResolveModel(ctx, "non-existent-model")

	assert.Error(t, err)
	assert.Nil(t, mappings)

	// 验证错误类型
	routerErr, ok := err.(*RouterError)
	assert.True(t, ok)
	assert.Equal(t, "MODEL_NOT_FOUND", routerErr.Code)
	assert.Contains(t, routerErr.Message, "non-existent-model")
}

func TestRouter_ResolveModel_NoAvailableProviders(t *testing.T) {
	router, repo := setupTestRouter(t)
	defer router.Close()

	// 创建模型但不创建映射
	model := &models.UnifiedModel{
		Name:        "empty-model",
		Description: "Model without mappings",
	}
	err := repo.Create(model)
	require.NoError(t, err)

	ctx := context.Background()
	mappings, err := router.ResolveModel(ctx, "empty-model")

	assert.Error(t, err)
	assert.Nil(t, mappings)

	// 验证错误类型
	routerErr, ok := err.(*RouterError)
	assert.True(t, ok)
	assert.Equal(t, "NO_AVAILABLE_PROVIDERS", routerErr.Code)
}

func TestRouter_ResolveModel_EmptyModelName(t *testing.T) {
	router, _ := setupTestRouter(t)
	defer router.Close()

	ctx := context.Background()

	testCases := []string{"", "   ", "\t", "\n"}
	for _, modelName := range testCases {
		mappings, err := router.ResolveModel(ctx, modelName)
		assert.Error(t, err)
		assert.Nil(t, mappings)

		routerErr, ok := err.(*RouterError)
		assert.True(t, ok)
		assert.Equal(t, "MODEL_NOT_FOUND", routerErr.Code)
	}
}

func TestRouter_ResolveModel_HealthyProvidersOnly(t *testing.T) {
	router, repo := setupTestRouter(t)
	defer router.Close()

	model, providers := createTestModelAndProvidersForRouter(t, repo)

	// 设置一个供应商为不健康
	providers[1].HealthStatus = "unhealthy"
	err := repo.db.Save(providers[1]).Error
	require.NoError(t, err)

	// 创建映射
	mappings := []*models.ModelMapping{
		{
			UnifiedModelID: model.ID,
			ProviderID:     providers[0].ID, // 健康供应商
			TargetModel:    "gpt-4o-healthy",
			Weight:         70,
			Priority:       1,
			Enabled:        true,
		},
		{
			UnifiedModelID: model.ID,
			ProviderID:     providers[1].ID, // 不健康供应商
			TargetModel:    "gpt-4o-unhealthy",
			Weight:         30,
			Priority:       2,
			Enabled:        true,
		},
	}

	for _, mapping := range mappings {
		err := repo.CreateMapping(mapping)
		require.NoError(t, err)
	}

	ctx := context.Background()
	result, err := router.ResolveModel(ctx, "claude-sonnet-4")

	assert.NoError(t, err)
	assert.Len(t, result, 1) // 只有健康的供应商
	assert.Equal(t, "gpt-4o-healthy", result[0].TargetModel)
	assert.Equal(t, "healthy", result[0].Provider.HealthStatus)
}

func TestRouter_ResolveModel_Priority(t *testing.T) {
	router, repo := setupTestRouter(t)
	defer router.Close()

	model, providers := createTestModelAndProvidersForRouter(t, repo)

	// 创建不同优先级的映射
	mappings := []*models.ModelMapping{
		{
			UnifiedModelID: model.ID,
			ProviderID:     providers[0].ID,
			TargetModel:    "model-priority-3",
			Weight:         50,
			Priority:       3, // 低优先级
			Enabled:        true,
		},
		{
			UnifiedModelID: model.ID,
			ProviderID:     providers[1].ID,
			TargetModel:    "model-priority-1",
			Weight:         50,
			Priority:       1, // 高优先级
			Enabled:        true,
		},
	}

	for _, mapping := range mappings {
		err := repo.CreateMapping(mapping)
		require.NoError(t, err)
	}

	ctx := context.Background()
	result, err := router.ResolveModel(ctx, "claude-sonnet-4")

	assert.NoError(t, err)
	assert.Len(t, result, 2)

	// 验证按优先级排序（优先级1应该在前面）
	assert.Equal(t, 1, result[0].Priority)
	assert.Equal(t, 3, result[1].Priority)
	assert.Equal(t, "model-priority-1", result[0].TargetModel)
	assert.Equal(t, "model-priority-3", result[1].TargetModel)
}

func TestRouter_SelectProvider(t *testing.T) {
	router, _ := setupTestRouter(t)
	defer router.Close()

	// 创建测试映射
	mappings := []*ResolvedMapping{
		{
			ID:          1,
			TargetModel: "model-1",
			Weight:      70,
			Priority:    1,
			Enabled:     true,
		},
		{
			ID:          2,
			TargetModel: "model-2",
			Weight:      30,
			Priority:    1,
			Enabled:     true,
		},
	}

	// 测试空列表
	selected := router.SelectProvider(nil)
	assert.Nil(t, selected)

	selected = router.SelectProvider([]*ResolvedMapping{})
	assert.Nil(t, selected)

	// 测试单个映射
	selected = router.SelectProvider(mappings[:1])
	assert.NotNil(t, selected)
	assert.Equal(t, uint(1), selected.ID)

	// 测试加权选择
	selected = router.SelectProvider(mappings)
	assert.NotNil(t, selected)
	assert.Contains(t, []uint{1, 2}, selected.ID)
}

func TestRouter_InvalidateCache(t *testing.T) {
	router, repo := setupTestRouter(t)
	defer router.Close()

	model, providers := createTestModelAndProvidersForRouter(t, repo)
	createTestMappings(t, repo, model, providers)

	ctx := context.Background()

	// 第一次解析，存入缓存
	_, err := router.ResolveModel(ctx, "claude-sonnet-4")
	assert.NoError(t, err)

	// 验证缓存存在
	stats := router.GetCacheStats()
	assert.Equal(t, 1, stats.Size)

	// 清理缓存
	router.InvalidateCache("claude-sonnet-4")

	// 验证缓存被清理
	stats = router.GetCacheStats()
	assert.Equal(t, 0, stats.Size)
}

func TestRouter_ClearCache(t *testing.T) {
	router, repo := setupTestRouter(t)
	defer router.Close()

	model, providers := createTestModelAndProvidersForRouter(t, repo)
	createTestMappings(t, repo, model, providers)

	ctx := context.Background()

	// 解析多个模型，存入缓存
	_, err := router.ResolveModel(ctx, "claude-sonnet-4")
	assert.NoError(t, err)

	// 验证缓存存在
	stats := router.GetCacheStats()
	assert.GreaterOrEqual(t, stats.Size, 1)

	// 清空所有缓存
	router.ClearCache()

	// 验证缓存被清空
	stats = router.GetCacheStats()
	assert.Equal(t, 0, stats.Size)
	assert.Equal(t, int64(0), stats.HitCount)
	assert.Equal(t, int64(0), stats.MissCount)
}

func TestRouter_ResolveBatch(t *testing.T) {
	router, repo := setupTestRouter(t)
	defer router.Close()

	model, providers := createTestModelAndProvidersForRouter(t, repo)
	createTestMappings(t, repo, model, providers)

	// 创建另一个模型
	model2 := &models.UnifiedModel{
		Name:        "claude-haiku",
		Description: "Claude Haiku",
	}
	err := repo.Create(model2)
	require.NoError(t, err)

	// 为第二个模型创建映射
	mapping := &models.ModelMapping{
		UnifiedModelID: model2.ID,
		ProviderID:     providers[0].ID,
		TargetModel:    "haiku-provider1",
		Weight:         100,
		Priority:       1,
		Enabled:        true,
	}
	err = repo.CreateMapping(mapping)
	require.NoError(t, err)

	ctx := context.Background()
	modelNames := []string{"claude-sonnet-4", "claude-haiku"}

	// 批量解析
	results, err := router.ResolveBatch(ctx, modelNames)
	assert.NoError(t, err)
	assert.Len(t, results, 2)

	// 验证结果
	assert.Contains(t, results, "claude-sonnet-4")
	assert.Contains(t, results, "claude-haiku")
	assert.Len(t, results["claude-sonnet-4"], 2)
	assert.Len(t, results["claude-haiku"], 1)
	assert.Equal(t, "haiku-provider1", results["claude-haiku"][0].TargetModel)
}

func TestRouter_ResolveBatch_WithErrors(t *testing.T) {
	router, repo := setupTestRouter(t)
	defer router.Close()

	model, providers := createTestModelAndProvidersForRouter(t, repo)
	createTestMappings(t, repo, model, providers)

	ctx := context.Background()
	modelNames := []string{"claude-sonnet-4", "non-existent-model"}

	// 批量解析（包含不存在的模型）
	results, err := router.ResolveBatch(ctx, modelNames)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "non-existent-model")

	// 即使有错误，成功的结果应该被返回
	assert.Contains(t, results, "claude-sonnet-4")
	assert.Len(t, results["claude-sonnet-4"], 2)
}

func TestRouterFactory(t *testing.T) {
	factory := NewRouterFactory()
	defer factory.CloseAll()

	database := setupTestDB(t)
	repo := NewRepository(database)
	config := DefaultRouterConfig()

	// 获取路由器
	router1 := factory.GetRouter("test", repo, config)
	assert.NotNil(t, router1)

	// 再次获取相同名称的路由器应该返回相同实例
	router2 := factory.GetRouter("test", repo, config)
	assert.Same(t, router1, router2)

	// 获取不同名称的路由器应该返回不同实例
	router3 := factory.GetRouter("test2", repo, config)
	assert.NotSame(t, router1, router3)

	// 关闭所有路由器
	err := factory.CloseAll()
	assert.NoError(t, err)
}

func TestDefaultRouterConfig(t *testing.T) {
	config := DefaultRouterConfig()

	assert.NotNil(t, config.Cache)
	assert.True(t, config.HealthCheck)
	assert.True(t, config.EnableWeight)
	assert.Equal(t, 5*time.Minute, config.Cache.TTL)
	assert.Equal(t, 1000, config.Cache.MaxSize)
}

func BenchmarkRouter_ResolveModel_WithCache(b *testing.B) {
	router, repo := setupTestRouterForBench(b)
	defer router.Close()

	model, providers := createTestModelAndProvidersForRouter(b, repo)
	createTestMappings(b, repo, model, providers)

	ctx := context.Background()

	// 预热缓存
	_, err := router.ResolveModel(ctx, "claude-sonnet-4")
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := router.ResolveModel(ctx, "claude-sonnet-4")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRouter_ResolveModel_WithoutCache(b *testing.B) {
	router, repo := setupTestRouterForBench(b)
	defer router.Close()

	model, providers := createTestModelAndProvidersForRouter(b, repo)
	createTestMappings(b, repo, model, providers)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 每次清理缓存确保不命中
		router.ClearCache()
		_, err := router.ResolveModel(ctx, "claude-sonnet-4")
		if err != nil {
			b.Fatal(err)
		}
	}
}