---
title: Technical Overview
description: Understand the architecture and implementation details of agentic campaigns
---

This guide explains how campaigns work under the hood—from discovery to orchestration to project updates. If you're looking to understand the technical implementation or troubleshoot issues, you're in the right place.

## Architecture overview

Campaigns consist of four main components that work together:

### 1. Campaign Specification (`.campaign.md`)

A YAML frontmatter file that serves as the campaign's "contract":

```yaml
---
id: framework-upgrade
name: Framework Upgrade
project-url: https://github.com/orgs/myorg/projects/42
tracker-label: campaign:framework-upgrade
objective: Upgrade all services to Framework vNext
kpis:
  - id: services_upgraded
    name: Services upgraded
    priority: primary
    target: 50
workflows:
  - framework-scanner
  - framework-upgrader
governance:
  max-project-updates-per-run: 10
---
```

This file is version-controlled and reviewed like code—it's the single source of truth for campaign configuration.

### 2. Discovery Precomputation

A JavaScript step that runs **before** the AI agent to find work items efficiently:

- Searches GitHub for issues/PRs created by worker workflows
- Uses tracker labels and tracker-ids for discovery
- Enforces budgets (max items, max pages) to control API usage
- Produces a deterministic JSON manifest at `./.gh-aw/campaign.discovery.json`
- Maintains pagination cursor in repo-memory for incremental discovery

**Why precomputation?** AI agents performing discovery during execution would be non-deterministic, expensive, and slow. Precomputation ensures stable, budget-controlled discovery.

### 3. Orchestrator Workflow (`.campaign.lock.yml`)

An auto-generated GitHub Actions workflow that coordinates the campaign:

- Compiled from the campaign spec by `gh aw compile`
- Runs on schedule (daily by default) or manual trigger
- Executes discovery precomputation step first
- Then runs AI agent to process discovered items
- Updates GitHub Project board via safe-outputs
- Creates status updates for stakeholder visibility

**Debug artifact:** A `.campaign.g.md` file is generated locally to help you review the orchestrator structure, but it's not committed to git—only the `.campaign.lock.yml` is version-controlled.

### 4. GitHub Project Board

The campaign dashboard with custom fields and views:

- **Custom fields**: Worker/Workflow, Priority, Status, Start Date, End Date, Effort
- **Views**: Campaign Roadmap (timeline), Task Tracker (table), Progress Board (kanban)
- **Automatic updates**: Orchestrator adds items and updates status as work progresses
- **Status reports**: Progress summaries appear in the Updates tab

Worker workflows remain campaign-agnostic—they don't know about campaigns. They just create issues/PRs with tracker labels, and the orchestrator discovers and tracks them.

## Orchestrator execution flow

When an orchestrator runs, it follows a strict sequence of phases:

### Phase 0: Discovery Precomputation (JavaScript)

**Runs first**, before the AI agent:

```javascript
// Pseudocode showing discovery logic
const discoveredItems = [];

// Search by tracker-id for each workflow
for (const workflow of campaign.workflows) {
  const items = await searchGitHub(`"tracker-id: ${workflow}" type:issue`);
  discoveredItems.push(...items);
}

// Search by tracker label (if configured)
if (campaign.trackerLabel) {
  const items = await searchGitHub(`label:"${campaign.trackerLabel}"`);
  discoveredItems.push(...items);
}

// Deduplicate and sort for stable ordering
const uniqueItems = deduplicateByUrl(discoveredItems);
uniqueItems.sort((a, b) => a.updated_at.localeCompare(b.updated_at));

// Enforce budgets
const limitedItems = uniqueItems.slice(0, maxDiscoveryItems);

// Write manifest for agent
writeManifest('./.gh-aw/campaign.discovery.json', {
  items: limitedItems,
  summary: {
    needs_add_count: limitedItems.filter(i => i.state === 'open').length,
    needs_update_count: limitedItems.filter(i => i.state === 'closed').length
  }
});
```

**Output:** Discovery manifest with normalized item metadata, summary counts, and cursor position.

### Phase 1: Agent Discovery (Read-Only)

The AI agent reads the precomputed manifest:

1. Load discovery manifest from `./.gh-aw/campaign.discovery.json`
2. Read current GitHub Project board state (all items + custom fields)
3. Parse discovered items from manifest
4. Check summary counts to determine if work is needed

**No GitHub API calls**—everything is read from the precomputed manifest.

### Phase 2: Agent Planning (Read-Only)

The agent plans which items to process:

1. Determine desired status from GitHub state:
   - Open issue/PR → `Todo` or `In Progress`
   - Closed issue → `Done`
   - Merged PR → `Done`
2. Calculate date fields (`start_date` from `created_at`, `end_date` from `closed_at`)
3. Apply write budget (`max-project-updates-per-run`)
4. Select items using deterministic order (oldest `updated_at` first)

**Critical**: No writes yet—all planning happens in memory.

### Phase 3: Agent Updates (Write-Only)

The agent updates the project board:

1. For each selected item, send `update-project` request to safe-output
2. **Do NOT** interleave reads and writes
3. **Do NOT** pre-check if item is on board (safe-output handles this)
4. Record per-item outcome (success/failure + error details)

**Safe-output enforcement:** The safe-output system enforces the `max-project-updates-per-run` limit. Once the limit is reached, additional requests are blocked.

