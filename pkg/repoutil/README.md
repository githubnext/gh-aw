# repoutil Package

The `repoutil` package provides utility functions for working with GitHub repository slugs.

## Overview

This package offers a single focused helper for parsing and validating `owner/repo` repository slug strings, which are used throughout the codebase wherever GitHub repositories are referenced.

## Functions

### `SplitRepoSlug(slug string) (owner, repo string, err error)`

Splits a repository slug of the form `owner/repo` into its two components. Returns an error when the slug does not contain exactly one `/` separator or when either the owner or repository name is empty.

```go
import "github.com/github/gh-aw/pkg/repoutil"

owner, repo, err := repoutil.SplitRepoSlug("github/gh-aw")
if err != nil {
    return fmt.Errorf("invalid repository: %w", err)
}
// owner = "github", repo = "gh-aw"
```

**Error cases**:

```go
// Missing separator
repoutil.SplitRepoSlug("github")          // error: invalid repo format: github

// Empty component
repoutil.SplitRepoSlug("/gh-aw")          // error: invalid repo format: /gh-aw
repoutil.SplitRepoSlug("github/")         // error: invalid repo format: github/

// Too many separators
repoutil.SplitRepoSlug("github/gh-aw/x") // error: invalid repo format: github/gh-aw/x
```

## Design Notes

- All debug output uses `logger.New("repoutil:repoutil")` and is only emitted when `DEBUG=repoutil:*`.
- For paths that include sub-folders (e.g. GitHub Actions `uses:` fields such as `github/codeql-action/upload-sarif`), use `gitutil.ExtractBaseRepo` first to strip the sub-path before calling `SplitRepoSlug`.
