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
```

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
```
# .trivyignore
CVE-2023-XXXXX  # False positive: not exploitable in our usage
```

### CI/CD Integration

Security scans run automatically on:
- Daily scheduled scan (6:00 AM UTC)
- Manual workflow dispatch

Results are uploaded to the GitHub Security tab in SARIF format.

### Security Scanning Exclusions

This project uses gosec for security scanning with specific exclusions documented below. These exclusions are configured in `.golangci.yml` and have been reviewed for security impact.

#### Global Exclusions

The following gosec rules are globally excluded across the entire codebase:

##### G101: Hardcoded Credentials
- **CWE**: CWE-798 (Use of Hard-coded Credentials)
- **Rationale**: High false positive rate on variable names containing terms like `token`, `secret`, `password`, `key`, etc. The rule triggers on identifiers, not actual values.
- **Examples of False Positives**:
  - Variable names: `tokenURL`, `secretName`, `apiKeyHeader`
  - Constants: `DefaultTokenEnv`, `SecretPrefix`
  - Function parameters: `func getToken(tokenName string)`
- **Mitigation**: 
  - Actual secrets are stored in GitHub Secrets or environment variables
  - No credentials are committed to source code
  - Pre-commit hooks and code review catch actual credential leaks
- **Review Date**: 2025-12-25

##### G115: Integer Overflow Conversion
- **CWE**: CWE-190 (Integer Overflow or Wraparound)
- **Rationale**: Integer conversions are acceptable in most cases in this codebase. The code primarily deals with configuration values, file sizes, and counts that are within safe ranges.
- **Context**: 
  - Configuration values are bounded and validated
  - File operations use appropriate size types
  - Counter values are within safe integer ranges
- **Mitigation**:
  - Input validation on all external data
  - Unit tests cover edge cases including boundary values
  - Runtime panics are acceptable for truly invalid conversions
- **Review Date**: 2025-12-25

##### G602: Slice Bounds Check
- **CWE**: CWE-118 (Improper Access of Indexable Resource)
- **Rationale**: Go runtime provides automatic bounds checking. Out-of-bounds access causes a panic rather than undefined behavior or memory corruption.
- **Context**:
  - Go's built-in bounds checking is a language safety feature
  - Panics are recovered at appropriate boundaries
  - Known false positives with switch statement bounds checks
- **Mitigation**:
  - Comprehensive unit tests cover slice operations
  - Integration tests validate real-world usage patterns
  - Panics are handled gracefully at API boundaries
- **Review Date**: 2025-12-25

#### File-Specific Exclusions

The following files have specific gosec rule exclusions with documented rationale:

##### G204: Subprocess Execution with Variable Arguments
- **CWE**: CWE-78 (OS Command Injection)
- **Files**: 
  - `pkg/awmg/gateway.go` - MCP gateway server commands
  - `pkg/cli/actionlint.go` - Docker commands for actionlint
  - `pkg/parser/remote_fetch.go` - Git commands for remote workflow fetching
  - `pkg/cli/download_workflow.go` - Git operations for workflow downloads
  - `pkg/cli/mcp_inspect.go` - Exec commands for MCP inspector
  - `pkg/cli/mcp_inspect_mcp.go` - MCP server execution
  - `pkg/cli/poutine.go` - Docker commands for Poutine scanner
  - `pkg/cli/zizmor.go` - Docker commands for Zizmor scanner
  - `pkg/workflow/js_comments_test.go` - Node command in tests
  - `pkg/workflow/playwright_mcp_integration_test.go` - npx commands in integration tests
  - `pkg/cli/status_command_test.go` - Binary execution in tests
- **Rationale**: Commands are constructed from:
  - Validated workflow configurations (parsed and type-checked)
  - Controlled Docker image references from known registries
  - Git operations on validated repository references
  - Tool invocations with allowlisted arguments
- **Mitigation**:
  - Input validation before command construction
  - Allowlist checks for command arguments
  - Docker images from trusted registries only
  - Git URLs validated against GitHub patterns
  - Test files use controlled test data
- **Review Date**: 2025-12-25

##### G404: Insecure Random Number Generation
- **CWE**: CWE-338 (Use of Cryptographically Weak PRNG)
- **Files**:
  - `pkg/cli/add_command.go` - Random IDs for temporary resources
  - `pkg/cli/update_git.go` - Non-cryptographic random operations
- **Rationale**: `math/rand` is used for non-cryptographic purposes such as:
  - Generating temporary file names
  - Creating random test data
  - Non-security-sensitive unique identifiers
- **Mitigation**:
  - Cryptographic operations use `crypto/rand`
  - Security tokens and secrets use cryptographically secure sources
  - Random values are not used for security decisions
- **Review Date**: 2025-12-25

##### G306: Weak File Permissions
- **CWE**: CWE-276 (Incorrect Default Permissions)
- **Files**:
  - `_test.go` (all test files) - Test file creation with 0644
  - `pkg/cli/mcp_inspect.go` - Executable script with 0755
  - `pkg/cli/actions_build_command.go` - Shell scripts with 0755
- **Configuration**: `G204: "0644"`, `G306: "0644"` allowed
- **Rationale**:
  - 0644 permissions are appropriate for regular files (owner read/write, group/others read)
  - 0755 permissions are correct for executable scripts (owner read/write/execute, group/others read/execute)
  - Test files are in temporary directories and cleaned up
  - Production files are in user-controlled directories
- **Mitigation**:
  - File permissions match Unix conventions
  - Sensitive data files use restrictive permissions (0600)
  - Executable scripts correctly marked as executable
- **Review Date**: 2025-12-25

##### G305: File Traversal in Archive Extraction
- **CWE**: CWE-22 (Path Traversal)
- **Files**:
  - `pkg/cli/logs_download.go` - Workflow log archive extraction
- **Rationale**: GitHub Actions workflow logs are downloaded from GitHub API (trusted source)
- **Mitigation**:
  - Logs downloaded only from authenticated GitHub API
  - Extraction performed in controlled temporary directories
  - Paths validated before file operations
  - Archives from trusted sources only
- **Review Date**: 2025-12-25

##### G110: Potential Decompression Bomb
- **CWE**: CWE-400 (Uncontrolled Resource Consumption)
- **Files**:
  - `pkg/cli/logs_download.go` - Workflow log decompression
- **Rationale**: Workflow logs are from GitHub Actions (trusted source) with known size constraints
- **Mitigation**:
  - Logs have maximum size limits imposed by GitHub Actions
  - Decompression in temporary directories with disk space monitoring
  - User-initiated operation with visible progress
- **Review Date**: 2025-12-25

#### Suppression Guidelines

When you need to suppress gosec warnings in code, use `#nosec` annotations with proper justification:

