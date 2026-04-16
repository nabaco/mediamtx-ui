# ---------- Stage 1: Build frontend ----------
FROM node:22-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# ---------- Stage 2: Build Go binary ----------
FROM golang:1.23-alpine AS go-builder
WORKDIR /app

# Install build tools
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend-builder /app/frontend/dist ./internal/web/dist

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -X main.Version=${VERSION}" \
    -o /bin/mediamtx-ui \
    ./cmd/mediamtx-ui

# ---------- Stage 3: Final minimal image ----------
FROM scratch

COPY --from=go-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=go-builder /bin/mediamtx-ui /mediamtx-ui

# Run as non-root (UID 1000)
USER 1000

EXPOSE 9996

ENTRYPOINT ["/mediamtx-ui"]
