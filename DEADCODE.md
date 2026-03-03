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

## Batch plan (107 dead functions as of 2026-03-02)

Batches 1–4 have been completed. The original batches 5–16 are superseded by this plan;
many of those functions were removed during prior work, and the remainder (plus newly
discovered dead code) are redistributed below into 30 focused phases.

Each phase: delete the dead functions, delete tests that exclusively test them,
run verification, commit, open PR.

**WASM false positives (do not delete):**
- `Compiler.CompileToYAML` (`compiler_string_api.go:15`) — used by `cmd/gh-aw-wasm`
- `Compiler.ParseWorkflowString` (`compiler_string_api.go:52`) — used by `cmd/gh-aw-wasm`

**Shared test infrastructure (do not delete):**
- `containsInNonCommentLines`, `indexInNonCommentLines`, `extractJobSection` (`compiler_test_helpers.go`) — used by ≥15 test files

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

### Phase 9 — compiler option functions part 1 (5 functions)
File: `pkg/workflow/compiler_types.go`

| Function | Line |
|----------|------|
| `WithCustomOutput` | 26 |
| `WithVersion` | 31 |
| `WithSkipValidation` | 36 |
| `WithNoEmit` | 41 |
| `WithStrictMode` | 46 |

Tests to check: `compiler_types_test.go`, `compiler_test.go` — remove tests for these `With*` option constructors.

### Phase 10 — compiler option functions part 2 (5 functions)
File: `pkg/workflow/compiler_types.go`

| Function | Line |
|----------|------|
| `WithForceRefreshActionPins` | 56 |
| `WithWorkflowIdentifier` | 61 |
| `NewCompilerWithVersion` | 160 |
| `Compiler.GetSharedActionResolverForTest` | 305 |
| `Compiler.GetArtifactManager` | 333 |

Note: `GetSharedActionResolverForTest` may be used only in tests — delete it AND any test callers.
After this phase, clean up the `CompilerOption` type if no live `With*` functions remain.

### Phase 11 — agentic engine (3 functions)
File: `pkg/workflow/agentic_engine.go`

| Function | Line |
|----------|------|
| `BaseEngine.convertStepToYAML` | 333 |
| `GenerateSecretValidationStep` | 430 |
| `EngineRegistry.GetAllEngines` | 502 |

Tests to check: `agentic_engine_test.go`.

### Phase 12 — error handling utilities (5 functions)
Files: `pkg/workflow/error_aggregation.go` (3), `pkg/workflow/error_helpers.go` (2)

| Function | File | Line |
|----------|------|------|
| `ErrorCollector.HasErrors` | `error_aggregation.go` | 92 |
| `FormatAggregatedError` | `error_aggregation.go` | 144 |
| `SplitJoinedErrors` | `error_aggregation.go` | 174 |
| `EnhanceError` | `error_helpers.go` | 165 |
| `WrapErrorWithContext` | `error_helpers.go` | 187 |

Tests to check: `error_aggregation_test.go`, `error_helpers_test.go`.

### Phase 13 — safe outputs env vars (4 functions)
File: `pkg/workflow/safe_outputs_env.go`

| Function | Line |
|----------|------|
| `applySafeOutputEnvToSlice` | 47 |
| `buildTitlePrefixEnvVar` | 311 |
| `buildLabelsEnvVar` | 321 |
| `buildCategoryEnvVar` | 332 |

Tests to check: `safe_outputs_env_test.go`, `safe_output_helpers_test.go`.

### Phase 14 — safe outputs config helpers (3 functions)
File: `pkg/workflow/safe_outputs_config_helpers.go`

| Function | Line |
|----------|------|
| `getEnabledSafeOutputToolNamesReflection` | 85 |
| `Compiler.formatDetectionRunsOn` | 127 |
| `GetEnabledSafeOutputToolNames` | 216 |

Tests to check: `safe_outputs_config_helpers_test.go`, `threat_detection_test.go`.

### Phase 15 — Playwright MCP config (3 functions)
File: `pkg/workflow/mcp_playwright_config.go`

| Function | Line |
|----------|------|
| `getPlaywrightDockerImageVersion` | 15 |
| `getPlaywrightMCPPackageVersion` | 26 |
| `generatePlaywrightDockerArgs` | 32 |

