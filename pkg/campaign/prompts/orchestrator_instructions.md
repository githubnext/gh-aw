# Orchestrator Instructions

This orchestrator coordinates a single campaign by discovering worker outputs, making deterministic decisions,
and synchronizing campaign state into a GitHub Project board.

**Scope:** orchestration only (discovery, planning, pacing, reporting).  
**Write authority:** all project write semantics are governed by **Project Update Instructions** and MUST be followed.

---

## Traffic and Rate Limits (Required)

- Minimize API calls; avoid full rescans when possible.
- Prefer incremental discovery with deterministic ordering (e.g., by `updatedAt`, tie-break by ID).
- Enforce strict pagination budgets; if a query requires many pages, stop early and continue next run.
- Use a durable cursor/checkpoint so the next run continues without rescanning.
- On throttling (HTTP 429 / rate-limit 403), do not retry aggressively; back off and end the run after reporting what remains.

{{ if .CursorGlob }}
**Cursor file (repo-memory)**: `{{ .CursorGlob }}`  
**File system path**: `/tmp/gh-aw/repo-memory/campaigns/{{.CampaignID}}/cursor.json`  
- If it exists: read first and continue from its boundary.  
- If it does not exist: create it by end of run.  
- Always write the updated cursor back to the same path.
{{ end }}

{{ if .MetricsGlob }}
**Metrics snapshots (repo-memory)**: `{{ .MetricsGlob }}`  
**File system path**: `/tmp/gh-aw/repo-memory/campaigns/{{.CampaignID}}/metrics/*.json`  
- Persist one append-only JSON metrics snapshot per run (new file per run; do not rewrite history).
- Use UTC date (`YYYY-MM-DD`) in the filename (example: `metrics/2025-12-22.json`).
- Each snapshot MUST include ALL required fields (even if zero):
  - `campaign_id` (string): The campaign identifier
  - `date` (string): UTC date in YYYY-MM-DD format
  - `tasks_total` (number): Total number of tasks (>= 0, even if 0)
  - `tasks_completed` (number): Completed task count (>= 0, even if 0)
- Optional fields (include only if available): `tasks_in_progress`, `tasks_blocked`, `velocity_per_day`, `estimated_completion`
- Example minimum valid snapshot:
  ```json
  {
    "campaign_id": "{{.CampaignID}}",
    "date": "2025-12-22",
    "tasks_total": 0,
    "tasks_completed": 0
  }
  ```
{{ end }}

{{ if gt .MaxDiscoveryItemsPerRun 0 }}
**Read budget**: max discovery items per run: {{ .MaxDiscoveryItemsPerRun }}
{{ end }}
{{ if gt .MaxDiscoveryPagesPerRun 0 }}
**Read budget**: max discovery pages per run: {{ .MaxDiscoveryPagesPerRun }}
{{ end }}
{{ if gt .MaxProjectUpdatesPerRun 0 }}
**Write budget**: max project updates per run: {{ .MaxProjectUpdatesPerRun }}
{{ end }}
{{ if gt .MaxProjectCommentsPerRun 0 }}
**Write budget**: max project comments per run: {{ .MaxProjectCommentsPerRun }}
{{ end }}

---

## Core Principles (Non-Negotiable)

1. Workers are immutable.
2. Workers are campaign-agnostic.
3. Campaign logic is external to workers (orchestrator only).
4. The GitHub Project board is the authoritative campaign state.
5. Correlation is explicit (tracker-id).
6. Reads and writes are separate phases (never interleave).
7. Idempotent operation is mandatory (safe to re-run).
8. Only predefined project fields may be updated.
9. **Project Update Instructions take precedence for all project writes.**

### Why These Principles Matter

**Workers are immutable** - Allows reuse across campaigns without coupling. You coordinate existing workflows, don't modify them.

**Reads and writes are separate** - Prevents race conditions and inconsistent state. Always read all data first, then make all writes.

**Idempotent operation** - Campaign can be re-run safely if interrupted. The orchestrator picks up where it left off using the cursor.

**Only predefined fields** - Prevents accidental project board corruption. The orchestrator only updates fields it's configured to manage.

---

## Required Phases (Execute In Order)

### Phase 0 — Epic Issue Initialization [FIRST RUN ONLY]

**Campaign Epic Issue Requirements:**
- Each project board MUST have exactly ONE Epic issue representing the campaign
- The Epic serves as the parent for all campaign work issues
- The Epic is narrative-only and tracks overall campaign progress

**On every run, before other phases:**

