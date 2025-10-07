---
title: Concurrency Control
description: Complete guide to concurrency control in GitHub Agentic Workflows, including max-concurrency configuration, global locks, and engine isolation.
sidebar:
  order: 6
---

GitHub Agentic Workflows provides sophisticated concurrency control to manage how many AI-powered workflows can run simultaneously. This helps prevent resource exhaustion, control costs, and ensure predictable workflow execution.

## Overview

Concurrency control in GitHub Agentic Workflows uses a dual-level approach:
- **Workflow-level concurrency**: Limits concurrent workflow runs
- **Job-level concurrency**: Limits concurrent agentic job executions

Both levels use the same concurrency group key, creating a global lock across all workflows and refs for each engine.

## Max Concurrency Configuration

The `max-concurrency` option is configured under the `engine` section and controls how many agentic jobs can run concurrently across **all workflows** in your repository:

```yaml
engine:
  id: claude
  max-concurrency: 5
```

### Default Value

- **Default**: 3 concurrent slots (when not specified or set to 0)
- **Minimum**: 1 (sequential execution)
- **No maximum**: Set to any positive integer based on your needs

### Configuration Examples

**Sequential execution (one at a time):**
```yaml
engine:
  id: copilot
  max-concurrency: 1
```

**Moderate parallelism (default):**
```yaml
engine:
  id: claude
  # max-concurrency not specified, defaults to 3
```

**High parallelism for busy repositories:**
```yaml
engine:
  id: claude
  max-concurrency: 10
```

## How It Works

### Concurrency Group Generation

The system generates a concurrency group key using:
1. **Prefix**: `gh-aw-` (standardized identifier)
2. **Engine ID**: `copilot`, `claude`, `codex`, or custom engine name
3. **Slot Number**: `${{ github.run_id % max-concurrency }}`

**Generated pattern:**
```yaml
concurrency:
  group: "gh-aw-{engine-id}-${{ github.run_id % max-concurrency }}"
```

**Example for Claude with max-concurrency of 5:**
```yaml
concurrency:
  group: "gh-aw-claude-${{ github.run_id % 5 }}"
```

### Slot Distribution

Workflows are distributed across available slots using modulo arithmetic:
- `github.run_id % max-concurrency` calculates the slot number (0 to max-concurrency-1)
- Each slot can only run one workflow at a time
- Workflows are automatically assigned to the next available slot

**Example with max-concurrency: 3**
- Run ID 1001 → Slot 2 (`1001 % 3 = 2`)
- Run ID 1002 → Slot 0 (`1002 % 3 = 0`)
- Run ID 1003 → Slot 1 (`1003 % 3 = 1`)
- Run ID 1004 → Slot 2 (`1004 % 3 = 2`)

### Dual-Level Application

Concurrency is applied at both workflow and job levels:

```yaml
name: "Issue Responder"
on:
  issues:
    types: [opened]

permissions: {}

# Workflow-level concurrency
concurrency:
  group: "gh-aw-claude-${{ github.run_id % 5 }}"

jobs:
  agent:
    runs-on: ubuntu-latest
    permissions: read-all
    # Job-level concurrency (same group)
    concurrency:
      group: "gh-aw-claude-${{ github.run_id % 5 }}"
    steps:
      - name: Execute workflow
        ...
```

## Global Lock Behavior

The concurrency group uses **only** engine ID and slot number, creating a true global lock:

### What's Included
- ✅ Engine ID (`copilot`, `claude`, `codex`)
- ✅ Slot number (from `run_id % max-concurrency`)
- ✅ `gh-aw-` prefix

### What's NOT Included
- ❌ Workflow name
- ❌ Issue number
- ❌ Pull request number
- ❌ Branch/ref name
- ❌ Event type

This ensures the limit applies **repository-wide** across all workflows and refs for each engine.

## Engine Isolation

Different engines can run concurrently without interfering with each other:

