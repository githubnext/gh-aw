# Safe Output Handler Manager Refactoring - Implementation Guide

## Status: Foundation Complete ✅

The core infrastructure for the handler manager refactoring is complete and tested:
- Handler manager that loads config and dispatches to factory-based handlers
- Example handler demonstrating the factory pattern
- Integration tests proving the architecture works

## Completed Work

### 1. Handler Manager (`safe_output_handler_manager.cjs`)
**Purpose**: Central orchestrator that:
- Loads `GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG` (JSON with handler configs)
- Loads agent output items from `GH_AW_AGENT_OUTPUT`
- Initializes handler factories for each enabled type
- Processes messages sequentially
- Maintains shared temporary ID map across handlers

**Tests**: 9 passing tests in `safe_output_handler_manager.test.cjs`

### 2. Factory Pattern Example (`example_handler.cjs`)
**Demonstrates**:
- `async function main(config = {})` - Initialize with config, return message processor
- Message processor: `async function(outputItem, resolvedTemporaryIds)`
- Return value: `{ temporaryId?, repo, number }` for tracking

**Tests**: 6 passing tests in `example_handler.test.cjs`

## Remaining Work

### Phase 2-5: Convert Handlers to Factory Pattern (4-6 hours)

Each handler needs refactoring to match this pattern:

```javascript
// OLD PATTERN (current)
async function main() {
  const result = loadAgentOutput();  // Loads ALL items
  const items = result.items.filter(i => i.type === "my_type");
  // Process all items
  for (const item of items) {
    // ... process item ...
  }
}

// NEW PATTERN (target)
async function main(config = {}) {
  // Extract config from parameter instead of env vars
  const { max, expires, allowed } = config;
  
  // Return message processor function
  return async function(outputItem, resolvedTemporaryIds) {
    // Process SINGLE item
    // ... process outputItem ...
    
    // Return result with temporary ID if applicable
    return { temporaryId, repo, number };
  };
}
```

#### Phase 2: Simple Handlers (~2 hours)
- [ ] `add_labels.cjs` - Uses `processSafeOutput` wrapper
- [ ] `close_issue.cjs` - Uses `close_entity_helpers`
- [ ] `close_discussion.cjs` - Uses shared close logic

**Challenge**: These use shared helper functions that also need refactoring.

#### Phase 3: Medium Complexity (~1-2 hours)
- [ ] `add_comment.cjs` - Comment handling with hiding/replacing
- [ ] `create_discussion.cjs` - Discussion creation

#### Phase 4: Complex Handler (~2 hours)
- [ ] `create_issue.cjs` - 355 lines with:
  - Sub-issue linking logic
  - Cross-repository support
  - Temporary ID generation and resolution
  - Parent-child relationships

**Critical**: This handler generates temporary IDs that other handlers consume.

#### Phase 5: Update Factory (~1 hour)
- [ ] `update_runner.cjs` - Factory that generates update handlers
- [ ] `update_issue.cjs` - Issue updates
- [ ] `update_discussion.cjs` - Discussion updates

### Phase 6: Update Go Compiler (1 hour)

#### Current Architecture
Each handler gets individual environment variables:
```go
// compiler_safe_outputs_issues.go
customEnvVars = append(customEnvVars, buildTitlePrefixEnvVar("GH_AW_ISSUE_TITLE_PREFIX", cfg.TitlePrefix)...)
customEnvVars = append(customEnvVars, buildLabelsEnvVar("GH_AW_ISSUE_LABELS", cfg.Labels)...)
customEnvVars = append(customEnvVars, buildLabelsEnvVar("GH_AW_ISSUE_ALLOWED_LABELS", cfg.AllowedLabels)...)
// ... 20+ more env vars
```

#### Target Architecture
Single JSON config for all handlers:
```go
handlerConfig := map[string]interface{}{
  "create_issue": map[string]interface{}{
    "max": cfg.Max,
    "expires": cfg.Expires,
    "allowedLabels": cfg.AllowedLabels,
    "titlePrefix": cfg.TitlePrefix,
  },
  "add_comment": map[string]interface{}{
    "max": cfg.Max,
  },
  // ... other handlers
}
configJSON, _ := json.Marshal(handlerConfig)
envVars = append(envVars, fmt.Sprintf("GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG: %s\n", configJSON))
```

