package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
)

var (
	// ErrMissingEncryptionKey 缺少加密密钥
	ErrMissingEncryptionKey = errors.New("missing ENCRYPTION_KEY environment variable")
	// ErrInvalidEncryptionKey 加密密钥格式错误
	ErrInvalidEncryptionKey = errors.New("invalid ENCRYPTION_KEY: must be 32 bytes (Base64 encoded)")
)

// LoadEncryptionKey 从环境变量加载加密密钥
func LoadEncryptionKey() ([]byte, error) {
	// 读取环境变量
	keyStr := os.Getenv("ENCRYPTION_KEY")
	if keyStr == "" {
		return nil, ErrMissingEncryptionKey
	}

	// Base64 解码
	key, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ENCRYPTION_KEY: %w", err)
	}

	// 验证密钥长度
	if len(key) != 32 {
		return nil, fmt.Errorf("%w: got %d bytes, expected 32", ErrInvalidEncryptionKey, len(key))
	}

	return key, nil
}

// GenerateEncryptionKey 生成新的加密密钥（用于初始化）
// 返回 Base64 编码的密钥字符串
func GenerateEncryptionKey() (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("failed to generate encryption key: %w", err)
	}

	return base64.StdEncoding.EncodeToString(key), nil
}

// ValidateEncryptionKey 验证加密密钥是否有效
func ValidateEncryptionKey(keyStr string) error {
	if keyStr == "" {
		return ErrMissingEncryptionKey
	}

	key, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		return fmt.Errorf("invalid Base64 encoding: %w", err)
	}

	if len(key) != 32 {
		return fmt.Errorf("%w: got %d bytes, expected 32", ErrInvalidEncryptionKey, len(key))
	}

	return nil
}
