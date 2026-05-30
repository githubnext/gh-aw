# CLI Performance Engineering Guide

This guide helps you optimize the gh-aw CLI for fast command response times and efficient resource usage.

## Performance Targets

- **Help commands**: < 50ms
- **List operations**: < 100ms
- **Single workflow compilation**: < 300ms
- **Full compilation (all workflows)**: < 10s
- **MCP inspection**: < 2s

## Quick Performance Checks

```bash
# Measure command latency
time ./gh-aw --help
time ./gh-aw compile workflow.md
time ./gh-aw mcp list

# Profile CLI startup
go build -o gh-aw-debug ./cmd/gh-aw
time ./gh-aw-debug --help
```

## Common Performance Bottlenecks

### 1. Startup Time / Initialization Overhead

**Problem**: CLI loads unnecessary resources at startup.

**Measurement**:
```bash
# Measure startup overhead
time ./gh-aw --version  # Should be instant

# Compare with more complex command
time ./gh-aw compile --help
```

**Optimization Strategy**:

**Lazy Loading**:
```go
// BAD: Load schemas at package init
var Schemas = loadAllSchemas()

// GOOD: Load schemas when needed
var schemasOnce sync.Once
var schemas map[string]*jsonschema.Schema

func GetSchemas() map[string]*jsonschema.Schema {
    schemasOnce.Do(func() {
        schemas = loadAllSchemas()
    })
    return schemas
}
```

**Defer Expensive Operations**:
```go
// Only load if subcommand needs it
func runCompile(cmd *cobra.Command, args []string) error {
    // Load schemas here, not at package init
    schemas := GetSchemas()
    // ...
}
```

**Impact**: 50-100ms reduction in startup time for simple commands

### 2. Sequential Workflow Compilation

**Problem**: `gh-aw compile` processes workflows one at a time.

**Current Behavior**:
```go
for _, workflow := range workflows {
    compile(workflow)  // Sequential
}
```

**Measurement**:
```bash
# Time current sequential compilation
time ./gh-aw compile  # All workflows

# Estimate parallel potential
WORKFLOW_COUNT=$(ls .github/workflows/*.md | wc -l)
echo "Workflows: $WORKFLOW_COUNT"
```

**Optimization Strategy**:

**Parallel Compilation**:
```go
var wg sync.WaitGroup
semaphore := make(chan struct{}, runtime.NumCPU())

for _, workflow := range workflows {
    wg.Add(1)
    go func(w Workflow) {
        defer wg.Done()
        semaphore <- struct{}{}        // Acquire
        defer func() { <-semaphore }() // Release
        
        compile(w)
    }(workflow)
}
wg.Wait()
```

**Impact**: 30-50% reduction for full compilation (depends on CPU cores)

### 3. Unnecessary File I/O

**Problem**: Reading files multiple times or reading unnecessary files.

**Measurement**:
```bash
# Trace system calls
strace -c ./gh-aw compile workflow.md 2>&1 | grep -E "read|open|stat"
```

**Optimization Strategy**:

**Cache File Reads**:
```go
type FileCache struct {
    cache map[string][]byte
    mu    sync.RWMutex
}

func (fc *FileCache) ReadFile(path string) ([]byte, error) {
    fc.mu.RLock()
    if data, ok := fc.cache[path]; ok {
        fc.mu.RUnlock()
        return data, nil
    }
    fc.mu.RUnlock()
    
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    
    fc.mu.Lock()
    fc.cache[path] = data
    fc.mu.Unlock()
    
    return data, nil
}
```

**Only Read What's Needed**:
```go
// BAD: Read entire workflow directory
files, _ := os.ReadDir(".github/workflows")

// GOOD: Use specific patterns
files, _ := filepath.Glob(".github/workflows/*.md")
```

### 4. Regex Compilation in Hot Paths

**Problem**: Compiling regex patterns inside loops.

**Bad Pattern**:
```go
for _, line := range lines {
    re := regexp.MustCompile(`pattern`)  // BAD - recompiled every iteration
    if re.MatchString(line) {
        // ...
    }
}
```

**Good Pattern**:
```go
var linePattern = regexp.MustCompile(`pattern`)  // Compiled once

for _, line := range lines {
    if linePattern.MatchString(line) {
        // ...
    }
}
```

