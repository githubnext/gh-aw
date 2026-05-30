# Build Performance Engineering Guide

This guide helps you measure and optimize build, test, and compilation performance for gh-aw.

## Quick Measurement Commands

```bash
# Measure full build time
time make build

# Measure test suite performance
make test-perf  # Shows slowest 10 tests

# Measure full development cycle
time make agent-finish

# Measure workflow compilation
time ./gh-aw compile --validate
```

## Performance Targets

- **Build time**: < 2s (current: ~1.5s) ✓
- **Unit tests**: < 20s (current: ~25s)
- **Full tests**: < 30s (current: ~30s) ✓
- **Workflow compilation**: < 300ms per workflow
- **agent-finish**: < 8s (current: 10-15s)

## Common Bottlenecks

### 1. Workflow Compilation (All-or-Nothing)

**Problem**: `make recompile` rebuilds all 37+ workflows even when only one changed.

**Measurement**:
```bash
# Baseline
time ./gh-aw compile --validate

# With verbose mode
time ./gh-aw compile --validate --verbose
```

**Optimization Strategy**:
- Implement incremental compilation (only rebuild changed .md files)
- Add compilation cache based on file hashes
- Use parallel compilation for multiple workflows

**Before/After Testing**:
```bash
# Before
time ./gh-aw compile
# Record total time

# After optimization
time ./gh-aw compile
# Compare against baseline

# Test correctness
make test
./gh-aw compile --validate
```

### 2. Test Suite Performance

**Problem**: Sequential test execution doesn't leverage parallelism.

**Measurement**:
```bash
# Identify slow tests
make test-perf

# Run specific package tests
go test -v -timeout=3m ./pkg/workflow

# With benchmarking
go test -bench=. -benchmem ./pkg/workflow
```

**Optimization Strategy**:
- Use `go test -p N` for parallel package testing
- Identify and optimize slowest tests (>1s)
- Add Go benchmarks for critical paths
- Use `t.Parallel()` for network-bound test subtests

**Success Story: Parallel Git Ls-Remote Test (2025-10)**:
```go
// BEFORE: Sequential network calls
for key, pin := range actionPins {
    t.Run(key, func(t *testing.T) {
        cmd := exec.Command("git", "ls-remote", repoURL, tag)
        // ... validation
    })
}
// Result: 3.83s (slowest test)

// AFTER: Parallel network calls
for key, pin := range actionPins {
    key := key   // Capture for parallel
    pin := pin   // Capture for parallel
    t.Run(key, func(t *testing.T) {
        t.Parallel() // Run concurrently
        cmd := exec.Command("git", "ls-remote", repoURL, tag)
        // ... validation
    })
}
// Result: ~1.1s (70% faster, 3.5x speedup)
```

**Key Insight**: Network-bound tests benefit significantly from parallelization since they're I/O-bound rather than CPU-bound. Always capture loop variables when using `t.Parallel()`.

**Anti-Pattern: Parent-Aggregates-Results Tests (2025-11)**:
```go
// ✗ NOT SUITABLE for t.Parallel() - parent aggregates results
func TestMultipleServers(t *testing.T) {
    setup := setupTest(t)
    defer setup.cleanup()  // Runs before parallel subtests!
    
    successCount := 0  // Shared state
    
    for _, server := range servers {
        t.Run(server, func(t *testing.T) {
            t.Parallel()  // ✗ BAD: Parent continues and cleans up
            successCount++  // ✗ BAD: Race condition
        })
    }
    
    // ✗ BAD: This runs before parallel subtests execute!
    if successCount == 0 {
        t.Error("No servers succeeded")
    }
}
```

**Why it fails:**
1. `t.Parallel()` pauses subtests until parent function returns
2. Parent's `defer cleanup()` executes before subtests resume
3. Cleanup deletes resources subtests need
4. Parent checks `successCount` before subtests update it

**Solution:** Either:
- Remove `t.Parallel()` for tests with parent aggregation
- Restructure to avoid parent aggregation pattern
- Use separate parent/child test functions

**Example Benchmark**:
```go
func BenchmarkCompileWorkflow(b *testing.B) {
    // Setup
    content := loadTestWorkflow()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := compiler.Compile(content)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

### 3. Schema Changes Requiring Full Rebuild

**Problem**: Schemas are embedded via `//go:embed`, requiring binary rebuild for any schema change.

**Measurement**:
```bash
# Measure rebuild time after schema change
time make build
```

**Optimization Strategy**:
- Add validation-only mode that doesn't require rebuild
- Consider external schema loading for development
- Use schema versioning to avoid unnecessary rebuilds

**Development Workflow**:
```bash
# 1. Make schema change
vim pkg/parser/schemas/frontmatter.json

# 2. Rebuild (required)
make build

# 3. Test with workflow
./gh-aw compile test-workflow.md

# 4. Validate
make test-unit
```

### 4. Dependency Management Overhead

**Problem**: `go mod tidy` runs unnecessarily often.

**Measurement**:
```bash
# Measure dependency operations
time go mod download
time go mod verify
time go mod tidy
```

**Optimization Strategy**:
- Only run `go mod tidy` when go.mod/go.sum changes
- Cache dependency operations in CI
- Use conditional execution in Makefile

