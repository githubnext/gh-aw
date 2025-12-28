# Developer Guide

## Development Environment Setup

### 1. Clone and Setup Repository

```bash
# Clone the repository
git clone https://github.com/githubnext/gh-aw.git
cd gh-aw
```

### 2. Install Development Dependencies

```bash
# Install basic Go dependencies
make deps

# For full development (including linter)
make deps-dev
```

### 3. Build and Verify Development Environment

```bash
# Verify GitHub CLI is authenticated
gh auth status

# Run all tests to ensure everything works
make test

# Check code formatting
make fmt-check

# Run linter (may require golangci-lint installation)
make lint

# Build and test the binary
make build
./gh-aw --help

# Build the awmg (MCP gateway) standalone binary
make build-awmg
./awmg --help

# Build both binaries
make all
```

### 4. Install the Extension Locally for Testing

```bash
# Install the local version of gh-aw extension
make install

# Verify installation
gh aw --help
```

## Build Tools

This project uses `tools.go` to track build-time tool dependencies. This ensures everyone uses the same tool versions.

### Install Tools

```bash
make tools
```

This installs all tools listed in `tools.go` at the versions specified in `go.mod`:

- **golangci-lint**: Go linter with comprehensive checks
- **actionlint**: GitHub Actions workflow linter
- **gosec**: Go security linter
- **gopls**: Go language server for IDE support
- **govulncheck**: Go vulnerability scanner

### Adding a New Tool

1. Add blank import to `tools.go`:
   ```go
   _ "github.com/example/tool/cmd/tool"
   ```

2. Update dependencies:
   ```bash
   go get github.com/example/tool/cmd/tool@latest
   go mod tidy
   ```

3. Install: `make tools`

### Tool Version Management

Tool versions are locked in `go.mod` and `go.sum`, ensuring:

- **Consistency**: Same tool versions in CI and local development
- **Reproducibility**: Tool versions are version-controlled
- **Simplicity**: Single command to install all tools
- **Discoverability**: `tools.go` shows all build tools at a glance

```bash
# Install the local version of gh-aw extension
make install

# Verify installation
gh aw --help
```

## Testing

### Test Structure

The project has comprehensive testing at multiple levels:

#### Unit Tests
```bash
# Run specific package tests
go test ./pkg/cli -v
go test ./pkg/parser -v  
go test ./pkg/workflow -v

# Run all unit tests
make test
```

#### End-to-End Tests
```bash
# Comprehensive test validation
make test-script
```

### Adding New Tests

1. **Unit tests**: Add to `pkg/*/package_test.go`
2. **Follow existing patterns**: Look at current tests for structure

### CI Test Artifacts

The CI workflow generates JSON test result artifacts with timing information that can be downloaded and analyzed:

#### Available Artifacts

- **test-result-unit.json**: Unit test results with timing data
- **test-result-integration-*.json**: Integration test results for each test group

#### JSON Format

Each test result file contains newline-delimited JSON (ndjson) with test events:

```json
{"Time":"2025-12-12T13:17:30Z","Action":"pass","Package":"github.com/githubnext/gh-aw/pkg/logger","Elapsed":0.022}
```

Key fields:
- `Time`: ISO 8601 timestamp
- `Action`: Test event (`start`, `run`, `pass`, `fail`, `output`)
- `Package`: Go package being tested
- `Test`: Test name (if applicable)
- `Elapsed`: Test duration in seconds
- `Output`: Test output (for `output` actions)

#### Analyzing Test Timing

To extract timing information from artifacts:

```bash
# Download artifacts from a workflow run
gh run download <run-id>

# Extract slowest tests
cat test-result-unit.json | jq -r 'select(.Action == "pass" and .Test != null) | "\(.Elapsed)s \(.Test)"' | sort -rn | head -20

# Get package-level timing
cat test-result-unit.json | jq -r 'select(.Action == "pass" and .Test == null) | "\(.Elapsed)s \(.Package)"' | sort -rn
```

#### Mining Test Data

The JSON format enables various analyses:
- Identify slow tests across multiple runs
- Track test performance trends over time
- Detect flaky tests by comparing results
- Generate test execution reports

