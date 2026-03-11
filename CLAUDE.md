# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

CreateMod.com is a Go web application that serves as a community platform for Minecraft Create mod schematics. It uses server-side rendered Go templates with HTMX for progressive enhancement, PostgreSQL (via pgx + sqlc) for data, chi for HTTP routing, Bleve for full-text search, and Minio/S3 for file storage.

## Development Commands

```bash
# Initial setup
cd ./template && npm install && npm run build && cd ..
go run ./cmd/server/main.go

# Frontend (from ./template directory)
npm run dev           # Vite dev server
npm run build         # Production build (vite build + postcss)
npm run build:css     # CSS only (PostCSS)

# Backend
go run ./cmd/server/main.go

# Testing
go test ./internal/pages/...                       # All page tests
go test -run Test_TrendingScore ./internal/pages/   # Single test by name
go test -v ./...                                    # All tests verbose

# Database: regenerate sqlc after changing queries or migrations
cd ./internal/database && sqlc generate
```

Create `.env` from `.env.example`. Required: `DATABASE_URL` (PostgreSQL connection string). Key variables: `AUTO_MIGRATE=true`, `CREATE_ADMIN=true` (creates a dev admin), `DUMMY_DATA=true` (seed data), `DEV=true`. S3 storage (`S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `S3_BUCKET`) is optional but needed for file features. Main branch is `master`.

## Architecture

### Request Flow

```
HTTP request → chi middleware (legacy compat, security headers, CSRF, cookie auth) → Adapt() → Page handler (server.RequestEvent) → Store query (sqlc/pgx) → Template render → HTML response
```

### Entry Point (`cmd/server/main.go`)

Connects to PostgreSQL, runs database migrations (`database.RunMigrations`), runs River queue migrations, initializes S3 storage (optional), then creates and starts the server.

### Server Initialization (`server/start.go`)

`Server` struct holds all services: `search.Service` (Bleve), `cache.Service` (go-cache), `discord.Service`, `moderation.Service` (OpenAI), `translation.Service` (OpenAI), `session.Store` (PostgreSQL sessions), `storage.Service` (S3/Minio). Services are created in `New()` and passed to handlers via the router's `RegisterParams`.

On boot: builds per-pod search index from the store, warms caches in background goroutines, starts River job worker for periodic tasks (search rebuild, sitemap generation, trending scores, temp upload cleanup).

### Custom Server Framework (`internal/server/`)

`server.RequestEvent` is a drop-in replacement for the former PocketBase `core.RequestEvent`. It wraps `http.ResponseWriter` + `*http.Request` with helper methods (`HTML`, `JSON`, `String`, `Redirect`, `RealIP`, etc.). `server.Registry` handles template loading and rendering. `server.APIError` provides typed HTTP errors.

### Router (`internal/router/main.go`)

Central route registration using chi. `Adapt()` converts `func(e *server.RequestEvent) error` handlers into `http.HandlerFunc`. Creates a `server.Registry` with custom FuncMap (`ToLower`, `mod`, `HumanDate`, `printf`, `T` for i18n, `SignedOutURL`, `LangURL`). Middleware chain: `headMethodSupport` → `requestLogger` → `securityHeaders` → `maintenanceModeMiddleware` → `legacyFileCompat` → `legacySearchCompat` → `legacyCategoryCompat` → `legacyTagCompat` → `cookieAuth` → `csrfOriginCheck`.

### Page Handlers (`internal/pages/`)

**Pattern:** Each handler is a closure factory that captures dependencies and returns a `func(e *server.RequestEvent) error`:

```go
func LoginHandler(registry *server.Registry, appStore *store.Store) func(e *server.RequestEvent) error {
    return func(e *server.RequestEvent) error {
        d := LoginData{}
        d.Populate(e)     // sets auth state, language, categories
        d.Title = "Login"
        html, err := registry.LoadFiles(loginTemplates...).Render(d)
        if err != nil { return err }
        return e.HTML(http.StatusOK, html)
    }
}
```

**Data structs** embed `DefaultData` (defined in `default.go`) which provides `IsAuthenticated`, `Username`, `UserID`, `Language`, `Categories`, `Avatar`, `IsContributor`. Call `d.Populate(e)` to fill from request context. Authentication state comes from `session.UserFromContext(e.Request.Context())`, populated by the `cookieAuth` middleware.

**Template loading:** Each handler defines its templates as `append([]string{pageTemplate}, commonTemplates...)`. Common templates are `head.html`, `sidebar.html`, `header.html`, `footer.html`, `foot.html` (defined in `templates.go`). Page templates live in `template/*.html`, partials in `template/include/`.

**Auth helpers:** `requireAuth(e)` checks authentication and redirects to `/login` if not. `authenticatedUserID(e)` returns the user ID from the session.

### HTMX Integration

HTMX is enabled globally via `hx-boost="true"` on `<body>`. Handlers detect HTMX requests via `e.Request.Header.Get("HX-Request")` and respond differently:
- Normal request: `e.Redirect(http.StatusSeeOther, "/target")`
- HTMX request: set `HX-Redirect` header + return `204 No Content`

### Data Layer

**sqlc + pgx:** SQL queries are defined in `internal/database/queries/*.sql` using sqlc annotations. Running `sqlc generate` (from `internal/database/`) produces type-safe Go code in `internal/database/gen/`. The `sqlc.yaml` config targets PostgreSQL with pgx/v5.

**Store interfaces** (`internal/store/store.go`): Each domain (Users, Schematics, Sessions, etc.) has an interface. Models are plain Go structs. The `Store` struct aggregates all sub-stores for dependency injection.

**PostgreSQL implementation** (`internal/database/postgres.go`): Each interface is implemented by a `*StoreImpl` struct wrapping `*db.Queries`. Conversion functions (e.g., `userFromDB`, `schematicFromDB`) map sqlc-generated types to store models. `NewStoreFromPool(pool)` wires everything together.

**Migrations:** SQL files in `internal/database/migrations/` (e.g., `014_user_webhooks.up.sql`). Run automatically on startup via `database.RunMigrations()` using golang-migrate with embedded FS.

### Authentication (`internal/session/`)

PostgreSQL-backed session store. Sessions are tokens stored in the `sessions` table. The `cookieAuth` middleware validates the session cookie, loads user data, and puts a `SessionUser` into the request context via `session.ContextWithSession`.

### Background Jobs (`internal/jobs/`)

River queue with PostgreSQL backing. Handles periodic tasks: search index rebuild, sitemap generation, trending score computation, temp upload cleanup, AI description generation. Jobs use `UniqueOpts` for deduplication across pods. Started via `startJobWorker()` in `server/start.go`.

### Services

- **Search** (`internal/search/`): Bleve full-text indexing. Per-pod in-memory index built on startup; periodically rebuilt via River job.
- **Cache** (`internal/cache/`): go-cache (per-pod in-memory, 60min default TTL). Used for categories, trending calculations, rendered content.
- **Storage** (`internal/storage/`): Minio/S3 SDK for file storage. Use `StorageService.DownloadRaw(ctx, path)` to read files. Optional — if S3 not configured, file features are unavailable.
- **i18n** (`internal/i18n/`): Translation via `T(lang, key)` template function. Language detected from URL prefix or `Accept-Language` header. Supports: en, pt-BR, pt-PT, es, de, pl, ru, zh-Hans, fr.
- **Discord** (`internal/discord/`): Webhook notifications. `Post()` sends to site webhook; `PostWithUserWebhooks()` also sends to all active user webhooks.
- **Webhook** (`internal/webhook/`): AES-256-GCM encryption/decryption for user webhook URLs; Discord webhook URL validation.

### Frontend

Vite builds from `template/src/` to `template/dist/`. CSS uses Tailwind + PurgeCSS via PostCSS. Static libraries (TinyMCE, Tom Select, Plyr, Masonry, fslightbox) served from `template/static/`. UI framework is Tabler (Bootstrap-based).

## Common Pitfalls

**PostgreSQL required:** The server will not start without a valid `DATABASE_URL`. Migrations run automatically on boot.

**sqlc regeneration:** After modifying SQL queries in `internal/database/queries/` or migration files in `internal/database/migrations/`, run `cd internal/database && sqlc generate` to regenerate Go code.

**Store interface compliance:** When adding a new store, add a compile-time check (`var _ store.NewInterface = (*NewStoreImpl)(nil)`) in `postgres.go` and wire it into `NewStoreFromPool()`.

**Settings sidebar duplication:** The settings sidebar is duplicated across all settings page templates (`user-settings.html`, `user-api-keys.html`, `user-webhooks.html`, `user-points.html`, `user-statistics.html`, `user-password.html`, `blacklist_request.html`). When adding a new settings page, update the sidebar in all of them.

## Testing Patterns

**Template tests** (`*_template_test.go`): Render templates with test data and assert HTML output (presence of elements, attributes, text content).

**HTTP tests** (`*_http_test.go`): Use `testutil.NewTestServer(t)` which provides a minimal HTTP server simulating key endpoints without booting the full server. Use `testutil.WithHTMX(req)` to add HTMX headers. Use `testutil.NewHTTPClient(t)` for stateful requests with cookie jars.

**Unit tests** (e.g., `trending_test.go`): Direct function testing for business logic.

**Playwright E2E tests** (`tests/e2e/specs/`): Run via GitHub Actions against Docker containers. Not configured for local execution.

## Production Deployment (pre-big-deployment baseline, 2026-03-09)

Last known-good production state before the CI/CD consolidation and dependency updates. Use this to revert if a future deployment causes issues.

- **Image:** `ghcr.io/uberswe/createmod.com:master-2a18cbb`
- **Namespace:** `createmod-com-prod`
- **Resource:** `StatefulSet/createmod`
- **Replicas:** 1
- **Resources:** CPU 200m-1000m, Memory 2Gi-16Gi
- **Storage:** 20Gi PVC (`local-path` StorageClass)
- **Port:** 8090
- **Health:** `/api/health` (liveness: 10s init, 30s period; readiness: 5s init, 10s period)
- **Annotations:** `linkerd.io/inject: enabled`, `config.linkerd.io/skip-outbound-ports: "443"`

To revert production, update `k8s/createmod/prod/deployment.yaml` image to `ghcr.io/uberswe/createmod.com:master-2a18cbb` and apply.

**Dev baseline:** `ghcr.io/uberswe/createmod.com:development-4fbe4d6` in namespace `createmod-com-dev`.
