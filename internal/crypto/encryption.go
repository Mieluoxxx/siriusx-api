package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

var (
	// ErrInvalidKeySize 密钥长度错误
	ErrInvalidKeySize = errors.New("invalid key size: must be 32 bytes for AES-256")
	// ErrInvalidCiphertext 密文格式错误
	ErrInvalidCiphertext = errors.New("invalid ciphertext: too short or corrupted")
	// ErrDecryptionFailed 解密失败
	ErrDecryptionFailed = errors.New("decryption failed: authentication tag verification failed")
)

// Encrypt 使用 AES-256-GCM 加密明文
// 返回 Base64 编码的密文（包含 Nonce）
func Encrypt(plaintext []byte, key []byte) (string, error) {
	// 验证密钥长度
	if len(key) != 32 {
		return "", ErrInvalidKeySize
	}

	// 创建 AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// 创建 GCM mode
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 生成随机 Nonce (12 字节)
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// 加密：Seal 会自动附加认证标签
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)

	// Base64 编码（nonce + ciphertext + tag）
	encoded := base64.StdEncoding.EncodeToString(ciphertext)

	return encoded, nil
}

// Decrypt 使用 AES-256-GCM 解密密文
// 输入是 Base64 编码的密文
func Decrypt(ciphertext string, key []byte) ([]byte, error) {
	// 验证密钥长度
	if len(key) != 32 {
		return nil, ErrInvalidKeySize
	}

	// Base64 解码
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, errors.New("invalid base64 encoding: " + err.Error())
	}

	// 创建 AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// 创建 GCM mode
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// 验证数据长度（至少要有 nonce）
	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return nil, ErrInvalidCiphertext
	}

	// 提取 Nonce 和密文
	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]

	// 解密并验证认证标签
	plaintext, err := aesGCM.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// EncryptString 加密字符串（便捷函数）
func EncryptString(plaintext string, key []byte) (string, error) {
	return Encrypt([]byte(plaintext), key)
}

// DecryptString 解密到字符串（便捷函数）
func DecryptString(ciphertext string, key []byte) (string, error) {
	plaintext, err := Decrypt(ciphertext, key)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
