package mapping

import (
	"testing"

	"github.com/Mieluoxxx/Siriusx-API/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestService(t *testing.T) *Service {
	database := setupTestDB(t)
	repo := NewRepository(database)
	return NewService(repo)
}

func createTestModelAndProviderForService(t *testing.T, service *Service) (*ModelResponse, uint) {
	// 创建测试统一模型
	createReq := CreateModelRequest{
		Name:        "test-model",
		Description: "Test model",
	}
	model, err := service.CreateModel(createReq)
	require.NoError(t, err)

	// 创建测试供应商（直接插入数据库）
	provider := &models.Provider{
		Name:    "test-provider",
		BaseURL: "https://api.test.com",
		APIKey:  "sk-test123",
		Enabled: true,
	}
	err = service.repo.db.Create(provider).Error
	require.NoError(t, err)

	return model, provider.ID
}

func TestService_CreateModel_Success(t *testing.T) {
	service := setupTestService(t)

	req := CreateModelRequest{
		Name:        "claude-sonnet-4",
		Description: "Claude Sonnet 4 model",
	}

	response, err := service.CreateModel(req)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "claude-sonnet-4", response.Name)
	assert.Equal(t, "Claude Sonnet 4 model", response.Description)
	assert.NotZero(t, response.ID)
	assert.NotZero(t, response.CreatedAt)
}

func TestService_CreateModel_WithOptionalFields(t *testing.T) {
	service := setupTestService(t)

	req := CreateModelRequest{
		Name: "gpt-4o",
		// Description 为空，应该允许
	}

	response, err := service.CreateModel(req)
	assert.NoError(t, err)
	assert.Equal(t, "gpt-4o", response.Name)
	assert.Equal(t, "", response.Description)
}

func TestService_CreateModel_EmptyName(t *testing.T) {
	service := setupTestService(t)

	req := CreateModelRequest{
		Name:        "",
		Description: "Test description",
	}

	_, err := service.CreateModel(req)
	assert.ErrorIs(t, err, ErrModelNameEmpty)
}

func TestService_CreateModel_InvalidName(t *testing.T) {
	service := setupTestService(t)

	testCases := []struct {
		name        string
		modelName   string
		expectedErr error
	}{
		{
			name:        "invalid characters",
			modelName:   "model with spaces",
			expectedErr: ErrInvalidModelName,
		},
		{
			name:        "special characters",
			modelName:   "model@special!",
			expectedErr: ErrInvalidModelName,
		},
		{
			name:        "too long",
			modelName:   string(make([]byte, 101)), // 101 字符
			expectedErr: ErrModelNameTooLong,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := CreateModelRequest{
				Name:        tc.modelName,
				Description: "Test description",
			}

			_, err := service.CreateModel(req)
			assert.ErrorIs(t, err, tc.expectedErr)
		})
	}
}

func TestService_CreateModel_ValidNames(t *testing.T) {
	service := setupTestService(t)

	validNames := []string{
		"claude-sonnet-4",
		"gpt_4o",
		"model123",
		"simple",
		"a-very-long-but-valid-model-name-with-hyphens-and-underscores_123",
	}

	for _, name := range validNames {
		t.Run(name, func(t *testing.T) {
			req := CreateModelRequest{
				Name:        name,
				Description: "Test description",
			}

			response, err := service.CreateModel(req)
			assert.NoError(t, err, "Valid name should not produce error: %s", name)
			assert.Equal(t, name, response.Name)
		})
	}
}

func TestService_CreateModel_DescriptionTooLong(t *testing.T) {
	service := setupTestService(t)

	req := CreateModelRequest{
		Name:        "test-model",
		Description: string(make([]byte, 501)), // 501 字符
	}

	_, err := service.CreateModel(req)
	assert.ErrorIs(t, err, ErrDescriptionTooLong)
}

