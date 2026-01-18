---
title: Campaign Flow & Lifecycle
description: What happens when the campaign orchestrator runs, and how to pause or finish a campaign.
banner:
  content: '<strong>Do not use.</strong> Campaigns are still in active development and may have unexpected consequences.'
---

This page explains what the orchestrator does when it runs, and what you do to pause or end a campaign.

## Lifecycle states

Campaign specs include a `state` field.

| State | Meaning |
| --- | --- |
| `planned` | Drafting and review; not intended to run yet |
| `active` | Running on schedule |
| `paused` | Temporarily stopped |
| `completed` | Finished and no longer running |
| `archived` | Kept for reference only |

> [!CAUTION]
> The current implementation does not automatically disable workflows based on `state`. To pause/stop execution, disable the workflow in the GitHub Actions UI.

## What happens on each run

At a high level, the orchestrator:

1. (Optional) dispatches worker workflows via `workflow_dispatch`
2. discovers relevant issues/PRs
3. updates the GitHub Project (within governance limits)
4. posts a Project status update (summary + next steps)

## Dispatching worker workflows

If your campaign lists `workflows`, the orchestrator dispatches them sequentially.

> [!NOTE]
> Dispatch is fire-and-forget: the orchestrator does not wait for worker workflows to finish. Results are picked up on later runs.

### Worker workflow requirements

- The workflow must exist (compiled `.lock.yml` or standard `.yml`).
- The workflow must support `workflow_dispatch`.
- If the campaign is responsible for running it, remove other triggers (cron/push) to avoid duplicates.

## Pausing and ending a campaign

### Pause

1. Set `state: paused` in the campaign spec (for clarity).
2. Disable the orchestrator workflow in GitHub Actions.

### Finish

1. Set `state: completed` (or `archived`).
2. Disable the orchestrator workflow.

> [!TIP]
> Consider running the orchestrator one last time before disabling it, so the Project gets a final status update.

## When something goes wrong

Campaigns are designed to keep going and report what happened in the Project status update.

- **Dispatch failed**: fix the worker workflow (missing, not dispatchable), then wait for the next run.
- **Project updates hit a limit**: increase governance limits or let the campaign catch up over multiple runs.
- **Permissions errors**: ensure the workflow token has the required Projects permissions.

<details>
<summary>Implementation notes (advanced)</summary>

The orchestrator precomputes discovery before the agent phase and uses budgets to avoid scanning too much in one run. It can write a discovery manifest under `./.gh-aw/` for deterministic processing.
</details>
 
## Limits and governance

Campaigns typically enforce per-run budgets (for discovery, project updates, comments, etc.) so a run can’t “do everything at once”. When a budget is reached, the campaign reports what was deferred and continues on the next run.

See [Campaign Specs](/gh-aw/guides/campaigns/specs/) for the `governance` fields.

**Example scenario**:

```yaml
# Campaign spec
governance:
  max-discovery-items-per-run: 50
  max-project-updates-per-run: 10
```

**Run 1**:

- Discovers: 50 items (budget reached)
- Processes: 10 items (budget reached)
- Deferred: 40 items

**Run 2**:
- Discovers: 50 more items (starting from cursor)
- Processes: 10 items (from deferred 40 + newly discovered)
- Deferred: 30 + 50 = 80 items
- Cursor: Saved at item 100

**Run 3-N**: Continues until all items processed

### Should the Campaign Stop?

**No** - campaigns do NOT stop when max items are reached. They continue processing incrementally:

- Discovery budget limits **pace** the work (prevents overwhelming API)
- Project update limits **throttle** writes (prevents project board spam)
- Cursor-based pagination ensures **all items are eventually processed**

**Campaign stops only when**:

1. All discovered items have been processed
2. No new items are being created by workers
3. State is changed to `completed` or `archived` (manual action)
4. Orchestrator workflow is disabled/deleted (manual action)

## Campaign Ending & Termination

### How to End a Campaign

Campaigns are ended through **manual actions** - there is no automatic termination:

#### Option 1: Update State to `completed`

Edit the campaign spec (`.campaign.md`):

```yaml
---
id: security-q1-2025
name: Security Q1 2025
state: completed  # Changed from 'active'
---
```
1. Compile: `gh aw compile security-q1-2025`
2. Commit updated files
3. **Manually disable** the workflow in GitHub Actions UI

