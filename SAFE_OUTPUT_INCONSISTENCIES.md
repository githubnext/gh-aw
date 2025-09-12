# Safe Output Types Analysis - Inconsistencies Found

## Overview
This document outlines inconsistencies found while reviewing the safe output related JavaScript/Go code and prompting to generate TypeScript definitions.

## Inconsistencies Identified

### 1. Missing `create-discussion` from Main Output Types Table

**Location**: `docs/src/content/docs/reference/safe-outputs.md`

**Issue**: The main table of output types (lines 10-20) is missing the `create-discussion` output type, but it is:
- Properly documented in the "New Discussion Creation" section (line 76+)
- Implemented in Go config as `CreateDiscussionsConfig` 
- Implemented in JavaScript validation (`collect_ndjson_output.cjs`)
- Has a dedicated JavaScript processor (`create_discussion.cjs`)

**Fix Required**: Add `create-discussion` to the main output types table:
```markdown
| **New Discussion Creation** | `create-discussion:` | Create GitHub discussions based on workflow output | 1 |
```

### 2. Inconsistent Default Max Values Documentation

**Issue**: There are inconsistencies between documented default max values and implemented defaults:

- Documentation says `add-issue-label` has default max of 3
- JavaScript validation code uses default max of 5 for `add-issue-label` (line 177 in `collect_ndjson_output.cjs`)
- Documentation says `create-pull-request-review-comment` has default max of 1  
- JavaScript validation code uses default max of 10 for `create-pull-request-review-comment` (line 175)

**Current Implementation in `collect_ndjson_output.cjs`**:
```javascript
case "create-pull-request-review-comment":
  return 10; // Default to 10 review comments allowed
case "add-issue-label":
  return 5; // Only one labels operation allowed
```

### 3. Minor Naming Inconsistencies

**Issue**: Go struct field names use different casing/format than YAML keys:
- Go: `CreatePullRequestReviewComments` vs YAML: `create-pull-request-review-comment`
- Go: `CreateCodeScanningAlerts` vs YAML: `create-code-scanning-alert`  
- Go: `PushToPullRequestBranch` vs YAML: `push-to-pr-branch`

This is expected behavior for Go struct tags, so not a real inconsistency.

### 4. Environment Variable Naming Patterns

**Issue**: Some environment variables follow different naming patterns:
- Most use `GITHUB_AW_` prefix consistently
- Some use specific prefixes like `GITHUB_AW_PR_`, `GITHUB_AW_ISSUE_`, `GITHUB_AW_SECURITY_REPORT_`
- Target configurations vary: `GITHUB_AW_PUSH_TARGET` vs contextual targets

This appears to be intentional design for clarity.

## Validation Against Code

### Go Configuration Structs ✅
All safe output types have corresponding Go configuration structs in `pkg/workflow/compiler.go`:
- `CreateIssuesConfig`
- `CreateDiscussionsConfig` ✅ (properly implemented)
- `AddIssueCommentsConfig`
- `CreatePullRequestsConfig`
- `CreatePullRequestReviewCommentsConfig`
- `CreateCodeScanningAlertsConfig`
- `AddIssueLabelsConfig`
- `UpdateIssuesConfig`
- `PushToPullRequestBranchConfig`
- `MissingToolConfig`

### JavaScript Processors ✅
All safe output types have corresponding JavaScript processors:
- `create_issue.cjs`
- `create_discussion.cjs` ✅ (properly implemented)
- `create_comment.cjs`
- `create_pull_request.cjs`
- `create_pr_review_comment.cjs`
- `create_code_scanning_alert.cjs`
- `add_labels.cjs`
- `update_issue.cjs`
- `push_to_pr_branch.cjs`
- `missing_tool.cjs`

### JavaScript Validation ✅
All safe output types are properly validated in `collect_ndjson_output.cjs`:
- Input validation with proper error messages
- Content sanitization applied consistently
- Type-specific field validation
- Max count enforcement

## Summary

The main inconsistency is the missing `create-discussion` entry in the documentation table. The implementation is otherwise complete and consistent across Go and JavaScript code. The TypeScript definitions have been created to accurately reflect the actual implementation.

## Recommendations

1. **Fix Documentation**: Add `create-discussion` to the main output types table
2. **Clarify Default Values**: Document the actual default max values used in code
3. **Consider**: Standardizing on the JavaScript validation defaults or updating them to match documentation