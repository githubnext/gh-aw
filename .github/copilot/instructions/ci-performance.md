# CI/CD Performance Engineering Guide

This guide helps you optimize GitHub Actions CI/CD pipeline performance for gh-aw.

## Performance Targets

- **Total CI time**: < 10 minutes (current: 8-12 minutes) ✓
- **Test job**: < 3 minutes (current: 3-5 minutes)
- **Build job**: < 5 minutes (current: 5-7 minutes)
- **JS tests**: < 2 minutes (current: 2-3 minutes)
- **Lint job**: < 3 minutes (current: 3-4 minutes)

## Current CI Pipeline Analysis

### Pipeline Structure

```yaml
# .github/workflows/ci.yml
jobs:
  test:     # 3-5 minutes - Go tests + coverage
  build:    # 5-7 minutes - Matrix (ubuntu, macos) - parallel
  js:       # 2-3 minutes - JavaScript tests
  lint:     # 3-4 minutes - golangci-lint + format check
```

**Total**: ~8-12 minutes (jobs run in parallel)

### Job Breakdown

**Test Job** (3-5 min):
- Checkout: ~5s
- Setup Go: ~10s (with cache hit)
- Verify dependencies: ~5s
- Run tests + coverage: 2-4min
- Upload coverage: ~10s

**Build Job** (5-7 min per OS):
- Checkout: ~5s
- Setup Node + Go: ~15s (with cache)
- npm ci: ~30s
- Build: ~1.5s
- Recompile workflows: ~10-15s
- Check changes: ~1s

**JS Job** (2-3 min):
- Checkout: ~5s
- Setup Node: ~10s (with cache)
- npm ci: ~30s
- npm test: 1.5-2min

**Lint Job** (3-4 min):
- Checkout: ~5s
- Setup Go: ~10s (with cache)
- Install deps: ~10s
- golangci-lint: 2.5-3min
- Format check: ~5s

## Optimization Opportunities

### 1. Test Caching

**Problem**: Tests run with `-count=1` which disables Go test caching.

**Current**:
```makefile
test:
	go test -v -timeout=3m -tags 'integration' ./...
```

**Optimization**:
```yaml
# In CI, allow test caching for unchanged code
- name: Run tests with cache
  run: go test -v -timeout=3m -tags 'integration' ./...
  env:
    GOCACHE: ${{ github.workspace }}/.gocache
```

**Selective Cache Invalidation**:
```yaml
- name: Cache Go test results
  uses: actions/cache@v4
  with:
    path: |
      ~/.cache/go-build
      .gocache
    key: go-test-${{ runner.os }}-${{ hashFiles('**/*.go') }}
    restore-keys: |
      go-test-${{ runner.os }}-
```

**Impact**: 20-30% faster test runs on cache hit

### 2. Parallel Test Execution

**Problem**: Tests run sequentially by package.

**Optimization**:
```yaml
- name: Run tests in parallel
  run: go test -v -timeout=3m -p 4 -tags 'integration' ./...
```

**Impact**: Potential 30-40% reduction (depends on test distribution)

### 3. Workflow Compilation Optimization

**Problem**: `make recompile` rebuilds all workflows even when few changed.

**Detection**:
```yaml
- name: Detect changed workflows
  id: changed
  run: |
    CHANGED=$(git diff --name-only ${{ github.event.before }} ${{ github.sha }} -- '.github/workflows/*.md' || echo "")
    echo "workflows=$CHANGED" >> $GITHUB_OUTPUT
    
- name: Recompile changed workflows
  if: steps.changed.outputs.workflows != ''
  run: |
    for workflow in ${{ steps.changed.outputs.workflows }}; do
      ./gh-aw compile "$workflow"
    done
```

**Impact**: 80-90% faster when only few workflows changed

### 4. Dependency Caching Improvements

**Current**: Good caching via `actions/setup-go@v6` and `actions/setup-node@v6`.