Tests to check: `mcp_playwright_config_test.go`.

### Phase 16 — MCP config builtins (3 functions)
File: `pkg/workflow/mcp_config_builtin.go`

| Function | Line |
|----------|------|
| `renderSafeOutputsMCPConfig` | 113 |
| `renderSafeOutputsMCPConfigTOML` | 295 |
| `renderAgenticWorkflowsMCPConfigTOML` | 308 |

Tests to check: `mcp_config_builtin_test.go`, `mcp_config_refactor_test.go`, `mcp_config_shared_test.go`.

### Phase 17 — MCP config miscellaneous (4 functions)
Files: `pkg/workflow/mcp_config_custom.go` (1), `mcp_config_playwright_renderer.go` (1), `mcp_config_types.go` (1), `mcp_config_validation.go` (1)

| Function | File | Line |
|----------|------|------|
| `renderCustomMCPConfigWrapper` | `mcp_config_custom.go` | 21 |
| `renderPlaywrightMCPConfig` | `mcp_config_playwright_renderer.go` | 71 |
| `MapToolConfig.GetAny` | `mcp_config_types.go` | 99 |
| `getTypeString` | `mcp_config_validation.go` | 176 |

Tests to check: `mcp_config_custom_test.go`, `mcp_config_playwright_renderer_test.go`, `mcp_config_types_test.go`, `mcp_config_validation_test.go`.

### Phase 18 — safe inputs system (4 functions)
Files: `pkg/workflow/safe_inputs_generator.go` (1), `safe_inputs_parser.go` (2), `safe_inputs_renderer.go` (1)

| Function | File | Line |
|----------|------|------|
| `GenerateSafeInputGoToolScriptForInspector` | `safe_inputs_generator.go` | 391 |
| `IsSafeInputsHTTPMode` | `safe_inputs_parser.go` | 64 |
| `ParseSafeInputs` | `safe_inputs_parser.go` | 210 |
| `getSafeInputsEnvVars` | `safe_inputs_renderer.go` | 14 |

Tests to check: `safe_inputs_generator_test.go`, `safe_inputs_parser_test.go`, `safe_inputs_renderer_test.go`.

### Phase 19 — safe outputs validation & safe jobs (3 functions)
Files: `pkg/workflow/safe_output_validation_config.go` (2), `safe_jobs.go` (1)

| Function | File | Line |
|----------|------|------|
| `GetValidationConfigForType` | `safe_output_validation_config.go` | 409 |
| `GetDefaultMaxForType` | `safe_output_validation_config.go` | 415 |
| `HasSafeJobsEnabled` | `safe_jobs.go` | 34 |

Tests to check: `safe_output_validation_config_test.go`, `safe_jobs_test.go`.

### Phase 20 — safe output job builders: comments & discussions (4 functions)
Files: `pkg/workflow/add_comment.go` (1), `create_code_scanning_alert.go` (1), `create_discussion.go` (1), `create_pr_review_comment.go` (1)

| Function | File | Line |
|----------|------|------|
| `Compiler.buildCreateOutputAddCommentJob` | `add_comment.go` | 34 |
| `Compiler.buildCreateOutputCodeScanningAlertJob` | `create_code_scanning_alert.go` | 21 |
| `Compiler.buildCreateOutputDiscussionJob` | `create_discussion.go` | 132 |
| `Compiler.buildCreateOutputPullRequestReviewCommentJob` | `create_pr_review_comment.go` | 24 |

Note: If these are the only functions in their files, consider deleting the entire file.

Tests to check: `add_comment_test.go`, `create_code_scanning_alert_test.go`, `create_discussion_test.go`, `create_pr_review_comment_test.go`.

### Phase 21 — safe output job builders: sessions & missing data (3 functions)
Files: `pkg/workflow/create_agent_session.go` (1), `missing_data.go` (1), `missing_tool.go` (1)

| Function | File | Line |
|----------|------|------|
| `Compiler.buildCreateOutputAgentSessionJob` | `create_agent_session.go` | 88 |
| `Compiler.buildCreateOutputMissingDataJob` | `missing_data.go` | 12 |
| `Compiler.buildCreateOutputMissingToolJob` | `missing_tool.go` | 12 |

