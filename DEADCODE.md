# Dead Code Removal Guide

## How to find dead code

```bash
deadcode ./cmd/... ./internal/tools/... 2>/dev/null
```

**Critical:** Always include `./internal/tools/...` — it covers separate binaries called by the Makefile (e.g. `make actions-build`). Running `./cmd/...` alone gives false positives.

## Correct methodology

`deadcode` analyses the production binary entry points only. **Test files compile into a separate test binary** and do not keep production code alive. A function flagged by `deadcode` is dead regardless of whether test files call it.

**Correct approach:**
1. `deadcode` flags `Foo` as unreachable
2. `grep -rn "Foo" --include="*.go"` shows callers only in `*_test.go` files
3. **Delete `Foo` AND any test functions that exclusively test `Foo`**

**Wrong approach (batch 4 mistake):** treating test-only callers as evidence the function is "live" and skipping it.

**Exception — `compiler_test_helpers.go`:** the 3 functions there (`containsInNonCommentLines`, `indexInNonCommentLines`, `extractJobSection`) are production-file helpers used by ≥15 test files as shared test infrastructure. They're dead in the production binary but valuable as test utilities. Leave them.

## Verification after every batch

```bash
go build ./...
go vet ./...
go vet -tags=integration ./...   # catches integration test files invisible without this tag
make fmt
```

## Known pitfalls

**WASM binary** — `cmd/gh-aw-wasm/main.go` has `//go:build js && wasm` so deadcode cannot analyse it. Before deleting anything from `pkg/workflow/`, check that file. Currently uses:
- `compiler.ParseWorkflowString`
- `compiler.CompileToYAML`

**`pkg/console/console_wasm.go`** — this file provides WASM-specific stub implementations of many `pkg/console` functions (gated with `//go:build js || wasm`). Before deleting any function from `pkg/console/`, `grep` for it in `console_wasm.go`. If the function is called there, either inline the logic in `console_wasm.go` or delete the call. Batch 10 mistake: deleted `renderTreeSimple` from `render.go` but `console_wasm.go`'s `RenderTree` still called it, breaking the WASM build. Fix: replaced the `RenderTree` body in `console_wasm.go` with an inlined closure that no longer calls the deleted helper.

**`compiler_test_helpers.go`** — shows 3 dead functions but serves as shared test infrastructure for ≥15 test files. Do not delete.

**Constant/embed rescue** — Some otherwise-dead files contain live constants or `//go:embed` directives. Extract them before deleting the file.

---

## Batch plan (85 dead functions as of 2026-03-02, after phases 5–8)

Phases 5–8 are complete. The original phases 9–34 are superseded by this revised plan,
which groups work by domain, includes LOC estimates, and skips functions too small to
be worth the disruption.

Each phase: delete the dead functions, delete tests that exclusively test them,
run verification, commit, open PR. Branches are stacked: dc-9 based on dc-8, dc-10
based on dc-9, etc.

**WASM false positives (do not delete):**
- `Compiler.CompileToYAML` (`compiler_string_api.go:15`) — used by `cmd/gh-aw-wasm`
- `Compiler.ParseWorkflowString` (`compiler_string_api.go:52`) — used by `cmd/gh-aw-wasm`

**Shared test infrastructure (do not delete):**
- `containsInNonCommentLines`, `indexInNonCommentLines`, `extractJobSection` (`compiler_test_helpers.go`) — used by ≥15 test files

**Compiler option test infrastructure (do not delete):**
- `WithCustomOutput`, `WithVersion`, `WithSkipValidation`, `WithNoEmit`, `WithStrictMode`, `WithForceRefreshActionPins`, `WithWorkflowIdentifier` (`compiler_types.go`) — used at 50+ call sites in test files; removing would require massive test refactoring for minimal gain
- `NewCompilerWithVersion` (`compiler_types.go:160`) — primary test entry point for compiler construction
- `Compiler.GetSharedActionResolverForTest` (`compiler_types.go:305`) — test helper

