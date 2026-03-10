<p align="center">
  <a href="https://createmod.com">
    <img src="https://createmod.com/assets/x/logo.png" alt="CreateMod.com" width="120" />
  </a>
</p>

<h1 align="center">CreateMod.com</h1>

<p align="center">
  Community platform for sharing and discovering Minecraft <a href="https://modrinth.com/mod/create">Create mod</a> schematics.
</p>

<p align="center">
  <a href="https://github.com/uberswe/createmod.com/actions/workflows/ci.yml">
    <img src="https://github.com/uberswe/createmod.com/actions/workflows/ci.yml/badge.svg" alt="CI" />
  </a>
  <a href="https://github.com/uberswe/createmod.com/blob/master/LICENSE.md">
    <img src="https://img.shields.io/github/license/uberswe/createmod.com" alt="License" />
  </a>
  <img src="https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white" alt="Go 1.25" />
  <img src="https://img.shields.io/badge/PostgreSQL-17-4169E1?logo=postgresql&logoColor=white" alt="PostgreSQL 17" />
  <a href="https://github.com/uberswe/createmod.com/stargazers">
    <img src="https://img.shields.io/github/stars/uberswe/createmod.com?style=flat" alt="Stars" />
  </a>
</p>

---

## Overview

CreateMod.com is a Go web application that serves as a community hub for Minecraft Create mod schematics. Users can upload, browse, search and download `.nbt` schematic files, explore mod metadata, watch Create mod videos, and read guides.

**Key technologies:**

- **Go** with server-side rendered HTML templates and [HTMX](https://htmx.org/) for progressive enhancement
- **PostgreSQL** for primary data storage
- **Bleve** for full-text schematic search
- **S3-compatible storage** (MinIO) for schematic files
- **Vite + Tailwind CSS** for frontend assets
- **Docker** for local development

## Getting Started

### Prerequisites

- Go 1.25+
- PostgreSQL 17
- Node.js (for frontend assets)
- Redis (optional, for multi-pod caching)

### Quick Start

```bash
# 1. Clone and configure
git clone https://github.com/uberswe/createmod.com.git
cd createmod.com
cp .env.example .env

# 2. Build frontend assets
cd template && npm install && npm run build && cd ..

# 3. Run the server
go run ./cmd/server/main.go serve
```

The application will be available at [http://localhost:8080](http://localhost:8080).

### Docker

For a fully containerized setup:

```bash
docker compose up
```

Rebuild after Go code changes:

```bash
docker compose up --build
```

## Configuration

Create a `.env` file from `.env.example`. Key variables:

| Variable | Description | Default |
|---|---|---|
| `DATABASE_URL` | PostgreSQL connection string | `postgres://createmod:localdev@localhost:5432/createmod?sslmode=disable` |
| `AUTO_MIGRATE` | Auto-generate migrations from Admin UI changes | `true` |
| `CREATE_ADMIN` | Create a dev admin account on startup | `true` |
| `DUMMY_DATA` | Seed with sample data (**deletes `pb_data/`**) | `true` |
| `DEV` | Enable development mode | `true` |
| `S3_ENDPOINT` | S3-compatible storage endpoint | `localhost:9000` |
| `DISCORD_WEBHOOK_URL` | Discord webhook for notifications | |
| `OPENAI_API_KEY` | OpenAI API key for AI descriptions | |

See `.env.example` for the full list including OAuth, SMTP, and S3 settings.

## Development

```bash
# Frontend dev server (hot reload)
cd template && npm run dev

# Run all tests
go test ./...

# Run specific tests
go test ./internal/pages/...
go test -run Test_TrendingScore ./internal/pages/
```

Admin UI is available at `/_/` when the server is running. Default dev credentials: `local@createmod.com` / `jfq.utb*jda2abg!WCR`.

## Project Structure

```
cmd/server/          # Application entrypoint
internal/
  pages/             # HTTP handlers (one file per page)
  models/            # Data models and mapping
  search/            # Bleve full-text search
  cache/             # Caching layer
  i18n/              # Internationalization (en, de, es, pl, pt-BR, pt-PT, ru, zh-Hans)
  router/            # Route registration and middleware
  modmeta/           # Mod metadata enrichment (Modrinth, CurseForge, BlocksItems)
  nbtparser/         # Minecraft NBT schematic parser
  store/             # Database access layer
  migrate/           # Data migration utilities
migrations/          # Database migrations
template/            # HTML templates, Vite frontend, static assets
server/              # Server initialization and lifecycle hooks
k8s/                 # Kubernetes deployment manifests
tests/e2e/           # Playwright end-to-end tests
```

## Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on submitting issues and pull requests.

## License

This project is licensed under the MIT License. See [LICENSE.md](LICENSE.md) for details.
