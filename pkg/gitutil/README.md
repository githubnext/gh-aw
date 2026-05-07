# gitutil Package

The `gitutil` package provides utility functions for interacting with Git repositories and classifying GitHub API errors.

## Overview

This package contains helpers for:
- Detecting rate-limit and authentication errors from GitHub API responses.
- Validating hex strings (e.g. commit SHAs).
- Extracting base repository slugs from action paths.
- Finding the root directory of the current Git repository.
- Reading file contents from the `HEAD` commit.

## Public API

### Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `IsRateLimitError` | `func(errMsg string) bool` | Returns `true` when `errMsg` indicates a GitHub API rate-limit error (case-insensitive match against "api rate limit exceeded", "rate limit exceeded", or "secondary rate limit") |
| `IsAuthError` | `func(errMsg string) bool` | Returns `true` when `errMsg` indicates an authentication or authorization failure (`GH_TOKEN`, `GITHUB_TOKEN`, `unauthorized`, `forbidden`, SAML enforcement, etc.) |
| `IsHexString` | `func(s string) bool` | Returns `true` if `s` consists entirely of hexadecimal characters (`0–9`, `a–f`, `A–F`); returns `false` for the empty string |
| `IsValidFullSHA` | `func(s string) bool` | Returns `true` if `s` is a valid 40-character lowercase hexadecimal SHA |
| `ExtractBaseRepo` | `func(repoPath string) string` | Extracts the `owner/repo` portion from an action path that may include a sub-folder (e.g. `github/codeql-action/upload-sarif` → `github/codeql-action`) |
| `FindGitRoot` | `func() (string, error)` | Returns the absolute path of the root directory of the current Git repository by running `git rev-parse --show-toplevel` |
| `ReadFileFromHEADWithRoot` | `func(filePath, gitRoot string) (string, error)` | Reads a file's content from the `HEAD` commit without touching the working tree; rejects paths that escape the repository |

## Usage Examples

```go
import "github.com/github/gh-aw/pkg/gitutil"

// Check for rate-limit errors from GitHub API
if gitutil.IsRateLimitError(err.Error()) {
    // Back off and retry
}

// Validate a commit SHA
if gitutil.IsValidFullSHA(commitSHA) {
    fmt.Println("Valid 40-character commit SHA")
}

// Find the git repository root
root, err := gitutil.FindGitRoot()
if err != nil {
    return fmt.Errorf("not in a git repository: %w", err)
}

// Read a file from the HEAD commit
content, err := gitutil.ReadFileFromHEADWithRoot("go.mod", root)
```

## Dependencies

**Internal**:
- `pkg/logger` — debug logging

## Design Notes

- All debug output uses `logger.New("gitutil:gitutil")` and is only emitted when `DEBUG=gitutil:*`.
- Error classification is case-insensitive string matching — no external dependency on GitHub API client types.
- `ReadFileFromHEADWithRoot` uses `git show HEAD:<relpath>` and resolves paths with `filepath.Rel` to prevent path-traversal attacks.

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
