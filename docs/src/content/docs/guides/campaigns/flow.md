---
title: Campaign Flow & Lifecycle
description: Complete guide to campaign execution flow, from startup to completion, including incident handling and governance.
---

This guide provides a comprehensive overview of how campaigns execute from start to finish, covering the complete lifecycle including startup, ongoing execution, incident handling, governance limits, and campaign termination.

## Campaign Lifecycle Overview

Campaigns progress through distinct lifecycle states that determine their behavior:

| State | Description | Orchestrator Behavior |
|-------|-------------|----------------------|
| **planned** | Campaign is being designed, not yet active | Workflows can be compiled but orchestrator should not execute automatically |
| **active** | Campaign is running normally | Orchestrator executes on schedule (daily by default) |
| **paused** | Campaign is temporarily suspended | Orchestrator can be manually triggered but should not run on schedule |
| **completed** | Campaign objectives achieved | Orchestrator stops executing, final status report generated |
| **archived** | Campaign is historical reference only | Workflows are retained but not executed |

> [!CAUTION]
> The current implementation does not automatically disable orchestrator execution based on state. Users must manually disable the workflow in GitHub Actions UI for `paused` campaigns, or delete/disable the `.campaign.lock.yml` workflow file for `completed` or `archived` campaigns.

## Campaign Startup

### When a Campaign Starts

A campaign begins execution when its orchestrator workflow runs for the first time. The orchestrator is triggered by:

1. **Scheduled execution**: Daily at 6 PM UTC by default (`cron: "0 18 * * *"`)
2. **Manual trigger**: Via `workflow_dispatch` from GitHub Actions UI
3. **Programmatic trigger**: Via GitHub API or `gh workflow run` command

> [!TIP]
> For testing, use manual triggers to run the orchestrator immediately without waiting for the scheduled time.

### Orchestrator Workflow Structure

The generated orchestrator (`.campaign.lock.yml`) consists of two main components:

#### 1. Discovery Precomputation (GitHub Script)

**Runs first** - before the AI agent executes:

```yaml
- name: Run campaign discovery precomputation
  id: discovery
  uses: actions/github-script@v8.0.0
  env:
    GH_AW_CAMPAIGN_ID: security-q1-2025
    GH_AW_WORKFLOWS: "vulnerability-scanner,dependency-updater"
    GH_AW_TRACKER_LABEL: campaign:security-q1-2025
    GH_AW_MAX_DISCOVERY_ITEMS: 200
    GH_AW_MAX_DISCOVERY_PAGES: 10
```

**Purpose**: 
- Queries GitHub for issues/PRs created by worker workflows or labeled with tracker label
- Enforces discovery budgets (max items, max pages) 
- Produces deterministic manifest at `./.gh-aw/campaign.discovery.json`
- Maintains pagination cursor in repo-memory for incremental discovery

> [!NOTE]
> The discovery precomputation runs **before** the AI agent, ensuring deterministic and budget-controlled discovery.

**Output**: Discovery manifest with normalized item metadata:

```json
{
  "schema_version": "v1",
  "campaign_id": "security-q1-2025",
  "generated_at": "2025-01-08T12:00:00.000Z",
  "project_url": "https://github.com/orgs/ORG/projects/1",
  "discovery": {
    "total_items": 42,
    "items_scanned": 100,
    "pages_scanned": 2,
    "max_items_budget": 200,
    "max_pages_budget": 10,
    "cursor": { "page": 3, "trackerId": "vulnerability-scanner" }
  },
  "summary": {
    "needs_add_count": 25,
    "needs_update_count": 17,
    "open_count": 25,
    "closed_count": 10,
    "merged_count": 7
  },
  "items": [
    {
      "url": "https://github.com/org/repo/issues/123",
      "content_type": "issue",
      "number": 123,
      "repo": "org/repo",
      "created_at": "2025-01-01T00:00:00Z",
      "updated_at": "2025-01-07T12:00:00Z",
      "state": "open",
      "title": "Upgrade dependency X"
    }
  ]
}
```

#### 2. Agent Coordination Job

**Runs second** - processes the discovery manifest:

The AI agent follows a strict execution sequence:

**Phase 0: Workflow Execution** (Optional - only if `workflows` configured):
- Check if configured workflows exist
- Create missing workflows (with testing requirement)
- Execute workflows sequentially
- Collect outputs from workflow runs

**Phase 1: Discovery** (Read-Only):
1. Read precomputed discovery manifest (`./.gh-aw/campaign.discovery.json`)
2. Read current GitHub Project board state (all items + fields)
3. Parse discovered items from manifest
4. Check summary counts to determine if work is needed

**Phase 2: Planning** (Read-Only):
5. Determine desired status from GitHub state:
   - Open issue/PR → `Todo` or `In Progress`
   - Closed issue → `Done`
   - Merged PR → `Done`
6. Calculate required date fields (`start_date`, `end_date`)
7. Apply write budget (max project updates per run)
8. Select items for this run using deterministic order (oldest updated_at first)

