# Dead Code Removal Guide

## How to find dead code

```bash
deadcode ./cmd/... ./internal/tools/... 2>/dev/null
```

**Critical:** Always include `./internal/tools/...` — it covers separate binaries called by the Makefile (e.g. `make actions-build`). Running `./cmd/...` alone gives false positives.

## Verification after every batch

```bash
go build ./...
go vet ./...
go vet -tags=integration ./...   # catches integration test files invisible without the tag
make fmt
```

## Known pitfalls

**WASM binary** — `cmd/gh-aw-wasm/main.go` has `//go:build js && wasm` so deadcode cannot analyse it. Before deleting anything from `pkg/workflow/`, check that file. Currently uses:
- `compiler.ParseWorkflowString`
- `compiler.CompileToYAML`

**Test helpers** — `pkg/workflow/compiler_test_helpers.go` shows 3 dead functions but is used by 15 test files. Don't delete it.

**Constant/embed rescue** — Some otherwise-dead files contain live constants or `//go:embed` directives. Extract them before deleting the file.

---

## Current dead code (276 functions, as of 2026-02-28)

Run the command above to regenerate. Top files by dead function count:

| File | Dead | Notes |
|------|------|-------|
| `pkg/workflow/js.go` | 17 | Get*/bundle stubs; many have no callers anywhere |
| `pkg/workflow/compiler_types.go` | 17 | `With*` option funcs + getters; check WASM first |
| `pkg/workflow/artifact_manager.go` | 14 | Many test callers; do last |
| `pkg/constants/constants.go` | 13 | All `String()`/`IsValid()` on semantic type aliases |
| `pkg/workflow/domains.go` | 10 | Check callers |
| `pkg/workflow/expression_builder.go` | 9 | Check callers |
| `pkg/workflow/validation_helpers.go` | 6 | Check callers |
| `pkg/cli/docker_images.go` | 6 | Check callers |
| `pkg/workflow/permissions_factory.go` | 5 | Check callers |
| `pkg/workflow/map_helpers.go` | 5 | Check callers |
| `pkg/workflow/engine_helpers.go` | 5 | Check callers |
| `pkg/console/console.go` | 5 | Check callers |
| `pkg/workflow/safe_outputs_env.go` | 4 | Check callers |
| `pkg/workflow/expression_nodes.go` | 4 | Check callers |

~80 additional files have 1–3 dead functions each.

## Suggested approach

1. Pick a file with 5+ dead functions.
2. For each dead function, check callers: `grep -rn "FuncName" --include="*.go"`. If only test callers, also remove the tests.
3. Remove the function and any now-unused imports.
4. Run the verification commands above.
5. Commit per logical group, keep PRs small and reviewable.
