# GitHub Agentic Workflows (gh-aw)

GitHub Agentic Workflows is a Go-based GitHub CLI extension for writing agentic workflows in natural language using markdown files, running them as GitHub Actions.

## Critical Requirements

**ALWAYS RUN AGENT-FINISH BEFORE COMMITTING:**
```bash
make agent-finish  # Runs build, test, recompile, fmt, lint
```

**ALWAYS RUN RECOMPILE BEFORE COMMITTING CHANGES:**
```bash
make recompile     # Recompile all workflow files after code changes
```

**ALWAYS RUN MAKE RECOMPILE TO ENSURE JAVASCRIPT IS PROPERLY FORMATTED:**
```bash
make recompile     # Ensures JavaScript is properly formatted and workflows are compiled
```

**NEVER ADD LOCK FILES TO .GITIGNORE** - `.lock.yml` files are compiled workflows that must be tracked.

**ALWAYS REBUILD AFTER SCHEMA CHANGES:**
```bash
make build       # Rebuild gh-aw after modifying JSON schemas in pkg/parser/schemas/
```
Schema files are embedded in the binary using `//go:embed` directives, so changes require rebuilding the binary.

## Quick Setup

```bash
# Fresh clone setup
make deps        # ~1.5min first run  
make deps-dev    # +5-8min for linter
make build       # ~1.5s
./gh-aw --help
```

## Development Workflow

### Build & Test Commands
```bash
make fmt         # Format code (run before linting)
make lint        # ~5.5s
make test-unit   # Unit tests only (~25s, recommended for development)
make test        # All tests including integration tests (~30s)
make recompile   # Recompile workflows
make agent-finish # Complete validation
```

### Manual Testing
```bash
./gh-aw --help
./gh-aw compile
./gh-aw mcp list      # MCP server management
./gh-aw logs          # Download and analyze workflow logs
./gh-aw audit 123456  # Audit a specific workflow run
```

## Repository Structure

```
cmd/gh-aw/           # CLI entry point
pkg/
├── cli/             # Command implementations  
├── parser/          # Markdown frontmatter parsing
└── workflow/        # Workflow compilation
.github/workflows/   # Sample workflows (*.md + *.lock.yml)
```

## Console Message Formatting

**ALWAYS use console formatting for user output:**

```go
import "github.com/githubnext/gh-aw/pkg/console"

// Success, info, warning, error messages
fmt.Fprintln(os.Stderr, console.FormatSuccessMessage("Success!"))
fmt.Fprintln(os.Stderr, console.FormatInfoMessage("Info"))
fmt.Fprintln(os.Stderr, console.FormatWarningMessage("Warning"))
fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))

// Other types: CommandMessage, ProgressMessage, PromptMessage, 
// CountMessage, VerboseMessage, LocationMessage
```

**Error handling:**
```go
// WRONG
fmt.Fprintln(os.Stderr, err)

// CORRECT  
fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
```

**Logging Guidelines:**
- **ALWAYS** use `fmt.Fprintln(os.Stderr, ...)` or `fmt.Fprintf(os.Stderr, ...)` for CLI logging
- **NEVER** use `fmt.Println()` or `fmt.Printf()` directly - all output should go to stderr
- Use console formatting helpers with `os.Stderr` for consistent styling
- For simple messages without console formatting: `fmt.Fprintf(os.Stderr, "message\n")`

## Debug Logging

**ALWAYS use the logger package for debug logging:**

```go
import "github.com/githubnext/gh-aw/pkg/logger"

// Create a logger with namespace following pkg:filename convention
var log = logger.New("pkg:filename")

// Log debug messages (only shown when DEBUG environment variable matches)
log.Printf("Processing %d items", count)
log.Print("Simple debug message")

// Check if logging is enabled before expensive operations
if log.Enabled() {
    log.Printf("Expensive debug info: %+v", expensiveOperation())
}
```

**Category Naming Convention:**
- Follow the pattern: `pkg:filename` (e.g., `cli:compile_command`, `workflow:compiler`)
- Use colon (`:`) as separator between package and file/component name
- Be consistent with existing loggers in the codebase

