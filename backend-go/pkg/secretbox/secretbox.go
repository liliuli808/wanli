// Package secretbox 提供基于 AES-256-GCM 的对称加密，用于存放用户敏感凭证。
package secretbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// Box 是配置好密钥的加解密器。
type Box struct {
	aead cipher.AEAD
}

// New 从任意长度的密钥材料派生 32 字节密钥并创建 Box。
func New(keyMaterial string) (*Box, error) {
	if keyMaterial == "" {
		return nil, errors.New("加密密钥不能为空")
	}
	sum := sha256.Sum256([]byte(keyMaterial))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return nil, fmt.Errorf("初始化 AES 失败: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("初始化 GCM 失败: %w", err)
	}
	return &Box{aead: aead}, nil
}

// Encrypt 加密明文，返回 base64(nonce||ciphertext)。
func (b *Box) Encrypt(plaintext string) (string, error) {
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("生成 nonce 失败: %w", err)
	}
	sealed := b.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt 解密 Encrypt 的输出。
func (b *Box) Decrypt(token string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return "", fmt.Errorf("解码密文失败: %w", err)
	}
	nonceSize := b.aead.NonceSize()
	if len(raw) < nonceSize {
		return "", errors.New("密文长度不足")
	}
	plain, err := b.aead.Open(nil, raw[:nonceSize], raw[nonceSize:], nil)
	if err != nil {
		return "", errors.New("解密失败：密钥不匹配或密文被篡改")
	}
	return string(plain), nil
}