func TestService_CreateModel_DuplicateName(t *testing.T) {
	service := setupTestService(t)

	// 创建第一个模型
	req := CreateModelRequest{
		Name:        "claude-sonnet-4",
		Description: "First model",
	}

	_, err := service.CreateModel(req)
	require.NoError(t, err)

	// 尝试创建同名模型
	req2 := CreateModelRequest{
		Name:        "claude-sonnet-4",
		Description: "Second model",
	}

	_, err = service.CreateModel(req2)
	assert.ErrorIs(t, err, ErrModelNameExists)
}

func TestService_GetModel(t *testing.T) {
	service := setupTestService(t)

	// 创建测试模型
	req := CreateModelRequest{
		Name:        "test-model",
		Description: "Test description",
	}

	created, err := service.CreateModel(req)
	require.NoError(t, err)

	// 测试获取模型
	found, err := service.GetModel(created.ID)
	assert.NoError(t, err)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, created.Name, found.Name)
	assert.Equal(t, created.Description, found.Description)

	// 测试获取不存在的模型
	_, err = service.GetModel(9999)
	assert.ErrorIs(t, err, ErrModelNotFound)
}

func TestService_ListModels(t *testing.T) {
	service := setupTestService(t)

	// 创建测试数据
	models := []CreateModelRequest{
		{Name: "claude-sonnet-4", Description: "Claude Sonnet 4"},
		{Name: "gpt-4o", Description: "GPT-4o"},
		{Name: "claude-haiku", Description: "Claude Haiku"},
	}

	for _, model := range models {
		_, err := service.CreateModel(model)
		require.NoError(t, err)
	}

	// 测试基本列表查询
	req := ListModelsRequest{
		Page:     1,
		PageSize: 10,
	}

	response, err := service.ListModels(req)
	assert.NoError(t, err)
	assert.Len(t, response.Models, 3)
	assert.Equal(t, int64(3), response.Pagination.Total)
	assert.Equal(t, 1, response.Pagination.TotalPages)

	// 测试分页
	req = ListModelsRequest{
		Page:     1,
		PageSize: 2,
	}

	response, err = service.ListModels(req)
	assert.NoError(t, err)
	assert.Len(t, response.Models, 2)
	assert.Equal(t, int64(3), response.Pagination.Total)
	assert.Equal(t, 2, response.Pagination.TotalPages)

	// 测试搜索
	req = ListModelsRequest{
		Page:     1,
		PageSize: 10,
		Search:   "claude",
	}

	response, err = service.ListModels(req)
	assert.NoError(t, err)
	assert.Len(t, response.Models, 2)
	assert.Equal(t, int64(2), response.Pagination.Total)
}

func TestService_ListModels_InvalidParams(t *testing.T) {
	service := setupTestService(t)

	testCases := []struct {
		name     string
		page     int
		pageSize int
		expected struct {
			page     int
			pageSize int
		}
	}{
		{
			name:     "negative page",
			page:     -1,
			pageSize: 10,
			expected: struct {
				page     int
				pageSize int
			}{page: 1, pageSize: 10},
		},
		{
			name:     "zero page",
			page:     0,
			pageSize: 10,
			expected: struct {
				page     int
				pageSize int
			}{page: 1, pageSize: 10},
		},
		{
			name:     "negative page size",
			page:     1,
			pageSize: -10,
			expected: struct {
				page     int
				pageSize int
			}{page: 1, pageSize: 20},
		},
		{
			name:     "zero page size",
			page:     1,
			pageSize: 0,
			expected: struct {
				page     int
				pageSize int
			}{page: 1, pageSize: 20},
		},
		{
			name:     "page size too large",
			page:     1,
			pageSize: 200,
			expected: struct {
				page     int
				pageSize int
			}{page: 1, pageSize: 100},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := ListModelsRequest{
				Page:     tc.page,
				PageSize: tc.pageSize,
			}

			response, err := service.ListModels(req)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected.page, response.Pagination.Page)
			assert.Equal(t, tc.expected.pageSize, response.Pagination.PageSize)
		})
	}
}

