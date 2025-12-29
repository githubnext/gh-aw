# Contributing to GitHub Agentic Workflows

Thank you for your interest in contributing to GitHub Agentic Workflows! We welcome contributions from the community and are excited to work with you.

**⚠️ IMPORTANT: This project requires agentic development using GitHub Copilot Agent. No local development environment is needed or expected.**

## 🤖 Agentic Development Workflow

GitHub Agentic Workflows is developed **exclusively through GitHub Copilot Agent**. This means:

- ✅ **All development happens in pull requests** created by GitHub Copilot Agent
- ✅ **No local setup required** - agents handle building, testing, and validation
- ✅ **Automated quality assurance** - CI runs all checks on agent-created PRs
- ❌ **Local development is not supported** - all work is done through the agent

### Why Agentic Development?

This project practices what it preaches: agentic workflows are used to build agentic workflows. Benefits include:

- **Consistency**: All changes go through the same automated quality gates
- **Accessibility**: No need to set up local development environments
- **Best practices**: Agents follow established patterns and guidelines automatically
- **Dogfooding**: We use our own tools to build our tools

## 🚀 Quick Start for Contributors

### Step 1: Fork the Repository

Fork <https://github.com/githubnext/gh-aw/> to your GitHub account

### Step 2: Open an Issue or Discussion

- Describe what you want to contribute
- Explain the use case and expected behavior
- Provide examples if applicable
- Tag with appropriate labels (see [Label Guidelines](specs/labels.md))

### Step 3: Create a Pull Request with GitHub Copilot Agent

Use GitHub Copilot Agent to implement your contribution:

1. **Start from the issue**: Reference the issue number in your PR description
2. **Provide clear instructions**: Tell the agent what changes you want
3. **Let the agent work**: The agent will read guidelines, make changes, run tests
4. **Review and iterate**: The agent will respond to feedback and update the PR

**Example PR description:**

```markdown
Fix #123 - Add support for custom MCP server timeout configuration

@github-copilot agent, please:
- Add a `timeout` field to MCP server configuration schema
- Update validation to accept timeout values between 5-300 seconds
- Add tests for timeout validation
- Update documentation with timeout examples
- Follow error message style guide for validation messages
```

### Step 4: Agent Handles Everything

The GitHub Copilot Agent will:

- Read relevant documentation and specifications
- Make code changes following established patterns
- Run `make agent-finish` to validate changes
- Format code, run linters, execute tests
- Recompile workflows to ensure compatibility
- Respond to review feedback and make adjustments

### No Local Setup Needed

You don't need to install Go, Node.js, or any dependencies. The agent runs in GitHub's infrastructure with all tools pre-configured.

## 📝 How to Contribute via GitHub Copilot Agent

All contributions are made through GitHub Copilot Agent in pull requests. The agent has access to comprehensive documentation and follows established patterns automatically.

### What the Agent Handles

The GitHub Copilot Agent automatically:

- **Reads specifications** from `specs/`, `skills/`, and `.github/instructions/`
- **Follows code organization patterns** (see [specs/code-organization.md](specs/code-organization.md))
- **Implements validation** following the architecture in [specs/validation-architecture.md](specs/validation-architecture.md)
- **Uses console formatting** from `pkg/console` for CLI output
- **Writes error messages** following the [Error Message Style Guide](.github/instructions/error-messages.instructions.md)
- **Runs all quality checks**: `make agent-finish` (build, test, recompile, format, lint)
- **Updates documentation** for new features
- **Creates tests** for new functionality

### Reporting Issues

Use the GitHub issue tracker to report bugs or request features:

- Include detailed steps to reproduce issues
- Explain the use case for feature requests
- Provide examples if applicable
- Follow [Label Guidelines](specs/labels.md)
- The agent will read the issue and implement fixes in a PR

### Code Quality Standards

GitHub Copilot Agent automatically enforces:

#### Error Messages

All validation errors follow the template: **[what's wrong]. [what's expected]. [example]**

```go
// Agent produces error messages like this:
return fmt.Errorf("invalid time delta format: +%s. Expected format like +25h, +3d, +1w, +1mo. Example: +3d", deltaStr)
```

The agent runs `make lint-errors` to verify error message quality.

#### Console Output

The agent uses styled console functions from `pkg/console`:

