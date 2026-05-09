# repoutil Package

The `repoutil` package provides utility functions for working with GitHub repository slugs.

## Overview

This package offers a single focused helper for parsing and validating `owner/repo` repository slug strings, which are used throughout the codebase wherever GitHub repositories are referenced.

## Public API

### Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `SplitRepoSlug` | `func(slug string) (owner, repo string, err error)` | Splits a repository slug of the form `owner/repo` into its two components; returns an error when the slug does not contain exactly one `/` or when either component is empty |

## Usage Examples

```go
import "github.com/github/gh-aw/pkg/repoutil"

owner, repo, err := repoutil.SplitRepoSlug("github/gh-aw")
if err != nil {
    return fmt.Errorf("invalid repository: %w", err)
}
// owner = "github", repo = "gh-aw"
```

## Dependencies

**Internal**:
- `github.com/github/gh-aw/pkg/logger` — debug logging

## Design Notes

- All debug output uses `logger.New("repoutil:repoutil")` and is only emitted when `DEBUG=repoutil:*`.
- For paths that include sub-folders (e.g. GitHub Actions `uses:` fields such as `github/codeql-action/upload-sarif`), use `gitutil.ExtractBaseRepo` first to strip the sub-path before calling `SplitRepoSlug`.

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
