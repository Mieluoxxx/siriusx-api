package config

import (
	"fmt"
	"os"
	"time"
)

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Path            string        `mapstructure:"path"`              // 数据库文件路径
	MaxOpenConns    int           `mapstructure:"max_open_conns"`    // 最大连接数
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`    // 最大空闲连接数
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"` // 连接最大生命周期
	AutoMigrate     bool          `mapstructure:"auto_migrate"`      // 是否自动迁移
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port     int    `mapstructure:"port"`
	LogLevel string `mapstructure:"log_level"`
}

// Config 应用配置
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
}

// LoadConfig 加载配置（简化版，暂不依赖 Viper）
func LoadConfig(configPath string) (*Config, error) {
	// 默认配置
	config := &Config{
		Server: ServerConfig{
			Port:     8080,
			LogLevel: "info",
		},
		Database: DatabaseConfig{
			Path:            "./data/siriusx.db",
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: time.Hour,
			AutoMigrate:     true,
		},
	}

	// 支持环境变量覆盖
	if dbPath := os.Getenv("DATABASE_PATH"); dbPath != "" {
		config.Database.Path = dbPath
	}

	if port := os.Getenv("SERVER_PORT"); port != "" {
		var p int
		if _, err := fmt.Sscanf(port, "%d", &p); err == nil {
			config.Server.Port = p
		}
	}

	return config, nil
}
