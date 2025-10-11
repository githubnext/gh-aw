---
title: Concurrency Control
description: Complete guide to concurrency control in GitHub Agentic Workflows, including agent job concurrency configuration and engine isolation.
sidebar:
  order: 1300
---

GitHub Agentic Workflows provides concurrency control to manage how many AI-powered agentic workflow runs can run simultaneously. This helps prevent resource exhaustion, control costs, and ensure predictable workflow execution.

Concurrency control uses a dual-level approach:
- **Workflow-level concurrency**: Limit based on workflow name and type (name, issue, PR, branch, etc.)
- **Agent job concurrency**: Limit based on AI engine via the `engine.concurrency` field

This provides both fine-grained control per workflow and flexible resource management for AI execution.

## Workflow-Level Concurrency

By default, the workflow-level concurrency uses context-specific keys based on workflow name and the trigger type:

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

This controls concurrent execution of agentic workflow runs across all workflows, preventing resource exhaustion from too many concurrent AI executions.

### Complete Example

Here's how both levels work together in a workflow:

```yaml
name: "Issue Responder"
on:
  issues:
    types: [opened]

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

- Copilot agentic workflow runs use the `gh-aw-copilot` concurrency group
- Claude agentic workflow runs use the `gh-aw-claude` concurrency group
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

## Related Documentation

- [AI Engines](/gh-aw/reference/engines/) - Engine configuration and capabilities
- [Frontmatter Options](/gh-aw/reference/frontmatter/) - Complete frontmatter reference
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Overall workflow organization
