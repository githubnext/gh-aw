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

## Batch plan (248 dead functions as of 2026-02-28)

Each batch: delete the dead functions, delete the tests that exclusively test them, run verification, commit, open PR.

### Batch 5 — simple helpers (11 functions)
Files: `pkg/workflow/validation_helpers.go` (6), `pkg/workflow/map_helpers.go` (5)

Dead functions:
- `ValidateRequired`, `ValidateMaxLength`, `ValidateMinLength`, `ValidateInList`, `ValidatePositiveInt`, `ValidateNonNegativeInt`
- `isEmptyOrNil`, `getMapFieldAsString`, `getMapFieldAsMap`, `getMapFieldAsBool`, `getMapFieldAsInt`

Tests to remove from `validation_helpers_test.go`:
- `TestValidateRequired`, `TestValidateMaxLength`, `TestValidateMinLength`, `TestValidateInList`, `TestValidatePositiveInt`, `TestValidateNonNegativeInt`
- `TestIsEmptyOrNil`, `TestGetMapFieldAsString`, `TestGetMapFieldAsMap`, `TestGetMapFieldAsBool`, `TestGetMapFieldAsInt`

### Batch 6 — engine helpers (5 functions)
File: `pkg/workflow/engine_helpers.go` (5)

Dead functions: `ExtractAgentIdentifier`, `GetHostedToolcachePathSetup`, `GetSanitizedPATHExport`, `GetToolBinsSetup`, `GetToolBinsEnvArg`

Tests to remove from `engine_helpers_test.go`:
- `TestExtractAgentIdentifier`, `TestGetHostedToolcachePathSetup`, `TestGetHostedToolcachePathSetup_Consistency`, `TestGetHostedToolcachePathSetup_UsesToolBins`, `TestGetToolBinsSetup`, `TestGetToolBinsEnvArg`, `TestGetSanitizedPATHExport`, `TestGetSanitizedPATHExport_ShellExecution`

### Batch 7 — domain helpers (10 functions)
File: `pkg/workflow/domains.go` (10)

Dead functions: `mergeDomainsWithNetwork`, `mergeDomainsWithNetworkAndTools`, `GetCopilotAllowedDomains`, `GetCopilotAllowedDomainsWithSafeInputs`, `GetCopilotAllowedDomainsWithTools`, `GetCodexAllowedDomains`, `GetCodexAllowedDomainsWithTools`, `GetClaudeAllowedDomains`, `GetClaudeAllowedDomainsWithSafeInputs`, `GetClaudeAllowedDomainsWithTools`

Tests to remove from `domains_test.go`, `domains_protocol_test.go`, `domains_sort_test.go`, `safe_inputs_firewall_test.go`, `http_mcp_domains_test.go` — remove only the specific test functions that call these dead helpers; keep tests for live functions in those files.

### Batch 8 — expression graph (16 functions)
Files: `pkg/workflow/expression_nodes.go` (4), `pkg/workflow/expression_builder.go` (9), `pkg/workflow/known_needs_expressions.go` (3)

Dead functions in `expression_nodes.go`: `ParenthesesNode.Render`, `NumberLiteralNode.Render`, `TernaryNode.Render`, `ContainsNode.Render`

Dead functions in `expression_builder.go`: `BuildNumberLiteral`, `BuildContains`, `BuildTernary`, `BuildLabelContains`, `BuildActionEquals`, `BuildRefStartsWith`, `BuildExpressionWithDescription`, `BuildPRCommentCondition`, `AddDetectionSuccessCheck`

Dead functions in `known_needs_expressions.go`: `getSafeOutputJobNames`, `hasMultipleSafeOutputTypes`, `getCustomJobNames`

Tests to find and remove: check `expressions_test.go`, `expression_coverage_test.go`, `known_needs_expressions_test.go`.

### Batch 9 — constants & console (18 functions)
Files: `pkg/constants/constants.go` (13), `pkg/console/console.go` (5)

Dead functions in `constants.go`: all `String()`/`IsValid()` methods on `LineLength`, `FeatureFlag`, `URL`, `ModelName`, `WorkflowID`, `EngineName`, plus `MCPServerID.IsValid`

Dead functions in `console.go`: `FormatLocationMessage`, `FormatCountMessage`, `FormatListHeader`, `RenderTree`, `buildLipglossTree`

Tests to remove: relevant subtests in `constants_test.go`; `TestFormatLocationMessage`, `TestRenderTree`, `TestRenderTreeSimple`, `TestFormatCountMessage`, `TestFormatListHeader` in `console_test.go` and related files.

### Batch 10 — agent session builder (1 function)
File: `pkg/workflow/create_agent_session.go`

Dead function: `Compiler.buildCreateOutputAgentSessionJob`

Find and remove its test(s): `grep -rn "buildCreateOutputAgentSessionJob" --include="*_test.go"`.

### Batch 11 — safe-outputs & MCP helpers (13 functions)
Files: `pkg/workflow/safe_outputs_env.go` (4), `pkg/workflow/safe_outputs_config_helpers.go` (3), `pkg/workflow/mcp_playwright_config.go` (3), `pkg/workflow/mcp_config_builtin.go` (3)

