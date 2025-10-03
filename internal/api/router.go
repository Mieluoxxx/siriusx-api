package api

import (
	"github.com/Mieluoxxx/Siriusx-API/internal/api/handlers"
	"github.com/Mieluoxxx/Siriusx-API/internal/api/middleware"
	"github.com/Mieluoxxx/Siriusx-API/internal/mapping"
	"github.com/Mieluoxxx/Siriusx-API/internal/provider"
	"github.com/Mieluoxxx/Siriusx-API/internal/token"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SetupRouter 配置路由
func SetupRouter(db *gorm.DB, encryptionKey []byte) *gin.Engine {
	// 创建 Gin 引擎
	router := gin.Default()

	// 配置 CORS 中间件
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:4321", "http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// 健康检查端点
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "Siriusx-API",
		})
	})

	// OpenAI 兼容的 API 路由
	v1Group := router.Group("/v1")
	{
		setupProxyRoutes(v1Group, db, encryptionKey)
	}

	// API 路由组
	apiGroup := router.Group("/api")
	{
		// 供应商 API
		setupProviderRoutes(apiGroup, db, encryptionKey)

		// 统一模型 API
		setupModelRoutes(apiGroup, db)

		// Token API
		setupTokenRoutes(apiGroup, db)
	}

	return router
}

// setupProxyRoutes 配置代理路由
func setupProxyRoutes(group *gin.RouterGroup, db *gorm.DB, encryptionKey []byte) {
	// 创建依赖
	providerRepo := provider.NewRepository(db)
	var providerService *provider.Service
	if len(encryptionKey) > 0 {
		providerService = provider.NewServiceWithEncryption(providerRepo, encryptionKey)
	} else {
		providerService = provider.NewService(providerRepo)
	}

	mappingRepo := mapping.NewRepository(db)
	mappingService := mapping.NewService(mappingRepo)

	tokenRepo := token.NewRepository(db)
	tokenService := token.NewService(tokenRepo)

	// 创建代理处理器
	proxyHandler := handlers.NewProxyHandler(providerService, mappingService)

	// 注册路由（需要 Token 验证）
	group.POST("/chat/completions",
		middleware.TokenAuthMiddleware(tokenService),
		proxyHandler.ChatCompletions,
	)

	group.POST("/messages",
		middleware.TokenAuthMiddleware(tokenService),
		proxyHandler.Messages,
	)

	group.POST("/messages/count_tokens",
		middleware.TokenAuthMiddleware(tokenService),
		proxyHandler.MessagesCountTokens,
	)
}

// setupProviderRoutes 配置供应商路由
func setupProviderRoutes(group *gin.RouterGroup, db *gorm.DB, encryptionKey []byte) {
	// 创建依赖
	repo := provider.NewRepository(db)

	// 根据是否有加密密钥创建不同的 Service
	var service *provider.Service
	if len(encryptionKey) > 0 {
		service = provider.NewServiceWithEncryption(repo, encryptionKey)
	} else {
		service = provider.NewService(repo)
	}

	handler := handlers.NewProviderHandler(service)

	// 注册路由
	providers := group.Group("/providers")
	{
		providers.POST("", handler.CreateProvider)
		providers.GET("", handler.ListProviders)
		providers.GET("/:id", handler.GetProvider)
		providers.PUT("/:id", handler.UpdateProvider)
		providers.DELETE("/:id", handler.DeleteProvider)

		// 供应商健康检查
		providers.POST("/:id/health-check", handler.HealthCheckProvider)

		// 启用/禁用供应商
		providers.PATCH("/:id/enabled", handler.ToggleProviderEnabled)

		// 获取供应商可用模型
		providers.GET("/:id/models", handler.GetAvailableModels)

		// 测试供应商模型
		providers.POST("/:id/test-model", handler.TestProviderModel)
	}
}

// setupModelRoutes 配置统一模型路由
func setupModelRoutes(group *gin.RouterGroup, db *gorm.DB) {
	// 创建依赖
	repo := mapping.NewRepository(db)
	service := mapping.NewService(repo)
	modelHandler := handlers.NewModelHandler(service)
	mappingHandler := handlers.NewMappingHandler(service)

	// 注册模型路由
	models := group.Group("/models")
	{
		models.POST("", modelHandler.CreateModel)
		models.GET("", modelHandler.ListModels)
		models.GET("/:id", modelHandler.GetModel)
		models.PUT("/:id", modelHandler.UpdateModel)
		models.DELETE("/:id", modelHandler.DeleteModel)

		// 模型映射路由
		models.POST("/:id/mappings", mappingHandler.CreateMapping)
		models.GET("/:id/mappings", mappingHandler.ListMappings)
	}

	// 注册映射路由
	mappings := group.Group("/mappings")
	{
		mappings.GET("/:id", mappingHandler.GetMapping)
		mappings.PUT("/:id", mappingHandler.UpdateMapping)
		mappings.DELETE("/:id", mappingHandler.DeleteMapping)
	}
}

// setupTokenRoutes 配置 Token 路由
func setupTokenRoutes(group *gin.RouterGroup, db *gorm.DB) {
	// 创建依赖
	repo := token.NewRepository(db)
	service := token.NewService(repo)
	handler := handlers.NewTokenHandler(service)

	// 注册路由
	tokens := group.Group("/tokens")
	{
		tokens.POST("", handler.CreateToken)
		tokens.GET("", handler.ListTokens)
		tokens.GET("/:id", handler.GetToken) // 获取单个 Token（包含完整值）
		tokens.DELETE("/:id", handler.DeleteToken)
	}
}
