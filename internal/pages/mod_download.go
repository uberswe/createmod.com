package pages

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"createmod/internal/cache"
	"createmod/internal/ratelimit"
	"createmod/internal/server"
	"createmod/internal/storage"
	"createmod/internal/store"
)

// modDownloadRequest is the JSON body for POST /api/mod/download.
type modDownloadRequest struct {
	Message   string `json:"message"`
	Signature string `json:"signature"`
	Type      string `json:"type"`
}

// maxModTimestampAge is the maximum age for a mod download request timestamp.
const maxModTimestampAge = 5 * time.Minute

// maxModTimestampFuture is the maximum amount a timestamp may be in the future
// (to allow minor clock skew).
const maxModTimestampFuture = 30 * time.Second

// validateModSignature verifies that signature is a valid HMAC-SHA256 of message
// using the given secret.
func validateModSignature(message, signature, secret string) bool {
	if message == "" || signature == "" || secret == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// parseModMessage parses a "timestamp:modversion:mcusername:identifier" message
// and validates the timestamp is within the allowed window.
func parseModMessage(message string, maxAge time.Duration) (timestamp int64, modVersion, mcUsername, identifier string, err error) {
	parts := strings.SplitN(message, ":", 4)
	if len(parts) != 4 {
		return 0, "", "", "", fmt.Errorf("expected 4 colon-separated fields, got %d", len(parts))
	}

	timestamp, err = strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, "", "", "", fmt.Errorf("invalid timestamp: %w", err)
	}

	now := time.Now().Unix()
	if timestamp > now+int64(maxModTimestampFuture.Seconds()) {
		return 0, "", "", "", fmt.Errorf("timestamp is in the future")
	}
	if now-timestamp > int64(maxAge.Seconds()) {
		return 0, "", "", "", fmt.Errorf("timestamp expired")
	}

	modVersion = parts[1]
	mcUsername = parts[2]
	identifier = parts[3]

	return timestamp, modVersion, mcUsername, identifier, nil
}

// deriveXORKey produces a 32-byte XOR key from SHA256(secret + timestamp_string).
func deriveXORKey(secret string, timestamp int64) []byte {
	h := sha256.New()
	h.Write([]byte(secret))
	h.Write([]byte(strconv.FormatInt(timestamp, 10)))
	return h.Sum(nil)
}

// xorEncode applies a cyclical XOR of data with key.
func xorEncode(data, key []byte) []byte {
	if len(key) == 0 {
		return data
	}
	out := make([]byte, len(data))
	for i, b := range data {
		out[i] = b ^ key[i%len(key)]
	}
	return out
}

