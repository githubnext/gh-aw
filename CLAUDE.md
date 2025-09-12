# GitHub Agentic Workflows (gh-aw)

GitHub Agentic Workflows is a Go-based GitHub CLI extension that enables writing agentic workflows in natural language using markdown files, and running them as GitHub Actions workflows.

**ALWAYS FOLLOW THESE INSTRUCTIONS FIRST** and only fallback to additional search and context gathering if the information here is incomplete or found to be in error.

## Critical Requirements

**ALWAYS RUN AGENT-FINISH TASK BEFORE FINISHING ANY SESSION OR COMMITTING CHANGES**

Before concluding any development session or making any commits, you MUST:
```bash
make agent-finish
```
This runs the complete validation sequence: `build`, `test`, `recompile`, and includes formatting and linting. This is a non-negotiable requirement.

**NEVER ADD LOCK FILES TO .GITIGNORE**

Never add `*.lock.yml` files to `.gitignore`. These are compiled workflow files that must be tracked in the repository as they represent the actual GitHub Actions workflows that get executed. Ignoring them would break the workflow system.

## Working Effectively

### Bootstrap and Build Environment
Execute these commands in exact order on a fresh clone:

```bash
# Clone and navigate to repository
git clone https://github.com/githubnext/gh-aw.git
cd gh-aw

# Install basic dependencies - NEVER CANCEL: Takes ~1.5 minutes for first run (refreshes go.sum too)
make deps
# TIMEOUT WARNING: Set timeout to 15+ minutes for dependency installation

# For full development including linter (adds ~5-8 minutes)
make deps-dev

# Build the binary - fast build (~1.5 seconds)
make build

# Verify the build works
./gh-aw --help
./gh-aw version
```

### Development Dependencies Setup
The repository requires these tools to be installed and accessible:

```bash
# Go 1.24.5+ is required (check with: go version)
# GitHub CLI is required (check with: gh --version)

# Ensure Go bin directory is in PATH for linting tools
export PATH=$PATH:$(go env GOPATH)/bin

# Verify all tools are available
which golangci-lint  # Should be available after 'make deps-dev'
which gh             # Should be available system-wide

# Optional: For local workflow execution (not required for development)
# which claude       # Claude AI processor (for claude workflows)
# which codex        # Codex AI processor (for codex workflows)
```

### Environment Variables and Configuration
```bash
# Recommended: Add Go bin to your shell profile
echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.bashrc
# Or for zsh: echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.zshrc

# For GitHub CLI operations
gh auth status  # Must be authenticated for extension installation
```

### Running Tests and Quality Checks

```bash
# Run comprehensive test suite - NEVER CANCEL: Takes ~4 seconds
make test
# TIMEOUT WARNING: Set timeout to 10+ minutes for safety

# Run tests with coverage - generates coverage.html report
make test-coverage

# Fix code formatting first - ALWAYS run before linting
make fmt

# Run linter - NEVER CANCEL: Takes ~5.5 seconds  
make lint
# TIMEOUT WARNING: Set timeout to 10+ minutes for safety

# Check code formatting (very fast ~0.3 seconds)
make fmt-check
```

### Installing the Extension Locally for Testing

```bash
# Requires GitHub CLI authentication
gh auth status  # Verify you're authenticated first

# Build and install the extension locally
make install

# Verify installation worked
gh aw --help
gh aw version
```

## Validation and Testing

### Manual Functionality Testing
**CRITICAL**: After making any changes, always build the compiler, and validate functionality with these steps:

```bash
# 1. Test basic CLI interface
./gh-aw --help
./gh-aw list
./gh-aw version

# 2. Test workflow compilation (works with existing .github/workflows/*.md files)
./gh-aw compile

# 3. After CLI changes: Recompile all workflows to ensure compatibility
make recompile

# 4. Test status command (shows installed workflows)
./gh-aw status
# Expected: Lists local workflow files with installation status

# 5. Test add command help and syntax
./gh-aw add --help

# 6. Test with sample workflow repository (requires internet access)
./gh-aw list --packages
# Note: May show "No workflows or packages found" if no packages installed

# 7. Validate build artifacts are clean
make clean
make build

# 8. MANDATORY: Run agent-finish before finishing session
make agent-finish  # Runs build, test, recompile, fmt, and lint
```

