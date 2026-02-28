# Dead Code Removal Plan

## ⚠️ Critical Lesson Learned (Session 1 failure)

Running `deadcode ./cmd/...` only analyses the main binary entry points in `cmd/`.
It is **blind to `internal/tools/` programs**, which are separate binaries with their own `main()` functions, called by the Makefile and CI.

In Session 1 we deleted `pkg/cli/actions_build_command.go` and `internal/tools/actions-build/` because `deadcode ./cmd/...` reported them as unreachable — but they are actively used by `make actions-build` / `make actions-validate` in CI.

**Correct command:**
```bash
deadcode ./cmd/... ./internal/tools/... 2>/dev/null
```

This covers all entry points. The 381-entry list from Session 1 is **invalid** — regenerate it.

---

## Methodology

Dead code is identified using:
```bash
deadcode ./cmd/... ./internal/tools/... 2>/dev/null
```

The tool reports unreachable functions/methods from ALL entry points (`cmd/` + `internal/tools/`).
It does NOT report unreachable constants, variables, or types — only functions.

**Important rules:**
- **Always include `./internal/tools/...` in the deadcode command**
- **Beware `//go:build js && wasm` files** — `cmd/gh-aw-wasm/` uses functions like `ParseWorkflowString` and `CompileToYAML` that deadcode can't see because the WASM binary can't be compiled without `GOOS=js GOARCH=wasm`. Always check `cmd/gh-aw-wasm/main.go` before deleting functions from `pkg/workflow/`.
- Run `go build ./...` after every batch
- Run `go vet ./...` **AND** `go vet -tags=integration ./...` to catch unit AND integration test errors
- Run `go test -tags=integration ./pkg/affected/...` to spot-check
- Always check if a "fully dead" file contains live constants/vars before deleting
- The deadcode list was generated before any deletions; re-run after major batches

---

## ⚠️ Status: Plan needs regeneration

The phases and batches below were based on the **incorrect** `./cmd/...`-only scan.
Before proceeding, reset to main and regenerate the list with:

```bash
deadcode ./cmd/... ./internal/tools/... 2>/dev/null | tee /tmp/deadcode-correct.txt | wc -l
```

The groups below are a rough guide but individual entries may differ.

---

## Session 2 Analysis (2026-02-28)

**Command:** `deadcode ./cmd/... ./internal/tools/... 2>/dev/null`  
**Total dead entries:** 362  
**Fully dead files:** 25  
**Partially dead files:** 117  

Confirmed NOT dead (correctly excluded now): `pkg/cli/actions_build_command.go`, `pkg/cli/generate_action_metadata_command.go`

---

## Phase 1: Fully Dead Files

These files have ALL their functions dead. Each must be checked for:
- [ ] Live constants, variables, or types used elsewhere
- [ ] Test files that reference the deleted functions
- [ ] `internal/tools/` dependencies

### Group 1A: CLI fully dead files (4 files)
- [ ] `pkg/cli/exec.go` (4/4 dead)
- [ ] `pkg/cli/logs_display.go` (1/1 dead) → surgery on `logs_overview_test.go`
- [ ] `pkg/cli/mcp_inspect_safe_inputs_inspector.go` (1/1 dead) → delete `mcp_inspect_safe_inputs_test.go`
- [ ] `pkg/cli/validation_output.go` (2/2 dead)

### Group 1B: Console fully dead files (3 files)
- [ ] `pkg/console/form.go` (1/1 dead) → delete `form_test.go`
- [ ] `pkg/console/layout.go` (4/4 dead) → surgery on `golden_test.go`
- [ ] `pkg/console/select.go` (2/2 dead)

### Group 1C: Misc utility fully dead files (4 files)
- [ ] `pkg/logger/error_formatting.go` (1/1 dead)
- [ ] `pkg/parser/ansi_strip.go` (1/1 dead) → surgery on frontmatter tests
- [ ] `pkg/parser/virtual_fs_test_helpers.go` (1/1 dead, test helper only)
- [ ] `pkg/stringutil/paths.go` (1/1 dead) → delete `paths_test.go`

### Group 1D: Workflow bundler fully dead files (5 files)
These are the JS bundler subsystem — entirely unused.
- [ ] `pkg/workflow/bundler.go` (6/6 dead) → delete 14+ bundler test files
- [ ] `pkg/workflow/bundler_file_mode.go` (12/12 dead) — **CAUTION: contains live const `SetupActionDestination`**
- [ ] `pkg/workflow/bundler_runtime_validation.go` (3/3 dead)
- [ ] `pkg/workflow/bundler_safety_validation.go` (3/3 dead)
- [ ] `pkg/workflow/bundler_script_validation.go` (2/2 dead)

