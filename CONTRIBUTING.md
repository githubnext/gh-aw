# Contributing to GitHub Agentic Workflows

Thank you for your interest in contributing to GitHub Agentic Workflows! We welcome contributions from the community and are excited to work with you.

## 🚀 Quick Start for Contributors

1. **Fork and clone the repository**
   ```bash
   git clone https://github.com/your-username/gh-aw.git
   cd gh-aw
   ```

2. **Set up the development environment**
   ```bash
   # Install dependencies
   make deps-dev
   
   # Build the project
   make build
   
   # Run tests to ensure everything works
   make test
   ```

3. **Make your changes and test them**
   ```bash
   # Format your code
   make fmt
   
   # Run linter
   make lint
   
   # Run tests
   make test
   
   # Compile workflows to ensure compatibility
   make recompile
   ```

4. **Submit your contribution**
   - Create a new branch for your feature or fix
   - Make your changes
   - Run `make agent-finish` to ensure all checks pass
   - Submit a pull request

## 🛠️ Development Setup

For detailed development setup instructions, see the [Development Guide](DEVGUIDE.md).

### Prerequisites
- Go 1.24.5 or later
- GitHub CLI (`gh`) installed and authenticated
- Git

### Build Commands
- `make deps` - Install basic dependencies
- `make deps-dev` - Install development dependencies (including linter)
- `make build` - Build the binary
- `make test` - Run tests
- `make lint` - Run linter
- `make fmt` - Format code
- `make agent-finish` - Run complete validation (build, test, recompile, format, lint)

## 📝 How to Contribute

### Reporting Issues
- Use the GitHub issue tracker to report bugs
- Include detailed steps to reproduce the issue
- Include version information (`./gh-aw version`)

### Suggesting Features
- Open an issue describing your feature request
- Explain the use case and how it would benefit users
- Include examples if applicable

### Contributing Code

#### Code Style
- Follow Go best practices and idioms
- Use `make fmt` to format your code
- Ensure `make lint` passes without errors
- Write tests for new functionality

#### Console Output
When adding CLI output, always use the styled console functions from `pkg/console`:

```go
import "github.com/githubnext/gh-aw/pkg/console"

// Use styled messages instead of plain fmt.Printf
fmt.Println(console.FormatSuccessMessage("Operation completed"))
fmt.Println(console.FormatInfoMessage("Processing workflow..."))
fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
```

#### File Organization

Follow these principles when organizing code:

- **Prefer many small files** over large monolithic files
- **Group by functionality**, not by type (avoid generic `utils.go` files)
- **Use descriptive names** that clearly indicate the file's purpose
- **Follow established patterns** from the codebase

**Key Patterns to Follow**:

1. **Create Functions Pattern** - One file per GitHub entity creation
   - Examples: `create_issue.go`, `create_pull_request.go`, `create_discussion.go`
   - Use when: Implementing new safe output types or GitHub API operations

2. **Engine Separation Pattern** - Each engine has its own file
   - Examples: `copilot_engine.go`, `claude_engine.go`, `codex_engine.go`
   - Shared helpers go in `engine_helpers.go`

3. **Focused Utilities Pattern** - Self-contained feature files
   - Examples: `expressions.go`, `strings.go`, `artifacts.go`
   - Keep files under 500 lines when possible

**File Placement**:
- Place new CLI commands in `pkg/cli/`
- Place workflow processing logic in `pkg/workflow/`
- Add tests alongside your code (e.g., `feature.go` and `feature_test.go`)
- Use descriptive test names: `feature_scenario_test.go`, `feature_integration_test.go`

**When to Create a New File**:
- Implementing a new safe output type → `create_<entity>.go`
- Adding a new AI engine → `<engine>_engine.go`
- Building a distinct feature module → `<feature>.go`
- Current file exceeds 800 lines → Split by logical boundaries

**File Size Guidelines**:
- Small files (50-200 lines): Utilities, simple features
- Medium files (200-500 lines): Most feature implementations
- Large files (500-800 lines): Complex features (consider splitting)
- Very large files (800+ lines): Core infrastructure only (refactor if possible)

For detailed guidance, see [Code Organization Patterns](docs/CODE_ORGANIZATION.md).

### Documentation
- Update documentation for any new features
- Add examples where helpful
- Ensure documentation is clear and concise

### Testing
- Write unit tests for new functionality
- Ensure all tests pass (`make test`)
- Test manually with real workflows when possible

## 🔄 Pull Request Process

1. **Before submitting:**
   - Run `make agent-finish` to ensure all checks pass
   - Test your changes manually
   - Update documentation if needed

2. **Pull request requirements:**
   - Clear description of what the PR does
   - Reference any related issues
   - Include tests for new functionality
   - Ensure CI passes

3. **Review process:**
   - Maintainers will review your PR
   - Address any feedback
   - Once approved, your PR will be merged

## 🏗️ Project Structure

```
/
├── cmd/gh-aw/           # Main CLI application
├── pkg/                 # Core Go packages
│   ├── cli/             # CLI command implementations
│   ├── console/         # Console formatting utilities
│   ├── parser/          # Markdown frontmatter parsing
│   └── workflow/        # Workflow compilation and processing
├── docs/                # Documentation
├── .github/workflows/   # Sample workflows and CI
└── Makefile             # Build automation
```

## 🤝 Community

- Join the `#continuous-ai` channel in the [GitHub Next Discord](https://gh.io/next-discord)
- Participate in discussions on GitHub issues
- Help other contributors and users

## 📜 Code of Conduct

This project follows the GitHub Community Guidelines. Please be respectful and inclusive in all interactions.

## ❓ Getting Help

- Read the [Development Guide](DEVGUIDE.md)
- Ask questions in GitHub issues or Discord
- Look at existing code and tests for examples

Thank you for contributing to GitHub Agentic Workflows! 🎉
