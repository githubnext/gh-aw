# Campaign Orchestrator/Worker Flow Analysis

## Executive Summary

GitHub Agentic Workflows (gh-aw) implements a sophisticated campaign orchestration system that coordinates multiple AI-powered workflows to accomplish large-scale, multi-repository objectives. The system uses a **orchestrator/worker pattern** where a central orchestrator manages campaign lifecycle, discovery, project board synchronization, and metrics tracking, while specialized worker workflows execute focused tasks.

**Key Architecture Principles:**
1. **Separation of Concerns**: Orchestrator handles coordination and state management; workers execute tasks
2. **Deterministic Discovery**: Pre-computation phase runs before agent, producing consistent manifests
3. **Incremental Processing**: Budget-based pagination with cursor persistence for gradual completion
4. **Idempotent Operations**: Both orchestrators and workers are safe to re-run without side effects
5. **Explicit Correlation**: Campaign items identified via standardized labels and tracker-ids
6. **Project-as-Source-of-Truth**: GitHub Projects board represents authoritative campaign state

---

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

---

### Phase 1: Orchestrator Execution - Discovery Precomputation

**Trigger:** Schedule (default: daily at 6pm UTC) or manual workflow_dispatch

**Discovery Steps:**

#### Step 1.1: Discovery Script Execution
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

1. **Load Cursor** (if exists):
   ```javascript
   // Loads from /tmp/gh-aw/repo-memory/campaigns/<id>/cursor.json
   const cursor = loadCursor(cursorPath);
   // cursor = { page: 3, trackerId: "vulnerability-scanner" }
   ```

2. **Search by Tracker-ID** (for each workflow):
   ```javascript
   // Searches for: "gh-aw-tracker-id: vulnerability-scanner" type:issue
   const searchQuery = `"gh-aw-tracker-id: ${trackerId}" type:issue`;
   const response = await octokit.rest.search.issuesAndPullRequests({
     q: searchQuery,
     per_page: 100,
     page: cursor?.page || 1,
     sort: "updated",
     order: "asc",  // Stable ordering for determinism
   });
   ```

3. **Search by Tracker Label** (if configured):
   ```javascript
   // Searches for: label:"campaign:security-q1-2025"
   const searchQuery = `label:"${trackerLabel}"`;
   ```

4. **Normalize Items**:
   ```javascript
   function normalizeItem(item, contentType) {
     return {
       url: item.html_url,
       content_type: contentType,  // "issue" or "pull_request"
       number: item.number,
       repo: item.repository.full_name,
       created_at: item.created_at,
       updated_at: item.updated_at,
       state: item.state,
       title: item.title,
       closed_at: item.closed_at,
       merged_at: item.merged_at
     };
   }
   ```

5. **Apply Pagination Budgets**:
   ```javascript
   if (itemsScanned >= maxItems || pagesScanned >= maxPages) {
     core.warning(`Reached discovery budget limits. Stopping discovery.`);
     break;
   }
   ```

6. **Generate Discovery Manifest** (`./.gh-aw/campaign.discovery.json`):
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

7. **Save Cursor** (for next run):
   ```javascript
   saveCursor(cursorPath, {
     page: nextPage,
     trackerId: currentTrackerId
   });
   ```

**Why Precomputation?**
- **Deterministic**: Same inputs → same manifest
- **Fast**: Parallel search possible, no AI latency
- **Budget-controlled**: Enforces API limits strictly
- **Cacheable**: Manifest can be reused/debugged

**Implementation:**
- `pkg/campaign/orchestrator.go:buildDiscoverySteps()` - Generates discovery steps
- `actions/setup/js/campaign_discovery.cjs:main()` - Discovery script

---

### Phase 2: Agent Job - State Synchronization

**Orchestrator reads the discovery manifest and synchronizes campaign state to the GitHub Project board.**

#### Step 2.1: Epic Issue Initialization (First Run Only)

