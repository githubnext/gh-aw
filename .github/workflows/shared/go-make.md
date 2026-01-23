---
safe-inputs:
  go:
    description: "Execute any Go command. This tool is accessible as 'safeinputs-go'. Provide the full command after 'go' (e.g., args: 'test ./...'). The tool will run: go <args>. Use single quotes ' for complex args to avoid shell interpretation issues."
    inputs:
      args:
        type: string
        description: "Arguments to pass to go CLI (without the 'go' prefix). Examples: 'test ./...', 'build ./cmd/gh-aw', 'mod tidy', 'fmt ./...', 'vet ./...'"
        required: true
    run: |
      echo "go $INPUT_ARGS"
      go $INPUT_ARGS

  make:
    description: "Execute any Make target. This tool is accessible as 'safeinputs-make'. Provide the target name(s) (e.g., args: 'build'). The tool will run: make <args>. Use single quotes ' for complex args to avoid shell interpretation issues."
    inputs:
      args:
        type: string
        description: "Arguments to pass to make (target names and options). Examples: 'build', 'test-unit', 'lint', 'recompile', 'agent-finish', 'fmt build test-unit'"
        required: true
    run: |
      echo "make $INPUT_ARGS"
      make $INPUT_ARGS
---

**IMPORTANT**: Always use the `safeinputs-go` and `safeinputs-make` tools for Go and Make commands instead of running them directly via bash. These safe-input tools provide consistent execution and proper logging.

**Correct**:
```
Use the safeinputs-go tool with args: "test ./..."
Use the safeinputs-make tool with args: "build"
Use the safeinputs-make tool with args: "lint"
Use the safeinputs-make tool with args: "test-unit"
```

**Incorrect**:
```
Use the go safe-input tool with args: "test ./..."  ❌ (Wrong tool name - use safeinputs-go)
Run: go test ./...  ❌ (Use safeinputs-go instead)
Execute bash: make build  ❌ (Use safeinputs-make instead)
```

<!--
## safeinputs-go and safeinputs-make Tools

Safe-input tools that wrap Go and Make commands for consistent execution in agentic workflows.

### Usage

```yaml
imports:
  - shared/go-make.md
```

### Invocation

#### safeinputs-go

The tool is accessible as `safeinputs-go` (or `safeinputs_go` after normalization). Provide go CLI arguments via the `args` parameter:

```
safeinputs-go with args: "test ./..."
safeinputs-go with args: "build ./cmd/gh-aw"
safeinputs-go with args: "mod tidy"
safeinputs-go with args: "fmt ./..."
safeinputs-go with args: "vet ./..."
safeinputs-go with args: "test -v -run TestCompile ./pkg/cli"
```

The tool executes: `go <args>`

#### safeinputs-make

The tool is accessible as `safeinputs-make` (or `safeinputs_make` after normalization). Provide make target(s) via the `args` parameter:

```
safeinputs-make with args: "build"
safeinputs-make with args: "test-unit"
safeinputs-make with args: "lint"
safeinputs-make with args: "recompile"
safeinputs-make with args: "fmt lint build"
safeinputs-make with args: "agent-finish"
```

The tool executes: `make <args>`

### Common Make Targets

From the gh-aw Makefile:

- **build** - Build the gh-aw binary
- **test** - Run all tests (unit + integration)
- **test-unit** - Run unit tests only (~25s, fast feedback)
- **test-integration-*** - Run specific integration test groups
- **lint** - Run linters
- **fmt** - Format code (Go, JavaScript, JSON)
- **recompile** - Recompile all workflow lock files
- **agent-finish** - Complete validation (fmt, lint, build, test, recompile)
- **fix** - Run gh-aw fix on all workflows
- **clean** - Remove build artifacts
- **deps** - Install Go dependencies
- **deps-dev** - Install development dependencies (linters)

### Common Go Commands

- **go test ./...** - Run all tests
- **go test ./pkg/...** - Run tests in pkg directory
- **go build ./cmd/gh-aw** - Build the main binary
- **go mod tidy** - Clean up go.mod and go.sum
- **go fmt ./...** - Format all Go code
- **go vet ./...** - Run Go vet static analyzer
- **go test -v -run TestName ./pkg/cli** - Run specific test
-->