Dead functions in `safe_outputs_env.go`: `applySafeOutputEnvToSlice`, `buildTitlePrefixEnvVar`, `buildLabelsEnvVar`, `buildCategoryEnvVar`

Dead functions in `safe_outputs_config_helpers.go`: `getEnabledSafeOutputToolNamesReflection`, `Compiler.formatDetectionRunsOn`, `GetEnabledSafeOutputToolNames`

Dead functions in `mcp_playwright_config.go`: `getPlaywrightDockerImageVersion`, `getPlaywrightMCPPackageVersion`, `generatePlaywrightDockerArgs`

Dead functions in `mcp_config_builtin.go`: `renderSafeOutputsMCPConfig`, `renderSafeOutputsMCPConfigTOML`, `renderAgenticWorkflowsMCPConfigTOML`

Tests to remove: check `safe_output_helpers_test.go`, `version_field_test.go`, `mcp_benchmark_test.go`, `mcp_config_refactor_test.go`, `mcp_config_shared_test.go`, `threat_detection_test.go`.

### Batch 12 — small utilities (9 functions)
Files: `pkg/sliceutil/sliceutil.go` (3), `pkg/stringutil/pat_validation.go` (3), `pkg/workflow/error_aggregation.go` (3)

Dead functions in `sliceutil.go`: `ContainsAny`, `ContainsIgnoreCase`, `FilterMap`

Dead functions in `pat_validation.go`: `IsFineGrainedPAT`, `IsClassicPAT`, `IsOAuthToken`

Dead functions in `error_aggregation.go`: `ErrorCollector.HasErrors`, `FormatAggregatedError`, `SplitJoinedErrors`

### Batch 13 — parser utilities (9 functions)
Files: `pkg/parser/include_expander.go` (3), `pkg/parser/schema_validation.go` (3), `pkg/parser/yaml_error.go` (3)

Dead functions in `include_expander.go`: `ExpandIncludes`, `ProcessIncludesForEngines`, `ProcessIncludesForSafeOutputs`

Dead functions in `schema_validation.go`: `ValidateMainWorkflowFrontmatterWithSchema`, `ValidateIncludedFileFrontmatterWithSchema`, `ValidateMCPConfigWithSchema`

Dead functions in `yaml_error.go`: `ExtractYAMLError`, `extractFromGoccyFormat`, `extractFromStringParsing`

### Batch 14 — agentic engine & compiler types (16 functions)
Files: `pkg/workflow/agentic_engine.go` (3), `pkg/workflow/compiler_types.go` (10+), `pkg/cli/docker_images.go` (6)

Dead functions in `agentic_engine.go`: `BaseEngine.convertStepToYAML`, `GenerateSecretValidationStep`, `EngineRegistry.GetAllEngines`

Dead functions in `compiler_types.go` (check WASM binary first): `WithCustomOutput`, `WithVersion`, `WithSkipValidation`, `WithNoEmit`, `WithStrictMode`, `WithForceRefreshActionPins`, `WithWorkflowIdentifier`, `NewCompilerWithVersion`, `Compiler.GetSharedActionResolverForTest`, `Compiler.GetArtifactManager`

Dead functions in `docker_images.go`: `isDockerAvailable`, `ResetDockerPullState`, `ValidateMCPServerDockerAvailability`, `SetDockerImageDownloading`, `SetMockImageAvailable`, `PrintDockerPullStatus`

### Batch 15 — js.go stubs (6 functions)
File: `pkg/workflow/js.go`

Dead functions: the remaining 6 unreachable `get*Script()` / public `Get*` stubs reported by deadcode.

### Batch 16 — artifact manager (14 functions)
File: `pkg/workflow/artifact_manager.go`

Save for last — most complex, with deep coupling to `artifact_manager_integration_test.go`.

### Remaining (~120 functions)
~80+ files each with 1–3 dead functions. Tackle after the above batches clear the larger clusters.

---

## Per-batch checklist

For each batch:

- [ ] Run `deadcode ./cmd/... ./internal/tools/... 2>/dev/null` to confirm current dead list
- [ ] For each dead function, `grep -rn "FuncName" --include="*.go"` to find all callers
- [ ] Delete the function
- [ ] Delete test functions that exclusively call the deleted function (not shared helpers)
- [ ] Check for now-unused imports in edited files
- [ ] If editing `pkg/console/`, check `pkg/console/console_wasm.go` for calls to the deleted functions
- [ ] `go build ./...`
- [ ] `GOARCH=wasm GOOS=js go build ./pkg/console/...` (if `pkg/console/` was touched)
- [ ] `go vet ./...`
- [ ] `go vet -tags=integration ./...`
- [ ] `make fmt`
- [ ] Run selective tests for touched packages: `go test -v -run "TestAffected" ./pkg/...`
- [ ] Commit with message: `chore: remove dead functions (batch N) — X -> Y dead`
- [ ] Open PR, confirm CI passes before merging