func TestService_UpdateModel_Success(t *testing.T) {
	service := setupTestService(t)

	// 创建测试模型
	createReq := CreateModelRequest{
		Name:        "test-model",
		Description: "Original description",
	}

	created, err := service.CreateModel(createReq)
	require.NoError(t, err)

	// 更新模型
	newName := "updated-model"
	newDesc := "Updated description"
	updateReq := UpdateModelRequest{
		Name:        &newName,
		Description: &newDesc,
	}

	updated, err := service.UpdateModel(created.ID, updateReq)
	assert.NoError(t, err)
	assert.Equal(t, "updated-model", updated.Name)
	assert.Equal(t, "Updated description", updated.Description)
	assert.Equal(t, created.ID, updated.ID)
}

func TestService_UpdateModel_PartialUpdate(t *testing.T) {
	service := setupTestService(t)

	// 创建测试模型
	createReq := CreateModelRequest{
		Name:        "test-model",
		Description: "Original description",
	}

	created, err := service.CreateModel(createReq)
	require.NoError(t, err)

	// 只更新名称
	newName := "updated-model"
	updateReq := UpdateModelRequest{
		Name: &newName,
		// Description 不更新
	}

	updated, err := service.UpdateModel(created.ID, updateReq)
	assert.NoError(t, err)
	assert.Equal(t, "updated-model", updated.Name)
	assert.Equal(t, "Original description", updated.Description) // 保持原来的描述
}

func TestService_UpdateModel_NotFound(t *testing.T) {
	service := setupTestService(t)

	newName := "updated-model"
	updateReq := UpdateModelRequest{
		Name: &newName,
	}

	_, err := service.UpdateModel(9999, updateReq)
	assert.ErrorIs(t, err, ErrModelNotFound)
}

func TestService_UpdateModel_DuplicateName(t *testing.T) {
	service := setupTestService(t)

	// 创建两个模型
	_, err := service.CreateModel(CreateModelRequest{Name: "model-1"})
	require.NoError(t, err)

	model2, err := service.CreateModel(CreateModelRequest{Name: "model-2"})
	require.NoError(t, err)

	// 尝试将 model-2 的名称改为 model-1（冲突）
	conflictName := "model-1"
	updateReq := UpdateModelRequest{
		Name: &conflictName,
	}

	_, err = service.UpdateModel(model2.ID, updateReq)
	assert.ErrorIs(t, err, ErrModelNameExists)
}

func TestService_UpdateModel_EmptyName(t *testing.T) {
	service := setupTestService(t)

	// 创建测试模型
	created, err := service.CreateModel(CreateModelRequest{Name: "test-model"})
	require.NoError(t, err)

	// 尝试设置空名称
	emptyName := ""
	updateReq := UpdateModelRequest{
		Name: &emptyName,
	}

	_, err = service.UpdateModel(created.ID, updateReq)
	assert.ErrorIs(t, err, ErrModelNameEmpty)
}

func TestService_DeleteModel(t *testing.T) {
	service := setupTestService(t)

	// 创建测试模型
	created, err := service.CreateModel(CreateModelRequest{Name: "test-model"})
	require.NoError(t, err)

	// 删除模型
	err = service.DeleteModel(created.ID)
	assert.NoError(t, err)

	// 验证模型已删除
	_, err = service.GetModel(created.ID)
	assert.ErrorIs(t, err, ErrModelNotFound)

	// 测试删除不存在的模型
	err = service.DeleteModel(9999)
	assert.ErrorIs(t, err, ErrModelNotFound)
}

// ==================== 映射相关测试 ====================

func TestService_CreateMapping_Success(t *testing.T) {
	service := setupTestService(t)
	model, providerID := createTestModelAndProviderForService(t, service)

	req := CreateMappingRequest{
		UnifiedModelID: model.ID,
		ProviderID:     providerID,
		TargetModel:    "gpt-4o",
		Weight:         70,
		Priority:       1,
		Enabled:        true,
	}

	response, err := service.CreateMapping(req)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, model.ID, response.UnifiedModelID)
	assert.Equal(t, providerID, response.ProviderID)
	assert.Equal(t, "gpt-4o", response.TargetModel)
	assert.Equal(t, 70, response.Weight)
	assert.Equal(t, 1, response.Priority)
	assert.True(t, response.Enabled)
	assert.NotZero(t, response.ID)
	assert.NotZero(t, response.CreatedAt)
}

