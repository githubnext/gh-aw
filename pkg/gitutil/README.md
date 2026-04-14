# gitutil Package

The `gitutil` package provides utility functions for interacting with Git repositories and classifying GitHub API errors.

## Overview

This package contains helpers for:
- Detecting rate-limit and authentication errors from GitHub API responses.
- Validating hex strings (e.g. commit SHAs).
- Extracting base repository slugs from action paths.
- Finding the root directory of the current Git repository.
- Reading file contents from the `HEAD` commit.

## Functions

### Error Classification

#### `IsRateLimitError(errMsg string) bool`

Returns `true` when `errMsg` indicates a GitHub API rate-limit error (HTTP 403 "API rate limit exceeded" or HTTP 429).

```go
if gitutil.IsRateLimitError(err.Error()) {
    // Back off and retry
}
```

#### `IsAuthError(errMsg string) bool`

Returns `true` when `errMsg` indicates an authentication or authorization failure (`GH_TOKEN`, `GITHUB_TOKEN`, `unauthorized`, `forbidden`, SAML enforcement, etc.).

```go
if gitutil.IsAuthError(err.Error()) {
    fmt.Fprintln(os.Stderr, "Check that GH_TOKEN is set correctly")
}
```

### String Utilities

#### `IsHexString(s string) bool`

Returns `true` if `s` consists entirely of hexadecimal characters (`0–9`, `a–f`, `A–F`). Returns `false` for the empty string.

```go
if gitutil.IsHexString(sha) {
    // Valid commit SHA
}
```

#### `ExtractBaseRepo(repoPath string) string`

Extracts the `owner/repo` portion from an action path that may include a sub-folder.

```go
gitutil.ExtractBaseRepo("actions/checkout")                        // → "actions/checkout"
gitutil.ExtractBaseRepo("github/codeql-action/upload-sarif")      // → "github/codeql-action"
```

### Repository Operations

#### `FindGitRoot() (string, error)`

Returns the absolute path of the root directory of the current Git repository by running `git rev-parse --show-toplevel`. Returns an error if the working directory is not inside a Git repository.

```go
root, err := gitutil.FindGitRoot()
if err != nil {
    return fmt.Errorf("not in a git repository: %w", err)
}
```

#### `ReadFileFromHEADWithRoot(filePath, gitRoot string) (string, error)`

Reads a file's content from the `HEAD` commit without touching the working tree. `gitRoot` must be the repository root (typically from `FindGitRoot`). The function rejects paths that escape the repository (i.e. paths containing `..` after resolution).

```go
root, _ := gitutil.FindGitRoot()
content, err := gitutil.ReadFileFromHEADWithRoot("pkg/workflow/compiler.go", root)
```

## Design Notes

- All debug output uses `logger.New("gitutil:gitutil")` and is only emitted when `DEBUG=gitutil:*`.
- Error classification is case-insensitive string matching — no external dependency on GitHub API client types.
- `ReadFileFromHEADWithRoot` uses `git show HEAD:<relpath>` and resolves paths with `filepath.Rel` to prevent path-traversal attacks.