```go
import "github.com/githubnext/gh-aw/pkg/console"

fmt.Println(console.FormatSuccessMessage("Operation completed"))
fmt.Println(console.FormatInfoMessage("Processing workflow..."))
fmt.Fprintln(os.Stderr, console.FormatErrorMessage(err.Error()))
```

#### File Organization

The agent follows these principles:

- **Prefer many small files** over large monolithic files
- **Group by functionality**, not by type
- **Use descriptive names** that clearly indicate purpose
- **Follow established patterns** from the codebase

**Key Patterns the Agent Uses**:

1. **Create Functions Pattern** - One file per GitHub entity creation
   - Examples: `create_issue.go`, `create_pull_request.go`, `create_discussion.go`

2. **Engine Separation Pattern** - Each engine has its own file
   - Examples: `copilot_engine.go`, `claude_engine.go`, `codex_engine.go`
   - Shared helpers in `engine_helpers.go`

3. **Focused Utilities Pattern** - Self-contained feature files
   - Examples: `expressions.go`, `strings.go`, `artifacts.go`

See [Code Organization Patterns](specs/code-organization.md) for details.

#### Validation Patterns

The agent places validation logic appropriately:

**Centralized validation** (`pkg/workflow/validation.go`):

- Cross-cutting concerns
- Core workflow integrity
- GitHub Actions compatibility

**Domain-specific validation** (dedicated files):

- `strict_mode_validation.go` - Security enforcement
- `pip_validation.go` - Python packages
- `npm_validation.go` - NPM packages
- `docker_validation.go` - Docker images
- `expression_safety.go` - Expression security

See [Validation Architecture](specs/validation-architecture.md) for the complete decision tree.

#### CLI Breaking Changes

The agent evaluates whether changes are breaking:

- **Breaking**: Removing/renaming commands or flags, changing JSON output structure, altering defaults
- **Non-breaking**: Adding new commands/flags, adding output fields, bug fixes

For breaking changes, the agent:

- Uses `major` changeset type
- Provides migration guidance
- Documents in CHANGELOG.md

See [Breaking CLI Rules](specs/breaking-cli-rules.md) for details.

## 🔄 Pull Request Process via GitHub Copilot Agent

All pull requests are created and managed by GitHub Copilot Agent:

1. **Issue or discussion first:**
   - Open an issue describing what needs to be done
   - Provide clear context and examples
   - Tag appropriately using [Label Guidelines](specs/labels.md)

2. **Agent creates the PR:**
   - Mention `@github-copilot agent` with instructions
   - Agent reads specifications and guidelines
   - Agent makes changes following established patterns
   - Agent runs `make agent-finish` automatically

3. **Automated quality checks:**
   - CI runs on agent-created PRs
   - All checks must pass (build, test, lint, recompile)
   - Agent responds to CI failures and fixes them

4. **Review and iterate:**
   - Maintainers review the PR
   - Provide feedback as comments
   - Agent responds to feedback and makes adjustments
   - Once approved, PR is merged

### What Gets Validated

Every agent-created PR automatically runs:

- `make build` - Ensures Go code compiles
- `make test` - Runs all unit and integration tests
- `make lint` - Checks code quality and style
- `make recompile` - Recompiles all workflows to ensure compatibility
- `make fmt` - Formats Go code
- `make lint-errors` - Validates error message quality

## 🏗️ Project Structure (For Agent Reference)

The agent understands this structure:

```text
/
├── cmd/gh-aw/           # Main CLI application
├── pkg/                 # Core Go packages
│   ├── cli/             # CLI command implementations
│   ├── console/         # Console formatting utilities
│   ├── parser/          # Markdown frontmatter parsing
│   └── workflow/        # Workflow compilation and processing
├── specs/               # Technical specifications the agent reads
├── skills/              # Specialized knowledge for agents
├── .github/             # Instructions and sample workflows
│   ├── instructions/    # Agent instructions
│   └── workflows/       # Sample workflows and CI
└── Makefile             # Build automation (agent uses this)
```

## 📋 Dependency License Policy

This project uses an MIT license and only accepts dependencies with compatible licenses.

### Allowed Licenses

The following open-source licenses are compatible with our MIT license:

- **MIT** - Most permissive, allows reuse with minimal restrictions
- **Apache-2.0** - Permissive license with patent grant
- **BSD-2-Clause, BSD-3-Clause** - Simple permissive licenses
- **ISC** - Simplified permissive license similar to MIT