**Epic Issue Requirements:**
- One Epic issue per campaign (parent for all work items)
- Labels: `agentic-campaign`, `z_campaign_<id>`, `epic`, `type:epic`
- Body contains: `campaign_id: <campaign-id>`

**Epic Creation:**
```yaml
create-issue:
  title: "Security Q1 2025"
  body: |
    ## Campaign Overview
    
    **Objective**: Resolve all critical security vulnerabilities
    
    **Campaign Details:**
    - Campaign ID: `security-q1-2025`
    - Project Board: https://github.com/orgs/ORG/projects/1
    - Worker Workflows: `vulnerability-scanner`, `dependency-updater`
    
    ---
    `campaign_id: security-q1-2025`
  labels:
    - agentic-campaign
    - z_campaign_security-q1-2025
    - epic
    - type:epic
```

**Add Epic to Project:**
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
    repository: "owner/repo"
    priority: "High"
    size: "Large"
    start_date: "<YYYY-MM-DD>"
    end_date: "<YYYY-MM-DD>"
```

#### Step 2.2: Read State (Discovery) [NO WRITES]

**Agent Instructions (.github/aw/orchestrate-agentic-campaign.md):**
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

**Governance Example:**
```yaml
governance:
  max-discovery-items-per-run: 200    # Discovery budget
  max-discovery-pages-per-run: 10      # API pages budget
  max-project-updates-per-run: 10      # Write budget
  max-comments-per-run: 10             # Comment budget
  opt-out-labels: ["no-campaign", "no-bot"]
```

**Processing Flow:**
```
Run 1:
- Discover: 50 items (budget reached)
- Process: 10 items (write budget reached)
- Defer: 40 items
- Cursor: Saved at item 50

Run 2:
- Discover: 50 more items (starting from cursor)
- Process: 10 items (from deferred 40 + newly discovered)
- Defer: 30 + 50 = 80 items
- Cursor: Saved at item 100

Run 3-N:
- Continue until all items processed
```

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

**Safe Output Handling:**
- Orchestrator calls `update-project` for each item
- Safe output system handles deduplication
- Idempotency: First add → full fields; existing item → status-only update

**Implementation:**
- `.github/aw/orchestrate-agentic-campaign.md` - Agent instructions
- `.github/aw/update-agentic-campaign-project.md` - Project update instructions

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
    
    **Accessibility Score** (Supporting KPI):
    - Baseline: 90% → Current: 91% → Target: 98%
    - Direction: ↑ +1% this month
    - Status: AT RISK - Slower than expected
    
    ## Next Steps
    
    1. Address 3 critical accessibility issues (high priority)
    2. Process remaining 15 discovered items
    3. Focus on accessibility improvements
```

**Required Sections:**
- Most Important Findings (2-3 critical discoveries/blockers)
- What Was Learned (insights, patterns)
- KPI Trends (baseline → current → target with velocity)
- Campaign Summary (tasks completed, in progress, blocked)
- Next Steps (clear action items)

**Metrics Snapshot** (saved to repo-memory):
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

**File System Path:**
```
/tmp/gh-aw/repo-memory/campaigns/security-q1-2025/metrics/2025-01-22.json
```

**Implementation:**
- `.github/aw/orchestrate-agentic-campaign.md` - Orchestration instructions
- Safe outputs: `create-project-status-update`, `update-project`

---

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
- No schedule, push, or pull_request triggers
- Campaign orchestrator controls execution timing
- Prevents duplicate executions

#### Worker Input Contract

**Standard Inputs:**
- `campaign_id` (string): Campaign identifier
- `payload` (string): JSON-encoded work item data

**Payload Structure Example:**
```json
{
  "repository": "owner/repo",
  "work_item_id": "alert-123",
  "target_ref": "main",
  "alert_type": "sql-injection",
  "file_path": "src/db.go",
  "line_number": 42
}
```

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
  // Optionally update with new information
  return;
}

