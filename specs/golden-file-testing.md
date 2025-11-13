# Golden File Testing Specification

## Overview

Golden file testing validates compiler output against expected YAML files, ensuring that workflow compilation produces consistent results and catches unintended changes.

**Status**: ✅ Implemented  
**Implementation**: `pkg/workflow/compiler_golden_test.go`, `pkg/workflow/testing_helpers.go`

## Architecture

### Directory Structure

```
pkg/workflow/testdata/
├── workflows/          # Source workflow markdown files
│   ├── issue_trigger.md
│   ├── pr_trigger.md
│   ├── scheduled.md
│   └── ...
└── golden/            # Expected compiled YAML output
    ├── issue_trigger.lock.yml
    ├── pr_trigger.lock.yml
    ├── scheduled.lock.yml
    └── ...
```

## Implementation

### Testing Helper

**File**: `pkg/workflow/testing_helpers.go`

```go
func CompareGoldenFile(t *testing.T, got []byte, goldenPath string)
```

- Compares generated output with golden files
- Supports `-update-golden` flag to regenerate golden files
- Provides clear error messages when output differs

### Test Suite

**File**: `pkg/workflow/compiler_golden_test.go`

- Table-driven tests covering 10 workflow patterns
- Automatic temp directory setup/teardown
- Import file handling for shared configuration tests

## Representative Workflow Examples

The golden file tests cover these workflow patterns:

1. **Issue trigger workflow** - Basic issue event handling with GitHub toolsets
2. **Pull request trigger workflow** - PR event processing with Claude engine
3. **Scheduled workflow** - Cron-based execution with safe outputs and web-search tools
4. **Command trigger workflow** - /mention-based activation
5. **Multi-job workflow** - Workflows with custom jobs and bash tools
6. **Workflow with MCP servers** - Custom MCP server integration
7. **Workflow with safe outputs** - Safe issue and comment creation
8. **Workflow with network permissions** - Custom network access control
9. **Workflow with imports** - Shared configuration imports
10. **Custom engine workflow** - Custom execution engine configuration

## Usage

### Running Tests

```bash
# Run tests and compare with golden files
go test ./pkg/workflow -run TestCompilerGoldenFiles

# Run all workflow tests including golden file tests
make test-unit
```

### Updating Golden Files

```bash
# Update golden files when intentional changes occur
go test ./pkg/workflow -run TestCompilerGoldenFiles -update-golden

# Or use the Makefile target
make update-golden
```

### Adding New Golden File Tests

1. Create a new workflow file in `pkg/workflow/testdata/workflows/`
2. Add a test case to `TestCompilerGoldenFiles` in `compiler_golden_test.go`
3. Run `make update-golden` to generate the golden file
4. Verify the golden file contains expected output
5. Run tests normally to ensure comparison works

## Benefits

1. **Regression Detection**: Automatically catches unintended changes to compiler output
2. **Easy Maintenance**: Simple flag to update golden files when changes are intentional
3. **Comprehensive Coverage**: 10 different workflow patterns covering common scenarios
4. **Clear Feedback**: Helpful error messages showing what changed and how to update
5. **Fast Execution**: Tests run in ~130ms total

## Test Execution

Tests execute quickly and provide immediate feedback:

```
=== RUN   TestCompilerGoldenFiles
    --- PASS: TestCompilerGoldenFiles/issue_trigger_workflow (0.02s)
    --- PASS: TestCompilerGoldenFiles/pull_request_trigger_workflow (0.01s)
    --- PASS: TestCompilerGoldenFiles/scheduled_workflow (0.01s)
    --- PASS: TestCompilerGoldenFiles/command_trigger_workflow (0.01s)
    --- PASS: TestCompilerGoldenFiles/multi-job_workflow (0.04s)
    --- PASS: TestCompilerGoldenFiles/workflow_with_MCP_servers (0.01s)
    --- PASS: TestCompilerGoldenFiles/workflow_with_safe_outputs (0.01s)
    --- PASS: TestCompilerGoldenFiles/workflow_with_network_permissions (0.01s)
    --- PASS: TestCompilerGoldenFiles/workflow_with_imports (0.01s)
    --- PASS: TestCompilerGoldenFiles/custom_engine_workflow (0.00s)
--- PASS: TestCompilerGoldenFiles (0.12s)
```

## Related Documentation

- [Code Organization Patterns](./code-organization.md)
- [Validation Architecture](./validation-architecture.md)
- Testing documentation: See `TESTING.md`

---

**Last Updated**: 2025-11-13