### Disallowed Licenses

The following licenses are **not allowed** as they conflict with our MIT license or impose unacceptable restrictions:

- **GPL, LGPL, AGPL** - Copyleft licenses that would force us to release under GPL
- **SSPL** - Server Side Public License with restrictive requirements
- **Proprietary/Commercial** - Closed-source licenses requiring payment or special terms

### Before Adding a Dependency

GitHub Copilot Agent automatically checks licenses when adding dependencies. However, if you're evaluating a dependency:

1. **Check its license**: Run `make license-check` after adding the dependency
2. **Review the report**: Run `make license-report` to generate a CSV of all licenses
3. **If unsure**: Ask in your PR - maintainers will help evaluate edge cases

### License Checking

The project includes automated license compliance checking:

- **CI Workflow**: `.github/workflows/license-check.yml` runs on every PR that changes `go.mod`
- **Local Check**: Run `make license-check` to verify all dependencies (installs `go-licenses` on-demand)
- **License Report**: Run `make license-report` to see detailed license information

All dependencies are automatically scanned using Google's `go-licenses` tool in CI, which classifies licenses by type and identifies potential compliance issues. Note that `go-licenses` is not actively maintained, so we install it on-demand rather than as a regular build dependency.

## 🧪 Testing Guidelines

GitHub Agentic Workflows has comprehensive testing practices (699 test files, 1,061+ table-driven tests). Understanding these patterns helps maintain code quality and consistency.

### Test Organization

Tests are co-located with implementation files:

- **Unit tests**: `feature.go` + `feature_test.go`
- **Integration tests**: `feature_integration_test.go` (marked with `//go:build integration`)
- **Security tests**: `feature_security_regression_test.go`
- **Fuzz tests**: `feature_fuzz_test.go`

### Assert vs Require

Use **testify** assertions appropriately:

- **`require.*`** - For critical setup steps that make the test invalid if they fail
  - Stops test execution immediately on failure
  - Use for: creating test files, parsing input, setting up test data
  
- **`assert.*`** - For actual test validations
  - Allows test to continue checking other conditions
  - Use for: verifying behavior, checking output values, testing multiple conditions

**Example from the codebase:**

```go
func TestSafeOutputsAppConfiguration(t *testing.T) {
    compiler := NewCompiler(false, "", "1.0.0")
    
    // Create test file - use require (setup step)
    tmpDir := t.TempDir()
    testFile := filepath.Join(tmpDir, "test.md")
    err := os.WriteFile(testFile, []byte(markdown), 0644)
    require.NoError(t, err, "Failed to write test file")
    
    // Parse file - use require (critical for test to continue)
    workflowData, err := compiler.ParseWorkflowFile(testFile)
    require.NoError(t, err, "Failed to parse markdown content")
    require.NotNil(t, workflowData.SafeOutputs, "SafeOutputs should not be nil")
    
    // Verify behavior - use assert (actual test validations)
    assert.Equal(t, "${{ vars.APP_ID }}", workflowData.SafeOutputs.App.AppID)
    assert.Equal(t, "${{ secrets.APP_PRIVATE_KEY }}", workflowData.SafeOutputs.App.PrivateKey)
    assert.Equal(t, []string{"repo1", "repo2"}, workflowData.SafeOutputs.App.Repositories)
}
```

### Table-Driven Tests

Use table-driven tests with `t.Run()` for testing multiple scenarios:

```go
func TestSortStrings(t *testing.T) {
    tests := []struct {
        name     string
        input    []string
        expected []string
    }{
        {
            name:     "already sorted",
            input:    []string{"a", "b", "c"},
            expected: []string{"a", "b", "c"},
        },
        {
            name:     "reverse order",
            input:    []string{"c", "b", "a"},
            expected: []string{"a", "b", "c"},
        },
        {
            name:     "empty slice",
            input:    []string{},
            expected: []string{},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := make([]string, len(tt.input))
            copy(result, tt.input)
            
            SortStrings(result)
            
            if len(result) != len(tt.expected) {
                t.Errorf("length = %d, want %d", len(result), len(tt.expected))
                return
            }
            
            for i := range result {
                if result[i] != tt.expected[i] {
                    t.Errorf("at index %d = %q, want %q", i, result[i], tt.expected[i])
                }
            }
        })
    }
}
```

