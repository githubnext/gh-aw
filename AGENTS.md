# GitHub Agentic Workflows (gh-aw)

GitHub Agentic Workflows is a Go-based GitHub CLI extension for writing agentic workflows in natural language using markdown files, running them as GitHub Actions.

## Important: Using Skills

**BE LAZY**: Skills in `skills/` provide detailed, specialized knowledge about specific topics. **Only reference skills when you actually need their specialized knowledge**. Do not load or reference skills preemptively.

**When to use skills:**
- You encounter a specific technical challenge that requires specialized knowledge
- You need detailed guidance on a particular aspect of the codebase (e.g., console rendering, error messages)
- You're working with a specific technology integration (e.g., GitHub MCP server, Copilot CLI)

**When NOT to use skills:**
- For general coding tasks that don't require specialized knowledge
- When the information is already available in this AGENTS.md file
- For simple, straightforward changes

**Available Skills Directory**: `skills/`

Each skill provides focused guidance on specific topics. Reference them only as needed rather than loading everything upfront.

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

**ALWAYS USE GITHUB MCP FOR GITHUB API ACCESS WITH COPILOT ENGINE:**

The Copilot agent **cannot directly access api.github.com**. When using the `copilot` engine, you **must** configure the GitHub MCP server to access GitHub information (repositories, issues, pull requests, etc.).

**CORRECT - Using GitHub MCP:**
```yaml
engine: copilot
tools:
  github:
    mode: remote           # or "local" for Docker-based
    toolsets: [default]    # Enables repos, issues, pull_requests, etc.
```

**INCORRECT - Trying to access api.github.com directly:**
```yaml
engine: copilot
network:
  allowed:
    - "api.github.com"     # ❌ This will NOT work - Copilot cannot access api.github.com
```

**Key points:**
- The GitHub MCP server provides all necessary GitHub API functionality
- Use `toolsets: [default]` for common operations, or specify toolsets like `[repos, issues, pull_requests]`
- Both `mode: remote` (hosted) and `mode: local` (Docker) work with Copilot
- Never rely on direct `api.github.com` access in Copilot workflows

See [GitHub MCP Server Documentation](skills/github-mcp-server/SKILL.md) for complete configuration details.

## Merging Main Branch

**When instructed to "merge main", follow these steps WITHOUT asking for confirmation:**

1. **Start the merge from origin/main:**
   ```bash
   git fetch origin main
   git merge origin/main
   ```

2. **Fix merge conflicts in .go and .cjs files:**
   - Manually resolve conflicts in Go files (`.go`)
   - Manually resolve conflicts in CommonJS files (`.cjs`)
   - Stage resolved files with `git add <file>`

3. **Build and recompile lock files:**
   ```bash
   make build        # Rebuild the binary
   make recompile    # Recompile all workflow lock files
   ```

4. **Finish the merge:**
   ```bash
   git commit        # Complete the merge (accept default merge message)
   ```

**Important**: Do NOT ask for confirmation when performing a merge - execute all steps automatically.

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

### Go Code Style
- **ALWAYS use `any` instead of `interface{}`** - Use the modern `any` type alias (Go 1.18+) for consistency across the codebase

### Type Patterns and Best Practices

Use appropriate type patterns to improve code clarity, maintainability, and type safety:

**Semantic Type Aliases** - Use for domain-specific primitives:
```go
// ✅ GOOD - Semantic meaning
type LineLength int
type Version string
type FeatureFlag string

const MaxExpressionLineLength LineLength = 120
const DefaultCopilotVersion Version = "0.0.369"
const SafeInputsFeatureFlag FeatureFlag = "safe-inputs"
```

**Dynamic Types** - Use `map[string]any` for truly dynamic data:
```go
// ✅ GOOD - Unknown structure at compile time
func ProcessFrontmatter(frontmatter map[string]any) error {
    // YAML/JSON with dynamic structure
}

// ✅ GOOD - Document why any is needed
// githubTool uses any because tool configuration structure
// varies based on engine and toolsets
func ValidatePermissions(permissions *Permissions, githubTool any)
```