// Proceed with fix and PR creation
```

**Required Labels:**
- `campaign:<campaign-id>` - Enables discovery by orchestrator
- Tracker label from campaign spec (if configured)

#### Worker Dispatch by Orchestrator

**Dispatch Example:**
```javascript
// Read discovery manifest
const manifest = JSON.parse(fs.readFileSync('./.gh-aw/campaign.discovery.json', 'utf8'));

// For each work item needing processing
for (const item of manifest.items) {
  // Construct payload
  const payload = {
    repository: item.repo,
    work_item_id: `alert-${item.number}`,
    target_ref: "main",
    alert_type: "sql-injection",
    file_path: "src/db.go",
    line_number: 42
  };
  
  // Dispatch worker
  await github.rest.actions.createWorkflowDispatch({
    owner: context.repo.owner,
    repo: context.repo.repo,
    workflow_id: "security-fix-worker.yml",
    ref: "main",
    inputs: {
      campaign_id: "security-q1-2025",
      payload: JSON.stringify(payload)
    }
  });
}
```

**Dispatch Strategy:**
- Fire-and-forget: Orchestrator doesn't wait for completion
- Workers create PRs/issues with campaign labels
- Next run discovers worker outputs via labels/tracker-ids
- Failure handling: Log failure, continue with other workers

#### Worker Discovery

**How Orchestrator Finds Worker Outputs:**
1. **Tracker Label**: Items labeled with `campaign:${campaign_id}`
2. **Tracker ID**: Items with `gh-aw-tracker-id: worker-name` in description
3. **Discovery Script**: Searches for both via GitHub API

**Worker Responsibilities:**
- Apply campaign tracker label to all created items
- Include worker's tracker-id in issue/PR descriptions (optional)
- Implement idempotency via deterministic keys

**Implementation:**
- `.github/aw/execute-agentic-campaign-workflow.md` - Worker orchestration instructions
- `actions/setup/js/campaign_discovery.cjs` - Discovery script

---

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

### Agent Job Flow

```
┌───────────────────────────────────────────────────────────────┐
│ Agent Job (Phase 1)                                           │
│ Orchestrator reads manifest and project state                │
└───────────────┬───────────────────────────────────────────────┘
                │
                │ Reads manifest
                ▼
┌───────────────────────────────────────────────────────────────┐
│ Discovery Manifest                                            │
│ ./.gh-aw/campaign.discovery.json                             │
└───────────────┬───────────────────────────────────────────────┘
                │
                │ Reads project state
                ▼
┌───────────────────────────────────────────────────────────────┐
│ GitHub Project Board (via GitHub MCP)                         │
│ - Current items on board                                      │
│ - Field values (status, campaign_id, worker_workflow, ...)   │
└───────────────┬───────────────────────────────────────────────┘
                │
                │ Plans updates (deterministic order, budgets)
                ▼
┌───────────────────────────────────────────────────────────────┐
│ Update Decisions                                              │
│ - Items to add (with full fields)                            │
│ - Items to update (status-only)                              │
│ - Items deferred (exceeded budget)                           │
└───────────────┬───────────────────────────────────────────────┘
                │
                │ Executes via safe outputs
                ▼
┌───────────────────────────────────────────────────────────────┐
│ Safe Outputs (update-project)                                 │
│ - Adds items to project board                                │
│ - Updates status fields                                       │
│ - Handles deduplication                                       │
└───────────────┬───────────────────────────────────────────────┘
                │
                │ Updated board state
                ▼
┌───────────────────────────────────────────────────────────────┐
│ GitHub Project Board (Updated)                                │
│ - New items added with campaign fields                        │
│ - Existing items updated with new status                      │
└───────────────┬───────────────────────────────────────────────┘
                │
                │ Creates status update
                ▼
