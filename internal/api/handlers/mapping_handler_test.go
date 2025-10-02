package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Mieluoxxx/Siriusx-API/internal/mapping"
	"github.com/Mieluoxxx/Siriusx-API/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupMappingTestHandler(t *testing.T) (*MappingHandler, *gin.Engine, *models.UnifiedModel, *models.Provider) {
	// 直接创建内存数据库
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// 手动迁移所有需要的模型
	err = database.AutoMigrate(&models.UnifiedModel{}, &models.Provider{}, &models.ModelMapping{})
	require.NoError(t, err)

	// 创建依赖
	repo := mapping.NewRepository(database)
	service := mapping.NewService(repo)
	handler := NewMappingHandler(service)

	// 设置 Gin 路由
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 设置路由
	api := router.Group("/api")
	{
		models := api.Group("/models")
		{
			models.POST("/:id/mappings", handler.CreateMapping)
			models.GET("/:id/mappings", handler.ListMappings)
		}

		mappings := api.Group("/mappings")
		{
			mappings.GET("/:id", handler.GetMapping)
			mappings.PUT("/:id", handler.UpdateMapping)
			mappings.DELETE("/:id", handler.DeleteMapping)
		}
	}

	// 创建测试统一模型
	testModel := &models.UnifiedModel{
		Name:        "test-model",
		Description: "Test model",
	}
	err = repo.Create(testModel)
	require.NoError(t, err)

	// 创建测试供应商
	testProvider := &models.Provider{
		Name:    "test-provider",
		BaseURL: "https://api.test.com",
		APIKey:  "sk-test123",
		Enabled: true,
	}
	err = database.Create(testProvider).Error
	require.NoError(t, err)

	return handler, router, testModel, testProvider
}

