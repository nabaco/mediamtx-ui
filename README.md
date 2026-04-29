# mediamtx-ui

A self-hosted management frontend for [mediamtx](https://github.com/bluenviron/mediamtx).

**Features:**
- Dashboard with live stream status and in-browser preview (WebRTC / HLS)
- Stream management via mediamtx API (add, edit, delete paths) with inline row editing
- User & group management with ACL-based stream access control
- Separate UI password and stream token per user
  - Stream tokens embed safely in RTSP/HLS/WebRTC/RTMP URLs
  - Slug-based anonymous RTMP publish (no credentials in URL — rotates with token)
- mediamtx auth callback integration — no manual auth config in mediamtx.yml per-stream
- Audit log of all stream access attempts
- Read-only display of mediamtx.yml (when accessible on the same host)
- Deployment-aware instructions page (Docker / Podman / Compose / Quadlets / systemd)
- English + Hebrew (RTL) UI with one-click language switching
- Prometheus metrics at `/metrics`
- Single binary — frontend embedded via `go:embed`
- Ready for containerization

## Quick Start

### With Docker

```bash
docker run -d \
  --name mediamtx-ui \
  --restart unless-stopped \
  -p 9996:9996 \
  -v mediamtx-ui-data:/data \
  -e MEDIAMTX_UI_MEDIAMTX_API_ADDRESS=http://YOUR_MEDIAMTX_HOST:9997 \
  -e MEDIAMTX_UI_MEDIAMTX_PUBLIC_HOST=YOUR_SERVER_IP \
  -e MEDIAMTX_UI_AUTH_JWT_SECRET=your-secret-here \
  -e MEDIAMTX_UI_AUTH_INITIAL_ADMIN_PASS=your-admin-pass \
  -e MEDIAMTX_UI_DB_PATH=/data/mediamtx-ui.db \
  mediamtx-ui:latest
```

Open `http://YOUR_SERVER_IP:9996` and log in with `admin` / `your-admin-pass`.

### With docker-compose

```bash
cp deploy/docker-compose.yml docker-compose.yml
# Edit environment variables
docker compose up -d
```

### As a systemd service (binary)

```bash
# Install dependencies and build
make

# Copy binary and config
sudo cp bin/mediamtx-ui /opt/mediamtx-ui/
sudo cp deploy/config.example.yml /opt/mediamtx-ui/config.yml
# Edit config.yml

# Install service (see scripts/deploy.sh for automation)
sudo systemctl enable --now mediamtx-ui
```

## Configuration

See `deploy/config.example.yml` for all options. Key settings:

```yaml
mediamtx:
  api_address: "http://localhost:9997"   # mediamtx API
  public_host: "192.168.1.100"           # public IP for stream URLs

auth:
  jwt_secret: "change-me-in-production"  # CHANGE THIS
  initial_admin_pass: "changeme"         # initial admin password

db:
  path: "mediamtx-ui.db"
```

All settings can be set via environment variables: `MEDIAMTX_UI_<KEY>` (dots become underscores).

## Connecting mediamtx

Add to your `mediamtx.yml`:

```yaml
auth:
  method: http
  httpAddress: http://mediamtx-ui-host:9996/api/v1/mediamtx/auth
```

See `deploy/mediamtx-auth-snippet.yml` for more options.

## User Management

1. Log in as admin
2. Go to **Users** → **Add User** to create accounts
3. Click **Generate Stream Token** to create a stream credential for each user
4. Go to **Access Control** → **Add Rule** to grant stream access
5. Share the stream URLs from the stream detail page with users

**Stream URL formats (populated automatically on the stream detail page):**

| Protocol | URL format | Use |
|---|---|---|
| RTSP | `rtsp://username:TOKEN@host:8554/stream` | VLC, ffmpeg, IP cameras |
| HLS | `http://host:8888/stream/index.m3u8` | Browser, VLC |
| WebRTC | `http://host:8889/stream` | Built-in browser player |
| RTMP (view) | `rtmp://username:TOKEN@host:1935/stream` | Viewers |
| RTMP (publish) | `rtmp://host:1935/stream?token=SLUG` | OBS, encoders (no credentials in URL) |
| SRT (publish) | `srt://host:8890?streamid=publish:stream:username:TOKEN` | SRT encoders |

**RTMP anonymous publish** — the `?token=SLUG` is a short 8-character slug derived from the user's stream token. When the token is regenerated, the slug changes automatically. In OBS: set **Server** to `rtmp://host:1935/` and **Stream Key** to `stream?token=SLUG`.

## Building from Source

```bash
# Install Node.js and Go, then:
make          # build frontend + binary → bin/mediamtx-ui
make dev      # development mode: hot-reload frontend + Go backend
```

## Architecture

See [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) for a detailed design overview including:
- Component diagram
- Authentication flows (UI login, stream access, slug-based publish)
- Database schema
- ACL evaluation logic
- Stream preview protocol selection

## Prometheus Metrics

Exposed at `GET /metrics` (no auth required — protect via firewall/network).

| Metric | Description |
|---|---|
| `mediamtx_ui_auth_callbacks_total` | mediamtx auth callback results (by action and outcome) |
| `mediamtx_ui_active_streams` | Currently active streams |
| `mediamtx_ui_stream_readers` | Readers per stream |
| `mediamtx_ui_login_attempts_total` | UI login attempts by outcome |
| `mediamtx_ui_api_requests_total` | API requests by method/path/status |

## Security Notes

- Change `auth.jwt_secret` and `auth.initial_admin_pass` before first deployment
- The `/metrics` endpoint has no auth — restrict access at the network level
- The mediamtx auth callback (`/api/v1/mediamtx/auth`) should be on an internal network
- Stream tokens are stored **plaintext** in the database — they must be to support embedding in RTSP/HLS URLs and WHEP playback. Protect the database file accordingly.
- UI passwords are bcrypt-hashed
- The UI does not handle TLS — put it behind nginx/Caddy/Traefik for HTTPS

## License

MIT
