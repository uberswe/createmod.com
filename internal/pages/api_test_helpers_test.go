package pages

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http/httptest"
	"time"

	"createmod/internal/server"
	"createmod/internal/store"
)

// testModSecret is the HMAC secret used across API handler tests.
const testModSecret = "test-mod-secret"

// --- fake rate limiter -------------------------------------------------------

// fakeLimiter satisfies ratelimit.Limiter. Allow always permits (so rate
// limiting never interferes with a test unless allow is set false); Check
// reports whether a key was previously Mark-ed, giving realistic dedup.
type fakeLimiter struct {
	allow bool
	marks map[string]bool
}

func newFakeLimiter() *fakeLimiter { return &fakeLimiter{allow: true, marks: map[string]bool{}} }

func (f *fakeLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, int) {
	if f.allow {
		return true, limit
	}
	return false, 0
}
func (f *fakeLimiter) Check(ctx context.Context, key string) bool { return f.marks[key] }
func (f *fakeLimiter) Mark(ctx context.Context, key string, ttl time.Duration) {
	f.marks[key] = true
}
func (f *fakeLimiter) Close() error { return nil }

// --- fake sub-stores ---------------------------------------------------------
// Each embeds the store interface so it satisfies the full contract with zero
// boilerplate; only the methods a handler actually calls are overridden.

type fakeSchematics struct {
	store.SchematicStore
	byName      map[string]*store.Schematic
	stats       []store.SchematicStat
	statsGot    []string
	changes     []store.SchematicChange
	changeSince struct {
		at    time.Time
		name  string
		kind  string
		limit int
	}
	byIDs []store.Schematic
}

func (f *fakeSchematics) GetByName(ctx context.Context, name string) (*store.Schematic, error) {
	return f.byName[name], nil
}
func (f *fakeSchematics) StatsByNames(ctx context.Context, names []string) ([]store.SchematicStat, error) {
	f.statsGot = names
	return f.stats, nil
}
func (f *fakeSchematics) ChangesSince(ctx context.Context, sinceAt time.Time, sinceName, sinceKind string, limit int) ([]store.SchematicChange, error) {
	f.changeSince.at = sinceAt
	f.changeSince.name = sinceName
	f.changeSince.kind = sinceKind
	f.changeSince.limit = limit
	return f.changes, nil
}
func (f *fakeSchematics) ListByIDs(ctx context.Context, ids []string) ([]store.Schematic, error) {
	return f.byIDs, nil
}

type fakeSchematicFiles struct {
	store.SchematicFileStore
	byID map[string]*store.SchematicFile
}

func (f *fakeSchematicFiles) GetByID(ctx context.Context, id string) (*store.SchematicFile, error) {
	return f.byID[id], nil
}

type fakeAPIKeys struct {
	store.APIKeyStore
	byLast8 map[string]*store.APIKey
}

func (f *fakeAPIKeys) GetByLast8(ctx context.Context, last8 string) (*store.APIKey, error) {
	return f.byLast8[last8], nil
}
func (f *fakeAPIKeys) LogUsage(ctx context.Context, apiKeyID, endpoint string) error { return nil }

type fakeViewRatings struct {
	store.ViewRatingStore
	downloads int
}

func (f *fakeViewRatings) RecordDownload(ctx context.Context, schematicID string, userID *string) error {
	f.downloads++
	return nil
}
func (f *fakeViewRatings) GetDownloadCount(ctx context.Context, schematicID string) (int, error) {
	return f.downloads, nil
}

type fakeComments struct {
	store.CommentStore
	bySchematic map[string][]store.Comment
	listCalls   int
}

func (f *fakeComments) ListBySchematic(ctx context.Context, schematicID string) ([]store.Comment, error) {
	f.listCalls++
	return f.bySchematic[schematicID], nil
}

// --- request/event helpers ---------------------------------------------------

// newAPIKey registers a valid API key in the given fake key store and returns
// the plaintext to send in the X-API-Key header.
func newAPIKey(keys *fakeAPIKeys, id string) string {
	plaintext := "apikey-" + id + "-12345678"
	last8 := plaintext[len(plaintext)-8:]
	sum := sha256.Sum256([]byte(plaintext))
	keys.byLast8[last8] = &store.APIKey{ID: id, KeyHash: hex.EncodeToString(sum[:]), Last8: last8}
	return plaintext
}

// newEvent builds a RequestEvent + recorder. pathValues sets stdlib path values
// (what the API handlers read via e.Request.PathValue), apiKey sets the header.
func newEvent(method, target, apiKey string, pathValues map[string]string) (*server.RequestEvent, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(method, target, nil)
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	for k, v := range pathValues {
		req.SetPathValue(k, v)
	}
	rec := httptest.NewRecorder()
	return server.NewRequestEvent(rec, req), rec
}

// publicSchematic returns a minimal published schematic usable by handlers.
func publicSchematic(id, name, file string) *store.Schematic {
	return &store.Schematic{ID: id, Name: name, SchematicFile: file, ModerationState: store.ModerationPublished}
}
