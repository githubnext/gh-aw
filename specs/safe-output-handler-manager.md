# Safe Output Handler Manager Implementation

## Overview

This document describes the safe output handler manager implementation, which refactors the safe output processing system from multiple individual steps to a single dispatcher step.

## Architecture

### Current Multi-Step Approach

Each safe output type (create_issue, add_comment, etc.) generates a separate step in the GitHub Actions workflow:

```yaml
jobs:
  safe_outputs:
    steps:
      - name: Setup Scripts
      - name: Download agent output
      - name: Create Issue
        if: contains(fromJSON(env.GH_AW_AGENT_OUTPUT).items[*].type, 'create_issue')
        uses: actions/github-script@...
        with:
          script: |
            const { main } = require('./create_issue.cjs');
            await main();
      - name: Add Comment
        if: contains(fromJSON(env.GH_AW_AGENT_OUTPUT).items[*].type, 'add_comment')
        uses: actions/github-script@...
        with:
          script: |
            const { main } = require('./add_comment.cjs');
            await main();
      # ... more steps for each safe output type
```

### New Handler Manager Approach

A single step uses the handler manager to dispatch to appropriate handlers:

```yaml
jobs:
  safe_outputs:
    steps:
      - name: Setup Scripts
      - name: Download agent output
      - name: Process Safe Outputs
        uses: actions/github-script@...
        with:
          script: |
            const { main } = require('./safe_output_handler_manager.cjs');
            await main();
```

## Implementation Details

### Handler Manager (`safe_output_handler_manager.cjs`)

The handler manager is responsible for:

1. **Configuration Loading**:
   ```javascript
   function loadConfig() {
     // Read from /tmp/gh-aw/safeoutputs/config.json
     // Normalize keys (create-issue → create_issue)
     return config;
   }
   ```

2. **Handler Registration**:
   ```javascript
   const HANDLER_MAP = {
     create_issue: "./create_issue.cjs",
     add_comment: "./add_comment.cjs",
     create_discussion: "./create_discussion.cjs",
     close_issue: "./close_issue.cjs",
     close_discussion: "./close_discussion.cjs",
   };
   
   function loadHandlers(config) {
     // Only load handlers for enabled types
     // Store in Map<string, {main: Function}>
   }
   ```

3. **Message Processing**:
   ```javascript
   function processMessages(handlers, messages) {
     // Group messages by type
     const grouped = groupMessagesByType(messages);
     
     // Process in dependency order
     const order = [
       "create_issue",
       "create_discussion", 
       "create_pull_request",
       "add_comment",  // Must be after creates
       "close_issue",
       "close_discussion",
     ];
     
     // Dispatch to each handler
     for (const type of order) {
       if (handlers.has(type) && grouped.has(type)) {
         await handlers.get(type).main();
       }
     }
   }
   ```

### Go Compiler Integration

The Go compiler generates the handler manager step:

```go
func (c *Compiler) buildHandlerManagerStep(data *WorkflowData) []string {
    var steps []string
    
    steps = append(steps, "      - name: Process Safe Outputs\n")
    steps = append(steps, "        id: process_safe_outputs\n")
    steps = append(steps, fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script")))
    
    // Add environment variables for all enabled handlers
    c.addAllSafeOutputConfigEnvVars(&steps, data)
    
    // Call handler manager
    steps = append(steps, "          script: |\n")
    steps = append(steps, "            const { main } = require('"+SetupActionDestination+"/safe_output_handler_manager.cjs');\n")
    steps = append(steps, "            await main();\n")
    
    return steps
}
```

## Handler Compatibility

All existing handlers are compatible without modification because they:

1. **Export a `main()` function**:
   ```javascript
   async function main() {
     // Handler logic
   }
   module.exports = { main };
   ```

2. **Access agent output internally**:
   ```javascript
   const result = loadAgentOutput();
   const items = result.items.filter(item => item.type === "create_issue");
   ```

3. **Read configuration from environment and config file**:
   ```javascript
   const titlePrefix = process.env.GH_AW_ISSUE_TITLE_PREFIX;
   const config = getSafeOutputConfig("create_issue");
   ```

4. **Set outputs independently**:
   ```javascript
   core.setOutput("issue_number", issueNumber);
   core.setOutput("issue_url", issueUrl);
   ```

## Benefits

### 1. Reduced Workflow Complexity
- **Before**: N steps (one per safe output type)
- **After**: 1 step (handler manager)
- Easier to read and understand workflow YAML

### 2. Better Temporary ID Management
- Shared temporary ID map across all handlers
- Consistent ID resolution for cross-references
- Simpler debugging of ID-related issues

### 3. Enforced Processing Order
- Manager ensures correct dependency order
- Prevents issues like add_comment running before create_issue
- Centralized ordering logic

### 4. Easier Extensibility
- Add new handlers by updating `HANDLER_MAP`
- No Go code changes needed for new handler types
- Handlers remain isolated and testable

### 5. Improved Error Handling
- Centralized error reporting
- Handlers can fail independently
- Better logging and diagnostics

## Testing

### Unit Tests (`safe_output_handler_manager.test.cjs`)

Tests cover:
- Configuration loading and normalization
- Handler registration based on configuration
- Message grouping by type
- Processing order enforcement
- Error handling and recovery

### Integration Testing

To test the handler manager:

1. Create a workflow with multiple safe output types
2. Compile the workflow
3. Verify single "Process Safe Outputs" step is generated
4. Run the workflow and verify correct behavior
5. Check that outputs match existing implementation

## Migration Path

### Phase 1: Foundation (Current)
- [x] Implement handler manager
- [x] Add tests
- [x] Add Go compiler functions

### Phase 2: Integration (Next)
- [ ] Modify `buildConsolidatedSafeOutputsJob` to use handler manager
- [ ] Keep complex operations (PRs, assets) as separate steps
- [ ] Run integration tests

### Phase 3: Validation
- [ ] Test with real workflows
- [ ] Compare outputs with existing implementation
- [ ] Performance testing

### Phase 4: Cleanup
- [ ] Remove old multi-step approach
- [ ] Update documentation
- [ ] Release

## Configuration Example

```json
{
  "create-issue": {
    "enabled": true,
    "max": 5,
    "title-prefix": "[AI]",
    "labels": ["ai-generated"]
  },
  "add-comment": {
    "enabled": true,
    "max": 3,
    "target": "triggering"
  },
  "create-discussion": {
    "enabled": true
  },
  "close-issue": {
    "enabled": false
  }
}
```

With this configuration, the handler manager will:
1. Load handlers for create_issue, add_comment, and create_discussion
2. Skip close_issue (disabled)
3. Process messages in order: create_issue → add_comment → create_discussion

## Performance Considerations

- **Handler Loading**: Handlers are loaded once at startup, not per message
- **Message Grouping**: O(n) operation to group messages by type
- **Processing**: Each handler processes all its messages in one call
- **Memory**: Temporary ID map is maintained in memory for the duration of the step

## Future Enhancements

1. **Parallel Processing**: Some handlers could run in parallel
2. **Retry Logic**: Add retry for transient failures
3. **Metrics**: Collect processing metrics per handler
4. **Dynamic Loading**: Load handlers on-demand rather than upfront
5. **Handler Plugins**: Support external handler modules

## Conclusion

The handler manager implementation provides a cleaner, more maintainable architecture for safe output processing while maintaining full backward compatibility with existing handlers.
