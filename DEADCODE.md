# Dead Code Removal Plan

## Methodology

Dead code is identified using:
```bash
deadcode ./cmd/... 2>/dev/null
```

The tool reports unreachable functions/methods from the main entry points in `cmd/`.
It does NOT report unreachable constants, variables, or types — only functions.

**Important rules:**
- Run `go build ./...` after every batch
- Run `go vet ./...` to catch test compilation errors (cheaper than `go test`)
- Run `go test -tags=integration ./pkg/affected/...` to spot-check
- Always check if a "fully dead" file contains live constants/vars before deleting
- Check if any `internal/tools/` programs reference deleted CLI functions
- The deadcode list was generated before any deletions; re-run after major batches

**Analysis date:** 2026-02-28  
**Total dead entries:** 381  
**Fully dead files:** 27 (delete entire file)  
**Partially dead files:** ~100 (remove individual functions)

---

## Phase 1: Fully Dead Files (27 files)

These files have ALL their functions dead. Each must be checked for:
- [ ] Live constants, variables, or types used elsewhere
- [ ] Test files that reference the deleted functions
- [ ] `internal/tools/` dependencies

### Group 1A: CLI fully dead files (6 files)
- [ ] `pkg/cli/actions_build_command.go` (9/9 dead) → also delete `internal/tools/actions-build/`
- [ ] `pkg/cli/exec.go` (4/4 dead)
- [ ] `pkg/cli/generate_action_metadata_command.go` (9/9 dead) → also delete `internal/tools/generate-action-metadata/`
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
- [ ] `pkg/workflow/compiler_string_api.go` (2/2 dead) → delete `compiler_string_api_test.go`
- [ ] `pkg/workflow/compiler_test_helpers.go` (3/3 dead) — test helper, check usage
- [ ] `pkg/workflow/copilot_participant_steps.go` (3/3 dead)
- [ ] `pkg/workflow/dependency_tracker.go` (2/2 dead)
- [ ] `pkg/workflow/env_mirror.go` (2/2 dead)
- [ ] `pkg/workflow/markdown_unfencing.go` (1/1 dead)
- [ ] `pkg/workflow/prompt_step.go` (2/2 dead) — **CAUTION: may be referenced by tests**
- [ ] `pkg/workflow/safe_output_builder.go` (10/10 dead) — **CAUTION: contains live type `ListJobBuilderConfig`**
- [ ] `pkg/workflow/sh.go` (5/5 dead) — **CAUTION: contains live constants (prompts dir, file names) and embed directive**

---

## Phase 2: Near-Fully Dead Files (high value, some surgery)

These files are mostly dead and worth cleaning next:

- [ ] `pkg/workflow/script_registry.go` (11/13 dead) — keep only `GetActionPath`, `DefaultScriptRegistry`
- [ ] `pkg/workflow/artifact_manager.go` (14/16 dead) — remove 14 functions  
- [ ] `pkg/constants/constants.go` (13/27 dead) — remove 13 constants
- [ ] `pkg/workflow/map_helpers.go` (5/7 dead) — remove 5 functions
- [ ] `pkg/workflow/js.go` (17/47 dead) — remove 17 JS bundle functions
- [ ] `pkg/workflow/compiler_types.go` (17/45 dead) — remove 17 types/methods

---

## Phase 3: Partially Dead Files (1–6 dead per file)

Individual function removals across ~100 files. To be tackled after Phase 1 and 2.

High-count files to prioritize:
- `pkg/workflow/expression_builder.go` (9/27 dead)
- `pkg/workflow/validation_helpers.go` (6/10 dead)
- `pkg/cli/docker_images.go` (6/11 dead)
- `pkg/workflow/domains.go` (10/27 dead)

---

## Batch Execution Log

### Batch 1: Group 1A (CLI fully dead) — TODO
### Batch 2: Group 1B (Console fully dead) + 1C (Misc utilities) — TODO
### Batch 3: Groups 1D + 1E (Workflow fully dead) — TODO
### Batch 4: Phase 2 (Near-fully dead) — TODO
### Batch 5: Phase 3 (Partial removals) — TODO

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
| `pkg/cli/actions_build_command_test.go` | Tests deleted CLI commands |
| `pkg/cli/exec_test.go` | Tests deleted exec functions |
| `pkg/cli/generate_action_metadata_command_test.go` | Tests deleted command |
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
