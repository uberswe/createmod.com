package pages

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"createmod/internal/cache"
	"createmod/internal/store"
)

func hmacHex(msg, secret string) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(msg))
	return hex.EncodeToString(m.Sum(nil))
}

func Test_matchModSignature_MultipleSecrets(t *testing.T) {
	secrets := []string{"alpha", "bravo", "charlie"}
	msg := "1700000000:1.0:steve:slug"
	sig := hmacHex(msg, "bravo")

	got, ok := matchModSignature(msg, sig, secrets)
	if !ok || got != "bravo" {
		t.Fatalf("expected to match 'bravo', got %q ok=%v", got, ok)
	}
	if _, ok := matchModSignature(msg, sig, []string{"x", "y"}); ok {
		t.Fatalf("expected no match when the signing secret is absent")
	}
	if _, ok := matchModSignature(msg, sig, nil); ok {
		t.Fatalf("expected no match against an empty secret list")
	}
}

func Test_authenticateHMAC_AcceptsAnyAcceptedSecret(t *testing.T) {
	secrets := []string{"s1", "s2"}
	ts := time.Now().Unix()
	msg := fmt.Sprintf("%d:1.0:steve:slug", ts)
	sig := hmacHex(msg, "s2") // signed with the second secret

	req := httptest.NewRequest("GET", "/api/home", nil)
	req.Header.Set("X-Mod-Message", msg)
	req.Header.Set("X-Mod-Signature", sig)

	if auth := authenticateHMAC(req, secrets); auth == nil {
		t.Fatalf("expected HMAC auth to succeed with a secret from the list")
	}
	if auth := authenticateHMAC(req, []string{"other"}); auth != nil {
		t.Fatalf("expected HMAC auth to fail when no accepted secret matches")
	}
}

func Test_resolveModSecrets_MergesEnvAndDB_AndCaches(t *testing.T) {
	// Isolate the package-level env secrets.
	old := envModSecrets
	t.Cleanup(func() { envModSecrets = old })
	SetModSecrets([]string{"env-secret"})

	ms := &fakeModSecrets{active: []string{"db-secret-1", "db-secret-2"}}
	appStore := &store.Store{ModSecrets: ms}
	c := cache.New()

	got := resolveModSecrets(appStore, c)
	want := map[string]bool{"env-secret": true, "db-secret-1": true, "db-secret-2": true}
	if len(got) != len(want) {
		t.Fatalf("expected %d secrets, got %d: %v", len(want), len(got), got)
	}
	for _, s := range got {
		if !want[s] {
			t.Errorf("unexpected secret %q", s)
		}
	}

	// Second call within the TTL must be served from cache (no extra DB hit).
	_ = resolveModSecrets(appStore, c)
	if ms.listActiveCall != 1 {
		t.Fatalf("expected DB ListActive to be called once (cached after), got %d", ms.listActiveCall)
	}
}

func Test_resolveModSecrets_NilStoreSafe(t *testing.T) {
	old := envModSecrets
	t.Cleanup(func() { envModSecrets = old })
	SetModSecrets([]string{"only-env"})
	got := resolveModSecrets(&store.Store{}, cache.New())
	if len(got) != 1 || got[0] != "only-env" {
		t.Fatalf("expected just the env secret when no ModSecrets store, got %v", got)
	}
}
