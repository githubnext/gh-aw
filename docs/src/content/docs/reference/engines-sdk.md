---
title: Copilot SDK Engine
description: Complete guide to the Copilot SDK engine mode for advanced agentic workflows with multi-turn conversations, custom tools, and event handling.
sidebar:
  order: 601
---

The Copilot SDK engine provides advanced capabilities for building sophisticated agentic workflows with multi-turn conversations, custom inline tools, real-time event handling, and multi-agent orchestration.

> [!NOTE]
> SDK Mode Availability
> The SDK engine mode is part of the Copilot SDK integration feature. Ensure your gh-aw version supports SDK mode before using these features.

## What is SDK Mode?

SDK mode uses the [GitHub Copilot SDK](https://github.com/github/copilot-sdk) for programmatic control over agent interactions. Unlike the traditional CLI mode which processes workflows in a single pass, SDK mode enables:

- **Multi-turn conversations**: Maintain context across multiple agent interactions
- **Custom inline tools**: Define workflow-specific tools without external MCP servers
- **Event streaming**: Real-time monitoring and control of agent execution
- **Programmatic control**: Retry logic, branching, and dynamic workflow adaptation
- **Multi-agent orchestration**: Coordinate multiple specialized agents
- **Budget controls**: Cost-aware execution with token usage monitoring

## SDK vs CLI Comparison

| Feature | CLI Mode | SDK Mode |
|---------|----------|----------|
| **Execution Model** | Single-pass | Multi-turn conversations |
| **Context Retention** | Per-job only | Persistent across turns |
| **Custom Tools** | MCP servers only | Inline tool definitions |
| **Event Handling** | Limited (job outputs) | Real-time streaming |
| **Control Flow** | Static workflow | Dynamic, programmatic |
| **Multi-Agent** | Sequential jobs | Coordinated orchestration |
| **Maturity** | Stable, production-ready | Experimental |
| **Performance** | Optimized for simple tasks | Optimized for complex interactions |

## When to Use SDK Mode

**Use SDK mode when you need:**

✅ Multi-turn conversations with context retention across interactions  
✅ Custom inline tools with workflow-specific business logic  
✅ Real-time event handling and streaming responses  
✅ Multi-agent orchestration with shared context  
✅ Programmatic control flow (retry, branching, conditional logic)  
✅ Cost-aware execution with budget controls  
✅ Dynamic tool generation based on workflow state

**Use CLI mode when you need:**

✅ Simple single-pass workflows  
✅ No conversation context required  
✅ Standard MCP tools are sufficient  
✅ Battle-tested stability for production  
✅ Simpler configuration and setup  
✅ Faster execution for straightforward tasks

## Basic Configuration

### Minimal SDK Configuration

```yaml
---
engine:
  id: copilot
  mode: sdk
---
# Your workflow instructions
Process the issue and provide recommendations.
```

### Extended SDK Configuration

```yaml
---
engine:
  id: copilot
  mode: sdk
  version: latest
  model: gpt-5
  session:
    persistent: true
    max-turns: 10
    state-size-limit: 50000
  budget:
    max-tokens: 100000
    warn-threshold: 80000
  events:
    streaming: true
    handlers:
      - on: token_usage
        action: log
tools:
  github:
    allowed: [issue_read, add_issue_comment]
---
# Your workflow instructions
```

## Configuration Options

### Session Configuration

Control conversation state and persistence:

```yaml
engine:
  id: copilot
  mode: sdk
  session:
    persistent: true          # Maintain state across turns
    max-turns: 10            # Maximum conversation turns
    state-size-limit: 50000  # Max state size in bytes
    restore: auto            # auto, manual, or disabled
```

**Session Options:**
- `persistent` (boolean): Enable state persistence between turns
- `max-turns` (integer): Maximum number of conversation turns (default: 5)
- `state-size-limit` (integer): Maximum state size in bytes (default: 100000)
- `restore` (string): Session restoration strategy (`auto`, `manual`, `disabled`)

### Budget Controls

Monitor and control token usage:

```yaml
engine:
  id: copilot
  mode: sdk
  budget:
    max-tokens: 100000       # Hard limit on token usage
    warn-threshold: 80000    # Warning threshold (80%)
    action: terminate        # terminate, warn, or continue
```

**Budget Options:**
- `max-tokens` (integer): Maximum tokens allowed for the workflow
- `warn-threshold` (integer): Threshold for budget warnings
- `action` (string): Action when budget exceeded (`terminate`, `warn`, `continue`)

### Event Configuration

Enable real-time event handling:

```yaml
engine:
  id: copilot
  mode: sdk
  events:
    streaming: true          # Enable streaming events
    handlers:
      - on: token_usage
        action: log
        threshold: 10000
      - on: tool_call
        action: notify
      - on: error
        action: retry
        max-attempts: 3
```

**Event Types:**
- `token_usage`: Monitor token consumption
- `tool_call`: Tool invocation events
- `error`: Error and exception events
- `completion`: Agent completion events
- `state_change`: Session state changes

### Inline Tools

Define custom tools directly in the workflow:

```yaml
engine:
  id: copilot
  mode: sdk
  tools:
    inline:
      - name: validate_code
        description: "Run custom code validation"
        parameters:
          type: object
          properties:
            file_path:
              type: string
              description: "Path to file to validate"
        implementation: |
          const fs = require('fs');
          const content = fs.readFileSync(file_path, 'utf8');
          // Custom validation logic
          return { valid: true, issues: [] };
tools:
  github:
    allowed: [issue_read]
```

## Architecture

### SDK Execution Flow

```
┌──────────────┐
│   Workflow   │
│   Trigger    │
└──────┬───────┘
       │
       ▼
┌──────────────────┐
│  SDK Engine Init │
│  - Load config   │
│  - Setup session │
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│  Turn 1: Agent   │◄─────┐
│  - Process input │      │
│  - Call tools    │      │
│  - Generate resp │      │
└──────┬───────────┘      │
       │                  │
       ▼                  │
┌──────────────────┐      │
│  State Update    │      │
│  - Save context  │      │
│  - Check budget  │      │
└──────┬───────────┘      │
       │                  │
       ▼                  │
┌──────────────────┐      │
│  Continue?       │──YES─┘
│  - More turns?   │
│  - Budget OK?    │
└──────┬───────────┘
       NO
       ▼
┌──────────────────┐
│  Finalize        │
│  - Save results  │
│  - Cleanup       │
└──────────────────┘
```

### Component Architecture

```
┌─────────────────────────────────────────┐
│         Workflow Frontmatter            │
│  (engine config, tools, permissions)    │
└────────────────┬────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────┐
│         SDK Engine Runtime              │
├─────────────────────────────────────────┤
│  ┌──────────┐  ┌─────────────────────┐ │
│  │ Session  │  │  Event System       │ │
│  │ Manager  │  │  - Streaming        │ │
│  └──────────┘  │  - Handlers         │ │
│                └─────────────────────┘ │
│  ┌──────────┐  ┌─────────────────────┐ │
│  │  Budget  │  │  Tool Manager       │ │
│  │  Control │  │  - MCP servers      │ │
│  └──────────┘  │  - Inline tools     │ │
│                └─────────────────────┘ │
└────────────────┬────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────┐
│     GitHub Actions Infrastructure       │
│  (jobs, steps, outputs, artifacts)      │
└─────────────────────────────────────────┘
```

## Performance Characteristics

### Latency

- **First Response**: ~2-5 seconds (includes SDK initialization)
- **Subsequent Turns**: ~1-3 seconds (session already initialized)
- **Tool Calls**: Variable (depends on tool complexity)
- **State Persistence**: ~100-500ms overhead per turn

### Token Usage

SDK mode typically uses more tokens than CLI mode due to:
- Context retention across turns
- Event metadata
- Session state serialization

**Optimization Tips:**
- Set appropriate `max-turns` limits
- Use `state-size-limit` to cap context size
- Implement budget controls
- Prune context in custom handlers

### Resource Requirements

- **Memory**: 512MB-1GB per workflow run
- **CPU**: 1-2 cores recommended
- **Storage**: 10-100MB for session state
- **Network**: Persistent connection for streaming

## Limitations and Known Issues

### Current Limitations

1. **Model Support**: Currently supports GPT-5 and Claude Sonnet 4 only
2. **State Size**: Maximum state size of 1MB per session
3. **Concurrent Agents**: Maximum 5 concurrent agents per workflow
4. **Tool Complexity**: Inline tools limited to 50KB of code
5. **Event Buffer**: Event history limited to last 1000 events

### Known Issues

1. **State Persistence**: Session state may be lost on workflow timeout
2. **Event Ordering**: Events may arrive out of order during high concurrency
3. **Budget Accounting**: Token counting may lag actual usage by 1-2 turns
4. **Tool Conflicts**: Inline tools cannot override MCP server tools
5. **Error Recovery**: Limited automatic retry for SDK-level errors

### Workarounds

**State Loss on Timeout:**
```yaml
engine:
  session:
    restore: manual
    # Implement manual state saving in event handlers
```

**Event Ordering:**
```yaml
engine:
  events:
    streaming: false  # Disable streaming for strict ordering
```

**Budget Lag:**
```yaml
engine:
  budget:
    warn-threshold: 70000  # Set threshold lower for safety margin
```

## Migration Considerations

When migrating from CLI to SDK mode:

1. **Test Thoroughly**: SDK mode behavior differs from CLI
2. **Monitor Costs**: Multi-turn conversations use more tokens
3. **Adjust Timeouts**: SDK workflows may run longer
4. **Update Error Handling**: SDK has different error patterns
5. **Review Permissions**: Some SDK features need additional permissions

See the [Migration Guide](/gh-aw/guides/migrate-to-sdk/) for detailed migration instructions.

## Security Considerations

### Inline Tool Security

- **Code Isolation**: Inline tools run in sandboxed environment
- **Input Validation**: Always validate tool parameters
- **Secret Access**: Inline tools have access to workflow secrets
- **Network Restrictions**: Subject to network permissions

### Session State Security

- **Encryption**: Session state encrypted at rest
- **Access Control**: State accessible only within workflow run
- **Data Retention**: State auto-deleted after workflow completion
- **Audit Logging**: All state access logged

### Best Practices

1. **Validate Input**: Sanitize all parameters in inline tools
2. **Limit Scope**: Grant minimal permissions required
3. **Monitor Usage**: Use budget controls to prevent abuse
4. **Audit Events**: Log all tool calls and state changes
5. **Secure Secrets**: Use GitHub secrets, not hardcoded values

## Troubleshooting

### Common Issues

**Issue: Session state not persisted**
```yaml
# Solution: Enable persistent session
engine:
  session:
    persistent: true
```

**Issue: Budget exceeded unexpectedly**
```yaml
# Solution: Lower max-turns and add monitoring
engine:
  session:
    max-turns: 5  # Reduce from default
  budget:
    warn-threshold: 50000  # Get earlier warnings
```

**Issue: Inline tool not found**
```yaml
# Solution: Check tool name matches exactly
engine:
  tools:
    inline:
      - name: my_tool  # Must match agent's tool call
```

**Issue: Events not streaming**
```yaml
# Solution: Verify streaming enabled
engine:
  events:
    streaming: true
    handlers:
      - on: token_usage
        action: log
```

## Examples

### Basic Multi-Turn Workflow

```yaml
---
engine:
  id: copilot
  mode: sdk
  session:
    persistent: true
    max-turns: 5
tools:
  github:
    allowed: [issue_read, add_issue_comment]
---
# Code Review Assistant

Perform a multi-stage code review:
1. Initial review of the PR
2. Follow up on any questions
3. Provide final approval or request changes
```

### Custom Tool Example

```yaml
---
engine:
  id: copilot
  mode: sdk
  tools:
    inline:
      - name: check_test_coverage
        description: "Calculate test coverage percentage"
        parameters:
          type: object
          properties:
            directory:
              type: string
        implementation: |
          const { execSync } = require('child_process');
          const result = execSync(`coverage run ${directory}`);
          return { coverage: parseFloat(result) };
tools:
  github:
    allowed: [issue_read]
---
# Test Coverage Checker

Use check_test_coverage tool to validate coverage meets 80% threshold.
```

### Budget-Controlled Workflow

```yaml
---
engine:
  id: copilot
  mode: sdk
  session:
    max-turns: 20
  budget:
    max-tokens: 50000
    warn-threshold: 40000
    action: terminate
  events:
    handlers:
      - on: token_usage
        action: log
---
# Long-Running Analysis

Perform comprehensive codebase analysis with budget limits.
```

## Related Documentation

- [Session Management Guide](/gh-aw/guides/sdk-sessions/) - Managing multi-turn conversations
- [Custom Tools Guide](/gh-aw/guides/sdk-custom-tools/) - Creating inline tools
- [Event Handling Guide](/gh-aw/guides/sdk-events/) - Real-time event processing
- [Multi-Agent Guide](/gh-aw/guides/sdk-multi-agent/) - Coordinating multiple agents
- [Migration Guide](/gh-aw/guides/migrate-to-sdk/) - Migrating from CLI to SDK
- [AI Engines Reference](/gh-aw/reference/engines/) - General engine configuration