**Phase 3: Project Updates** (Write-Only):
9. For each selected item, send `update-project` request
10. Do NOT interleave reads and writes
11. Do NOT pre-check if item is on board (safe-output handles this)
12. Record per-item outcome (success/failure + error details)

**Phase 4: Status Reporting**:
13. **REQUIRED**: Create project status update using `create-project-status-update`
14. Report counts discovered, processed, deferred, failed
15. Report KPI trends and campaign progress
16. Document next steps

> [!IMPORTANT]
> Status reporting is **required** for every orchestrator run to maintain campaign visibility and stakeholder communication.

### First Run Behavior

On the **first orchestrator execution**, special initialization occurs:

#### Epic Issue Creation

The orchestrator creates a campaign Epic issue that serves as the parent for all work items:

```yaml
create-issue:
  title: "Security Q1 2025"
  body: |
    ## Campaign Overview
    
    **Objective**: Resolve all high-severity vulnerabilities
    
    This Epic issue tracks the overall progress of the campaign.
    
    **Campaign Details:**
    - Campaign ID: `security-q1-2025`
    - Project Board: https://github.com/orgs/ORG/projects/1
    - Worker Workflows: `vulnerability-scanner`, `dependency-updater`
    
    ---
    `campaign_id: security-q1-2025`
  labels:
    - epic
    - type:epic
```

The Epic is then added to the project board:

```yaml
update-project:
  project: "https://github.com/orgs/ORG/projects/1"
  campaign_id: "security-q1-2025"
  content_type: "issue"
  content_number: <EPIC_ISSUE_NUMBER>
  fields:
    status: "In Progress"
    campaign_id: "security-q1-2025"
    worker_workflow: "unknown"
    priority: "High"
```

**Subsequent runs**: Check if Epic exists, verify it's on board, but do not recreate.

## Incident Handling

### What Constitutes an "Incident"

In the context of campaigns, incidents are **failures during orchestrator execution**:

1. **Discovery failures**: API rate limits, network errors, malformed responses
2. **Workflow execution failures**: Worker workflow crashes, timeouts, invalid outputs
3. **Project update failures**: Invalid item URLs, permissions issues, project access errors
4. **Safe-output limit violations**: Exceeding max updates, max comments, etc.

### Failure Handling Strategy

Campaign orchestrators follow **fail-safe patterns** to ensure robustness:

#### Discovery Failures

**Scenario**: GitHub API returns 429 (rate limited) or times out

**Behavior**:
- Discovery script logs warning: `Reached discovery budget limits. Stopping discovery.`
- Partial results are written to manifest
- Cursor is NOT advanced (will retry same items next run)
- Orchestrator continues with items discovered so far

**Recovery**: Next run will retry from the saved cursor position.

#### Workflow Execution Failures

**Scenario**: Worker workflow fails or times out during execution

**Behavior**:
- Failure is logged in status update
- Remaining workflows continue executing
- Discovery and project updates proceed normally
- Status update includes failure context

**Example**:
```
## Workflow Execution

- ✅ vulnerability-scanner: Completed (15 issues created)
- ❌ dependency-updater: Failed after 30 minutes (timeout)
  - Error: Workflow run exceeded timeout limit
  - Action: Will retry on next orchestrator run

## Next Steps

1. Investigate dependency-updater timeout issue
2. Consider splitting into smaller scoped workflows
3. Continue processing discovered items
```

#### Project Update Failures

**Scenario**: Update fails for specific item (deleted issue, permissions error)

**Behavior**:
- Failure recorded for that item
- **Processing continues** for remaining items
- Failure reported in status update with details
- Item will be retried on next run

**Example failure handling**:
```javascript
// From project update instructions
// Invalid/deleted/inaccessible URL → Record failure and continue
try {
  await updateProject(item);
  successCount++;
} catch (error) {
  failureCount++;
  failures.push({
    url: item.url,
    error: error.message,
    reason: "Item deleted or inaccessible"
  });
  // Continue processing remaining items
}
```

#### Safe-Output Limit Violations

**Scenario**: Campaign attempts to exceed `max-project-updates-per-run: 10`

**Behavior**:
- Safe-output system enforces hard limit
- First 10 updates succeed
- Additional update attempts are blocked
- Agent receives error indicating limit reached
- Deferred items are processed on next run

**Status update reporting**:
```
## Project Updates

- **Processed**: 10 items (limit reached)
- **Deferred**: 15 items (will be processed next run)
- **Reason**: max-project-updates-per-run governance limit (10)
```

### Incident Recovery

Campaigns are designed for **automatic recovery**:

1. **Idempotent operations**: Safe to re-run without side effects
2. **Cursor-based pagination**: Resumes from last successful position
3. **Deterministic ordering**: Always processes items in consistent order
4. **Partial progress**: Work completed before failure is preserved

**No manual intervention required** - the next scheduled run will continue where the previous run left off.

## Governance Limits & Max Items Budget