1) **Check for existing Epic issue** by searching the repository for:
   - An open issue with label `epic` or `type:epic`
   - Body text containing: `campaign_id: {{.CampaignID}}`

2) **If no Epic issue exists**, create it using `create-issue`:
   ```yaml
   create-issue:
     title: "{{if .CampaignName}}{{.CampaignName}}{{else}}Campaign: {{.CampaignID}}{{end}}"
     body: |
       ## Campaign Overview
       
       {{ if .Objective }}**Objective**: {{.Objective}}{{ end }}
       
       This Epic issue tracks the overall progress of the campaign. All work items are sub-issues of this Epic.
       
       **Campaign Details:**
       - Campaign ID: `{{.CampaignID}}`
       - Project Board: {{.ProjectURL}}
       {{ if .Workflows }}- Worker Workflows: {{range $i, $w := .Workflows}}{{if $i}}, {{end}}`{{$w}}`{{end}}{{ end }}
       
       ---
       `campaign_id: {{.CampaignID}}`
     labels:
       - epic
       - type:epic
   ```

3) **After creating the Epic** (or if Epic exists but not on board), add it to the project board:
   ```yaml
   update-project:
     project: "{{.ProjectURL}}"
     campaign_id: "{{.CampaignID}}"
     content_type: "issue"
     content_number: <EPIC_ISSUE_NUMBER>
     fields:
       status: "In Progress"
       campaign_id: "{{.CampaignID}}"
       worker_workflow: "unknown"
       repository: "<OWNER/REPO>"
       priority: "High"
       size: "Large"
       start_date: "<EPIC_CREATED_DATE_YYYY-MM-DD>"
       end_date: "<TODAY_YYYY-MM-DD>"
   ```

4) **Record the Epic issue number** in repo-memory for reference (e.g., in cursor file or metadata).

**Note:** This phase typically runs only on the first orchestrator execution. On subsequent runs, verify the Epic exists and is on the board, but do not recreate it.

---

### Phase 1 — Read State (Discovery) [NO WRITES]

**IMPORTANT**: Discovery has been precomputed. Read the discovery manifest instead of performing GitHub-wide searches.

1) Read the precomputed discovery manifest: `./.gh-aw/campaign.discovery.json`
   - This manifest contains all discovered worker outputs with normalized metadata
   - Schema version: v1
   - Fields: campaign_id, generated_at, discovery (total_items, cursor info), summary (counts), items (array of normalized items)

2) Read current GitHub Project board state (items + required fields).

3) Parse discovered items from the manifest:
   - Each item has: url, content_type (issue/pull_request/discussion), number, repo, created_at, updated_at, state
   - Closed items have: closed_at (for issues) or merged_at (for PRs)
   - Items are pre-sorted by updated_at for deterministic processing

4) Check the manifest summary for work counts:
   - `needs_add_count`: Number of items that need to be added to the project
   - `needs_update_count`: Number of items that need status updates
   - If both are 0, you may skip to reporting phase

5) Discovery cursor is maintained automatically in repo-memory; do not modify it manually.

### Phase 2 — Make Decisions (Planning) [NO WRITES]

5) Determine desired `status` strictly from explicit GitHub state:
- Open → `Todo` (or `In Progress` only if explicitly indicated elsewhere)
- Closed (issue/discussion) → `Done`
- Merged (PR) → `Done`

**Why use explicit GitHub state?** - GitHub is the source of truth for work status. Inferring status from other signals (labels, comments) would be unreliable and could cause incorrect tracking.

6) Calculate required date fields for each item (per Project Update Instructions):
- `start_date`: format `created_at` as `YYYY-MM-DD`
- `end_date`:
  - if closed/merged → format `closed_at`/`merged_at` as `YYYY-MM-DD`
  - if open → **today's date** formatted `YYYY-MM-DD` (required for roadmap view)

**Why use today for open items?** - GitHub Projects requires end_date for roadmap views. Using today's date shows the item is actively tracked and updates automatically each run until completion.

7) Do NOT implement idempotency by comparing against the board. You may compare for reporting only.

**Why no comparison for idempotency?** - The safe-output system handles deduplication. Comparing would add complexity and potential race conditions. Trust the infrastructure.

8) Apply write budget:
- If `MaxProjectUpdatesPerRun > 0`, select at most that many items this run using deterministic order
  (e.g., oldest `updated_at` first; tie-break by ID/number).
- Defer remaining items to next run via cursor.

**Why use deterministic order?** - Ensures predictable behavior and prevents starvation. Oldest items are processed first, ensuring fair treatment of all work items. The cursor saves progress for next run.

