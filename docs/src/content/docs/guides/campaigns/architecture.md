---
title: Campaign Architecture
description: Detailed analysis of the orchestrator/worker flow for campaign lifecycle management.
sidebar:
  label: Architecture
banner:
  content: '<strong>Do not use.</strong> Campaigns are still incomplete and may produce unreliable or unintended results.'
---

This document provides a comprehensive analysis of the orchestrator/worker flow that powers the campaign lifecycle in GitHub Agentic Workflows.

## Executive Summary

GitHub Agentic Workflows (gh-aw) implements a sophisticated campaign orchestration system that coordinates multiple AI-powered workflows to accomplish large-scale, multi-repository objectives. The system uses an **orchestrator/worker pattern** where a central orchestrator manages campaign lifecycle, discovery, project board synchronization, and metrics tracking, while specialized worker workflows execute focused tasks.

**Key Architecture Principles:**
1. **Separation of Concerns**: Orchestrator handles coordination and state management; workers execute tasks
2. **Deterministic Discovery**: Pre-computation phase runs before agent, producing consistent manifests
3. **Incremental Processing**: Budget-based pagination with cursor persistence for gradual completion
4. **Idempotent Operations**: Both orchestrators and workers are safe to re-run without side effects
5. **Explicit Correlation**: Campaign items identified via standardized labels and tracker-ids
6. **Project-as-Source-of-Truth**: GitHub Projects board represents authoritative campaign state

## Campaign Lifecycle Overview

### Phase 0: Campaign Creation & Compilation

**User Actions:**
1. Create campaign spec file (`.github/workflows/<campaign-id>.campaign.md`)
2. Define campaign configuration in YAML frontmatter
3. Compile: `gh aw compile` generates orchestrator workflow

**Compiler Actions:**
- Scans for `.campaign.md` files in `.github/workflows/`
- Validates campaign spec structure
- Generates orchestrator markdown (`.campaign.g.md` - local debug artifact)
- Compiles to lock file (`.campaign.lock.yml` - committed)

**Key Files:**
```
.github/workflows/
├── security-q1-2025.campaign.md        # Campaign spec (source of truth)
├── security-q1-2025.campaign.g.md      # Generated orchestrator (local only)
└── security-q1-2025.campaign.lock.yml  # Compiled workflow (committed)
```

**Implementation:**
- `pkg/campaign/loader.go:LoadSpecs()` - Discovers campaign specs
- `pkg/cli/compile_workflow_processor.go:processCampaignSpec()` - Triggers compilation
- `pkg/campaign/orchestrator.go:BuildOrchestrator()` - Generates orchestrator

### Phase 1: Orchestrator Execution - Discovery Precomputation

**Trigger:** Schedule (default: daily at 6pm UTC) or manual workflow_dispatch

#### Discovery Script Execution

The discovery phase runs **before** the agent job to produce a deterministic manifest:

```yaml
steps:
  - name: Create workspace directory
    run: mkdir -p ./.gh-aw
  
  - name: Run campaign discovery precomputation
    uses: actions/github-script@v8.0.0
    env:
      GH_AW_CAMPAIGN_ID: security-q1-2025
      GH_AW_WORKFLOWS: "vulnerability-scanner,dependency-updater"
      GH_AW_TRACKER_LABEL: campaign:security-q1-2025
      GH_AW_PROJECT_URL: https://github.com/orgs/ORG/projects/1
      GH_AW_MAX_DISCOVERY_ITEMS: 200
      GH_AW_MAX_DISCOVERY_PAGES: 10
      GH_AW_CURSOR_PATH: /tmp/gh-aw/repo-memory/campaigns/security-q1-2025/cursor.json
    with:
      script: |
        const { setupGlobals } = require('/opt/gh-aw/actions/setup_globals.cjs');
        setupGlobals(core, github, context, exec, io);
        const { main } = require('/opt/gh-aw/actions/campaign_discovery.cjs');
        await main();
```

**Discovery Logic (`actions/setup/js/campaign_discovery.cjs`):**

1. **Load Cursor** (if exists) for continuation from previous run
2. **Search by Tracker-ID** for each workflow: `"gh-aw-tracker-id: workflow-name" type:issue`
3. **Search by Tracker Label** (if configured): `label:"campaign:security-q1-2025"`
4. **Normalize Items** to standard format with metadata
5. **Apply Pagination Budgets** to prevent unbounded API usage
6. **Generate Discovery Manifest** (`./.gh-aw/campaign.discovery.json`)
7. **Save Cursor** for next run

