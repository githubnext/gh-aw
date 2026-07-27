---
private: true
name: Daily Go Test Parallelizer
description: Adds t.Parallel to safe Go tests using daily round-robin analysis
on:
  schedule: daily
  workflow_dispatch:
  skip-if-match: 'is:pr is:open in:title "[test-parallel]"'
permissions:
  contents: read
  pull-requests: read
strict: true
timeout-minutes: 30
network:
  allowed:
    - defaults
    - go
sandbox:
  agent:
    sudo: false
tools:
  cache-memory:
    retention-days: 30
    allowed-extensions: [".json"]
  edit:
  bash:
    - "*"
safe-outputs:
  create-pull-request:
    title-prefix: "[test-parallel] "
    labels: [automation, testing]
    draft: true
    expires: 3d
    if-no-changes: ignore
    protected-files: blocked
    allowed-files:
      - "**/*_test.go"
    max-patch-files: 1
    max-patch-size: 2048
  noop:
---

# Daily Go Test Parallelizer

Analyze one Go test file per run and add `t.Parallel()` only where parallel execution is demonstrably safe.

## Select a file

1. Use `grep` to list tracked `*_test.go` files containing top-level `Test` functions. Exclude `vendor/` and generated files, then sort paths lexicographically.
2. Read `/tmp/gh-aw/cache-memory/go-test-parallelizer/state.json` when it exists. It has this shape:
   `{"last_file":"path/to/file_test.go"}`.
3. Select the path after `last_file`, wrapping to the first path. If the cache is absent, malformed, or names a removed file, select the first path.
4. Analyze and modify at most that one file.

## Analyze safety

Add `t.Parallel()` at the start of eligible top-level tests. Also add it to eligible table-driven subtests when loop variables are safely captured.

Do not parallelize tests that use or may conflict through:

- `t.Setenv`, `os.Setenv`, `os.Chdir`, or other process-wide state
- shared mutable globals, singleton state, ordering assumptions, or package-level mocks
- fixed ports, fixed filesystem paths, shared databases, or shared external services
- unsafe loop-variable capture, timing dependencies, or explicit synchronization between tests

Do not change assertions, test behavior, production code, dependencies, generated files, or vendored files. When safety is uncertain, make no change.

## Validate

After editing:

1. Run the selected package with the race detector.
2. Run `go test ./...`.
3. Inspect the diff and confirm it contains only safe `t.Parallel()` additions in the selected file.
4. Revert the edit and use `noop` if either test command fails or the diff contains any other change.

## Persist and report

Always create `/tmp/gh-aw/cache-memory/go-test-parallelizer/` and write the selected path to `state.json`, even when no edit is safe, so the next daily run advances round-robin.

If validation succeeds with a change, create one draft pull request describing the safety analysis and test results. Otherwise use `noop` with the selected path and a short reason.
