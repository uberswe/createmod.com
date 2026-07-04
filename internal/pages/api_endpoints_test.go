package pages

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"createmod/internal/cache"
	"createmod/internal/models"
	"createmod/internal/search"
	"createmod/internal/server"
	"createmod/internal/store"
)

// ---------------------------------------------------------------------------
// GET /api/schematics/stats  (bulk stats, public)
// ---------------------------------------------------------------------------

func TestAPIBulkStats(t *testing.T) {
	sch := &fakeSchematics{stats: []store.SchematicStat{
		{Name: "alpha", Views: 10, Downloads: 3, AvgRating: 4.5, RatingCount: 2, CommentCount: 1},
		{Name: "beta", Views: 20, Downloads: 6, AvgRating: 3.0, RatingCount: 4, CommentCount: 0},
	}}
	appStore := &store.Store{Schematics: sch}
	h := APISchematicBulkStatsHandler(appStore)

	t.Run("missing names -> 400", func(t *testing.T) {
		e, rec := newEvent("GET", "/api/schematics/stats", "", nil)
		_ = h(e)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("want 400, got %d", rec.Code)
		}
	})

	t.Run("happy path returns stats array", func(t *testing.T) {
		e, rec := newEvent("GET", "/api/schematics/stats?names=alpha,beta", "", nil)
		if err := h(e); err != nil {
			t.Fatalf("handler err: %v", err)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("want 200, got %d", rec.Code)
		}
		var out []apiBulkStatItem
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("bad json: %v", err)
		}
		if len(out) != 2 {
			t.Fatalf("want 2 items, got %d", len(out))
		}
		if out[0].Name != "alpha" || out[0].CommentCount != 1 || out[0].Rating != 4.5 {
			t.Fatalf("unexpected first item: %+v", out[0])
		}
	})

	t.Run("names capped at 100", func(t *testing.T) {
		names := make([]string, 150)
		for i := range names {
			names[i] = fmt.Sprintf("s%d", i)
		}
		e, _ := newEvent("GET", "/api/schematics/stats?names="+strings.Join(names, ","), "", nil)
		_ = h(e)
		if len(sch.statsGot) != 100 {
			t.Fatalf("expected names capped to 100, store got %d", len(sch.statsGot))
		}
	})
}

// ---------------------------------------------------------------------------
// GET /api/schematics/changes  (public, keyset cursor)
// ---------------------------------------------------------------------------

