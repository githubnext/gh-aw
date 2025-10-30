# Follow-Up Issues for String Handling Consolidation

Based on the comprehensive audit documented in `STRING_HANDLING.md` and `docs/string-handling-consolidation-recommendations.md`, here are the recommended follow-up issues to create.

## Issue 1: [High Priority] Consolidate JavaScript Sanitization Functions

**Title**: Consolidate duplicated JavaScript sanitization functions for security

**Labels**: `security`, `refactor`, `priority-high`, `good-first-issue`

**Estimated Effort**: 3-4 hours

**Description**:

### Problem
Three security-critical sanitization functions are duplicated across multiple JavaScript files:

1. **`sanitizeContent`** - Duplicated in 3 files (~450 lines total)
   - `pkg/workflow/js/compute_text.cjs`
   - `pkg/workflow/js/sanitize_output.cjs`
   - `pkg/workflow/js/collect_ndjson_output.cjs`

2. **`sanitizeLabelContent`** - Duplicated in 2 files (~30 lines total)
   - `pkg/workflow/js/add_labels.cjs`
   - `pkg/workflow/js/create_issue.cjs`

3. **`normalizeBranchName`** - Duplicated in 2 files with manual sync requirement
   - `pkg/workflow/js/upload_assets.cjs`
   - `pkg/workflow/js/safe_outputs_mcp_server.cjs`
   - Both files have comments: "IMPORTANT: Keep this function in sync"

### Risk
- **HIGH**: Security fixes must be manually applied to all copies
- **HIGH**: Manual sync requirement is error-prone and risky
- **MEDIUM**: Inconsistent behavior if one copy is updated and others aren't

### Solution
Create shared utility modules:

```javascript
// pkg/workflow/js/utils/sanitize.cjs
module.exports = {
  sanitizeContent,
  sanitizeLabelContent,
  sanitizeUrlDomains,
  sanitizeUrlProtocols,
  convertXmlTagsToParentheses,
  neutralizeMentions,
  neutralizeBotTriggers,
};

// pkg/workflow/js/utils/branch.cjs
module.exports = {
  normalizeBranchName,
};
```

### Tasks
- [ ] Create `pkg/workflow/js/utils/` directory
- [ ] Create `pkg/workflow/js/utils/sanitize.cjs` with all sanitization functions
- [ ] Create `pkg/workflow/js/utils/branch.cjs` with branch name normalization
- [ ] Update `compute_text.cjs` to import from utilities
- [ ] Update `sanitize_output.cjs` to import from utilities
- [ ] Update `collect_ndjson_output.cjs` to import from utilities
- [ ] Update `add_labels.cjs` to import from utilities
- [ ] Update `create_issue.cjs` to import from utilities
- [ ] Update `upload_assets.cjs` to import from utilities
- [ ] Update `safe_outputs_mcp_server.cjs` to import from utilities
- [ ] Add tests for utility modules
- [ ] Run full test suite to verify no regressions
- [ ] Update `STRING_HANDLING.md` to reflect new structure

### Benefits
- Single source of truth for security-critical code
- Security fixes apply universally
- Eliminates manual sync requirement
- Reduces codebase by ~480 lines
- Easier to maintain and test

### Testing Strategy
1. Run existing tests before changes to establish baseline
2. Extract functions to utility modules
3. Update imports in all files
4. Run tests again to verify no breakage
5. Add specific tests for utility modules
6. Test edge cases and security scenarios

### References
- Audit document: `STRING_HANDLING.md`
- Recommendations: `docs/string-handling-consolidation-recommendations.md`
- Summary: `docs/string-handling-audit-summary.md`

---

## Issue 2: [Medium Priority] Align SanitizeWorkflowName Go and JavaScript Implementations

**Title**: Fix inconsistency between Go and JavaScript workflow name sanitization

**Labels**: `consistency`, `refactor`, `priority-medium`

**Estimated Effort**: 1 hour

**Description**:

### Problem
The `SanitizeWorkflowName` function has different behavior in Go vs JavaScript:

**Go version** (`pkg/workflow/strings.go`):
- Consolidates multiple consecutive hyphens into single hyphen
- Example: `"My::Workflow"` → `"my-workflow"` (single hyphen)

**JavaScript version** (`pkg/workflow/js/parse_firewall_logs.cjs`):
- Does NOT consolidate multiple hyphens
- Example: `"My::Workflow"` → `"my--workflow"` (double hyphen)

### Risk
- **MEDIUM**: Could lead to inconsistent workflow naming between Go and JavaScript contexts
- **LOW**: May cause confusion when comparing workflow names

### Investigation Required
Before fixing, determine:
1. Is the JavaScript version intentionally different for firewall log compatibility?
2. Does any existing code depend on the double-hyphen behavior?
3. Are there firewall logs that already use the double-hyphen format?

### Solution Options