**Not worth deleting (< 10 lines, isolated):**
`envVarPrefix` (2), `GenerateSafeInputGoToolScriptForInspector` (2), `MapToolConfig.GetAny` (3),
`HasErrors` (5), `extractMapFromFrontmatter` (5), `GetDefaultMaxForType` (5),
`GetValidationConfigForType` (6), `getPlaywrightMCPPackageVersion` (6), `RunGit` (6),
`ExecGHWithOutput` (7), `convertGoPatternToJavaScript` (7), `ValidateExpressionSafetyPublic` (7),
`IsSafeInputsHTTPMode` (7), `HasSafeJobsEnabled` (7), `renderSafeOutputsMCPConfig` (8),
`renderPlaywrightMCPConfig` (8), `SecurityFinding.String` (8), `GetAllEngines` (11)

---

### Phase 5 — CLI git helpers (3 functions)
File: `pkg/cli/git.go`

| Function | Line |
|----------|------|
| `getDefaultBranch` | 496 |
| `checkOnDefaultBranch` | 535 |
| `confirmPushOperation` | 569 |

Tests to check: `git_test.go` — remove `TestGetDefaultBranch`, `TestCheckOnDefaultBranch`, `TestConfirmPushOperation` if they exist.

### Phase 6 — parser frontmatter parsing & hashing (5 functions)
Files: `pkg/parser/frontmatter_content.go` (2), `pkg/parser/frontmatter_hash.go` (3)

| Function | File | Line |
|----------|------|------|
| `ExtractFrontmatterString` | `frontmatter_content.go` | 141 |
| `ExtractYamlChunk` | `frontmatter_content.go` | 181 |
| `ComputeFrontmatterHash` | `frontmatter_hash.go` | 50 |
| `buildCanonicalFrontmatter` | `frontmatter_hash.go` | 80 |
| `ComputeFrontmatterHashWithExpressions` | `frontmatter_hash.go` | 346 |

Tests to check: `frontmatter_content_test.go`, `frontmatter_hash_test.go`.

### Phase 7 — parser URL & schema helpers (4 functions)
Files: `pkg/parser/github_urls.go` (3), `pkg/parser/schema_compiler.go` (1)

| Function | File | Line |
|----------|------|------|
| `ParseRunURL` | `github_urls.go` | 316 |
| `GitHubURLComponents.GetRepoSlug` | `github_urls.go` | 422 |
| `GitHubURLComponents.GetWorkflowName` | `github_urls.go` | 427 |
| `GetMainWorkflowSchema` | `schema_compiler.go` | 382 |

Tests to check: `github_urls_test.go`, `schema_compiler_test.go`.

### Phase 8 — parser import system (4 functions)
Files: `pkg/parser/import_error.go` (2), `pkg/parser/import_processor.go` (2)

| Function | File | Line |
|----------|------|------|
| `ImportError.Error` | `import_error.go` | 30 |
| `ImportError.Unwrap` | `import_error.go` | 35 |
| `ProcessImportsFromFrontmatter` | `import_processor.go` | 78 |
| `ProcessImportsFromFrontmatterWithManifest` | `import_processor.go` | 90 |

Note: If `ImportError` struct has no remaining methods, consider deleting the type entirely.

Tests to check: `import_error_test.go`, `import_processor_test.go`.

### Phase 9 — Output job builders (~480 LOC, 6 functions)
Files: `pkg/workflow/add_comment.go`, `create_code_scanning_alert.go`, `create_discussion.go`, `create_pr_review_comment.go`, `create_agent_session.go`, `missing_issue_reporting.go`

Dead `buildCreateOutput*Job` methods — the primary job-assembly step for each safe-output type
is not reachable from any live code path.

| Function | File | LOC |
|----------|------|-----|
| `Compiler.buildCreateOutputAddCommentJob` | `add_comment.go` | 116 |
| `Compiler.buildCreateOutputDiscussionJob` | `create_discussion.go` | 78 |
| `Compiler.buildCreateOutputCodeScanningAlertJob` | `create_code_scanning_alert.go` | 73 |
| `Compiler.buildCreateOutputPullRequestReviewCommentJob` | `create_pr_review_comment.go` | 63 |
| `Compiler.buildIssueReportingJob` | `missing_issue_reporting.go` | 61 |
| `Compiler.buildCreateOutputAgentSessionJob` | `create_agent_session.go` | 53 |

Note: The `parse*Config` helpers in the same files are **live** (called from `safe_outputs_config.go`). Only delete the `buildCreateOutput*` methods.

Tests to check: no dedicated test files were found for these builders specifically.

### Phase 10 — Safe output compilation helpers (~250 LOC, 6 functions)
Files: `pkg/workflow/compiler_safe_outputs.go`, `compiler_safe_outputs_specialized.go`, `unified_prompt_step.go`, `missing_data.go`, `missing_tool.go`