```yaml
# Workflow A uses Copilot
engine:
  id: copilot
  max-concurrency: 3

# Workflow B uses Claude
engine:
  id: claude
  max-concurrency: 5
```

- Copilot workflows have their own 3-slot concurrency pool
- Claude workflows have their own 5-slot concurrency pool
- Both can run simultaneously without conflict

## Cancellation Behavior

Concurrency cancellation varies by workflow trigger type:

| Trigger Type | Cancel-in-Progress | Reason |
|--------------|-------------------|--------|
| `pull_request` | ✅ Enabled | New commits should cancel outdated PR runs |
| All other triggers | ❌ Disabled | Issue/discussion workflows should run to completion |

**Example for pull request workflow:**
```yaml
concurrency:
  group: "gh-aw-copilot-${{ github.run_id % 3 }}"
  cancel-in-progress: true
```

## Benefits

### Cost Control
- **Prevents runaway costs**: Limits the number of concurrent AI executions
- **Predictable spending**: Maximum concurrent workflows are known in advance
- **Flexible budgeting**: Adjust limits based on repository needs

### Resource Management
- **Prevents resource exhaustion**: Ensures system stability
- **Fair resource distribution**: Workflows queue when slots are full
- **Maintains throughput**: Multiple workflows can still run concurrently

### Engine Isolation
- **Independent limits**: Each engine has its own concurrency pool
- **No cross-engine interference**: Copilot workflows don't block Claude workflows
- **Flexible configuration**: Different limits for different engines

### Simplicity
- **Global lock**: Same limit across all workflows and refs
- **Automatic distribution**: No manual slot assignment needed
- **Consistent behavior**: Predictable execution patterns

## Custom Concurrency

You can override the automatic concurrency generation by specifying your own `concurrency` section in the frontmatter:

```yaml
---
on: push
concurrency:
  group: custom-group-${{ github.ref }}
  cancel-in-progress: true
tools:
  github:
    allowed: [list_issues]
---
```

**Note**: Custom concurrency bypasses the max-concurrency limit and engine isolation features.

## Best Practices

### Setting Max Concurrency

**Start conservative:**
```yaml
engine:
  id: claude
  max-concurrency: 1  # Start with sequential execution
```

**Increase as needed:**
```yaml
engine:
  id: claude
  max-concurrency: 5  # Increase after monitoring costs and performance
```

### Different Limits for Different Engines

**Cost-sensitive engine:**
```yaml
engine:
  id: claude  # Expensive model
  max-concurrency: 2
```

**Budget-friendly engine:**
```yaml
engine:
  id: copilot  # More affordable
  max-concurrency: 5
```

### Monitoring and Adjustment

1. **Monitor workflow execution**: Use GitHub Actions insights
2. **Track costs**: Review AI model usage and expenses
3. **Adjust limits**: Increase or decrease based on needs
4. **Test changes**: Validate new limits with test workflows

## Troubleshooting

### Workflows Queuing

**Symptom**: Workflows wait in queue instead of running

**Cause**: All concurrency slots are full

**Solution**: 
- Increase `max-concurrency` value
- Check for long-running workflows
- Consider using different engines for different workflows

### Too Many Concurrent Runs

**Symptom**: High costs or resource usage

**Cause**: `max-concurrency` set too high

**Solution**:
- Decrease `max-concurrency` value
- Monitor usage patterns
- Set appropriate limits per engine

### Workflows Not Canceling

**Symptom**: Old pull request workflows continue running after new commits

**Cause**: Custom concurrency without `cancel-in-progress`

**Solution**: Ensure pull request workflows have `cancel-in-progress: true` in custom concurrency configuration

## Related Documentation

- [AI Engines](/gh-aw/reference/engines/) - Engine configuration and capabilities
- [Frontmatter Options](/gh-aw/reference/frontmatter/) - Complete frontmatter reference
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Overall workflow organization
