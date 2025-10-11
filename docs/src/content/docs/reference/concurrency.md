---
title: Concurrency Control
description: Complete guide to concurrency control in GitHub Agentic Workflows, including agent job concurrency configuration and engine isolation.
sidebar:
  order: 600
---

GitHub Agentic Workflows provides sophisticated concurrency control to manage how many AI-powered agent jobs can run simultaneously. This helps prevent resource exhaustion, control costs, and ensure predictable workflow execution.

## Overview

Concurrency control in GitHub Agentic Workflows uses a dual-level approach:
- **Workflow-level concurrency**: Context-specific limiting based on workflow type (issue, PR, branch, etc.)
- **Agent job concurrency**: Controls concurrent execution of agent jobs using the `engine.concurrency` field

This dual-level approach provides both fine-grained control per workflow and flexible resource management for AI execution.

## Agent Job Concurrency Configuration

The `concurrency` field under the `engine` section controls concurrency for the agent job. It uses GitHub Actions concurrency syntax:

```yaml
engine:
  id: claude
  concurrency:
    group: "my-group-${{ github.workflow }}"
    cancel-in-progress: true
```

### Default Behavior

**Default:** Single job per engine across all workflows

When no `engine.concurrency` is specified, the default pattern is:
```yaml
concurrency:
  group: "gh-aw-{engine-id}"
```

This ensures only one agent job runs at a time for each engine across all workflows and refs, preventing resource exhaustion.

### Configuration Examples

**Default (single job per engine):**
```yaml
engine:
  id: claude
  # No concurrency specified - uses gh-aw-claude
```

**Per-workflow concurrency:**
```yaml
engine:
  id: claude
  concurrency:
    group: "gh-aw-claude-${{ github.workflow }}"
```

**Per-branch concurrency with cancellation:**
```yaml
engine:
  id: copilot
  concurrency:
    group: "gh-aw-copilot-${{ github.ref }}"
    cancel-in-progress: true
```

**Simple string format:**
```yaml
engine:
  id: claude
  concurrency: "custom-group-${{ github.workflow }}"
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

### Agent Job Concurrency

The agent job concurrency is configured via `engine.concurrency` and uses the specified pattern:

**Default pattern (single job per engine):**
```yaml
jobs:
  agent:
    concurrency:
      group: "gh-aw-{engine-id}"
```

**Custom pattern example:**
```yaml
jobs:
  agent:
    concurrency:
      group: "custom-${{ github.workflow }}"
      cancel-in-progress: true
```

This controls concurrent execution of agent jobs across all workflows, preventing resource exhaustion from too many concurrent AI executions.

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
    # Agent concurrency: Default single job per engine
    concurrency:
      group: "gh-aw-claude"
    steps:
      - name: Execute workflow
        ...
```

### Dual-Level Application

The dual-level concurrency provides complementary control:

1. **Workflow-level**: Prevents conflicts between runs of the same workflow on different contexts (e.g., different issues or PRs)
2. **Agent job concurrency**: Controls concurrent execution of agent jobs based on configured pattern

**Example scenario:**
- 5 different issues trigger the same workflow
- Workflow-level concurrency allows all 5 to start (different issue numbers)
- Agent job concurrency with default `gh-aw-claude` means only 1 agent job runs at a time
- The other 4 workflows queue until the agent job completes

This approach balances:
- **Workflow isolation**: Different contexts don't block each other at the workflow level
- **Resource management**: Agent job execution is controlled via concurrency configuration

## Global Lock Behavior

The **agent job concurrency** creates a lock based on the configured pattern:

### What's Included in Default Agent Concurrency
- ✅ Engine ID (`copilot`, `claude`, `codex`)
- ✅ `gh-aw-` prefix

### What's NOT Included in Default Agent Concurrency
- ❌ Workflow name
- ❌ Issue number
- ❌ Pull request number
- ❌ Branch/ref name
- ❌ Event type

The default pattern `gh-aw-{engine-id}` ensures only one agent job runs per engine across **all workflows and refs**.