## Debugging and Troubleshooting

### Common Development Issues

#### Build Failures
```bash
# Clean and rebuild
make clean
make deps-dev  # Use deps-dev for full development dependencies
make build
```

#### Test Failures
```bash
# Run specific test with verbose output
go test ./pkg/cli -v -run TestSpecificFunction

# Check test dependencies
go mod verify
go mod tidy
```

#### Linter Issues
```bash
# Fix formatting issues
make fmt

# Address linter warnings
make lint

# Validate workflows with actionlint
make actionlint
```

### Local Incremental Linting

Speed up linting by only checking changed files:

```bash
# Lint changes since origin/main
make golint-incremental BASE_REF=origin/main

# This is what CI uses on PRs - 50-75% faster!
```

This runs the same incremental linting strategy as CI, checking only files changed since the base reference. It's particularly useful when working on pull requests where you want quick feedback on your changes without waiting for a full repository scan.

The incremental approach uses `golangci-lint --new-from-rev` to analyze only the files that differ from the specified base reference, providing significant performance improvements:
- **Full lint** (`make lint`): Scans entire repository
- **Incremental lint** (`make golint-incremental`): Scans only changed files - typically 50-75% faster on PRs

**When to use each approach:**
- Use `make golint-incremental BASE_REF=origin/main` during development for fast feedback
- Use `make lint` before final commits to ensure comprehensive coverage

## Security Scanning

The project includes automated security scanning to detect vulnerabilities, code smells, and dependency issues.

### Running Security Scans Locally

```bash
# Run all security scans (gosec, govulncheck, trivy)
make security-scan

# Run individual scans
make security-gosec      # Go security linter
make security-govulncheck # Go vulnerability database check
make security-trivy       # Filesystem/dependency scanner (requires trivy)
```

### Security Scan Tools

- **gosec**: Static analysis tool for Go that detects security issues in source code
- **govulncheck**: Official Go tool that checks for known vulnerabilities in dependencies
- **trivy**: Comprehensive scanner for filesystem vulnerabilities, misconfigurations, and secrets

### Interpreting Results

#### Gosec Results
- Results are saved to `gosec-report.json`
- Review findings by severity (HIGH, MEDIUM, LOW)
- False positives can be suppressed with `// #nosec G<rule-id>` comments

#### Govulncheck Results
- Shows vulnerabilities in direct and indirect dependencies
- Indicates if vulnerable code paths are actually called
- Update affected dependencies to resolve issues

#### Trivy Results
- Displays HIGH and CRITICAL severity findings
- Covers Go dependencies, npm packages, and configuration files
- Shows CVE details and available fix versions

### Suppressing False Positives

#### Gosec
```go
// Suppress a specific rule
// #nosec G104
err := someFunction() // Error explicitly ignored

// Suppress multiple rules
// #nosec G101 G102
secret := "example" // Known test value
```

#### Govulncheck
- No inline suppression available
- Update dependencies or document accepted risks in security review

#### Trivy
- Use `.trivyignore` file to exclude specific CVEs:
```text
# .trivyignore
CVE-2023-XXXXX  # False positive: not exploitable in our usage
```

### CI/CD Integration

Security scans run automatically on:
- Daily scheduled scan (6:00 AM UTC)
- Manual workflow dispatch

Results are uploaded to the GitHub Security tab in SARIF format.

### Security Scanning Exclusions

For comprehensive documentation of gosec security exclusions, see **<a>Gosec Security Exclusions</a>**.

This documentation provides:
- Complete list of global and file-specific exclusions
- CWE mappings for compliance tracking
- Detailed rationale and mitigation strategies
- Suppression guidelines for `#nosec` annotations
- Compliance and audit trail information

### Development Tips

1. **Use verbose testing**: `go test -v` for detailed output
2. **Run tests frequently**: Ensure changes don't break existing functionality
3. **Check formatting**: Run `make fmt` before committing
4. **Validate thoroughly**: Use `go run test_validation.go` before pull requests

## Release Process

## Architectural Patterns