### Phase 4: Status Reporting (Required)

The agent creates a project status update:

```yaml
create-project-status-update:
  project: https://github.com/orgs/myorg/projects/42
  status: ON_TRACK
  start_date: "2026-01-17"
  target_date: "2026-02-01"
  body: |
    ## Campaign Run Summary
    
    **Discovered:** 42 items
    **Processed:** 10 items added/updated
    **Deferred:** 32 items (will process next run)
    
    ### Progress
    - 60% complete (30/50 services upgraded)
    - On track for February completion
    
    ### Next Steps
    - Continue processing remaining 32 items
    - Monitor for new issues from scanner workflow
```

**Status reporting is required** for every orchestrator run to maintain visibility and stakeholder communication.

## Repo-memory for state persistence

Campaigns can use repo-memory (a git branch) to maintain durable state across runs:

### Cursor file

**Path:** `memory/campaigns/<campaign-id>/cursor.json`

**Purpose:** Checkpoint for incremental discovery—campaigns resume where they left off

**Format:**
```json
{
  "page": 3,
  "trackerId": "vulnerability-scanner"
}
```

**How it works:**
1. Discovery precomputation loads cursor at start of run
2. Continues from saved page number for each workflow
3. Updates cursor after processing each workflow's items
4. Saves updated cursor back to repo-memory
5. Next run picks up where previous run left off

### Metrics snapshots

**Path:** `memory/campaigns/<campaign-id>/metrics/<date>.json`

**Purpose:** Append-only progress tracking for retrospectives and trend analysis

**Format:**
```json
{
  "date": "2026-01-17",
  "campaign_id": "framework-upgrade",
  "kpis": {
    "services_upgraded": {
      "current": 30,
      "target": 50,
      "percentage": 60
    }
  },
  "items": {
    "total": 50,
    "open": 20,
    "closed": 25,
    "merged": 5
  },
  "velocity": {
    "items_per_day": 1.2,
    "estimated_completion": "2026-02-01"
  }
}
```

**Benefits:**
- Historical data for retrospectives
- Trend analysis across campaign lifecycle
- Evidence for decision-making
- Audit trail of campaign progress

> [!TIP]
> Repo-memory is optional but recommended. It enables powerful features like incremental discovery and historical metrics without requiring external databases.

## Campaign compilation

### How compilation works

When you run `gh aw compile`, the system:

1. **Scans for campaigns** in `.github/workflows/*.campaign.md`
2. **Validates the spec** using campaign validation rules
3. **Checks for meaningful details** (workflows, tracker-label, memory paths, etc.)
4. **Generates orchestrator** if spec has actionable configuration
5. **Renders to markdown** as `.campaign.g.md` (local debug artifact)
6. **Compiles to YAML** as `.campaign.lock.yml` (committed to git)

### Generated files

| File | Purpose | Version Control |
|------|---------|-----------------|
| `.campaign.md` | Campaign spec (source) | ✅ Committed |
| `.campaign.g.md` | Generated orchestrator (debug) | ❌ Local only (`.gitignore`) |
| `.campaign.lock.yml` | Compiled workflow (runs on GitHub) | ✅ Committed |

**Why `.g.md` is not committed:**
- It's a generated artifact, not source code
- Users edit `.campaign.md`, not `.campaign.g.md`
- Keeping it local aids debugging without cluttering git history
- Can always regenerate with `gh aw compile`

### Compilation triggers

Compilation happens automatically in these scenarios:

1. **Automated campaign creation** – During Phase 2 of campaign generation
2. **Manual compilation** – When you run `gh aw compile`
3. **After spec updates** – When you modify `.campaign.md` and need to regenerate the orchestrator

> [!IMPORTANT]
> Always run `gh aw compile` after editing `.campaign.md` to regenerate the `.campaign.lock.yml` file.

## Campaign item protection

Items with campaign labels are automatically protected from other workflows to prevent conflicts:

### Protection mechanism

When the orchestrator adds items to the project board, it applies the campaign label:

```yaml
# Orchestrator applies label automatically
update-project:
  project: https://github.com/orgs/myorg/projects/42
  item_url: https://github.com/myorg/repo/issues/123
  fields:
    status: In Progress
    campaign_id: framework-upgrade  # Implies campaign:framework-upgrade label
```

### How other workflows respect protection

Other workflows check for campaign labels and skip protected items:

```javascript
// Example from issue-monster workflow
if (issueLabels.some(label => label.startsWith('campaign:'))) {
  core.info(`Skipping #${issue.number}: managed by campaign orchestrator`);
  return false;  // Don't process this issue
}
```

### Additional opt-out labels

Campaigns also respect manual opt-out labels:

```yaml
governance:
  opt-out-labels: ["no-campaign", "no-bot"]
```

Items with these labels are excluded from campaign discovery and will not be added to the project board.

## Learn more

### Related documentation

- **[Campaign Specs](/gh-aw/guides/campaigns/specs/)** – Complete specification reference
- **[Campaign Flow](/gh-aw/guides/campaigns/flow/)** – Detailed lifecycle and incident handling
- **[Project Management](/gh-aw/guides/campaigns/project-management/)** – Project board configuration
- **[Safe Outputs](/gh-aw/reference/safe-outputs/)** – Safe-output system reference
