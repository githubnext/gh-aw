---
title: SDK Event Handling
description: Guide to real-time event handling and monitoring in SDK mode workflows.
sidebar:
  order: 727
---

SDK mode provides real-time event streaming that enables monitoring, control, and dynamic adaptation of agent workflows during execution.

## Overview

**Events** are real-time notifications emitted by the SDK engine during workflow execution. Event handlers enable you to:

- Monitor agent activity in real-time
- Track token usage and costs
- Implement custom logging and alerting
- Control workflow execution dynamically
- Respond to errors and anomalies
- Collect metrics and analytics

## Event Types

### Core Event Types

**`token_usage`** - Token consumption events
```yaml
events:
  handlers:
    - on: token_usage
      action: log
```

**`tool_call`** - Tool invocation events
```yaml
events:
  handlers:
    - on: tool_call
      action: notify
```

**`error`** - Error and exception events
```yaml
events:
  handlers:
    - on: error
      action: retry
```

**`completion`** - Agent task completion events
```yaml
events:
  handlers:
    - on: completion
      action: save_results
```

**`state_change`** - Session state modification events
```yaml
events:
  handlers:
    - on: state_change
      action: checkpoint
```

### Extended Event Types

**`message`** - Agent message events (streaming)
```yaml
events:
  handlers:
    - on: message
      action: stream
```

**`turn_start`** - Beginning of conversation turn
```yaml
events:
  handlers:
    - on: turn_start
      action: log_turn
```

**`turn_end`** - End of conversation turn
```yaml
events:
  handlers:
    - on: turn_end
      action: validate_turn
```

**`budget_warning`** - Budget threshold reached
```yaml
events:
  handlers:
    - on: budget_warning
      action: alert
```

**`timeout_warning`** - Approaching timeout
```yaml
events:
  handlers:
    - on: timeout_warning
      action: checkpoint_and_continue
```

## Basic Configuration

### Enabling Events

```yaml
---
engine:
  id: copilot
  mode: sdk
  events:
    streaming: true  # Enable real-time streaming
---
```

### Simple Handler

```yaml
---
engine:
  id: copilot
  mode: sdk
  events:
    streaming: true
    handlers:
      - on: token_usage
        action: log
---
```

### Multiple Handlers

```yaml
---
engine:
  id: copilot
  mode: sdk
  events:
    streaming: true
    handlers:
      - on: token_usage
        action: log
        threshold: 10000
      
      - on: tool_call
        action: notify
      
      - on: error
        action: retry
        max-attempts: 3
      
      - on: completion
        action: save_results
---
```

## Handler Configuration

### Handler Structure

```yaml
handlers:
  - on: <event_type>          # Event to handle (required)
    action: <action_type>      # Action to perform (required)
    filter: <filter_expr>      # Optional filter expression
    threshold: <value>         # Optional threshold for triggers
    config: <action_config>    # Action-specific configuration
```

### Built-in Actions

**`log`** - Log event to workflow output
```yaml
handlers:
  - on: token_usage
    action: log
    config:
      level: info  # debug, info, warn, error
      format: json # json, text
```

**`notify`** - Send notification
```yaml
handlers:
  - on: error
    action: notify
    config:
      channel: slack
      webhook: ${{ secrets.SLACK_WEBHOOK }}
```

**`retry`** - Retry on failure
```yaml
handlers:
  - on: error
    action: retry
    config:
      max-attempts: 3
      backoff: exponential  # linear, exponential, fixed
      initial-delay: 1      # seconds
```

**`terminate`** - Stop workflow execution
```yaml
handlers:
  - on: budget_warning
    action: terminate
    threshold: 90000  # 90% of budget
```

**`checkpoint`** - Save session state
```yaml
handlers:
  - on: state_change
    action: checkpoint
    config:
      interval: 5  # Every 5 turns
```

**`custom`** - Execute custom handler code
```yaml
handlers:
  - on: tool_call
    action: custom
    config:
      implementation: |
        const core = require('@actions/core');
        core.info(`Tool called: ${event.tool_name}`);
        
        if (event.tool_name === 'dangerous_operation') {
          core.warning('Dangerous operation detected');
        }
```

