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
make test-unit   # Unit tests only (~2.5s, recommended for development)
make test        # All tests (~4s)
make test-integration # Integration tests only (~3.5s)
make recompile   # Recompile workflows
make agent-finish # Complete validation
```

### Manual Testing
```bash
./gh-aw --help
./gh-aw compile
./gh-aw list
./gh-aw mcp list      # MCP server management
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

### Build Times (Don't Cancel)
- `make agent-finish`: ~10-15s
- `make deps`: ~1.5min  
- `make deps-dev`: ~5-8min
- `make test`: ~4s
- `make lint`: ~5.5s

### Documentation

The documentation for this project is available in the `docs/` directory. It includes information on how to use the CLI, API references, and examples.
It uses the Astro Starlight system.

- neutral tone, not promotional
- avoid "we", "our", "us"
- avoid "Key Features" section
- avoid long list of bullet points
- use the `aw` language for agentic workflows snippets. It handles YAML frontmatter and markdown content.

    ```aw
    ---
    on: push
    ---
    # Your workflow steps here
    ```

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
```yaml
---
engine: claude  # Options: claude, codex, custom
tools:
  playwright:
    docker_image_version: "v1.41.0"
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
- **Integration tests**: Command behavior and binary compilation - run with `make test-integration`
- **Combined testing**: Use `make test` to run all tests (unit + integration) 
- **Workflow compilation tests**: Markdown to YAML conversion
- **Manual validation**: Always test after changes
- **Test agentic workflows**: Should be added to `pkg/cli/workflows` directory

**Recommended workflow**: Run `make test-unit` first for quick validation, then `make test-integration` or `make test` for complete coverage.

## Release Process
```bash
make minor-release  # Automated via GitHub Actions
```

## Quick Reference for AI Agents
- Go project with Makefile-managed build/test/lint
- Always run `make agent-finish` before commits
- Use `make test-unit` for fast development testing, `make test` for full coverage
- Use console formatting for user output
- Repository: `githubnext/gh-aw`
- Include issue numbers in PR titles when fixing issues
- Read issue comments for context before making changes