func TestAPIChanges(t *testing.T) {
	t.Run("no cursor returns current cursor and empty list", func(t *testing.T) {
		appStore := &store.Store{Schematics: &fakeSchematics{}}
		e, rec := newEvent("GET", "/api/schematics/changes", "", nil)
		if err := APISchematicChangesHandler(appStore)(e); err != nil {
			t.Fatalf("err: %v", err)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("want 200, got %d", rec.Code)
		}
		var out apiChangesResponse
		json.Unmarshal(rec.Body.Bytes(), &out)
		if len(out.Changes) != 0 || out.Cursor == "" || out.HasMore {
			t.Fatalf("unexpected response: %+v", out)
		}
		if _, _, _, ok := decodeChangesCursor(out.Cursor); !ok {
			t.Fatalf("returned cursor not decodable: %q", out.Cursor)
		}
	})

	t.Run("invalid cursor -> 400", func(t *testing.T) {
		appStore := &store.Store{Schematics: &fakeSchematics{}}
		e, rec := newEvent("GET", "/api/schematics/changes?cursor=%21%21bad", "", nil)
		_ = APISchematicChangesHandler(appStore)(e)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("want 400, got %d", rec.Code)
		}
	})

	t.Run("decodes cursor and passes keyset to store", func(t *testing.T) {
		at := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
		sch := &fakeSchematics{changes: []store.SchematicChange{
			{Name: "gearbox", Kind: "updated", At: at.Add(time.Minute)},
			{Name: "press", Kind: "removed", At: at.Add(2 * time.Minute)},
		}}
		appStore := &store.Store{Schematics: sch}
		cursor := encodeChangesCursor(at, "anvil", "updated")
		e, rec := newEvent("GET", "/api/schematics/changes?cursor="+cursor, "", nil)
		if err := APISchematicChangesHandler(appStore)(e); err != nil {
			t.Fatalf("err: %v", err)
		}
		if !sch.changeSince.at.Equal(at) || sch.changeSince.name != "anvil" || sch.changeSince.kind != "updated" {
			t.Fatalf("store got wrong keyset: %+v", sch.changeSince)
		}
		var out apiChangesResponse
		json.Unmarshal(rec.Body.Bytes(), &out)
		if len(out.Changes) != 2 || out.HasMore {
			t.Fatalf("unexpected changes: %+v", out)
		}
		// Next cursor must point at the last returned row.
		gotAt, gotName, gotKind, ok := decodeChangesCursor(out.Cursor)
		if !ok || gotName != "press" || gotKind != "removed" || !gotAt.Equal(at.Add(2*time.Minute)) {
			t.Fatalf("next cursor wrong: at=%v name=%q kind=%q", gotAt, gotName, gotKind)
		}
	})

	t.Run("hasMore true when over limit", func(t *testing.T) {
		big := make([]store.SchematicChange, 501)
		base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		for i := range big {
			big[i] = store.SchematicChange{Name: fmt.Sprintf("s%d", i), Kind: "updated", At: base.Add(time.Duration(i) * time.Second)}
		}
		sch := &fakeSchematics{changes: big}
		appStore := &store.Store{Schematics: sch}
		cursor := encodeChangesCursor(base, "", "")
		e, rec := newEvent("GET", "/api/schematics/changes?cursor="+cursor, "", nil)
		_ = APISchematicChangesHandler(appStore)(e)
		var out apiChangesResponse
		json.Unmarshal(rec.Body.Bytes(), &out)
		if !out.HasMore || len(out.Changes) != 500 {
			t.Fatalf("want hasMore + 500 items, got hasMore=%v len=%d", out.HasMore, len(out.Changes))
		}
		if sch.changeSince.limit != 501 {
			t.Fatalf("handler should request limit+1 (501), got %d", sch.changeSince.limit)
		}
	})

	t.Run("legacy RFC3339 cursor still accepted", func(t *testing.T) {
		sch := &fakeSchematics{}
		appStore := &store.Store{Schematics: sch}
		e, rec := newEvent("GET", "/api/schematics/changes?cursor=2026-01-02T03:04:05Z", "", nil)
		if err := APISchematicChangesHandler(appStore)(e); err != nil {
			t.Fatalf("err: %v", err)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("want 200 for legacy cursor, got %d", rec.Code)
		}
		want := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
		if !sch.changeSince.at.Equal(want) || sch.changeSince.name != "" {
			t.Fatalf("legacy cursor decoded wrong: %+v", sch.changeSince)
		}
	})
}

// ---------------------------------------------------------------------------
// GET /api/schematics/{name}/download  (also the /api/download/{name} alias)
// ---------------------------------------------------------------------------

func downloadDeps() (*store.Store, *fakeAPIKeys, *fakeViewRatings, string) {
	keys := &fakeAPIKeys{byLast8: map[string]*store.APIKey{}}
	apiKey := newAPIKey(keys, "k1")
	vr := &fakeViewRatings{}
	appStore := &store.Store{
		APIKeys:     keys,
		ViewRatings: vr,
		Schematics: &fakeSchematics{byName: map[string]*store.Schematic{
			"gearbox": publicSchematic("id-gear", "gearbox", "gearbox.nbt"),
			"hidden":  {ID: "id-h", Name: "hidden", SchematicFile: "h.nbt", ModerationState: store.ModerationRejected},
			"nofile":  publicSchematic("id-nf", "nofile", ""),
		}},
		SchematicFiles: &fakeSchematicFiles{byID: map[string]*store.SchematicFile{
			"var1": {ID: "var1", SchematicID: "id-gear", Filename: "variant.nbt"},
		}},
	}
	return appStore, keys, vr, apiKey
}

