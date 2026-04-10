# Functional Patterns Reference

This document contains reference implementations and guidelines for the **Functional Pragmatist** workflow. The agent reads this file when implementing functional/immutability improvements.

## Helper Implementations

### Slice Helpers (`pkg/fp/`)

```go
// pkg/fp/slice.go - Example helpers for common operations
package fp

// Map transforms each element in a slice
// Note: uses var+append to avoid CodeQL violations from make([]U, len(slice))
func Map[T, U any](slice []T, fn func(T) U) []U {
    var result []U
    for _, v := range slice {
        result = append(result, fn(v))
    }
    return result
}

// Filter returns elements that match the predicate
func Filter[T any](slice []T, fn func(T) bool) []T {
    var result []T
    for _, v := range slice {
        if fn(v) {
            result = append(result, v)
        }
    }
    return result
}

// Reduce aggregates slice elements
func Reduce[T, U any](slice []T, initial U, fn func(U, T) U) U {
    result := initial
    for _, v := range slice {
        result = fn(result, v)
    }
    return result
}
```

### Reusable Logic Wrappers

```go
// Retry wrapper with exponential backoff
func Retry[T any](attempts int, delay time.Duration, fn func() (T, error)) (T, error) {
    var result T
    var err error
    for i := 0; i < attempts; i++ {
        result, err = fn()
        if err == nil {
            return result, nil
        }
        if i < attempts-1 {
            time.Sleep(delay * time.Duration(1<<i))  // Exponential backoff
        }
    }
    return result, fmt.Errorf("failed after %d attempts: %w", attempts, err)
}

// Usage:
data, err := Retry(3, time.Second, func() ([]byte, error) {
    return fetchFromAPI(url)
})
```

```go
// Timing wrapper for performance logging
func WithTiming[T any](name string, logger Logger, fn func() T) T {
    start := time.Now()
    result := fn()
    logger.Printf("%s took %v", name, time.Since(start))
    return result
}

// Usage:
result := WithTiming("database query", logger, func() []Record {
    return db.Query(sql)
})
```

```go
// Memoization wrapper for caching
func Memoize[K comparable, V any](fn func(K) V) func(K) V {
    cache := make(map[K]V)
    var mu sync.RWMutex

    return func(key K) V {
        mu.RLock()
        if val, ok := cache[key]; ok {
            mu.RUnlock()
            return val
        }
        mu.RUnlock()

        val := fn(key)

        mu.Lock()
        cache[key] = val
        mu.Unlock()

        return val
    }
}

// Usage:
expensiveCalc := Memoize(func(n int) int {
    // expensive computation
    return fibonacci(n)
})
```

```go
// Error handling wrapper
func Must[T any](val T, err error) T {
    if err != nil {
        panic(err)
    }
    return val
}

// Usage in initialization:
config := Must(LoadConfig("config.yaml"))
```

```go
// Conditional execution wrapper
func When[T any](condition bool, fn func() T, defaultVal T) T {
    if condition {
        return fn()
    }
    return defaultVal
}

// Usage:
result := When(useCache, func() Data { return cache.Get(key) }, fetchFromDB(key))
```

## Transformation Examples

### Immutability Improvements

```go
// Before: Multiple mutations
var config Config
config.Host = getHost()
config.Port = getPort()
config.Timeout = getTimeout()

// After: Single initialization
config := Config{
    Host:    getHost(),
    Port:    getPort(),
    Timeout: getTimeout(),
}
```

### Functional Initialization Patterns

```go
// Before: Imperative building
result := make(map[string]string)
result["name"] = name
result["version"] = version
result["status"] = "active"

// After: Declarative initialization
result := map[string]string{
    "name":    name,
    "version": version,
    "status":  "active",
}
```

### Transformative Operations