| Function | File | LOC |
|----------|------|-----|
| `Compiler.generateUnifiedPromptStep` | `unified_prompt_step.go` | 118 |
| `Compiler.buildCreateProjectStepConfig` | `compiler_safe_outputs_specialized.go` | 41 |
| `Compiler.generateJobName` | `compiler_safe_outputs.go` | 34 |
| `Compiler.mergeSafeJobsFromIncludes` | `compiler_safe_outputs.go` | 24 |
| `Compiler.buildCreateOutputMissingDataJob` | `missing_data.go` | 17 |
| `Compiler.buildCreateOutputMissingToolJob` | `missing_tool.go` | 17 |

Note: `unified_prompt_step.go` may be entirely removable if `generateUnifiedPromptStep` is its only function.

### Phase 11 — Large compiler infrastructure helpers (~195 LOC, 3 functions)
Files: `pkg/workflow/compiler_yaml_helpers.go`, `repo_memory.go`, `jobs.go`

| Function | File | LOC |
|----------|------|-----|
| `generateRepoMemoryPushSteps` | `repo_memory.go` | 87 |
| `Compiler.generateCheckoutGitHubFolder` | `compiler_yaml_helpers.go` | 55 |
| `JobManager.GetTopologicalOrder` | `jobs.go` | 52 |

Tests to check: `repo_memory_test.go`, `compiler_yaml_helpers_test.go`, `jobs_test.go`.

### Phase 12 — Engine helpers (~57 LOC, 2 functions)
File: `pkg/workflow/agentic_engine.go`

| Function | LOC |
|----------|-----|
| `GenerateSecretValidationStep` | 31 |
| `BaseEngine.convertStepToYAML` | 15 |

Note: `EngineRegistry.GetAllEngines` (11 LOC) is also dead but below the threshold — skip.

Tests to check: `agentic_engine_test.go`.

### Phase 13 — Error handling cluster (~85 LOC, 4 functions)
Files: `pkg/workflow/error_aggregation.go`, `error_helpers.go`

| Function | File | LOC |
|----------|------|-----|
| `FormatAggregatedError` | `error_aggregation.go` | 30 |
| `EnhanceError` | `error_helpers.go` | 22 |
| `SplitJoinedErrors` | `error_aggregation.go` | 21 |
| `WrapErrorWithContext` | `error_helpers.go` | 12 |

Note: `ErrorCollector.HasErrors` (5 LOC) is also dead but below the threshold — skip.

Tests to check: `error_aggregation_test.go`, `error_helpers_test.go`.

### Phase 14 — Safe outputs env & config helpers (~129 LOC, 7 functions)
Files: `pkg/workflow/safe_outputs_env.go`, `safe_outputs_config_helpers.go`

| Function | File | LOC |
|----------|------|-----|
| `getEnabledSafeOutputToolNamesReflection` | `safe_outputs_config_helpers.go` | 31 |
| `Compiler.formatDetectionRunsOn` | `safe_outputs_config_helpers.go` | 30 |
| `applySafeOutputEnvToSlice` | `safe_outputs_env.go` | 25 |
| `GetEnabledSafeOutputToolNames` | `safe_outputs_config_helpers.go` | 12 |
| `buildLabelsEnvVar` | `safe_outputs_env.go` | 11 |
| `buildTitlePrefixEnvVar` | `safe_outputs_env.go` | 10 |
| `buildCategoryEnvVar` | `safe_outputs_env.go` | 10 |

Tests to check: `safe_outputs_env_test.go`, `safe_outputs_config_helpers_test.go`.

### Phase 15 — MCP config rendering (~106 LOC, 4 functions)
Files: `pkg/workflow/mcp_config_builtin.go`, `mcp_config_custom.go`, `mcp_config_validation.go`, `mcp_playwright_config.go`

| Function | File | LOC |
|----------|------|-----|
| `renderAgenticWorkflowsMCPConfigTOML` | `mcp_config_builtin.go` | 67 |
| `renderCustomMCPConfigWrapper` | `mcp_config_custom.go` | 26 |
| `getTypeString` | `mcp_config_validation.go` | 25 |
| `renderSafeOutputsMCPConfigTOML` | `mcp_config_builtin.go` | 13 |
| `generatePlaywrightDockerArgs` | `mcp_playwright_config.go` | 12 |

