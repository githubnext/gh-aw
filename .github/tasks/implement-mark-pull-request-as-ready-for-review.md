---
title: Implement mark-pull-request-as-ready-for-review safe output type
description: Reimplementation guide for the mark-pull-request-as-ready-for-review safe output type from a clean branch
labels: [enhancement, safe-outputs, agent-capabilities]
priority: medium
estimated_effort: 2-3 hours
---

# Implement mark-pull-request-as-ready-for-review Safe Output Type

## Objective

Create a new safe output type that enables AI agents to mark draft pull requests as ready for review by setting `draft: false` and posting a reason comment.

## Implementation Guide

This task reimplements the changes from PR #[number] on a clean branch. Follow these steps systematically.

### Core Files to Create/Modify

**New Files:**
1. `actions/setup/js/mark_pull_request_as_ready_for_review.cjs` - JavaScript implementation
2. `pkg/workflow/mark_pull_request_as_ready_for_review.go` - Go config and parser
3. `pkg/cli/workflows/test-claude-mark-pull-request-as-ready-for-review.md` - Claude test
4. `pkg/cli/workflows/test-codex-mark-pull-request-as-ready-for-review.md` - Codex test
5. `pkg/cli/workflows/test-copilot-mark-pull-request-as-ready-for-review.md` - Copilot test

**Modified Files:**
1. `schemas/agent-output.json` - Add MarkPullRequestAsReadyForReviewOutput
2. `pkg/parser/schemas/main_workflow_schema.json` - Add to safe-outputs
3. `actions/setup/js/types/safe-outputs.d.ts` - Add TypeScript interface
4. `actions/setup/js/types/safe-outputs-config.d.ts` - Add config interface
5. `pkg/workflow/js/safe_outputs_tools.json` - Add tool signature
6. `pkg/workflow/compiler_types.go` - Add to SafeOutputsConfig
7. `pkg/workflow/safe_outputs_config.go` - Add parser call
8. `pkg/workflow/safe_outputs_config_generation.go` - Add config generation
9. `pkg/workflow/compiler_safe_outputs_prs.go` - Add step builder
10. `pkg/workflow/compiler_safe_outputs_core.go` - Integrate into job
11. `pkg/workflow/js.go` - Add script getter

### Step-by-Step Implementation

#### 1. JSON Schemas (15 min)

Update `schemas/agent-output.json`:
- Add `MarkPullRequestAsReadyForReviewOutput` to `$defs`
- Add to `SafeOutput` oneOf array
- Fields: type (const), pull_request_number (optional), reason (required)

Update `pkg/parser/schemas/main_workflow_schema.json`:
- Add `mark-pull-request-as-ready-for-review` property
- Standard config: max, target, target-repo, required-labels, required-title-prefix, github-token

#### 2. TypeScript Types (10 min)

Update `actions/setup/js/types/safe-outputs.d.ts`:
- Add `MarkPullRequestAsReadyForReviewItem` interface
- Add to `SafeOutputItem` union
- Export in list

Update `actions/setup/js/types/safe-outputs-config.d.ts`:
- Add `MarkPullRequestAsReadyForReviewConfig` interface
- Add to `SpecificSafeOutputConfig` union
- Export in list

#### 3. JavaScript Implementation (30 min)

Create `actions/setup/js/mark_pull_request_as_ready_for_review.cjs`:
```javascript
// Key features:
- Check staged mode (GH_AW_SAFE_OUTPUTS_STAGED)
- Parse GH_AW_AGENT_OUTPUT
- Filter type === "mark_pull_request_as_ready_for_review"
- Staged: Show preview with 🎭 emoji
- Live: Update PR draft=false, post comment
- Error handling with core.setFailed()
```

#### 4. Tool Signature (5 min)

Update `pkg/workflow/js/safe_outputs_tools.json`:
- Add tool with name "mark_pull_request_as_ready_for_review"
- inputSchema: reason (required), pull_request_number (optional)
- **Run `make build` after this change**

#### 5. Go Configuration (20 min)

Create `pkg/workflow/mark_pull_request_as_ready_for_review.go`:
- Define `MarkPullRequestAsReadyForReviewConfig` struct
- Implement parser function
- Add script getter

#### 6. Compiler Integration (30 min)

Update `pkg/workflow/compiler_types.go`:
- Add field to `SafeOutputsConfig`

Update `pkg/workflow/safe_outputs_config.go`:
- Call parser in `extractSafeOutputsConfig()`

Update `pkg/workflow/safe_outputs_config_generation.go`:
- Add to `generateSafeOutputsConfig()`
- Add to `generateFilteredToolsJSON()`

Update `pkg/workflow/compiler_safe_outputs_prs.go`:
- Add `buildMarkPullRequestAsReadyForReviewStepConfig()` function

Update `pkg/workflow/js.go`:
- Add getter function

Update `pkg/workflow/compiler_safe_outputs_core.go`:
- Add to script collection (~line 100)
- Add step building (~line 200)
- Add permissions

#### 7. Test Workflows (15 min)

Create three test files with identical structure except engine name:
```markdown
---
on: workflow_dispatch
permissions:
  contents: read
engine: [claude|codex|copilot]
safe-outputs:
  mark-pull-request-as-ready-for-review:
    max: 1
timeout-minutes: 5
strict: false
---

# Test Mark Pull Request as Ready for Review

Instructions for AI to call mark_pull_request_as_ready_for_review tool
```

### Build and Validation (15 min)

```bash
# 1. Build
make build

# 2. Format & lint
make fmt-cjs
make lint-cjs

# 3. Test
make test-unit

# 4. Compile test workflows
./gh-aw compile pkg/cli/workflows/test-copilot-mark-pull-request-as-ready-for-review.md

# 5. Recompile all
make recompile
```

### Verification Checklist

- [ ] Build succeeds
- [ ] JavaScript lints clean
- [ ] Unit tests pass
- [ ] Test workflows compile
- [ ] Tool in filtered JSON
- [ ] Step in compiled workflow
- [ ] `TestGetSafeOutputsToolsJSON/tool_mark_pull_request_as_ready_for_review` passes

## Key Patterns to Follow

### Naming Conventions
- kebab-case: `mark-pull-request-as-ready-for-review` (YAML config)
- snake_case: `mark_pull_request_as_ready_for_review` (tool name, JS)
- PascalCase: `MarkPullRequestAsReadyForReview` (Go types)
- camelCase: `markPullRequestAsReadyForReview` (TypeScript)

### Code Patterns
- Use `BaseSafeOutputConfig` and `SafeOutputTargetConfig` for common fields
- Use shared helpers: `ParseTargetConfig()`, `BuildTargetEnvVar()`, etc.
- Follow error handling pattern from `update_pull_request.cjs`
- Use `core.info()` for logging, `core.setFailed()` for errors

### Testing
- All three engines need test workflows
- Use `strict: false` to avoid permission issues
- Test both staged and live modes

## Reference Implementation

See original PR for complete implementation details. Key files to reference:
- `actions/setup/js/update_pull_request.cjs` - Similar PR logic
- `pkg/workflow/update_pull_request.go` - Config pattern
- `pkg/workflow/compiler_safe_outputs_prs.go` - Other PR steps

## Notes

- File `safe_outputs_tools.json` is embedded - rebuild required
- Run `make recompile` before final commit
- Verify no merge conflicts from main
- Test on actual draft PR if possible

## Estimated Time

Total: 2-3 hours for clean implementation