func TestService_CreateMapping_ValidationErrors(t *testing.T) {
	service := setupTestService(t)
	model, providerID := createTestModelAndProviderForService(t, service)

	testCases := []struct {
		name        string
		req         CreateMappingRequest
		expectedErr error
	}{
		{
			name: "invalid unified model id",
			req: CreateMappingRequest{
				UnifiedModelID: 0,
				ProviderID:     providerID,
				TargetModel:    "gpt-4o",
				Weight:         70,
				Priority:       1,
				Enabled:        true,
			},
			expectedErr: ErrModelNotFound,
		},
		{
			name: "invalid provider id",
			req: CreateMappingRequest{
				UnifiedModelID: model.ID,
				ProviderID:     0,
				TargetModel:    "gpt-4o",
				Weight:         70,
				Priority:       1,
				Enabled:        true,
			},
			expectedErr: ErrProviderNotFound,
		},
		{
			name: "empty target model",
			req: CreateMappingRequest{
				UnifiedModelID: model.ID,
				ProviderID:     providerID,
				TargetModel:    "",
				Weight:         70,
				Priority:       1,
				Enabled:        true,
			},
			expectedErr: ErrTargetModelEmpty,
		},
		{
			name: "invalid weight - too low",
			req: CreateMappingRequest{
				UnifiedModelID: model.ID,
				ProviderID:     providerID,
				TargetModel:    "gpt-4o",
				Weight:         0,
				Priority:       1,
				Enabled:        true,
			},
			expectedErr: ErrInvalidWeight,
		},
		{
			name: "invalid weight - too high",
			req: CreateMappingRequest{
				UnifiedModelID: model.ID,
				ProviderID:     providerID,
				TargetModel:    "gpt-4o",
				Weight:         101,
				Priority:       1,
				Enabled:        true,
			},
			expectedErr: ErrInvalidWeight,
		},
		{
			name: "invalid priority",
			req: CreateMappingRequest{
				UnifiedModelID: model.ID,
				ProviderID:     providerID,
				TargetModel:    "gpt-4o",
				Weight:         70,
				Priority:       0,
				Enabled:        true,
			},
			expectedErr: ErrInvalidPriority,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.CreateMapping(tc.req)
			assert.ErrorIs(t, err, tc.expectedErr)
		})
	}
}

func TestService_CreateMapping_ModelNotFound(t *testing.T) {
	service := setupTestService(t)
	_, providerID := createTestModelAndProviderForService(t, service)

	req := CreateMappingRequest{
		UnifiedModelID: 9999, // 不存在的模型
		ProviderID:     providerID,
		TargetModel:    "gpt-4o",
		Weight:         70,
		Priority:       1,
		Enabled:        true,
	}

	_, err := service.CreateMapping(req)
	assert.ErrorIs(t, err, ErrModelNotFound)
}

func TestService_CreateMapping_DuplicateMapping(t *testing.T) {
	service := setupTestService(t)
	model, providerID := createTestModelAndProviderForService(t, service)

	// 创建第一个映射
	req := CreateMappingRequest{
		UnifiedModelID: model.ID,
		ProviderID:     providerID,
		TargetModel:    "gpt-4o",
		Weight:         70,
		Priority:       1,
		Enabled:        true,
	}

	_, err := service.CreateMapping(req)
	require.NoError(t, err)

	// 尝试创建重复映射
	_, err = service.CreateMapping(req)
	assert.ErrorIs(t, err, ErrMappingExists)
}

