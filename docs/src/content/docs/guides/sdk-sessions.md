---
title: SDK Session Management
description: Guide to managing multi-turn conversations, state persistence, and session restoration in SDK mode.
sidebar:
  order: 725
---

Session management is a core feature of SDK mode, enabling multi-turn conversations with persistent context across agent interactions.

## Overview

In SDK mode, a **session** represents a stateful conversation between the workflow and the AI agent. Unlike CLI mode where each invocation is independent, SDK sessions:

- Maintain conversation history across multiple turns
- Persist context and intermediate results
- Enable iterative refinement of responses
- Support continuation after interruption

## Session Lifecycle

### 1. Session Creation

A session is automatically created when a workflow with SDK mode starts:

```yaml
---
engine:
  id: copilot
  mode: sdk
  session:
    persistent: true
---
# Session starts automatically
```

**Lifecycle Steps:**
1. Workflow triggered (on issue, PR, schedule, etc.)
2. SDK engine initializes
3. New session created with unique ID
4. Initial context loaded from workflow frontmatter

### 2. Multi-Turn Interaction

Each agent response constitutes one "turn":

```
Turn 1: Agent analyzes the issue
        ↓
Turn 2: Agent asks clarifying questions
        ↓
Turn 3: Agent provides solution
        ↓
Turn 4: Agent refines based on feedback
```

**State Evolution:**
```yaml
# Turn 1 State
context:
  history: ["User: Analyze issue #123"]
  tools_called: []
  
# Turn 2 State  
context:
  history: [
    "User: Analyze issue #123",
    "Agent: I found several issues. Which priority?",
    "User: Focus on high priority"
  ]
  tools_called: ["issue_read"]
```

### 3. Session Termination

Sessions end when:
- Maximum turns reached (`max-turns`)
- Agent explicitly terminates
- Budget limit exceeded
- Workflow timeout
- Manual termination via event handler

## Configuration

### Basic Session Setup

```yaml
engine:
  id: copilot
  mode: sdk
  session:
    persistent: true    # Enable state persistence
    max-turns: 10      # Maximum conversation turns
```

### Complete Session Configuration

```yaml
engine:
  id: copilot
  mode: sdk
  session:
    # Core settings
    persistent: true
    max-turns: 10
    
    # State management
    state-size-limit: 50000      # Max state size (bytes)
    history-limit: 20            # Max messages in history
    prune-strategy: fifo         # fifo, lifo, or smart
    
    # Restoration
    restore: auto                # auto, manual, disabled
    restore-on-error: true       # Restore on failure
    
    # Timeouts
    turn-timeout: 300            # Max seconds per turn
    session-timeout: 3600        # Max seconds total
    
    # Compression
    compress-state: true         # Compress large states
    compression-threshold: 10000 # Compress if > 10KB
```

## Session Options Reference

### Core Settings

**`persistent`** (boolean, default: `false`)  
Enable state persistence between turns. When disabled, each turn starts fresh.

**`max-turns`** (integer, default: `5`)  
Maximum number of conversation turns before session terminates.

### State Management

**`state-size-limit`** (integer, default: `100000`)  
Maximum size of session state in bytes. When exceeded, oldest messages pruned.

**`history-limit`** (integer, default: `50`)  
Maximum number of messages kept in conversation history.

**`prune-strategy`** (string, default: `"smart"`)  
Strategy for pruning old messages:
- `fifo`: Remove oldest messages first
- `lifo`: Remove newest messages first
- `smart`: Remove least important messages

### Restoration

**`restore`** (string, default: `"auto"`)  
Session restoration strategy:
- `auto`: Automatically restore from saved state
- `manual`: Require manual restoration trigger
- `disabled`: Never restore, always start fresh

**`restore-on-error`** (boolean, default: `false`)  
Attempt to restore session when errors occur.

### Timeouts

**`turn-timeout`** (integer, default: `300`)  
Maximum seconds for a single turn before timeout.

**`session-timeout`** (integer, default: `1800`)  
Maximum seconds for entire session before timeout.