### End-to-End Workflow Testing
For testing complete workflows (requires GitHub authentication):

```bash
# Install sample workflows from the agentics repository
gh aw install githubnext/agentics

# List available workflows
gh aw list

# Add a workflow (creates PR if --pr flag used)
gh aw add weekly-research --pr

# Test workflow compilation
gh aw compile weekly-research
```

### Build Time Expectations
**NEVER CANCEL these operations** - they are expected to take this long:

- **Agent finish (`make agent-finish`)**: ~10-15 seconds (runs build, test, recompile, fmt, lint)
- **Basic dependencies (`make deps`)**: ~1.5 minutes (first run)
- **Full development deps (`make deps-dev`)**: ~5-8 minutes (first run, includes linter)
- **Build (`make build`)**: ~1.5 seconds (very fast)
- **Recompile workflows (`make recompile`)**: ~2-3 seconds (depends on number of workflows)
- **Test suite (`make test`)**: ~4 seconds  
- **Test with coverage (`make test-coverage`)**: ~3 seconds
- **Formatting (`make fmt`)**: ~0.3 seconds **[ALWAYS RUN FIRST - BEFORE LINTING]**
- **Linting (`make lint`)**: ~5.5 seconds **[ALWAYS REQUIRED - NEVER SKIP]**
- **Format check (`make fmt-check`)**: ~0.3 seconds

## Repository Structure and Navigation

### Key Directories and Files
```
/
├── cmd/gh-aw/           # Main CLI application entry point
│   └── main.go          # CLI commands and cobra setup
├── pkg/                 # Core Go packages
│   ├── cli/             # CLI command implementations
│   ├── parser/          # Markdown frontmatter parsing
│   └── workflow/        # Workflow compilation and processing
├── .github/workflows/   # Sample agentic workflow files (*.md) and compiled versions (*.lock.yml)
├── docs/                # Documentation files
├── Makefile             # Build automation (PRIMARY BUILD TOOL)
├── go.mod               # Go module dependencies (Go 1.24.5+)
├── go.sum               # Go module checksums
├── DEVGUIDE.md          # Comprehensive developer documentation
├── TESTING.md           # Testing framework documentation
└── README.md            # User-facing documentation
```

### Split Code in Files By Features

Prefer generating many smaller files
grouped by functionality/feature than adding more and more code to the same existing files.

If you are implementing a new feature, prefer adding
a new file.

### Generated Workflow Style

The generated code produced by the compiler is a GitHub Action Workflow.
Here are additional guidelines to consider when designing a feature that generates workflows:

1. **Prefer JavaScript actions over shell scripts** for GitHub API interactions
2. **Use @actions/core utilities** for proper workflow integration

### Important Files to Check When Making Changes
- **Always check** `cmd/gh-aw/main.go` after modifying CLI commands
- **Always check** `pkg/cli/` after changing command implementations  
- **Always check** `pkg/workflow/compiler.go` after changing workflow compilation
- **Always check** `Makefile` after changing build or test processes
- **Always check** `.github/workflows/ci.yml` for CI pipeline changes

## Console Message Formatting

The gh-aw CLI provides rich console formatting functionality through the `console` package (`pkg/console/console.go`). **ALWAYS use these styled functions instead of plain fmt.Printf or fmt.Println for user-facing output**.

### Available Console Functions

Import the console package:
```go
import "github.com/githubnext/gh-aw/pkg/console"
```

#### Core Message Types
```go
// Success Messages (✓ in green)
fmt.Println(console.FormatSuccessMessage("Workflow compilation completed successfully"))

// Information Messages (ℹ in blue)
fmt.Println(console.FormatInfoMessage("Starting compilation of: example.md"))
fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Output file: %s", lockFile)))

// Warning Messages (⚠ in yellow)
fmt.Println(console.FormatWarningMessage("Schema validation available but skipped"))

// Error Messages (✗ in red)
fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
```