┌───────────────────────────────────────────────────────────────┐
│ Safe Outputs (create-project-status-update)                   │
│ - Progress summary                                            │
│ - KPI trends                                                  │
│ - Most important findings                                     │
│ - Next steps                                                  │
└───────────────┬───────────────────────────────────────────────┘
                │
                │ Saves metrics
                ▼
┌───────────────────────────────────────────────────────────────┐
│ Repo-Memory (Metrics Snapshot)                                │
│ /tmp/gh-aw/repo-memory/campaigns/<id>/metrics/<date>.json    │
│ { campaign_id, date, tasks_total, tasks_completed, ... }     │
└───────────────────────────────────────────────────────────────┘
```

### Worker Orchestration Flow

```
┌───────────────────────────────────────────────────────────────┐
│ Orchestrator (Optional Worker Dispatch)                       │
└───────────────┬───────────────────────────────────────────────┘
                │
                │ Reads manifest for work items
                ▼
┌───────────────────────────────────────────────────────────────┐
│ Discovery Manifest                                            │
│ ./.gh-aw/campaign.discovery.json                             │
└───────────────┬───────────────────────────────────────────────┘
                │
                │ For each work item
                ▼
┌───────────────────────────────────────────────────────────────┐
│ Workflow Dispatch (GitHub API)                                │
│ - workflow_id: "security-fix-worker.yml"                      │
│ - inputs:                                                     │
│     campaign_id: "security-q1-2025"                           │
│     payload: JSON.stringify({ repository, work_item_id, ... })│
└───────────────┬───────────────────────────────────────────────┘
                │
                │ Fire-and-forget dispatch
                ▼
┌───────────────────────────────────────────────────────────────┐
│ Worker Workflow Execution                                     │
│ 1. Parse campaign_id and payload                              │
│ 2. Generate deterministic work key                            │
│ 3. Check for existing PR/issue                                │
│ 4. If not exists: Create PR/issue with campaign labels        │
│ 5. If exists: Update or skip                                  │
└───────────────┬───────────────────────────────────────────────┘
                │
                │ Creates PR/issue with labels
                ▼
┌───────────────────────────────────────────────────────────────┐
│ GitHub Repository (PR/Issue Created)                          │
│ - Labels: campaign:<id>, agentic-campaign                     │
│ - Title includes deterministic work key                       │
│ - Description includes tracker-id                             │
└───────────────┬───────────────────────────────────────────────┘
                │
                │ Next run discovers via labels/tracker-id
                ▼
┌───────────────────────────────────────────────────────────────┐
│ Discovery (Next Run)                                          │
│ - Searches for campaign labels                                │
│ - Finds worker-created items                                  │
│ - Adds to discovery manifest                                  │
│ - Orchestrator synchronizes to project board                  │
└───────────────────────────────────────────────────────────────┘
```

---

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
  - name: Mean Time to Resolution
    priority: supporting
    baseline: 14
    target: 5
    unit: "days"
    time-window-days: 90
    direction: decreasing

# Project integration
project-url: https://github.com/orgs/ORG/projects/1
tracker-label: campaign:security-q1-2025

# Associated workflows
workflows:
  - vulnerability-scanner
  - dependency-updater

# Discovery scope
discovery-repos:
  - org/repo1
  - org/repo2
discovery-orgs:
  - org-name

# Repo-memory configuration
memory-paths:
  - memory/campaigns/security-q1-2025/**
metrics-glob: memory/campaigns/security-q1-2025/metrics/*.json
cursor-glob: memory/campaigns/security-q1-2025/cursor.json

# Governance
governance:
  max-new-items-per-run: 25
  max-discovery-items-per-run: 200
  max-discovery-pages-per-run: 10
  opt-out-labels: [no-campaign, no-bot]
  max-project-updates-per-run: 10
  max-comments-per-run: 10
---

# Campaign Description

Detailed description of campaign objectives, background, and context.
```

