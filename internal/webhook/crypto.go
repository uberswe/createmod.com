// Package webhook provides encryption and validation helpers for
// user-configured Discord webhook URLs.
package webhook

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"os"
)

// deriveKey returns a 32-byte AES-256 key from the secret string.
func deriveKey(secret string) []byte {
	h := sha256.Sum256([]byte(secret))
	return h[:]
}

// Encrypt encrypts plaintext using AES-256-GCM with a random nonce and
// returns a base64url-encoded ciphertext.
func Encrypt(plaintext, secret string) (string, error) {
	key := deriveKey(secret)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

// Decrypt reverses Encrypt: it decodes the base64url token and decrypts
// with AES-256-GCM.
func Decrypt(token, secret string) (string, error) {
	key := deriveKey(secret)
	data, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize+gcm.Overhead() {
		return "", errors.New("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", errors.New("decryption failed")
	}
	return string(plaintext), nil
}

// Secret returns the webhook encryption secret from the environment.
// It prefers WEBHOOK_SECRET and falls back to OUT_SECRET.
func Secret() string {
	if s := os.Getenv("WEBHOOK_SECRET"); s != "" {
		return s
	}
	if s := os.Getenv("OUT_SECRET"); s != "" {
		return s
	}
	return "createmod-webhook-default-insecure"
}