### Group 1E: Workflow other fully dead files (9 files)
- [x] `pkg/workflow/compiler_string_api.go` ~~(2/2 dead) → delete~~ **⚠️ DO NOT DELETE — used by `cmd/gh-aw-wasm/` (WASM binary has `//go:build js && wasm` constraint invisible to deadcode)**
- [x] `pkg/workflow/compiler_test_helpers.go` (3/3 dead) — test helper, **DO NOT DELETE** (used by 15 test files)
- [ ] `pkg/workflow/copilot_participant_steps.go` (3/3 dead)
- [ ] `pkg/workflow/dependency_tracker.go` (2/2 dead)
- [ ] `pkg/workflow/env_mirror.go` (2/2 dead)
- [ ] `pkg/workflow/markdown_unfencing.go` (1/1 dead)
- [ ] `pkg/workflow/prompt_step.go` (2/2 dead) — **CAUTION: may be referenced by tests**
- [ ] `pkg/workflow/safe_output_builder.go` (10/10 dead) — **CAUTION: contains live type `ListJobBuilderConfig`**
- [ ] `pkg/workflow/sh.go` (5/5 dead) — **CAUTION: contains live constants (prompts dir, file names) and embed directive**

---

## Phase 2: Near-Fully Dead Files (high value, some surgery)

- [x] `pkg/workflow/script_registry.go` — rewritten minimal in batch 2 ✅
- [x] `pkg/workflow/compiler_types.go` — 7 dead `With*` option funcs + 3 getters removed in batch 3; **10 dead remain** (see batch 4)
- [x] `pkg/workflow/js.go` — 10 dead bundle/Get* funcs removed in batch 3; **7 dead remain** (see batch 4)
- [ ] `pkg/workflow/artifact_manager.go` — **14 dead** — but tests call many of these; skip or do last
- [ ] `pkg/constants/constants.go` — **13 dead** (all `String()`/`IsValid()` methods on type aliases) — safe to remove
- [ ] `pkg/workflow/map_helpers.go` — **5 dead** — check test callers before removing

---

## Phase 3 / Batch 4 Targets (current dead count: 259)

Remaining high-value clusters from `deadcode ./cmd/... ./internal/tools/...`:

| File | Dead | Notes |
|------|------|-------|
| `pkg/workflow/artifact_manager.go` | 14 | Many test callers; do last |
| `pkg/constants/constants.go` | 13 | All `String()`/`IsValid()` on semantic types; safe |
| `pkg/workflow/domains.go` | 10 | Check callers |
| `pkg/workflow/compiler_types.go` | 10 | Remaining With*/Get* |
| `pkg/workflow/expression_builder.go` | 9 | Check callers |
| `pkg/workflow/js.go` | 7 | Remaining Get* stubs |
| `pkg/workflow/validation_helpers.go` | 6 | Check callers |
| `pkg/cli/docker_images.go` | 6 | Check callers |
| `pkg/workflow/permissions_factory.go` | 5 | Check callers |
| `pkg/workflow/map_helpers.go` | 5 | Check callers |
| `pkg/workflow/engine_helpers.go` | 5 | Check callers |
| `pkg/console/console.go` | 5 | Check callers |
| `pkg/workflow/safe_outputs_env.go` | 4 | Check callers |
| `pkg/workflow/expression_nodes.go` | 4 | Check callers |

**Long tail:** ~80 remaining files with 1–3 dead functions each.

---

## Batch Execution Log

### Session 1 — ABORTED (incorrect deadcode command)

Used `deadcode ./cmd/...` — missed `internal/tools/` entry points. Deleted:
- `pkg/cli/actions_build_command.go` — **WRONG: used by `make actions-build` via `internal/tools/actions-build/`**
- `pkg/cli/exec.go`, `pkg/cli/generate_action_metadata_command.go`, etc.
- `internal/tools/actions-build/`, `internal/tools/generate-action-metadata/`
- CI job `actions-build` from `.github/workflows/ci.yml`

PR #18782 failed CI with `make: *** No rule to make target 'actions-build'`. Reset to main.

### Session 2 — In Progress

#### Batch 1: Groups 1A (CLI) + 1B (Console) + 1C (Misc) — COMPLETE ✅

Deleted 17 files, surgery on 6 test files. `go build ./...` + `go vet ./...` + `make fmt` all clean.

Deferred `pkg/stringutil/paths.go` to Batch 2 — callers in bundler files still present.

#### Batch 2: Groups 1D + 1E (Workflow fully dead) — COMPLETE ✅

Deleted 35 files (bundler subsystem + env_mirror, copilot_participant_steps, dependency_tracker,
markdown_unfencing, prompt_step, safe_output_builder, sh.go, stringutil/paths.go).
Rescued: `prompt_constants.go`, `setup_action_paths.go`. Rewrote `script_registry.go` minimal.
Surgery on 12 test files. ~7,856 lines deleted.

