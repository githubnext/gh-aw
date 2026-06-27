# gitutil Package

> Utility functions for Git repository operations and GitHub API error classification.

## Overview

The `gitutil` package contains helpers for:
- Detecting rate-limit and authentication errors from GitHub API responses.
- Validating hex strings (e.g. commit SHAs).
- Extracting base repository slugs from action paths.
- Finding the root directory of the current Git repository using pure Go filesystem traversal.
- Reading file contents from the `HEAD` commit via a `git` subprocess.

## Public API

### Variables

| Variable | Type | Description |
|----------|------|-------------|
| `ErrNotGitRepository` | `error` | Sentinel error returned by `FindGitRoot` and `FindGitRootFrom` when no `.git` entry is found while traversing up to the filesystem root |

### Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `IsRateLimitError` | `func(errMsg string) bool` | Returns `true` when `errMsg` indicates a GitHub API rate-limit error (case-insensitive match against "api rate limit exceeded", "rate limit exceeded", or "secondary rate limit") |
| `IsAuthError` | `func(errMsg string) bool` | Returns `true` when `errMsg` indicates an authentication or authorization failure (case-insensitive match against `GH_TOKEN`, `GITHUB_TOKEN`, `authentication`, `not logged into`, `unauthorized`, `forbidden`, `permission denied`, or `SAML enforcement`) |
| `IsHexString` | `func(s string) bool` | Returns `true` if `s` consists entirely of hexadecimal characters (`0–9`, `a–f`, `A–F`); returns `false` for the empty string |
| `IsValidFullSHA` | `func(s string) bool` | Returns `true` if `s` is a valid 40-character lowercase hexadecimal SHA (matches `^[0-9a-f]{40}$`) |
| `ExtractBaseRepo` | `func(repoPath string) string` | Extracts the `owner/repo` portion from an action path that may include a sub-folder (e.g. `github/codeql-action/upload-sarif` → `github/codeql-action`) |
| `FindGitRoot` | `func() (string, error)` | Returns the absolute path of the root directory of the current Git repository using pure Go filesystem traversal (no `git` subprocess); starts from the current working directory |
| `FindGitRootFrom` | `func(startDir string) (string, error)` | Like `FindGitRoot` but starts from `startDir`; traverses upward looking for a `.git` directory or worktree marker file |
| `ReadFileFromHEAD` | `func(filePath, gitRoot string) (string, error)` | Reads a file's content from the `HEAD` commit via `git show`; rejects paths that escape the repository; requires `git` on `PATH` |

**Behavioral contracts**:

- `IsRateLimitError` and `IsAuthError` MUST perform case-insensitive string matching.
- `IsAuthError` MUST return `true` for messages containing any of: `gh_token`, `github_token`, `authentication`, `not logged into`, `unauthorized`, `forbidden`, `permission denied`, or `saml enforcement`.
- `IsHexString` MUST return `false` for the empty string.
- `IsValidFullSHA` MUST require exactly 40 lowercase hexadecimal characters; mixed-case or shorter strings MUST return `false`.
- `FindGitRoot` and `FindGitRootFrom` MUST return `ErrNotGitRepository` (not a wrapped error) when the filesystem root is reached without finding a `.git` entry.
- `FindGitRootFrom` MUST accept both `.git` directories (normal repositories) and `.git` files whose content begins with `gitdir:` (worktrees and submodules).
- `ReadFileFromHEAD` MUST return an error when `gitRoot` is empty.
- `ReadFileFromHEAD` MUST return an error when `filePath` resolves to a path outside `gitRoot`.

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
if errors.Is(err, gitutil.ErrNotGitRepository) {
    return fmt.Errorf("must be run inside a git repository")
} else if err != nil {
    return fmt.Errorf("failed to find git root: %w", err)
}

// Find the git root starting from a specific directory
root, err = gitutil.FindGitRootFrom("/some/subdir")
if err != nil {
    return fmt.Errorf("not in a git repository: %w", err)
}

// Read a file from the HEAD commit (prefer absolute paths under root)
content, err := gitutil.ReadFileFromHEAD(filepath.Join(root, "go.mod"), root)
```

## Thread Safety

All exported functions are safe for concurrent use. The error-classification functions (`IsRateLimitError`, `IsAuthError`) and SHA-validation functions (`IsHexString`, `IsValidFullSHA`) are pure functions with no shared state. `FindGitRoot` and `FindGitRootFrom` read only the filesystem and the process working directory. `ReadFileFromHEAD` spawns a `git` subprocess per call with no shared state.

## Dependencies

**Internal**:
- `github.com/github/gh-aw/pkg/logger` — debug logging

**External** (runtime):
- `git` executable on `PATH` — required only by `ReadFileFromHEAD`

## Design Decisions

- `FindGitRoot` and `FindGitRootFrom` use pure Go filesystem traversal (walking up the directory tree looking for `.git`), avoiding the need for a `git` executable on `PATH`. This is important for Rosetta 2 compatibility on macOS ARM64 and restricted environments where `git` may not be available.
- `FindGitRootFrom` verifies worktree marker file content (must begin with `gitdir:`) in addition to existence, guarding against false positives from unrelated files named `.git`.
- `ReadFileFromHEAD` requires `git` on `PATH` because reading object data from a bare `git show` invocation is more reliable than re-implementing pack-file parsing in pure Go.

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
