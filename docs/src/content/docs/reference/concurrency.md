---
title: Concurrency Control
description: Complete guide to concurrency control in GitHub Agentic Workflows, including max-concurrency configuration, global locks, and engine isolation.
sidebar:
  order: 6
---

GitHub Agentic Workflows provides sophisticated concurrency control to manage how many AI-powered workflows can run simultaneously. This helps prevent resource exhaustion, control costs, and ensure predictable workflow execution.

## Overview

Concurrency control in GitHub Agentic Workflows uses a dual-level approach with different strategies at each level:
- **Workflow-level concurrency**: Context-specific limiting based on workflow type (issue, PR, branch, etc.)
- **Job-level concurrency (max-concurrency)**: Global limiting across all workflows using the same engine

This dual-level approach provides both fine-grained control per workflow and global resource management across all workflows.

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

### Workflow-Level Concurrency

The workflow-level concurrency uses context-specific keys based on the trigger type:

**For issue workflows:**
```yaml
concurrency:
  group: "gh-aw-${{ github.workflow }}-${{ github.event.issue.number }}"
```

**For pull request workflows:**
```yaml
concurrency:
  group: "gh-aw-${{ github.workflow }}-${{ github.event.pull_request.number || github.ref }}"
  cancel-in-progress: true
```

**For push workflows:**
```yaml
concurrency:
  group: "gh-aw-${{ github.workflow }}-${{ github.ref }}"
```

**For schedule/other workflows:**
```yaml
concurrency:
  group: "gh-aw-${{ github.workflow }}"
```

This ensures workflows operating on different issues, PRs, or branches can run concurrently without interfering with each other.

### Job-Level Concurrency (Max-Concurrency)

The job-level concurrency uses **only** the engine ID and slot number for global limiting:

```yaml
jobs:
  agent:
    concurrency:
      group: "gh-aw-{engine-id}-${{ github.run_id % max-concurrency }}"
```

**Example for Claude with max-concurrency of 5:**
```yaml
jobs:
  agent:
    concurrency:
      group: "gh-aw-claude-${{ github.run_id % 5 }}"
```

This creates a global lock across **all workflows and refs** for each engine, preventing resource exhaustion from too many concurrent AI executions.

### Complete Example

Here's how both levels work together in a generated workflow:

```yaml
name: "Issue Responder"
on:
  issues:
    types: [opened]

permissions: {}

# Workflow-level: Context-specific concurrency
concurrency:
  group: "gh-aw-${{ github.workflow }}-${{ github.event.issue.number }}"

jobs:
  agent:
    runs-on: ubuntu-latest
    permissions: read-all
    # Job-level: Global max-concurrency limiting
    concurrency:
      group: "gh-aw-claude-${{ github.run_id % 5 }}"
    steps:
      - name: Execute workflow
        ...
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

The dual-level concurrency provides complementary control:

1. **Workflow-level**: Prevents conflicts between runs of the same workflow on different contexts (e.g., different issues or PRs)
2. **Job-level**: Prevents resource exhaustion by limiting total concurrent AI executions across all workflows

**Example scenario:**
- 5 different issues trigger the same workflow
- Workflow-level concurrency allows all 5 to start (different issue numbers)
- Job-level max-concurrency (e.g., 3) ensures only 3 AI jobs run simultaneously
- The other 2 workflows queue until slots become available

This approach balances:
- **Workflow isolation**: Different contexts don't block each other at the workflow level
- **Global resource management**: Total AI resource usage is controlled at the job level

## Global Lock Behavior

The **job-level** concurrency (max-concurrency) uses **only** engine ID and slot number, creating a true global lock:

### What's Included in Job-Level Concurrency
- ✅ Engine ID (`copilot`, `claude`, `codex`)
- ✅ Slot number (from `run_id % max-concurrency`)
- ✅ `gh-aw-` prefix

### What's NOT Included in Job-Level Concurrency
- ❌ Workflow name
- ❌ Issue number
- ❌ Pull request number
- ❌ Branch/ref name
- ❌ Event type

This ensures the max-concurrency limit applies **repository-wide** across all workflows and refs for each engine.

### Workflow-Level Concurrency Includes Context

The **workflow-level** concurrency includes context-specific information:
- ✅ Workflow name
- ✅ Issue/PR/discussion number (when applicable)
- ✅ Branch ref (for push workflows)
- ✅ `gh-aw-` prefix

This allows different contexts to run concurrently while preventing conflicts within the same context.

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