Understanding the architectural patterns used in this codebase will help you make consistent contributions.

### Core Design Patterns

#### 1. Create Functions Pattern

The codebase uses a consistent pattern for GitHub API operations:

- **One file per entity type**: `create_issue.go`, `create_pull_request.go`, `create_discussion.go`
- **Consistent structure**: Configuration parsing, validation, job generation
- **Parallel development**: Each creation type is independent

**Example Structure**:
```go
// In create_issue.go
type CreateIssuesConfig struct { ... }
func (c *Compiler) parseIssuesConfig(...) *CreateIssuesConfig
func (c *Compiler) generateCreateIssuesJob(...) map[string]any
```

#### 2. Engine Architecture

Each AI engine follows a consistent pattern:

- **Separate files**: `copilot_engine.go`, `claude_engine.go`, `codex_engine.go`
- **Shared utilities**: `engine_helpers.go` contains common functionality
- **Clear interfaces**: All engines implement common methods

**Key Files**:
- `agentic_engine.go` - Base engine interface
- `<engine>_engine.go` - Engine-specific implementation
- `engine_helpers.go` - Shared helper functions
- `engine_helpers_test.go` - Common test utilities

#### 3. Compiler Architecture

The compiler is organized by responsibility:

- `compiler.go` - Main compilation orchestration
- `compiler_yaml.go` - YAML generation logic
- `compiler_jobs.go` - Job generation logic
- `compiler_test.go` - Comprehensive test coverage

This separation allows working on different aspects without conflicts.

#### 4. Expression Building

The expression system (`expressions.go`) demonstrates cohesive design:

- All expression-related logic in one file
- Tree-based structure for complex conditions
- Clean abstractions (ConditionNode interface)
- Comprehensive tests in `expressions_test.go`

### File Organization Best Practices

#### ✅ Good Patterns

1. **Focused files**: Each file has a clear, single responsibility
2. **Descriptive names**: File names clearly indicate their purpose
3. **Collocated tests**: Tests live next to implementation
4. **Reasonable size**: Most files under 500 lines

#### ❌ Anti-Patterns to Avoid

1. **God files**: Single file doing too many things
2. **Vague naming**: `utils.go`, `helpers.go` without context
3. **Mixed concerns**: Unrelated functionality in one file
4. **Massive tests**: All tests in one huge file

### When to Create New Files

Use this decision tree:

1. **New safe output type?** → `create_<entity>.go`
2. **New AI engine?** → `<engine>_engine.go`
3. **New domain feature?** → `<feature>.go`
4. **File over 800 lines?** → Consider splitting
5. **Independent functionality?** → Create new file

### Code Organization Guidelines

#### File Size Targets

- **Small** (50-200 lines): Simple utilities, helpers
- **Medium** (200-500 lines): Feature implementations
- **Large** (500-800 lines): Complex features
- **Very Large** (800+ lines): Core infrastructure only

#### Naming Conventions

- **Create operations**: `create_<entity>.go`
- **Engines**: `<engine>_engine.go`
- **Features**: `<feature>.go`
- **Helpers**: `<subsystem>_helpers.go`
- **Tests**: `<feature>_test.go`, `<feature>_integration_test.go`

### Package Structure

```text
pkg/workflow/
├── create_*.go              # GitHub entity creation
├── *_engine.go              # AI engine implementations
├── engine_helpers.go        # Shared engine utilities
├── compiler*.go             # Compilation logic
├── expressions.go           # Expression building
├── validation.go            # Schema validation
├── strings.go               # String utilities
└── *_test.go                # Tests alongside code
```

### Testing Architecture

#### Test Organization

- **Unit tests**: `feature_test.go` - Fast, focused tests
- **Integration tests**: `feature_integration_test.go` - Cross-component tests
- **Scenario tests**: `feature_scenario_test.go` - Specific use cases

#### Test Naming

Use descriptive names that explain what's being tested:
- ✅ `create_issue_assignees_test.go` - Clear purpose
- ✅ `engine_error_patterns_infinite_loop_test.go` - Specific scenario
- ❌ `test_utils.go` - Too vague

### Contributing to Architecture

