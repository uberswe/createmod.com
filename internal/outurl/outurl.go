package outurl

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/url"
)

// Payload represents the data encrypted inside an /out token.
type Payload struct {
	URL        string `json:"url"`
	SourceType string `json:"source_type,omitempty"`
	SourceID   string `json:"source_id,omitempty"`
}

// deriveKey returns a 32-byte AES-256 key from the secret string.
func deriveKey(secret string) []byte {
	h := sha256.Sum256([]byte(secret))
	return h[:]
}

// deriveNonce creates a deterministic 12-byte GCM nonce from the URL,
// so the same URL always produces the same token.
func deriveNonce(rawURL string, key []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(rawURL))
	return mac.Sum(nil)[:12]
}

// Encode encrypts a URL into a URL-safe base64 token using AES-GCM.
func Encode(rawURL string, secret string) string {
	key := deriveKey(secret)
	block, err := aes.NewCipher(key)
	if err != nil {
		return ""
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return ""
	}
	nonce := deriveNonce(rawURL, key)
	ciphertext := gcm.Seal(nonce, nonce, []byte(rawURL), nil)
	return base64.RawURLEncoding.EncodeToString(ciphertext)
}

// Decode decrypts a token back to the original URL.
func Decode(token string, secret string) (string, error) {
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
		return "", errors.New("token too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", errors.New("invalid token")
	}
	// Validate the decrypted value is a valid http(s) URL
	u, err := url.Parse(string(plaintext))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return "", errors.New("invalid url in token")
	}
	return u.String(), nil
}

// BuildPath returns "/out/<token>" for use in href attributes.
func BuildPath(rawURL string, secret string) string {
	return "/out/" + Encode(rawURL, secret)
}

// EncodeWithSource encrypts a JSON payload containing a URL and source context.
func EncodeWithSource(rawURL, sourceType, sourceID, secret string) string {
	p := Payload{URL: rawURL, SourceType: sourceType, SourceID: sourceID}
	data, err := json.Marshal(p)
	if err != nil {
		return ""
	}
	key := deriveKey(secret)
	block, err := aes.NewCipher(key)
	if err != nil {
		return ""
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return ""
	}
	nonce := deriveNonce(string(data), key)
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return base64.RawURLEncoding.EncodeToString(ciphertext)
}

// DecodePayload decrypts a token and returns a Payload.
// It first tries to parse the plaintext as JSON; if that fails it falls back
// to treating the plaintext as a bare URL (backward compat with old tokens).
func DecodePayload(token string, secret string) (Payload, error) {
	key := deriveKey(secret)
	data, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return Payload{}, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return Payload{}, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return Payload{}, err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize+gcm.Overhead() {
		return Payload{}, errors.New("token too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return Payload{}, errors.New("invalid token")
	}

	// Try JSON first (new-style token)
	var p Payload
	if err := json.Unmarshal(plaintext, &p); err == nil && p.URL != "" {
		u, err := url.Parse(p.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			return Payload{}, errors.New("invalid url in token")
		}
		p.URL = u.String()
		return p, nil
	}

	// Fallback: plain URL (old-style token)
	u, err := url.Parse(string(plaintext))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return Payload{}, errors.New("invalid url in token")
	}
	return Payload{URL: u.String()}, nil
}

// BuildPathWithSource returns "/out/<token>" with source context embedded.
func BuildPathWithSource(rawURL, secret, sourceType, sourceID string) string {
	return "/out/" + EncodeWithSource(rawURL, sourceType, sourceID, secret)
}
