package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Mieluoxxx/Siriusx-API/internal/api"
	"github.com/Mieluoxxx/Siriusx-API/internal/config"
	"github.com/Mieluoxxx/Siriusx-API/internal/db"
)

const (
	// Version 项目版本
	Version = "0.3.0"
	// AppName 应用名称
	AppName = "Siriusx-API"
)

func main() {
	log.Printf("=== %s v%s ===\n", AppName, Version)
	log.Println("轻量级 AI 模型聚合网关")

	// 1. 加载配置
	cfg, err := config.LoadConfig("")
	if err != nil {
		log.Fatalf("❌ 加载配置失败: %v", err)
	}
	log.Println("✅ 配置加载成功")

	// 1.1 验证加密密钥（如果启用加密功能）
	if len(cfg.EncryptionKey) > 0 {
		log.Println("🔐 加密功能已启用 (ENCRYPTION_KEY 已配置)")
	} else {
		log.Println("⚠️  加密功能未启用 (未配置 ENCRYPTION_KEY)")
		log.Println("   提示: API Key 将以明文存储，建议在生产环境中启用加密")
	}

	// 2. 初始化数据库
	database, err := db.InitDatabase(&cfg.Database)
	if err != nil {
		log.Fatalf("❌ 数据库初始化失败: %v", err)
	}

	// 3. 自动迁移数据表
	if cfg.Database.AutoMigrate {
		if err := db.AutoMigrate(database); err != nil {
			log.Fatalf("❌ 数据库迁移失败: %v", err)
		}
	}

	// 4. 配置路由
	router := api.SetupRouter(database, cfg.EncryptionKey)
	log.Println("✅ 路由配置成功")

	// 5. 启动 HTTP 服务器
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// 在 goroutine 中启动服务器
	go func() {
		log.Printf("🚀 HTTP 服务器启动在 %s\n", addr)
		fmt.Println("\n🎉 项目启动成功！")
		fmt.Println("📋 当前状态: 供应商 CRUD API 已就绪")
		fmt.Println("🗄️  数据库: SQLite + GORM")
		fmt.Println("📊 数据表: providers, unified_models, model_mappings, tokens")
		fmt.Printf("🌐 API 地址: http://localhost%s\n", addr)
		fmt.Println("📖 API 文档:")
		fmt.Println("   - POST   /api/providers      创建供应商")
		fmt.Println("   - GET    /api/providers      查询供应商列表")
		fmt.Println("   - GET    /api/providers/:id  查询单个供应商")
		fmt.Println("   - PUT    /api/providers/:id  更新供应商")
		fmt.Println("   - DELETE /api/providers/:id  删除供应商")
		fmt.Println("\n按 Ctrl+C 退出...")

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ 服务器启动失败: %v", err)
		}
	}()

	// 6. 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("\n🛑 正在关闭服务...")

	// 关闭 HTTP 服务器（5秒超时）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("⚠️  服务器关闭失败: %v", err)
	}

	// 关闭数据库连接
	if err := db.CloseDatabase(database); err != nil {
		log.Printf("⚠️  关闭数据库失败: %v", err)
	}

	log.Println("👋 服务已停止")
}
