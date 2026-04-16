# mediamtx-ui

A self-hosted management frontend for [mediamtx](https://github.com/bluenviron/mediamtx).

**Features:**
- Dashboard with live stream status and in-browser preview (WebRTC / HLS)
- Stream management via mediamtx API (add, edit, delete paths)
- User & group management with ACL-based stream access control
- Separate UI password and stream token per user (stream tokens embed safely in URLs)
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
# Build
./scripts/bootstrap.sh
./scripts/build.sh

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

**Stream URL format:**
```
rtsp://username:STREAM_TOKEN@host:8554/stream-name
http://host:8888/stream-name/index.m3u8   (HLS)
http://host:8889/stream-name              (WebRTC)
```

## Building from Source

```bash
# Install dependencies
./scripts/bootstrap.sh

# Build everything (frontend + Go binary)
./scripts/build.sh
# Output: bin/mediamtx-ui

# Development mode (hot-reload frontend + Go backend)
make dev
```

## Architecture

See [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) for a detailed design overview including:
- Component diagram
- Authentication flows
- Database schema
- ACL evaluation logic
- Stream preview protocol selection

## Prometheus Metrics

Exposed at `GET /metrics` (no auth required — protect via firewall/network).

| Metric | Description |
|---|---|
| `mediamtx_ui_auth_callbacks_total` | mediamtx auth callback results |
| `mediamtx_ui_active_streams` | Currently active streams |
| `mediamtx_ui_stream_readers` | Readers per stream |
| `mediamtx_ui_login_attempts_total` | UI login attempts by outcome |
| `mediamtx_ui_api_requests_total` | API requests by method/path/status |

## Security Notes

- Change `auth.jwt_secret` and `auth.initial_admin_pass` before first deployment
- The `/metrics` endpoint has no auth — restrict access at the network level
- The mediamtx auth callback (`/api/v1/mediamtx/auth`) should be on an internal network
- Stream tokens are stored as bcrypt hashes; shown only once at generation
- The UI does not handle TLS — put it behind nginx/caddy for HTTPS

## License

MIT
