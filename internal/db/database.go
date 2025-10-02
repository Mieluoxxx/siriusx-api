package db

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/Mieluoxxx/Siriusx-API/internal/config"
	"github.com/Mieluoxxx/Siriusx-API/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitDatabase 初始化数据库连接
func InitDatabase(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	// 确保数据目录存在
	dbDir := filepath.Dir(cfg.Path)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}

	// 配置 GORM 日志级别
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	// 连接数据库
	db, err := gorm.Open(sqlite.Open(cfg.Path), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 获取底层 SQL DB 以配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取 SQL DB 失败: %w", err)
	}

	// 配置连接池参数
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	log.Printf("✅ 数据库连接成功: %s", cfg.Path)
	log.Printf("📊 连接池配置: MaxOpen=%d, MaxIdle=%d, Lifetime=%s",
		cfg.MaxOpenConns, cfg.MaxIdleConns, cfg.ConnMaxLifetime)

	return db, nil
}

// AutoMigrate 自动迁移所有数据模型
func AutoMigrate(db *gorm.DB) error {
	log.Println("🔄 开始数据库迁移...")

	// 迁移所有模型
	err := db.AutoMigrate(
		&models.Provider{},
		&models.UnifiedModel{},
		&models.ModelMapping{},
		&models.Token{},
		&models.SystemEvent{},
	)

	if err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}

	log.Println("✅ 数据库迁移完成")
	log.Println("   - providers 表")
	log.Println("   - unified_models 表")
	log.Println("   - model_mappings 表")
	log.Println("   - tokens 表")
	log.Println("   - system_events 表")

	return nil
}

// CloseDatabase 关闭数据库连接
func CloseDatabase(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("获取 SQL DB 失败: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("关闭数据库失败: %w", err)
	}

	log.Println("👋 数据库连接已关闭")
	return nil
}