**Why Precomputation?**
- **Deterministic**: Same inputs → same manifest
- **Fast**: Parallel search possible, no AI latency
- **Budget-controlled**: Enforces API limits strictly
- **Cacheable**: Manifest can be reused/debugged

### Phase 2: Agent Job - State Synchronization

**Orchestrator reads the discovery manifest and synchronizes campaign state to the GitHub Project board.**

#### Step 2.1: Epic Issue Initialization (First Run Only)

**Epic Issue Requirements:**
- One Epic issue per campaign (parent for all work items)
- Labels: `agentic-campaign`, `z_campaign_<id>`, `epic`, `type:epic`
- Body contains: `campaign_id: <campaign-id>`

#### Step 2.2: Read State (Discovery) [NO WRITES]

**Agent Instructions:**
1. Read precomputed discovery manifest: `./.gh-aw/campaign.discovery.json`
2. Read current GitHub Project board state (items + fields)
3. Parse discovered items from manifest (pre-sorted by updated_at)
4. Check manifest summary for work counts (needs_add_count, needs_update_count)

**Key Principle:** Discovery is already done. Agent reads manifest, not performs searches.

#### Step 2.3: Make Decisions (Planning) [NO WRITES]

**Agent Instructions:**
1. **Determine Status** (from explicit GitHub state):
   - Open → `Todo` (or `In Progress` if indicated)
   - Closed (issue/discussion) → `Done`
   - Merged (PR) → `Done`

2. **Calculate Date Fields**:
   - `start_date`: Format `created_at` as `YYYY-MM-DD`
   - `end_date`:
     - Closed/merged → Format `closed_at`/`merged_at` as `YYYY-MM-DD`
     - Open → Today's date as `YYYY-MM-DD` (required for roadmap view)

3. **Apply Write Budget**:
   - Select at most `max-project-updates-per-run` items
   - Use deterministic order (oldest `updated_at` first, tie-break by number)
   - Defer remaining items to next run

#### Step 2.4: Write State (Execution) [WRITES ONLY]

**Agent Instructions:**
1. For each selected item, send `update-project` request
2. Do NOT interleave reads
3. Do NOT pre-check if item is on board
4. Follow Project Update Instructions for all writes

**Update Project Safe Output:**
```yaml
update-project:
  project: "https://github.com/orgs/ORG/projects/1"
  campaign_id: "security-q1-2025"
  content_type: "issue"
  content_number: 123
  fields:
    status: "Todo"
    campaign_id: "security-q1-2025"
    worker_workflow: "vulnerability-scanner"
    repository: "org/repo"
    priority: "High"
    size: "Medium"
    start_date: "2025-01-01"
    end_date: "2025-01-22"  # Today for open items
```

#### Step 2.5: Report & Status Update

**Required Status Update:**
```yaml
create-project-status-update:
  project: "https://github.com/orgs/ORG/projects/1"
  status: "ON_TRACK"  # ON_TRACK, AT_RISK, OFF_TRACK, COMPLETE
  start_date: "2026-01-06"
  target_date: "2026-01-31"
  body: |
    ## Campaign Run Summary
    
    **Discovered:** 25 items (15 issues, 10 PRs)
    **Processed:** 10 items added to project, 5 updated
    **Completion:** 60% (30/50 total tasks)
    
    ## Most Important Findings
    
    1. Critical accessibility gaps identified in mobile navigation
    2. Documentation coverage improved 5% this week (best velocity)
    3. Worker efficiency up 40% (daily-doc-updater)
    
    ## What Was Learned
    
    - Multi-device testing reveals issues desktop-only misses
    - Doc updates tied to code changes have higher accuracy
    
    ## KPI Trends
    
    **Documentation Coverage** (Primary KPI):
    - Baseline: 85% → Current: 88% → Target: 95%
    - Direction: ↑ +3% this week, +1% velocity/week
    - Status: ON TRACK - Will reach 95% in 7 weeks
    
    ## Next Steps
    
    1. Address 3 critical accessibility issues (high priority)
    2. Process remaining 15 discovered items
    3. Focus on accessibility improvements
```

### Phase 3: Worker Orchestration (Optional)

**Workers are specialized workflows that execute focused tasks. They are dispatch-only and follow a standardized contract.**

#### Worker Pattern Requirements

