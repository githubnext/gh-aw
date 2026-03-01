---
title: Concurrency Control
description: Complete guide to concurrency control in GitHub Agentic Workflows, including agent job concurrency configuration and engine isolation.
sidebar:
  order: 1400
---

GitHub Agentic Workflows uses dual-level concurrency control to prevent resource exhaustion and ensure predictable execution:
- **Per-workflow**: Limits based on workflow name and trigger context (issue, PR, branch)
- **Per-engine**: Limits AI execution across all workflows via `engine.concurrency`

## Per-Workflow Concurrency

Workflow-level concurrency groups include the workflow name plus context-specific identifiers:

| Trigger Type | Concurrency Group | Cancel In Progress |
|--------------|-------------------|-------------------|
| Issues | `gh-aw-${{ github.workflow }}-${{ issue.number }}` | No |
| Pull Requests | `gh-aw-${{ github.workflow }}-${{ pr.number \|\| ref }}` | Yes (new commits cancel outdated runs) |
| Push | `gh-aw-${{ github.workflow }}-${{ github.ref }}` | No |
| Schedule/Other | `gh-aw-${{ github.workflow }}` | No |

This ensures workflows on different issues, PRs, or branches run concurrently without interference.

## Per-Engine Concurrency

The default per-engine pattern `gh-aw-{engine-id}` ensures only one agent job runs per engine across all workflows, preventing AI resource exhaustion. The group includes only the engine ID and `gh-aw-` prefix - workflow name, issue/PR numbers, and branches are excluded.

```yaml wrap
jobs:
  agent:
    concurrency:
      group: "gh-aw-{engine-id}"
```

## Custom Concurrency

Override either level independently:

```yaml wrap
---
on: push
concurrency:  # Workflow-level
  group: custom-group-${{ github.ref }}
  cancel-in-progress: true
engine:
  id: copilot
  concurrency:  # Engine-level
    group: "gh-aw-copilot-${{ github.workflow }}"
tools:
  github:
    allowed: [list_issues]
---
```

## Safe Outputs Job Concurrency

The `safe_outputs` job runs independently from the agent job and can process outputs concurrently across workflow runs. Use `safe-outputs.concurrency-group` to serialize access when needed:

```yaml wrap
safe-outputs:
  concurrency-group: "safe-outputs-${{ github.repository }}"
  create-issue:
```

When set, the `safe_outputs` job uses `cancel-in-progress: false` — meaning queued runs wait for the in-progress run to finish rather than being cancelled. This is useful for workflows that create issues or pull requests where duplicate operations would be undesirable.

See [Safe Outputs](/gh-aw/reference/safe-outputs/#safe-outputs-job-concurrency-concurrency-group) for details.

## Related Documentation

- [AI Engines](/gh-aw/reference/engines/) - Engine configuration and capabilities
- [Frontmatter](/gh-aw/reference/frontmatter/) - Complete frontmatter reference
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Overall workflow organization
- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Safe output processing and job configuration
