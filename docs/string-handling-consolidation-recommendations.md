# String Handling Consolidation Recommendations

This document outlines specific recommendations for consolidating string handling functions identified in the comprehensive audit (see `specs/STRING_HANDLING.md`).

## Executive Summary

The audit identified **35 functions** related to string normalization, sanitization, and cleaning across the codebase. Key findings:

- **3 critical duplications** requiring immediate attention (security-related)
- **6 consolidation opportunities** with varying priority
- **2 critical sync requirements** between JavaScript files
- **1 inconsistency** between Go and JavaScript implementations

## Priority 1: Critical Security Duplications

### Issue #1: `sanitizeContent` Triplication (JavaScript)

**Problem**: The `sanitizeContent` function is duplicated identically in three files:
- `pkg/workflow/js/compute_text.cjs`
- `pkg/workflow/js/sanitize_output.cjs`
- `pkg/workflow/js/collect_ndjson_output.cjs`

**Risk**: **HIGH** - Security vulnerabilities or bug fixes must be applied to all three copies. Missing one could leave security holes.

**Current State**: All three appear identical but are copy-pasted ~150 lines of code each.

**Recommendation**: Create shared utility module

```javascript
// pkg/workflow/js/utils/sanitize.cjs

/**
 * Shared sanitization utilities for GitHub Actions content
 * @module utils/sanitize
 */

function sanitizeContent(content) {
  // Move implementation here
  // ...
}

function sanitizeUrlDomains(s, allowedDomains) {
  // Move implementation here
  // ...
}

function sanitizeUrlProtocols(s) {
  // Move implementation here
  // ...
}

function convertXmlTagsToParentheses(s) {
  // Move implementation here
  // ...
}

function neutralizeMentions(s) {
  // Move implementation here
  // ...
}

function neutralizeBotTriggers(s) {
  // Move implementation here
  // ...
}

module.exports = {
  sanitizeContent,
  sanitizeUrlDomains,
  sanitizeUrlProtocols,
  convertXmlTagsToParentheses,
  neutralizeMentions,
  neutralizeBotTriggers,
};
```

Then update all three files to import:
```javascript
const { sanitizeContent } = require('./utils/sanitize.cjs');
```

**Benefits**:
- Single source of truth for security-critical code
- Security fixes apply universally
- Reduced LOC (saves ~300 lines)
- Easier to maintain and test

**Estimated Effort**: 2-3 hours
- Create utility module (1 hour)
- Update imports in 3 files (30 min)
- Update tests (1 hour)
- Verify no regressions (30 min)

**Testing Strategy**:
1. Extract function to utility module
2. Run existing tests to verify no breakage
3. Add tests for utility module itself
4. Test in all three contexts (compute_text, sanitize_output, collect_ndjson_output)

### Issue #2: `sanitizeLabelContent` Duplication (JavaScript)

**Problem**: The `sanitizeLabelContent` function is duplicated identically in:
- `pkg/workflow/js/add_labels.cjs`
- `pkg/workflow/js/create_issue.cjs`

**Risk**: **MEDIUM** - Less critical than `sanitizeContent` but still security-related (XSS in labels)

**Recommendation**: Add to the same utility module created for Issue #1

```javascript
// In pkg/workflow/js/utils/sanitize.cjs
function sanitizeLabelContent(content) {
  // Move implementation here
  // ...
}

module.exports = {
  sanitizeContent,
  sanitizeLabelContent,
  // ... other exports
};
```

**Benefits**:
- Consistency with `sanitizeContent` consolidation
- Single source of truth for label sanitization
- Reduced LOC (saves ~15-20 lines)

