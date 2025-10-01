package crypto

import (
	"encoding/base64"
	"os"
	"testing"
)

// TestLoadEncryptionKey_Success 测试成功加载加密密钥
func TestLoadEncryptionKey_Success(t *testing.T) {
	// 生成测试密钥
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	keyStr := base64.StdEncoding.EncodeToString(key)

	// 设置环境变量
	os.Setenv("ENCRYPTION_KEY", keyStr)
	defer os.Unsetenv("ENCRYPTION_KEY")

	// 加载密钥
	loaded, err := LoadEncryptionKey()
	if err != nil {
		t.Fatalf("LoadEncryptionKey() failed: %v", err)
	}

	if len(loaded) != 32 {
		t.Errorf("LoadEncryptionKey() returned %d bytes, want 32", len(loaded))
	}

	for i, b := range loaded {
		if b != byte(i) {
			t.Errorf("LoadEncryptionKey() byte %d = %v, want %v", i, b, byte(i))
		}
	}
}

// TestLoadEncryptionKey_Missing 测试缺少环境变量
func TestLoadEncryptionKey_Missing(t *testing.T) {
	// 确保环境变量未设置
	os.Unsetenv("ENCRYPTION_KEY")

	_, err := LoadEncryptionKey()
	if err != ErrMissingEncryptionKey {
		t.Errorf("LoadEncryptionKey() error = %v, want %v", err, ErrMissingEncryptionKey)
	}
}

// TestLoadEncryptionKey_InvalidBase64 测试无效的 Base64
func TestLoadEncryptionKey_InvalidBase64(t *testing.T) {
	os.Setenv("ENCRYPTION_KEY", "not-valid-base64!@#$")
	defer os.Unsetenv("ENCRYPTION_KEY")

	_, err := LoadEncryptionKey()
	if err == nil {
		t.Error("LoadEncryptionKey() should fail with invalid Base64")
	}
}

// TestLoadEncryptionKey_WrongLength 测试错误的密钥长度
func TestLoadEncryptionKey_WrongLength(t *testing.T) {
	testCases := []struct {
		name   string
		length int
	}{
		{"16 bytes", 16},
		{"24 bytes", 24},
		{"64 bytes", 64},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			key := make([]byte, tc.length)
			keyStr := base64.StdEncoding.EncodeToString(key)

			os.Setenv("ENCRYPTION_KEY", keyStr)
			defer os.Unsetenv("ENCRYPTION_KEY")

			_, err := LoadEncryptionKey()
			if err == nil {
				t.Errorf("LoadEncryptionKey() should fail with %d bytes key", tc.length)
			}
		})
	}
}

// TestGenerateEncryptionKey 测试生成加密密钥
func TestGenerateEncryptionKey(t *testing.T) {
	keyStr, err := GenerateEncryptionKey()
	if err != nil {
		t.Fatalf("GenerateEncryptionKey() failed: %v", err)
	}

	if keyStr == "" {
		t.Error("GenerateEncryptionKey() returned empty string")
	}

	// 验证可以被 Base64 解码
	key, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		t.Fatalf("Generated key is not valid Base64: %v", err)
	}

	if len(key) != 32 {
		t.Errorf("Generated key length = %d, want 32", len(key))
	}
}

// TestGenerateEncryptionKey_Randomness 测试生成密钥的随机性
func TestGenerateEncryptionKey_Randomness(t *testing.T) {
	keys := make(map[string]bool)

	for i := 0; i < 10; i++ {
		keyStr, err := GenerateEncryptionKey()
		if err != nil {
			t.Fatalf("GenerateEncryptionKey() failed: %v", err)
		}

		if keys[keyStr] {
			t.Error("GenerateEncryptionKey() generated duplicate key")
		}
		keys[keyStr] = true
	}

	if len(keys) != 10 {
		t.Errorf("Expected 10 unique keys, got %d", len(keys))
	}
}

// TestValidateEncryptionKey_Success 测试验证有效密钥
func TestValidateEncryptionKey_Success(t *testing.T) {
	key := make([]byte, 32)
	keyStr := base64.StdEncoding.EncodeToString(key)

	err := ValidateEncryptionKey(keyStr)
	if err != nil {
		t.Errorf("ValidateEncryptionKey() failed: %v", err)
	}
}

// TestValidateEncryptionKey_Empty 测试验证空密钥
func TestValidateEncryptionKey_Empty(t *testing.T) {
	err := ValidateEncryptionKey("")
	if err != ErrMissingEncryptionKey {
		t.Errorf("ValidateEncryptionKey() error = %v, want %v", err, ErrMissingEncryptionKey)
	}
}

// TestValidateEncryptionKey_InvalidBase64 测试验证无效 Base64
func TestValidateEncryptionKey_InvalidBase64(t *testing.T) {
	err := ValidateEncryptionKey("not-valid-base64!@#")
	if err == nil {
		t.Error("ValidateEncryptionKey() should fail with invalid Base64")
	}
}

// TestValidateEncryptionKey_WrongLength 测试验证错误长度
func TestValidateEncryptionKey_WrongLength(t *testing.T) {
	testCases := []int{16, 24, 64}

	for _, length := range testCases {
		t.Run(string(rune(length)), func(t *testing.T) {
			key := make([]byte, length)
			keyStr := base64.StdEncoding.EncodeToString(key)

			err := ValidateEncryptionKey(keyStr)
			if err == nil {
				t.Errorf("ValidateEncryptionKey() should fail with %d bytes key", length)
			}
		})
	}
}