## Event Data Structures

### Token Usage Event

```javascript
{
  type: 'token_usage',
  timestamp: '2026-01-16T01:00:00Z',
  data: {
    prompt_tokens: 1250,
    completion_tokens: 380,
    total_tokens: 1630,
    cumulative_tokens: 15840,
    budget_remaining: 84160,
    budget_used_percent: 15.84
  }
}
```

### Tool Call Event

```javascript
{
  type: 'tool_call',
  timestamp: '2026-01-16T01:00:15Z',
  data: {
    tool_name: 'issue_read',
    parameters: {
      issue_number: 123,
      repo: 'owner/repo'
    },
    duration_ms: 245,
    success: true,
    result: { /* tool result */ }
  }
}
```

### Error Event

```javascript
{
  type: 'error',
  timestamp: '2026-01-16T01:00:30Z',
  data: {
    error_type: 'ToolExecutionError',
    message: 'API rate limit exceeded',
    tool_name: 'github_api_call',
    recoverable: true,
    retry_after: 60,
    stack_trace: '...'
  }
}
```

### Completion Event

```javascript
{
  type: 'completion',
  timestamp: '2026-01-16T01:05:00Z',
  data: {
    status: 'success',
    turns: 8,
    total_tokens: 45320,
    duration_seconds: 187,
    tools_called: ['issue_read', 'add_issue_comment'],
    final_output: '...'
  }
}
```

### State Change Event

```javascript
{
  type: 'state_change',
  timestamp: '2026-01-16T01:01:00Z',
  data: {
    turn: 3,
    state_size_bytes: 34520,
    messages_count: 12,
    change_type: 'message_added',
    delta_bytes: 1280
  }
}
```

## Handler Patterns

### Pattern 1: Budget Monitoring

Track and control token usage:

```yaml
---
engine:
  id: copilot
  mode: sdk
  budget:
    max-tokens: 100000
  events:
    streaming: true
    handlers:
      # Log every 10k tokens
      - on: token_usage
        action: log
        threshold: 10000
        config:
          level: info
      
      # Warn at 80% budget
      - on: token_usage
        action: notify
        filter: "data.budget_used_percent >= 80"
        config:
          channel: slack
          message: "Budget 80% consumed"
      
      # Terminate at 95% budget
      - on: token_usage
        action: terminate
        filter: "data.budget_used_percent >= 95"
---
# Budget-Controlled Workflow

Perform analysis with strict budget controls.
```

### Pattern 2: Error Recovery

Automatic retry with exponential backoff:

```yaml
---
engine:
  id: copilot
  mode: sdk
  events:
    streaming: true
    handlers:
      # Retry transient errors
      - on: error
        action: retry
        filter: "data.recoverable === true"
        config:
          max-attempts: 3
          backoff: exponential
          initial-delay: 2
      
      # Alert on persistent errors
      - on: error
        action: notify
        filter: "data.attempt_number >= 3"
        config:
          channel: slack
          webhook: ${{ secrets.SLACK_WEBHOOK }}
      
      # Log all errors
      - on: error
        action: log
        config:
          level: error
          include-stack: true
---
# Resilient Workflow

Automatic error recovery with notifications.
```

### Pattern 3: Performance Monitoring

Track execution metrics:

```yaml
---
engine:
  id: copilot
  mode: sdk
  events:
    streaming: true
    handlers:
      # Log turn metrics
      - on: turn_end
        action: custom
        config:
          implementation: |
            const core = require('@actions/core');
            const metrics = {
              turn: event.data.turn,
              duration: event.data.duration_ms,
              tokens: event.data.tokens_used,
              tools: event.data.tools_called.length
            };
            core.info(`Turn ${metrics.turn}: ${JSON.stringify(metrics)}`);
      
      # Track slow operations
      - on: tool_call
        action: log
        filter: "data.duration_ms > 5000"
        config:
          level: warn
          message: "Slow tool execution detected"
---
# Performance Monitoring

Track and log performance metrics.
```

### Pattern 4: State Management

Automatic checkpointing:

```yaml
---
engine:
  id: copilot
  mode: sdk
  session:
    persistent: true
  events:
    streaming: true
    handlers:
      # Checkpoint every 5 turns
      - on: turn_end
        action: checkpoint
        filter: "data.turn % 5 === 0"
      
      # Checkpoint before risky operations
      - on: tool_call
        action: checkpoint
        filter: "data.tool_name === 'execute_code'"
      
      # Monitor state size
      - on: state_change
        action: log
        filter: "data.state_size_bytes > 50000"
        config:
          level: warn
          message: "Large state detected"
---
# Checkpoint-Based Workflow

Automatic state checkpointing for recovery.
```

### Pattern 5: Real-Time Streaming

Stream agent responses:

```yaml
---
engine:
  id: copilot
  mode: sdk
  events:
    streaming: true
    handlers:
      # Stream messages to output
      - on: message
        action: custom
        config:
          implementation: |
            const core = require('@actions/core');
            // Stream each message chunk
            if (event.data.delta) {
              process.stdout.write(event.data.delta);
            }
      
      # Log when streaming completes
      - on: message
        action: log
        filter: "data.final === true"
        config:
          message: "Message streaming complete"
---
# Streaming Workflow

Real-time response streaming.
```

## Custom Handler Implementation

### Basic Custom Handler

```yaml
handlers:
  - on: tool_call
    action: custom
    config:
      implementation: |
        const core = require('@actions/core');
        
        // Access event data
        const { tool_name, parameters, duration_ms } = event.data;
        
        // Log details
        core.info(`Tool: ${tool_name}, Duration: ${duration_ms}ms`);
        
        // Custom logic
        if (duration_ms > 5000) {
          core.warning(`Slow tool execution: ${tool_name}`);
        }
```

### Advanced Custom Handler

```yaml
handlers:
  - on: token_usage
    action: custom
    config:
      implementation: |
        const core = require('@actions/core');
        const fs = require('fs');
        
        // Track cumulative usage
        const logFile = '/tmp/token-usage.json';
        let usage = [];
        
        if (fs.existsSync(logFile)) {
          usage = JSON.parse(fs.readFileSync(logFile, 'utf8'));
        }
        
        usage.push({
          timestamp: event.timestamp,
          tokens: event.data.total_tokens,
          cumulative: event.data.cumulative_tokens
        });
        
        fs.writeFileSync(logFile, JSON.stringify(usage, null, 2));
        
        // Alert if usage spike detected
        if (usage.length >= 2) {
          const current = usage[usage.length - 1].tokens;
          const previous = usage[usage.length - 2].tokens;
          const increase = ((current - previous) / previous * 100).toFixed(1);
          
          if (increase > 50) {
            core.warning(`Token usage increased by ${increase}%`);
          }
        }
```

### Handler with External API

```yaml
handlers:
  - on: completion
    action: custom
    config:
      implementation: |
        const https = require('https');
        
        // Send metrics to external service
        const data = JSON.stringify({
          workflow: process.env.GITHUB_WORKFLOW,
          status: event.data.status,
          duration: event.data.duration_seconds,
          tokens: event.data.total_tokens,
          timestamp: event.timestamp
        });
        
        const options = {
          hostname: 'metrics.example.com',
          port: 443,
          path: '/api/workflow-metrics',
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Content-Length': data.length,
            'Authorization': `Bearer ${process.env.METRICS_API_KEY}`
          }
        };
        
        const req = https.request(options, (res) => {
          console.log(`Metrics sent: ${res.statusCode}`);
        });
        
        req.on('error', (error) => {
          console.error('Failed to send metrics:', error);
        });
        
        req.write(data);
        req.end();
```

## Event Filtering

### Filter Expressions

Use JavaScript expressions to filter events:

```yaml
handlers:
  # Only high token usage
  - on: token_usage
    action: log
    filter: "data.total_tokens > 10000"
  
  # Only failed tool calls
  - on: tool_call
    action: retry
    filter: "data.success === false"
  
  # Only recoverable errors
  - on: error
    action: retry
    filter: "data.recoverable === true && data.attempt_number < 3"
  
  # Complex condition
  - on: state_change
    action: checkpoint
    filter: "data.state_size_bytes > 50000 || data.turn % 10 === 0"
```

