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
build: sync-templates sync-action-pins
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
	go test -v -timeout=3m -tags 'integration' -run='^Test' ./...

# Test unit tests only (excludes integration tests)
.PHONY: test-unit
test-unit:
	go test -v -timeout=3m -tags '!integration' -run='^Test' ./...

# Test specific integration test groups (matching CI workflow)
.PHONY: test-integration-compile
test-integration-compile:
	go test -v -timeout=3m -tags 'integration' -run 'TestCompile|TestPoutine' ./pkg/cli

.PHONY: test-integration-mcp-playwright
test-integration-mcp-playwright:
	go test -v -timeout=3m -tags 'integration' -run 'TestMCPInspectPlaywright' ./pkg/cli

.PHONY: test-integration-mcp-other
test-integration-mcp-other:
	go test -v -timeout=3m -tags 'integration' -run 'TestMCPAdd|TestMCPInspectGitHub|TestMCPServer|TestMCPConfig' ./pkg/cli

.PHONY: test-integration-logs
test-integration-logs:
	go test -v -timeout=3m -tags 'integration' -run 'TestLogs|TestFirewall|TestNoStopTime|TestLocalWorkflow' ./pkg/cli

.PHONY: test-integration-workflow
test-integration-workflow:
	go test -v -timeout=3m -tags 'integration' ./pkg/workflow ./cmd/gh-aw

.PHONY: test-perf
test-perf:
	go test -v -count=1 -timeout=3m -tags 'integration' -run='^Test' ./... | tee /tmp/gh-aw/test-output.log; \
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

# Run benchmarks for performance testing
.PHONY: bench
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem -benchtime=3x -run=^$$ ./pkg/... | tee bench_results.txt

# Run benchmarks with comparison output
.PHONY: bench-compare
bench-compare:
	@echo "Running benchmarks and saving results..."
	go test -bench=. -benchmem -benchtime=100x -run=^$$ ./pkg/... | tee bench_results.txt
	@echo "Benchmark results saved to bench_results.txt"

# Run fuzz tests
.PHONY: fuzz
fuzz:
	@echo "Running fuzz tests for 30 seconds..."
	go test -fuzz=FuzzParseFrontmatter -fuzztime=30s ./pkg/parser/
	go test -fuzz=FuzzExpressionParser -fuzztime=30s ./pkg/workflow/

# Test JavaScript files
.PHONY: test-js
test-js: build-js
	cd pkg/workflow/js && npm run test:js -- --no-file-parallelism

.PHONY: build-js
build-js:
	cd pkg/workflow/js && npm run typecheck

# Bundle JavaScript files with local requires
.PHONY: bundle-js
bundle-js:
	@echo "Building bundle-js tool..."
	@go build -o bundle-js ./cmd/bundle-js
	@echo "✓ bundle-js tool built"
	@echo "To bundle a JavaScript file: ./bundle-js <input-file> [output-file]"

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

# Check Node.js version
.PHONY: check-node-version
check-node-version:
	@if ! command -v node >/dev/null 2>&1; then \
		echo "Error: Node.js is not installed."; \
		echo ""; \
		echo "This project requires Node.js 20 or higher."; \
		echo "Please install Node.js 20+ and try again."; \
		echo ""; \
		echo "For installation instructions, see:"; \
		echo "  https://github.com/githubnext/gh-aw/blob/main/CONTRIBUTING.md#prerequisites"; \
		exit 1; \
	fi; \
	NODE_VERSION=$$(node --version); \
	NODE_VERSION_NUM=$$(echo "$$NODE_VERSION" | sed 's/v//'); \
	NODE_MAJOR=$$(echo "$$NODE_VERSION_NUM" | cut -d. -f1); \
	if [ "$$NODE_MAJOR" -lt 20 ]; then \
		echo "Error: Node.js version $$NODE_VERSION is not supported."; \
		echo ""; \
		echo "This project requires Node.js 20 or higher."; \
		echo "Your current version: $$NODE_VERSION"; \
		echo ""; \
		echo "Please upgrade Node.js and try again."; \
		echo ""; \
		echo "For installation instructions, see:"; \
		echo "  https://github.com/githubnext/gh-aw/blob/main/CONTRIBUTING.md#prerequisites"; \
		exit 1; \
	fi; \
	echo "✓ Node.js version check passed ($$NODE_VERSION)"