Note: `renderSafeOutputsMCPConfig` (8 LOC), `renderPlaywrightMCPConfig` (8 LOC), `getPlaywrightDockerImageVersion` (11 LOC) are also dead — include them in this phase for completeness.

Tests to check: `mcp_config_builtin_test.go`, `mcp_config_custom_test.go`, `mcp_playwright_config_test.go`.

### Phase 16 — Validation & git helpers (~140 LOC, 4 functions)
Files: `pkg/workflow/tools_validation.go`, `concurrency_validation.go`, `config_helpers.go`, `git_helpers.go`, `permissions_validation.go`

| Function | File | LOC |
|----------|------|-----|
| `isGitToolAllowed` | `tools_validation.go` | 45 |
| `extractGroupExpression` | `concurrency_validation.go` | 39 |
| `parseParticipantsFromConfig` | `config_helpers.go` | 32 |
| `GetCurrentGitTag` | `git_helpers.go` | 24 |
| `GetToolsetsData` | `permissions_validation.go` | 19 |

Tests to check: `tools_validation_test.go`, `concurrency_validation_test.go`, `config_helpers_test.go`, `git_helpers_test.go`.

### Phase 17 — Diverse medium functions (~210 LOC, 10 functions)
Various files that each have one dead function of moderate size (12–26 LOC).

| Function | File | LOC |
|----------|------|-----|
| `getSafeInputsEnvVars` | `safe_inputs_renderer.go` | 26 |
| `ExtractMCPServer` | `metrics.go` | 24 |
| `ClearRepositoryFeaturesCache` | `repository_features_validation.go` | 25 |
| `unmarshalFromMap` | `frontmatter_types.go` | 23 |
| `Compiler.extractYAMLValue` | `frontmatter_extraction_yaml.go` | 22 |
| `GetMappings` | `expression_extraction.go` | 20 |
| `ParseSafeInputs` | `safe_inputs_parser.go` | 20 |
| `validateSecretReferences` | `secrets_validation.go` | 18 |
| `WorkflowStep.ToYAML` | `step_types.go` | 14 |
| `FilterMap` | `pkg/sliceutil/sliceutil.go` | 13 |

Note: `CheckoutManager.GetCurrentRepository` (12 LOC), `getCurrentCheckoutRepository` (19 LOC), `ParseIntFromConfig` (19 LOC), `NormalizeExpressionForComparison` (12 LOC), and `mergeSafeJobsFromIncludes` may also be included if still dead after prior phases.

Tests to check: per-file test files as named.

---

## Summary

| Metric | Value |
|--------|-------|
| Dead functions after phases 5–8 | 85 |
| WASM false positives (skip) | 2 |
| Shared test infrastructure (skip) | 3 |
| Compiler option test infrastructure (skip) | 9 |
| Functions < 10 LOC (skip) | ~18 |
| **Functions to delete across phases 9–17** | **~53** |
| Phases remaining | 9 |
| Estimated LOC to remove | ~1,650 |

**Estimated effort per phase:** 20–45 minutes (larger phases have many test updates).
**Recommended execution order:** Phases 9–17 top-to-bottom; phases are stacked (dc-N based on dc-(N-1)).

---

## Per-phase checklist

For each phase:

- [ ] Run `deadcode ./cmd/... ./internal/tools/... 2>/dev/null` to confirm current dead list
- [ ] For each dead function, `grep -rn "FuncName" --include="*.go"` to find all callers
- [ ] Delete the function
- [ ] Delete test functions that exclusively call the deleted function (not shared helpers)
- [ ] Check for now-unused imports in edited files
- [ ] If deleting the last function in a file, delete the entire file
- [ ] If editing `pkg/console/`, check `pkg/console/console_wasm.go` for calls to deleted functions
- [ ] `go build ./...`
- [ ] `GOARCH=wasm GOOS=js go build ./pkg/console/...` (if `pkg/console/` was touched)
- [ ] `go vet ./...`
- [ ] `go vet -tags=integration ./...`
- [ ] `make fmt`
- [ ] Run selective tests for touched packages: `go test -v -run "TestAffected" ./pkg/...`
- [ ] Commit with message: `chore: remove dead functions (phase N) — X -> Y dead`
- [ ] Open PR, confirm CI passes before merging
