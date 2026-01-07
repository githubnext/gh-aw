# Safe Output Handlers Refactoring Status

## ✅ COMPLETE - 100% Refactored

All safe output handlers have been successfully refactored to follow the handler factory pattern where `main(config)` returns a message handler function.

## Architecture Overview

Safe outputs are processed through two distinct patterns:

### 1. Handler Manager Pattern (Default)
Most safe outputs are processed through `safe_output_handler_manager.cjs`:
- Handler exports `main(config)` which returns a message handler function
- Handler manager calls the handler once per message
- Examples: `create_issue`, `add_comment`, `create_discussion`, `create_project_status_update`

### 2. Standalone Step Pattern
Some safe outputs require dedicated standalone steps in the workflow:
- Each safe output gets its own step in the `safe_outputs` job
- Direct script execution without handler manager
- Used when requiring special environment variables, tokens, or permissions

**Standalone step types** (defined in `safe_output_handler_manager.cjs`):
- `assign_to_agent` - Agent assignment operations
- `create_agent_task` - Creates GitHub Copilot agent tasks
- `update_project` - Updates GitHub Projects v2 (requires PAT/GitHub App token)
- `copy_project` - Copies GitHub Projects v2 structure
- `upload_asset` - Uploads files to orphaned branches
- `noop` - No-operation placeholder

**Why use standalone steps?**
- Custom environment variables needed (e.g., `GH_AW_COPY_PROJECT_SOURCE`)
- Special token requirements (Projects v2 requires PAT, not GITHUB_TOKEN)
- Specialized permissions (organization-projects write)
- Git operations requiring repository checkout (upload_asset)

The handler manager silently skips messages with standalone step types without logging warnings, since they're handled by their dedicated steps.

## Pattern
**Old:** `main()` loads all items via `loadAgentOutput()` and processes them in a loop
**New:** `main(config)` returns `async function(message, resolvedTemporaryIds)` that processes ONE message

## Completed (11/11) ✅

### 1. create_issue.cjs
- **Status:** ✅ Refactored (commit a637c3e)
- **Complexity:** Medium - handles temporary IDs, parent linking, sub-issues
- **Tests:** New test suite created (9 passing tests)

### 2. close_issue.cjs  
- **Status:** ✅ Refactored (commit da59b5e)
- **Complexity:** Low - validates labels/title, closes issue

### 3. link_sub_issue.cjs
- **Status:** ✅ Already using new pattern
- **Pattern:** Already returns message handler function

### 4. update_release.cjs
- **Status:** ✅ Already using new pattern
- **Pattern:** Already returns message handler function

### 5. add_labels.cjs
- **Status:** ✅ Refactored (commit 504982b)
- **Complexity:** Medium - validates and adds labels
- **Change:** Removed dependency on processSafeOutput helper

### 6. close_discussion.cjs
- **Status:** ✅ Refactored (commit 504982b)
- **Complexity:** Medium - GraphQL operations, validates filters

### 7. create_discussion.cjs
- **Status:** ✅ Refactored (commit cd7bd01)
- **Complexity:** High - GraphQL, category resolution, repo caching

### 8. add_comment.cjs
- **Status:** ✅ Refactored (commit ed08e3d)
- **Complexity:** High - multi-context (issues/PRs/discussions), hides older comments
- **Change:** Simplified from 585 to 438 lines

### 9. update_issue.cjs
- **Status:** ✅ Refactored (commit fd68469)
- **Complexity:** Medium - standalone implementation
- **Change:** Replaced factory pattern with direct implementation (49 → 148 lines)

### 10. update_discussion.cjs
- **Status:** ✅ Refactored (commit fd68469)
- **Complexity:** Medium - GraphQL-based updates
- **Change:** Standalone implementation (300 → 172 lines)

### 11. mark_pull_request_as_ready_for_review.cjs
- **Status:** ✅ Refactored
- **Complexity:** Medium - validates draft status, updates PR, adds comment
- **Change:** Converted from loop-based to handler factory pattern

## Implementation Summary

### Phase 1 (Simpler handlers) ✅
1. close_issue.cjs
2. add_labels.cjs
3. close_discussion.cjs

### Phase 2 (Complex handlers) ✅
4. create_discussion.cjs
5. add_comment.cjs

### Phase 3 (Update handlers) ✅
6. update_issue.cjs
7. update_discussion.cjs

## Handler Factory Pattern

All handlers now follow this architecture:

```javascript
async function main(config = {}) {
  // 1. Extract configuration
  const maxCount = config.max || 10;
  
  // 2. Initialize state in closure
  let processedCount = 0;
  const caches = new Map();
  
  // 3. Return message handler function
  return async function handleMessage(message, resolvedTemporaryIds) {
    // Check max count
    if (processedCount >= maxCount) {
      return { success: false, error: "Max count reached" };
    }
    processedCount++;
    
    // Process the single message
    // ...
    
    // Return result with status
    return { success: true, ...result };
  };
}
```

## Key Benefits

1. **Message-by-message processing** - Handler manager calls handler once per message
2. **State management** - Closures maintain state (count, caches, temporary IDs)
3. **Max count enforcement** - Each handler enforces its own limits
4. **Temporary ID resolution** - Shared map passed between handlers
5. **Consistent interface** - All handlers follow same pattern
6. **Error handling** - Standardized result format with success/error

## Testing Strategy

Each handler can be tested independently:

```javascript
const { main } = require("./handler.cjs");
const handler = await main({ max: 10, labels: ["bug"] });
const result = await handler(message, resolvedTemporaryIds);
expect(result.success).toBe(true);
```

## Validation

All handlers are compatible with `safe_output_handler_manager.cjs` which:
1. Calls `main(config)` to get handler function
2. Calls handler function for each message: `handler(message, resolvedTemporaryIds)`
3. Collects results and manages temporary ID map
4. Reports errors and tracks outputs

## Impact

This refactoring resolves the architectural incompatibility where handlers were being skipped because they didn't return functions. All safe output operations now work correctly with the handler manager's message processing architecture.
