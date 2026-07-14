# stats Package

> Incremental descriptive statistics for float64 observation streams.

## Overview

The `stats` package provides `StatVar`, a compact accumulator for numeric metrics. It tracks count, min, max, mean, sample variance, sample standard deviation, and median. Mean and variance are maintained with Welford's online algorithm, while exact median is computed from stored observations.

## Public API

### Types

| Type | Kind | Description |
|------|------|-------------|
| `StatVar` | struct | Accumulates observations and exposes descriptive statistics |

### `StatVar` Methods

| Method | Signature | Description |
|--------|-----------|-------------|
| `Add` | `func(v float64)` | Adds one observation |
| `Count` | `func() int` | Returns the number of observations |
| `Min` | `func() float64` | Returns the minimum observed value (or `0` if empty) |
| `Max` | `func() float64` | Returns the maximum observed value (or `0` if empty) |
| `Mean` | `func() float64` | Returns the arithmetic mean (or `0` if empty) |
| `SampleVariance` | `func() float64` | Returns sample variance (`N-1` denominator) |
| `SampleStdDev` | `func() float64` | Returns sample standard deviation |
| `Median` | `func() float64` | Returns the exact median (middle value or midpoint of two middle values) |

## Usage Examples

```go
var s stats.StatVar

s.Add(10)
s.Add(20)
s.Add(30)

fmt.Println(s.Count())  // 3
fmt.Println(s.Mean())   // 20
fmt.Println(s.SampleStdDev()) // 10.0 (sample std dev)
fmt.Println(s.Median()) // 20
```

## Dependencies

**Internal**:
- `github.com/github/gh-aw/pkg/logger` — debug logging for non-finite observations

**Standard library**:
- `math` — square root for standard deviation
- `sort` — sorting copied values for exact median

## Thread Safety

`StatVar` is not concurrency-safe. Use external synchronization when a single instance is shared across goroutines.

<!-- BEGIN SOURCE-VERIFIED EXPORT COVERAGE -->
## Source-verified export coverage

This appendix is generated from the current non-test Go source files in this package and records any exported top-level symbols that are not already described above.

| Category | Count |
|----------|------:|
| Types | 1 |
| Constants | 0 |
| Variables | 0 |
| Functions and methods | 8 |
| Additional symbols documented in this appendix | 0 |

The sections above already mention every exported top-level symbol in the current source tree.
<!-- END SOURCE-VERIFIED EXPORT COVERAGE -->

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
