.PHONY: setup deps web generate build dev run clean help

VERSION  ?= dev
COMMIT   ?= $(shell git rev-parse --short=12 HEAD 2>/dev/null || echo none)
BUILD    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS   = -s -w -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.Build=$(BUILD)
OUT_DIR   = dist
BINARY    = $(OUT_DIR)/backend-go

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

setup: deps web generate ## Full init: deps + web SPA + codegen
	@echo "==> Setup complete"

deps: ## Download and tidy Go modules
	go mod download
	go mod tidy

web: ## Install web deps and build admin SPA
	cd web && pnpm install --no-frozen-lockfile && pnpm build

generate: ## Generate module import file (autogen_imports.go)
	go generate ./internal/bootstrap/mod

build: setup ## Full build: setup + compile binary
	@mkdir -p $(OUT_DIR)
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/server
	@echo "==> Built $(BINARY)"

dev: deps generate ## Quick dev build (skip web)
	@mkdir -p $(OUT_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/server
	@echo "==> Built $(BINARY)"

run: dev ## Build and run
	$(BINARY)

clean: ## Remove build artifacts
	rm -rf $(OUT_DIR)
	rm -f internal/bootstrap/mod/autogen_imports.go