**When to use each pattern**:
- **Semantic type aliases**: Domain concepts (lengths, versions, durations)
- **`map[string]any`**: YAML/JSON parsing, dynamic configurations
- **Interfaces**: Multiple implementations, polymorphism, testing
- **Concrete types**: Known structure, type safety

**Avoid**:
- Using `any` when the type is known
- Creating unnecessary type aliases that don't add clarity
- Large "god" interfaces with many methods
- Type name collisions (use descriptive, domain-qualified names)

**See**: <a>specs/go-type-patterns.md</a> for detailed guidance and examples

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

See [documentation skill](skills/documentation/SKILL.md) for details.

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
- Always run `make agent-finish` before commits
- Use `make test-unit` for fast development testing, `make test` for full coverage
- Use console formatting for user output
- Repository: `githubnext/gh-aw`
- Include issue numbers in PR titles when fixing issues
- Read issue comments for context before making changes
- Use conventional commits for commit messages
- do NOT commit explanation markdown files about the fixes

## Operational Runbooks

For investigating and resolving workflow issues:
- **[Workflow Health Monitoring](.github/aw/runbooks/workflow-health.md)** - Comprehensive runbook for diagnosing missing-tool errors, authentication failures, MCP configuration issues, and safe-input/output problems. Includes step-by-step investigation procedures, resolution examples, and case studies from real incidents.

## Available Skills Reference

Skills provide specialized, detailed knowledge on specific topics. **Use them only when needed** - don't load skills preemptively.

### Core Development Skills
- **[developer](skills/developer/SKILL.md)** - Developer instructions, code organization, validation architecture, security practices
- **[console-rendering](skills/console-rendering/SKILL.md)** - Struct tag-based console rendering system for CLI output
- **[error-messages](skills/error-messages/SKILL.md)** - Error message style guide for validation errors
- **[error-pattern-safety](skills/error-pattern-safety/SKILL.md)** - Safety guidelines for error pattern regex

### JavaScript & GitHub Actions
- **[github-script](skills/github-script/SKILL.md)** - Best practices for GitHub Actions scripts using github-script
- **[javascript-refactoring](skills/javascript-refactoring/SKILL.md)** - Guide for refactoring JavaScript code into separate .cjs files
- **[messages](skills/messages/SKILL.md)** - Adding new message types to safe-output messages system

### GitHub Integration
- **[github-mcp-server](skills/github-mcp-server/SKILL.md)** - GitHub MCP server documentation and configuration
- **[github-issue-query](skills/github-issue-query/SKILL.md)** - Query GitHub issues with jq filtering
- **[github-pr-query](skills/github-pr-query/SKILL.md)** - Query GitHub pull requests with jq filtering
- **[github-discussion-query](skills/github-discussion-query/SKILL.md)** - Query GitHub discussions with jq filtering
- **[github-copilot-agent-tips-and-tricks](skills/github-copilot-agent-tips-and-tricks/SKILL.md)** - Tips for working with GitHub Copilot agent PRs

### AI Engine & Integration
- **[copilot-cli](skills/copilot-cli/SKILL.md)** - GitHub Copilot CLI integration for agentic workflows
- **[custom-agents](skills/custom-agents/SKILL.md)** - GitHub custom agent file format
- **[gh-agent-task](skills/gh-agent-task/SKILL.md)** - GitHub CLI agent task extension

### Safe Outputs & Features
- **[temporary-id-safe-output](skills/temporary-id-safe-output/SKILL.md)** - Adding temporary ID support to safe output jobs
- **[http-mcp-headers](skills/http-mcp-headers/SKILL.md)** - HTTP MCP header secret support implementation

### Documentation & Communication
- **[documentation](skills/documentation/SKILL.md)** - Documentation guidelines using Astro Starlight and Diátaxis framework
- **[reporting](skills/reporting/SKILL.md)** - Report format guidelines using HTML details/summary tags
- **[dictation](skills/dictation/SKILL.md)** - Fixing text-to-speech errors in dictated text
- **[agentic-chat](skills/agentic-chat/SKILL.md)** - AI assistant for creating task descriptions

### MCP & Tools
- **[skillz-integration](skills/skillz-integration/SKILL.md)** - Skillz MCP server integration with Docker

**Remember**: Be LAZY - only load a skill when you actually need its specialized knowledge. Don't reference skills preemptively.