**1. Trigger Configuration:**
```yaml
on:
  workflow_dispatch:
    inputs:
      campaign_id:
        description: 'Campaign identifier'
        required: true
        type: string
      payload:
        description: 'JSON payload with work item details'
        required: true
        type: string
```

**Critical Rule:** Workers in campaign's `workflows` list MUST have ONLY `workflow_dispatch` trigger.

#### Worker Idempotency Pattern

**Deterministic Key Generation:**
```javascript
const campaignId = context.payload.inputs.campaign_id;
const payload = JSON.parse(context.payload.inputs.payload);

// Generate deterministic work key
const workKey = `campaign-${campaignId}-${payload.repository}-${payload.work_item_id}`;
const branchName = `fix/${workKey}`;
const prTitle = `[${workKey}] Fix: ${payload.alert_title}`;
```

**Idempotency Check:**
```javascript
// Search for existing PR with deterministic key
const existingPRs = await searchPullRequests({
  query: `repo:${repository} is:pr is:open "${workKey}" in:title`
});

if (existingPRs.length > 0) {
  core.info(`PR already exists: ${existingPRs[0].url}`);
  return;
}

// Proceed with fix and PR creation
```

## Data Flow Architecture

### Discovery Manifest Flow

```
┌─────────────────────────────────────────────────────────────┐
│ Discovery Precomputation (Phase 0)                          │
│ actions/setup/js/campaign_discovery.cjs                     │
└───────────────┬─────────────────────────────────────────────┘
                │
                │ Reads cursor
                ▼
┌───────────────────────────────────────────────────────────────┐
│ Repo-Memory (Cursor)                                          │
│ /tmp/gh-aw/repo-memory/campaigns/<id>/cursor.json            │
│ { "page": 3, "trackerId": "vulnerability-scanner" }          │
└───────────────┬───────────────────────────────────────────────┘
                │
                │ Continues from saved page
                ▼
┌───────────────────────────────────────────────────────────────┐
│ GitHub API (Issues & PRs Search)                              │
│ - Search by tracker-id: "gh-aw-tracker-id: worker-name"      │
│ - Search by tracker label: "campaign:security-q1-2025"       │
│ - Pagination with budgets (max items, max pages)             │
└───────────────┬───────────────────────────────────────────────┘
                │
                │ Normalized items
                ▼
┌───────────────────────────────────────────────────────────────┐
│ Discovery Manifest                                            │
│ ./.gh-aw/campaign.discovery.json                             │
│ - schema_version, campaign_id, generated_at                  │
│ - discovery: { total_items, cursor, budgets }                │
│ - summary: { needs_add_count, needs_update_count, ... }      │
│ - items: [ { url, content_type, number, repo, ... } ]        │
└───────────────┬───────────────────────────────────────────────┘
                │
                │ Saved cursor for next run
                ▼
┌───────────────────────────────────────────────────────────────┐
│ Repo-Memory (Updated Cursor)                                  │
│ /tmp/gh-aw/repo-memory/campaigns/<id>/cursor.json            │
│ { "page": 5, "trackerId": "dependency-updater" }             │
└───────────────────────────────────────────────────────────────┘
```

## Key Components

### Campaign Spec (`*.campaign.md`)

**Location:** `.github/workflows/<campaign-id>.campaign.md`

**Structure:**
```yaml
---
id: security-q1-2025
name: Security Q1 2025
version: v1
state: active  # planned, active, paused, completed, archived

# Objective and KPIs
objective: Resolve all critical security vulnerabilities across repositories
kpis:
  - name: Vulnerabilities Resolved
    priority: primary
    baseline: 0
    target: 100
    unit: "%"
    time-window-days: 90
    direction: increasing

# Project integration
project-url: https://github.com/orgs/ORG/projects/1
tracker-label: campaign:security-q1-2025

# Associated workflows
workflows:
  - vulnerability-scanner
  - dependency-updater

# Repo-memory configuration
memory-paths:
  - memory/campaigns/security-q1-2025/**
metrics-glob: memory/campaigns/security-q1-2025/metrics/*.json
cursor-glob: memory/campaigns/security-q1-2025/cursor.json

# Governance
governance:
  max-discovery-items-per-run: 200
  max-discovery-pages-per-run: 10
  max-project-updates-per-run: 10
  max-comments-per-run: 10
---

# Campaign Description

Detailed description of campaign objectives, background, and context.
```

### Discovery Manifest (`campaign.discovery.json`)

**Location:** `./.gh-aw/campaign.discovery.json`

**Purpose:** Precomputed discovery results for agent consumption

