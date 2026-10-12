---
network:
  allowed:
    - go
mcp-scripts:
  go:
    description: "Execute any Go command. This tool is accessible as 'mcpscripts-go'. Provide arguments as an array of strings without the 'go' prefix (e.g., args: ['test', './...']). Each array element is one literal argument; do not add shell quoting."
    inputs:
      args:
        type: array
        items:
          type: string
        description: "Array of argument strings to pass to go CLI (without the 'go' prefix). Examples: ['test', './...'], ['build', './cmd/gh-aw'], ['mod', 'tidy'], ['fmt', './...'], ['vet', './...']"
        required: true
    script: |
      const { execFile } = require("node:child_process");
      const { promisify } = require("node:util");
      const { buildEnhancedError } = require("./mcp_handler_process.cjs");
      if (!Array.isArray(args) || !args.every(arg => typeof arg === "string")) {
        throw new Error("args must be an array of strings");
      }
      try {
        const { stdout, stderr } = await promisify(execFile)("go", args, { maxBuffer: 10 * 1024 * 1024 });
        return { stdout, stderr, outputs: {} };
      } catch (error) {
        throw buildEnhancedError(error, "go", error.stdout || "", error.stderr || "");
      }

  make:
    description: "Execute any Make target. This tool is accessible as 'mcpscripts-make'. Provide target names and options as an array of strings (e.g., args: ['fmt', 'build']). Each array element is one literal argument; do not add shell quoting."
    inputs:
      args:
        type: array
        items:
          type: string
        description: "Array of target names and option strings to pass to make. Examples: ['build'], ['test-unit'], ['lint'], ['recompile'], ['agent-finish'], ['fmt', 'build', 'test-unit']"
        required: true
    script: |
      const { execFile } = require("node:child_process");
      const { promisify } = require("node:util");
      const { buildEnhancedError } = require("./mcp_handler_process.cjs");
      if (!Array.isArray(args) || !args.every(arg => typeof arg === "string")) {
        throw new Error("args must be an array of strings");
      }
      try {
        const { stdout, stderr } = await promisify(execFile)("make", args, { maxBuffer: 10 * 1024 * 1024 });
        return { stdout, stderr, outputs: {} };
      } catch (error) {
        throw buildEnhancedError(error, "make", error.stdout || "", error.stderr || "");
      }
---

Both tools require `args` as an array of strings, not a command string.
For example, pass `["test", "./..."]` or
`["build", "-o", "path with spaces/output"]`. Each element is passed unchanged,
including spaces, quotes, backslashes, and empty strings. Do not add shell
quoting or escaping. Globs, variables, command substitutions, and shell
operators are not expanded or executed. The JavaScript wrappers use Node's
`execFile` without a shell or Python, return stdout and stderr, and report
command failures with their exit status and output.

**IMPORTANT — bash vs. MCP for validation:**

- **Use `mcpscripts-go` / `mcpscripts-make` early in a session** (within the first few minutes) for consistent, logged execution of Go and Make commands.
- **Use direct bash commands for end-of-session validation** (i.e., after any file-exploration or analysis phase that may have taken more than ~5 minutes). MCP connections time out after ~5 minutes of inactivity; calling `mcpscripts-*` at the end of a long session fails with `MCP error -32003: context canceled`.

**Validation via bash (preferred at end of session or after long exploration):**
```bash
make build
make test-unit
make recompile
```

**MCP tools (use early in session, before long exploration phases):**
```
Use the mcpscripts-go tool with args: ["test", "./..."]
Use the mcpscripts-make tool with args: ["build"]
Use the mcpscripts-make tool with args: ["lint"]
Use the mcpscripts-make tool with args: ["test-unit"]
```

**Incorrect**:
```
Use the go mcp-script tool with args: ["test", "./..."]  ❌ (Wrong tool name - use mcpscripts-go)
Use the mcpscripts-make tool with args: ["build"]        ❌ (after a long exploration phase — use bash instead)
Use the mcpscripts-go tool with args: "test ./..."       ❌ (Use an array of argument strings)
```

<!--
## mcpscripts-go and mcpscripts-make Tools

Safe-input tools that wrap Go and Make commands for consistent execution in agentic workflows.

### Usage

```yaml
imports:
  - shared/go-make.md
```

### Invocation

#### mcpscripts-go

The tool is accessible as `mcpscripts-go` (or `mcpscripts_go` after normalization). Provide go CLI arguments via the `args` array:

```
mcpscripts-go with args: ["test", "./..."]
mcpscripts-go with args: ["build", "./cmd/gh-aw"]
mcpscripts-go with args: ["mod", "tidy"]
mcpscripts-go with args: ["fmt", "./..."]
mcpscripts-go with args: ["vet", "./..."]
mcpscripts-go with args: ["test", "-v", "-run", "TestCompile", "./pkg/cli"]
```

The tool executes: `go <args>`

#### mcpscripts-make

The tool is accessible as `mcpscripts-make` (or `mcpscripts_make` after normalization). Provide make target(s) via the `args` array:

```
mcpscripts-make with args: ["build"]
mcpscripts-make with args: ["test-unit"]
mcpscripts-make with args: ["lint"]
mcpscripts-make with args: ["recompile"]
mcpscripts-make with args: ["fmt", "lint", "build"]
mcpscripts-make with args: ["agent-finish"]
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