func TestAPIDownload(t *testing.T) {
	t.Run("401 without auth", func(t *testing.T) {
		appStore, _, _, _ := downloadDeps()
		h := APISchematicDownloadHandler(newFakeLimiter(), cache.New(), appStore, testModSecret)
		e, rec := newEvent("GET", "/api/schematics/gearbox/download", "", map[string]string{"name": "gearbox"})
		_ = h(e)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("want 401, got %d", rec.Code)
		}
	})

	t.Run("404 for non-public schematic", func(t *testing.T) {
		appStore, _, vr, key := downloadDeps()
		h := APISchematicDownloadHandler(newFakeLimiter(), cache.New(), appStore, testModSecret)
		e, rec := newEvent("GET", "/api/schematics/hidden/download", key, map[string]string{"name": "hidden"})
		_ = h(e)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("want 404, got %d", rec.Code)
		}
		if vr.downloads != 0 {
			t.Fatalf("non-public download must not be counted, got %d", vr.downloads)
		}
	})

	t.Run("302 redirect and counts on success", func(t *testing.T) {
		appStore, _, vr, key := downloadDeps()
		h := APISchematicDownloadHandler(newFakeLimiter(), cache.New(), appStore, testModSecret)
		e, rec := newEvent("GET", "/api/schematics/gearbox/download", key, map[string]string{"name": "gearbox"})
		_ = h(e)
		if rec.Code != http.StatusFound {
			t.Fatalf("want 302, got %d", rec.Code)
		}
		if loc := rec.Header().Get("Location"); loc != "/api/files/schematics/id-gear/gearbox.nbt" {
			t.Fatalf("unexpected redirect: %q", loc)
		}
		if vr.downloads != 1 {
			t.Fatalf("successful download should count once, got %d", vr.downloads)
		}
	})

	t.Run("variation ?f= redirects and counts", func(t *testing.T) {
		appStore, _, vr, key := downloadDeps()
		h := APISchematicDownloadHandler(newFakeLimiter(), cache.New(), appStore, testModSecret)
		e, rec := newEvent("GET", "/api/schematics/gearbox/download?f=var1", key, map[string]string{"name": "gearbox"})
		_ = h(e)
		if rec.Code != http.StatusFound {
			t.Fatalf("want 302, got %d", rec.Code)
		}
		if loc := rec.Header().Get("Location"); loc != "/api/files/schematics/id-gear/variant.nbt" {
			t.Fatalf("unexpected variation redirect: %q", loc)
		}
		if vr.downloads != 1 {
			t.Fatalf("variation download should count once, got %d", vr.downloads)
		}
	})

	t.Run("bad ?f= is 404 and does NOT count (ordering fix)", func(t *testing.T) {
		appStore, _, vr, key := downloadDeps()
		h := APISchematicDownloadHandler(newFakeLimiter(), cache.New(), appStore, testModSecret)
		e, rec := newEvent("GET", "/api/schematics/gearbox/download?f=nope", key, map[string]string{"name": "gearbox"})
		_ = h(e)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("want 404, got %d", rec.Code)
		}
		if vr.downloads != 0 {
			t.Fatalf("failed variation lookup must not inflate the counter, got %d", vr.downloads)
		}
	})

	t.Run("missing file yields 404", func(t *testing.T) {
		appStore, _, _, key := downloadDeps()
		h := APISchematicDownloadHandler(newFakeLimiter(), cache.New(), appStore, testModSecret)
		e, rec := newEvent("GET", "/api/schematics/nofile/download", key, map[string]string{"name": "nofile"})
		_ = h(e)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("want 404, got %d", rec.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// GET /api/schematics/{name}/comments
// ---------------------------------------------------------------------------

func TestAPIComments(t *testing.T) {
	keys := &fakeAPIKeys{byLast8: map[string]*store.APIKey{}}
	apiKey := newAPIKey(keys, "k1")
	appStore := &store.Store{
		APIKeys:  keys,
		Comments: &fakeComments{bySchematic: map[string][]store.Comment{}},
		Schematics: &fakeSchematics{byName: map[string]*store.Schematic{
			"gearbox": publicSchematic("id-gear", "gearbox", "g.nbt"),
			"hidden":  {ID: "id-h", Name: "hidden", ModerationState: store.ModerationRejected},
		}},
	}
	h := APISchematicCommentsHandler(newFakeLimiter(), cache.New(), appStore, testModSecret)

	t.Run("401 without auth", func(t *testing.T) {
		e, rec := newEvent("GET", "/api/schematics/gearbox/comments", "", map[string]string{"name": "gearbox"})
		_ = h(e)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("want 401, got %d", rec.Code)
		}
	})

	t.Run("404 for non-public schematic", func(t *testing.T) {
		e, rec := newEvent("GET", "/api/schematics/hidden/comments", apiKey, map[string]string{"name": "hidden"})
		_ = h(e)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("want 404, got %d", rec.Code)
		}
	})

	t.Run("happy path returns comment envelope", func(t *testing.T) {
		e, rec := newEvent("GET", "/api/schematics/gearbox/comments", apiKey, map[string]string{"name": "gearbox"})
		if err := h(e); err != nil {
			t.Fatalf("err: %v", err)
		}
		if rec.Code != http.StatusOK {
			t.Fatalf("want 200, got %d", rec.Code)
		}
		var out apiCommentsResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("bad json: %v", err)
		}
		if out.Comments == nil {
			t.Fatalf("comments should be a non-nil array")
		}
	})
}

