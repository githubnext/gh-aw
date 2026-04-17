# stats Package

> Incremental descriptive statistics for float64 observation streams.

## Overview

The `stats` package provides `StatVar`, a compact accumulator for numeric metrics. It tracks count, sum, min, max, mean, variance, standard deviation, and median. Mean and variance are maintained with Welford's online algorithm, while exact median is computed from stored observations.

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
| `Sum` | `func() float64` | Returns the arithmetic sum |
| `Min` | `func() float64` | Returns the minimum observed value (or `0` if empty) |
| `Max` | `func() float64` | Returns the maximum observed value (or `0` if empty) |
| `Mean` | `func() float64` | Returns the arithmetic mean (or `0` if empty) |
| `Variance` | `func() float64` | Returns population variance (`N` denominator) |
| `SampleVariance` | `func() float64` | Returns sample variance (`N-1` denominator) |
| `StdDev` | `func() float64` | Returns population standard deviation |
| `SampleStdDev` | `func() float64` | Returns sample standard deviation |
| `Median` | `func() float64` | Returns the exact median (middle value or midpoint of two middle values) |

## Usage Example

```go
var s stats.StatVar

s.Add(10)
s.Add(20)
s.Add(30)

fmt.Println(s.Count())  // 3
fmt.Println(s.Mean())   // 20
fmt.Println(s.StdDev()) // 8.164965...
fmt.Println(s.Median()) // 20
```

## Dependencies

**Standard library only**:
- `math` — square root for standard deviation
- `sort` — sorting copied values for exact median

## Thread Safety

`StatVar` is not concurrency-safe. Use external synchronization when a single instance is shared across goroutines.

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
