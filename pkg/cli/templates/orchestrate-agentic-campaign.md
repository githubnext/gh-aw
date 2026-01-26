# Orchestrator Instructions

This orchestrator coordinates a single campaign by discovering worker outputs and making deterministic decisions.

**Scope:** orchestration only (discovery, planning, pacing, reporting).
**Actuation model:** **dispatch-only** — the orchestrator may only act by dispatching allowlisted worker workflows.
**Write authority:** all GitHub writes (Projects, issues/PRs, comments, status updates) must happen in worker workflows.

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
{{ end }}

{{ if gt .MaxDiscoveryItemsPerRun 0 }}
**Read budget**: max discovery items per run: {{ .MaxDiscoveryItemsPerRun }}
{{ end }}
{{ if gt .MaxDiscoveryPagesPerRun 0 }}
**Read budget**: max discovery pages per run: {{ .MaxDiscoveryPagesPerRun }}
{{ end }}

---

## Core Principles

1. Workers are immutable and campaign-agnostic
2. The GitHub Project board is the authoritative campaign state
3. Correlation is explicit (tracker-id AND labels)
4. Reads and writes are separate steps (never interleave)
5. Idempotent operation is mandatory (safe to re-run)
6. Orchestrators do not write GitHub state directly

---

## Execution Steps (Required Order)

### Step 1 — Read State (Discovery) [NO WRITES]

**IMPORTANT**: Discovery has been precomputed. Read the discovery manifest instead of performing GitHub-wide searches.

1) Read the precomputed discovery manifest: `./.gh-aw/campaign.discovery.json`

2) Parse discovered items from the manifest:
   - Each item has: url, content_type (issue/pull_request/discussion), number, repo, created_at, updated_at, state
   - Closed items have: closed_at (for issues) or merged_at (for PRs)
   - Items are pre-sorted by updated_at for deterministic processing

3) Check the manifest summary for work counts.

4) Discovery cursor is maintained automatically in repo-memory; do not modify it manually.

### Step 2 — Make Decisions (Planning) [NO WRITES]

5) Determine desired `status` strictly from explicit GitHub state:
- Open → `Todo` (or `In Progress` only if explicitly indicated elsewhere)
- Closed (issue/discussion) → `Done`
- Merged (PR) → `Done`

6) Calculate required date fields (for workers that sync Projects):
- `start_date`: format `created_at` as `YYYY-MM-DD`
- `end_date`:
  - if closed/merged → format `closed_at`/`merged_at` as `YYYY-MM-DD`
  - if open → **today's date** formatted `YYYY-MM-DD`

7) Reads and writes are separate steps (never interleave).

### Step 3 — Dispatch Workers (Execution) [DISPATCH ONLY]

8) For each selected unit of work, dispatch a worker workflow using `dispatch-workflow`.

Constraints:
- Only dispatch allowlisted workflows.
- Keep within the dispatch-workflow max for this run.

### Step 4 — Report (No Writes)

9) Summarize what you dispatched, what remains, and what should run next.

If a status update is required on the GitHub Project, dispatch a dedicated reporting/sync worker to perform that write.

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

    ## Campaign Progress

    **Documentation Coverage** (Primary Metric):
    - Baseline: 85% → Current: 88% → Target: 95%
    - Direction: ↑ Increasing (+3% this week, +1% velocity/week)
    - Status: ON TRACK - At current velocity, will reach 95% in 7 weeks

    **Accessibility Score** (Supporting Metric):
    - Baseline: 90% → Current: 91% → Target: 98%
    - Direction: ↑ Increasing (+1% this month)
    - Status: AT RISK - Slower progress than expected, may need dedicated focus

    **User-Reported Issues** (Supporting Metric):
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