### Filter Context

Filters have access to:

```javascript
{
  event: {
    type: 'event_type',
    timestamp: '...',
    data: { /* event-specific data */ }
  },
  session: {
    turn: 5,
    state_size: 45000,
    tokens_used: 35000
  },
  workflow: {
    name: 'workflow-name',
    run_id: '12345',
    attempt: 1
  }
}
```

## Threshold-Based Handlers

### Simple Threshold

```yaml
handlers:
  # Trigger when threshold exceeded
  - on: token_usage
    action: log
    threshold: 50000  # Log when cumulative exceeds 50k
```

### Threshold with Action

```yaml
handlers:
  # Warn at 80% budget
  - on: token_usage
    action: notify
    threshold: 80000
    config:
      message: "Approaching budget limit"
  
  # Terminate at 100% budget
  - on: token_usage
    action: terminate
    threshold: 100000
```

### Multiple Thresholds

```yaml
handlers:
  # Info at 25%
  - on: token_usage
    action: log
    threshold: 25000
    config:
      level: info
  
  # Warn at 50%
  - on: token_usage
    action: log
    threshold: 50000
    config:
      level: warn
  
  # Error at 75%
  - on: token_usage
    action: log
    threshold: 75000
    config:
      level: error
```

## Best Practices

### 1. Use Appropriate Logging Levels

```yaml
handlers:
  # Debug: Detailed diagnostics
  - on: tool_call
    action: log
    config:
      level: debug
  
  # Info: Normal operations
  - on: turn_end
    action: log
    config:
      level: info
  
  # Warn: Potential issues
  - on: token_usage
    filter: "data.budget_used_percent > 80"
    action: log
    config:
      level: warn
  
  # Error: Failures
  - on: error
    action: log
    config:
      level: error
```

### 2. Implement Graceful Degradation

```yaml
handlers:
  # Try notification, fall back to logging
  - on: error
    action: custom
    config:
      implementation: |
        try {
          await sendSlackNotification(event);
        } catch (err) {
          console.error('Notification failed, logging locally:', err);
          fs.appendFileSync('/tmp/errors.log', JSON.stringify(event));
        }
```

### 3. Avoid Handler Overhead

```yaml
# ❌ Bad: Handler for every message
handlers:
  - on: message
    action: custom
    config:
      implementation: |
        // Heavy processing on every chunk
        complexAnalysis(event.data);

# ✅ Good: Batch processing
handlers:
  - on: turn_end
    action: custom
    config:
      implementation: |
        // Process accumulated messages once per turn
        processMessages(session.messages);
```

### 4. Set Appropriate Timeouts

```yaml
handlers:
  - on: tool_call
    action: custom
    config:
      timeout: 10  # Handler timeout in seconds
      implementation: |
        // Ensure handler completes quickly
```

### 5. Monitor Handler Performance

```yaml
handlers:
  - on: token_usage
    action: custom
    config:
      implementation: |
        const start = Date.now();
        
        // Handler logic
        processTokenUsage(event);
        
        const duration = Date.now() - start;
        if (duration > 1000) {
          console.warn(`Handler took ${duration}ms`);
        }
```

## Troubleshooting

### Issue: Events Not Firing

**Symptoms:** Handlers not triggered

**Solutions:**
```yaml
# 1. Enable streaming
engine:
  events:
    streaming: true

# 2. Check event type spelling
handlers:
  - on: token_usage  # Correct
  # - on: token-usage  # ❌ Wrong

# 3. Verify filter expression
handlers:
  - on: token_usage
    filter: "data.total_tokens > 1000"  # Check syntax
```

### Issue: Handler Errors

**Symptoms:** Handler fails silently

**Solutions:**
```yaml
handlers:
  - on: error
    action: custom
    config:
      implementation: |
        try {
          // Handler logic
        } catch (handlerError) {
          console.error('Handler failed:', handlerError);
          // Don't let handler failure stop workflow
        }
```

### Issue: Performance Degradation

**Symptoms:** Workflow slow with handlers