**Optimized Makefile Example**:
```makefile
.PHONY: deps
deps:
	go mod download
	go mod verify
	@if [ -n "$$(git status --porcelain go.mod go.sum)" ]; then \
		echo "Running go mod tidy..."; \
		go mod tidy; \
	else \
		echo "Skipping go mod tidy (no changes)"; \
	fi
```

## Performance Profiling

### CPU Profiling

```bash
# Profile specific test
go test -cpuprofile=cpu.prof -run=TestCompileWorkflow ./pkg/workflow

# Analyze profile
go tool pprof cpu.prof
# Interactive commands:
#   top10      - Show top 10 functions
#   list Func  - Show source for function
#   web        - Open browser visualization
```

### Memory Profiling

```bash
# Profile memory allocation
go test -memprofile=mem.prof -run=TestCompileWorkflow ./pkg/workflow

# Analyze allocations
go tool pprof -alloc_space mem.prof
go tool pprof -alloc_objects mem.prof
```

### Execution Tracing

```bash
# Generate execution trace
go test -trace=trace.out -run=TestCompileWorkflow ./pkg/workflow

# Analyze trace
go tool trace trace.out
# Opens browser with:
#   - Goroutine analysis
#   - Network blocking
#   - Synchronization blocking
#   - Syscall blocking
```

## Efficient Rebuild Strategies

### Incremental Builds

```bash
# Only rebuild changed workflows
./gh-aw compile changed-workflow.md

# Check which workflows changed
git diff --name-only .github/workflows/*.md

# Rebuild only those
for file in $(git diff --name-only .github/workflows/*.md); do
    ./gh-aw compile "$file"
done
```

### Parallel Compilation

```bash
# Compile multiple workflows in parallel (manual)
ls .github/workflows/*.md | xargs -P 4 -I {} ./gh-aw compile {}
```

### Build Caching

```bash
# Use Go build cache effectively
export GOCACHE=$(go env GOCACHE)
echo "Build cache: $GOCACHE"

# Clear cache if needed
go clean -cache

# Check cache stats
du -sh $GOCACHE
```

## Performance Testing Checklist

Before committing build performance changes:

- [ ] Run `make test-perf` and verify no regressions
- [ ] Measure `time make build` (should be < 2s)
- [ ] Measure `time make test-unit` (target: < 20s)
- [ ] Measure `time make agent-finish` (target: < 8s)
- [ ] Verify correctness with `make test`
- [ ] Check no new linting errors: `make lint`
- [ ] Test on both Linux and macOS if possible

## Common Performance Patterns

### ✓ Good Patterns

```go
// Reuse compiled regexes
var compiledRegex = regexp.MustCompile(`pattern`)

// Use string builder for concatenation
var sb strings.Builder
sb.WriteString("part1")
sb.WriteString("part2")
result := sb.String()

// Preallocate slices when size is known
items := make([]Item, 0, expectedSize)

// Use sync.Pool for temporary objects
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}
```

### ✗ Anti-Patterns to Avoid

```go
// Don't compile regex in hot paths
for _, item := range items {
    re := regexp.MustCompile(`pattern`) // BAD
}

// Don't use string concatenation in loops
result := ""
for _, s := range strings {
    result += s // BAD - use strings.Builder
}

// Don't defer in tight loops
for i := 0; i < 1000000; i++ {
    defer cleanup() // BAD - defer adds overhead
}
```

## Troubleshooting Slow Builds

### Problem: Build times suddenly increased

**Diagnosis**:
```bash
# Check build cache
go clean -cache
time make build  # Measure clean build

# Check for large binaries
ls -lh ./gh-aw

# Profile build
go build -x ./cmd/gh-aw  # Show all commands
```

**Solution**:
- Clear build cache and rebuild
- Check for accidental large dependencies
- Review recent code changes for imports

### Problem: Tests are slow

**Diagnosis**:
```bash
# Find slowest tests
make test-perf

# Run specific slow test with profiling
go test -v -cpuprofile=cpu.prof -run=SlowTest ./pkg/...
go tool pprof cpu.prof
```

**Solution**:
- Optimize identified bottlenecks
- Consider parallel test execution
- Review test setup/teardown overhead

### Problem: Compilation is slow

**Diagnosis**:
```bash
# Measure per-workflow compilation
time ./gh-aw compile workflow.md --verbose

# Check workflow complexity
wc -l .github/workflows/*.md | sort -n
```

**Solution**:
- Simplify complex workflows
- Implement compilation caching
- Use parallel compilation

## Performance Metrics to Track

Create a performance log for each optimization:

```
Optimization: Incremental workflow compilation
Date: 2025-10-24
Baseline: 
  - make recompile: 12.5s (37 workflows)
  - Per workflow: ~338ms average
After:
  - make recompile (1 change): 1.2s
  - Per workflow: ~340ms (unchanged)
Improvement: 90% faster for single workflow changes
Test Results: All tests pass ✓
```

## Resources

- Go Performance: https://go.dev/doc/effective_go#performance
- Go Profiling: https://go.dev/blog/pprof
- Benchmarking: https://pkg.go.dev/testing#hdr-Benchmarks
- Build Optimization: https://dave.cheney.net/high-performance-go-workshop
