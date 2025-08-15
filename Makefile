# Makefile for gh-aw Go project

# Variables
BINARY_NAME=gh-aw
VERSION ?= $(shell git describe --tags --always --dirty)

# Build flags
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"

# Default target
.PHONY: all
all: build

# Build the binary
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

# Test the code
.PHONY: test
test:
	go test -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
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

# Install development tools (including linter)
.PHONY: deps-dev
deps-dev: deps copy-copilot-to-claude
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

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

# Validate auto-compile workflow template
.PHONY: validate-template
validate-template:
	@echo "Validating auto-compile workflow template..."
	@if command -v actionlint >/dev/null 2>&1; then \
		actionlint pkg/cli/templates/auto-compile-workflow.yml; \
	else \
		echo "actionlint is not installed. Installing..."; \
		go install github.com/rhysd/actionlint/cmd/actionlint@latest; \
		actionlint pkg/cli/templates/auto-compile-workflow.yml; \
	fi
	@echo "✓ Auto-compile workflow template is valid"

# Validate compiled workflow lock files (models: read not supported yet)
.PHONY: validate-workflows
validate-workflows:
	@echo "Validating compiled workflow lock files..."
	actionlint .github/workflows/*.lock.yml; \

# Format code
.PHONY: fmt
fmt:
	go fmt ./...

# Check formatting
.PHONY: fmt-check
fmt-check:
	@if [ -n "$$(go fmt ./...)" ]; then \
		echo "Code is not formatted. Run 'make fmt' to fix."; \
		exit 1; \
	fi

# Validate all project files
.PHONY: lint
lint: fmt-check golint validate-template
	@echo "✓ All validations passed"

# Install the binary locally
.PHONY: install
install: build
	gh extension remove gh-aw || true
	gh extension install .

# Recompile all workflow files
.PHONY: recompile
recompile: build
	./$(BINARY_NAME) compile --validate --instructions

# Run development server
.PHONY: dev
dev: build
	./$(BINARY_NAME)

.PHONY: watch
watch: build
	./$(BINARY_NAME) compile --watch

# Create and push a patch release (increments patch version)
.PHONY: patch-release
patch-release:
	@echo "Creating patch release..."
	@LATEST_TAG=$$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"); \
	echo "Current latest tag: $$LATEST_TAG"; \
	VERSION_NUMS=$$(echo "$$LATEST_TAG" | sed 's/^v//'); \
	MAJOR=$$(echo "$$VERSION_NUMS" | cut -d. -f1); \
	MINOR=$$(echo "$$VERSION_NUMS" | cut -d. -f2); \
	PATCH=$$(echo "$$VERSION_NUMS" | cut -d. -f3); \
	MAJOR=$${MAJOR:-0}; MINOR=$${MINOR:-0}; PATCH=$${PATCH:-0}; \
	NEW_PATCH=$$((PATCH + 1)); \
	NEW_VERSION="v$$MAJOR.$$MINOR.$$NEW_PATCH"; \
	echo "New version will be: $$NEW_VERSION"; \
	printf "Create and push release $$NEW_VERSION? [y/N] "; \
	read REPLY; \
	case "$$REPLY" in \
		[Yy]|[Yy][Ee][Ss]) \
			echo "Creating tag $$NEW_VERSION..."; \
			git tag -a "$$NEW_VERSION" -m "Release $$NEW_VERSION"; \
			echo "Pushing tag to origin..."; \
			git push origin "$$NEW_VERSION"; \
			echo "Release $$NEW_VERSION created and pushed successfully!"; \
			;; \
		*) \
			echo "Release cancelled."; \
			;; \
	esac

# Create and push a minor release (increments minor version, resets patch to 0)
.PHONY: minor-release
minor-release:
	@echo "Creating minor release..."
	@LATEST_TAG=$$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"); \
	echo "Current latest tag: $$LATEST_TAG"; \
	VERSION_NUMS=$$(echo "$$LATEST_TAG" | sed 's/^v//'); \
	MAJOR=$$(echo "$$VERSION_NUMS" | cut -d. -f1); \
	MINOR=$$(echo "$$VERSION_NUMS" | cut -d. -f2); \
	PATCH=$$(echo "$$VERSION_NUMS" | cut -d. -f3); \
	MAJOR=$${MAJOR:-0}; MINOR=$${MINOR:-0}; PATCH=$${PATCH:-0}; \
	NEW_MINOR=$$((MINOR + 1)); \
	NEW_VERSION="v$$MAJOR.$$NEW_MINOR.0"; \
	echo "New version will be: $$NEW_VERSION"; \
	printf "Create and push release $$NEW_VERSION? [y/N] "; \
	read REPLY; \
	case "$$REPLY" in \
		[Yy]|[Yy][Ee][Ss]) \
			echo "Creating tag $$NEW_VERSION..."; \
			git tag -a "$$NEW_VERSION" -m "Release $$NEW_VERSION"; \
			echo "Pushing tag to origin..."; \
			git push origin "$$NEW_VERSION"; \
			echo "Release $$NEW_VERSION created and pushed successfully!"; \
			;; \
		*) \
			echo "Release cancelled."; \
			;; \
	esac

# Copy copilot instructions to Claude instructions file
.PHONY: copy-copilot-to-claude
copy-copilot-to-claude:
	@echo "Copying copilot instructions to Claude instructions file..."
	@cp .github/copilot-instructions.md CLAUDE.md
	@echo "✓ Copied .github/copilot-instructions.md to CLAUDE.md"

# Agent should run this task before finishing its turns
.PHONY: agent-finish
agent-finish: deps-dev fmt lint build test recompile
	@echo "Agent finished tasks successfully."

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build            - Build the binary for current platform"
	@echo "  build-all        - Build binaries for all platforms"
	@echo "  test             - Run tests"
	@echo "  test-coverage    - Run tests with coverage report"
	@echo "  clean            - Clean build artifacts"
	@echo "  deps             - Install dependencies"
	@echo "  lint             - Run linter"
	@echo "  fmt              - Format code"
	@echo "  fmt-check        - Check code formatting"
	@echo "  validate-template - Validate auto-compile workflow template"
	@echo "  validate-workflows - Validate compiled workflow lock files"
	@echo "  validate         - Run all validations (fmt-check, lint, validate-template, validate-workflows)"
	@echo "  install          - Install binary locally"
	@echo "  recompile        - Recompile all workflow files (depends on build)"
	@echo "  copy-copilot-to-claude - Copy copilot instructions to Claude instructions file"
	@echo "  agent-finish     - Complete validation sequence (build, test, recompile, fmt, lint)"
	@echo "  patch-release    - Create and push patch release (increments patch version)"
	@echo "  minor-release    - Create and push minor release (increments minor version, resets patch to 0)"
	@echo "  help             - Show this help message"