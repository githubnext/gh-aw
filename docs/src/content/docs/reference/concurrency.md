---
title: Concurrency Control
description: Complete guide to concurrency control in GitHub Agentic Workflows, including agent job concurrency configuration and engine isolation.
sidebar:
  order: 1400
---

GitHub Agentic Workflows provides concurrency control to manage how many AI-powered agentic workflow runs can run simultaneously. This helps prevent resource exhaustion, control costs, and ensure predictable workflow execution.

Concurrency control uses a dual-level approach:
- **Per-workflow concurrency**: Limit based on workflow name and type (name, issue, PR, branch, etc.)
- **Per-engine concurrency**: Limit based on AI engine via the `engine.concurrency` field

This provides both fine-grained control per workflow and flexible resource management for AI execution.

## Per-Workflow Concurrency

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

This ensures workflows operating on different issues, PRs, or branches can run concurrently without interfering with each other. The **workflow-level** concurrency includes context-specific information:
- ✅ Workflow name
- ✅ Issue/PR/discussion number (when applicable)
- ✅ Branch ref (for push workflows)
- ✅ `gh-aw-` prefix

Concurrency cancellation varies by workflow trigger type:

| Trigger Type | `cancel-in-progress:` | Reason |
|--------------|-------------------|--------|
| `pull_request` | ✅ Enabled | New commits should cancel outdated PR runs |
| All other triggers | ❌ Disabled | Issue/discussion workflows should run to completion |

### Per-engine Concurrency

The AI engine concurrency is configured via `engine.concurrency` and uses the specified pattern:

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

### What's Included in Default Per-engine Concurrency
- ✅ Engine ID (`copilot`, `claude`, `codex`)
- ✅ `gh-aw-` prefix

### What's NOT Included in Default Per-engine Concurrency
- ❌ Workflow name
- ❌ Issue number
- ❌ Pull request number
- ❌ Branch/ref name
- ❌ Event type

The default pattern `gh-aw-{engine-id}` ensures only one agent job runs per engine across **all workflows and refs**.

## Custom Concurrency

You can override per-engine concurrency by specifying a different `engine.concurrency` pattern:
```yaml
engine:
  id: claude
  concurrency:
    group: "gh-aw-claude-${{ github.workflow }}"  # Per-workflow concurrency
```

You can also override per-workflow concurrency by specifying your own `concurrency` section in the frontmatter (for workflow-level concurrency):

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

## Related Documentation

- [AI Engines](/gh-aw/reference/engines/) - Engine configuration and capabilities
- [Frontmatter](/gh-aw/reference/frontmatter/) - Complete frontmatter reference
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Overall workflow organization