func TestService_CreateMapping_DuplicatePriority(t *testing.T) {
	service := setupTestService(t)
	model, providerID := createTestModelAndProviderForService(t, service)

	// 创建第一个映射
	req1 := CreateMappingRequest{
		UnifiedModelID: model.ID,
		ProviderID:     providerID,
		TargetModel:    "gpt-4o",
		Weight:         70,
		Priority:       1,
		Enabled:        true,
	}

	_, err := service.CreateMapping(req1)
	require.NoError(t, err)

	// 尝试创建相同优先级的映射
	req2 := CreateMappingRequest{
		UnifiedModelID: model.ID,
		ProviderID:     providerID,
		TargetModel:    "gpt-4",
		Weight:         30,
		Priority:       1, // 相同优先级
		Enabled:        true,
	}

	_, err = service.CreateMapping(req2)
	assert.ErrorIs(t, err, ErrPriorityExists)
}

func TestService_ListMappings_Success(t *testing.T) {
	service := setupTestService(t)
	model, providerID := createTestModelAndProviderForService(t, service)

	// 创建两个映射
	req1 := CreateMappingRequest{
		UnifiedModelID: model.ID,
		ProviderID:     providerID,
		TargetModel:    "gpt-4o",
		Weight:         70,
		Priority:       1,
		Enabled:        true,
	}
	_, err := service.CreateMapping(req1)
	require.NoError(t, err)

	req2 := CreateMappingRequest{
		UnifiedModelID: model.ID,
		ProviderID:     providerID,
		TargetModel:    "gpt-4",
		Weight:         30,
		Priority:       2,
		Enabled:        true,
	}
	_, err = service.CreateMapping(req2)
	require.NoError(t, err)

	// 测试列表查询
	response, err := service.ListMappings(model.ID, true)
	assert.NoError(t, err)
	assert.Len(t, response.Mappings, 2)
	assert.Equal(t, int64(2), response.Total)

	// 验证按优先级排序
	assert.Equal(t, 1, response.Mappings[0].Priority)
	assert.Equal(t, 2, response.Mappings[1].Priority)

	// 验证包含供应商信息
	assert.NotNil(t, response.Mappings[0].Provider)
	assert.Equal(t, "test-provider", response.Mappings[0].Provider.Name)
}

func TestService_ListMappings_ModelNotFound(t *testing.T) {
	service := setupTestService(t)

	_, err := service.ListMappings(9999, false)
	assert.ErrorIs(t, err, ErrModelNotFound)
}

func TestService_GetMapping_Success(t *testing.T) {
	service := setupTestService(t)
	model, providerID := createTestModelAndProviderForService(t, service)

	// 创建映射
	req := CreateMappingRequest{
		UnifiedModelID: model.ID,
		ProviderID:     providerID,
		TargetModel:    "gpt-4o",
		Weight:         70,
		Priority:       1,
		Enabled:        true,
	}

	created, err := service.CreateMapping(req)
	require.NoError(t, err)

	// 测试获取映射
	found, err := service.GetMapping(created.ID)
	assert.NoError(t, err)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, created.TargetModel, found.TargetModel)
	assert.Equal(t, created.Weight, found.Weight)
}

func TestService_GetMapping_NotFound(t *testing.T) {
	service := setupTestService(t)

	_, err := service.GetMapping(9999)
	assert.ErrorIs(t, err, ErrMappingNotFound)
}

func TestService_UpdateMapping_Success(t *testing.T) {
	service := setupTestService(t)
	model, providerID := createTestModelAndProviderForService(t, service)

	// 创建映射
	createReq := CreateMappingRequest{
		UnifiedModelID: model.ID,
		ProviderID:     providerID,
		TargetModel:    "gpt-4o",
		Weight:         70,
		Priority:       1,
		Enabled:        true,
	}

	created, err := service.CreateMapping(createReq)
	require.NoError(t, err)

	// 更新映射
	newWeight := 80
	newTargetModel := "gpt-4o-latest"
	enabled := false
	updateReq := UpdateMappingRequest{
		TargetModel: &newTargetModel,
		Weight:      &newWeight,
		Enabled:     &enabled,
	}

	updated, err := service.UpdateMapping(created.ID, updateReq)
	assert.NoError(t, err)
	assert.Equal(t, created.ID, updated.ID)
	assert.Equal(t, "gpt-4o-latest", updated.TargetModel)
	assert.Equal(t, 80, updated.Weight)
	assert.False(t, updated.Enabled)
	assert.Equal(t, 1, updated.Priority) // 优先级未更新
}