// TestAPICommentsCaching proves the second identical request is served from the
// per-pod cache and does not re-hit the store.
func TestAPICommentsCaching(t *testing.T) {
	keys := &fakeAPIKeys{byLast8: map[string]*store.APIKey{}}
	apiKey := newAPIKey(keys, "k1")
	comments := &fakeComments{bySchematic: map[string][]store.Comment{"warp": nil}}
	appStore := &store.Store{
		APIKeys:    keys,
		Comments:   comments,
		Schematics: &fakeSchematics{byName: map[string]*store.Schematic{"warp": publicSchematic("id-w", "warp", "w.nbt")}},
	}
	h := APISchematicCommentsHandler(newFakeLimiter(), cache.New(), appStore, testModSecret)

	call := func() *httptest.ResponseRecorder {
		e, rec := newEvent("GET", "/api/schematics/warp/comments", apiKey, map[string]string{"name": "warp"})
		_ = h(e)
		return rec
	}

	first := call()
	if got := first.Header().Get("X-Cache"); got != "MISS" {
		t.Fatalf("first call want X-Cache MISS, got %q", got)
	}
	second := call()
	if got := second.Header().Get("X-Cache"); got != "HIT" {
		t.Fatalf("second call want X-Cache HIT, got %q", got)
	}
	if comments.listCalls != 1 {
		t.Fatalf("store should be hit once (second served from cache), got %d calls", comments.listCalls)
	}
	if first.Body.String() != second.Body.String() {
		t.Fatalf("cached body differs from origin body")
	}
}

// ---------------------------------------------------------------------------
// Query-param parsing (shared by list/search/home)
// ---------------------------------------------------------------------------

