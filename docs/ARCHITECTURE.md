# Architecture

## Overview

mediamtx-ui is a single-binary web application that provides a management frontend for [mediamtx](https://github.com/bluenviron/mediamtx). It runs as a separate process alongside (or alongside a network-reachable instance of) mediamtx.

```
┌─────────────────────────────────────────────────┐
│                  Browser                        │
│  Svelte SPA (served by mediamtx-ui on :9996)   │
└────────────────┬──────────────┬─────────────────┘
                 │ REST API     │ Stream preview
                 ▼              ▼
┌─────────────────────────┐   ┌──────────────────────┐
│      mediamtx-ui        │   │      mediamtx        │
│  Go HTTP server :9996   │──▶│  API       :9997     │
│  ┌─────────────────┐    │   │  RTSP      :8554     │
│  │  chi router     │    │   │  HLS       :8888     │
│  │  JWT middleware │    │   │  WebRTC    :8889     │
│  │  API handlers   │    │   │  RTMP      :1935     │
│  └────────┬────────┘    │   └──────────────────────┘
│           │             │
│  ┌────────▼────────┐    │◀── auth callback POST
│  │  SQLite DB      │    │    /api/v1/mediamtx/auth
│  │  users          │    │
│  │  groups         │    │
│  │  acls           │    │
│  │  audit_log      │    │
│  └─────────────────┘    │
│                         │
│  ┌─────────────────┐    │
│  │ embedded        │    │
│  │ frontend/dist   │    │
│  └─────────────────┘    │
└─────────────────────────┘
```

## Components

### Backend (Go)

| Package | Responsibility |
|---|---|
| `cmd/mediamtx-ui` | Entry point, wiring, initial admin seed |
| `internal/config` | Configuration loading (env vars / YAML file) |
| `internal/db` | SQLite schema, migrations, CRUD queries |
| `internal/auth` | JWT signing and verification |
| `internal/mediamtx` | Typed HTTP client for the mediamtx v3 API |
| `internal/api` | HTTP router (chi), all REST handlers |
| `internal/parser` | Optional mediamtx.yml config file reader |
| `internal/sysdetect` | Deployment method detection |
| `internal/metrics` | Prometheus metrics registration |
| `internal/web` | Embedded frontend assets (go:embed) |

### Frontend (Svelte 5 + Vite)

| File/Dir | Responsibility |
|---|---|
| `src/App.svelte` | Root component, router, auth gating |
| `src/lib/api.ts` | Typed fetch wrapper for all API calls |
| `src/lib/stores.ts` | Global state: token, user, lang, toasts |
| `src/lib/i18n/` | English + Hebrew translations |
| `src/lib/components/` | Shared UI components |
| `src/routes/` | Page components (one per route) |

## Authentication Flow

### UI Login

```
1. User POSTs username + password to POST /api/v1/auth/login
2. Backend looks up user in SQLite, compares bcrypt(password)
3. On success: signs a JWT (HMAC-SHA256) with {uid, username, role}
4. JWT stored in localStorage, sent as Bearer token on subsequent requests
```

### Stream Access (mediamtx auth callback)

```
1. User points VLC / browser at rtsp://username:stream_token@host:8554/stream
2. mediamtx receives the connection request
3. mediamtx POSTs to http://mediamtx-ui:9996/api/v1/mediamtx/auth
   Body: { user, password, path, action, protocol, remoteAddr }
4. mediamtx-ui:
   a. Looks up user by username
   b. bcrypt-compares the provided password against stored stream_token_hash
   c. Runs glob-based ACL check (user direct + group memberships)
   d. Returns 200 (allow) or 401 (deny)
5. mediamtx allows or rejects the connection
6. Auth result is written to SQLite audit_log asynchronously
```

### Credential Separation

| Credential | Purpose | Storage | Embeds in URLs? |
|---|---|---|---|
| UI password | Login to web UI only | bcrypt hash | No |
| Stream token | mediamtx auth callback | bcrypt hash | Yes (`rtsp://user:TOKEN@...`) |

Stream tokens are random 32-byte values (base64url encoded). Shown once on generation; must be saved by the user. Can be regenerated without affecting UI login.

## Database Schema

```sql
users           -- id, username, password_hash, stream_token_hash, role, enabled
groups          -- id, name
user_groups     -- user_id, group_id (many-to-many)
acls            -- id, subject_type, subject_id, stream_pattern, action
stream_metadata -- path_name, description (supplemental to mediamtx API)
audit_log       -- id, username, stream_path, action, protocol, remote_addr, allowed
```

### ACL Evaluation

```
For a given (username, stream_path, action):
1. Resolve all group_ids for the user
2. SELECT from acls WHERE
     (subject_type='user' AND subject_id=user.id)
   OR
     (subject_type='group' AND subject_id IN (user's groups))
3. For each matching ACL: check path.Match(acl.stream_pattern, stream_path)
4. If any ACL matches action (or action='both'): ALLOW
```

Stream patterns use Go's `path.Match` glob syntax: `*` matches any sequence within a path segment, `?` matches a single character. Examples:
- `cameras/*` — all streams under `cameras/`
- `live/event-*` — streams starting with `live/event-`
- `*` — all streams

## Stream Preview

The player component tries formats in this priority order, with manual override:

1. **WebRTC (WHEP)** — lowest latency, uses native browser `RTCPeerConnection`. Sends SDP offer to `http://host:8889/{stream}/whep`.
2. **HLS** — universal browser support via `hls.js`. Loads `http://host:8888/{stream}/index.m3u8`.

Both formats use the mediamtx public host and ports exposed via `/api/v1/system/info`.

## Configuration

All config keys map to `MEDIAMTX_UI_<KEY>` environment variables (with `_` replacing `.`). A YAML config file at `./config.yml` or `/etc/mediamtx-ui/config.yml` is also read.

| Key | Default | Description |
|---|---|---|
| `server.port` | `9996` | HTTP listen port |
| `mediamtx.api_address` | `http://localhost:9997` | mediamtx API base URL |
| `mediamtx.public_host` | *(from api_address)* | Public hostname for stream URLs |
| `mediamtx.config_path` | *(auto-detect)* | Path to mediamtx.yml |
| `auth.jwt_secret` | `change-me` | **Change in production** |
| `auth.initial_admin_user` | `admin` | Seeded on first run |
| `auth.initial_admin_pass` | `changeme` | Seeded on first run |
| `db.path` | `mediamtx-ui.db` | SQLite file path |

## Deployment

See `deploy/` for:
- `Dockerfile` — multi-stage build (Node → Go → scratch)
- `docker-compose.yml` — example compose file
- `mediamtx-auth-snippet.yml` — add to your mediamtx.yml
- `config.example.yml` — full configuration reference

See `scripts/` for:
- `bootstrap.sh` — install dev dependencies
- `build.sh` — build frontend then compile binary
- `deploy.sh` — SSH deploy to a remote host with systemd

## Internationalization

Two locales are supported: English (`en`) and Hebrew (`he`). Language selection is persisted to `localStorage`. When Hebrew is active, `document.documentElement.dir` is set to `rtl`, enabling Tailwind's `rtl:` variant across all components.
