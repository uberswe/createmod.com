package pages

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"testing"
	"time"
)

func Test_validateModSignature(t *testing.T) {
	secret := "test-secret-key"
	message := "1710500000:1.2.0:StevePlayer:train-car-3"

	// Compute correct signature.
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	correctSig := hex.EncodeToString(mac.Sum(nil))

	t.Run("correct signature", func(t *testing.T) {
		if !validateModSignature(message, correctSig, secret) {
			t.Fatal("expected valid signature to pass")
		}
	})

	t.Run("wrong signature", func(t *testing.T) {
		if validateModSignature(message, "deadbeef", secret) {
			t.Fatal("expected wrong signature to fail")
		}
	})

	t.Run("wrong secret", func(t *testing.T) {
		if validateModSignature(message, correctSig, "wrong-secret") {
			t.Fatal("expected wrong secret to fail")
		}
	})

	t.Run("empty message", func(t *testing.T) {
		if validateModSignature("", correctSig, secret) {
			t.Fatal("expected empty message to fail")
		}
	})

	t.Run("empty signature", func(t *testing.T) {
		if validateModSignature(message, "", secret) {
			t.Fatal("expected empty signature to fail")
		}
	})

	t.Run("empty secret", func(t *testing.T) {
		if validateModSignature(message, correctSig, "") {
			t.Fatal("expected empty secret to fail")
		}
	})
}

func Test_parseModMessage(t *testing.T) {
	now := time.Now().Unix()
	maxAge := 5 * time.Minute

	t.Run("valid message", func(t *testing.T) {
		msg := strconv.FormatInt(now, 10) + ":1.2.0:StevePlayer:train-car-3"
		ts, ver, user, id, err := parseModMessage(msg, maxAge)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ts != now {
			t.Fatalf("expected timestamp %d, got %d", now, ts)
		}
		if ver != "1.2.0" {
			t.Fatalf("expected version 1.2.0, got %s", ver)
		}
		if user != "StevePlayer" {
			t.Fatalf("expected username StevePlayer, got %s", user)
		}
		if id != "train-car-3" {
			t.Fatalf("expected identifier train-car-3, got %s", id)
		}
	})

	t.Run("expired timestamp", func(t *testing.T) {
		old := now - 600 // 10 minutes ago
		msg := strconv.FormatInt(old, 10) + ":1.0.0:Player:slug"
		_, _, _, _, err := parseModMessage(msg, maxAge)
		if err == nil {
			t.Fatal("expected error for expired timestamp")
		}
	})

	t.Run("future timestamp within tolerance", func(t *testing.T) {
		future := now + 20 // 20 seconds in the future (within 30s tolerance)
		msg := strconv.FormatInt(future, 10) + ":1.0.0:Player:slug"
		_, _, _, _, err := parseModMessage(msg, maxAge)
		if err != nil {
			t.Fatalf("expected near-future timestamp to pass: %v", err)
		}
	})

	t.Run("future timestamp beyond tolerance", func(t *testing.T) {
		future := now + 120 // 2 minutes in the future
		msg := strconv.FormatInt(future, 10) + ":1.0.0:Player:slug"
		_, _, _, _, err := parseModMessage(msg, maxAge)
		if err == nil {
			t.Fatal("expected error for far-future timestamp")
		}
	})

	t.Run("malformed - too few fields", func(t *testing.T) {
		_, _, _, _, err := parseModMessage("123:abc", maxAge)
		if err == nil {
			t.Fatal("expected error for too few fields")
		}
	})

	t.Run("malformed - non-numeric timestamp", func(t *testing.T) {
		_, _, _, _, err := parseModMessage("notanumber:1.0.0:Player:slug", maxAge)
		if err == nil {
			t.Fatal("expected error for non-numeric timestamp")
		}
	})

	t.Run("empty identifier", func(t *testing.T) {
		msg := strconv.FormatInt(now, 10) + ":1.0.0:Player:"
		_, _, _, _, err := parseModMessage(msg, maxAge)
		if err == nil {
			t.Fatal("expected error for empty identifier")
		}
	})

	t.Run("identifier with colons", func(t *testing.T) {
		msg := strconv.FormatInt(now, 10) + ":1.0.0:Player:id:with:colons"
		_, _, _, id, err := parseModMessage(msg, maxAge)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "id:with:colons" {
			t.Fatalf("expected identifier 'id:with:colons', got %s", id)
		}
	})
}

func Test_xorEncode(t *testing.T) {
	t.Run("roundtrip", func(t *testing.T) {
		data := []byte("Hello, World! This is test data for XOR encoding.")
		key := []byte("secret-key-1234")

		encoded := xorEncode(data, key)
		decoded := xorEncode(encoded, key)

		if string(decoded) != string(data) {
			t.Fatalf("roundtrip failed: got %q, want %q", decoded, data)
		}
	})

	t.Run("encoding changes data", func(t *testing.T) {
		data := []byte("plaintext")
		key := []byte("key")
		encoded := xorEncode(data, key)

		same := true
		for i := range data {
			if data[i] != encoded[i] {
				same = false
				break
			}
		}
		if same {
			t.Fatal("expected XOR encoding to change data")
		}
	})

	t.Run("key wrapping", func(t *testing.T) {
		// Data longer than key should still work (cyclical)
		data := make([]byte, 100)
		for i := range data {
			data[i] = byte(i)
		}
		key := []byte{0xFF} // 1-byte key

		encoded := xorEncode(data, key)
		for i, b := range encoded {
			expected := byte(i) ^ 0xFF
			if b != expected {
				t.Fatalf("byte %d: got %d, want %d", i, b, expected)
			}
		}
	})

	t.Run("empty key returns original", func(t *testing.T) {
		data := []byte("hello")
		result := xorEncode(data, nil)
		if string(result) != string(data) {
			t.Fatalf("empty key should return original data")
		}
	})

	t.Run("empty data", func(t *testing.T) {
		result := xorEncode(nil, []byte("key"))
		if len(result) != 0 {
			t.Fatalf("expected empty result for empty data")
		}
	})
}

func Test_deriveXORKey(t *testing.T) {
	secret := "test-secret"

	t.Run("deterministic", func(t *testing.T) {
		k1 := deriveXORKey(secret, 1000)
		k2 := deriveXORKey(secret, 1000)
		if hex.EncodeToString(k1) != hex.EncodeToString(k2) {
			t.Fatal("expected same inputs to produce same key")
		}
	})

	t.Run("different timestamps produce different keys", func(t *testing.T) {
		k1 := deriveXORKey(secret, 1000)
		k2 := deriveXORKey(secret, 2000)
		if hex.EncodeToString(k1) == hex.EncodeToString(k2) {
			t.Fatal("expected different timestamps to produce different keys")
		}
	})

	t.Run("returns 32 bytes", func(t *testing.T) {
		k := deriveXORKey(secret, 12345)
		if len(k) != 32 {
			t.Fatalf("expected 32 bytes, got %d", len(k))
		}
	})

	t.Run("different secrets produce different keys", func(t *testing.T) {
		k1 := deriveXORKey("secret-a", 1000)
		k2 := deriveXORKey("secret-b", 1000)
		if hex.EncodeToString(k1) == hex.EncodeToString(k2) {
			t.Fatal("expected different secrets to produce different keys")
		}
	})
}
