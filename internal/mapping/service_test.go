package mapping

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestService(t *testing.T) *Service {
	database := setupTestDB(t)
	repo := NewRepository(database)
	return NewService(repo)
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