⚠️ **Lessons learned in batch 2:**
- `go vet ./...` misses integration tests — MUST also run `go vet -tags=integration ./...`
- `cmd/gh-aw-wasm/` has `//go:build js && wasm` — deadcode can't see it; `compiler_string_api.go` was wrongly deleted and restored
- Always check `cmd/gh-aw-wasm/main.go` before deleting `pkg/workflow` functions

#### Batch 3: Phase 2 partial (compiler_types + js.go) — COMPLETE ✅

Removed 7 dead `With*` option funcs + 3 dead getters from `compiler_types.go`.
Removed 10 dead Get*/bundle funcs from `js.go`.
~133 lines deleted. Dead count: 362 → 259.

#### Batch 4: Remaining Phase 2 + Phase 3 (individual removals) — TODO

---

## Key Constant/Var Dependencies (must rescue before deleting)

These live values are defined in files that are otherwise fully dead:

| Const/Var | Used by live code | Currently in |
|-----------|-------------------|--------------|
| `SetupActionDestination` | `safe_outputs_steps.go` etc. | `bundler_file_mode.go` |
| `cacheMemoryPromptFile` | `cache.go` | `sh.go` |
| `cacheMemoryPromptMultiFile` | `cache.go` | `sh.go` |
| `promptsDir` | `unified_prompt_step.go`, `repo_memory_prompt.go` | `sh.go` |
| `prContextPromptFile` | `unified_prompt_step.go` | `sh.go` |
| `tempFolderPromptFile` | `unified_prompt_step.go` | `sh.go` |
| `playwrightPromptFile` | `unified_prompt_step.go` | `sh.go` |
| `markdownPromptFile` | `unified_prompt_step.go` | `sh.go` |
| `xpiaPromptFile` | `unified_prompt_step.go` | `sh.go` |
| `repoMemoryPromptFile` | `repo_memory_prompt.go` | `sh.go` |
| `repoMemoryPromptMultiFile` | `repo_memory_prompt.go` | `sh.go` |
| `safeOutputsPromptFile` | `unified_prompt_step.go` | `sh.go` |
| `safeOutputsCreatePRFile` | `unified_prompt_step.go` | `sh.go` |
| `safeOutputsPushToBranchFile` | `unified_prompt_step.go` | `sh.go` |
| `safeOutputsAutoCreateIssueFile` | `unified_prompt_step.go` | `sh.go` |
| `githubContextPromptText` (embed) | `unified_prompt_step.go` | `sh.go` |
| `ListJobBuilderConfig` type | `add_labels.go` (dead), `safe_output_builder.go` (dead) | `safe_output_builder.go` |

**Strategy:** Create `pkg/workflow/workflow_constants.go` to hold rescued constants and embed.
`ListJobBuilderConfig` is only used by dead code, so needs no rescue.

---

## Test Files to Delete (when their entire subject is deleted)

| Test file | Reason to delete |
|-----------|-----------------|
| `pkg/cli/exec_test.go` | Tests deleted exec functions |
| `pkg/cli/validation_output_test.go` | Tests deleted functions |
| `pkg/cli/mcp_inspect_safe_inputs_test.go` | References `spawnSafeInputsInspector` (deleted) |
| `pkg/console/form_test.go` | Tests deleted `RunForm` |
| `pkg/stringutil/paths_test.go` | Tests deleted `NormalizePath` |
| `pkg/workflow/compiler_string_api_test.go` | Tests deleted `ParseWorkflowString` |
| `pkg/workflow/script_registry_test.go` | Tests dead registry methods |
| All `pkg/workflow/bundler_*_test.go` | Tests deleted bundler |

## Test Files Needing Surgery

| Test file | What to remove |
|-----------|---------------|
| `pkg/cli/logs_overview_test.go` | Remove 4 tests using deleted `DisplayLogsOverview` |
| `pkg/console/golden_test.go` | Remove tests using deleted `LayoutTitleBox` |
| `pkg/parser/frontmatter_utils_test.go` | Remove `TestStripANSI`, `BenchmarkStripANSI` |
| `pkg/parser/frontmatter_merge_test.go` | Remove stray comment |
| `pkg/workflow/compiler_custom_actions_test.go` | Remove tests using dead registry methods |
| `pkg/workflow/compiler_action_mode_test.go` | Remove tests using dead registry methods |
| `pkg/workflow/custom_action_copilot_token_test.go` | Remove test using `RegisterWithAction` |

---

## PR Strategy

**PR 1:** Phase 1 Groups 1A + 1B + 1C (CLI, console, misc utilities — no workflow risk)
- 13 files deleted
- Clean, low-risk, easy to review

**PR 2:** Phase 1 Groups 1D + 1E (bundler + workflow dead files)
- 14 files deleted
- More complex due to constant rescue and test surgery

**PR 3:** Phase 2 (near-fully dead)

**PR 4:** Phase 3 (individual function removals, many files)
