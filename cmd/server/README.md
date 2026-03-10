# CreateMod.com Server

A Go web application for the Minecraft Create mod community -- browse, upload, and share schematics.

## Quick Start (Docker Compose)

```bash
docker compose up --build
```

This starts all services:

| Service | URL | Notes |
|---------|-----|-------|
| App | http://localhost:8091 | Main application |
| MinIO Console | http://localhost:9001 | S3 storage UI (`minioadmin` / `minioadmin`) |
| MailHog | http://localhost:8025 | Catches outgoing emails |
| PostgreSQL | `localhost:5432` | `createmod` / `localdev` / `createmod` |

Other useful commands:

```bash
docker compose up -d            # run in background
docker compose logs -f app      # follow app logs
docker compose down             # stop everything
docker compose down -v          # stop and delete volumes (wipes data)
```

### Migrating from PocketBase/SQLite

## Local Development (without Docker)

### Prerequisites

- Go 1.24+
- Node.js 22+
- PostgreSQL 17+
- MinIO or S3-compatible storage (optional, needed for file uploads/images)

### Setup

```bash
# 1. Configure environment
cp .env.example .env
# Edit .env -- at minimum set DATABASE_URL

# 2. Build frontend assets
cd template && npm install && npm run build && cd ..

# 3. Run the server
go run ./cmd/server/main.go
```

The server listens on `:8090` by default (override with `PORT` or `LISTEN_ADDR`).

### Frontend Development

```bash
cd template
npm run dev     # Vite dev server with hot reload
npm run build   # Production build (Vite + PostCSS)
```

### Testing

```bash
go test ./internal/pages/...                       # Page handler tests
go test -run Test_TrendingScore ./internal/pages/   # Single test
go test -v ./...                                    # All tests
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | | PostgreSQL connection string |
| `S3_ENDPOINT` | No | | MinIO/S3 endpoint (e.g. `localhost:9000`) |
| `S3_ACCESS_KEY` | No | | S3 access key |
| `S3_SECRET_KEY` | No | | S3 secret key |
| `S3_BUCKET` | No | | S3 bucket name |
| `S3_USE_SSL` | No | `false` | Use HTTPS for S3 |
| `DEV` | No | `false` | Development mode |
| `CREATE_ADMIN` | No | `false` | Create dev admin on startup |
| `PORT` | No | `8090` | Listen port |
| `LISTEN_ADDR` | No | `:8090` | Full listen address (overrides `PORT`) |
| `BASE_URL` | No | | Public URL for OAuth callbacks and sitemaps |
| `DISCORD_CLIENT_ID` | No | | Discord OAuth client ID |
| `DISCORD_CLIENT_SECRET` | No | | Discord OAuth client secret |
| `GITHUB_CLIENT_ID` | No | | GitHub OAuth client ID |
| `GITHUB_CLIENT_SECRET` | No | | GitHub OAuth client secret |
| `DISCORD_WEBHOOK_URL` | No | | Discord webhook for notifications |
| `OPENAI_API_KEY` | No | | OpenAI API key (moderation, AI descriptions) |
| `CURSEFORGE_API_KEY` | No | | CurseForge API key (mod metadata) |
| `MAINTENANCE_MODE` | No | `false` | Force maintenance page on all routes |
| `SQLITE_PATH` | No | `./pb_data/data.db` | Path to PocketBase SQLite DB for auto-migration |
| `SMTP_HOST` | No | | SMTP server host |
| `SMTP_PORT` | No | | SMTP server port |