**Implementation:**
- `pkg/campaign/spec.go:CampaignSpec` - Data structure
- `pkg/campaign/validation.go:ValidateSpec()` - Validation

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
      "title": "Upgrade dependency X",
      "closed_at": null,
      "merged_at": null
    }
  ]
}
```

**Implementation:**
- `actions/setup/js/campaign_discovery.cjs` - Generation script

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

**Implementation:**
- `pkg/workflow/repo_memory.go` - Repo-memory tool configuration

---

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

Run N+2:
  └─ Continue...
```

**Benefits:**
- Fair processing (oldest items first)
- Predictable behavior
- No starvation
- Gradual completion

### Pattern 4: Label-Based Discovery

**Scenario:** Workers create items with campaign labels for discovery

**Flow:**
```
Worker Creates PR:
  ├─ PR #789 in org/repo
  ├─ Title: [campaign-security-q1-2025-org-repo-alert-123] Fix SQL injection
  ├─ Labels: campaign:security-q1-2025, agentic-campaign, security
  └─ Description includes: gh-aw-tracker-id: vulnerability-scanner

Discovery (Next Run):
  ├─ Search: label:"campaign:security-q1-2025"
  ├─ Search: "gh-aw-tracker-id: vulnerability-scanner"
  ├─ Finds PR #789
  ├─ Normalizes to manifest format
  └─ Adds to ./.gh-aw/campaign.discovery.json

Orchestrator:
  ├─ Reads manifest
  ├─ Calls update-project for PR #789
  └─ PR appears on project board
```

**Benefits:**
- Explicit correlation via labels
- Workers independent of campaign internals
- Discovery is straightforward GitHub search

---

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

### Label Application

**Workers:**
```yaml
create-pull-request:
  title: "[campaign-security-q1-2025-org-repo-alert-123] Fix SQL injection"
  labels:
    - agentic-campaign
    - z_campaign_security-q1-2025
    - security
```

**Epic Issue:**
```yaml
create-issue:
  title: "Security Q1 2025"
  labels:
    - agentic-campaign
    - z_campaign_security-q1-2025
    - epic
    - type:epic
```

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

**Benefits:**
- Clear ownership boundaries
- Prevents conflicts between workflows
- Enables campaign isolation

---

## Campaign States and Lifecycle

### State Transitions

```
planned → active → paused → completed → archived
   ↓         ↓        ↓          ↓
   └─────────┴────────┴──────────┘
           (Manual actions)
```

### State Definitions

| State | Meaning | Workflow Execution |
|-------|---------|-------------------|
| `planned` | Drafting and review; not intended to run yet | Manual trigger only |
| `active` | Running on schedule | Automatic + manual |
| `paused` | Temporarily stopped | Must disable workflow |
| `completed` | Finished and no longer running | Must disable workflow |
| `archived` | Kept for reference only | Delete `.lock.yml` |

### Pausing a Campaign

**Steps:**
1. Set `state: paused` in campaign spec (for clarity)
2. Compile: `gh aw compile`
3. Disable workflow in GitHub Actions UI

**Important:** State change alone doesn't stop execution. Must disable workflow.

### Completing a Campaign

**Steps:**
1. Run orchestrator one final time (generates completion status)
2. Set `state: completed` in campaign spec
3. Compile: `gh aw compile`
4. Disable workflow in GitHub Actions UI

**Final Status Update:**
```yaml
create-project-status-update:
  project: "https://github.com/orgs/ORG/projects/1"
  status: "COMPLETE"
  body: |
    ## Campaign Complete
    
    The Security Q1 2025 campaign has successfully completed all objectives.
    
    ## Final Metrics
    - Total tasks: 200/200 (100%)
    - Duration: 90 days
    - Average velocity: 7.5 tasks/day
    
    ## KPI Achievement
    - Vulnerabilities Resolved: 100% (TARGET ACHIEVED)
    - Mean Time to Resolution: 3 days (TARGET EXCEEDED)
```

### Archiving a Campaign