```go
// Before: Imperative filtering and mapping
var activeNames []string
for _, item := range items {
    if item.Active {
        activeNames = append(activeNames, item.Name)
    }
}

// After: Functional pipeline
activeItems := sliceutil.Filter(items, func(item Item) bool { return item.Active })
activeNames := sliceutil.Map(activeItems, func(item Item) string { return item.Name })

// Note: Sometimes inline is clearer - use judgment!
```

### Functional Options Pattern

```go
// Before: Constructor with many parameters
func NewClient(host string, port int, timeout time.Duration, retries int, logger Logger) *Client {
    return &Client{
        host:    host,
        port:    port,
        timeout: timeout,
        retries: retries,
        logger:  logger,
    }
}

// After: Functional options pattern
type ClientOption func(*Client)

func WithTimeout(d time.Duration) ClientOption {
    return func(c *Client) {
        c.timeout = d
    }
}

func WithRetries(n int) ClientOption {
    return func(c *Client) {
        c.retries = n
    }
}

func WithLogger(l Logger) ClientOption {
    return func(c *Client) {
        c.logger = l
    }
}

func NewClient(host string, port int, opts ...ClientOption) *Client {
    c := &Client{
        host:    host,
        port:    port,
        timeout: 30 * time.Second,  // sensible default
        retries: 3,                  // sensible default
        logger:  defaultLogger,      // sensible default
    }
    for _, opt := range opts {
        opt(c)
    }
    return c
}

// Usage: client := NewClient("localhost", 8080, WithTimeout(time.Minute), WithRetries(5))
```

### Eliminating Shared Mutable State

```go
// Before: Global mutable state
var (
    globalConfig *Config
    configMutex  sync.RWMutex
)

func GetSetting(key string) string {
    configMutex.RLock()
    defer configMutex.RUnlock()
    return globalConfig.Settings[key]
}

// After: Explicit parameter passing
type Service struct {
    config *Config  // Immutable after construction
}

func NewService(config *Config) *Service {
    return &Service{config: config}
}

func (s *Service) ProcessRequest(req Request) Response {
    setting := s.config.Settings["timeout"]
    // ... use setting
}
```

### Extracting Pure Functions

```go
// Before: Mixed pure and impure logic
func ProcessOrder(order Order) error {
    log.Printf("Processing order %s", order.ID)  // Side effect

    total := 0.0
    for _, item := range order.Items {
        total += item.Price * float64(item.Quantity)
    }

    if total > 1000 {
        total *= 0.9  // 10% discount
    }

    db.Save(order.ID, total)  // Side effect
    return nil
}

// After: Pure calculation extracted
func CalculateOrderTotal(items []OrderItem) float64 {
    total := 0.0
    for _, item := range items {
        total += item.Price * float64(item.Quantity)
    }
    return total
}

func ApplyDiscounts(total float64) float64 {
    if total > 1000 {
        return total * 0.9
    }
    return total
}

// Impure orchestration - side effects are explicit and isolated
func ProcessOrder(order Order, db Database, logger Logger) error {
    logger.Printf("Processing order %s", order.ID)
    total := ApplyDiscounts(CalculateOrderTotal(order.Items))
    return db.Save(order.ID, total)
}
```

## Guidelines

### Test-Driven Refactoring

**CRITICAL: Always verify test coverage before refactoring:**

```bash
go test -cover ./pkg/path/to/package/
```

**Workflow:**
1. **Check coverage** - Verify tests exist (minimum 60% coverage)
2. **Write tests first** - If coverage is low, add tests for current behavior
3. **Verify tests pass** - Green tests before refactoring
4. **Refactor** - Make functional/immutability improvements
5. **Verify tests pass** - Green tests after refactoring

**For new helper functions (`pkg/fp/`):** Write tests FIRST, aim for >80% coverage, use table-driven tests.

### Balance Pragmatism and Purity

