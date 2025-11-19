# Comprehensive Testing Framework for Go Implementation

This document describes the comprehensive testing framework added to ensure the Go implementation of gh-aw matches the bash version exactly and maintains high quality standards.

## Overview

The testing framework implements **Phase 6 (Quality Assurance)** of the Go reimplementation project, providing comprehensive validation that the Go implementation behaves identically to the bash version while maintaining code quality and reliability.

## Testing Structure

### 1. Unit Tests (`pkg/*/`)

### 2. Copilot CLI Integration Test (`.github/workflows/ci.yml`)

An integration test for the Copilot CLI with MCP (Model Context Protocol) servers is available in the CI workflow. This test:

- Installs GitHub Copilot CLI with the default version (0.0.358)
- Configures MCP servers for GitHub and Playwright
- Executes a verification prompt to test MCP server loading
- Validates that both MCP servers are accessible and functional
- Uploads execution logs as artifacts for inspection

**Enabling the Test:**

The test is conditionally enabled via a repository variable:

1. Go to repository Settings → Secrets and variables → Actions → Variables
2. Create a new variable: `COPILOT_CLI_INTEGRATION_ENABLED` with value `true`
3. Ensure the `COPILOT_GITHUB_TOKEN` secret is configured (required for authentication)

**Note:** This test requires `COPILOT_GITHUB_TOKEN` secret which may not be available when building the copilot agent itself. The test will be skipped when the token is unavailable.

**Running the Test Locally:**

```bash
# The test requires COPILOT_GITHUB_TOKEN to be set
export COPILOT_GITHUB_TOKEN="your-token"

# Install Copilot CLI
npm install -g @github/copilot@0.0.358

# Setup MCP configuration (see .github/workflows/ci.yml for config structure)
mkdir -p ~/.copilot
cat > ~/.copilot/mcp-config.json << 'EOF'
{
  "mcpServers": {
    "github": {
      "type": "local",
      "command": "npx",
      "args": ["@github/github-mcp-server@v0.21.0"],
      "tools": ["*"]
    },
    "playwright": {
      "type": "local",
      "command": "npx",
      "args": ["@playwright/mcp@0.0.47", "--allowed-hosts", "example.com"],
      "tools": ["*"]
    }
  }
}
EOF

# Run the Copilot CLI with MCP verification prompt
copilot --add-dir /tmp/gh-aw/ --log-level all --disable-builtin-mcps --prompt "Verify that GitHub and Playwright MCP servers are loaded"
```

### 2. Fuzz Tests (`pkg/*/_fuzz_test.go`)

Fuzz tests use Go's built-in fuzzing support to test functions with randomly generated inputs, helping discover edge cases and security vulnerabilities that traditional tests might miss.

**Running Fuzz Tests:**
```bash
# Run expression parser fuzz test for 10 seconds
go test -fuzz=FuzzExpressionParser -fuzztime=10s ./pkg/workflow/

# Run for extended duration (1 minute)
go test -fuzz=FuzzExpressionParser -fuzztime=1m ./pkg/workflow/

# Run seed corpus only (no fuzzing)
go test -run FuzzExpressionParser ./pkg/workflow/
```

**Available Fuzz Tests:**
- **FuzzExpressionParser** (`pkg/workflow/expression_parser_fuzz_test.go`): Tests GitHub expression validation against injection attacks
  - 59 seed cases covering allowed expressions, malicious injections, and edge cases
  - Validates security controls against secret injection, script tags, command injection
  - Ensures parser handles malformed input without panic

**Fuzz Test Results:**
- Seed corpus includes authorized and unauthorized expression patterns
- Fuzzer generates thousands of variations per second
- Typical coverage: 87+ test cases in baseline, discovers additional interesting cases during fuzzing
- All inputs should be handled without panic, unauthorized expressions properly rejected

**Continuous Integration:**
Fuzz tests can be run in CI with time limits:
```yaml
- name: Fuzz test expression parser
  run: go test -fuzz=FuzzExpressionParser -fuzztime=30s ./pkg/workflow/
```

### 3. Benchmarks (`pkg/*/_benchmark_test.go`)

Performance benchmarks measure the speed of critical operations. Run benchmarks to:
- Detect performance regressions
- Identify optimization opportunities
- Track performance trends over time

