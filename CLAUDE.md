# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

CreateMod.com is a Go web application built on PocketBase (v0.29.3, Go 1.24) that serves as a community platform for Minecraft Create mod schematics. It uses server-side rendered Go templates with HTMX for progressive enhancement, PocketBase as both database (SQLite) and backend framework, and Bleve for full-text search.

## Development Commands

```bash
# Initial setup
cd ./template && npm install && npm run build && cd ..
go run ./cmd/server/main.go serve

# Frontend (from ./template directory)
npm run dev           # Vite dev server
npm run build         # Production build (vite build + postcss)
npm run build:css     # CSS only (PostCSS)

# Backend
go run ./cmd/server/main.go serve

# Testing
go test ./internal/pages/...                       # All page tests
go test -run Test_TrendingScore ./internal/pages/   # Single test by name
go test -v ./...                                    # All tests verbose
```

Create `.env` from `.env.example`. Key variables: `AUTO_MIGRATE=true` (auto-generate migrations from Admin UI changes), `CREATE_ADMIN=true` (creates a dev admin account — see `.env.example` for credentials), `DUMMY_DATA=true` (seed data, WARNING: deletes `pb_data/`), `DEV=true`.

Admin UI at `/_/` when server is running. Main branch is `master`.

## Architecture

### Request Flow

```
Request → Router middleware (legacy compat, cookie auth) → Page handler → PocketBase query → Model mapping → Template render → HTML response
```

### Server Initialization (`server/start.go`)

`Server` struct holds all services: `search.Service` (Bleve), `cache.Service` (go-cache), `sitemap.Service`, `discord.Service`, `moderation.Service` (OpenAI), `aidescription.Service` (OpenAI). Services are created in `New()` and passed to handlers via the router.

On boot (`OnServe` hook): indexes all approved schematics into Bleve, generates sitemaps, starts AI description scheduler (30min poll). PocketBase lifecycle hooks in `start.go` handle schematic creation validation (NBT files), search index updates, version snapshots on updates, and achievement/points awarding.

### Router (`internal/router/main.go`)

Central route registration. Creates a `template.Registry` with custom FuncMap (`ToLower`, `mod`, `HumanDate`, `printf`, `T` for i18n). Middleware chain: `legacyFileCompat` → `legacySearchCompat` → `legacyCategoryCompat` → `legacyTagCompat` → `cookieAuth`. Cookie auth extracts PocketBase auth token from cookies and populates `e.Auth`.

### Page Handlers (`internal/pages/`)

**Pattern:** Each handler is a closure factory that captures dependencies and returns a `func(e *core.RequestEvent) error`:

```go
func LoginHandler(app *pocketbase.PocketBase, registry *template.Registry) func(e *core.RequestEvent) error {
    return func(e *core.RequestEvent) error {
        d := LoginData{}
        d.Populate(e)     // sets auth state, language, categories
        d.Title = "Login"
        html, err := registry.LoadFiles(loginTemplates...).Render(d)
        if err != nil { return err }
        return e.HTML(http.StatusOK, html)
    }
}
```

**Data structs** embed `DefaultData` (defined in `default.go`) which provides `IsAuthenticated`, `Username`, `UserID`, `Language`, `Categories`, `Avatar`, `IsContributor`. Call `d.Populate(e)` to fill from request context.

**Template loading:** Each handler defines its templates as `append([]string{pageTemplate}, commonTemplates...)`. Common templates are `head.html`, `sidebar.html`, `header.html`, `footer.html`, `foot.html` (defined in `templates.go`). Page templates live in `template/*.html`, partials in `template/include/`.

### HTMX Integration

HTMX is enabled globally via `hx-boost="true"` on `<body>`. Handlers detect HTMX requests via `e.Request.Header.Get("HX-Request")` and respond differently:
- Normal request: `e.Redirect(http.StatusFound, "/target")`
- HTMX request: set `HX-Redirect` header + return `204 No Content`

Key HTMX patterns in templates: `hx-post` with `hx-target="body"` and `hx-push-url="true"` for form submissions, `hx-select` for partial page updates (e.g., pagination), `hx-vals='js:{...}'` for dynamic values.

### Models (`internal/models/`)

PocketBase records are mapped to typed Go structs. `Schematic` is the primary content type. `DatabaseSchematic` is a flattened variant for efficient queries. Mapping functions like `mapResultToSchematic()` convert PocketBase records to presentation models, keeping database concerns out of templates.

### Migrations (`migrations/`)

Go-based PocketBase migrations, auto-imported via `_ "createmod/migrations"` in `server/start.go`. If `AUTO_MIGRATE=true`, PocketBase Admin UI changes generate migration files automatically.

### Services

- **Search** (`internal/search/`): Bleve full-text indexing. Reindexed on boot; updated async via PocketBase hooks on schematic create/update.
- **Cache** (`internal/cache/`): go-cache with 60min default TTL. Used for categories, trending calculations, rendered content.
- **i18n** (`internal/i18n/`): Translation via `T(lang, key)` template function. Language detected from `Accept-Language` header per request. Supports: en, pt-BR, pt-PT, es, de, pl, ru, zh-Hans.
- **NBT Parser** (`internal/nbtparser/`): Validates and extracts stats from Minecraft NBT schematic files.

### Frontend

Vite builds from `template/src/` to `template/dist/`. CSS uses Tailwind + PurgeCSS via PostCSS. Static libraries (TinyMCE, PocketBase JS SDK, Tom Select, Plyr, Masonry, fslightbox) served from `template/static/`. UI framework is Tabler (Bootstrap-based).

## Common Pitfalls

**Running the server binary from the wrong directory:** PocketBase determines its data directory (`pb_data/`) relative to the working directory. If you build to `/tmp/createmod` and run it from there, PocketBase will create a fresh empty `pb_data/` in `/tmp` instead of using the project's database. Always run the server from the project root, or pass `--dir=./pb_data` explicitly. Symptom: the site loads but shows zero schematics/content with no errors.

**Schematic files are stored via S3:** Both locally and in production, schematic files are stored through PocketBase's S3-compatible filesystem, not on local disk. Use `app.NewFilesystem()` + `fsys.GetReader()` to read files, not `os.ReadFile()`. Use `filesystem.NewFileFromBytes()` to write files back.

## Testing Patterns

**Template tests** (`*_template_test.go`): Render templates with test data and assert HTML output (presence of elements, attributes, text content).

**HTTP tests** (`*_http_test.go`): Use `testutil.NewTestServer(t)` which provides a minimal HTTP server simulating key endpoints without booting PocketBase. Use `testutil.WithHTMX(req)` to add HTMX headers. Use `testutil.NewHTTPClient(t)` for stateful requests with cookie jars.

**Unit tests** (e.g., `trending_test.go`): Direct function testing for business logic.

**Playwright E2E tests** (`tests/e2e/specs/`): Run via GitHub Actions against Docker containers. Not configured for local execution.