### Compression

**`compress-state`** (boolean, default: `true`)  
Enable compression for large session states.

**`compression-threshold`** (integer, default: `20000`)  
Size threshold (bytes) for triggering compression.

## Multi-Turn Conversation Patterns

### Pattern 1: Iterative Refinement

Agent progressively refines solution through multiple turns:

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
# Progressive Code Review

Stage 1: Initial quick scan for obvious issues
Stage 2: Deep dive into flagged areas
Stage 3: Check for edge cases
Stage 4: Verify test coverage
Stage 5: Final summary and recommendations
```

### Pattern 2: Clarification Loop

Agent asks questions to gather requirements:

```yaml
---
engine:
  id: copilot
  mode: sdk
  session:
    persistent: true
    max-turns: 8
---
# Requirements Gathering

1. Analyze initial request
2. Ask clarifying questions
3. Wait for user response (external input)
4. Refine understanding
5. Propose solution
6. Incorporate feedback
7. Finalize plan
```

### Pattern 3: Phased Processing

Long task broken into sequential phases:

```yaml
---
engine:
  id: copilot
  mode: sdk
  session:
    persistent: true
    max-turns: 15
    state-size-limit: 100000
---
# Codebase Migration

Phase 1 (turns 1-3): Analyze dependencies
Phase 2 (turns 4-6): Plan migration steps
Phase 3 (turns 7-10): Execute migration
Phase 4 (turns 11-13): Validate changes
Phase 5 (turns 14-15): Generate report
```

### Pattern 4: Error Recovery

Agent retries with accumulated context:

```yaml
---
engine:
  id: copilot
  mode: sdk
  session:
    persistent: true
    max-turns: 10
    restore-on-error: true
  events:
    handlers:
      - on: error
        action: retry
        max-attempts: 3
---
# Resilient Task Execution

Execute task with automatic retry on errors,
maintaining context across retry attempts.
```

## State Persistence

### What Gets Persisted

Session state includes:

1. **Conversation History**
   - User messages
   - Agent responses
   - System messages

2. **Tool Call History**
   - Tools invoked
   - Parameters used
   - Results returned

3. **Intermediate Results**
   - Partial computations
   - Cached data
   - Decision points

4. **Metadata**
   - Session ID
   - Turn count
   - Token usage
   - Timestamps

### State Storage

State is persisted as GitHub Actions artifacts:

```yaml
# Automatic state artifact
artifacts:
  - name: session-state-{run-id}
    path: .gh-aw/session/state.json
    retention-days: 7
```

**State File Structure:**
```json
{
  "session_id": "abc123",
  "created_at": "2026-01-16T01:00:00Z",
  "turn_count": 5,
  "history": [
    {"role": "user", "content": "..."},
    {"role": "agent", "content": "..."}
  ],
  "tools_called": [...],
  "metadata": {...}
}
```

### Manual State Management

Access session state in workflows:

```yaml
---
engine:
  id: copilot
  mode: sdk
  session:
    persistent: true
    restore: manual
  tools:
    inline:
      - name: save_checkpoint
        description: "Save session checkpoint"
        implementation: |
          const fs = require('fs');
          const state = global.session.getState();
          fs.writeFileSync('checkpoint.json', JSON.stringify(state));
          return { saved: true };
      
      - name: restore_checkpoint
        description: "Restore from checkpoint"
        implementation: |
          const fs = require('fs');
          const state = JSON.parse(fs.readFileSync('checkpoint.json'));
          global.session.setState(state);
          return { restored: true };
---
# Checkpoint-Based Processing

Use save_checkpoint and restore_checkpoint for manual state control.
```

## Session Restoration

### Automatic Restoration

With `restore: auto`, session automatically restores on workflow restart:

```yaml
engine:
  id: copilot
  mode: sdk
  session:
    persistent: true
    restore: auto
    restore-on-error: true