- **DO** make data immutable when it improves safety and clarity
- **DO** use functional patterns for data transformations
- **DO** use functional options for extensible APIs
- **DO** extract pure functions to improve testability
- **DO** eliminate shared mutable state where practical
- **DON'T** force functional patterns where imperative is clearer
- **DON'T** create overly complex abstractions for simple operations
- **DON'T** add unnecessary wrappers for one-off operations

### Functional Options Pattern Guidelines

**Use when:** Constructor has 4+ optional parameters, API needs to be extended without breaking changes, configuration has sensible defaults.

**Don't use when:** All parameters are required, constructor has 1-2 simple parameters, configuration is unlikely to change.

```go
// Option type convention
type Option func(*Config)

// Option function naming: With* prefix
func WithTimeout(d time.Duration) Option

// Required parameters stay positional
func New(required1 string, required2 int, opts ...Option) *T
```

### Pure Functions Guidelines

**Characteristics:** Same input → same output, no side effects, no dependency on external mutable state.

**When to extract:** Business logic, validation logic, formatting/parsing, any computation without I/O.

```go
// Pure core, impure shell pattern
func ProcessOrder(order Order, db Database, logger Logger) error {
    validated := ValidateOrder(order)      // Pure
    total := CalculateTotal(validated)     // Pure
    discounted := ApplyDiscounts(total)    // Pure
    return db.Save(order.ID, discounted)   // Side effect isolated here
}
```

### Avoiding Shared Mutable State

**Strategies:**
1. Pass dependencies through constructors
2. Load configuration once, never modify
3. Use context for per-request data
4. Keep mutable state at the edges

```go
// ❌ Global mutable state
var config *Config
var cache = make(map[string]Result)

// ✅ Explicit dependency
type Service struct { config *Config }

// ✅ Encapsulated state
type Cache struct {
    mu   sync.RWMutex
    data map[string]Result
}
```

### Reusable Wrappers Guidelines

**Create when:** Pattern appears 3+ times, cross-cutting concern, complex logic benefits from abstraction.

**Don't create when:** One-off usage, simple inline code is clearer, wrapper hides important details.

**Design:** Keep focused on one concern, use generics for type safety, handle errors appropriately.

### When to Use Inline vs Helpers

**Use inline when:** Operation is simple and used once, inline version is clearer.

**Use helper when:** Pattern appears 3+ times, helper significantly improves clarity, operation is complex.

### Go-Specific Considerations

- Go doesn't have built-in map/filter/reduce — that's okay!
- Inline loops are often clearer than generic helpers
- Use type parameters (generics) for helpers to avoid reflection
- Avoid `make([]T, len(input))` — use `var result []T` + `append` (CodeQL flags length-derived allocation)
- Simple for-loops are idiomatic Go — don't force functional style
- Functional options is a well-established Go pattern — use it confidently
- Explicit parameter passing is idiomatic Go — prefer it over globals

### Immutability by Convention

```go
// Unexported fields signal "don't modify"
type Config struct {
    host string
    port int
}

// Exported getters, no setters
func (c *Config) Host() string { return c.host }

// Constructor enforcement
func NewConfig(host string, port int) (*Config, error) {
    if host == "" {
        return nil, errors.New("host required")
    }
    return &Config{host: host, port: port}, nil
}

// Defensive copying
func (s *Service) GetItems() []Item {
    return slices.Clone(s.items)
}
```

### Risk Management

**Low Risk (Prioritize):**
- Converting `var x T; x = value` to `x := value`
- Replacing empty slice/map initialization with literals
- Making struct initialization more declarative
- Extracting pure helper functions (no API change)

**Medium Risk (Review carefully):**
- Converting range loops to functional patterns
- Adding new helper functions
- Applying functional options to internal constructors
- Extracting pure functions from larger functions

**High Risk (Avoid or verify thoroughly):**
- Changes to public APIs
- Modifications to concurrency patterns
- Changes affecting error handling flow
- Eliminating shared state used across packages
- Adding wrappers that change control flow
