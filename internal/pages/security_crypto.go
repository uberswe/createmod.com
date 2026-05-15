package pages

import (
	"createmod/internal/webhook"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
)

func securitySecret() string {
	if s := os.Getenv("SECURITY_SECRET"); s != "" {
		return s
	}
	if s := os.Getenv("WEBHOOK_SECRET"); s != "" {
		return s
	}
	if s := os.Getenv("OUT_SECRET"); s != "" {
		return s
	}
	slog.Warn("SECURITY_SECRET is not set — using insecure default; set SECURITY_SECRET in production")
	return "createmod-security-default-insecure"
}

func encryptTOTPSecret(plaintext string) (string, error) {
	return webhook.Encrypt(plaintext, securitySecret())
}

func decryptTOTPSecret(ciphertext string) (string, error) {
	return webhook.Decrypt(ciphertext, securitySecret())
}

func generateVerificationCode() (raw string, codeHash string, err error) {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generating verification code: %w", err)
	}
	num := (int(b[0])<<16 | int(b[1])<<8 | int(b[2])) % 1000000
	raw = fmt.Sprintf("%06d", num)
	return raw, hashCode(raw), nil
}

func hashCode(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