**Measurement**:
```bash
# Benchmark regex usage
go test -bench=BenchmarkRegex -benchmem ./pkg/parser
```

## CLI-Specific Optimizations

### Fast Help Display

```go
// Pre-compute help text
var helpText = buildHelpText()

func runHelp(cmd *cobra.Command, args []string) error {
    fmt.Print(helpText)  // Instant display
    return nil
}
```

### Efficient Logging

```go
// Use debug logger for expensive operations
var log = logger.New("cli:compile")

func compile(workflow string) {
    if log.Enabled() {  // Only compute if debug mode
        log.Printf("Compiling workflow with %d steps", expensiveCount())
    }
}
```

### Progress Indicators for Long Operations

```go
import "github.com/briandowns/spinner"

func compileAll(workflows []string) error {
    sp := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
    sp.Suffix = " Compiling workflows..."
    sp.Start()
    defer sp.Stop()
    
    for _, w := range workflows {
        compile(w)
    }
    return nil
}
```

## Performance Profiling for CLI

### CPU Profiling

```bash
# Build with profiling
go build -o gh-aw-prof ./cmd/gh-aw

# Run with CPU profiling
./gh-aw-prof compile -cpuprofile=cpu.prof

# Analyze
go tool pprof cpu.prof
# Commands:
#   top10      - Hottest functions
#   list main  - Show main package
#   web        - Visual graph
```

### Memory Profiling

```bash
# Run with memory profiling
./gh-aw-prof compile -memprofile=mem.prof

# Analyze allocations
go tool pprof -alloc_space mem.prof

# Find memory leaks
go tool pprof -inuse_space mem.prof
```

### Benchmark CLI Commands

```go
// In pkg/cli/compile_test.go
func BenchmarkCompileSingleWorkflow(b *testing.B) {
    content := loadTestWorkflow()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := compile.Workflow(content)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkCompileAllWorkflows(b *testing.B) {
    workflows := loadAllWorkflows()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        for _, w := range workflows {
            _, err := compile.Workflow(w)
            if err != nil {
                b.Fatal(err)
            }
        }
    }
}
```

### Run Benchmarks

```bash
# Run all CLI benchmarks
go test -bench=. -benchmem ./pkg/cli/...

# Compare before/after
go test -bench=. -benchmem ./pkg/cli/... > baseline.txt
# Make changes
go test -bench=. -benchmem ./pkg/cli/... > optimized.txt

# Compare with benchstat
go install golang.org/x/perf/cmd/benchstat@latest
benchstat baseline.txt optimized.txt
```

## Incremental Compilation Strategy

**Problem**: Full recompilation is wasteful when only one workflow changed.

**Detection of Changes**:
```go
func findChangedWorkflows() ([]string, error) {
    // Get git diff
    cmd := exec.Command("git", "diff", "--name-only", ".github/workflows/*.md")
    output, err := cmd.Output()
    if err != nil {
        return nil, err
    }
    
    // Parse changed files
    files := strings.Split(string(output), "\n")
    return files, nil
}
```

**Compilation Cache**:
```go
type CompilationCache struct {
    hashes map[string]string  // workflow -> hash
    path   string
}

func (c *CompilationCache) NeedsRecompile(workflow string) (bool, error) {
    content, err := os.ReadFile(workflow)
    if err != nil {
        return false, err
    }
    
    hash := computeHash(content)
    oldHash, exists := c.hashes[workflow]
    
    if !exists || oldHash != hash {
        c.hashes[workflow] = hash
        c.save()
        return true, nil
    }
    
    return false, nil
}
```

**Usage**:
```bash
# Only recompile changed workflows
./gh-aw compile --incremental

# Force full recompilation
./gh-aw compile --force
```

## Memory Optimization

### String Building

```go
// BAD: Repeated concatenation
result := ""
for _, s := range strings {
    result += s  // Creates new string each time
}

// GOOD: Use strings.Builder
var sb strings.Builder
for _, s := range strings {
    sb.WriteString(s)
}
result := sb.String()
```

### Slice Preallocation

