---
title: Campaign Lifecycle
description: Campaign execution phases, state management, and workflow coordination
banner:
  content: '<strong>Do not use.</strong> Campaigns are still incomplete and may produce unreliable or unintended results.'
---

Campaign orchestrators execute on a schedule to coordinate worker workflows, discover outputs, and update project boards. This page explains the execution model, state management, and workflow coordination.

## Execution flow

```mermaid
graph TD
    A[Orchestrator Triggered] --> B[Phase 0: Dispatch Workers]
    B --> C[Phase 1: Discover Items]
    C --> D[Phase 2: Plan Updates]
    D --> E[Phase 3: Apply Updates]
    E --> F[Phase 4: Status Report]
```

Each run follows this sequence:

1. **Phase 0** - Dispatches worker workflows via `workflow_dispatch` (if configured)
2. **Phase 1** - Discovers issues and pull requests with campaign tracker labels
3. **Phase 2** - Plans project board updates within governance limits
4. **Phase 3** - Applies updates to project board
5. **Phase 4** - Posts status update to project with progress summary

## Campaign states

| State | Description | Execution |
|-------|-------------|-----------|
| `planned` | Draft configuration under review | Not running |
| `active` | Production campaign | Runs on schedule |
| `paused` | Temporarily stopped | Not running |
| `completed` | Objectives achieved | Not running |
| `archived` | Historical reference | Not running |

:::caution
The `state` field is documentation only. To stop execution, disable the workflow in GitHub Actions settings.
:::

## Worker workflows

Worker workflows perform campaign tasks (scanning, analysis, remediation). The orchestrator dispatches them via `workflow_dispatch` and discovers their outputs.

### Requirements

Worker workflows in the campaign's `workflows` list must:

- Accept `workflow_dispatch` as the **only** trigger
- Remove all other triggers (`schedule`, `push`, `pull_request`)
- Label created items with `campaign:<id>`
- Accept standardized inputs: `campaign_id` (string) and `payload` (string JSON)

```yaml
# Worker workflow configuration
on:
  workflow_dispatch:
    inputs:
      campaign_id:
        description: 'Campaign identifier'
        required: true
        type: string
      payload:
        description: 'JSON payload with work details'
        required: true
        type: string
```

### Independent workflows

Workflows not in the `workflows` list can keep their original triggers. The campaign discovers their outputs via tracker labels without controlling execution.

```yaml
# Campaign spec
tracker-label: "campaign:security-audit"
workflows:
  - vulnerability-scanner  # Orchestrator controls this one
  # dependency-check runs independently with its cron schedule
```

## Discovery and governance

Discovery finds items created by workers based on tracker labels. Governance limits control the pace of work.

```yaml
governance:
  max-discovery-items-per-run: 50
  max-project-updates-per-run: 10
```

When limits are reached:
- Discovery cursor saves the current position
- Remaining items are deferred to the next run
- Status update reports deferred count
- Campaign continues on next schedule

The campaign processes items incrementally across multiple runs until all are handled.

## Pausing and ending campaigns

### Pause temporarily

1. Update spec: `state: paused`
2. Disable workflow in Actions settings

### Complete permanently

1. Run orchestrator one final time for completion status
2. Update spec: `state: completed`
3. Disable workflow in Actions settings
4. Optionally delete `.campaign.lock.yml` (keep `.campaign.md` for history)

### Archive for reference

```yaml
---
id: security-q1-2025
state: archived
---

Completed 2025-03-15. Final metrics:
- Tasks: 200/200
- Duration: 90 days
- Velocity: 7.5 tasks/day
```

## Troubleshooting

**Worker dispatch fails**
- Verify workflow exists and has `workflow_dispatch` trigger
- Check workflow file name matches spec
- Ensure no compilation errors in worker

**Discovery finds no items**
- Verify tracker label matches campaign ID
- Check workers are creating items with correct labels
- Confirm discovery scope includes correct repos/orgs

**Project updates hit limit**
- Increase `max-project-updates-per-run` in governance
- Accept incremental processing across multiple runs
- Verify project token has required permissions

**Items processed multiple times**
- Ensure workers use deterministic keys
- Check for duplicate labels on items
- Verify idempotency logic in worker code

## Advanced: Pre-existing workflows

### Converting scheduled workflows

When adding an existing scheduled workflow to a campaign:

**Before** (independent):
```yaml
on:
  schedule: daily
  workflow_dispatch:
```

**After** (campaign-controlled):
```yaml
on:
  workflow_dispatch:  # Only this trigger
  # schedule: daily   # Removed - campaign controls timing
```

### Event-driven workflows

Workflows triggered by code events (`push`, `pull_request`) should not be campaign-controlled. These respond to specific events, not campaign schedules.

**Recommended**: Keep them independent and let the campaign discover their outputs.

**Not recommended**: Adding them to campaign's `workflows` list requires removing event triggers, which defeats their purpose.

## Further reading

- [Campaign specs](/gh-aw/guides/campaigns/specs/) - Configuration reference
- [Getting started](/gh-aw/guides/campaigns/getting-started/) - Create your first campaign
- [CLI commands](/gh-aw/guides/campaigns/cli-commands/) - Management commands