// ModDownloadHandler handles POST /api/mod/download.
// It validates an HMAC-signed request from the Minecraft mod, looks up the
// schematic or private upload, and returns the NBT bytes XOR-encoded.
func ModDownloadHandler(rl ratelimit.Limiter, cacheService *cache.Service, appStore *store.Store, storageSvc *storage.Service, modSecret string) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if storageSvc == nil {
			return e.JSON(http.StatusServiceUnavailable, map[string]string{"error": "file storage not configured"})
		}

		// Parse JSON body.
		var req modDownloadRequest
		if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		}

		// Default type to "schematic".
		if req.Type == "" {
			req.Type = "schematic"
		}
		if req.Type != "schematic" && req.Type != "upload" {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid type; use 'schematic' or 'upload'"})
		}

		// Validate HMAC signature.
		if !validateModSignature(req.Message, req.Signature, modSecret) {
			return e.JSON(http.StatusForbidden, map[string]string{"error": "invalid signature"})
		}

		// Parse and validate the message fields.
		timestamp, modVersion, mcUsername, identifier, err := parseModMessage(req.Message, maxModTimestampAge)
		if err != nil {
			return e.JSON(http.StatusForbidden, map[string]string{"error": "expired or invalid timestamp"})
		}

		// Log the request for analytics.
		slog.Info("mod_download",
			"mod_version", modVersion,
			"mc_username", mcUsername,
			"identifier", identifier,
			"type", req.Type,
			"ip", e.RealIP(),
		)

		// Daily download rate limit (shared with browser downloads).
		if ok, _ := downloadRateLimitAllow(rl, e.RealIP()); !ok {
			return e.JSON(http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
		}

		var fileBytes []byte

		switch req.Type {
		case "schematic":
			fileBytes, err = modDownloadSchematic(e.Request.Context(), appStore, storageSvc, rl, cacheService, identifier, e.RealIP())
		case "upload":
			fileBytes, err = modDownloadUpload(e.Request.Context(), appStore, storageSvc, identifier)
		}
		if err != nil {
			if apiErr, ok := err.(*modDownloadError); ok {
				return e.JSON(apiErr.status, map[string]string{"error": apiErr.message})
			}
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to retrieve file"})
		}

		// XOR-encode the response.
		xorKey := deriveXORKey(modSecret, timestamp)
		encoded := xorEncode(fileBytes, xorKey)

		e.Response.Header().Set("Content-Type", "application/octet-stream")
		e.Response.Header().Set("X-Mod-Encoding", "xor-v1")
		e.Response.Header().Set("X-Original-Size", strconv.Itoa(len(fileBytes)))
		e.Response.WriteHeader(http.StatusOK)
		_, writeErr := e.Response.Write(encoded)
		return writeErr
	}
}

// modDownloadError is a typed error carrying an HTTP status and message.
type modDownloadError struct {
	status  int
	message string
}

func (e *modDownloadError) Error() string { return e.message }

func modDownloadSchematic(ctx context.Context, appStore *store.Store, storageSvc *storage.Service, rl ratelimit.Limiter, cacheService *cache.Service, slug string, clientIP string) ([]byte, error) {
	s, err := appStore.Schematics.GetByName(ctx, slug)
	if err != nil || s == nil || (s.Deleted != nil && !s.Deleted.IsZero()) {
		return nil, &modDownloadError{http.StatusNotFound, "schematic not found"}
	}

	if !s.Moderated {
		return nil, &modDownloadError{http.StatusNotFound, "schematic not found"}
	}

	if s.Paid {
		return nil, &modDownloadError{http.StatusForbidden, "paid schematic; use external link"}
	}

	if s.Blacklisted {
		return nil, &modDownloadError{http.StatusForbidden, "schematic is blacklisted"}
	}

	primary := strings.TrimSpace(s.SchematicFile)
	if primary == "" {
		return nil, &modDownloadError{http.StatusNotFound, "schematic not found"}
	}

	s3Collection := storage.CollectionPrefix("schematics")
	reader, err := storageSvc.Download(ctx, s3Collection, s.ID, primary)
	if err != nil {
		return nil, &modDownloadError{http.StatusInternalServerError, "failed to retrieve file"}
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, &modDownloadError{http.StatusInternalServerError, "failed to retrieve file"}
	}

	// Count the download (best-effort, IP-deduped) — reuses existing logic.
	countSchematicDownloadStore(appStore, s.ID, clientIP, rl, cacheService)

	return data, nil
}

func modDownloadUpload(ctx context.Context, appStore *store.Store, storageSvc *storage.Service, token string) ([]byte, error) {
	entry, err := appStore.TempUploads.GetByToken(ctx, token)
	if err != nil || entry == nil {
		return nil, &modDownloadError{http.StatusNotFound, "upload not found"}
	}

	if entry.NbtS3Key == "" {
		return nil, &modDownloadError{http.StatusNotFound, "upload not found"}
	}

	reader, err := storageSvc.DownloadRaw(ctx, entry.NbtS3Key)
	if err != nil {
		return nil, &modDownloadError{http.StatusInternalServerError, "failed to retrieve file"}
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, &modDownloadError{http.StatusInternalServerError, "failed to retrieve file"}
	}

	return data, nil
}
