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
| `FindGitRoot` | `func() (string, error)` | Returns the absolute path of the root directory of the current Git repository using pure Go filesystem traversal (no `git` subprocess); starts from the current working directory |
| `FindGitRootFrom` | `func(startDir string) (string, error)` | Like `FindGitRoot` but starts from `startDir`; traverses upward looking for a `.git` directory or worktree marker file |
| `ReadFileFromHEAD` | `func(filePath, gitRoot string) (string, error)` | Reads a file's content from the `HEAD` commit without touching the working tree; rejects paths that escape the repository. `filePath` is resolved from the current working directory, so prefer an absolute path under `gitRoot` |

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

// Find the git repository root (pure Go, no git subprocess)
root, err := gitutil.FindGitRoot()
if err != nil {
    return fmt.Errorf("not in a git repository: %w", err)
}

// Find the git root starting from a specific directory
root, err = gitutil.FindGitRootFrom("/some/subdir")
if err != nil {
    return fmt.Errorf("not in a git repository: %w", err)
}

// Read a file from the HEAD commit (prefer absolute paths under root)
content, err := gitutil.ReadFileFromHEAD(filepath.Join(root, "go.mod"), root)
```

## Dependencies

**Internal**:
- `github.com/github/gh-aw/pkg/logger` — debug logging

## Design Notes

- `FindGitRoot` uses pure Go filesystem traversal (walking up the directory tree looking for `.git`), avoiding the need for a `git` executable on `PATH`. This is important for Rosetta 2 compatibility on macOS ARM64 and restricted environments.
- `FindGitRootFrom` accepts both normal `.git` directories and worktree marker files (a `.git` file starting with `gitdir:`).

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