```

**Restoration Flow:**
1. Workflow restarts (retry, manual trigger, etc.)
2. SDK engine checks for saved state artifact
3. If found, loads state and resumes
4. If not found, starts new session

### Manual Restoration

With `restore: manual`, implement custom restoration logic:

```yaml
---
engine:
  id: copilot
  mode: sdk
  session:
    persistent: true
    restore: manual
  tools:
    inline:
      - name: should_restore
        description: "Check if session should be restored"
        implementation: |
          const stateExists = checkForStateFile();
          const withinTimeLimit = checkTimestamp();
          return { restore: stateExists && withinTimeLimit };
---
# Conditional Restoration

Use should_restore tool to determine if session restoration is appropriate.
```

### Restoration Scenarios

**Scenario 1: Workflow Timeout and Retry**
```yaml
# Original run times out at turn 7
# Retry automatically restores and continues from turn 7
engine:
  session:
    restore: auto
    max-turns: 20
    session-timeout: 3600
```

**Scenario 2: Manual Re-trigger**
```yaml
# User manually re-triggers workflow
# Session restores if within time window
engine:
  session:
    restore: manual
  tools:
    inline:
      - name: check_restore_window
        implementation: |
          const hoursSince = getHoursSinceLastRun();
          return { should_restore: hoursSince < 24 };
```

**Scenario 3: Error Recovery**
```yaml
# Agent encounters error at turn 5
# Automatic retry restores context
engine:
  session:
    restore-on-error: true
  events:
    handlers:
      - on: error
        action: retry
```

## State Size Management

### Monitoring State Size

Track state size to prevent limit issues:

```yaml
engine:
  id: copilot
  mode: sdk
  session:
    state-size-limit: 50000
  events:
    handlers:
      - on: state_change
        action: monitor
  tools:
    inline:
      - name: check_state_size
        description: "Monitor current state size"
        implementation: |
          const state = global.session.getState();
          const size = JSON.stringify(state).length;
          const limit = 50000;
          const usage = (size / limit * 100).toFixed(1);
          return { 
            size_bytes: size,
            limit_bytes: limit,
            usage_percent: usage
          };
---
# State Size Monitoring

Use check_state_size to monitor state growth.
```

### Pruning Strategies

**FIFO Pruning (First In, First Out):**
```yaml
session:
  prune-strategy: fifo
  history-limit: 20
# Keeps most recent 20 messages
```

**Smart Pruning:**
```yaml
session:
  prune-strategy: smart
  state-size-limit: 50000
# Intelligently removes less important messages
# Preserves: recent messages, tool results, key decisions
```

**Manual Pruning:**
```yaml
tools:
  inline:
    - name: prune_history
      description: "Manually prune conversation history"
      implementation: |
        const history = global.session.getHistory();
        const pruned = history.filter(msg => msg.important);
        global.session.setHistory(pruned);
        return { messages_removed: history.length - pruned.length };
```

### Optimization Tips

1. **Set Appropriate Limits**
   ```yaml
   session:
     history-limit: 30        # Keep only essential history
     state-size-limit: 50000  # Reasonable for most workflows
   ```

2. **Enable Compression**
   ```yaml
   session:
     compress-state: true
     compression-threshold: 10000
   ```

3. **Summarize Periodically**
   ```yaml
   tools:
     inline:
       - name: summarize_progress
         implementation: |
           // Replace detailed history with summary every 5 turns
           if (turn % 5 === 0) {
             const summary = generateSummary(history);
             global.session.replaceHistory([summary]);
           }
   ```

4. **Store Large Data Separately**
   ```yaml
   tools:
     inline:
       - name: process_large_data
         implementation: |
           // Store large results in artifacts, not session state
           fs.writeFileSync('/tmp/results.json', largeData);
           return { result_path: '/tmp/results.json' };
   ```

## Best Practices

### 1. Set Realistic Turn Limits

```yaml
# Good: Reasonable limit for task complexity
engine:
  session:
    max-turns: 10  # Sufficient for multi-stage review

# Bad: Unnecessarily high limit
engine:
  session:
    max-turns: 100  # Excessive, increases costs