**Schema:**
```json
{
  "schema_version": "v1",
  "campaign_id": "security-q1-2025",
  "generated_at": "2025-01-22T12:00:00.000Z",
  "project_url": "https://github.com/orgs/ORG/projects/1",
  "discovery": {
    "total_items": 42,
    "items_scanned": 100,
    "pages_scanned": 2,
    "max_items_budget": 200,
    "max_pages_budget": 10,
    "cursor": {
      "page": 3,
      "trackerId": "vulnerability-scanner"
    }
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

### Repo-Memory

**Purpose:** Durable state persistence across workflow runs

#### Cursor File

**Location:** `/tmp/gh-aw/repo-memory/campaigns/<campaign-id>/cursor.json`

**Structure:**
```json
{
  "page": 3,
  "trackerId": "vulnerability-scanner"
}
```

**Usage:** Enables incremental discovery across runs without rescanning

#### Metrics Snapshots

**Location:** `/tmp/gh-aw/repo-memory/campaigns/<campaign-id>/metrics/<date>.json`

**Structure:**
```json
{
  "campaign_id": "security-q1-2025",
  "date": "2025-01-22",
  "tasks_total": 50,
  "tasks_completed": 30,
  "tasks_in_progress": 15,
  "tasks_blocked": 5,
  "velocity_per_day": 2.1,
  "estimated_completion": "2025-02-15"
}
```

**Usage:** Append-only history for trend analysis and retrospectives

## Orchestrator/Worker Interaction Patterns

### Pattern 1: Fire-and-Forget Dispatch

**Scenario:** Orchestrator dispatches workers but doesn't wait for completion

**Flow:**
```
Orchestrator Run N:
  ├─ Dispatch worker 1 (alert-123)
  ├─ Dispatch worker 2 (alert-456)
  └─ Complete run

Workers Execute Async:
  ├─ Worker 1 creates PR #789
  └─ Worker 2 creates PR #790

Orchestrator Run N+1:
  ├─ Discovery finds PR #789, PR #790 (via labels)
  ├─ Adds to project board
  └─ Status update reports progress
```

**Benefits:**
- No blocking on worker completion
- Orchestrator runs complete quickly
- Workers can take as long as needed

### Pattern 2: Idempotent Re-dispatch

**Scenario:** Orchestrator can safely re-dispatch same work item

**Flow:**
```
Orchestrator Run N:
  └─ Dispatch worker (alert-123)

Worker Execution:
  ├─ Generate work key: campaign-<id>-<repo>-alert-123
  ├─ Search for existing PR with work key
  ├─ Not found → Create PR #789
  └─ Complete

Orchestrator Run N+1:
  └─ Dispatch worker (alert-123) AGAIN

Worker Execution:
  ├─ Generate work key: campaign-<id>-<repo>-alert-123
  ├─ Search for existing PR with work key
  ├─ Found PR #789 → Skip or update
  └─ Complete
```

**Benefits:**
- Orchestrator doesn't track dispatched work
- Workers handle their own idempotency
- Safe to re-run on failures

### Pattern 3: Deterministic Processing Order

**Scenario:** Budget-limited processing with cursor continuation

**Flow:**
```
Run N:
  ├─ Discovery finds 100 items (budget limit)
  ├─ Sort by updated_at (deterministic)
  ├─ Process first 10 items (write budget)
  ├─ Save cursor at item 100
  └─ Defer 90 items

Run N+1:
  ├─ Discovery continues from cursor (items 101-200)
  ├─ Combine with deferred 90 items
  ├─ Sort by updated_at
  ├─ Process first 10 items
  ├─ Save cursor at item 200
  └─ Defer 170 items
```

**Benefits:**
- Fair processing (oldest items first)
- Predictable behavior
- No starvation
- Gradual completion

## Campaign Labeling Strategy

### Required Labels

**All campaign items MUST have TWO labels:**

1. **`agentic-campaign`** (Generic Campaign Label)
   - Marks content as part of ANY campaign
   - Prevents other workflows from processing campaign items
   - Enables campaign-wide queries

2. **`z_campaign_<campaign-id>`** (Campaign-Specific Label)
   - Enables precise discovery of items for THIS campaign
   - Format: `z_campaign_<id>` (lowercase, hyphen-separated)
   - Examples: `z_campaign_security-q1-2025`, `z_campaign_docs-quality`

### Protection Mechanism

**Non-Campaign Workflows:**
```yaml
on:
  issues:
    types: [opened, labeled]
    skip-if-match:
      query: "label:agentic-campaign"
      max: 0  # Skip if ANY campaign items match