**Required Format**:
```go
// #nosec G<rule-id> -- <brief justification>
<code that triggers the warning>
```

**Best Practices**:
1. **Always include the rule ID**: `G204`, `G404`, etc.
2. **Use `--` separator**: Clearly separates rule ID from justification
3. **Keep justifications brief**: Under 80 characters
4. **Be specific**: Explain why this particular instance is safe
5. **Consider alternatives**: Is there a way to avoid the suppression?

**Examples**:

```go
// #nosec G204 -- Command arguments validated via allowlist
cmd := exec.Command("docker", validatedArgs...)

// #nosec G404 -- Random ID for temporary file, not security-sensitive
tmpID := fmt.Sprintf("tmp-%d", rand.Intn(1000000))

// #nosec G306 -- Executable script requires 0755 permissions
os.WriteFile(scriptPath, content, 0755)
```

**When to Use Suppressions**:
- ✅ After confirming the code is actually safe
- ✅ When the security risk is mitigated by other controls
- ✅ For false positives that cannot be avoided
- ✅ In test files for controlled test scenarios

**When NOT to Use Suppressions**:
- ❌ To bypass legitimate security issues
- ❌ Without understanding the security implication
- ❌ Without documenting the justification
- ❌ As a shortcut instead of fixing the code

**Review Process**:
1. All `#nosec` annotations are reviewed during code review
2. Reviewers must verify the justification is valid
3. Security-sensitive suppressions require additional review
4. Suppressions should be rare - prefer secure alternatives

#### Compliance and Audit Trail

This documentation provides an audit trail for compliance requirements:

- **Security Audits**: Documents accepted risks and mitigation strategies
- **Compliance Standards**: Supports SOC2, ISO 27001, and similar frameworks
- **Change Management**: Review dates track when exclusions were last evaluated
- **Incident Response**: Provides context for security incident investigations

**Review Schedule**:
- Security exclusions reviewed quarterly or when:
  - New gosec rules are added
  - Major security incidents occur in the Go ecosystem
  - Significant codebase refactoring is performed
  - Compliance requirements change

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

```
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