Note: If the file contains only the dead function, delete the entire file.

Tests to check: `create_agent_session_test.go`, `missing_data_test.go`, `missing_tool_test.go`.

### Phase 22 — safe output compilation (3 functions)
Files: `pkg/workflow/compiler_safe_outputs.go` (2), `compiler_safe_outputs_specialized.go` (1)

| Function | File | Line |
|----------|------|------|
| `Compiler.generateJobName` | `compiler_safe_outputs.go` | 185 |
| `Compiler.mergeSafeJobsFromIncludes` | `compiler_safe_outputs.go` | 219 |
| `Compiler.buildCreateProjectStepConfig` | `compiler_safe_outputs_specialized.go` | 139 |

Tests to check: `compiler_safe_outputs_test.go`, `compiler_safe_outputs_specialized_test.go`.

### Phase 23 — issue reporting (2 functions)
File: `pkg/workflow/missing_issue_reporting.go`

| Function | Line |
|----------|------|
| `Compiler.buildIssueReportingJob` | 48 |
| `envVarPrefix` | 175 |

Note: If these are the only non-trivial functions in the file, consider deleting it entirely.

Tests to check: `missing_issue_reporting_test.go`.

### Phase 24 — checkout manager (2 functions)
File: `pkg/workflow/checkout_manager.go`

| Function | Line |
|----------|------|
| `CheckoutManager.GetCurrentRepository` | 186 |
| `getCurrentCheckoutRepository` | 553 |

Tests to check: `checkout_manager_test.go`.

### Phase 25 — expression processing (3 functions)
Files: `pkg/workflow/expression_extraction.go` (1), `expression_parser.go` (1), `expression_validation.go` (1)

| Function | File | Line |
|----------|------|------|
| `ExpressionExtractor.GetMappings` | `expression_extraction.go` | 239 |
| `NormalizeExpressionForComparison` | `expression_parser.go` | 463 |
| `ValidateExpressionSafetyPublic` | `expression_validation.go` | 359 |

Tests to check: `expression_extraction_test.go`, `expression_parser_test.go`, `expression_validation_test.go`.

### Phase 26 — frontmatter extraction (3 functions)
Files: `pkg/workflow/frontmatter_extraction_metadata.go` (1), `frontmatter_extraction_yaml.go` (1), `frontmatter_types.go` (1)

| Function | File | Line |
|----------|------|------|
| `extractMapFromFrontmatter` | `frontmatter_extraction_metadata.go` | 246 |
| `Compiler.extractYAMLValue` | `frontmatter_extraction_yaml.go` | 18 |
| `unmarshalFromMap` | `frontmatter_types.go` | 196 |

Tests to check: `frontmatter_extraction_metadata_test.go`, `frontmatter_extraction_yaml_test.go`, `frontmatter_types_test.go`.

### Phase 27 — git, GitHub CLI & shell helpers (4 functions)
Files: `pkg/workflow/git_helpers.go` (2), `github_cli.go` (1), `shell.go` (1)

| Function | File | Line |
|----------|------|------|
| `GetCurrentGitTag` | `git_helpers.go` | 69 |
| `RunGit` | `git_helpers.go` | 119 |
| `ExecGHWithOutput` | `github_cli.go` | 84 |
| `shellEscapeCommandString` | `shell.go` | 82 |

Tests to check: `git_helpers_test.go`, `github_cli_test.go`, `shell_test.go`.

### Phase 28 — config & concurrency validation (4 functions)
Files: `pkg/workflow/config_helpers.go` (2), `concurrency_validation.go` (1), `permissions_validation.go` (1)

| Function | File | Line |
|----------|------|------|
| `parseParticipantsFromConfig` | `config_helpers.go` | 131 |
| `ParseIntFromConfig` | `config_helpers.go` | 218 |
| `extractGroupExpression` | `concurrency_validation.go` | 289 |
| `GetToolsetsData` | `permissions_validation.go` | 77 |

Tests to check: `config_helpers_test.go`, `concurrency_validation_test.go`, `permissions_validation_test.go`.

### Phase 29 — security & error types (4 functions)
Files: `pkg/workflow/markdown_security_scanner.go` (1), `secrets_validation.go` (1), `tools_validation.go` (1), `shared_workflow_error.go` (1)