**Steps:**
1. Set `state: archived` in campaign spec
2. Keep `.campaign.md` for historical reference
3. Delete `.campaign.lock.yml` to prevent execution

**Repo-Memory Preservation:**
- Cursor file remains at final position
- Metrics snapshots preserved for analysis
- Valuable for retrospectives and reporting

---

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

**Cursor Persistence:**
```json
{
  "page": 10,
  "trackerId": "vulnerability-scanner"
}
```

**Next run continues from saved cursor without rescanning.**

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

**Deferred Items:**
- Items beyond write budget are deferred to next run
- Cursor tracks progress
- Gradual completion over multiple runs

### Rate Limiting Handling

**HTTP 429 Response:**
- Orchestrator backs off and ends run
- Reports remaining work in status update
- Next scheduled run continues processing

**No Aggressive Retries:**
- Prevents further rate limit violations
- GitHub Actions schedule handles retry timing

---

## Security and Governance

### Campaign Item Protection

**Problem:** Non-campaign workflows may interfere with campaign-managed items

**Solution:** Campaign label filtering

**Protection Labels:**
- `agentic-campaign` - Generic campaign marker
- `z_campaign_<id>` - Campaign-specific marker
- `no-bot`, `no-campaign` - Additional opt-out

**Workflow Filtering:**
```yaml
on:
  issues:
    skip-if-match:
      query: "label:agentic-campaign"
```

### Worker Trigger Isolation

**Problem:** Workers with multiple triggers can execute outside campaign control

**Solution:** Dispatch-only workers

**Required Configuration:**
```yaml
on:
  workflow_dispatch:  # ONLY this trigger
    inputs:
      campaign_id:
        required: true
      payload:
        required: true
```

**Incorrect Configuration:**
```yaml
on:
  schedule:           # ❌ NO - Creates duplicate executions
    - cron: "0 9 * * *"
  workflow_dispatch:  # ✓ OK
  push:               # ❌ NO - Conflicts with campaign timing
```

### Governance Policies

**Opt-Out Labels:**
```yaml
governance:
  opt-out-labels: ["no-campaign", "no-bot", "wontfix"]
```

**Effect:** Items with opt-out labels are excluded from discovery

**Budget Limits:**
```yaml
governance:
  max-discovery-items-per-run: 200
  max-discovery-pages-per-run: 10
  max-project-updates-per-run: 10
  max-comments-per-run: 10
```

**Effect:** Prevents unbounded API usage and runaway executions

---

## Error Handling and Resilience

### Discovery Failures

**Scenario:** Discovery script fails (API error, timeout)

**Handling:**
- Partial results used if available
- Cursor NOT advanced (prevents skipping items)
- Error logged in workflow output
- Next run retries from same cursor position

### Worker Dispatch Failures

**Scenario:** Workflow dispatch API call fails

**Handling:**
- Failure logged with context
- Other workers continue dispatching
- Status update reports failed dispatches
- Next run can retry dispatch (idempotency handles duplicates)

### Project Update Failures

**Scenario:** Individual item fails to add to project board

**Handling:**
- Failure recorded for that item
- Processing continues with other items
- Status update reports failures with reasons
- Next run can retry (safe outputs handle deduplication)

### Budget Limit Reached

**Scenario:** Discovery or write budget exhausted mid-run

**Handling:**
- Processing stops gracefully
- Cursor saved at current position
- Status update explains deferred items
- Next run continues from cursor

**Key Principle:** Campaigns never stop due to limits—they process incrementally.

---

## Testing and Validation

### Campaign Spec Validation

**Command:**
```bash
gh aw campaign validate
gh aw campaign validate my-campaign
gh aw campaign validate --json
```

**Checks:**
- Required fields present (id, name, version)
- Valid state value
- Valid KPI configurations
- Valid governance policies
- Valid URL formats

**Implementation:**
- `pkg/campaign/validation.go:ValidateSpec()`