**Key principles:**
- Use descriptive test case names (e.g., "already sorted", "empty slice", "invalid input")
- Structure: Define test cases → Loop with `t.Run()` → Test logic
- Each sub-test runs independently (supports parallel execution with `t.Parallel()`)

### Writing Good Tests

**Use specific assertions:**

```go
// ✅ GOOD - Specific assertions with context
assert.NotEmpty(t, result, "Result should not be empty")
assert.Contains(t, output, "expected text", "Output should contain expected text")
assert.Error(t, err, "Should return error for invalid input")
assert.NoError(t, err, "Failed to parse valid input")

// ❌ BAD - Generic checks without context
if result == "" {
    t.Error("empty")
}
```

**Always include helpful assertion messages:**
- Explain what failed and why it matters
- Include relevant context (input values, expected behavior)
- Make failures immediately understandable

**Test structure (Arrange-Act-Assert):**

```go
func TestFeature(t *testing.T) {
    // Arrange - Set up test data
    input := "test input"
    expected := "expected output"
    
    // Act - Execute the code being tested
    result := ProcessInput(input)
    
    // Assert - Verify the results
    assert.Equal(t, expected, result, "ProcessInput should transform input correctly")
}
```

### Why No Mocks or Test Suites?

This project **intentionally avoids** mocking frameworks and test suites:

**No mocks because:**
- **Simplicity**: Tests use real component interactions
- **Reliability**: Tests verify actual behavior, not mock behavior
- **Maintainability**: No mock setup/teardown boilerplate
- **Confidence**: Tests catch real integration issues

**No test suites (testify/suite) because:**
- **Parallel execution**: Standard Go tests run in parallel efficiently
- **Simplicity**: No suite lifecycle methods to understand
- **Explicitness**: Setup is visible in each test
- **Compatibility**: Works seamlessly with `go test` tooling

This approach keeps tests simple, fast, and maintainable. Tests verify real component interactions rather than mocked behavior.

### Running Tests

```bash
# Fast unit tests (recommended during development)
make test-unit       # ~25s - Unit tests only

# Full test suite
make test            # ~30s - All tests including integration

# Specific tests
go test -v ./pkg/workflow/...                    # Test specific package
go test -run TestSafeOutputs ./pkg/workflow/...  # Run specific test

# Security regression tests
make test-security   # Run security-focused tests

# With coverage
make test-coverage   # Generate coverage report

# Benchmarks
make bench          # Run performance benchmarks

# Fuzz testing
make fuzz           # Run fuzz tests for 30 seconds

# Linting (includes test quality checks)
make lint           # Runs golangci-lint with testifylint rules

# Complete validation (before committing)
make agent-finish   # Runs build, test, recompile, fmt, lint
```

**Note**: The project uses testifylint (via golangci-lint) to enforce consistent test assertion usage. Common rules enforced:
- Prefer specific assertions (`NotEmpty`, `NotNil`) over generic ones
- Use `require` for setup, `assert` for validations
- Include helpful assertion messages

### Additional Resources

- **[testify documentation](https://github.com/stretchr/testify)** - Assertion library reference
- **[specs/testing.md](specs/testing.md)** - Comprehensive testing framework documentation
- **[Go testing package](https://pkg.go.dev/testing)** - Official Go testing documentation
- **[Table-driven tests in Go](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)** - Best practices

## 🤝 Community

- Join the `#continuous-ai` channel in the [GitHub Next Discord](https://gh.io/next-discord)
- Participate in discussions on GitHub issues
- Collaborate through GitHub Copilot Agent PRs

## 📜 Code of Conduct

This project follows the GitHub Community Guidelines. Please be respectful and inclusive in all interactions.

## ❓ Getting Help

- **For bugs or features**: Open a GitHub issue and work with the agent
- **For questions**: Ask in issues, discussions, or Discord
- **For examples**: Look at existing agent-created PRs

## 🎯 Why No Local Development?

This project is built using agentic workflows to demonstrate their capabilities:

- **Dogfooding**: We use our own tools to build our tools
- **Accessibility**: No need for complex local setup
- **Consistency**: All changes go through the same automated process
- **Best practices**: Agents follow guidelines automatically
- **Focus on outcomes**: Describe what you want, not how to build it

The [Development Guide](DEVGUIDE.md) exists as reference for the agent, not for local setup.

Thank you for contributing to GitHub Agentic Workflows! 🤖🎉
