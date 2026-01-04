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
- If it exists: read first and continue from its boundary.  
- If it does not exist: create it by end of run.  
- Always write the updated cursor back to the same path.
{{ end }}

{{ if .MetricsGlob }}
**Metrics snapshots (repo-memory)**: `{{ .MetricsGlob }}`  
- Persist one append-only JSON metrics snapshot per run (new file per run; do not rewrite history).
- Use UTC date (`YYYY-MM-DD`) in the filename (example: `metrics/2025-12-22.json`).
- Each snapshot MUST include `campaign_id` and `date` (UTC).
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
       repo: "<OWNER/REPO>"
       priority: "High"
       size: "Large"
       start_date: "<EPIC_CREATED_DATE_YYYY-MM-DD>"
       end_date: "<TODAY_YYYY-MM-DD>"
   ```

4) **Record the Epic issue number** in repo-memory for reference (e.g., in cursor file or metadata).

**Note:** This phase typically runs only on the first orchestrator execution. On subsequent runs, verify the Epic exists and is on the board, but do not recreate it.

---

### Phase 1 — Read State (Discovery) [NO WRITES]

1) Read current GitHub Project board state (items + required fields).

2) Discover worker outputs (if workers are configured):
{{ if .Workflows }}
- Perform separate discovery per worker workflow:
{{ range .Workflows }}
  - Search for tracker-id `{{ . }}` across issues/PRs/discussions/comments (parent issue/PR is the unit of work).
{{ end }}
{{ end }}

3) Normalize discovered items into a single list with:
- URL, `content_type` (issue/pull_request/discussion), `content_number`
- `repo` (owner/repo), `created_at`, `updated_at`
- `state` (open/closed/merged), `closed_at`/`merged_at` when applicable

4) Respect read budgets and cursor; stop early if needed and persist cursor.

### Phase 2 — Make Decisions (Planning) [NO WRITES]

5) Determine desired `status` strictly from explicit GitHub state:
- Open → `Todo` (or `In Progress` only if explicitly indicated elsewhere)
- Closed (issue/discussion) → `Done`
- Merged (PR) → `Done`

6) Calculate required date fields for each item (per Project Update Instructions):
- `start_date`: format `created_at` as `YYYY-MM-DD`
- `end_date`:
  - if closed/merged → format `closed_at`/`merged_at` as `YYYY-MM-DD`
  - if open → **today's date** formatted `YYYY-MM-DD` (required for roadmap view)

7) Do NOT implement idempotency by comparing against the board. You may compare for reporting only.

8) Apply write budget:
- If `MaxProjectUpdatesPerRun > 0`, select at most that many items this run using deterministic order
  (e.g., oldest `updated_at` first; tie-break by ID/number).
- Defer remaining items to next run via cursor.

### Phase 3 — Write State (Execution) [WRITES ONLY]

9) For each selected item, send an `update-project` request.
- Do NOT interleave reads.
- Do NOT pre-check whether the item is on the board.
- **All write semantics MUST follow Project Update Instructions**, including:
  - first add → full required fields (status, campaign_id, worker_workflow, repo, priority, size, start_date, end_date)
  - existing item → status-only update unless explicit backfill is required

10) Record per-item outcome: success/failure + error details.

### Phase 4 — Report

11) Report:
- counts discovered (by type)
- counts processed this run (by action: add/status_update/backfill/noop/failed)
- counts deferred due to budgets
- failures (with reasons)
- completion state (work items only)
- cursor advanced / remaining backlog estimate

---

## Authority

If any instruction in this file conflicts with **Project Update Instructions**, the Project Update Instructions win for all project writes.