**Solutions:**
```yaml
# 1. Use filtering to reduce handler calls
handlers:
  - on: message
    filter: "data.final === true"  # Only final messages

# 2. Optimize handler implementation
handlers:
  - on: tool_call
    action: custom
    config:
      implementation: |
        // ❌ Slow: Synchronous file I/O
        // fs.writeFileSync('log.txt', data);
        
        // ✅ Fast: Async or batch
        setImmediate(() => fs.appendFile('log.txt', data, ()=>{}));
```

### Issue: Memory Leaks

**Symptoms:** Increasing memory usage

**Solutions:**
```yaml
handlers:
  - on: token_usage
    action: custom
    config:
      implementation: |
        // ❌ Bad: Accumulates in memory
        // global.allEvents.push(event);
        
        // ✅ Good: Write to disk or limit size
        const maxEvents = 100;
        if (!global.recentEvents) global.recentEvents = [];
        global.recentEvents.push(event);
        if (global.recentEvents.length > maxEvents) {
          global.recentEvents.shift();
        }
```

## Examples

### Example 1: Comprehensive Monitoring

```yaml
---
engine:
  id: copilot
  mode: sdk
  budget:
    max-tokens: 100000
  events:
    streaming: true
    handlers:
      # Token monitoring
      - on: token_usage
        action: log
        threshold: 10000
      
      # Tool performance tracking
      - on: tool_call
        action: custom
        config:
          implementation: |
            if (event.data.duration_ms > 5000) {
              console.warn(`Slow tool: ${event.data.tool_name} (${event.data.duration_ms}ms)`);
            }
      
      # Error alerting
      - on: error
        action: notify
        config:
          channel: slack
          webhook: ${{ secrets.SLACK_WEBHOOK }}
      
      # Completion metrics
      - on: completion
        action: custom
        config:
          implementation: |
            const summary = {
              status: event.data.status,
              turns: event.data.turns,
              tokens: event.data.total_tokens,
              duration: event.data.duration_seconds,
              efficiency: (event.data.total_tokens / event.data.duration_seconds).toFixed(2)
            };
            console.log('Workflow Summary:', JSON.stringify(summary, null, 2));
---
# Monitored Workflow

Comprehensive monitoring with alerts.
```

### Example 2: Cost-Aware Execution

```yaml
---
engine:
  id: copilot
  mode: sdk
  budget:
    max-tokens: 50000
  events:
    streaming: true
    handlers:
      # Track costs
      - on: token_usage
        action: custom
        config:
          implementation: |
            const COST_PER_1K_TOKENS = 0.002;  # Example rate
            const tokens = event.data.cumulative_tokens;
            const cost = (tokens / 1000 * COST_PER_1K_TOKENS).toFixed(4);
            console.log(`Current cost: $${cost}`);
      
      # Warn at budget thresholds
      - on: token_usage
        action: log
        filter: "data.budget_used_percent >= 80"
        config:
          level: warn
          message: "80% budget consumed"
      
      # Terminate at limit
      - on: token_usage
        action: terminate
        filter: "data.budget_used_percent >= 100"
---
# Cost-Controlled Workflow

Track and control execution costs.
```

### Example 3: Automatic Recovery

```yaml
---
engine:
  id: copilot
  mode: sdk
  session:
    persistent: true
  events:
    streaming: true
    handlers:
      # Auto-retry on transient errors
      - on: error
        action: retry
        filter: "data.recoverable && data.attempt_number < 3"
        config:
          backoff: exponential
      
      # Checkpoint before retry
      - on: error
        action: checkpoint
        filter: "data.attempt_number === 1"
      
      # Alert on persistent failure
      - on: error
        action: notify
        filter: "data.attempt_number >= 3"
---
# Resilient Workflow

Automatic recovery with checkpointing.
```

## Related Documentation

- [SDK Engine Reference](/gh-aw/reference/engines-sdk/) - Complete SDK configuration
- [Session Management](/gh-aw/guides/sdk-sessions/) - Multi-turn conversations
- [Custom Tools](/gh-aw/guides/sdk-custom-tools/) - Creating inline tools
- [Migration Guide](/gh-aw/guides/migrate-to-sdk/) - Migrating from CLI mode
