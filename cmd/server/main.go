package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Mieluoxxx/Siriusx-API/internal/config"
	"github.com/Mieluoxxx/Siriusx-API/internal/db"
)

const (
	// Version 项目版本
	Version = "0.2.0"
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

	fmt.Println("\n🎉 项目启动成功！")
	fmt.Println("📋 当前状态: 数据库已集成")
	fmt.Println("🗄️  数据库: SQLite + GORM")
	fmt.Println("📊 数据表: providers, unified_models, model_mappings, tokens")
	fmt.Println("\n按 Ctrl+C 退出...")

	// 4. 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("\n🛑 正在关闭服务...")

	// 关闭数据库连接
	if err := db.CloseDatabase(database); err != nil {
		log.Printf("⚠️  关闭数据库失败: %v", err)
	}

	log.Println("👋 服务已停止")
}