**Enhanced Caching**:
```yaml
- name: Cache Go dependencies
  uses: actions/cache@v4
  with:
    path: |
      ~/go/pkg/mod
      ~/.cache/go-build
    key: go-deps-${{ runner.os }}-${{ hashFiles('**/go.sum') }}
    restore-keys: |
      go-deps-${{ runner.os }}-

- name: Cache npm dependencies
  uses: actions/cache@v4
  with:
    path: |
      ~/.npm
      actions/setup/js/node_modules
    key: npm-deps-${{ runner.os }}-${{ hashFiles('**/package-lock.json') }}
    restore-keys: |
      npm-deps-${{ runner.os }}-
```

**Impact**: Marginal improvement (already well-cached)

### 5. Lint Job Optimization

**Problem**: golangci-lint can be slow on first run.

**Current Approach**:
```yaml
- name: Install dev dependencies
  run: make deps-dev

- name: Run linter
  run: make lint
```

**Optimization Options**:
```makefile
# In Makefile, add selective linting for PRs
lint-changes:
	golangci-lint run --new-from-rev=origin/main
```

**Selective Linting** (for PRs):
```yaml
- name: Run golangci-lint on changed files
  if: github.event_name == 'pull_request'
  run: golangci-lint run --new-from-rev=origin/${{ github.base_ref }}
```

**Impact**: 50-70% faster on PRs with few changes

## Advanced Optimization Strategies

### 1. Matrix Strategy Optimization

**Current**: Build matrix runs on both ubuntu and macos (in parallel).

**Selective OS Testing**:
```yaml
strategy:
  matrix:
    os: 
      - ubuntu-latest
      - ${{ github.event_name == 'push' && github.ref == 'refs/heads/main' && 'macos-latest' || '' }}
    exclude:
      - os: ''
```

**Impact**: Faster PR checks (ubuntu only), full validation on main

### 2. Job Dependencies and Parallelization

**Current**: All jobs run in parallel (good).

**Optimization**: Keep parallel structure, but add fail-fast for long-running jobs:

```yaml
strategy:
  fail-fast: false  # Let all jobs complete for full info
```

### 3. Conditional Job Execution

**Skip CI for Docs-Only Changes**:
```yaml
on:
  push:
    paths-ignore:
      - 'docs/**'
      - '**.md'
      - 'README.md'
```

**Skip Tests for Specific Paths**:
```yaml
jobs:
  test:
    if: |
      !contains(github.event.head_commit.message, '[skip ci]') &&
      !contains(github.event.head_commit.message, '[ci skip]')
```

### 4. Concurrency Groups

**Current**: Good use of concurrency groups with cancel-in-progress.

```yaml
concurrency:
  group: ci-${{ github.workflow }}-${{ github.event_name }}-${{ github.ref }}-test
  cancel-in-progress: true  # Cancel old runs on new push
```

**Keep this pattern** - it prevents wasted CI time.

## Performance Measurement

### Measure CI Job Duration

```bash
# Using GitHub CLI
gh run list --workflow=ci.yml --limit 10 --json durationMs,conclusion

# Analyze trends
gh run list --workflow=ci.yml --limit 50 --json durationMs,conclusion | \
  jq '.[] | select(.conclusion=="success") | .durationMs' | \
  awk '{sum+=$1; count++} END {print "Average:", sum/count/1000/60, "minutes"}'
```

### Identify Slow Steps

```bash
# Download run logs
gh run view <run-id> --log

# Find slow steps
grep "took" run.log | sort -t: -k2 -rn | head -10
```

### Compare Before/After

```bash
# Baseline (10 runs)
gh run list --workflow=ci.yml --limit 10 --json durationMs | \
  jq '[.[].durationMs] | add/length/1000/60'

# After optimization
gh run list --workflow=ci.yml --limit 10 --json durationMs | \
  jq '[.[].durationMs] | add/length/1000/60'
```

