---
name: go-codemod
description: Implement and test Go codemods for the gh aw fix command.
---

# Go Codemod Implementation Guide

Use this skill when adding or updating codemods used by `gh aw fix`.

## Understand the fix pipeline first

1. Review `pkg/cli/fix_command.go` to understand execution flow:
   - codemods are loaded once via `GetAllCodemods()`
   - each codemod is applied in registry order
   - frontmatter is re-parsed before each codemod
   - codemods must return `(newContent, applied, error)` and be safe for no-op input
2. Review the codemod registry in `pkg/cli/fix_codemods.go`.
3. Review helper utilities in `pkg/cli/yaml_frontmatter_utils.go` and reusable codemod helper constructors in `pkg/cli/codemod_factory.go`.

## Implementation steps

1. Create a new codemod file in `pkg/cli/` named `codemod_<feature>.go`.
2. Add a package logger with `logger.New("cli:codemod_<feature>")`.
3. Implement `get<Feature>Codemod() Codemod` and populate all metadata fields:
   - `ID` (stable, unique)
   - `Name` (human-readable)
   - `Description` (clear migration behavior)
   - `IntroducedIn` (release version)
4. In `Apply`:
   - check frontmatter preconditions first
   - return unchanged content and `applied=false` when migration is not needed
   - transform only frontmatter using `applyFrontmatterLineTransform` from `pkg/cli/yaml_frontmatter_utils.go`
   - preserve comments/formatting/markdown body
   - avoid lossy rewrites and avoid touching unrelated keys
5. Prefer existing helpers before writing custom parsing logic:
   - `findAndReplaceInLine`
   - `removeFieldFromBlock`
   - `removeParentBlockIfTrulyEmpty`
   - `newFieldRemovalCodemod`
   - `newMoveTopLevelKeyToOnBlockCodemod`
6. If the codemod depends on external data or side effects, inject dependencies through a `...WithDeps` constructor so tests can mock behavior.
7. Register the codemod in `pkg/cli/fix_codemods.go` within `GetAllCodemods()`.

## Testing requirements

Create `pkg/cli/codemod_<feature>_test.go` and cover:

1. Metadata correctness (`ID`, `Name`, `Description`, `IntroducedIn`, `Apply != nil`).
2. Happy path migration with expected output.
3. No-op behavior when deprecated input is absent.
4. Idempotence behavior (already-migrated input remains unchanged).
5. Preservation guarantees:
   - inline comments
   - indentation
   - markdown body after frontmatter
6. Edge cases specific to the codemod (nested fields, mixed forms, ordering constraints, strict-mode checks, etc.).
7. Dependency-injection behavior if `...WithDeps` exists (success and fallback/error paths).

When complexity is high (expression rewriting, parser-like behavior), add fuzz tests in `codemod_<feature>_fuzz_test.go` using the pattern in `pkg/cli/codemod_steps_run_secrets_env_fuzz_test.go`.

## Registry/order tests

After adding a codemod, update `pkg/cli/fix_codemods_test.go`:

1. Expected codemod IDs list.
2. Expected codemod order list.

Order matters because codemods run sequentially and later codemods observe prior transformations.

## Validation commands

Run targeted checks first:

1. `go test -v ./pkg/cli -run Codemod -count=1`
2. `go test -v ./pkg/cli -run Fix -count=1`

Then run repository standards:

3. `make build`
4. `make test-unit`
5. `make lint`

## Quality bar

A codemod is ready only when it is:

- deterministic
- safe on repeated runs
- conservative (no unrelated rewrites)
- fully covered by tests for migration and no-op paths
- registered and order-verified in fix codemod registry tests