**Option A**: Align JavaScript with Go (add hyphen consolidation)
```javascript
function sanitizeWorkflowName(name) {
  return name
    .toLowerCase()
    .replace(/[:\\/\s]/g, "-")
    .replace(/[^a-z0-9._-]/g, "-")
    .replace(/-+/g, "-");  // Add this line to consolidate hyphens
}
```

**Option B**: Document the difference
If the JavaScript version must remain different (e.g., for firewall log compatibility), add clear comments explaining why.

### Tasks
- [ ] Research firewall log parsing to understand requirements
- [ ] Check if any existing logs use double-hyphen format
- [ ] Determine if consolidation would break anything
- [ ] If safe, add hyphen consolidation to JavaScript version
- [ ] If not safe, add detailed comments explaining the difference
- [ ] Update tests to cover the change
- [ ] Update `STRING_HANDLING.md` to document decision

### Benefits
- Consistent behavior across Go and JavaScript
- Clearer understanding of why implementations differ (if they must)
- Reduced confusion for developers

### References
- Go implementation: `pkg/workflow/strings.go`
- JavaScript implementation: `pkg/workflow/js/parse_firewall_logs.cjs`
- Documentation: `STRING_HANDLING.md`

---

## Issue 3: [Low Priority] Create Organized JavaScript Utilities Directory Structure

**Title**: Organize JavaScript utilities into structured directory

**Labels**: `refactor`, `code-organization`, `priority-low`

**Estimated Effort**: 3-4 hours

**Description**:

### Problem
JavaScript utility functions are scattered across individual files with no clear organization structure.

### Solution
Create organized utilities directory:

```
pkg/workflow/js/
  utils/
    sanitize.cjs         # Content sanitization functions
    branch.cjs           # Git branch name normalization
    workflow.cjs         # Workflow name sanitization
    index.cjs            # Re-export all utilities for convenience
```

### Benefits
- Better code organization
- Easier discoverability of utility functions
- Natural location for new utility functions
- Clearer separation of concerns
- Consistency with other projects

### Tasks
- [ ] Design directory structure
- [ ] Create `pkg/workflow/js/utils/` directory
- [ ] Move functions to appropriate modules
- [ ] Create `index.cjs` for convenient imports
- [ ] Update all imports across codebase
- [ ] Update documentation
- [ ] Run tests to verify no breakage

### Note
This issue should be completed AFTER Issue #1 (consolidation). The consolidation creates the utility modules; this issue organizes them into a clean structure.

### References
- Related to Issue #1
- Documentation: `STRING_HANDLING.md`

---

## Issue 4: [Future] Evaluate Go String Utilities Package Reorganization

**Title**: [Discussion] Consider reorganizing Go string handling functions into utilities package

**Labels**: `refactor`, `code-organization`, `discussion`, `future`

**Estimated Effort**: 4-6 hours

**Description**:

### Problem
Go string handling functions are spread across multiple packages:
- `pkg/workflow/strings.go` - `SanitizeWorkflowName`
- `pkg/workflow/workflow_name.go` - `SanitizeIdentifier`
- `pkg/cli/resolver.go` - `NormalizeWorkflowFile`
- `pkg/cli/mcp_registry.go` - `cleanMCPToolID`
- `pkg/cli/trial_command.go` - `sanitizeRepoSlugForFilename`
- etc.

### Considerations
This is a larger refactoring that affects:
- Entire codebase (imports)
- In-flight PRs (merge conflicts)
- Package dependency structure

### Questions to Answer
1. Would consolidation provide significant benefits?
2. What's the right package structure?
3. How do we handle package dependencies?
4. What's the migration path?
5. Is the disruption worth the benefits?

### Recommendation
**DEFER** this issue until:
1. JavaScript consolidation is complete (Issue #1)
2. Team has bandwidth for larger refactor
3. Can be coordinated with other package reorganization efforts

### Next Steps
1. Create discussion issue to gather team input
2. Consider as part of broader architecture review
3. Evaluate after JavaScript utilities are established

### References
- Documentation: `STRING_HANDLING.md`
- Related to Issue #3 (JavaScript utilities)

---

## Summary

### Immediate Action Items
1. **Create Issue #1** (High Priority) - Consolidate JavaScript functions
2. **Create Issue #2** (Medium Priority) - Align Go/JS implementations

### Future Consideration
3. **Create Issue #3** (Low Priority) - Organize JS utilities directory
4. **Create Issue #4** (Discussion) - Evaluate Go package reorganization

### Success Criteria
- [ ] All critical duplications eliminated (Issue #1)
- [ ] Go/JS consistency achieved or documented (Issue #2)
- [ ] Clear documentation maintained in `STRING_HANDLING.md`
- [ ] No test regressions
- [ ] Team has clear guidance on string handling

---

**Created**: 2024-10-30  
**Based on**: String Handling Audit (githubnext/gh-aw#2773)  
**Documentation**: `STRING_HANDLING.md`, `docs/string-handling-consolidation-recommendations.md`
