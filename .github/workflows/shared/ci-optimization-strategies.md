---
# CI Optimization Analysis Strategies
# Reusable analysis patterns for CI optimization workflows
#
# Usage:
#   imports:
#     - shared/ci-optimization-strategies.md
#
# This import provides:
# - Test coverage analysis patterns
# - Performance bottleneck identification
# - Matrix strategy optimization techniques
---

# CI Optimization Analysis Strategies

Comprehensive strategies for analyzing CI workflows to identify optimization opportunities.

## Phase 1: CI Configuration Study

Read and understand the current CI workflow structure:

```bash
# Read the CI workflow configuration
cat .github/workflows/ci.yml

# Understand the job structure
# - lint (runs first)
# - test (depends on lint)
# - integration (depends on test, matrix strategy)
# - build (depends on lint)
# etc.
```

**Key aspects to analyze:**
- Job dependencies and parallelization opportunities
- Cache usage patterns (Go cache, Node cache)
- Matrix strategy effectiveness
- Timeout configurations
- Concurrency groups
- Artifact retention policies

## Phase 2: Test Coverage Analysis

### Critical: Ensure ALL Tests are Executed

**Step 1: Get complete list of all tests**
```bash
# List all test functions in the repository
go test -list='^Test' ./... 2>&1 | grep -E '^Test' > /tmp/all-tests.txt

# Count total tests
TOTAL_TESTS=$(wc -l < /tmp/all-tests.txt)
echo "Total tests found: $TOTAL_TESTS"
```

**Step 2: Analyze unit test coverage**
```bash
# Unit tests run all non-integration tests
# Verify the test job's command captures all non-integration tests
# Current: go test -v -parallel=8 -timeout=3m -tags '!integration' -run='^Test' ./...

# Get list of integration tests (tests with integration build tag)
grep -r "//go:build integration" --include="*_test.go" . | cut -d: -f1 | sort -u > /tmp/integration-test-files.txt

# Estimate number of integration tests
echo "Files with integration tests:"
wc -l < /tmp/integration-test-files.txt
```

**Step 3: Analyze integration test matrix coverage**
```bash
# The integration job has a matrix with specific patterns
# Each matrix entry targets specific packages and test patterns

# CRITICAL CHECK: Are there tests that don't match ANY pattern?

# Extract all integration test patterns from ci.yml
cat .github/workflows/ci.yml | grep -A 2 'pattern:' | grep 'pattern:' > /tmp/matrix-patterns.txt

# Check for catch-all groups
cat .github/workflows/ci.yml | grep -B 2 'pattern: ""' | grep 'name:' > /tmp/catchall-groups.txt
```

**Step 4: Identify coverage gaps**
```bash
# Check if each package with tests is covered by at least one matrix group
# Compare packages with tests vs. packages in CI matrix
# Identify any "orphaned" tests not executed by any job
```

**Required Action if Gaps Found:**
If any tests are not covered by the CI matrix, propose adding:
1. **Catch-all matrix groups** for packages with specific patterns but no catch-all
2. **New matrix entries** for packages not in the matrix at all

Example fix for missing catch-all (add to `.github/workflows/ci.yml`):
```yaml
# Add to the integration job's matrix.include section:
- name: "CLI Other"  # Catch-all for tests not matched by specific patterns
  packages: "./pkg/cli"
  pattern: ""  # Empty pattern runs all remaining tests
```

## Phase 3: Test Performance Optimization

### A. Test Splitting Analysis
- Review current test matrix configuration
- Analyze if test groups are balanced in execution time
- Suggest rebalancing to minimize longest-running group

### B. Test Parallelization Within Jobs
- Check if tests run sequentially when they could run in parallel
- Suggest using `go test -parallel=N` to increase parallelism
- Analyze if `-count=1` is necessary for all tests

### C. Test Selection Optimization
- Suggest path-based test filtering to skip irrelevant tests
- Recommend running only affected tests for non-main branch pushes

### D. Test Timeout Optimization
- Review current timeout settings
- Check if timeouts are too conservative or too tight
- Suggest adjusting per-job timeouts based on historical data

### E. Test Dependencies Analysis
- Examine test job dependencies
- Suggest removing unnecessary dependencies to enable more parallelism

### F. Selective Test Execution
- Suggest running expensive tests only on main branch or on-demand
- Recommend running security scans conditionally

### G. Matrix Strategy Optimization
- Analyze if all integration test matrix jobs are necessary
- Check if some matrix jobs could be combined or run conditionally
- Suggest reducing matrix size for PR builds vs. main branch builds

## Phase 4: Resource Optimization

### Job Parallelization
- Identify jobs that could run in parallel but currently don't
- Restructure dependencies to reduce critical path
- Example: Could some test jobs start earlier?

### Cache Optimization
- Analyze cache hit rates
- Suggest caching more aggressively (dependencies, build artifacts)
- Check if cache keys are properly scoped

### Resource Right-Sizing
- Check if timeouts are set appropriately
- Evaluate if jobs could run on faster runners
- Review concurrency groups

### Artifact Management
- Check if retention days are optimal
- Identify unnecessary artifacts
- Example: Coverage reports only need 7 days retention

### Dependency Installation
- Check for redundant dependency installations
- Suggest using dependency caching more effectively
- Example: Sharing `node_modules` between jobs

## Phase 5: Cost-Benefit Analysis

For each potential optimization:
- **Impact**: How much time/cost savings?
- **Effort**: How difficult to implement?
- **Risk**: Could it break the build or miss issues?
- **Priority**: High/Medium/Low

## Optimization Categories

1. **Job Parallelization** - Reduce critical path
2. **Cache Optimization** - Improve cache hit rates
3. **Test Suite Restructuring** - Balance test execution
4. **Resource Right-Sizing** - Optimize timeouts and runners
5. **Artifact Management** - Reduce unnecessary uploads
6. **Matrix Strategy** - Balance breadth vs. speed
7. **Conditional Execution** - Skip unnecessary jobs
8. **Dependency Installation** - Reduce redundant work

## Expected Metrics

Track these metrics before and after optimization:
- Total CI duration (wall clock time)
- Critical path duration
- Cache hit rates
- Test execution time
- Resource utilization
- Cost per CI run