| Function | File | Line |
|----------|------|------|
| `SecurityFinding.String` | `markdown_security_scanner.go` | 64 |
| `validateSecretReferences` | `secrets_validation.go` | 31 |
| `isGitToolAllowed` | `tools_validation.go` | 31 |
| `NewSharedWorkflowError` | `shared_workflow_error.go` | 21 |

Note: If `SharedWorkflowError` has no remaining constructor, consider deleting the type entirely.

Tests to check: `markdown_security_scanner_test.go`, `secrets_validation_test.go`, `tools_validation_test.go`, `shared_workflow_error_test.go`.

### Phase 30 — step & job types (3 functions)
Files: `pkg/workflow/step_types.go` (2), `jobs.go` (1)

| Function | File | Line |
|----------|------|------|
| `WorkflowStep.IsRunStep` | `step_types.go` | 36 |
| `WorkflowStep.ToYAML` | `step_types.go` | 171 |
| `JobManager.GetTopologicalOrder` | `jobs.go` | 412 |

Tests to check: `step_types_test.go`, `jobs_test.go`.

### Phase 31 — utilities cleanup (4 functions)
Files: `pkg/sliceutil/sliceutil.go` (1), `pkg/workflow/semver.go` (1), `repository_features_validation.go` (1), `compiler_yaml_ai_execution.go` (1)

| Function | File | Line |
|----------|------|------|
| `FilterMap` | `pkg/sliceutil/sliceutil.go` | 49 |
| `extractMajorVersion` | `semver.go` | 41 |
| `ClearRepositoryFeaturesCache` | `repository_features_validation.go` | 83 |
| `Compiler.convertGoPatternToJavaScript` | `compiler_yaml_ai_execution.go` | 116 |

Tests to check: `sliceutil_test.go`, `semver_test.go`, `repository_features_validation_test.go`, `compiler_yaml_ai_execution_test.go`.

### Phase 32 — compiler helpers (3 functions)
Files: `pkg/workflow/compiler_yaml_helpers.go` (1), `unified_prompt_step.go` (1), `repo_memory.go` (1)

| Function | File | Line |
|----------|------|------|
| `Compiler.generateCheckoutGitHubFolder` | `compiler_yaml_helpers.go` | 221 |
| `Compiler.generateUnifiedPromptStep` | `unified_prompt_step.go` | 30 |
| `generateRepoMemoryPushSteps` | `repo_memory.go` | 520 |

Note: If `unified_prompt_step.go` contains only the dead function, delete the entire file.

Tests to check: `compiler_yaml_helpers_test.go`, `unified_prompt_step_test.go`, `repo_memory_test.go`.

### Phase 33 — metrics extraction (2 functions)
File: `pkg/workflow/metrics.go`

| Function | Line |
|----------|------|
| `ExtractFirstMatch` | 39 |
| `ExtractMCPServer` | 274 |

Tests to check: `metrics_test.go`.

### Phase 34 — WASM string API audit (0 deletions)
File: `pkg/workflow/compiler_string_api.go`

Functions: `Compiler.CompileToYAML` (line 15), `Compiler.ParseWorkflowString` (line 52)

**Action:** Do not delete. Verify that `cmd/gh-aw-wasm/main.go` still calls these functions.
If WASM binary is ever removed, both functions become deletable.

Run: `grep -rn "CompileToYAML\|ParseWorkflowString" cmd/gh-aw-wasm/`

---

## Summary

| Metric | Value |
|--------|-------|
| Total dead functions reported | 107 |
| WASM false positives (skip) | 2 |
| Shared test infrastructure (skip) | 3 |
| Functions to delete | **102** |
| Phases with deletions | 29 |
| Audit-only phases | 1 |
| Average functions per phase | 3.4 |

**Estimated effort per phase:** 15–30 minutes (delete, test, verify, commit).
**Estimated total effort:** ~10–15 hours across all 30 phases.

**Recommended execution order:** Phases are designed to be executed top-to-bottom. Phases within the same domain (e.g., MCP phases 15–17, safe output phases 13–14, 19–22) can be combined into larger PRs if velocity is high.

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
