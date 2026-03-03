# Dead Code Removal Guide

## How to find dead code

```bash
deadcode ./cmd/... ./internal/tools/... 2>/dev/null
```

**Critical:** Always include `./internal/tools/...` — it covers separate binaries called by the Makefile (e.g. `make actions-build`). Running `./cmd/...` alone gives false positives.

## Correct methodology

`deadcode` analyses the production binary entry points only. **Test files compile into a separate test binary** and do not keep production code alive. A function flagged by `deadcode` is dead regardless of whether test files call it.

**However:** before deleting any dead function, always check whether it has **test callers**:
```bash
grep -rn "FunctionName" --include="*_test.go" pkg/ cmd/
```

Functions with test callers are **test infrastructure** — they exercise production logic and should **not** be deleted. Only delete functions that have no callers in either production code or test files (i.e., truly orphaned).

**Correct approach:**
1. `deadcode` flags `Foo` as unreachable from production binary
2. Check: `grep -rn "Foo" --include="*_test.go"` — if any test callers exist, leave it
3. Only if no test callers: delete `Foo` **and** any test functions that exclusively test `Foo`

**Key learning (phase 10 investigation):** After phases 5–9, a systematic analysis of all 76 remaining dead functions found that **69 of them have direct test callers**. Most of the originally-planned phases 10–17 targeted functions that are valuable test infrastructure. The effort is substantially complete.

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

### ✅ Phase 9 — Output job builders (~480 LOC, 8 functions) — COMPLETE
Files: `pkg/workflow/add_comment.go`, `create_code_scanning_alert.go`, `create_discussion.go`, `create_pr_review_comment.go`, `create_agent_session.go`, `missing_issue_reporting.go`, `missing_data.go`, `missing_tool.go`

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
| `Compiler.buildCreateOutputMissingDataJob` | `missing_data.go` | 17 |
| `Compiler.buildCreateOutputMissingToolJob` | `missing_tool.go` | 17 |

---

## ⭕ Effort complete — remaining dead code is test infrastructure

After a systematic analysis of all 76 remaining dead functions (phase 10 investigation),
the dead code removal effort is **effectively complete**.

**Findings:**
- 69 of the 76 remaining dead functions have **direct test callers** → test infrastructure, keep
- 5 are truly orphaned from all callers but are **tiny** (≤ 7 LOC each) → not worth removing
- 2 are called transitively by test infrastructure (`getPlaywrightMCPPackageVersion`, `getEnabledSafeOutputToolNamesReflection`) → keep

**Remaining 5 truly orphaned functions (too small to remove):**

| Function | File | LOC |
|----------|------|-----|
| `Compiler.GetArtifactManager` | `compiler_types.go:333` | 6 |
| `RunGit` | `git_helpers.go:119` | 6 |
| `ExecGHWithOutput` | `github_cli.go:84` | 7 |
| `GenerateSafeInputGoToolScriptForInspector` | `safe_inputs_generator.go:391` | 3 |
| `IsSafeInputsHTTPMode` | `safe_inputs_parser.go:64` | 7 |

**Previously-planned phases 10–17 are superseded:**

All phase 10–17 targets were found to have test callers and are valuable test infrastructure:
- `generateUnifiedPromptStep`, `buildCreateProjectStepConfig`, `generateJobName`, `mergeSafeJobsFromIncludes` (phase 10) → all have test callers
- `generateRepoMemoryPushSteps`, `generateCheckoutGitHubFolder`, `GetTopologicalOrder` (phase 11) → all have test callers
- All remaining phases → targets have test callers

---

## Summary

| Metric | Value |
|--------|-------|
| Dead functions at start (after phases 1–4) | 107 |
| Dead functions removed in phases 5–9 | ~31 |
| Dead functions remaining | 76 |
| WASM false positives (skip) | 2 |
| Shared test infrastructure (skip) | ≥69 |
| Functions < 10 LOC, no test callers (skip) | 5 |
| **Functions still requiring removal** | **0** |

**Phases 5–9 are complete. Phases 10–17 are superseded — all remaining dead functions are either test infrastructure (69), WASM API surface (2), or too small to bother with (5).**

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