## CI Performance Best Practices

### ✓ Good Patterns

```yaml
# Use caching effectively
- uses: actions/cache@v4
  with:
    key: ${{ runner.os }}-${{ hashFiles('**/*.lock') }}
    restore-keys: ${{ runner.os }}-

# Parallel jobs
jobs:
  test:    # Runs in parallel
  build:   # Runs in parallel
  lint:    # Runs in parallel

# Fail fast when appropriate
strategy:
  fail-fast: true  # For quick feedback

# Cancel old runs
concurrency:
  group: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true

# Conditional execution
- name: Expensive step
  if: github.event_name == 'push'
  run: expensive-command
```

### ✗ Anti-Patterns to Avoid

```yaml
# Don't run jobs sequentially unless required
jobs:
  build:
  test:
    needs: build  # BAD - adds latency

# Don't skip caching
- name: Install dependencies
  run: npm install  # BAD - should use cache

# Don't run unnecessary steps
- name: Run on all events
  run: expensive-command  # BAD - should be conditional

# Don't use inefficient checkout
- uses: actions/checkout@v5
  with:
    fetch-depth: 0  # BAD - slow, usually not needed
```

## CI Performance Checklist

Before deploying CI optimizations:

- [ ] Measure baseline job durations
- [ ] Identify slowest jobs/steps
- [ ] Apply targeted optimizations
- [ ] Test on feature branch first
- [ ] Measure improvement (should be >15%)
- [ ] Verify all tests still pass
- [ ] Check no jobs were accidentally skipped
- [ ] Monitor for 1 week after deployment

## Troubleshooting

### Problem: Lint job is slow

**Diagnosis**:
```bash
# Check golangci-lint config
cat .golangci.yml

# Run locally to compare
time golangci-lint run
```

**Solution**:
- Use `make deps-dev` to ensure golangci-lint is installed and cached
- Use `--new-from-rev` for PRs to lint only changed code
- Tune linter configuration in `.golangci.yml`
- Split into separate linters if needed

### Problem: Tests timing out

**Diagnosis**:
```bash
# Find slow tests
make test-perf

# Check for deadlocks/infinite loops
go test -v -timeout=30s -run=SlowTest ./pkg/...
```

**Solution**:
- Increase timeout: `-timeout=5m`
- Optimize slow tests
- Skip integration tests in quick checks

### Problem: Build matrix is slow

**Diagnosis**:
```bash
# Compare ubuntu vs macos times
gh run list --workflow=ci.yml --json jobs | \
  jq '.[] | .jobs[] | select(.name | contains("Build")) | {name, durationMs}'
```

**Solution**:
- Use ubuntu for PR checks
- Run macos only on main/release
- Consider using self-hosted runners

### Problem: Cache misses frequently

**Diagnosis**:
```bash
# Check cache hit rate in logs
gh run view <run-id> --log | grep -i "cache"
```

**Solution**:
- Review cache key patterns
- Check if dependencies change frequently
- Use restore-keys for fallback
- Consider increasing cache retention

## CI Performance Metrics to Track

```
Metric                     Target    Current    Status
---------------------------------------------------------
Total CI time             < 10min    8-12min    ✓
Test job                  < 3min     3-5min     ⚠
Build job (ubuntu)        < 5min     5-7min     ⚠  
Build job (macos)         < 7min     5-7min     ✓
JS test job              < 2min     2-3min     ⚠
Lint job                 < 3min     3-4min     ⚠
Cache hit rate           > 80%      ~70%       ⚠
```

## Resources

- GitHub Actions best practices: https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions
- Caching dependencies: https://docs.github.com/en/actions/using-workflows/caching-dependencies-to-speed-up-workflows
- Matrix strategies: https://docs.github.com/en/actions/using-jobs/using-a-matrix-for-your-jobs
- golangci-lint optimization: https://golangci-lint.run/usage/performance/