```

**Example Filtering:**
```javascript
// In issue-monster workflow
if (issueLabels.some(label => label.startsWith('campaign:'))) {
  core.info(`Skipping #${issue.number}: has campaign label`);
  return false;
}
```

## Performance and Scalability

### Discovery Budget Management

**Problem:** Unbounded discovery can overwhelm GitHub API and cause rate limiting

**Solution:** Strict pagination budgets

**Configuration:**
```yaml
governance:
  max-discovery-items-per-run: 200  # Stop after 200 items
  max-discovery-pages-per-run: 10   # Stop after 10 API pages
```

**Enforcement:**
```javascript
if (itemsScanned >= maxItems || pagesScanned >= maxPages) {
  core.warning(`Reached discovery budget limits. Stopping discovery.`);
  break;
}
```

### Write Budget Management

**Problem:** Updating too many project board items in one run can be slow

**Solution:** Write budgets with deterministic ordering

**Configuration:**
```yaml
governance:
  max-project-updates-per-run: 10  # Update at most 10 items per run
```

**Deterministic Order:**
- Sort by `updated_at` ascending (oldest first)
- Tie-break by `number` ascending
- Ensures fair processing without starvation

## Best Practices

### Campaign Design

1. **Start Small**: Begin with a pilot campaign (1-2 repos, 1-2 workers)
2. **Define Clear KPIs**: Measurable, time-bound, with baseline/target
3. **Use Conservative Budgets**: Start with low limits, increase as needed
4. **Test Workers Independently**: Verify worker behavior before campaign use
5. **Document Objectives**: Clear objective statement in campaign spec

### Worker Development

1. **Single Purpose**: Each worker does ONE thing well
2. **Idempotency**: Implement deterministic keys and duplicate checks
3. **Dispatch-Only**: Remove all other triggers
4. **Standard Contract**: Always accept campaign_id and payload
5. **Proper Labels**: Apply campaign labels to all created items

### Orchestration

1. **Fire-and-Forget**: Don't wait for worker completion
2. **Incremental Discovery**: Use budgets to prevent overwhelming API
3. **Deterministic Order**: Process oldest items first
4. **Regular Status Updates**: Report progress, findings, next steps
5. **Metrics Tracking**: Persist snapshots for trend analysis

## Common Patterns and Anti-Patterns

### ✅ Good Patterns

**1. Cursor-Based Pagination**
```javascript
// Load cursor, continue from saved position
const cursor = loadCursor(cursorPath);
const startPage = cursor?.page || 1;

// Save cursor after discovery
saveCursor(cursorPath, { page: nextPage, trackerId });
```

**2. Deterministic Work Keys**
```javascript
const workKey = `campaign-${campaignId}-${repo}-${workItemId}`;
const branchName = `fix/${workKey}`;
const prTitle = `[${workKey}] Fix: ${title}`;
```

**3. Budget-Limited Processing**
```javascript
if (itemsProcessed >= maxItems) {
  core.warning(`Reached budget limit. Deferring remaining items.`);
  break;
}
```

### ❌ Anti-Patterns

**1. Unbounded Discovery**
```javascript
// ❌ BAD - No budget limits
while (hasMorePages) {
  const result = await searchIssues(page++);
  items.push(...result.items);
}
```

**2. Non-Deterministic Processing**
```javascript
// ❌ BAD - Random order
items.sort(() => Math.random() - 0.5);
```

**3. Mixed Worker Triggers**
```yaml
# ❌ BAD - Multiple triggers create duplicates
on:
  schedule:
    - cron: "0 9 * * *"
  workflow_dispatch:
```

**4. Missing Idempotency**
```javascript
// ❌ BAD - Always creates new PR
await createPullRequest({ title, body });

// ✅ GOOD - Check first
const existing = await searchPR(workKey);
if (!existing) {
  await createPullRequest({ title, body });
}
```

## Related Documentation

- [Campaign Flow & Lifecycle](/gh-aw/guides/campaigns/flow/) - What happens on each orchestrator run
- [Campaign Specs](/gh-aw/guides/campaigns/specs/) - Configuration reference
- [Getting Started](/gh-aw/guides/campaigns/getting-started/) - Create your first campaign
- [CLI Commands](/gh-aw/guides/campaigns/cli-commands/) - Inspect and validate campaigns