func TestParseAPIPerPage(t *testing.T) {
	cases := map[string]int{"": 24, "8": 8, "16": 16, "24": 24, "32": 32, "64": 64, "100": 100, "50": 24, "abc": 24, "-5": 24, "1000": 24}
	for in, want := range cases {
		if got := parseAPIPerPage(in); got != want {
			t.Errorf("parseAPIPerPage(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestParseAPISearchQuery(t *testing.T) {
	appStore := &store.Store{}
	cacheSvc := cache.New()

	t.Run("empty term & no sort -> trending", func(t *testing.T) {
		e, _ := newEvent("GET", "/api/schematics", "", nil)
		if q := parseAPISearchQuery(e, appStore, cacheSvc); q.Order != search.TrendingOrder {
			t.Fatalf("want trending order, got %d", q.Order)
		}
	})

	t.Run("term present -> best match order and normalized term", func(t *testing.T) {
		e, _ := newEvent("GET", "/api/schematics?query=foo-bar", "", nil)
		q := parseAPISearchQuery(e, appStore, cacheSvc)
		if q.Term != "foo bar" {
			t.Fatalf("term not normalized: %q", q.Term)
		}
		if q.Order != search.BestMatchOrder {
			t.Fatalf("want best-match order, got %d", q.Order)
		}
	})

	t.Run("valid sort honored", func(t *testing.T) {
		e, _ := newEvent("GET", "/api/schematics?sort=2", "", nil)
		if q := parseAPISearchQuery(e, appStore, cacheSvc); q.Order != 2 {
			t.Fatalf("want sort 2, got %d", q.Order)
		}
	})

	t.Run("invalid sort ignored (falls back to trending on empty term)", func(t *testing.T) {
		for _, bad := range []string{"99", "0", "abc", "-1"} {
			e, _ := newEvent("GET", "/api/schematics?sort="+bad, "", nil)
			if q := parseAPISearchQuery(e, appStore, cacheSvc); q.Order != search.TrendingOrder {
				t.Fatalf("sort=%q should be ignored -> trending, got %d", bad, q.Order)
			}
		}
	})

	t.Run("rating validated to 0..5", func(t *testing.T) {
		if q := parseAPISearchQuery(mustEvent("/api/schematics?rating=3"), appStore, cacheSvc); q.Rating != 3 {
			t.Fatalf("rating=3 want 3, got %d", q.Rating)
		}
		for _, bad := range []string{"10", "-1", "6", "abc"} {
			if q := parseAPISearchQuery(mustEvent("/api/schematics?rating="+bad), appStore, cacheSvc); q.Rating != -1 {
				t.Fatalf("rating=%q should be ignored (-1), got %d", bad, q.Rating)
			}
		}
	})
}

func mustEvent(target string) *server.RequestEvent {
	e, _ := newEvent("GET", target, "", nil)
	return e
}

// commentWithInternals returns a comment with every field populated, so the JSON
// hiding test can assert internal fields don't leak.
func commentWithInternals() models.Comment {
	return models.Comment{
		ID:              "c1",
		Created:         "2026-01-01",
		Published:       "2026-01-02",
		Author:          "internal-author-slug",
		AuthorID:        "user-secret-id",
		AuthorUsername:  "steve",
		AuthorHasAvatar: true,
		AuthorAvatar:    "/avatar/steve.png",
		Indent:          0,
		Content:         "hello world",
		OriginalContent: "hello world original",
		Approved:        true,
		ParentID:        "",
		ReplyToAuthor:   "",
		IsTranslated:    false,
	}
}

// ---------------------------------------------------------------------------
// models.Comment JSON must hide internal fields (security regression guard)
// ---------------------------------------------------------------------------

func TestCommentJSONHidesInternalFields(t *testing.T) {
	c := commentWithInternals()
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	for _, hidden := range []string{"AuthorID", "OriginalContent", "Approved", "IsTranslated", "AuthorHasAvatar", `"Author"`, "Created"} {
		if strings.Contains(s, hidden) {
			t.Errorf("comment JSON leaked internal field %s: %s", hidden, s)
		}
	}
	for _, want := range []string{"AuthorUsername", "AuthorAvatar", "Published", "Content"} {
		if !strings.Contains(s, want) {
			t.Errorf("comment JSON missing public field %s: %s", want, s)
		}
	}
}
