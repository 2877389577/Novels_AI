package common

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

const (
	apiKeyCryptoIterations = 100000
	apiKeyCryptoKeySize    = 32
)

// APIKeyCrypto 负责 ai 提供商 API Key 的可逆加密。
//
// 数据库只保存 Encrypt 生成的密文；需要展示详情时，再通过 Decrypt 还原明文。
type APIKeyCrypto struct {
	key []byte
}

// NewAPIKeyCrypto 使用配置中的 secret 和 salt 派生 AES-256-GCM 密钥。
func NewAPIKeyCrypto(secret, salt string) (*APIKeyCrypto, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, fmt.Errorf("ai provider crypto secret is empty")
	}
	if strings.TrimSpace(salt) == "" {
		return nil, fmt.Errorf("ai provider crypto salt is empty")
	}

	key := pbkdf2.Key([]byte(secret), []byte(salt), apiKeyCryptoIterations, apiKeyCryptoKeySize, sha256.New)
	if _, err := aes.NewCipher(key); err != nil {
		return nil, fmt.Errorf("create ai provider crypto cipher: %w", err)
	}

	return &APIKeyCrypto{key: key}, nil
}

// Encrypt 使用随机 nonce 加密明文 API Key，并把 nonce 与密文一起编码为 base64。
func (c *APIKeyCrypto) Encrypt(plainText string) (string, error) {
	aesBlock, err := aes.NewCipher(c.key)
	if err != nil {
		return "", fmt.Errorf("create ai provider crypto cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(aesBlock)
	if err != nil {
		return "", fmt.Errorf("create ai provider crypto gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate ai provider crypto nonce: %w", err)
	}

	cipherText := gcm.Seal(nil, nonce, []byte(plainText), nil)
	payload := append(nonce, cipherText...)

	return base64.StdEncoding.EncodeToString(payload), nil
}

// Decrypt 解析 base64 密文，拆出 nonce 后还原 API Key 明文。
func (c *APIKeyCrypto) Decrypt(cipherText string) (string, error) {
	payload, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", fmt.Errorf("decode ai provider api key cipher text: %w", err)
	}

	aesBlock, err := aes.NewCipher(c.key)
	if err != nil {
		return "", fmt.Errorf("create ai provider crypto cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(aesBlock)
	if err != nil {
		return "", fmt.Errorf("create ai provider crypto gcm: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(payload) <= nonceSize {
		return "", fmt.Errorf("ai provider api key cipher text is invalid")
	}

	nonce := payload[:nonceSize]
	encrypted := payload[nonceSize:]
	plainText, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt ai provider api key: %w", err)
	}

	return string(plainText), nil
}
