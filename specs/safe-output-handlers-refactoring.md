# Safe Output Handlers Refactoring Status

## Objective
Refactor all safe output handlers to follow the handler factory pattern where `main(config)` returns a message handler function.

## Pattern
**Old:** `main()` loads all items via `loadAgentOutput()` and processes them in a loop
**New:** `main(config)` returns `async function(message, resolvedTemporaryIds)` that processes ONE message

## Completed (4/10) ✅

### 1. create_issue.cjs
- **Status:** ✅ Refactored (commit 78faadf, a637c3e)
- **Size:** ~350 lines
- **Complexity:** Medium - handles temporary IDs, parent linking, sub-issues
- **Tests:** New test suite created (9 passing tests)

### 2. close_issue.cjs  
- **Status:** ✅ Refactored (commit da59b5e)
- **Size:** ~200 lines
- **Complexity:** Low - validates labels/title, closes issue
- **Tests:** Existing tests need migration

### 3. link_sub_issue.cjs
- **Status:** ✅ Already using new pattern
- **Size:** ~300 lines
- **Pattern:** Already returns message handler function

### 4. update_release.cjs
- **Status:** ✅ Already using new pattern
- **Size:** ~200 lines  
- **Pattern:** Already returns message handler function

## Remaining (6/10) 🔄

### 5. close_discussion.cjs
- **Status:** 🔄 Needs refactoring
- **Size:** 359 lines
- **Complexity:** Medium - GraphQL API, validates labels/title/category
- **Dependencies:** Uses loadAgentOutput, GraphQL operations
- **Estimate:** 2-3 hours

### 6. add_labels.cjs
- **Status:** 🔄 Needs refactoring  
- **Size:** 128 lines
- **Complexity:** Medium - uses processSafeOutput helper
- **Dependencies:** processSafeOutput, validateLabels
- **Estimate:** 1-2 hours

### 7. add_comment.cjs
- **Status:** 🔄 Needs refactoring
- **Size:** 585 lines
- **Complexity:** High - comment tracking, mentions, multiple contexts
- **Dependencies:** Uses loadAgentOutput, complex logic
- **Estimate:** 4-5 hours

### 8. create_discussion.cjs
- **Status:** 🔄 Needs refactoring
- **Size:** 356 lines
- **Complexity:** High - GraphQL, category resolution, repo validation
- **Dependencies:** Uses loadAgentOutput, GraphQL operations
- **Estimate:** 3-4 hours

### 9. update_issue.cjs
- **Status:** 🔄 Needs refactoring
- **Size:** 49 lines (uses update_runner.cjs factory)
- **Complexity:** High - requires refactoring shared 445-line factory
- **Dependencies:** createUpdateHandler factory, update_runner.cjs
- **Estimate:** 5-6 hours (includes factory refactor)

### 10. update_discussion.cjs
- **Status:** 🔄 Needs refactoring
- **Size:** 300 lines (uses update_runner.cjs factory)
- **Complexity:** High - requires refactoring shared factory
- **Dependencies:** createUpdateHandler factory, update_runner.cjs
- **Estimate:** Same as #9 (factory refactor covers both)

## Total Effort Estimate
- Completed: ~6-8 hours
- Remaining: ~15-20 hours

## Recommended Approach

### Phase 1 (Simpler handlers)
1. close_discussion.cjs - Similar to close_issue
2. add_labels.cjs - Smaller, clearer logic

### Phase 2 (Complex handlers)
3. create_discussion.cjs - GraphQL operations
4. add_comment.cjs - Large, complex logic

### Phase 3 (Factory refactor)
5. update_runner.cjs - Refactor the 445-line factory
6. update_issue.cjs - Test with refactored factory
7. update_discussion.cjs - Test with refactored factory

## Testing Strategy
- Each refactored handler needs new tests following the pattern in create_issue_new_arch.test.cjs
- Old tests can be migrated or kept as integration tests
- Handler manager integration should be tested end-to-end

## Notes
- All handlers must return a function that accepts (message, resolvedTemporaryIds)
- State must be maintained in closure (processedCount, temporaryIdMap, etc.)
- Return objects must include success/error status
- Max count limits must be enforced in closure
