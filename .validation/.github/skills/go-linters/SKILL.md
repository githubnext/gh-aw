---
name: go-linters
description: Add and validate custom Go analysis linters in gh-aw.
---

# Go Linters

Use this guide when adding a new custom Go analysis linter in this repository.

For PR-driven linter generation (derive a rule from a specific pull request pattern), use `.github/skills/pr-to-go-linter/SKILL.md`.

## Where to add a new linter

1. Create a new package under `pkg/linters/<linter-name>/`.
2. Define an analyzer in that package (exported as `Analyzer`).
3. Add tests in the same package using `analysistest` with fixtures under `testdata/src/...`.
4. Register the analyzer in `cmd/linters/main.go` so it runs via the multichecker binary.

## Build and test linters

- Test only your linter package:
  - `go test ./pkg/linters/<linter-name>/...`
- Build the custom linter runner:
  - `go build ./cmd/linters`
- Run all custom linters across the repo:
  - `make golint-custom`

`make golint-custom` builds `cmd/linters` and runs it against `./cmd/...` and `./pkg/...`.