func TestService_UpdateMapping_PriorityConflict(t *testing.T) {
	service := setupTestService(t)
	model, providerID := createTestModelAndProviderForService(t, service)

	// 创建两个映射
	req1 := CreateMappingRequest{
		UnifiedModelID: model.ID,
		ProviderID:     providerID,
		TargetModel:    "gpt-4o",
		Weight:         70,
		Priority:       1,
		Enabled:        true,
	}
	_, err := service.CreateMapping(req1)
	require.NoError(t, err)

	req2 := CreateMappingRequest{
		UnifiedModelID: model.ID,
		ProviderID:     providerID,
		TargetModel:    "gpt-4",
		Weight:         30,
		Priority:       2,
		Enabled:        true,
	}
	mapping2, err := service.CreateMapping(req2)
	require.NoError(t, err)

	// 尝试将第二个映射的优先级改为1（冲突）
	conflictPriority := 1
	updateReq := UpdateMappingRequest{
		Priority: &conflictPriority,
	}

	_, err = service.UpdateMapping(mapping2.ID, updateReq)
	assert.ErrorIs(t, err, ErrPriorityExists)
}

func TestService_UpdateMapping_NotFound(t *testing.T) {
	service := setupTestService(t)

	newWeight := 80
	updateReq := UpdateMappingRequest{
		Weight: &newWeight,
	}

	_, err := service.UpdateMapping(9999, updateReq)
	assert.ErrorIs(t, err, ErrMappingNotFound)
}

func TestService_UpdateMapping_InvalidValues(t *testing.T) {
	service := setupTestService(t)
	model, providerID := createTestModelAndProviderForService(t, service)

	// 创建映射
	createReq := CreateMappingRequest{
		UnifiedModelID: model.ID,
		ProviderID:     providerID,
		TargetModel:    "gpt-4o",
		Weight:         70,
		Priority:       1,
		Enabled:        true,
	}

	created, err := service.CreateMapping(createReq)
	require.NoError(t, err)

	// 测试无效的权重
	invalidWeight := 0
	updateReq := UpdateMappingRequest{
		Weight: &invalidWeight,
	}

	_, err = service.UpdateMapping(created.ID, updateReq)
	assert.ErrorIs(t, err, ErrInvalidWeight)

	// 测试无效的优先级
	invalidPriority := 0
	updateReq = UpdateMappingRequest{
		Priority: &invalidPriority,
	}

	_, err = service.UpdateMapping(created.ID, updateReq)
	assert.ErrorIs(t, err, ErrInvalidPriority)

	// 测试空的目标模型
	emptyTargetModel := ""
	updateReq = UpdateMappingRequest{
		TargetModel: &emptyTargetModel,
	}

	_, err = service.UpdateMapping(created.ID, updateReq)
	assert.ErrorIs(t, err, ErrTargetModelEmpty)
}

func TestService_DeleteMapping_Success(t *testing.T) {
	service := setupTestService(t)
	model, providerID := createTestModelAndProviderForService(t, service)

	// 创建映射
	req := CreateMappingRequest{
		UnifiedModelID: model.ID,
		ProviderID:     providerID,
		TargetModel:    "gpt-4o",
		Weight:         70,
		Priority:       1,
		Enabled:        true,
	}

	created, err := service.CreateMapping(req)
	require.NoError(t, err)

	// 删除映射
	err = service.DeleteMapping(created.ID)
	assert.NoError(t, err)

	// 验证映射已删除
	_, err = service.GetMapping(created.ID)
	assert.ErrorIs(t, err, ErrMappingNotFound)
}

func TestService_DeleteMapping_NotFound(t *testing.T) {
	service := setupTestService(t)

	err := service.DeleteMapping(9999)
	assert.ErrorIs(t, err, ErrMappingNotFound)
}