```

### 2. Monitor and Optimize State Size

```yaml
engine:
  session:
    state-size-limit: 50000
  events:
    handlers:
      - on: state_change
        action: log
        threshold: 40000  # Warn at 80% capacity
```

### 3. Use Appropriate Restore Strategy

```yaml
# For long-running tasks: auto restore
engine:
  session:
    restore: auto

# For one-time tasks: disabled restore
engine:
  session:
    restore: disabled
```

### 4. Implement Checkpoints

```yaml
# Save checkpoints at critical points
tools:
  inline:
    - name: checkpoint
      implementation: |
        if (turn === 5 || turn === 10) {
          saveState();
        }
```

### 5. Handle Session Timeout

```yaml
engine:
  session:
    session-timeout: 3600  # 1 hour
  events:
    handlers:
      - on: timeout
        action: save_and_terminate
```

## Troubleshooting

### Issue: State Not Persisting

**Symptoms:** Session starts fresh on each turn

**Solutions:**
```yaml
# 1. Verify persistent flag
engine:
  session:
    persistent: true  # Must be explicitly enabled

# 2. Check state size
engine:
  session:
    state-size-limit: 100000  # Increase if needed
```

### Issue: Session Timing Out

**Symptoms:** Workflow terminates before completion

**Solutions:**
```yaml
# 1. Increase timeouts
engine:
  session:
    turn-timeout: 600      # 10 minutes per turn
    session-timeout: 7200  # 2 hours total

# 2. Reduce turn count
engine:
  session:
    max-turns: 5  # Complete task in fewer turns
```

### Issue: State Size Exceeded

**Symptoms:** "State size limit exceeded" error

**Solutions:**
```yaml
# 1. Enable aggressive pruning
engine:
  session:
    prune-strategy: smart
    history-limit: 10  # Keep fewer messages

# 2. Enable compression
engine:
  session:
    compress-state: true

# 3. Increase limit if justified
engine:
  session:
    state-size-limit: 200000  # If truly needed
```

### Issue: Restoration Failing

**Symptoms:** Session doesn't restore from saved state

**Solutions:**
```yaml
# 1. Check restore setting
engine:
  session:
    restore: auto  # Not disabled

# 2. Verify artifact retention
artifacts:
  retention-days: 7  # Long enough for restoration

# 3. Add error handling
engine:
  session:
    restore-on-error: true
```

## Examples

### Example 1: Simple Multi-Turn Review

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
# Iterative PR Review

Turn 1: Initial code review
Turn 2: Check test coverage
Turn 3: Review security implications
Turn 4: Assess performance impact
Turn 5: Final recommendations
```

### Example 2: Long-Running Analysis

```yaml
---
engine:
  id: copilot
  mode: sdk
  session:
    persistent: true
    max-turns: 20
    state-size-limit: 100000
    session-timeout: 7200
  events:
    handlers:
      - on: state_change
        action: checkpoint
        interval: 5  # Every 5 turns
---
# Comprehensive Codebase Analysis

Perform deep analysis with automatic checkpointing.
```

### Example 3: Checkpoint-Based Processing

```yaml
---
engine:
  id: copilot
  mode: sdk
  session:
    persistent: true
    restore: manual
    max-turns: 15
  tools:
    inline:
      - name: save_checkpoint
        implementation: |
          const state = global.session.getState();
          saveToArtifact('checkpoint.json', state);
      
      - name: restore_from_checkpoint
        implementation: |
          const state = loadFromArtifact('checkpoint.json');
          global.session.setState(state);
---
# Manual Checkpoint Control

Use save_checkpoint at critical points for manual state management.
```

## Related Documentation

- [SDK Engine Reference](/gh-aw/reference/engines-sdk/) - Complete SDK configuration
- [Event Handling Guide](/gh-aw/guides/sdk-events/) - Real-time event processing
- [Migration Guide](/gh-aw/guides/migrate-to-sdk/) - Migrating from CLI mode
