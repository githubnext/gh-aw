# Orchestrator Instructions

Coordinate campaign by: discovering worker outputs → making deterministic decisions → syncing state to GitHub Project board.

**Scope**: orchestration only (discovery, planning, pacing, reporting)  
**Authority**: Project Update Instructions govern all writes

## Budgets & Limits
{{ if .CursorGlob }}
**Cursor**: `/tmp/gh-aw/repo-memory/campaigns/{{.CampaignID}}/cursor.json` - Read to continue from checkpoint, write updated state.
{{ end }}
{{ if .MetricsGlob }}
**Metrics**: `/tmp/gh-aw/repo-memory/campaigns/{{.CampaignID}}/metrics/YYYY-MM-DD.json` - Append-only snapshots with required fields: `campaign_id`, `date`, `tasks_total`, `tasks_completed`.
{{ end }}
{{ if gt .MaxDiscoveryItemsPerRun 0 }}Max discovery items: {{ .MaxDiscoveryItemsPerRun }}{{ end }}{{ if gt .MaxDiscoveryPagesPerRun 0 }} | Max pages: {{ .MaxDiscoveryPagesPerRun }}{{ end }}{{ if gt .MaxProjectUpdatesPerRun 0 }} | Max project updates: {{ .MaxProjectUpdatesPerRun }}{{ end }}{{ if gt .MaxProjectCommentsPerRun 0 }} | Max comments: {{ .MaxProjectCommentsPerRun }}{{ end }}

## Core Principles
1. Workers are immutable/campaign-agnostic
2. Project board = authoritative state
3. Correlation is explicit (tracker-id)
4. Reads and writes separate (no interleaving)
5. Idempotent operations
6. Project Update Instructions take precedence

## Execution (4 Steps, Strict Order)

### Step 0: Epic Issue [First Run Only]
Search for existing Epic issue (label `epic`/`type:epic`, body contains `campaign_id: {{.CampaignID}}`). If missing, create via `create-issue` with title "{{if .CampaignName}}{{.CampaignName}}{{else}}{{.CampaignID}}{{end}}", body containing objective/details/`campaign_id: {{.CampaignID}}`, labels `epic`+`type:epic`. Add to project with status `In Progress`, worker_workflow `unknown`, appropriate dates. Record issue number in cursor.

### Step 1: Read State [NO WRITES]
1. Read the precomputed discovery manifest: `./.gh-aw/campaign.discovery.json`
2. Read project board state
3. Parse items: url, content_type, number, repo, dates, state
4. Check summary: `needs_add_count`, `needs_update_count` (skip to Step 4 if both 0)

### Step 2: Plan [NO WRITES]  
1. Map status from GitHub state: Open→`Todo`, Closed→`Done`, Merged→`Done`
2. Calculate dates: `start_date` from `created_at`, `end_date` from `closed_at`/`merged_at` or today
3. Apply write budget (select max items using oldest-first ordering)
4. Trust safe-output deduplication (no board comparison needed)

### Step 3: Write [WRITES ONLY]
Send `update-project` for each selected item per Project Update Instructions:
- First add: full fields (status, campaign_id, worker_workflow, repo, priority, size, dates)
- Existing: status-only update (unless backfill repair needed)
- Record outcomes (success/failure)

### Step 4: Report & Status Update
**REQUIRED**: Create `create-project-status-update` summarizing run with:
- Most Important Findings (2-3 key insights)
- What Was Learned
- KPI Trends{{ if .KPIs }} ({{ range $i, $kpi := .KPIs }}{{if $i}}, {{end}}{{ $kpi.Name }}{{end}}){{ end }} - baseline → current → target with velocity
- Campaign Summary (tasks completed/in-progress/blocked, completion %)
- Next Steps

Set status (ON_TRACK/AT_RISK/OFF_TRACK/COMPLETE), dates, concise body (200-400 words).

Report counts: discovered (by type), processed (add/update/backfill/noop/failed), deferred, failures, cursor state.

## Authority
Project Update Instructions take precedence for all writes.
