package outurl

import (
	"testing"
)

const testSecret = "test-secret-key-for-outurl"

func TestEncodeDecodeRoundTrip(t *testing.T) {
	rawURL := "https://example.com/page?q=1"
	token := Encode(rawURL, testSecret)
	if token == "" {
		t.Fatal("Encode returned empty token")
	}
	got, err := Decode(token, testSecret)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}
	if got != rawURL {
		t.Errorf("got %q, want %q", got, rawURL)
	}
}

func TestEncodeDeterministic(t *testing.T) {
	rawURL := "https://example.com/stable"
	token1 := Encode(rawURL, testSecret)
	token2 := Encode(rawURL, testSecret)
	if token1 != token2 {
		t.Errorf("tokens differ: %q vs %q", token1, token2)
	}
}

func TestDecodeInvalidToken(t *testing.T) {
	_, err := Decode("not-a-valid-token!!!", testSecret)
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestDecodeWrongSecret(t *testing.T) {
	token := Encode("https://example.com", testSecret)
	_, err := Decode(token, "wrong-secret")
	if err == nil {
		t.Error("expected error for wrong secret")
	}
}

func TestEncodeWithSourceRoundTrip(t *testing.T) {
	rawURL := "https://example.com/schematic"
	token := EncodeWithSource(rawURL, "schematic", "abc123", testSecret)
	if token == "" {
		t.Fatal("EncodeWithSource returned empty token")
	}
	p, err := DecodePayload(token, testSecret)
	if err != nil {
		t.Fatalf("DecodePayload error: %v", err)
	}
	if p.URL != rawURL {
		t.Errorf("URL: got %q, want %q", p.URL, rawURL)
	}
	if p.SourceType != "schematic" {
		t.Errorf("SourceType: got %q, want %q", p.SourceType, "schematic")
	}
	if p.SourceID != "abc123" {
		t.Errorf("SourceID: got %q, want %q", p.SourceID, "abc123")
	}
}

func TestDecodePayloadBackwardCompat(t *testing.T) {
	// Old-style tokens encrypt a bare URL, not JSON.
	rawURL := "https://example.com/old-link"
	token := Encode(rawURL, testSecret)
	p, err := DecodePayload(token, testSecret)
	if err != nil {
		t.Fatalf("DecodePayload error on old token: %v", err)
	}
	if p.URL != rawURL {
		t.Errorf("URL: got %q, want %q", p.URL, rawURL)
	}
	if p.SourceType != "" {
		t.Errorf("SourceType should be empty for old tokens, got %q", p.SourceType)
	}
	if p.SourceID != "" {
		t.Errorf("SourceID should be empty for old tokens, got %q", p.SourceID)
	}
}

func TestDecodePayloadInvalidToken(t *testing.T) {
	_, err := DecodePayload("garbage!!!", testSecret)
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestDecodePayloadWrongSecret(t *testing.T) {
	token := EncodeWithSource("https://example.com", "video", "v1", testSecret)
	_, err := DecodePayload(token, "wrong-secret")
	if err == nil {
		t.Error("expected error for wrong secret")
	}
}

func TestBuildPath(t *testing.T) {
	path := BuildPath("https://example.com", testSecret)
	if path == "" || path == "/out/" {
		t.Errorf("BuildPath returned empty or stub path: %q", path)
	}
	if path[:5] != "/out/" {
		t.Errorf("BuildPath should start with /out/, got %q", path)
	}
}

func TestBuildPathWithSource(t *testing.T) {
	path := BuildPathWithSource("https://example.com", testSecret, "guide", "g1")
	if path == "" || path == "/out/" {
		t.Errorf("BuildPathWithSource returned empty or stub path: %q", path)
	}
	if path[:5] != "/out/" {
		t.Errorf("BuildPathWithSource should start with /out/, got %q", path)
	}
	// Verify the token part decodes correctly
	token := path[5:]
	p, err := DecodePayload(token, testSecret)
	if err != nil {
		t.Fatalf("DecodePayload error: %v", err)
	}
	if p.SourceType != "guide" || p.SourceID != "g1" {
		t.Errorf("source context mismatch: type=%q id=%q", p.SourceType, p.SourceID)
	}
}

func TestEncodeRejectsNonHTTP(t *testing.T) {
	// Encode itself doesn't validate, but Decode should reject non-http(s)
	token := Encode("ftp://evil.com/file", testSecret)
	_, err := Decode(token, testSecret)
	if err == nil {
		t.Error("expected error for ftp URL")
	}
}

func TestDecodePayloadRejectsNonHTTP(t *testing.T) {
	token := EncodeWithSource("ftp://evil.com/file", "test", "id1", testSecret)
	_, err := DecodePayload(token, testSecret)
	if err == nil {
		t.Error("expected error for ftp URL in payload")
	}
}
