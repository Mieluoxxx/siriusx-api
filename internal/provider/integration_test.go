package provider_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Mieluoxxx/Siriusx-API/internal/api"
	"github.com/Mieluoxxx/Siriusx-API/internal/config"
	"github.com/Mieluoxxx/Siriusx-API/internal/crypto"
	"github.com/Mieluoxxx/Siriusx-API/internal/db"
	"github.com/Mieluoxxx/Siriusx-API/internal/provider"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// Epic3集成测试:供应商管理完整流程
// 包含Story 3.1 (CRUD API) 和 Story 3.2 (加密存储)

type Epic3IntegrationTestSuite struct {
	router        *gin.Engine
	db            *gorm.DB
	encryptionKey []byte
}

func setupEpic3IntegrationTest(t *testing.T) *Epic3IntegrationTestSuite {
	// 生成测试加密密钥 (返回的是 Base64 编码的字符串)
	encryptionKeyStr, err := crypto.GenerateEncryptionKey()
	require.NoError(t, err)

	// 设置环境变量
	os.Setenv("ENCRYPTION_KEY", encryptionKeyStr)
	defer os.Unsetenv("ENCRYPTION_KEY")

	// 解码密钥为 []byte
	encryptionKey, err := base64.StdEncoding.DecodeString(encryptionKeyStr)
	require.NoError(t, err)

	// 创建配置 (使用内存数据库)
	cfg := &config.DatabaseConfig{
		Path:         ":memory:",
		MaxOpenConns: 10,
		MaxIdleConns: 5,
		AutoMigrate:  true,
	}

	// 初始化数据库
	database, err := db.InitDatabase(cfg)
	require.NoError(t, err)

	// 自动迁移
	err = db.AutoMigrate(database)
	require.NoError(t, err)

	// 设置Gin为测试模式
	gin.SetMode(gin.TestMode)

	// 创建路由
	router := api.SetupRouter(database, encryptionKey)

	return &Epic3IntegrationTestSuite{
		router:        router,
		db:            database,
		encryptionKey: encryptionKey,
	}
}

// TestEpic3_IntegrationFlow 测试Epic3的完整集成流程
func TestEpic3_IntegrationFlow(t *testing.T) {
	suite := setupEpic3IntegrationTest(t)

	t.Run("Story 3.1 & 3.2: Complete Provider Lifecycle with Encryption", func(t *testing.T) {
		// === Step 1: 创建供应商 ===
		priority := 80
		createReq := provider.CreateProviderRequest{
			Name:     "OneAPI Provider",
			BaseURL:  "https://api.oneapi.com",
			APIKey:   "sk-test-key-12345",
			Priority: &priority,
		}

		createBody, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/providers", bytes.NewBuffer(createBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var createResp provider.ProviderResponse
		err := json.Unmarshal(w.Body.Bytes(), &createResp)
		require.NoError(t, err)

		providerID := createResp.ID
		assert.Equal(t, "OneAPI Provider", createResp.Name)

		// 验证API Key脱敏
		assert.NotEqual(t, "sk-test-key-12345", createResp.APIKey, "API Key should be masked")
		assert.Contains(t, createResp.APIKey, "****", "API Key should contain mask")

		// === Step 2: 验证数据库中加密存储 ===
		var dbProvider struct {
			ID     uint
			APIKey string
		}
		suite.db.Table("providers").Select("id, api_key").Where("id = ?", providerID).First(&dbProvider)

		// 验证数据库中存储的是加密数据
		assert.NotEqual(t, "sk-test-key-12345", dbProvider.APIKey, "API Key in DB should be encrypted")

		// 验证可以正确解密
		decrypted, err := crypto.DecryptString(dbProvider.APIKey, suite.encryptionKey)
		require.NoError(t, err)
		assert.Equal(t, "sk-test-key-12345", decrypted, "Decrypted API Key should match original")
	})
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}
