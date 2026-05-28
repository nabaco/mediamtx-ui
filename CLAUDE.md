# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build everything (frontend then Go binary)
make                  # → bin/mediamtx-ui

# Development (hot-reload frontend + Go backend in parallel)
make dev              # Vite on :5173, Go on :9996

# Individual steps
make build-frontend   # npm ci + vite build → internal/web/dist
make build-backend    # go build → bin/mediamtx-ui

# Tests and linting
make test             # go test ./...
make lint             # golangci-lint run

# Version management
make bump-patch       # 0.2.0 → 0.2.1
make bump-minor       # 0.2.0 → 0.3.0

# Docker
make docker-build
```

**Frontend only** (from `frontend/`):
```bash
npm run dev     # Vite dev server with proxy to :9996
npm run build   # produces frontend/dist/
```

**Key dev detail**: `make dev` sets `MEDIAMTX_UI_DEV_PROXY=http://localhost:5173`, which makes the Go backend reverse-proxy all non-API traffic to Vite instead of serving embedded assets. The Vite dev server proxies `/api` and `/metrics` back to `:9996`.

## Architecture

Single Go binary with the Svelte SPA embedded via `go:embed`. Runs on `:9996` alongside a mediamtx instance (API on `:9997`).

### Backend layout (`internal/`)

| Package | Role |
|---|---|
| `api/` | chi router + all HTTP handlers; `router.go` is the wiring point |
| `db/` | SQLite schema (inline in `db.go`), CRUD per entity |
| `auth/` | JWT sign/verify (HMAC-SHA256) |
| `mediamtx/` | Typed HTTP client for the mediamtx v3 API |
| `config/` | Viper-based config; env var prefix `MEDIAMTX_UI_` |
| `parser/` | Optional mediamtx.yml config file reader |
| `sysdetect/` | Detects Docker / Podman / systemd deployment type |
| `metrics/` | Prometheus counter/gauge registration |
| `web/` | `go:embed all:dist` — mounts the compiled frontend |

Entry point: `cmd/mediamtx-ui/main.go` wires all packages and seeds the initial admin on first run.

### Frontend layout (`frontend/src/`)

- **`lib/api.ts`** — single typed fetch wrapper; all API calls go through `request<T>()` here. A 401 triggers auto-logout.
- **`lib/stores.ts`** — Svelte stores for JWT token, current user, language, and toast queue.
- **`lib/types.ts`** — TypeScript interfaces mirroring the Go API response shapes.
- **`routes/`** — one Svelte component per page (Dashboard, StreamView, Login, Instructions, NotFound).
- **`App.svelte`** — root component handling route switching and auth gating.

Stack: Svelte 5, Vite 6, Tailwind CSS 4, `svelte-spa-router`, `svelte-i18n`, `hls.js`.

### Auth model

One credential per user: the **UI password** (bcrypt hashed). Used both for `POST /api/v1/auth/login` → JWT and for stream auth. A separate stream-token mechanism was tried and dropped (too complex); the `stream_token_hash` column and the `PublishSlug`/`GetUserByPublishSlug` helpers are dead remnants of it.

Stream clients embed the UI password directly:
- RTSP: `rtsp://username:password@host/path`
- RTMP: `rtmp://host/path?user=username&pass=password` (mediamtx uses `pass`, not `password`)
- SRT: `streamid=publish:path:username:password`

JWT is `Bearer`-checked by the `requireAuth` middleware on all `/api/v1/` routes except `/auth/login` and `/mediamtx/auth`.

mediamtx calls `POST /api/v1/mediamtx/auth` (no JWT) for every stream connection. The handler (`checkMediaMTXAuth`) validates the supplied password against the user's bcrypt hash — or, for the browser player, a UI JWT passed as the password — then runs glob ACL evaluation.

### Database

SQLite with WAL mode. Schema is applied idempotently (`CREATE TABLE IF NOT EXISTS`) in `db.Open()` — no migration files, just the schema string in `internal/db/db.go`. `SetMaxOpenConns(1)` avoids WAL write contention.

### Frontend build → embed pipeline

1. `npm run build` writes to `frontend/dist/`
2. Makefile copies `frontend/dist` → `internal/web/dist`
3. `go build` embeds `internal/web/dist` via `//go:embed all:dist`

The Makefile uses file-dependency tracking so only changed frontend sources trigger a rebuild.

## Ports

| Service | Port |
|---|---|
| mediamtx-ui HTTP | 9996 |
| mediamtx API | 9997 |
| Vite dev server | 5173 |
| mediamtx RTSP | 8554 |
| mediamtx HLS | 8888 |
| mediamtx WebRTC | 8889 |
| mediamtx RTMP | 1935 |
| mediamtx SRT | 8890 |

## Configuration

Config is loaded by Viper from (in order): `./config.yml`, `/etc/mediamtx-ui/config.yml`, environment variables with prefix `MEDIAMTX_UI_` (dots → underscores). See `deploy/config.example.yml` for all keys.

Critical keys to change in production: `auth.jwt_secret`, `auth.initial_admin_pass`.

## Internationalization

Two locales: English (`en`) and Hebrew (`he`). When Hebrew is active, `document.documentElement.dir = 'rtl'` is set, enabling Tailwind's `rtl:` variants. Translation files live in `frontend/src/lib/i18n/`.