**Important**: Changing state to `completed` does NOT automatically stop execution - you must disable the workflow.

#### Option 2: Disable the Workflow

In GitHub UI:

1. Go to Actions → Workflows
2. Select the campaign orchestrator workflow
3. Click "Disable workflow" (three-dot menu)

**Effect**: Scheduled executions stop, but workflow can be manually triggered.

#### Option 3: Delete Workflow Files

Remove the campaign workflow files:

git commit -m "End security-q1-2025 campaign"
git push
```

**Effect**: Workflow is completely removed and cannot execute.

### Archive Completed Campaigns

For historical reference, use the `archived` state:

```yaml
---
id: security-q1-2025
name: Security Q1 2025
state: archived
---

# Campaign completed on 2025-03-15

Final metrics:

- Tasks completed: 200/200
- Duration: 90 days
- Final velocity: 7.5 tasks/day
```

**Best practice**: Keep `.campaign.md` file with `state: archived` but delete `.campaign.lock.yml` to prevent accidental execution.

### Final Status Update

Before ending a campaign, run the orchestrator one final time to generate the **completion status update**:

```yaml
create-project-status-update:
  project: "https://github.com/orgs/ORG/projects/1"
  status: "COMPLETE"
  start_date: "2024-12-15"
  target_date: "2025-03-15"
  body: |
    ## Campaign Complete
    
    The Security Q1 2025 campaign has successfully completed all objectives.
    
    ## Final Metrics
    
    - **Total tasks**: 200/200 (100%)
    - **Duration**: 90 days
    - **Average velocity**: 7.5 tasks/day
    
    ## KPI Achievement
    
    **Vulnerabilities Resolved** (Primary KPI):
    - Baseline: 0% → Final: 100% → Target: 100%
    - Status: ✅ TARGET ACHIEVED
    
    **Mean Time to Resolution** (Supporting KPI):
    - Baseline: 14 days → Final: 3 days → Target: 5 days
    - Status: ✅ TARGET EXCEEDED
    
    ## Lessons Learned
    
    1. Automated vulnerability scanning reduced manual triage time by 80%
    2. Dependency upgrades prevented 15 potential security incidents
    3. Worker workflows enabled consistent, repeatable processes
    
    ## Next Steps
    
    - Archive campaign materials to `memory/campaigns/security-q1-2025/archive/`
    - Transition ongoing vulnerability monitoring to BAU workflows
    - Plan follow-up campaign for infrastructure modernization
