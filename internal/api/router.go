package api

import (
	"github.com/Mieluoxxx/Siriusx-API/internal/api/handlers"
	"github.com/Mieluoxxx/Siriusx-API/internal/mapping"
	"github.com/Mieluoxxx/Siriusx-API/internal/provider"
	"github.com/Mieluoxxx/Siriusx-API/internal/token"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SetupRouter 配置路由
func SetupRouter(db *gorm.DB, encryptionKey []byte) *gin.Engine {
	// 创建 Gin 引擎
	router := gin.Default()

	// 健康检查端点
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "Siriusx-API",
		})
	})

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
		tokens.DELETE("/:id", handler.DeleteToken)
	}
}