When adding new features:

1. **Follow existing patterns** - Look for similar features first
2. **Keep files focused** - One responsibility per file
3. **Use descriptive names** - Future you will thank present you
4. **Write tests alongside** - Don't defer testing
5. **Document patterns** - Update this guide when introducing new patterns

For complete details, see [Code Organization Patterns](specs/code-organization.md).



### Prerequisites for Releases

Before creating a release, ensure you have:

- **Maintainer access** to the GitHub repository
- **Push permissions** to create tags
- **Write access** to GitHub releases
- **All tests passing** on the main branch

### Release Types

The project uses semantic versioning (semver):
- **Major** (v2.0.0): Breaking API changes, incompatible updates
- **Minor** (v1.1.0): New features, backward compatible
- **Patch** (v1.0.1): Bug fixes, backward compatible

### Official Release Process

Releases are **automatically handled by GitHub Actions** when you create a git tag. The process is:

#### 1. Prepare for Release

```bash
# Ensure you're on the main branch with latest changes
git checkout main
git pull origin main

# Run all tests to ensure stability
make test
make lint
make fmt-check

# Test build locally
make build-all
```

#### 2. Create and Push Release Tag

For patch releases (bug fixes), you can use the automated make target:

```bash
# Automated patch release - finds current version and increments patch number
make patch-release

# Automated patch release - finds current version and increments minor number
make minor-release
```

Or create the tag manually:

```bash
# Create a new tag following semantic versioning
# Replace x.y.z with the actual version number
git tag -a v1.0.0 -m "Release v1.0.0"

# Push the tag to trigger the release workflow
git push origin v1.0.0
```

#### 3. Automated Release Process

When you push a tag matching `v*.*.*`, GitHub Actions automatically:

1. **Runs tests** to ensure code quality
2. **Builds cross-platform binaries** using `gh-extension-precompile`
3. **Creates GitHub release** with:
   - Pre-compiled binaries for Linux (amd64, arm64)
   - Pre-compiled binaries for macOS (amd64, arm64) 
   - Pre-compiled binaries for Windows (amd64)
   - Automatic changelog generation

#### 4. Verify Release

After the GitHub Actions workflow completes:

```bash
# Check the release was created successfully
gh release list

# Remove any existing extension
gh extension remove gh-aw || true

# Test installation as a GitHub CLI extension
gh extension install githubnext/gh-aw@v1.0.0
gh aw --help
```

### Release Workflow Details

The release is orchestrated by `.github/workflows/release.yml` which:

- **Triggers on**: Git tags matching `v*.*.*` pattern or manual workflow dispatch
- **Runs on**: Ubuntu latest with Go version from `go.mod`
- **Permissions**: Contents (write), packages (write), ID token (write)
- **Artifacts**: Cross-platform binaries, Docker images, checksums

### Rollback Process

If a release has critical issues:

1. **Immediate**: Delete the problematic release from GitHub
   ```bash
   gh release delete v1.0.0 --yes
   git tag -d v1.0.0
   git push origin :refs/tags/v1.0.0
   ```

2. **Long-term**: Create a new release with fixes

### Current Release Infrastructure Status

The project has a complete automated release system in place:

- ✅ **GitHub Actions workflow** (`.github/workflows/release.yml`)
- ✅ **Cross-platform binary builds** via `gh-extension-precompile`
- ✅ **Semantic versioning** with git tags

The release system is **production-ready** and uses GitHub's official `gh-extension-precompile` action, which is the recommended approach for GitHub CLI extensions.

### Release Notes and Changelog

Release notes are automatically generated from:
- **Commit messages** between releases
- **Pull request titles** and descriptions
- **Conventional commit format** is recommended for better changelog generation

To improve changelog quality, use conventional commit messages:
```bash
git commit -m "feat: add new workflow command"
git commit -m "fix: resolve path handling on Windows"
git commit -m "docs: update installation instructions"
```

### Version Management

- **Version information** is automatically injected at build time
- **Current version** comes from git tags (`git describe --tags`)
- **No manual version files** need to be updated
- **Build metadata** includes commit hash and build date