**Debug Output Control:**
```bash
# Enable all debug logs
DEBUG=* gh aw compile

# Enable specific package
DEBUG=cli:* gh aw compile

# Enable multiple packages
DEBUG=cli:*,workflow:* gh aw compile

# Exclude specific loggers
DEBUG=*,-workflow:test gh aw compile

# Disable colors (auto-disabled when piping)
DEBUG_COLORS=0 DEBUG=* gh aw compile
```

**Key Features:**
- **Zero overhead**: Logs only computed when DEBUG matches the logger's namespace
- **Time diff**: Shows elapsed time between log calls (e.g., `+50ms`, `+2.5s`)
- **Auto-colors**: Each namespace gets a consistent color in terminals
- **Pattern matching**: Supports wildcards (`*`) and exclusions (`-pattern`)

**When to Use:**
- Non-essential diagnostic information
- Performance insights and timing data
- Internal state tracking during development
- Detailed operation flow for debugging

**When NOT to Use:**
- Essential user-facing messages (use console formatting instead)
- Error messages (use `console.FormatErrorMessage`)
- Success/warning messages (use console formatting)
- Final output or results (use stdout/console formatting)

## Development Guidelines

### Code Organization
- Prefer many smaller files grouped by functionality
- Add new files for new features rather than extending existing ones
- Use console formatting instead of plain fmt.* for CLI output

### GitHub Actions Integration  
For JavaScript files in `pkg/workflow/js/*.cjs`:
- Use `core.info`, `core.warning`, `core.error` (not console.log)
- Use `core.setOutput`, `core.getInput`, `core.setFailed`
- Avoid `any` type, use specific types or `unknown`
- Run `make js` and `make lint-cjs` for validation

### Schema Changes
When modifying JSON schemas in `pkg/parser/schemas/`:
- Schema files are embedded using `//go:embed` directives
- **MUST rebuild the binary** with `make build` for changes to take effect
- Test changes by compiling a workflow: `./gh-aw compile test-workflow.md`
- Schema changes typically require corresponding Go struct updates

### Build Times (Don't Cancel)
- `make agent-finish`: ~10-15s
- `make deps`: ~1.5min  
- `make deps-dev`: ~5-8min
- `make test`: ~4s
- `make lint`: ~5.5s

### Documentation

The documentation for this project is available in the `docs/` directory. It includes information on how to use the CLI, API references, and examples.
It uses the Astro Starlight system and Diátaxis framework.

See [documentation instructions](.github/instructions/documentation.instructions.md) for details.

### Legacy Support

This project is still in an experimental phase. When you are requested to make a change, do not add fallback or legacy support unless explicitly instructed.

## Key Features

### MCP Server Management
```bash
gh aw mcp list                    # List workflows with MCP servers
gh aw mcp inspect workflow-name   # Inspect MCP servers
gh aw mcp inspect --inspector     # Web-based inspector
```

**Default MCP Registry**: Uses GitHub's MCP registry at `https://api.mcp.github.com/v0` by default.

### AI Engine Support
```aw
---
engine: copilot  # Options: copilot, claude, codex, custom
tools:
  playwright:
    version: "v1.41.0"
    allowed_domains: ["github.com"]
---
```

### Playwright Integration
- Containerized browser automation
- Domain-restricted network access
- Accessibility analysis and visual testing
- Multi-browser support (Chromium, Firefox, Safari)

## Testing Strategy
- **Unit tests**: All packages have coverage - run with `make test-unit` for fast feedback during development
- **Integration tests**: Command behavior and binary compilation - run all tests with `make test`
- **Combined testing**: Use `make test` to run all tests (unit + integration) 
- **Workflow compilation tests**: Markdown to YAML conversion
- **Manual validation**: Always test after changes
- **Test agentic workflows**: Should be added to `pkg/cli/workflows` directory

**Recommended workflow**: Run `make test-unit` first for quick validation, then `make test` for complete coverage.

## Release Process
```bash
make minor-release  # Automated via GitHub Actions
```

## Quick Reference for AI Agents
- Go project with Makefile-managed build/test/lint
- **ALWAYS run `make fmt` and commit before returning to the user**
- Always run `make agent-finish` before commits
- Use `make test-unit` for fast development testing, `make test` for full coverage
- Use console formatting for user output
- Repository: `githubnext/gh-aw`
- Include issue numbers in PR titles when fixing issues
- Read issue comments for context before making changes
- do NOT commit explanation markdown files about the fixes