```

### What Happens to Repo-Memory

Campaign repo-memory (cursor, metrics snapshots) is preserved when a campaign ends:

**Cursor file**: `memory/campaigns/<id>/cursor.json`

- Remains at final position
- Can be used for historical reference
- Not automatically deleted

**Metrics snapshots**: `memory/campaigns/<id>/metrics/*.json`

- Append-only history preserved
- Valuable for retrospectives and trend analysis
- Should be retained for organizational learning

**Best practice**: Keep repo-memory indefinitely for historical analysis and reporting.

## Pre-existing Workflows & Trigger Behavior

### Critical Requirement: Trigger Management

**When a campaign executes a workflow** (workflow is listed in campaign's `workflows` field), the workflow's original triggers (cron schedules, push events, pull_request events) **must be disabled**.

The campaign orchestrator controls execution timing via `workflow_dispatch`, and keeping other triggers active would cause:

- Duplicate executions (original trigger + campaign trigger)
- Resource waste and potential conflicts
- Loss of campaign control over execution timing

> [!CAUTION]
> **Required workflow trigger configuration:**
>
> ```yaml
> on:
>   workflow_dispatch:  # ONLY this trigger for campaign-executed workflows
> ```

**Alternative approach**: If a workflow should keep its original triggers, **do not add it to the campaign's `workflows` list**. Instead, let it run independently and have the campaign discover its outputs via tracker labels.

### Campaign Impact on Existing Workflows

A critical aspect of campaigns is understanding how they interact with workflows that have existing triggers (cron jobs, push events, etc.).

### Worker Workflows with Cron Jobs

**Scenario**: A repository has an existing workflow that runs on a schedule:

```yaml
# .github/workflows/daily-dependency-check.md
---
name: Daily Dependency Check
on: daily  # Runs once per day at automatically scattered time
  workflow_dispatch:
---
```

**When picked up by campaign**:

The campaign spec references this workflow:

```yaml
# .github/workflows/security-audit.campaign.md
---
id: security-audit
workflows:
  - daily-dependency-check
---
```

### Required: Disable Original Triggers

**IMPORTANT**: When a campaign picks up an existing workflow for execution, **you must disable the workflow's original triggers** (cron schedules, push events, etc.). The campaign orchestrator will control when the workflow runs.

Campaign orchestrators execute workflows programmatically using `workflow_dispatch`:

```yaml
# In orchestrator Phase 0
- name: Dispatch worker workflow
  uses: ./actions/safe-output
  with:
    type: daily_dependency_check
```

**Why disable original triggers?**

- Prevents duplicate executions (campaign + original schedule)
- Ensures campaign has full control over execution timing
- Avoids resource waste and potential conflicts
- Maintains clear ownership of workflow execution

#### How to Disable Original Triggers

Modify the worker workflow to remove/comment out the original trigger:

```yaml
# .github/workflows/daily-dependency-check.md
---
name: Daily Dependency Check
on:
  # schedule: daily  # DISABLED - controlled by campaign
  workflow_dispatch:  # REQUIRED - allows campaign to trigger workflow
---
```

**Result**: Workflow only runs when triggered by campaign orchestrator.

#### Alternative: Keep Workflow Independent

If the workflow should continue running on its own schedule (not controlled by campaign), **do not add it to the campaign's `workflows` list**. Instead, let it run independently and have the campaign discover its outputs via tracker labels:

```yaml
# .github/workflows/security-audit.campaign.md
---
id: security-audit
tracker-label: "campaign:security-audit"
workflows:
  - vulnerability-scanner  # Only workflows the campaign should execute
---
```

**How it works**:

- `daily-dependency-check` keeps its cron schedule and runs independently
- It creates issues/PRs with the tracker label `campaign:security-audit`
- Campaign orchestrator discovers these items via the tracker label
- Campaign tracks progress without executing the workflow

**Effect**: Campaign does not execute this workflow; it runs independently but campaign tracks its outputs.

### Push/PR Triggers

**Scenario**: A workflow has push or pull_request triggers:

```yaml
# .github/workflows/code-quality-check.md
---
name: Code Quality Check
on:
  push:
    branches: [main]
  pull_request:
  workflow_dispatch:
---
```

**IMPORTANT**: If you want the campaign to execute this workflow, you **must remove the push/PR triggers**:

```yaml
# .github/workflows/code-quality-check.md
---
name: Code Quality Check
on:
  # push:                    # DISABLED - controlled by campaign
  #   branches: [main]
  # pull_request:            # DISABLED - controlled by campaign
  workflow_dispatch:         # REQUIRED for campaign execution
---
```

**However**, push/PR triggers are usually **inappropriate for campaigns** because:

- Code quality checks should respond to code changes (push/PR events)
- Campaign schedules (e.g., daily) don't align with code change events
- The workflow's purpose (event-driven validation) conflicts with campaign control

**Recommended approach**: Do NOT add event-driven workflows to campaign's `workflows` list. Instead:

- Let them run independently on their original triggers
- Have the campaign discover their outputs via tracker labels

```yaml
# .github/workflows/quality-initiative.campaign.md
---
id: quality-initiative
tracker-label: "campaign:quality-initiative"
workflows:
  # code-quality-check NOT listed - runs independently on push/PR
  - quality-reporter  # Only campaign-controlled workflows here
---
```

**Result**: Event-driven workflows continue responding to code changes, while campaign tracks their outputs.

### Campaign Item Protection

A related concern is preventing non-campaign workflows from interfering with campaign-tracked items.

#### How Protection Works

Items with campaign labels (`campaign:*`) are automatically excluded from other workflows:

```javascript
// Example from issue-monster workflow
if (issueLabels.some(label => label.startsWith('campaign:'))) {
  core.info(`Skipping #${issue.number}: has campaign label`);
  return false;
}
```

**Protection mechanisms**:

1. **Automatic labeling**: When campaign adds items to project, applies `campaign:<id>` label
2. **Workflow filtering**: Other workflows skip items with `campaign:` labels
3. **Opt-out labels**: `no-bot`, `no-campaign` provide additional exclusion

#### Example: Issue Monster vs Campaign

**Scenario**: Both `issue-monster` and a campaign workflow process issues.

**Without protection**:

- Issue monster adds comment: "This issue needs attention"
- Campaign orchestrator adds comment: "Added to security-q1-2025 project"
- Result: Duplicate/conflicting actions

**With protection**:

- Campaign adds `campaign:security-q1-2025` label when adding to project
- Issue monster checks labels: `if (label.startsWith('campaign:'))` → skip
- Result: Only campaign orchestrator manages the issue

### Pre-existing Cron Jobs: Summary

| Scenario | Behavior | Recommendation |
| --- | --- | --- |
| **Worker with cron in campaign** | Workflow added to campaign's `workflows` list | **REQUIRED: Disable cron**, keep only `workflow_dispatch` |
| **Worker with push trigger in campaign** | Workflow added to campaign's `workflows` list | **REQUIRED: Remove push trigger**, or remove from campaign |
| **Worker with workflow_dispatch only** | Runs only when campaign triggers | ✅ Ideal for campaign workers |
| **Independent workflow with cron** | NOT in campaign's `workflows` list | Keep cron, campaign discovers outputs via tracker labels |

**Key requirement**: If a workflow is in the campaign's `workflows` list, it must have ONLY `workflow_dispatch` trigger. All other triggers (cron, push, pull_request) must be disabled to prevent duplicate executions.

## Summary: Complete Campaign Flow

### Startup (First Run)

1. **Orchestrator triggered** (schedule or manual)
2. **Discovery precomputation**: Searches GitHub, generates manifest
3. **Phase 0**: Creates Epic issue, adds to project
4. **Phase 1**: Reads discovery manifest, reads project board
5. **Phase 2**: Plans updates (apply budgets, deterministic order)
6. **Phase 3**: Writes updates to project board
7. **Phase 4**: Creates status update, reports initial state

### Ongoing Execution (Subsequent Runs)

1. **Orchestrator triggered** (daily schedule)
2. **Discovery precomputation**: Continues from cursor, finds new items
3. **Phase 0** (optional): Dispatches worker workflows if configured
4. **Phase 1**: Reads manifest + project state
5. **Phase 2**: Plans updates for discovered items
6. **Phase 3**: Writes up to max updates
7. **Phase 4**: Reports progress, KPIs, deferred items

### Incident Handling Summary

- **Discovery failure**: Partial results used, cursor not advanced, retry next run
- **Workflow dispatch failure**: Logged, other workflows continue, reported in status
- **Update failure**: Individual item failure recorded, processing continues
- **Limit reached**: Remaining items deferred, status update explains

### Max Items Budget

- **Discovery limit**: Stops discovery early, saves cursor
- **Update limit**: Processes first N items, defers rest
- **Next run**: Continues from cursor with deferred items
- **Campaign never stops** due to limits - processes incrementally

### Campaign Ending

1. **Completion criteria**: All items processed, objectives met
2. **Manual action**: Update state to `completed`, disable workflow
3. **Final status update**: Reports completion, KPI achievement, lessons learned
4. **Archival**: Keep `.campaign.md` with `state: archived`, delete `.campaign.lock.yml`
5. **Repo-memory**: Preserved for historical reference

### Pre-existing Workflows

- **REQUIRED**: Workflows in campaign's `workflows` list must have ONLY `workflow_dispatch` trigger
- **Original triggers must be disabled**: Cron, push, and PR triggers must be removed/commented out
- **Alternative**: Keep workflows independent with their triggers, let campaign discover outputs via tracker labels
- **Protection**: Campaign labels prevent interference from other workflows
- **Key rule**: Campaign-executed workflows = `workflow_dispatch` only; Independent workflows = keep their triggers

</details>

## Further Reading

- [Campaign Specs](/gh-aw/guides/campaigns/specs/) - Configuration reference
- [Getting Started](/gh-aw/guides/campaigns/getting-started/) - Create your first campaign
- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Project automation operations