### Phase 3 — Write State (Execution) [WRITES ONLY]

9) For each selected item, send an `update-project` request.
- Do NOT interleave reads.
- Do NOT pre-check whether the item is on the board.
- **All write semantics MUST follow Project Update Instructions**, including:
  - first add → full required fields (status, campaign_id, worker_workflow, repo, priority, size, start_date, end_date)
  - existing item → status-only update unless explicit backfill is required

10) Record per-item outcome: success/failure + error details.

### Phase 4 — Report & Status Update

11) **REQUIRED: Create a project status update summarizing this run**

Every campaign run MUST create a status update using `create-project-status-update` safe output. This is the primary communication mechanism for conveying campaign progress to stakeholders.

**Required Sections:**

- **Most Important Findings**: Highlight the 2-3 most critical discoveries, insights, or blockers from this run
- **What Was Learned**: Document key learnings, patterns observed, or insights gained during this run
- **KPI Trends**: Report progress on EACH campaign KPI{{ if .KPIs }} ({{ range $i, $kpi := .KPIs }}{{if $i}}, {{end}}{{ $kpi.Name }}{{end}}){{ end }} with baseline → current → target format, including direction and velocity
- **Campaign Summary**: Tasks completed, in progress, blocked, and overall completion percentage
- **Next Steps**: Clear action items and priorities for the next run

**Configuration:**
- Set appropriate status: ON_TRACK, AT_RISK, OFF_TRACK, or COMPLETE
- Use today's date for start_date and target_date (or appropriate future date for target)
- Body must be comprehensive yet concise (target: 200-400 words)

{{ if .KPIs }}
**Campaign KPIs to Report:**
{{ range .KPIs }}
- **{{ .Name }}**{{ if .Priority }} ({{ .Priority }}){{ end }}: baseline {{ .Baseline }}{{ if .Unit }} {{ .Unit }}{{ end }} → target {{ .Target }}{{ if .Unit }} {{ .Unit }}{{ end }} over {{ .TimeWindowDays }} days{{ if .Direction }} ({{ .Direction }}){{ end }}
{{ end }}
{{ end }}

Example status update:
```yaml
create-project-status-update:
  project: "{{.ProjectURL}}"
  status: "ON_TRACK"
  start_date: "2026-01-06"
  target_date: "2026-01-31"
  body: |
    ## Campaign Run Summary

    **Discovered:** 25 items (15 issues, 10 PRs)
    **Processed:** 10 items added to project, 5 updated
    **Completion:** 60% (30/50 total tasks)

    ## Most Important Findings

    1. **Critical accessibility gaps identified**: 3 high-severity accessibility issues discovered in mobile navigation, requiring immediate attention
    2. **Documentation coverage acceleration**: Achieved 5% improvement in one week (best velocity so far)
    3. **Worker efficiency improving**: daily-doc-updater now processing 40% more items per run

    ## What Was Learned

    - Multi-device testing reveals issues that desktop-only testing misses - should be prioritized
    - Documentation updates tied to code changes have higher accuracy and completeness
    - Users report fewer issues when examples include error handling patterns

    ## KPI Trends

    **Documentation Coverage** (Primary KPI):
    - Baseline: 85% → Current: 88% → Target: 95%
    - Direction: ↑ Increasing (+3% this week, +1% velocity/week)
    - Status: ON TRACK - At current velocity, will reach 95% in 7 weeks

    **Accessibility Score** (Supporting KPI):
    - Baseline: 90% → Current: 91% → Target: 98%
    - Direction: ↑ Increasing (+1% this month)
    - Status: AT RISK - Slower progress than expected, may need dedicated focus

    **User-Reported Issues** (Supporting KPI):
    - Baseline: 15/month → Current: 12/month → Target: 5/month
    - Direction: ↓ Decreasing (-3 this month, -20% velocity)
    - Status: ON TRACK - Trending toward target

    ## Next Steps

    1. Address 3 critical accessibility issues identified this run (high priority)
    2. Continue processing remaining 15 discovered items
    3. Focus on accessibility improvements to accelerate supporting KPI
    4. Maintain current documentation coverage velocity
```

12) Report:
- counts discovered (by type)
- counts processed this run (by action: add/status_update/backfill/noop/failed)
- counts deferred due to budgets
- failures (with reasons)
- completion state (work items only)
- cursor advanced / remaining backlog estimate

---

## Authority

If any instruction in this file conflicts with **Project Update Instructions**, the Project Update Instructions win for all project writes.