#### Specialized Message Types
```go
// Command Execution (⚡ in purple)
fmt.Println(console.FormatCommandMessage("gh workflow run example.yml"))

// Progress/Activity (🔨 in yellow)
fmt.Println(console.FormatProgressMessage("Compiling workflow files..."))

// User Prompts (❓ in green)
fmt.Print(console.FormatPromptMessage("Are you sure you want to continue? [y/N]: "))

// Counts/Statistics (📊 in cyan)
fmt.Println(console.FormatCountMessage(fmt.Sprintf("Found %d workflows to compile", count)))

// Verbose/Debug Output (🔍 in gray, italic)
if verbose {
    fmt.Println(console.FormatVerboseMessage("Debug: Parsing frontmatter section"))
}

// File/Directory Locations (📁 in orange)
fmt.Println(console.FormatLocationMessage(fmt.Sprintf("Workflow saved to: %s", filePath)))
```

#### List Formatting
```go
// List Headers (underlined green)
fmt.Println(console.FormatListHeader("Available Workflows"))
fmt.Println(console.FormatListHeader("=================="))

// List Items (• prefix)
fmt.Println(console.FormatListItem("weekly-research.md"))
fmt.Println(console.FormatListItem("daily-plan.md"))
```

#### Error Handling for CLI Commands
**CRITICAL**: Always use styled error messages for stderr output:
```go
// WRONG - Plain error output
fmt.Fprintln(os.Stderr, err)

// RIGHT - Styled error output  
fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))

// For structured errors with position info
fmt.Fprintln(os.Stderr, console.FormatError(console.CompilerError{
    Type:    "error",
    Message: fmt.Sprintf("running workflow: %v", err),
}))
```

### Usage Guidelines

**DO**: Always use console formatting functions for user-facing output
**DON'T**: Use plain fmt.Printf, fmt.Println, or fmt.Fprintln for CLI output
**CRITICAL**: Error outputs to stderr MUST use console.FormatErrorMessage() or console.FormatError()

**Before (WRONG)**:
```go
fmt.Printf("Installing package: %s\n", repo)
fmt.Println("Warning: Could not find workflow")
fmt.Fprintln(os.Stderr, err)
```

**After (CORRECT)**:
```go
fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Installing package: %s", repo)))
fmt.Println(console.FormatWarningMessage("Could not find workflow"))
fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
```

#### Error Formatting
For structured compiler errors with position information:
```go
err := console.CompilerError{
    Position: console.ErrorPosition{
        File:   "workflow.md",
        Line:   15,
        Column: 5,
    },
    Type:    "error", // "error", "warning", "info"
    Message: "Invalid frontmatter syntax",
    Context: []string{
        "---",
        "on: invalid syntax here",
        "permissions:",
    },
    Hint:    "Check YAML syntax and ensure proper indentation",
}
fmt.Print(console.FormatError(err))
```

This produces Rust-like error output with:
- IDE-parseable format: `file:line:column: type: message`
- Syntax highlighting for error location
- Context lines with line numbers
- Pointer indicating exact error position
- Optional hints for fixing the error

#### YAML Error Extraction
For YAML parsing errors, extract line and column information:
```go
line, column, message := console.ExtractYAMLError(yamlErr, frontmatterStartLine)
```

### Console Styling Features

- **TTY Detection**: Automatically detects if output is to a terminal and applies styling accordingly
- **IDE Integration**: Error format is parseable by IDEs for jump-to-error functionality  
- **Color Themes**: Uses carefully chosen colors that work in both light and dark terminals
- **Accessibility**: Includes symbolic prefixes (✓, ℹ, ⚠) for screen readers

### Usage Examples in Workflow Compilation

Example from `pkg/workflow/compiler.go`:
```go
if verbose {
    fmt.Println(console.FormatInfoMessage("Parsing workflow file..."))
}

// After successful parsing
fmt.Println(console.FormatSuccessMessage("Successfully parsed frontmatter and markdown content"))
fmt.Println(console.FormatInfoMessage(fmt.Sprintf("Workflow name: %s", workflowData.Name)))

// For validation warnings
fmt.Println(console.FormatWarningMessage("Schema validation available but skipped (use SetSkipValidation(false) to enable)"))
```

## Development Workflow

### Quick Reference for AI Agents
- You are working in Go project.
- Build, testing, linting, and formatting are managed by the `Makefile`.
- Always run linter and formatter on your code before creating a pull request.
- Use the `github` tool to interact with GitHub repositories.
- The current repository is `githubnext/gh-aw`.
- Before creating a pull request, ensure that all tests pass and the code is well-documented.
- Before creating a pull request make sure to change the branch name
- If you were told to fix the issue, make sure to include the issue number in the pull request title, and reference the issue in the pull request description.
- If you were told to fix the issue, remember to read all comments on the issue before making changes to understand context.

