---
name: pr-to-go-linter
description: Generate a new pkg/linters analyzer from a merged or open PR pattern.
---

# PR to Go Linter

Use this skill when a user asks to create a new custom Go linter based on a code pattern fixed in a pull request.

## Goal

Convert one concrete PR pattern into a new `go/analysis` linter under `pkg/linters/<name>/` with tests and runner registration.

## Inputs

- Repository owner/name
- Pull request number
- Target linter name (kebab-case)

## Workflow

1. Read PR metadata and changed files.
2. Read the PR diff and extract the repeated pattern that was fixed.
3. Define one precise diagnostic rule from that pattern.
4. Confirm no existing linter in `pkg/linters/` already covers it.
5. Implement:
   - `pkg/linters/<name>/<name>.go` with exported `Analyzer`
   - `pkg/linters/<name>/<name>_test.go` using `analysistest`
   - `pkg/linters/<name>/testdata/src/<name>/<name>.go` fixtures with `// want`
   - `cmd/linters/main.go` registration in `multichecker.Main(...)`
6. Validate:
   - `go test ./pkg/linters/<name>/...`
   - `go build ./cmd/linters`
   - `make golint-custom`

## Rule quality checks

- High signal, low false positives on this repository.
- Diagnostic is specific and fixable.
- Rule scope matches code in the PR (do not generalize beyond evidence).
- Do not change unrelated linter packages.

## Example pattern source

For PR `#33038` (`Refactor pkg mutex sites to use deferred unlocks consistently`), derive a linter idea that reports lock/unlock sections that manually unlock instead of deferring unlock immediately after lock when the function body matches the same cache/logger-style critical section pattern.

## Output expectations

- Minimal implementation-only diff in `pkg/linters/<name>/` and `cmd/linters/main.go`.
- Tests prove both flagged and non-flagged cases.
- PR summary explains: source PR, extracted pattern, and why the rule is safe.
