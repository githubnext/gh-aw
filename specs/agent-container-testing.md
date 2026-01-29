# Agent Container Testing

This document describes the testing strategy and implementation for validating agent container environment parity with GitHub Actions runners.

## Overview

The agent container must have parity with the GitHub Actions runner environment to ensure workflows execute consistently. This includes utilities, runtime tools, environment variables, and shared libraries.

## Testing Strategy

### 1. Integration Tests (`pkg/workflow/agent_parity_test.go`)

Integration tests validate that workflows can be compiled to test various aspects of the agent container environment. These tests do NOT execute workflows but verify the compilation process includes the necessary test specifications.

**Test Categories:**

- **Utilities**: Tests that workflow compilation includes checks for essential utilities (jq, curl, git, wget, tar, gzip, unzip, sed, awk, grep, find, xargs)
- **Runtimes**: Tests that workflow compilation includes runtime availability checks (node, python3, go, ruby)
- **Environment Variables**: Tests that workflow compilation includes environment variable checks (JAVA_HOME, ANDROID_HOME, GOROOT, PATH, HOME)
- **Shared Libraries**: Tests that workflow compilation includes ldd checks for shared library dependencies
- **Comprehensive**: Tests that workflow compilation includes all test categories in a single workflow

**Running Integration Tests:**

```bash
# Run all integration tests
make test

# Run only agent parity integration tests
go test -v -tags integration -run TestAgentRunnerParity ./pkg/workflow
```

### 2. Smoke Test Workflow (`.github/workflows/agent-runner-parity.md`)

The smoke test workflow is an actual agentic workflow that runs in the agent container and validates environment parity in production.

**Workflow Features:**

- Runs on a schedule (every 6 hours)
- Can be triggered manually via workflow_dispatch
- Can be triggered by adding "test-parity" label to PRs
- Posts results as comments on PRs or issues
- Provides concise pass/fail summary

**Test Execution:**

The workflow uses the Copilot engine with bash tools to:
1. Check utility accessibility using `which`
2. Verify runtime availability using version commands
3. Validate environment variables using `printenv` or `echo`
4. Test shared library linking using `ldd`

**Running Smoke Test:**

```bash
# Manually trigger the workflow
gh workflow run agent-runner-parity.md

# Add label to PR to trigger test
gh pr edit <PR_NUMBER> --add-label test-parity
```

### 3. Makefile Integration

A dedicated Make target runs the agent parity tests:

```bash
make test-parity
```

This target:
- Runs the integration tests with the `integration` build tag
- Filters to only `TestAgentRunnerParity*` tests
- Provides clear output for CI/CD integration

## Test Coverage

### Utilities (12 tested)

| Utility | Purpose | Verified By |
|---------|---------|-------------|
| jq | JSON processing | `which jq` |
| curl | HTTP client | `which curl` |
| git | Version control | `which git` |
| wget | File downloads | `which wget` |
| tar | Archive utility | `which tar` |
| gzip | Compression | `which gzip` |
| unzip | Archive extraction | `which unzip` |
| sed | Stream editing | `which sed` |
| awk | Pattern processing | `which awk` |
| grep | Text search | `which grep` |
| find | File finding | `which find` |
| xargs | Argument building | `which xargs` |

### Runtimes (4 tested)

| Runtime | Purpose | Verified By |
|---------|---------|-------------|
| Node.js | JavaScript runtime | `node --version` |
| Python | Python interpreter | `python3 --version` |
| Go | Go compiler | `go version` |
| Ruby | Ruby interpreter | `ruby --version` |

### Environment Variables (6 tested)

| Variable | Purpose | Verified By |
|----------|---------|-------------|
| JAVA_HOME | Java installation | `printenv JAVA_HOME` |
| ANDROID_HOME | Android SDK | `printenv ANDROID_HOME` |
| GOROOT | Go installation | `printenv GOROOT` |
| PATH | Binary search path | `printenv PATH` |
| HOME | User home directory | `printenv HOME` |
| USER | Current user | `printenv USER` |

### Shared Libraries (4 binaries tested)

| Binary | Purpose | Verified By |
|--------|---------|-------------|
| /usr/bin/python3 | Python interpreter | `ldd /usr/bin/python3` |
| /usr/bin/node | Node.js runtime | `ldd /usr/bin/node` |
| /usr/bin/git | Version control | `ldd /usr/bin/git` |
| /usr/bin/curl | HTTP client | `ldd /usr/bin/curl` |

## Test Execution Flow

### Integration Tests Flow

```
1. Create temporary test directory
2. Generate workflow markdown with test specifications
3. Compile workflow using NewCompiler
4. Verify lock file generation
5. Assert lock file contains expected test content
6. Clean up temporary files
```

### Smoke Test Flow

```
1. Workflow triggered (schedule/manual/PR label)
2. Agent container starts with Copilot engine
3. Bash tool executes verification commands
4. Results collected and summarized
5. Summary posted as comment (if PR/issue context)
6. Workflow succeeds/fails based on test results
```

## Extending Tests

### Adding New Utilities

1. Add utility to `TestAgentRunnerParity_Utilities` test data
2. Update smoke test workflow utilities list
3. Update this documentation's coverage table

### Adding New Runtimes

1. Add runtime to `TestAgentRunnerParity_Runtimes` test data
2. Update smoke test workflow runtimes list
3. Update this documentation's coverage table

### Adding New Environment Variables

1. Add variable to `TestAgentRunnerParity_EnvironmentVariables` test data
2. Update smoke test workflow environment variables list
3. Update this documentation's coverage table

### Adding New Binaries for Library Checks

1. Add binary to `TestAgentRunnerParity_SharedLibraries` test data
2. Update smoke test workflow shared libraries list
3. Update this documentation's coverage table

## Troubleshooting

### Test Compilation Failures

**Symptom**: Integration tests fail with "Failed to compile workflow"

**Solutions**:
- Verify workflow frontmatter YAML is valid
- Check that required permissions are specified
- Ensure engine and tools are properly configured

### Smoke Test Failures

**Symptom**: Smoke test workflow reports failures

**Solutions**:
- Check agent container has required utilities installed
- Verify environment variables are set correctly
- Ensure shared libraries are available in the container
- Review workflow logs for specific error messages

### Missing Utilities

**Symptom**: `which <utility>` returns empty or non-zero exit code

**Solutions**:
- Install missing utility in agent container image
- Update agent container PATH to include utility location
- Verify utility is mounted from runner to container

### Shared Library Not Found

**Symptom**: `ldd` reports "not found" for shared libraries

**Solutions**:
- Install missing shared library package
- Mount shared library directories from runner
- Update LD_LIBRARY_PATH in agent container

## CI/CD Integration

The agent parity tests are integrated into the CI/CD pipeline:

```yaml
# .github/workflows/ci.yml (example)
- name: Run Agent Parity Tests
  run: make test-parity
```

Tests run on:
- Every pull request
- Scheduled runs (smoke test)
- Manual workflow dispatch

## Related Documentation

- [Testing Guidelines](./testing.md) - General testing patterns and conventions
- [Sandbox Configuration](../pkg/workflow/sandbox.go) - Agent container configuration
- [Runtime Setup](../pkg/workflow/runtime_setup_test.go) - Runtime installation tests

## Maintenance

These tests should be updated when:
- New utilities are added to GitHub Actions runners
- Runtime versions are updated
- Environment variables are added or changed
- Agent container implementation changes

Regular reviews (quarterly recommended) ensure tests stay current with GitHub Actions runner updates.