You can customize this behavior by specifying a different `engine.concurrency` pattern:
```yaml
engine:
  id: claude
  concurrency:
    group: "gh-aw-claude-${{ github.workflow }}"  # Per-workflow concurrency
```

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
  # Default: gh-aw-copilot

# Workflow B uses Claude
engine:
  id: claude
  # Default: gh-aw-claude
```

- Copilot agent jobs use the `gh-aw-copilot` concurrency group
- Claude agent jobs use the `gh-aw-claude` concurrency group
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
- **Prevents runaway costs**: Controls concurrent AI job execution
- **Predictable resource usage**: Known concurrency patterns
- **Flexible configuration**: Customize per workflow or engine

### Resource Management
- **Prevents resource exhaustion**: Ensures system stability with default single-job-per-engine pattern
- **Fair resource distribution**: Agent jobs queue when concurrency limit is reached
- **Maintains throughput**: Activation and other jobs continue running

### Engine Isolation
- **Independent limits**: Each engine has its own default concurrency group
- **No cross-engine interference**: Copilot agent jobs don't block Claude agent jobs
- **Flexible configuration**: Customize concurrency per engine

### Simplicity
- **Default global lock**: Single job per engine by default
- **Standard GitHub Actions syntax**: Familiar concurrency configuration
- **Consistent behavior**: Predictable execution patterns

## Custom Concurrency

You can override the automatic concurrency generation by specifying your own `concurrency` section in the frontmatter (for workflow-level concurrency):

```yaml
---
on: push
concurrency:
  group: custom-group-${{ github.ref }}
  cancel-in-progress: true
engine:
  id: claude
  concurrency:  # Agent job concurrency (separate from workflow concurrency)
    group: "custom-agent-${{ github.workflow }}"
tools:
  github:
    allowed: [list_issues]
---
```

**Note**: Workflow-level concurrency and agent job concurrency are independent and can be configured separately.

## Best Practices

### Configuring Agent Concurrency

**Default (recommended for most cases):**
```yaml
engine:
  id: claude
  # No concurrency specified - single job per engine
```

**Allow per-workflow concurrency:**
```yaml
engine:
  id: claude
  concurrency:
    group: "gh-aw-claude-${{ github.workflow }}"
```

**Per-branch concurrency for PR workflows:**
```yaml
engine:
  id: copilot
  concurrency:
    group: "gh-aw-copilot-${{ github.ref }}"
    cancel-in-progress: true
```

### Different Patterns for Different Engines

**Conservative engine (default):**
```yaml
engine:
  id: claude
  # No concurrency - uses gh-aw-claude (single job)
```

**More permissive engine:**
```yaml
engine:
  id: copilot
  concurrency:
    group: "gh-aw-copilot-${{ github.workflow }}"  # Per-workflow concurrency
```

### Monitoring and Adjustment

1. **Monitor workflow execution**: Use GitHub Actions insights
2. **Track costs**: Review AI model usage and expenses
3. **Adjust patterns**: Change concurrency groups based on needs
4. **Test changes**: Validate new patterns with test workflows

## Troubleshooting

### Agent Jobs Queuing

**Symptom**: Agent jobs wait in queue instead of running

**Cause**: Concurrency group is blocking (e.g., default single-job-per-engine pattern)

**Solution**: 
- Customize `engine.concurrency` to allow more parallel execution
- Use per-workflow or per-branch patterns if appropriate
- Consider using different engines for different workflows

### Too Many Concurrent Runs

**Symptom**: High costs or resource usage from concurrent agent jobs

**Cause**: Concurrency pattern allows too many parallel executions

**Solution**:
- Use more restrictive concurrency pattern (e.g., default `gh-aw-{engine-id}`)
- Monitor usage patterns
- Set appropriate patterns per engine

### Workflows Not Canceling

**Symptom**: Old pull request workflows continue running after new commits

**Cause**: Custom concurrency without `cancel-in-progress`

**Solution**: Ensure pull request workflows have `cancel-in-progress: true` in custom concurrency configuration

## Related Documentation

- [AI Engines](/gh-aw/reference/engines/) - Engine configuration and capabilities
- [Frontmatter Options](/gh-aw/reference/frontmatter/) - Complete frontmatter reference
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Overall workflow organization