### Worker Testing

**Required Steps:**
1. **Prepare test payload**:
   ```json
   {
     "repository": "test-org/test-repo",
     "work_item_id": "test-1",
     "target_ref": "main"
   }
   ```

2. **Trigger test run**:
   ```bash
   gh workflow run security-fix-worker.yml \
     -f campaign_id=test-campaign \
     -f payload='{"repository":"test-org/test-repo","work_item_id":"test-1"}'
   ```

3. **Verify success**:
   - Workflow succeeded
   - Idempotency: Re-run with same inputs skips/updates
   - Created items have correct labels
   - Deterministic keys used in titles/branches

4. **Test failure actions**:
   - DO NOT use worker if testing fails
   - Analyze logs, make corrections
   - Recompile and retest

### Discovery Testing

**Manual Discovery Test:**
```bash
# Set environment variables
export GH_AW_CAMPAIGN_ID=test-campaign
export GH_AW_WORKFLOWS=worker1,worker2
export GH_AW_MAX_DISCOVERY_ITEMS=10
export GH_AW_MAX_DISCOVERY_PAGES=2

# Run discovery script
node actions/setup/js/campaign_discovery.cjs

# Check manifest
cat ./.gh-aw/campaign.discovery.json
```

---

## Debugging and Observability

### Enable Debug Logging

**Go Code:**
```bash
DEBUG=campaign:*,cli:* gh aw compile
```

**Workflow Execution:**
- View Actions run logs in GitHub UI
- Check discovery step output
- Review agent job output

### Inspect Generated Orchestrator

**Local Debug Artifact:**
```bash
cat .github/workflows/<campaign-id>.campaign.g.md
```

**Compiled Workflow:**
```bash
cat .github/workflows/<campaign-id>.campaign.lock.yml
```

### Inspect Discovery Manifest

**During Workflow Run:**
1. Download workflow artifacts
2. Check `./.gh-aw/campaign.discovery.json`

**Manifest Analysis:**
```json
{
  "discovery": {
    "total_items": 42,
    "items_scanned": 100,
    "cursor": { "page": 3, "trackerId": "..." }
  },
  "summary": {
    "needs_add_count": 25,
    "needs_update_count": 17
  }
}
```

### Validate Cursor State

**Check Repo-Memory:**
```bash
cat memory/campaigns/<campaign-id>/cursor.json
```

### Check Metrics History

**List Snapshots:**
```bash
ls -l memory/campaigns/<campaign-id>/metrics/
```

**View Snapshot:**
```bash
cat memory/campaigns/<campaign-id>/metrics/2025-01-22.json
```

---

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
6. **Test Thoroughly**: Test with sample inputs before production use

### Orchestration

1. **Fire-and-Forget**: Don't wait for worker completion
2. **Incremental Discovery**: Use budgets to prevent overwhelming API
3. **Deterministic Order**: Process oldest items first
4. **Regular Status Updates**: Report progress, findings, next steps
5. **Metrics Tracking**: Persist snapshots for trend analysis

### Project Board Management

1. **Dedicated Project**: One project per campaign (or per campaign group)
2. **Required Fields**: Ensure campaign_id, worker_workflow, status fields exist
3. **Governance Limits**: Set appropriate write budgets
4. **Status Updates**: Run orchestrator regularly for fresh status

### Monitoring

1. **Check Discovery Counts**: Verify items are being discovered
2. **Monitor Budgets**: Ensure budgets are appropriate for campaign scale
3. **Review Status Updates**: Read project status updates for progress
4. **Track Failures**: Investigate worker/dispatch failures promptly
5. **Analyze Metrics**: Review velocity and completion trends

---

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

**4. Status-Only Updates for Existing Items**
```yaml
update-project:
  fields:
    status: "Done"  # Only update status if item already on board
```

