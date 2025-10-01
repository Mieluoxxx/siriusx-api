package crypto

import (
	"crypto/rand"
	"testing"
)

// generateTestKey 生成测试用的 32 字节密钥
func generateTestKey() []byte {
	key := make([]byte, 32)
	rand.Read(key)
	return key
}

// TestEncrypt 测试加密功能
func TestEncrypt(t *testing.T) {
	key := generateTestKey()
	plaintext := []byte("sk-test-key-12345")

	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt() failed: %v", err)
	}

	if ciphertext == "" {
		t.Error("Encrypt() returned empty ciphertext")
	}

	if ciphertext == string(plaintext) {
		t.Error("Encrypt() returned plaintext unchanged")
	}
}

// TestDecrypt 测试解密功能
func TestDecrypt(t *testing.T) {
	key := generateTestKey()
	plaintext := []byte("sk-test-key-12345")

	// 加密
	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt() failed: %v", err)
	}

	// 解密
	decrypted, err := Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Decrypt() failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypt() got %s, want %s", string(decrypted), string(plaintext))
	}
}

// TestEncryptDecryptRoundTrip 测试加密/解密往返
func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := generateTestKey()

	testCases := []string{
		"sk-test-key-12345",
		"very-long-api-key-with-many-characters-1234567890",
		"short",
		"",
		"特殊字符!@#$%^&*()",
	}

	for _, plaintext := range testCases {
		t.Run(plaintext, func(t *testing.T) {
			ciphertext, err := EncryptString(plaintext, key)
			if err != nil {
				t.Fatalf("EncryptString() failed: %v", err)
			}

			decrypted, err := DecryptString(ciphertext, key)
			if err != nil {
				t.Fatalf("DecryptString() failed: %v", err)
			}

			if decrypted != plaintext {
				t.Errorf("Roundtrip failed: got %s, want %s", decrypted, plaintext)
			}
		})
	}
}

// TestNonceRandomness 测试 Nonce 随机性
func TestNonceRandomness(t *testing.T) {
	key := generateTestKey()
	plaintext := []byte("sk-test-key-12345")

	// 多次加密相同明文
	ciphertexts := make(map[string]bool)
	for i := 0; i < 10; i++ {
		ciphertext, err := Encrypt(plaintext, key)
		if err != nil {
			t.Fatalf("Encrypt() failed: %v", err)
		}

		if ciphertexts[ciphertext] {
			t.Error("Nonce not random: same ciphertext produced twice")
		}
		ciphertexts[ciphertext] = true
	}

	if len(ciphertexts) != 10 {
		t.Errorf("Expected 10 unique ciphertexts, got %d", len(ciphertexts))
	}
}

// TestDecryptWithWrongKey 测试使用错误密钥解密
func TestDecryptWithWrongKey(t *testing.T) {
	key1 := generateTestKey()
	key2 := generateTestKey()
	plaintext := []byte("sk-test-key-12345")

	// 使用 key1 加密
	ciphertext, err := Encrypt(plaintext, key1)
	if err != nil {
		t.Fatalf("Encrypt() failed: %v", err)
	}

	// 使用 key2 解密
	_, err = Decrypt(ciphertext, key2)
	if err == nil {
		t.Error("Decrypt() with wrong key should fail")
	}

	if err != ErrDecryptionFailed {
		t.Errorf("Expected ErrDecryptionFailed, got %v", err)
	}
}

// TestEncryptInvalidKey 测试使用无效密钥加密
func TestEncryptInvalidKey(t *testing.T) {
	testCases := []struct {
		name   string
		key    []byte
		expErr error
	}{
		{"empty key", []byte{}, ErrInvalidKeySize},
		{"short key", []byte("short"), ErrInvalidKeySize},
		{"long key", make([]byte, 64), ErrInvalidKeySize},
		{"16 bytes key", make([]byte, 16), ErrInvalidKeySize},
	}

	plaintext := []byte("test")

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Encrypt(plaintext, tc.key)
			if err != tc.expErr {
				t.Errorf("Expected error %v, got %v", tc.expErr, err)
			}
		})
	}
}

// TestDecryptInvalidKey 测试使用无效密钥解密
func TestDecryptInvalidKey(t *testing.T) {
	testCases := []struct {
		name   string
		key    []byte
		expErr error
	}{
		{"empty key", []byte{}, ErrInvalidKeySize},
		{"short key", []byte("short"), ErrInvalidKeySize},
	}

	ciphertext := "dmFsaWRfY2lwaGVydGV4dA==" // 随机 base64

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Decrypt(ciphertext, tc.key)
			if err != tc.expErr {
				t.Errorf("Expected error %v, got %v", tc.expErr, err)
			}
		})
	}
}

// TestDecryptInvalidCiphertext 测试解密无效密文
func TestDecryptInvalidCiphertext(t *testing.T) {
	key := generateTestKey()

	testCases := []struct {
		name       string
		ciphertext string
	}{
		{"invalid base64", "not-valid-base64!@#"},
		{"too short", "YWJj"}, // "abc" in base64 (3 bytes)
		{"corrupted data", "dmFsaWRfYmFzZTY0X2J1dF9pbnZhbGlkX2RhdGE="},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Decrypt(tc.ciphertext, key)
			if err == nil {
				t.Error("Decrypt() should fail with invalid ciphertext")
			}
		})
	}
}

// TestDecryptTamperedData 测试解密被篡改的数据
func TestDecryptTamperedData(t *testing.T) {
	key := generateTestKey()
	plaintext := []byte("sk-test-key-12345")

	// 加密
	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt() failed: %v", err)
	}

	// 篡改密文（修改最后一个字符）
	if len(ciphertext) > 0 {
		tamperedCiphertext := ciphertext[:len(ciphertext)-1] + "X"

		// 尝试解密篡改后的数据
		_, err = Decrypt(tamperedCiphertext, key)
		if err == nil {
			t.Error("Decrypt() should fail with tampered data")
		}
	}
}

// TestEncryptString 测试字符串加密便捷函数
func TestEncryptString(t *testing.T) {
	key := generateTestKey()
	plaintext := "sk-test-key-12345"

	ciphertext, err := EncryptString(plaintext, key)
	if err != nil {
		t.Fatalf("EncryptString() failed: %v", err)
	}

	if ciphertext == plaintext {
		t.Error("EncryptString() returned plaintext unchanged")
	}
}

// TestDecryptString 测试字符串解密便捷函数
func TestDecryptString(t *testing.T) {
	key := generateTestKey()
	plaintext := "sk-test-key-12345"

	ciphertext, _ := EncryptString(plaintext, key)

	decrypted, err := DecryptString(ciphertext, key)
	if err != nil {
		t.Fatalf("DecryptString() failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("DecryptString() got %s, want %s", decrypted, plaintext)
	}
}

// BenchmarkEncrypt 加密性能基准测试
func BenchmarkEncrypt(b *testing.B) {
	key := generateTestKey()
	plaintext := []byte("sk-test-key-12345")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Encrypt(plaintext, key)
	}
}

// BenchmarkDecrypt 解密性能基准测试
func BenchmarkDecrypt(b *testing.B) {
	key := generateTestKey()
	plaintext := []byte("sk-test-key-12345")
	ciphertext, _ := Encrypt(plaintext, key)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Decrypt(ciphertext, key)
	}
}