func TestMappingHandler_CreateMapping_Success(t *testing.T) {
	_, router, model, provider := setupMappingTestHandler(t)

	reqBody := mapping.CreateMappingRequest{
		ProviderID:  provider.ID,
		TargetModel: "gpt-4o",
		Weight:      70,
		Priority:    1,
		Enabled:     true,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/models/%d/mappings", model.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response mapping.MappingResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, model.ID, response.UnifiedModelID)
	assert.Equal(t, provider.ID, response.ProviderID)
	assert.Equal(t, "gpt-4o", response.TargetModel)
	assert.Equal(t, 70, response.Weight)
	assert.Equal(t, 1, response.Priority)
	assert.True(t, response.Enabled)
	assert.NotZero(t, response.ID)
}

func TestMappingHandler_CreateMapping_ValidationError(t *testing.T) {
	_, router, model, _ := setupMappingTestHandler(t)

	reqBody := mapping.CreateMappingRequest{
		ProviderID:  0, // 无效的供应商ID
		TargetModel: "gpt-4o",
		Weight:      70,
		Priority:    1,
		Enabled:     true,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/models/%d/mappings", model.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response.Error, "ProviderID")
}

func TestMappingHandler_CreateMapping_InvalidJSON(t *testing.T) {
	_, router, model, _ := setupMappingTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/models/%d/mappings", model.ID), bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMappingHandler_CreateMapping_InvalidModelID(t *testing.T) {
	_, router, _, provider := setupMappingTestHandler(t)

	reqBody := mapping.CreateMappingRequest{
		ProviderID:  provider.ID,
		TargetModel: "gpt-4o",
		Weight:      70,
		Priority:    1,
		Enabled:     true,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/models/invalid/mappings", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response.Error, "无效的模型ID")
}

func TestMappingHandler_CreateMapping_DuplicateMapping(t *testing.T) {
	_, router, model, provider := setupMappingTestHandler(t)

	reqBody := mapping.CreateMappingRequest{
		ProviderID:  provider.ID,
		TargetModel: "gpt-4o",
		Weight:      70,
		Priority:    1,
		Enabled:     true,
	}

	body, _ := json.Marshal(reqBody)

	// 创建第一个映射
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/models/%d/mappings", model.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	// 尝试创建重复映射
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/models/%d/mappings", model.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response.Error, "mapping already exists")
}

func TestMappingHandler_ListMappings_Success(t *testing.T) {
	_, router, model, provider := setupMappingTestHandler(t)

	// 创建测试映射
	mappings := []mapping.CreateMappingRequest{
		{ProviderID: provider.ID, TargetModel: "gpt-4o", Weight: 70, Priority: 1, Enabled: true},
		{ProviderID: provider.ID, TargetModel: "gpt-4", Weight: 30, Priority: 2, Enabled: true},
	}

	for _, mapping := range mappings {
		body, _ := json.Marshal(mapping)
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/models/%d/mappings", model.ID), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)
	}

	// 测试列表查询
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/models/%d/mappings", model.ID), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response mapping.ListMappingsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Len(t, response.Mappings, 2)
	assert.Equal(t, int64(2), response.Total)
}

func TestMappingHandler_ListMappings_WithProvider(t *testing.T) {
	_, router, model, provider := setupMappingTestHandler(t)

	// 创建测试映射
	reqBody := mapping.CreateMappingRequest{
		ProviderID:  provider.ID,
		TargetModel: "gpt-4o",
		Weight:      70,
		Priority:    1,
		Enabled:     true,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/models/%d/mappings", model.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	// 测试包含供应商信息的查询
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/models/%d/mappings?include_provider=true", model.ID), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response mapping.ListMappingsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Len(t, response.Mappings, 1)
	assert.NotNil(t, response.Mappings[0].Provider)
	assert.Equal(t, "test-provider", response.Mappings[0].Provider.Name)
}

func TestMappingHandler_ListMappings_InvalidModelID(t *testing.T) {
	_, router, _, _ := setupMappingTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/models/invalid/mappings", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response.Error, "无效的模型ID")
}

func TestMappingHandler_ListMappings_ModelNotFound(t *testing.T) {
	_, router, _, _ := setupMappingTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/models/9999/mappings", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response.Error, "model not found")
}

func TestMappingHandler_GetMapping_Success(t *testing.T) {
	_, router, model, provider := setupMappingTestHandler(t)

	// 创建测试映射
	reqBody := mapping.CreateMappingRequest{
		ProviderID:  provider.ID,
		TargetModel: "gpt-4o",
		Weight:      70,
		Priority:    1,
		Enabled:     true,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/models/%d/mappings", model.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var created mapping.MappingResponse
	err := json.Unmarshal(w.Body.Bytes(), &created)
	require.NoError(t, err)

	// 测试获取映射
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/mappings/%d", created.ID), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response mapping.MappingResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, created.ID, response.ID)
	assert.Equal(t, "gpt-4o", response.TargetModel)
	assert.Equal(t, 70, response.Weight)
}

func TestMappingHandler_GetMapping_NotFound(t *testing.T) {
	_, router, _, _ := setupMappingTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/mappings/9999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response.Error, "mapping not found")
}

func TestMappingHandler_GetMapping_InvalidID(t *testing.T) {
	_, router, _, _ := setupMappingTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/mappings/invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response.Error, "无效的映射ID")
}

func TestMappingHandler_UpdateMapping_Success(t *testing.T) {
	_, router, model, provider := setupMappingTestHandler(t)

	// 创建测试映射
	createReq := mapping.CreateMappingRequest{
		ProviderID:  provider.ID,
		TargetModel: "gpt-4o",
		Weight:      70,
		Priority:    1,
		Enabled:     true,
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/models/%d/mappings", model.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var created mapping.MappingResponse
	err := json.Unmarshal(w.Body.Bytes(), &created)
	require.NoError(t, err)

	// 更新映射
	newWeight := 80
	newTargetModel := "gpt-4o-latest"
	enabled := false
	updateReq := mapping.UpdateMappingRequest{
		TargetModel: &newTargetModel,
		Weight:      &newWeight,
		Enabled:     &enabled,
	}

	body, _ = json.Marshal(updateReq)
	req = httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/mappings/%d", created.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response mapping.MappingResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, created.ID, response.ID)
	assert.Equal(t, "gpt-4o-latest", response.TargetModel)
	assert.Equal(t, 80, response.Weight)
	assert.False(t, response.Enabled)
}

func TestMappingHandler_UpdateMapping_NotFound(t *testing.T) {
	_, router, _, _ := setupMappingTestHandler(t)

	newWeight := 80
	updateReq := mapping.UpdateMappingRequest{
		Weight: &newWeight,
	}

	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPut, "/api/mappings/9999", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestMappingHandler_DeleteMapping_Success(t *testing.T) {
	_, router, model, provider := setupMappingTestHandler(t)

	// 创建测试映射
	createReq := mapping.CreateMappingRequest{
		ProviderID:  provider.ID,
		TargetModel: "gpt-4o",
		Weight:      70,
		Priority:    1,
		Enabled:     true,
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/models/%d/mappings", model.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var created mapping.MappingResponse
	err := json.Unmarshal(w.Body.Bytes(), &created)
	require.NoError(t, err)

	// 删除映射
	req = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/mappings/%d", created.ID), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	// 验证映射已删除
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/mappings/%d", created.ID), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestMappingHandler_DeleteMapping_NotFound(t *testing.T) {
	_, router, _, _ := setupMappingTestHandler(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/mappings/9999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}