**Running Benchmarks:**
```bash
# Run all benchmarks with make (optimized for CI, runs in ~6 seconds)
make bench

# Run all benchmarks manually
go test -bench=. -benchtime=3x -run=^$ ./pkg/...

# Run benchmarks with more iterations for comparison
make bench-compare

# Run benchmarks for specific package
go test -bench=. -benchtime=3x -run=^$ ./pkg/workflow/

# Run specific benchmark
go test -bench=BenchmarkCompileWorkflow -benchtime=3x -run=^$ ./pkg/workflow/

# Run with custom iterations (default is 1 second per benchmark)
go test -bench=. -benchtime=100x -run=^$ ./pkg/workflow/

# Run with memory profiling
go test -bench=. -benchmem -benchtime=3x -run=^$ ./pkg/...

# Compare benchmark results over time
go test -bench=. -benchtime=3x -run=^$ ./pkg/... > bench_baseline.txt
# ... make changes ...
go test -bench=. -benchtime=3x -run=^$ ./pkg/... > bench_new.txt
benchstat bench_baseline.txt bench_new.txt
```

**Note**: Benchmarks use `-benchtime=3x` (3 iterations) for fast CI execution. For more accurate measurements, use `-benchtime=100x` or longer durations.

**Benchmark Coverage:**
- **Workflow Compilation**: Basic, with MCP, with imports, with validation, complex workflows
- **Frontmatter Parsing**: Simple, complex, minimal, with arrays, schema validation
- **Expression Validation**: Single expressions, complex expressions, full markdown validation, parsing
- **Log Processing**: Claude, Copilot, Codex log parsing, aggregation, JSON metrics extraction
- **MCP Configuration**: Playwright config, Docker args, expression extraction
- **Tool Processing**: Simple and complex tool configurations, safe outputs, network permissions

**Performance Baselines** (approximate, machine-dependent):
- Workflow compilation: ~100μs - 2ms depending on complexity
- Frontmatter parsing: ~10μs - 250μs depending on complexity
- Expression validation: ~700ns - 10μs per expression
- Log parsing: ~50μs - 1ms depending on log size
- Schema validation: ~35μs - 130μs depending on complexity

### 4. Test Validation Framework (`test_validation.go`)

Comprehensive validation system that ensures:

#### Unit Test Validation
- All package tests pass
- Test coverage information is available
- No test failures or build errors

#### Sample Workflow Validation
- At least 5 sample workflows are available
- All sample files are readable and valid
- Workflow structure meets expectations

#### Test Coverage Validation  
- Coverage reports are generated correctly
- All packages have test coverage
- Tests execute and pass consistently

#### CLI Behavior Validation
- Go binary builds successfully
- Basic commands execute without crashing
- Help system works correctly
- Command interface is stable

## Test Execution

### Running All Tests
```bash
# Run all unit tests
go test ./pkg/... -v

# Run comprehensive validation
go run test_validation.go
```

### Test Results Summary
- **Unit Tests**: ⚠️ Partial - Parser & Workflow packages pass, CLI package has known failures (see #48)
- **Sample Workflows**: ✅ 5 sample files validated
- **Test Coverage**: ✅ Coverage reporting functional
- **CLI Behavior**: ✅ Binary builds and executes correctly

## Testing Philosophy

### Current Implementation Status
The tests are designed to work with the current implementation state:
- **Completed functionality**: Fully tested with comprehensive coverage
- **Stub implementations**: Interface stability testing to ensure future compatibility
- **Missing functionality**: Framework prepared for when implementation is complete

### Future Expansion
As the Go implementation develops:
1. **Stub tests** will be enhanced with full behavioral validation
3. **Edge case tests** will be expanded based on real usage patterns

## Test Coverage Areas

### ✅ Fully Tested
- Markdown frontmatter parsing (100% coverage)
- YAML extraction and processing
- CLI interface structure and stability
- Basic workflow compilation interface
- Error handling for malformed inputs
- **Performance benchmarks** for critical operations (62+ benchmarks)

### 🔄 Interface Testing (Ready for Implementation)
- CLI command execution (stubs tested)
- Workflow compilation (interface validated)
- Management commands (add, remove, enable, disable)

### 📋 Ready for Enhancement
- Bash-Go output comparison (when compiler is complete)
- **Performance regression tracking** (baseline established)
- Cross-platform compatibility testing
- Real workflow execution testing

## Quality Assurance

This testing framework ensures:

1. **Regression Prevention**: Any changes that break existing functionality will be caught
2. **Interface Stability**: CLI and API interfaces remain consistent
3. **Behavioral Compatibility**: Go implementation will match bash behavior exactly
4. **Code Quality**: High test coverage and comprehensive validation
5. **Future Readiness**: Testing infrastructure scales with implementation progress

## Test Maintenance

The testing framework is designed to be:
- **Self-validating**: The validation script ensures all tests work correctly  
- **Comprehensive**: Covers all aspects of functionality and interface design
- **Maintainable**: Clear structure and documentation for future updates
- **Scalable**: Easy to add new tests as functionality is implemented

## Conclusion

This comprehensive testing framework provides a solid foundation for ensuring the Go implementation of gh-aw maintains compatibility with the bash version while providing high-quality, reliable functionality. The framework is immediately useful for current development and ready to scale as implementation progresses.