**Estimated Effort**: 30 minutes (as part of Issue #1 work)

### Issue #3: `normalizeBranchName` Duplication (JavaScript)

**Problem**: The `normalizeBranchName` function must be kept in sync between:
- `pkg/workflow/js/upload_assets.cjs`
- `pkg/workflow/js/safe_outputs_mcp_server.cjs`

**Current State**: Both files have comments stating "IMPORTANT: Keep this function in sync"

**Risk**: **MEDIUM** - Manual sync requirement is error-prone; divergence would cause inconsistent branch naming

**Recommendation**: Extract to shared utility module

```javascript
// pkg/workflow/js/utils/branch.cjs or utils/git.cjs

/**
 * Normalizes a branch name to be a valid git branch name.
 * 
 * Valid characters: alphanumeric (a-z, A-Z, 0-9), dash (-), underscore (_), 
 * forward slash (/), dot (.)
 * Max length: 128 characters
 * 
 * @param {string} branchName - The branch name to normalize
 * @returns {string} The normalized branch name
 */
function normalizeBranchName(branchName) {
  // Move implementation here
  // ...
}

module.exports = {
  normalizeBranchName,
};
```

**Benefits**:
- Guaranteed consistency (no manual sync required)
- Single point of maintenance
- Clearer code organization

**Estimated Effort**: 30 minutes
- Create utility module (15 min)
- Update imports in 2 files (10 min)
- Verify tests pass (5 min)

## Priority 2: Implementation Inconsistencies

### Issue #4: `SanitizeWorkflowName` Go vs JavaScript Inconsistency

**Problem**: The Go version consolidates multiple hyphens, but the JavaScript version does not:

**Go version** (`pkg/workflow/strings.go`):
```go
// Consolidate multiple consecutive hyphens into a single hyphen
name = multipleHyphens.ReplaceAllString(name, "-")
```

**JavaScript version** (`pkg/workflow/js/parse_firewall_logs.cjs`):
```javascript
function sanitizeWorkflowName(name) {
  return name
    .toLowerCase()
    .replace(/[:\\/\s]/g, "-")
    .replace(/[^a-z0-9._-]/g, "-");
  // Missing: hyphen consolidation
}
```

**Risk**: **LOW-MEDIUM** - Could lead to different workflow names in Go vs JavaScript contexts

**Recommendation**: Add hyphen consolidation to JavaScript version

```javascript
function sanitizeWorkflowName(name) {
  return name
    .toLowerCase()
    .replace(/[:\\/\s]/g, "-")
    .replace(/[^a-z0-9._-]/g, "-")
    .replace(/-+/g, "-");  // Add this line
}
```

**Alternative**: If the JavaScript version intentionally differs (e.g., for firewall log compatibility), document why in comments.

**Estimated Effort**: 15 minutes (plus testing to verify firewall logs aren't affected)

**Investigation Required**: Check if any firewall log parsing depends on the current behavior (multiple hyphens).

## Priority 3: Code Organization

### Issue #5: Create Shared JavaScript Utilities Directory

**Problem**: JavaScript utility functions are scattered across individual files with no shared module structure.

**Recommendation**: Create organized utilities directory

```
pkg/workflow/js/
  utils/
    sanitize.cjs         # sanitizeContent, sanitizeLabelContent, helpers
    branch.cjs           # normalizeBranchName
    workflow.cjs         # sanitizeWorkflowName
    index.cjs            # Re-export all utilities
```

**Benefits**:
- Better code organization
- Easier discoverability
- Natural location for new utility functions
- Clearer separation of concerns

**Estimated Effort**: 3-4 hours (includes moving functions and updating all imports)

### Issue #6: Consider Go String Utilities Package

**Problem**: Go string handling functions are spread across multiple packages:
- `pkg/workflow/strings.go` - `SanitizeWorkflowName`
- `pkg/workflow/workflow_name.go` - `SanitizeIdentifier`
- `pkg/cli/resolver.go` - `NormalizeWorkflowFile`
- etc.

**Recommendation**: Consider consolidating to `pkg/utils/strings/` or similar

**Benefits**:
- Better discoverability
- Clearer organization
- Easier to find right function for a task

**Challenges**:
- Requires updating imports across entire codebase
- May cause merge conflicts with in-flight PRs
- Need to consider package dependencies

**Recommendation**: **DEFER** - This is a larger refactor that should be considered separately, possibly as part of broader package reorganization.

**Estimated Effort**: 4-6 hours

## Implementation Plan

### Phase 1: High Priority Security (Week 1)

1. **Day 1-2**: Implement Issue #1 (`sanitizeContent` consolidation)
   - Create `pkg/workflow/js/utils/sanitize.cjs`
   - Move `sanitizeContent` and all helper functions
   - Update imports in 3 files
   - Run tests and verify

2. **Day 2**: Implement Issue #2 (`sanitizeLabelContent` consolidation)
   - Add to same utility module
   - Update imports in 2 files
   - Run tests and verify

3. **Day 3**: Implement Issue #3 (`normalizeBranchName` consolidation)
   - Create `pkg/workflow/js/utils/branch.cjs`
   - Move function
   - Update imports in 2 files
   - Run tests and verify

### Phase 2: Consistency Improvements (Week 2)

4. **Day 4**: Investigate and fix Issue #4 (Go/JS inconsistency)
   - Research firewall log requirements
   - Update JavaScript implementation if safe
   - Or document intentional difference

5. **Day 5**: Code review and documentation
   - Update `specs/STRING_HANDLING.md` with new structure
   - Add JSDoc comments to utility modules
   - Update any affected documentation

### Phase 3: Optional Reorganization (Future)

6. **Future**: Consider Issue #5 and #6 (broader reorganization)
   - Evaluate as part of larger refactoring initiative
   - Consider team bandwidth and priorities

## Testing Strategy

### For Each Consolidation

1. **Before changes**:
   - Run full test suite: `make test`
   - Document current behavior

2. **After extraction**:
   - Run unit tests for affected modules
   - Run integration tests
   - Verify no regressions

3. **Additional tests**:
   - Add tests for utility modules themselves
   - Test edge cases and security scenarios
   - Verify imports work correctly

### Test Cases to Cover

For `sanitizeContent`:
- @mention neutralization
- XML tag conversion
- URL filtering (protocols and domains)
- Length limits
- ANSI escape removal
- Bot trigger neutralization

For `sanitizeLabelContent`:
- HTML character removal
- @mention neutralization
- Control character removal

For `normalizeBranchName`:
- Invalid character replacement
- Length truncation
- Leading/trailing dashes
- Lowercase conversion

## Success Metrics

- **Code Reduction**: Remove ~350 lines of duplicated code
- **Maintenance Burden**: Reduce from 8 places to update to 3 utility modules
- **Security**: Single source of truth for security-critical sanitization
- **Test Coverage**: Maintain or improve test coverage
- **No Regressions**: All existing tests pass

## Risks and Mitigation

### Risk 1: Breaking Changes

**Mitigation**: 
- Comprehensive testing before/after
- Keep changes minimal (pure extraction, no logic changes)
- Test in isolated branch first

### Risk 2: Import Path Issues

**Mitigation**:
- Use relative paths consistently
- Test in both local development and CI environment
- Document any special requirements

### Risk 3: Merge Conflicts

**Mitigation**:
- Coordinate with team on timing
- Do high-traffic files first
- Communicate in team chat before starting

## Follow-Up Issues to Create

Based on this analysis, create the following issues:

1. **[High Priority] Consolidate JavaScript sanitization functions**
   - Covers Issues #1, #2, #3
   - Estimated effort: 3-4 hours
   - Labels: `security`, `refactor`, `priority-high`

2. **[Medium Priority] Align Go and JavaScript workflow name sanitization**
   - Covers Issue #4
   - Estimated effort: 1 hour
   - Labels: `consistency`, `refactor`, `priority-medium`

3. **[Low Priority] Consider JavaScript utilities directory structure**
   - Covers Issue #5
   - Estimated effort: 3-4 hours
   - Labels: `refactor`, `code-organization`, `priority-low`

4. **[Future] Evaluate Go string utilities package**
   - Covers Issue #6
   - Estimated effort: 4-6 hours
   - Labels: `refactor`, `code-organization`, `discussion`

## Conclusion

This audit identified significant opportunities for improvement, particularly in consolidating security-critical sanitization functions. The recommended phased approach prioritizes security and consistency while deferring larger organizational changes to future work.

Implementing Priority 1 recommendations (Issues #1-3) will:
- Reduce security risk by consolidating critical functions
- Eliminate manual sync requirements
- Reduce code duplication by ~350 lines
- Improve maintainability

Total estimated effort for Priority 1: **3-4 hours**

---

**Document Version**: 1.0  
**Date**: 2024-10-30  
**Related**: `specs/STRING_HANDLING.md`
