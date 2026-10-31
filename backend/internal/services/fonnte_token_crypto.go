package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

func EncryptFonnteToken(token, secret string) (string, error) {
	if strings.TrimSpace(token) == "" {
		return "", errors.New("Fonnte token is empty")
	}
	if len([]byte(secret)) < 32 {
		return "", errors.New("webhook secret must contain at least 32 bytes")
	}

	key := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", fmt.Errorf("create token cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create token cipher mode: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("create token nonce: %w", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(token), nil)
	return "v1:" + base64.RawStdEncoding.EncodeToString(ciphertext), nil
}

func DecryptFonnteToken(encrypted, secret string) (string, error) {
	if len([]byte(secret)) < 32 {
		return "", errors.New("webhook secret must contain at least 32 bytes")
	}
	if !strings.HasPrefix(encrypted, "v1:") {
		return "", errors.New("encrypted Fonnte token has an unsupported format")
	}

	payload, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(encrypted, "v1:"))
	if err != nil {
		return "", fmt.Errorf("decode encrypted Fonnte token: %w", err)
	}
	key := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", fmt.Errorf("create token cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create token cipher mode: %w", err)
	}
	if len(payload) < gcm.NonceSize()+gcm.Overhead() {
		return "", errors.New("encrypted Fonnte token is incomplete")
	}
	nonce, ciphertext := payload[:gcm.NonceSize()], payload[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt Fonnte token: %w", err)
	}
	return string(plaintext), nil
}
