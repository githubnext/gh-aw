# create-pull-request Handler Manager Conversion Plan

## Overview

This document outlines the complete plan for converting `create_pull_request.cjs` from a standalone script to the handler manager architecture.

## Status

**Infrastructure**: ✅ Complete
**JavaScript Conversion**: ❌ Not Started (requires careful systematic approach)

## Infrastructure Changes (Completed)

### Handler Manager (`safe_output_handler_manager.cjs`)
- ✅ Added `create_pull_request: "./create_pull_request.cjs"` to `HANDLER_MAP`
- ✅ Removed `"create_pull_request"` from `STANDALONE_STEP_TYPES` Set

### Go Compiler (`compiler_safe_outputs_prs.go`)
- ✅ Already passes `GH_AW_TEMPORARY_ID_MAP` environment variable (from earlier commits)

## JavaScript Conversion Plan

### File: `actions/setup/js/create_pull_request.cjs`

**Current State**: 705 lines, standalone script format
**Target State**: Handler factory pattern with message handler function

### Conversion Steps

#### Step 1: Add Type Definitions
```javascript
/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "create_pull_request";
```

#### Step 2: Convert Main Function Signature
```javascript
// OLD
async function main() {
  
// NEW
/**
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
```

#### Step 3: Extract Configuration
```javascript
// Extract from config parameter instead of environment variables
const titlePrefix = config.title_prefix || "";
const envLabels = config.labels ? (Array.isArray(config.labels) ? config.labels : config.labels.split(",")).map(label => String(label).trim()).filter(label => label) : [];
const draftDefault = config.draft !== undefined ? config.draft : true;
const ifNoChanges = config.if_no_changes || "warn";
const allowEmpty = config.allow_empty || false;
const expiresHours = config.expires ? parseInt(String(config.expires), 10) : 0;
const maxCount = config.max || 1;
```

#### Step 4: Remove Agent Output File Reading
```javascript
// REMOVE these lines (~40 lines, 70-110):
const agentOutputFile = process.env.GH_AW_AGENT_OUTPUT || "";
let outputContent = "";
// ... file reading logic ...
// ... JSON parsing logic ...
// ... finding pullRequestItem from JSON ...
```

The message is passed directly to the handler function.

#### Step 5: Wrap Logic in Message Handler
```javascript
// Track processed count for max limit
let processedCount = 0;

return async function handleCreatePullRequest(message, resolvedTemporaryIds) {
  // Check max limit
  if (processedCount >= maxCount) {
    core.warning(`Skipping create_pull_request: max count of ${maxCount} reached`);
    return {
      success: false,
      error: `Max count of ${maxCount} reached`,
    };
  }
  
  processedCount++;
  
  const pullRequestItem = message; // Message IS the pullRequestItem
  
  // ... rest of logic ...
}
```

#### Step 6: Convert Error Handling

**Pattern 1**: `throw new Error` in factory initialization (lines 60, 65)
- **Keep as-is** - these are factory initialization errors, not message processing errors

**Pattern 2**: `throw new Error` in message processing
```javascript
// OLD (7 locations: lines 111, 156, 193, 204, 583, etc.)
throw new Error(message);

// NEW
return { success: false, error: message };
```

**Pattern 3**: `core.setFailed` + `return`
```javascript
// OLD (6 locations: lines 76, 229, 427, 532, 574, 699)
core.setFailed("Error message");
return;

// NEW
return { success: false, error: "Error message" };
```

**Pattern 4**: Bare `return` statements
```javascript
// OLD
return; // Silent exit

// NEW
return { success: false, error: "...", skipped: true };
```

**Pattern 5**: Staged mode returns
```javascript
// OLD
await core.summary.addRaw(summaryContent).write();
return;

// NEW
await core.summary.addRaw(summaryContent).write();
return { success: true, staged: true, title, branch };
```

#### Step 7: Remove core.setOutput Calls

Replace ~18 `core.setOutput` calls with returning values in result object:

```javascript
// OLD
core.setOutput("pull_request_number", pr.number);
core.setOutput("pull_request_url", pr.html_url);
core.setOutput("branch_name", branchName);

// NEW (at end of function)
return {
  success: true,
  pull_request_number: pr.number,
  pull_request_url: pr.html_url,
  branch_name: branchName,
  temporary_id: pullRequestItem.temporary_id, // if present
};
```

For fallback to issue:
```javascript
// OLD  
core.setOutput("issue_number", issue.number);
core.setOutput("issue_url", issue.html_url);
core.setOutput("fallback_used", "true");

// NEW
return {
  success: true,
  fallback_used: true,
  issue_number: issue.number,
  issue_url: issue.html_url,
  branch_name: branchName,
};
```

#### Step 8: Update Configuration Reading

Replace environment variable reads with config parameter:

```javascript
// OLD
const titlePrefix = process.env.GH_AW_PR_TITLE_PREFIX;
const labelsEnv = process.env.GH_AW_PR_LABELS;
const draftEnv = process.env.GH_AW_PR_DRAFT;

// NEW (already extracted at factory level)
// Use titlePrefix, envLabels, draftDefault from config
```

Merge message-level config with factory config:
```javascript
// Apply title prefix
if (titlePrefix && !title.startsWith(titlePrefix)) {
  title = titlePrefix + title;
}

// Merge labels
let labels = [...envLabels];
if (pullRequestItem.labels && Array.isArray(pullRequestItem.labels)) {
  labels = [...labels, ...pullRequestItem.labels];
}

// Use draft setting from message if provided, otherwise factory default
const draft = pullRequestItem.draft !== undefined ? pullRequestItem.draft : draftDefault;
```

#### Step 9: Handle Temporary ID Resolution

The temporary ID resolution is already implemented correctly:
```javascript
let processedBody = pullRequestItem.body || "";
if (resolvedTemporaryIds && Object.keys(resolvedTemporaryIds).length > 0) {
  const tempIdMap = new Map(Object.entries(resolvedTemporaryIds));
  const currentRepo = `${context.repo.owner}/${context.repo.repo}`;
  processedBody = replaceTemporaryIdReferences(processedBody, tempIdMap, currentRepo);
  core.info(`Resolved ${tempIdMap.size} temporary ID references in PR body`);
}
```

Just ensure `resolvedTemporaryIds` parameter is used instead of `GH_AW_TEMPORARY_ID_MAP` env var.

## Testing Plan

### Unit Tests
1. Test handler factory returns a function
2. Test max count limiting
3. Test configuration extraction
4. Test error result objects
5. Test temporary ID resolution

### Integration Tests
1. Test full PR creation workflow
2. Test fallback to issue creation
3. Test empty patch handling with allow-empty
4. Test patch size validation
5. Test staged mode
6. Test all error paths return proper objects

### Manual Testing
1. Create workflow with create-issue and create-pull-request
2. Verify PR can reference issue using temporary ID
3. Verify PR creation succeeds
4. Verify fallback to issue works
5. Verify staged mode shows preview

## Estimated Effort

**Time Required**: 3-4 hours
**Complexity**: High
**Risk**: Medium (extensive git operations and fallback logic must be preserved)

## Success Criteria

- [ ] All syntax errors resolved
- [ ] All error handling converted to return result objects
- [ ] All core.setOutput calls removed
- [ ] Handler factory pattern implemented correctly
- [ ] All existing tests pass
- [ ] New tests added for handler pattern
- [ ] Manual testing confirms functionality
- [ ] Git operations work correctly
- [ ] Fallback logic works correctly
- [ ] Temporary ID resolution works correctly
- [ ] Staged mode works correctly

## Known Challenges

1. **File Size**: 705 lines makes this the largest handler conversion
2. **Git Operations**: Extensive git CLI operations that must work identically
3. **Fallback Logic**: Complex fallback from PR to issue creation
4. **Error Paths**: 31 error handling locations to convert
5. **Outputs**: 18 core.setOutput calls to remove and convert to return values
6. **Testing**: Must verify git operations don't break

## References

- ✅ `create_issue.cjs` - Example of completed handler conversion (356 lines)
- ✅ `safe_output_handler_manager.cjs` - Handler manager implementation
- ✅ `types/handler-factory.d.ts` - TypeScript types for handler factory
