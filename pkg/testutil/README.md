# testutil Package

The `testutil` package provides shared test helpers for isolating test artifacts and capturing output.

## Overview

This package is imported only in test files (`_test.go`). It provides:
- A shared, isolated temporary directory for each test run (outside the git repository).
- Per-test subdirectories that are cleaned up automatically.
- Helpers for capturing `os.Stderr` output during tests.
- A helper for stripping YAML comment headers from compiled workflow output.

## Functions

### `GetTestRunDir() string`

Returns the path to the unique top-level directory for the current test run. It is created once per process under `$TMPDIR/gh-aw-test-runs/<timestamp>-<pid>`. Using a directory outside the repository prevents `git` commands from interfering with test artifacts.

```go
dir := testutil.GetTestRunDir()
// e.g. /tmp/gh-aw-test-runs/20240101-120000-12345
```

### `TempDir(t *testing.T, pattern string) string`

Creates a temporary subdirectory inside the test run directory matching `pattern`. The directory is automatically removed when the test completes via `t.Cleanup`.

```go
func TestCompile(t *testing.T) {
    dir := testutil.TempDir(t, "compile-*")
    // Use dir for test artifacts; cleaned up automatically
}
```

### `CaptureStderr(t *testing.T, fn func()) string`

Runs `fn` and returns everything written to `os.Stderr` during its execution. `os.Stderr` is restored automatically via `t.Cleanup`.

```go
func TestWarningMessage(t *testing.T) {
    output := testutil.CaptureStderr(t, func() {
        myFunction() // writes to os.Stderr
    })
    assert.Contains(t, output, "expected warning")
}
```

### `StripYAMLCommentHeader(yamlContent string) string`

Removes the leading comment block from a generated YAML file and returns only the non-comment content. Useful for tests that need to verify compiled output without matching the auto-generated header.

```go
raw, _ := os.ReadFile("workflow.lock.yml")
yaml := testutil.StripYAMLCommentHeader(string(raw))
assert.Contains(t, yaml, "runs-on: ubuntu-latest")
```

## Design Notes

- `GetTestRunDir` uses `sync.Once` so the directory is created exactly once per process even when multiple test packages run concurrently.
- `TempDir` delegates to `os.MkdirTemp` to generate unique subdirectory names.
- Test artifacts placed in the test run directory are outside any git repository, which prevents `git` commands executed by tests from picking them up as untracked files.

---

*This specification is automatically maintained by the [spec-extractor](../../.github/workflows/spec-extractor.md) workflow.*
