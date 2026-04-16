.PHONY: all build build-frontend build-backend clean dev run lint test

BINARY_NAME=mediamtx-ui
BUILD_DIR=bin
FRONTEND_DIR=frontend
VERSION := $(shell cat VERSION)
GO_FLAGS=-trimpath -ldflags="-s -w -X main.Version=v$(VERSION)"

FRONTEND_SRCS := $(shell find $(FRONTEND_DIR)/src -type f) \
	$(FRONTEND_DIR)/package.json \
	$(FRONTEND_DIR)/vite.config.ts \
	$(FRONTEND_DIR)/svelte.config.js

all: build

## build: Build frontend then compile single Go binary
build: build-frontend build-backend

## build-frontend: Build the Svelte/Vite frontend into frontend/dist and copy to internal/web/dist
build-frontend: internal/web/dist/index.html

# Reinstall npm packages only when package-lock.json changes
$(FRONTEND_DIR)/node_modules/.package-lock.json: $(FRONTEND_DIR)/package-lock.json
	@echo ">>> Installing frontend dependencies..."
	cd $(FRONTEND_DIR) && npm ci
	@touch $@

# Rebuild frontend only when source files or config changes
internal/web/dist/index.html: $(FRONTEND_SRCS) $(FRONTEND_DIR)/node_modules/.package-lock.json
	@echo ">>> Building frontend..."
	cd $(FRONTEND_DIR) && npm run build
	@echo ">>> Copying frontend dist to internal/web/dist..."
	@rm -rf internal/web/dist
	@cp -r $(FRONTEND_DIR)/dist internal/web/dist

## build-backend: Compile Go binary (requires frontend/dist to exist)
build-backend:
	@echo ">>> Building backend ($(VERSION))..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build $(GO_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/mediamtx-ui

## dev: Run Vite dev server and Go backend in parallel (for development)
dev:
	@echo ">>> Starting dev environment..."
	@cd $(FRONTEND_DIR) && npm run dev &
	MEDIAMTX_UI_DEV_PROXY=http://localhost:5173 go run ./cmd/mediamtx-ui

## run: Run the compiled binary
run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

## clean: Remove build artifacts
clean:
	@rm -rf $(BUILD_DIR) $(FRONTEND_DIR)/dist $(FRONTEND_DIR)/node_modules internal/web/dist
	@go clean

## test: Run Go tests
test:
	go test ./...

## lint: Run Go linter
lint:
	golangci-lint run

## docker-build: Build Docker image
docker-build:
	docker build -f deploy/Dockerfile -t mediamtx-ui:$(VERSION) .

## tidy: Tidy Go modules
tidy:
	go mod tidy

## bump-patch: Increment patch version (0.1.0 → 0.1.1)
bump-patch:
	@v=$(VERSION); major=$${v%%.*}; rest=$${v#*.}; minor=$${rest%%.*}; patch=$${rest##*.}; \
	new="$$major.$$minor.$$(( patch + 1 ))"; \
	echo $$new > VERSION; echo "Version bumped: $$v → $$new"

## bump-minor: Increment minor version (0.1.0 → 0.2.0)
bump-minor:
	@v=$(VERSION); major=$${v%%.*}; rest=$${v#*.}; minor=$${rest%%.*}; \
	new="$$major.$$(( minor + 1 )).0"; \
	echo $$new > VERSION; echo "Version bumped: $$v → $$new"

## bump-major: Increment major version (0.1.0 → 1.0.0)
bump-major:
	@v=$(VERSION); major=$${v%%.*}; \
	new="$$(( major + 1 )).0.0"; \
	echo $$new > VERSION; echo "Version bumped: $$v → $$new"

help:
	@grep -E '^## ' Makefile | sed 's/## /  /'