# Install dependencies
.PHONY: deps
deps: check-node-version
	go mod download
	go mod tidy
	go install golang.org/x/tools/gopls@latest
	go install github.com/rhysd/actionlint/cmd/actionlint@latest
	cd pkg/workflow/js && npm ci

# Install development tools (including linter)
.PHONY: deps-dev
deps-dev: check-node-version deps download-github-actions-schema
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Download GitHub Actions workflow schema for embedded validation
.PHONY: download-github-actions-schema
download-github-actions-schema:
	@echo "Downloading GitHub Actions workflow schema..."
	@mkdir -p pkg/workflow/schemas
	@curl -s -o pkg/workflow/schemas/github-workflow.json \
		"https://raw.githubusercontent.com/SchemaStore/schemastore/master/src/schemas/json/github-workflow.json"
	@echo "Formatting schema with prettier..."
	@cd pkg/workflow/js && npm run format:schema >/dev/null 2>&1
	@echo "✓ Downloaded and formatted GitHub Actions schema to pkg/workflow/schemas/github-workflow.json"

# Run linter
.PHONY: golint
golint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint is not installed. Run 'make deps-dev' to install dependencies."; \
		exit 1; \
	fi

# Validate compiled workflow lock files (models: read not supported yet)
.PHONY: validate-workflows
validate-workflows:
	@echo "Validating compiled workflow lock files..."
	actionlint .github/workflows/*.lock.yml; \

# Format code
.PHONY: fmt
fmt: fmt-go fmt-cjs fmt-json
	@echo "✓ Code formatted successfully"

.PHONY: fmt-go
fmt-go:
	go fmt ./...

# Format JavaScript (.cjs and .js) and JSON files in pkg/workflow/js directory
.PHONY: fmt-cjs
fmt-cjs:
	cd pkg/workflow/js && npm run format:cjs

# Format JSON files in pkg directory (excluding pkg/workflow/js, which is handled by npm script)
.PHONY: fmt-json
fmt-json:
	cd pkg/workflow/js && npm run format:pkg-json

# Check formatting
.PHONY: fmt-check
fmt-check:
	@if [ -n "$$(go fmt ./...)" ]; then \
		echo "Code is not formatted. Run 'make fmt' to fix."; \
		exit 1; \
	fi

# Check JavaScript (.cjs and .js) and JSON file formatting in pkg/workflow/js directory
.PHONY: fmt-check-cjs
fmt-check-cjs:
	cd pkg/workflow/js && npm run lint:cjs

# Check JSON file formatting in pkg directory (excluding pkg/workflow/js, which is handled by npm script)
.PHONY: fmt-check-json
fmt-check-json:
	@if ! cd pkg/workflow/js && npm run check:pkg-json 2>&1 | grep -q "All matched files use Prettier code style"; then \
		echo "JSON files are not formatted. Run 'make fmt-json' to fix."; \
		exit 1; \
	fi

# Lint JavaScript (.cjs and .js) and JSON files in pkg/workflow/js directory
.PHONY: lint-cjs
lint-cjs: fmt-check-cjs
	@echo "✓ JavaScript formatting validated"

# Lint JSON files in pkg directory (excluding pkg/workflow/js, which is handled by npm script)
.PHONY: lint-json
lint-json: fmt-check-json
	@echo "✓ JSON formatting validated"

# Lint error messages for quality compliance
.PHONY: lint-errors
lint-errors:
	@echo "Running error message quality linter..."
	@go run scripts/lint_error_messages.go

# Validate all project files
.PHONY: lint
lint: fmt-check fmt-check-json lint-cjs golint
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

# Generate labs documentation page
.PHONY: generate-labs
generate-labs:
	node scripts/generate-labs.js

# Sync templates from .github to pkg/cli/templates
.PHONY: sync-templates
sync-templates:
	@echo "Syncing templates from .github to pkg/cli/templates..."
	@mkdir -p pkg/cli/templates
	@cp .github/instructions/github-agentic-workflows.instructions.md pkg/cli/templates/
	@cp .github/prompts/create-agentic-workflow.prompt.md pkg/cli/templates/create-agentic-workflow.md
	@cp .github/prompts/setup-agentic-workflows.prompt.md pkg/cli/templates/setup-agentic-workflows.md
	@cp .github/prompts/create-shared-agentic-workflow.prompt.md pkg/cli/templates/create-shared-agentic-workflow.md
	@cp .github/prompts/debug-agentic-workflow.prompt.md pkg/cli/templates/debug-agentic-workflow.md
	@echo "✓ Templates synced successfully"

# Sync action pins from .github/aw to pkg/workflow/data
.PHONY: sync-action-pins
sync-action-pins:
	@echo "Syncing actions-lock.json from .github/aw to pkg/workflow/data/action_pins.json..."
	@if [ -f .github/aw/actions-lock.json ]; then \
		cp .github/aw/actions-lock.json pkg/workflow/data/action_pins.json; \
		echo "✓ Action pins synced successfully"; \
	else \
		echo "⚠ Warning: .github/aw/actions-lock.json does not exist yet"; \
	fi

# Recompile all workflow files
.PHONY: recompile
recompile: sync-templates build
	./$(BINARY_NAME) init
	./$(BINARY_NAME) compile --validate --verbose --purge
#	./$(BINARY_NAME) compile --workflows-dir pkg/cli/workflows --validate --verbose --purge

# Generate Dependabot manifests for npm dependencies
.PHONY: dependabot
dependabot: build
	./$(BINARY_NAME) compile --dependabot --verbose

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
release: build
	@node scripts/changeset.js release

# Agent should run this task before finishing its turns
.PHONY: agent-finish
agent-finish: deps-dev fmt lint build test-all recompile dependabot generate-schema-docs generate-labs
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
	@echo "  bench            - Run benchmarks for performance testing"
	@echo "  bench-compare    - Run benchmarks with comparison output"
	@echo "  fuzz             - Run fuzz tests for 30 seconds"
	@echo "  bundle-js        - Build JavaScript bundler tool (./bundle-js <input> [output])"
	@echo "  clean            - Clean build artifacts"
	@echo "  deps             - Install dependencies"
	@echo "  check-node-version - Check Node.js version (20 or higher required)"
	@echo "  lint             - Run linter"
	@echo "  fmt              - Format code"
	@echo "  fmt-cjs          - Format JavaScript (.cjs and .js) and JSON files in pkg/workflow/js"
	@echo "  fmt-json         - Format JSON files in pkg directory (excluding pkg/workflow/js)"
	@echo "  fmt-check        - Check code formatting"
	@echo "  fmt-check-cjs    - Check JavaScript (.cjs) and JSON file formatting in pkg/workflow/js"
	@echo "  fmt-check-json   - Check JSON file formatting in pkg directory (excluding pkg/workflow/js)"
	@echo "  lint-cjs         - Lint JavaScript (.cjs) and JSON files in pkg/workflow/js"
	@echo "  lint-json        - Lint JSON files in pkg directory (excluding pkg/workflow/js)"
	@echo "  lint-errors      - Lint error messages for quality compliance"
	@echo "  validate-workflows - Validate compiled workflow lock files"
	@echo "  validate         - Run all validations (fmt-check, lint, validate-workflows)"
	@echo "  install          - Install binary locally"
	@echo "  sync-templates   - Sync templates from .github to pkg/cli/templates (runs automatically during build)"
	@echo "  sync-action-pins - Sync actions-lock.json from .github/aw to pkg/workflow/data (runs automatically during build)"
	@echo "  recompile        - Recompile all workflow files (runs init, depends on build)"
	@echo "  dependabot       - Generate Dependabot manifests for npm dependencies in workflows"
	@echo "  generate-schema-docs - Generate frontmatter full reference documentation from JSON schema"
	@echo "  generate-labs              - Generate labs documentation page"

	@echo "  agent-finish     - Complete validation sequence (build, test, recompile, fmt, lint)"
	@echo "  version   - Preview next version from changesets"
	@echo "  release   - Create release using changesets (depends on test)"
	@echo "  help             - Show this help message"