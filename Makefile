# Makefile for gh-aw Go project

# Variables
BINARY_NAME=gh-aw
VERSION ?= $(shell git describe --tags --always --dirty)

# Build flags
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"

# Default target
.PHONY: all
all: build

# Build the binary, run make deps before this
.PHONY: build
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/gh-aw

# Build for all platforms
.PHONY: build-all
build-all: build-linux build-darwin build-windows

.PHONY: build-linux
build-linux:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 ./cmd/gh-aw
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-arm64 ./cmd/gh-aw

.PHONY: build-darwin
build-darwin:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 ./cmd/gh-aw
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 ./cmd/gh-aw

.PHONY: build-windows
build-windows:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe ./cmd/gh-aw

# Test the code (runs both unit and integration tests)
.PHONY: test
test:
	go test -v -timeout=3m -tags 'integration' ./...

# Test unit tests only (excludes integration tests)
.PHONY: test-unit
test-unit:
	go test -v -timeout=3m -tags '!integration' ./...

.PHONY: test-perf
test-perf:
	go test -v -count=1 -timeout=3m -tags '!integration' ./... | tee /tmp/gh-aw/test-output.log; \
	EXIT_CODE=$$?; \
	echo ""; \
	echo "=== SLOWEST TESTS ==="; \
	grep -E "^\s*--- (PASS|FAIL):" /tmp/gh-aw/test-output.log | \
	grep -E "\([0-9]+\.[0-9]+s\)" | \
	sed 's/.*\(Test[^ ]*\).* (\([0-9]*\.[0-9]*s\)).*/\2 \1/' | \
	sort -nr | \
	head -10; \
	rm -f /tmp/gh-aw/test-output.log; \
	exit $$EXIT_CODE

# Test JavaScript files
.PHONY: test-js
test-js: build-js
	cd pkg/workflow/js && npm run test:js

.PHONY: build-js
build-js:
	cd pkg/workflow/js && npm run typecheck

# Test all code (Go and JavaScript)
.PHONY: test-all
test-all: test test-js

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	go test -v -count=1 -timeout=3m -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
.PHONY: clean
clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME)-* coverage.out coverage.html
	go clean

# Install dependencies
.PHONY: deps
deps:
	go mod download
	go mod tidy
	go install golang.org/x/tools/gopls@latest
	go install github.com/rhysd/actionlint/cmd/actionlint@latest
	npm install -g prettier

# Install development tools (including linter)
.PHONY: deps-dev
deps-dev: deps download-github-actions-schema
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	cd pkg/workflow/js && npm ci

# Download GitHub Actions workflow schema for embedded validation
.PHONY: download-github-actions-schema
download-github-actions-schema:
	@echo "Downloading GitHub Actions workflow schema..."
	@mkdir -p pkg/workflow/schemas
	@curl -s -o pkg/workflow/schemas/github-workflow.json \
		"https://raw.githubusercontent.com/SchemaStore/schemastore/master/src/schemas/json/github-workflow.json"
	@echo "✓ Downloaded GitHub Actions schema to pkg/workflow/schemas/github-workflow.json"

# Run linter
.PHONY: golint
golint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint is not installed. Install it with:"; \
		echo "  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		echo "Or on macOS with Homebrew:"; \
		echo "  brew install golangci-lint"; \
		echo "For other platforms, see: https://golangci-lint.run/usage/install/"; \
		exit 1; \
	fi

# Validate compiled workflow lock files (models: read not supported yet)
.PHONY: validate-workflows
validate-workflows:
	@echo "Validating compiled workflow lock files..."
	actionlint .github/workflows/*.lock.yml; \

# Format code
.PHONY: fmt
fmt:
	go fmt ./...

# Format JavaScript (.cjs and .js) files
.PHONY: fmt-cjs
fmt-cjs:
	cd pkg/workflow/js && npm run format:cjs

# Check formatting
.PHONY: fmt-check
fmt-check:
	@if [ -n "$$(go fmt ./...)" ]; then \
		echo "Code is not formatted. Run 'make fmt' to fix."; \
		exit 1; \
	fi

# Check JavaScript (.cjs) file formatting
.PHONY: fmt-check-cjs
fmt-check-cjs:
	cd pkg/workflow/js && npm run lint:cjs

# Lint JavaScript (.cjs) files 
.PHONY: lint-cjs
lint-cjs: fmt-check-cjs
	@echo "✓ JavaScript formatting validated"

# Validate all project files
.PHONY: lint
lint: fmt-check lint-cjs golint
	@echo "✓ All validations passed"

# Install the binary locally
.PHONY: install
install: build
	gh extension remove gh-aw || true
	gh extension install .

# Generate schema documentation
.PHONY: generate-schema-docs
generate-schema-docs:
	node scripts/generate-schema-docs.js

# Generate status badges documentation
.PHONY: generate-status-badges
generate-status-badges:
	node scripts/generate-status-badges.js

# Recompile all workflow files
.PHONY: recompile
recompile: build 
	./$(BINARY_NAME) init
	./$(BINARY_NAME) compile --validate --verbose --purge
	./$(BINARY_NAME) compile --workflows-dir pkg/cli/workflows --validate --verbose --purge;

# Run development server
.PHONY: dev
dev: build
	./$(BINARY_NAME)

.PHONY: watch
watch: build
	./$(BINARY_NAME) compile --watch

# Changeset management targets
.PHONY: version
version:
	@node scripts/changeset.js version

.PHONY: release
release: test
	@node scripts/changeset.js release

# Agent should run this task before finishing its turns
.PHONY: agent-finish
agent-finish: deps-dev fmt fmt-cjs lint build test-all recompile generate-schema-docs generate-status-badges
	@echo "Agent finished tasks successfully."

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build            - Build the binary for current platform"
	@echo "  build-all        - Build binaries for all platforms"
	@echo "  test             - Run Go tests (unit + integration)"
	@echo "  test-unit        - Run Go unit tests only (faster)"
	@echo "  test-js          - Run JavaScript tests"
	@echo "  test-all         - Run all tests (Go and JavaScript)"
	@echo "  test-coverage    - Run tests with coverage report"
	@echo "  clean            - Clean build artifacts"
	@echo "  deps             - Install dependencies"
	@echo "  lint             - Run linter"
	@echo "  fmt              - Format code"
	@echo "  fmt-cjs          - Format JavaScript (.cjs and .js) files"
	@echo "  fmt-check        - Check code formatting"
	@echo "  fmt-check-cjs    - Check JavaScript (.cjs) file formatting"
	@echo "  lint-cjs         - Lint JavaScript (.cjs) files"
	@echo "  validate-workflows - Validate compiled workflow lock files"
	@echo "  validate         - Run all validations (fmt-check, lint, validate-workflows)"
	@echo "  install          - Install binary locally"
	@echo "  recompile        - Recompile all workflow files (runs init, depends on build)"
	@echo "  generate-schema-docs - Generate frontmatter full reference documentation from JSON schema"
	@echo "  generate-status-badges - Generate workflow status badges documentation page"

	@echo "  agent-finish     - Complete validation sequence (build, test, recompile, fmt, lint)"
	@echo "  version   - Preview next version from changesets"
	@echo "  release   - Create release using changesets (depends on test)"
	@echo "  help             - Show this help message"