### Purpose of Governance Limits

Governance policies prevent campaigns from:
- Overwhelming project boards with updates
- Exhausting API rate limits
- Creating too many issues/comments in a single run
- Running indefinitely due to unbounded work queues

### Default Limits

If not specified in the campaign spec, these defaults apply:

| Limit | Default Value | Description |
|-------|---------------|-------------|
| `max-discovery-items-per-run` | 100 | Maximum items to discover per run |
| `max-discovery-pages-per-run` | 10 | Maximum API pages to fetch per run |
| `max-new-items-per-run` | N/A | Maximum new items to add to project (no default, controlled by agent) |
| `max-project-updates-per-run` | 10 | Maximum project item updates per run |
| `max-comments-per-run` | 10 | Maximum comments to add per run |

### Configuring Limits

Limits are set in the campaign spec's `governance` section:

```yaml
governance:
  max-new-items-per-run: 20
  max-discovery-items-per-run: 200
  max-discovery-pages-per-run: 15
  max-project-updates-per-run: 15
  max-comments-per-run: 5
  opt-out-labels: ["no-campaign", "no-bot"]
  do-not-downgrade-done-items: true
```

### How Limits are Enforced

#### Discovery Limits

Enforced **during discovery precomputation**:

```javascript
let itemsScanned = 0;
let pagesScanned = 0;

while (hasMorePages && itemsScanned < maxItems && pagesScanned < maxPages) {
  const result = await octokit.rest.search.issuesAndPullRequests({
    q: searchQuery,
    per_page: 100,
    page: currentPage
  });
  
  itemsScanned += result.data.items.length;
  pagesScanned++;
  
  if (itemsScanned >= maxItems || pagesScanned >= maxPages) {
    core.warning(`Reached discovery budget limits. Stopping discovery.`);
    break;
  }
}
```

**Behavior when limit reached**:
- Discovery stops immediately
- Partial results are saved to manifest
- Cursor is saved for next run
- Orchestrator processes items discovered so far

#### Project Update Limits

Enforced **in AI agent planning phase**:

```javascript
// Agent planning logic
const selectedItems = deterministic_sort(discoveredItems)
  .slice(0, maxProjectUpdatesPerRun);

const deferredItems = discoveredItems.slice(maxProjectUpdatesPerRun);

// Report deferred items in status update
if (deferredItems.length > 0) {
  statusUpdate += `\n**Deferred**: ${deferredItems.length} items`;
  statusUpdate += `\n**Reason**: max-project-updates-per-run limit (${maxProjectUpdatesPerRun})`;
}
```

**Behavior when limit reached**:
- Only first N items are processed (deterministic order)
- Remaining items are deferred to next run
- Cursor advances to include processed items
- Status update reports deferred count

#### Safe-Output Limits

Enforced **by safe-output system**:

```yaml
safe-outputs:
  update-project:
    max: 10
    github-token: ${{ secrets.GH_AW_PROJECT_TOKEN }}
  add-comment:
    max: 5
  create-issue:
    max: 1
```

**Behavior when limit reached**:
- Safe-output blocks additional operations
- Agent receives error: `Maximum update-project operations (10) reached`
- Agent must stop processing and report in status update

### What Happens When Max Items is Reached

**Complete flow**:

1. **Discovery phase**: Discovers 100 items (discovery limit)
2. **Planning phase**: Selects 10 items (project update limit)
3. **Execution phase**: Processes 10 items successfully
4. **Status reporting**: Reports 90 items deferred
5. **Next run**: Continues from cursor, discovers next batch

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
- Cursor: Saved at item 50

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

Then:
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

```bash
git rm .github/workflows/security-q1-2025.campaign.md
git rm .github/workflows/security-q1-2025.campaign.lock.yml
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
on:
  schedule:
    - cron: "0 10 * * *"  # Runs daily at 10 AM
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
- name: Execute worker workflow
  uses: mcp__github__run_workflow
  with:
    workflow_id: "daily-dependency-check"
    ref: "main"
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
  # schedule:
  #   - cron: "0 10 * * *"  # DISABLED - controlled by campaign
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
|----------|----------|----------------|
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
3. **Phase 0** (optional): Executes worker workflows if configured
4. **Phase 1**: Reads manifest + project state
5. **Phase 2**: Plans updates for discovered items
6. **Phase 3**: Writes up to max updates
7. **Phase 4**: Reports progress, KPIs, deferred items

### Incident Handling

- **Discovery failure**: Partial results used, cursor not advanced, retry next run
- **Workflow failure**: Logged, other workflows continue, reported in status
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

## Further Reading

- [Campaign Specs](/gh-aw/guides/campaigns/specs/) - Configuration reference
- [Getting Started](/gh-aw/guides/campaigns/getting-started/) - Create your first campaign
- [Project Management](/gh-aw/guides/campaigns/project-management/) - Project board setup
- [Technical Overview](/gh-aw/guides/campaigns/technical-overview/) - Architecture details
