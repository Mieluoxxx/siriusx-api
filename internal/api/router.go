package api

import (
	"github.com/Mieluoxxx/Siriusx-API/internal/api/handlers"
	"github.com/Mieluoxxx/Siriusx-API/internal/mapping"
	"github.com/Mieluoxxx/Siriusx-API/internal/provider"
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
	handler := handlers.NewModelHandler(service)

	// 注册路由
	models := group.Group("/models")
	{
		models.POST("", handler.CreateModel)
		models.GET("", handler.ListModels)
		models.GET("/:id", handler.GetModel)
		models.PUT("/:id", handler.UpdateModel)
		models.DELETE("/:id", handler.DeleteModel)
	}
}