### Before Committing Changes
**CRITICAL: ALWAYS run the agent-finish task before ANY commit:**

```bash
# Run the complete validation sequence
make agent-finish
# This automatically runs: build, test, recompile, fmt, and lint

# 4. Manual functionality validation
./gh-aw --help
./gh-aw compile
./gh-aw list

# 5. Verify no unwanted files are included
git status
git diff --name-only
```

**The `make agent-finish` task is mandatory and includes all necessary build, test, format, and lint steps. It cannot be skipped under any circumstances.**

### CI/CD Pipeline Expectations
The repository uses GitHub Actions for CI/CD:

- **CI pipeline** (`.github/workflows/ci.yml`): Runs on every push/PR
  - Tests all packages with coverage
  - Runs linting and format checks
  - Expected time: ~2-5 minutes

- **Release pipeline** (`.github/workflows/release.yml`): Triggered by version tags
  - Builds cross-platform binaries
  - Creates GitHub releases automatically
  - Uses `gh-extension-precompile` for GitHub CLI extensions

## Troubleshooting Common Issues

### Build Issues
```bash
# Clean and rebuild from scratch
make clean
go clean -cache
make deps-dev
make build
```

### Test Failures
```bash
# Run specific test with verbose output
go test ./pkg/cli -v -run TestSpecificFunction

# Check dependency consistency
go mod verify
go mod tidy

# Run tests individually by package
go test ./pkg/cli -v
go test ./pkg/parser -v  
go test ./pkg/workflow -v
```

### Linter Issues
```bash
# Ensure PATH includes Go bin directory
export PATH=$PATH:$(go env GOPATH)/bin

# Check if golangci-lint is installed
which golangci-lint || make deps-dev

# Fix common formatting issues
make fmt

# Run linter with specific config
golangci-lint run --config .golangci.yml
```

### GitHub CLI Integration Issues
```bash
# Verify GitHub CLI authentication
gh auth status

# Re-authenticate if needed
gh auth login

# Test GitHub CLI extension installation
gh extension list | grep gh-aw
```

## Project-Specific Information

### Core Functionality
The gh-aw tool provides:
- **Markdown workflow parsing**: Converts natural language workflows from markdown to GitHub Actions YAML
- **Workflow management**: Add, remove, enable, disable agentic workflows
- **AI processor integration**: Supports Claude, Codex, and other AI processors
- **Local execution**: Run workflows locally for testing
- **Package management**: Install workflow packages from GitHub repositories

### AI Processing Configuration
Workflows support different AI processors via frontmatter:
```yaml
---
engine: claude  # Options: claude, codex
on:
  schedule:
    - cron: "0 9 * * 1"
---
```

### Workflow File Structure
- **`.md` files**: Natural language workflow definitions in `.github/workflows/`
- **`.lock.yml` files**: Compiled GitHub Actions YAML (generated by `gh aw compile`)
- **Shared components**: Reusable workflow components in `.github/workflows/shared/`

### Testing Strategy
The project uses comprehensive testing at multiple levels:
- **Unit tests**: All packages have test coverage (see TESTING.md)
- **CLI integration tests**: Verify command behavior and error handling
- **Workflow compilation tests**: Validate markdown to YAML conversion
- **Interface stability tests**: Ensure API compatibility

### Known Limitations
- **GitHub CLI authentication required** for `make install` and most `gh aw` commands
- **Internet access required** for installing workflow packages
- **Local workflow execution** may have different permissions than GitHub Actions
- **Some operations require git working directory to be clean** (especially PR creation)

## Release Process

### Creating Releases
Releases are automated via GitHub Actions:

```bash
# Create a new release (patch version increment)
make minor-release

# Or manually create and push a tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

The release process automatically:
- Builds cross-platform binaries
- Creates GitHub release with artifacts
- Updates GitHub CLI extension registry

### Version Information
Version details are automatically injected at build time:
- Version comes from git tags
- Commit hash and build date included
- No manual version file updates needed