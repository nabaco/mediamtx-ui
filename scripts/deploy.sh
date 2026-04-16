#!/usr/bin/env bash
# deploy.sh — deploy mediamtx-ui to a remote host via SSH + systemd
# Reads settings from scripts/deploy.local.sh (gitignored)
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
LOCAL_CONFIG="$ROOT/scripts/deploy.local.sh"

if [[ ! -f "$LOCAL_CONFIG" ]]; then
  echo "ERROR: $LOCAL_CONFIG not found."
  echo "Copy scripts/deploy.local.sh.example to scripts/deploy.local.sh and fill in your values."
  exit 1
fi

# shellcheck source=/dev/null
source "$LOCAL_CONFIG"

# Required variables (set in deploy.local.sh):
: "${DEPLOY_HOST:?}"       # SSH host, e.g. user@192.168.1.100
: "${DEPLOY_PATH:?}"       # Remote path, e.g. /opt/mediamtx-ui
: "${DEPLOY_USER:?}"       # System user to run the service as
: "${DEPLOY_SERVICE:?}"    # systemd service name, e.g. mediamtx-ui

BINARY="$ROOT/bin/mediamtx-ui"

if [[ ! -f "$BINARY" ]]; then
  echo "Binary not found. Running build first..."
  "$ROOT/scripts/build.sh"
fi

echo "==> Deploying to $DEPLOY_HOST:$DEPLOY_PATH"

# Upload binary
echo "--- Uploading binary..."
ssh "$DEPLOY_HOST" "mkdir -p $DEPLOY_PATH"
rsync -az --progress "$BINARY" "$DEPLOY_HOST:$DEPLOY_PATH/mediamtx-ui"

# Upload example config if no config exists
ssh "$DEPLOY_HOST" "test -f $DEPLOY_PATH/config.yml || true" && \
  rsync -az "$ROOT/deploy/config.example.yml" "$DEPLOY_HOST:$DEPLOY_PATH/config.yml.example"

# Install / update systemd unit
echo "--- Installing systemd unit..."
cat <<EOF | ssh "$DEPLOY_HOST" "sudo tee /etc/systemd/system/$DEPLOY_SERVICE.service > /dev/null"
[Unit]
Description=MediaMTX UI
After=network.target

[Service]
Type=simple
User=$DEPLOY_USER
WorkingDirectory=$DEPLOY_PATH
ExecStart=$DEPLOY_PATH/mediamtx-ui
Restart=on-failure
RestartSec=5s
EnvironmentFile=-$DEPLOY_PATH/.env

[Install]
WantedBy=multi-user.target
EOF

# Reload and restart
echo "--- Restarting service..."
ssh "$DEPLOY_HOST" "sudo systemctl daemon-reload && sudo systemctl enable $DEPLOY_SERVICE && sudo systemctl restart $DEPLOY_SERVICE"
ssh "$DEPLOY_HOST" "sleep 2 && sudo systemctl is-active $DEPLOY_SERVICE"

echo "==> Deploy complete. Service $DEPLOY_SERVICE is running on $DEPLOY_HOST"