```go
// BAD: Unknown capacity
items := []Item{}
for i := 0; i < 1000; i++ {
    items = append(items, Item{})  // Frequent reallocation
}

// GOOD: Preallocate
items := make([]Item, 0, 1000)
for i := 0; i < 1000; i++ {
    items = append(items, Item{})  // No reallocation
}
```

### Object Pooling for Temporary Buffers

```go
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func processWorkflow(w Workflow) {
    buf := bufferPool.Get().(*bytes.Buffer)
    buf.Reset()
    defer bufferPool.Put(buf)
    
    // Use buf for temporary work
    // ...
}
```

## Performance Testing Workflow

### 1. Establish Baseline

```bash
# Measure current performance
time ./gh-aw --help
time ./gh-aw compile workflow.md
time ./gh-aw compile  # All workflows
time ./gh-aw mcp list
```

### 2. Profile Hot Paths

```bash
# CPU profile
go test -cpuprofile=cpu.prof -run=TestCompile ./pkg/cli
go tool pprof cpu.prof

# Memory profile  
go test -memprofile=mem.prof -run=TestCompile ./pkg/cli
go tool pprof mem.prof
```

### 3. Apply Optimization

Focus on:
- Lazy loading for initialization
- Parallel processing where safe
- Caching for repeated operations
- Efficient data structures

### 4. Measure Impact

```bash
# Compare timings
time ./gh-aw compile workflow.md  # Should be faster

# Run benchmarks
go test -bench=. -benchmem ./pkg/cli/... > after.txt
benchstat before.txt after.txt
```

### 5. Validate Correctness

```bash
# Ensure CLI still works
make test
./gh-aw compile --validate
```

## Common Performance Patterns

### ✓ Good Patterns

```go
// Lazy initialization
var configOnce sync.Once
var config *Config

func GetConfig() *Config {
    configOnce.Do(func() {
        config = loadConfig()
    })
    return config
}

// Early return on validation
func validate(input string) error {
    if input == "" {
        return ErrEmpty  // Fast path
    }
    // Expensive validation only if needed
    return deepValidation(input)
}

// Buffered I/O
writer := bufio.NewWriter(file)
writer.WriteString(data)
writer.Flush()
```

### ✗ Anti-Patterns to Avoid

```go
// Don't load everything at init
func init() {
    allSchemas := loadAllSchemas()      // BAD - may not be needed
    allWorkflows := loadAllWorkflows()  // BAD - expensive
}

// Don't use string concatenation in loops
for _, item := range items {
    result += item.String()  // BAD - use strings.Builder
}

// Don't ignore error handling for performance
data, _ := os.ReadFile(path)  // BAD - silent failures
```

## Performance Checklist

Before committing CLI performance changes:

- [ ] Measure baseline: `time ./gh-aw <command>`
- [ ] Profile: `go test -cpuprofile=cpu.prof`
- [ ] Run benchmarks: `go test -bench=.`
- [ ] Apply optimization
- [ ] Measure improvement (should be >10%)
- [ ] Run full test suite: `make test`
- [ ] Verify no regressions: `benchstat before.txt after.txt`
- [ ] Test on both Linux and macOS if possible

## Troubleshooting

### Problem: Slow startup time

**Diagnosis**:
```bash
time ./gh-aw --version  # Should be instant
go build -o gh-aw-debug ./cmd/gh-aw
time ./gh-aw-debug --help
```

**Solution**: Look for `init()` functions loading heavy resources. Move to lazy loading.

### Problem: High memory usage

**Diagnosis**:
```bash
go test -memprofile=mem.prof -run=TestCompile ./pkg/cli
go tool pprof -alloc_space mem.prof
```

**Solution**: Look for:
- String concatenation in loops
- Slice growth without preallocation
- Large objects not being freed

### Problem: Slow compilation

**Diagnosis**:
```bash
time ./gh-aw compile workflow.md --verbose
# Look for slow operations in verbose output
```

**Solution**:
- Cache schema validation
- Use parallel compilation
- Implement incremental compilation

## Resources

- Go Performance: https://go.dev/doc/effective_go#performance
- Profiling: https://go.dev/blog/pprof
- Benchmarking: https://dave.cheney.net/2013/06/30/how-to-write-benchmarks-in-go
- Memory optimization: https://go.dev/blog/ismmkeynote