**5. Comprehensive Status Updates**
```yaml
body: |
  ## Most Important Findings
  ## What Was Learned
  ## KPI Trends
  ## Next Steps
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

**4. Agent-Side Discovery**
```
# ❌ BAD - Agent performs GitHub-wide search
# Agent should read precomputed manifest instead
```

**5. Missing Idempotency**
```javascript
// ❌ BAD - Always creates new PR
await createPullRequest({ title, body });

// ✅ GOOD - Check first
const existing = await searchPR(workKey);
if (!existing) {
  await createPullRequest({ title, body });
}
```

**6. Incomplete Labels**
```yaml
# ❌ BAD - Missing campaign-specific label
labels:
  - agentic-campaign

# ✅ GOOD - Both labels
labels:
  - agentic-campaign
  - z_campaign_security-q1-2025
```

---

## Future Enhancements

### Planned Improvements

1. **Multi-Repository Discovery**: Search across organization repos automatically
2. **Advanced Filtering**: Filter items by milestone, assignee, custom fields
3. **Discovery Caching**: Cache discovery results to reduce API calls
4. **Incremental Updates**: Only update changed items on project board
5. **Workflow Templates**: Pre-built campaign templates for common scenarios

### Extension Points

1. **Custom Discovery Scripts**: Allow campaigns to provide custom discovery logic
2. **Discovery Plugins**: Plugin system for non-GitHub sources (Jira, Linear)
3. **Campaign Hierarchies**: Parent/child campaigns with rollup metrics
4. **Cross-Campaign Dependencies**: Express dependencies between campaigns
5. **Real-Time Notifications**: Slack/email notifications for campaign events

---

## References

### Documentation
- [Campaign Flow & Lifecycle](/docs/src/content/docs/guides/campaigns/flow.md)
- [Campaign Specs](/docs/src/content/docs/guides/campaigns/specs.md)
- [Getting Started with Campaigns](/docs/src/content/docs/guides/campaigns/getting-started.md)
- [Campaign Files Architecture](/specs/campaigns-files.md)

### Implementation Files

**Go Code:**
- `pkg/campaign/spec.go` - Data structures
- `pkg/campaign/loader.go` - Discovery and loading
- `pkg/campaign/orchestrator.go` - Orchestrator generation
- `pkg/campaign/validation.go` - Spec validation
- `pkg/cli/compile_workflow_processor.go` - Workflow processing

**JavaScript:**
- `actions/setup/js/campaign_discovery.cjs` - Discovery script
- `actions/setup/js/setup_globals.cjs` - Global utilities

**Agent Instructions:**
- `.github/aw/orchestrate-agentic-campaign.md` - Orchestration instructions
- `.github/aw/execute-agentic-campaign-workflow.md` - Worker orchestration
- `.github/aw/update-agentic-campaign-project.md` - Project update instructions

### CLI Commands

**Campaign Management:**
```bash
gh aw campaign              # List campaigns
gh aw campaign status       # Show campaign status
gh aw campaign new <id>     # Create new campaign
gh aw campaign validate     # Validate campaign specs
```

**Compilation:**
```bash
gh aw compile               # Compile all workflows (including campaigns)
gh aw compile <id>          # Compile specific campaign
```

---

## Conclusion

The campaign orchestrator/worker flow in GitHub Agentic Workflows provides a robust, scalable architecture for coordinating large-scale, multi-repository initiatives. Key strengths include:

1. **Separation of Concerns**: Clear boundaries between orchestration and execution
2. **Determinism**: Precomputed discovery ensures predictable behavior
3. **Scalability**: Budget-based pagination handles large campaigns incrementally
4. **Resilience**: Idempotent operations and error handling prevent failures
5. **Observability**: Comprehensive status updates and metrics tracking

The system is production-ready for managing complex campaigns across GitHub organizations, with strong foundations for future enhancements and extensions.

---

**Document Version:** 1.0  
**Last Updated:** 2025-01-22  
**Author:** Analysis of gh-aw campaign system
