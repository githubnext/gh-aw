# semverutil Package

The `semverutil` package provides shared semantic versioning primitives used across `pkg/workflow` and `pkg/cli`. Centralizing these helpers ensures that semver parsing, comparison, and compatibility logic is defined in one place.

## Overview

This package wraps `golang.org/x/mod/semver` with additional helpers for:
- Normalizing version strings (adding the required `v` prefix).
- Validating GitHub Actions version tags (`vmajor`, `vmajor.minor`, `vmajor.minor.patch`).
- Parsing versions into a structured `SemanticVersion` type.
- Comparing and checking compatibility between version strings.

## Types

### `SemanticVersion`

A parsed semantic version with individual numeric components.

```go
type SemanticVersion struct {
    Major int
    Minor int
    Patch int
    Pre   string // Prerelease identifier without leading hyphen (e.g. "beta.1")
    Raw   string // Original version string without leading "v"
}
```

#### Methods

| Method | Description |
|--------|-------------|
| `IsPreciseVersion() bool` | Returns `true` if the version has at least two dots (e.g. `v6.0.0` is precise, `v6` is not) |
| `IsNewer(other *SemanticVersion) bool` | Returns `true` if this version is newer than `other` |

## Functions

### `EnsureVPrefix(v string) string`

Adds a leading `"v"` if `v` does not already have one. Required because `golang.org/x/mod/semver` demands the prefix.

```go
semverutil.EnsureVPrefix("1.2.3")  // → "v1.2.3"
semverutil.EnsureVPrefix("v1.2.3") // → "v1.2.3"
```

### `IsActionVersionTag(s string) bool`

Reports whether `s` is a valid GitHub Actions version tag. Accepted forms: `vmajor`, `vmajor.minor`, `vmajor.minor.patch`. Prerelease and build-metadata suffixes are **not** accepted.

```go
semverutil.IsActionVersionTag("v4")       // true
semverutil.IsActionVersionTag("v4.1")     // true
semverutil.IsActionVersionTag("v4.1.0")   // true
semverutil.IsActionVersionTag("v4.1.0-rc") // false
```

### `IsValid(ref string) bool`

Reports whether `ref` is a valid semantic version string (accepts any valid semver including prerelease/build-metadata, and bare versions without `"v"`).

```go
semverutil.IsValid("1.2.3")       // true
semverutil.IsValid("v1.2.3-beta") // true
semverutil.IsValid("not-a-ver")   // false
```

### `ParseVersion(v string) *SemanticVersion`

Parses `v` into a `SemanticVersion`. Returns `nil` if `v` is not a valid semver string.

```go
ver := semverutil.ParseVersion("v1.2.3")
if ver != nil {
    fmt.Println(ver.Major, ver.Minor, ver.Patch) // 1 2 3
}
```

### `Compare(v1, v2 string) int`

Compares two semantic versions using `golang.org/x/mod/semver`. Returns `1` if `v1 > v2`, `-1` if `v1 < v2`, or `0` if equal. Bare versions (without `"v"`) are accepted.

```go
semverutil.Compare("v2.0.0", "v1.9.9") // 1  (v2 is newer)
semverutil.Compare("v1.0.0", "v1.0.0") // 0  (equal)
semverutil.Compare("v0.9.0", "v1.0.0") // -1 (v0.9 is older)
```

### `IsCompatible(pinVersion, requestedVersion string) bool`

Reports whether `pinVersion` is semver-compatible with `requestedVersion`. Compatibility is defined as sharing the same major version.

```go
semverutil.IsCompatible("v5.0.0", "v5")     // true
semverutil.IsCompatible("v5.1.0", "v5.0.0") // true
semverutil.IsCompatible("v6.0.0", "v5")     // false
```

## Design Notes

- All debug output uses `logger.New("semverutil:semverutil")` and is only emitted when `DEBUG=semverutil:*`.
- The package intentionally delegates to `golang.org/x/mod/semver` for canonical semver logic rather than implementing its own parsing.
- `ParseVersion` uses `semver.Canonical` before splitting into components, ensuring correct handling of short forms like `v1` (canonicalized to `v1.0.0`).
