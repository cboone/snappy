VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BINARY  := snappy
OUTDIR  := bin

LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: all build test lint lint-go lint-md lint-actions vet fmt fmt-check format format-check clean cover tidy help

all: fmt-check vet lint test build ## Run all checks and build

build: ## Build the binary
	mkdir -p $(OUTDIR)
	go build $(LDFLAGS) -o $(OUTDIR)/$(BINARY) .

test: ## Run tests
	go test ./...

lint: lint-go lint-md lint-actions ## Run all linters

lint-go: ## Run golangci-lint
	golangci-lint run ./...

lint-md: ## Lint Markdown files
	npx markdownlint-cli2 "**/*.md"

lint-actions: ## Lint GitHub Actions workflows
	actionlint

vet: ## Run go vet
	go vet ./...

fmt: ## Format all code (Go, Markdown, JSON, YAML, shell)
	golangci-lint fmt ./...
	npx prettier --write . --ignore-unknown
	git ls-files | xargs shfmt -f | xargs shfmt -w

fmt-check: ## Check formatting (exits non-zero if files need formatting)
	@test -z "$$(gofmt -l .)" || { gofmt -l . && exit 1; }
	npx prettier --check . --ignore-unknown
	git ls-files | xargs shfmt -f | xargs shfmt -d

format: fmt ## Alias for fmt

format-check: fmt-check ## Alias for fmt-check

clean: ## Remove build artifacts
	rm -rf $(OUTDIR) dist coverage.out

cover: ## Run tests with coverage
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

tidy: ## Tidy go.mod
	go mod tidy

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-16s %s\n", $$1, $$2}'
