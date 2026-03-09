VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BINARY  := snappy
OUTDIR  := bin
COMPDIR := completions

LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: all build completions test test-scrut test-scrut-update test-scrut-ez test-scrut-ez-update test-scrut-install test-scrut-install-update test-install test-all test-ci lint lint-all lint-md lint-actions lint-man vet fmt format format-check man clean cover tidy help

all: fmt vet lint test build ## Run all checks and build

build: ## Build the binary
	mkdir -p $(OUTDIR)
	go build $(LDFLAGS) -o $(OUTDIR)/$(BINARY) .

completions: build ## Generate shell completion scripts
	mkdir -p $(COMPDIR)
	$(OUTDIR)/$(BINARY) completion bash > $(COMPDIR)/snappy.bash
	$(OUTDIR)/$(BINARY) completion zsh > $(COMPDIR)/_snappy
	$(OUTDIR)/$(BINARY) completion fish > $(COMPDIR)/snappy.fish
	$(OUTDIR)/$(BINARY) completion powershell > $(COMPDIR)/snappy.ps1

test: ## Run tests
	go test ./...

lint-all: lint lint-md lint-actions lint-man ## Run all linters

lint: ## Run golangci-lint
	golangci-lint run ./...

lint-md: ## Lint Markdown files
	npx markdownlint-cli2 "**/*.md"

lint-actions: ## Lint GitHub Actions workflows
	actionlint

lint-man: ## Lint the man page with mandoc
	mandoc -Tlint docs/snappy.1

man: ## Preview the man page
	man ./docs/snappy.1

vet: ## Run go vet
	go vet ./...

fmt: ## Check Go formatting (exits non-zero if files need formatting)
	@test -z "$$(gofmt -l .)" || { gofmt -l . && exit 1; }

format: ## Format all code (Go, Markdown, JSON, YAML, shell)
	golangci-lint fmt ./...
	npx prettier --write . --ignore-unknown
	git ls-files | xargs shfmt -f | xargs shfmt -w

format-check: ## Check all formatting (Go, Markdown, JSON, YAML, shell)
	@test -z "$$(gofmt -l .)" || { gofmt -l . && exit 1; }
	npx prettier --check . --ignore-unknown
	git ls-files | xargs shfmt -f | xargs shfmt -d

test-scrut: build ## Run scrut CLI tests
	@echo "Running scrut CLI tests..."
	@if ! command -v scrut >/dev/null 2>&1; then \
		echo "scrut not installed. Install from https://github.com/facebookincubator/scrut"; \
		exit 1; \
	fi
	SNAPPY_BIN="$(CURDIR)/$(OUTDIR)/$(BINARY)" scrut test tests/scrut/*.md

test-scrut-update: build ## Update scrut test expectations
	SNAPPY_BIN="$(CURDIR)/$(OUTDIR)/$(BINARY)" scrut update --replace --assume-yes tests/scrut/*.md

test-scrut-ez: ## Run scrut tests for snappy-ez
	@echo "Running snappy-ez scrut tests..."
	@if ! command -v scrut >/dev/null 2>&1; then \
		echo "scrut not installed. Install from https://github.com/facebookincubator/scrut"; \
		exit 1; \
	fi
	SNAPPY_EZ_BIN="$(CURDIR)/bin/snappy-ez" scrut test tests/scrut/snappy-ez/

test-scrut-ez-update: ## Update snappy-ez scrut test expectations
	SNAPPY_EZ_BIN="$(CURDIR)/bin/snappy-ez" scrut update --replace --assume-yes tests/scrut/snappy-ez/

test-scrut-install: ## Run scrut tests for install.sh
	@echo "Running install.sh scrut tests..."
	@if ! command -v scrut >/dev/null 2>&1; then \
		echo "scrut not installed. Install from https://github.com/facebookincubator/scrut"; \
		exit 1; \
	fi
	INSTALL_SH_BIN="$(CURDIR)/install.sh" scrut test tests/scrut/install-script/

test-scrut-install-update: ## Update install.sh scrut test expectations
	INSTALL_SH_BIN="$(CURDIR)/install.sh" scrut update --replace --assume-yes tests/scrut/install-script/

INSTALL_TEST_VERSION ?= $(shell gh release view --json tagName --jq '.tagName' 2>/dev/null || echo "v0.3.1")

test-install: ## Run install.sh against a real GitHub release (requires network)
	@echo "Testing install.sh with version $(INSTALL_TEST_VERSION)..."
	@tmp_dir=$$(mktemp -d); \
		trap 'rm -rf "$${tmp_dir}"' EXIT; \
		set -e; \
		INSTALL_DIR="$${tmp_dir}" bash "$(CURDIR)/install.sh" --version $(INSTALL_TEST_VERSION); \
		"$${tmp_dir}/snappy" --version; \
		echo "install.sh test passed."

test-all: test test-scrut test-scrut-ez ## Run all tests (unit + scrut + scrut-ez)

test-ci: test-all test-scrut-install test-install ## All tests including CI-only (network, macOS)

clean: ## Remove build artifacts
	rm -f $(OUTDIR)/$(BINARY) dist coverage.out
	rm -rf $(COMPDIR)

cover: ## Run tests with coverage
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

tidy: ## Tidy go.mod
	go mod tidy

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-16s %s\n", $$1, $$2}'