#### Files to Modify
1. `pkg/workflow/compiler_safe_outputs_core.go` - Remove individual step generation, create single handler manager step
2. `pkg/workflow/compiler_safe_outputs_issues.go` - Convert to config JSON generation
3. `pkg/workflow/compiler_safe_outputs_discussions.go` - Convert to config JSON generation
4. `pkg/workflow/compiler_safe_outputs_shared.go` - Helper functions for JSON config

### Phase 7: Update Tests (2-3 hours)

#### Test Strategy
1. Update handler tests to use factory pattern
2. Mock `main(config)` to return test processor function
3. Test individual message processing instead of batch processing
4. Update integration tests

#### Example Test Migration
```javascript
// OLD
it("should process multiple items", async () => {
  setAgentOutput({ items: [item1, item2] });
  process.env.GH_AW_ALLOWED = "bug";
  await main();
  expect(mockGithub.rest.issues.create).toHaveBeenCalledTimes(2);
});

// NEW
it("should process multiple items", async () => {
  const processor = await main({ allowed: ["bug"] });
  await processor(item1, tempIdMap);
  await processor(item2, tempIdMap);
  expect(mockGithub.rest.issues.create).toHaveBeenCalledTimes(2);
});
```

### Phase 8: Build and Recompile (30 minutes)

```bash
# Rebuild binary with updated compiler
make build

# Recompile all workflow .lock.yml files
make recompile

# Verify workflows
./gh-aw compile --validate --verbose --stats
```

Expected changes in `.lock.yml` files:
- Replace 8+ individual safe output steps with single "Process Safe Outputs" step
- Replace 30+ environment variables with 2:
  - `GH_AW_SAFE_OUTPUTS_STAGED: "true/false"`
  - `GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG: '{"create_issue":{...},"add_comment":{...}}'`

## Key Considerations

### 1. Backward Compatibility
During transition, handlers may need to support BOTH patterns:
- Check if called with config parameter (new) or env vars (old)
- Dual-mode until all handlers are migrated

### 2. Staged Mode
Handler manager already checks `GH_AW_SAFE_OUTPUTS_STAGED`.
Individual handlers need to handle staged preview generation.

### 3. Temporary ID Resolution
- `create_issue` generates temporary IDs: `#aw_123456789abc`
- Other handlers (e.g., `link_sub_issue`, `add_comment`) resolve them
- Handler manager maintains shared `resolvedTemporaryIds` map
- Critical that processing order is maintained

### 4. Error Handling
- Handler loading errors don't stop other handlers
- Message processing errors are logged but don't stop workflow
- Final summary shows processed count and error count

### 5. Testing Strategy
- Test handler factories in isolation
- Test message processors with mock data
- Integration test through handler manager
- Validate compiled workflows

## Risks and Mitigation

### Risk: Breaking Existing Functionality
**Mitigation**: 
- Incremental migration with dual-mode support
- Comprehensive test coverage
- Validate each handler before moving to next

### Risk: Temporary ID Resolution Issues
**Mitigation**:
- Carefully test `create_issue` handler first
- Verify temporary ID map propagation
- Test cross-references in comments/sub-issues

### Risk: Configuration Mismatch
**Mitigation**:
- Schema validation for handler config JSON
- Clear error messages for missing/invalid config
- Document expected config structure per handler

## Next Steps

1. **Phase 2**: Start with `close_issue.cjs` (simplest, no sub-dependencies)
2. **Test thoroughly** after each handler conversion
3. **Document** any configuration schema changes
4. **Phase 6**: Update Go compiler only after all handlers work
5. **Phase 8**: Recompile and validate all workflows

## Success Criteria

- ✅ All 8 handlers use factory pattern
- ✅ No individual env vars except `STAGED` and `HANDLER_CONFIG`
- ✅ All tests pass
- ✅ 126 workflows compile successfully
- ✅ Workflows use single "Process Safe Outputs" step
- ✅ Handler manager loads and dispatches correctly

## Time Estimate

- Phases 2-5 (Handlers): 4-6 hours
- Phase 6 (Go Compiler): 1 hour
- Phase 7 (Tests): 2-3 hours  
- Phase 8 (Build/Recompile): 30 minutes

**Total: 8-10.5 hours**
