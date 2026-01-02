## Campaign Orchestrator Rules

This orchestrator follows system-agnostic rules that enforce clean separation between workers and campaign coordination. It also maintains the campaign dashboard by ensuring the GitHub Project stays in sync with the campaign's tracker label.

### Traffic and rate limits (required)

- Minimize API calls: avoid full rescans when possible and avoid repeated reads of the same data in a single run.
- Prefer incremental processing: use deterministic ordering (e.g., by updated time) and process a bounded slice each run.
- Use strict pagination budgets: if a query would require many pages, stop early and continue next run.
- Use a durable cursor/checkpoint: persist the last processed boundary (e.g., updatedAt cutoff + last seen ID) so the next run can continue without rescanning.
- On throttling (HTTP 429 / rate limit 403), do not retry aggressively. Use backoff and end the run after reporting what remains.

{{ if .CursorGlob }}
**Cursor file (repo-memory)**: `{{ .CursorGlob }}`

You must treat this file as the source of truth for incremental discovery:
- If it exists, read it first and continue from that boundary.
- If it does not exist yet, create it by the end of the run.
- Always write the updated cursor back to the same path.
{{ end }}

{{ if .MetricsGlob }}
**Metrics/KPI snapshots (repo-memory)**: `{{ .MetricsGlob }}`

You must persist a per-run metrics snapshot (including KPI values and trends) as a JSON file stored in the metrics directory implied by the glob above.

**Required JSON schema** for each metrics file:
```json
{
  "campaign_id": "{{ .CampaignID }}",
  "date": "YYYY-MM-DD",
  "tasks_total": 0,
  "tasks_completed": 0,
  "tasks_in_progress": 0,
  "tasks_blocked": 0,
  "velocity_per_day": 0.0,
  "estimated_completion": "YYYY-MM-DD",
  "kpi_trends": [
    {"name": "KPI Name", "trend": "Improving|Flat|Regressing", "value": 0.0}
  ]
}
```

**Required fields** (must be present):
- `campaign_id` (string): Must be exactly "{{ .CampaignID }}"
- `date` (string): ISO date in YYYY-MM-DD format (use UTC)
- `tasks_total` (integer): Total number of campaign tasks (≥0)
- `tasks_completed` (integer): Number of completed tasks (≥0)

**Optional fields** (omit or set to null if not applicable):
- `tasks_in_progress` (integer): Tasks currently being worked on (≥0)
- `tasks_blocked` (integer): Tasks that are blocked (≥0)
- `velocity_per_day` (number): Average tasks completed per day (≥0)
- `estimated_completion` (string): Estimated completion date in YYYY-MM-DD format
- `kpi_trends` (array): KPI trend information with name, trend status, and current value

Guidance:
- Use an ISO date (UTC) filename, for example: `metrics/2025-12-22.json`.
- Keep snapshots append-only: write a new file per run; do not rewrite historical snapshots.
- If a KPI is present, record its computed value and trend (Improving/Flat/Regressing) in the kpi_trends array.
- Count tasks from all sources: tracker-labeled issues, worker-created issues, and project board items.
- Set tasks_total to the total number of unique tasks discovered in this run.
- Set tasks_completed to the count of tasks with state "Done" or closed status.
{{ end }}
{{ if gt .MaxDiscoveryItemsPerRun 0 }}
**Read budget**: max discovery items per run: {{ .MaxDiscoveryItemsPerRun }}
{{ end }}
{{ if gt .MaxDiscoveryPagesPerRun 0 }}
**Read budget**: max discovery pages per run: {{ .MaxDiscoveryPagesPerRun }}
{{ end }}

### Core Principles

1. **Workers are immutable** - Worker workflows never change based on campaign state
2. **Workers are campaign-agnostic** - Workers execute the same way regardless of campaign context
3. **Campaign logic is external** - All orchestration, sequencing, and decision-making happens here
4. **Workers only execute work** - No progress tracking or campaign-aware decisions in workers
5. **Campaign owns all coordination** - Sequencing, retries, continuation, and termination are campaign responsibilities
6. **State is external** - Campaign state lives in GitHub Projects, not in worker execution
7. **Single source of truth** - The GitHub Project board is the authoritative campaign state
8. **Correlation is explicit** - All work shares the campaign's tracker-id for correlation
9. **Separation of concerns** - State reads and state writes are separate operations
10. **Predefined fields only** - Only update explicitly defined project board fields
11. **Explicit outcomes** - Record actual outcomes, never infer status
12. **Idempotent operations** - Re-execution produces the same result without corruption
13. **Dashboard synchronization** - Keep Project items in sync with tracker-labeled issues/PRs

### Objective and KPIs (first-class)

{{ if .Objective }}
**Objective**: {{ .Objective }}
{{ end }}

{{ if .KPIs }}
**KPIs** (max 3):
{{ range .KPIs }}
- {{ .Name }}{{ if .Priority }} ({{ .Priority }}){{ end }}: baseline {{ .Baseline }} → target {{ .Target }} over {{ .TimeWindowDays }} days{{ if .Unit }} (unit: {{ .Unit }}){{ end }}{{ if .Direction }} (direction: {{ .Direction }}){{ end }}{{ if .Source }} (source: {{ .Source }}){{ end }}
{{ end }}
{{ end }}

If objective/KPIs are present, you must:
- Compute a per-run KPI snapshot (as-of now) using GitHub signals.
- Determine trend status for each KPI: Improving / Flat / Regressing (use the KPI direction when present).
- Tie all decisions to the primary KPI first.

### Default signals (built-in)

Collect these signals every run (bounded by the read budgets above):
- **CI health**: recent check/workflow outcomes relevant to the repo(s) in scope.
- **PR cycle time**: recent PR open→merge latency and backlog size.
- **Security alerts**: open code scanning / Dependabot / secret scanning items (as available).

If a signal cannot be retrieved (permissions/tooling), explicitly report it as unavailable and proceed with the remaining signals.

### Orchestration Workflow

Execute these steps in sequence each time this orchestrator runs:

#### Phase 1: Read State (Discovery)

1. **Query tracker-labeled items** - Search for issues and PRs matching the campaign's tracker label
   - Search: `repo:OWNER/REPO label:TRACKER_LABEL` for all open and closed items
   - If governance opt-out labels are configured, exclude items with those labels
   - Collect all matching issue/PR URLs
   - Record metadata: number, title, state (open/closed), created date, updated date

2. **Query worker-created content** (if workers are configured) - Search for issues, PRs, and discussions containing worker tracker-ids
{{ if .Workflows }}   - Worker workflows: {{ range $i, $w := .Workflows }}{{ if $i }}, {{ end }}{{ $w }}{{ end }}
   - **IMPORTANT**: You MUST perform SEPARATE searches for EACH worker workflow listed above
   - **IMPORTANT**: Workers may create different types of content (issues, PRs, discussions). Search ALL content types to discover all worker outputs.
   - Perform these searches (one per worker, searching issues, PRs, and discussions):
{{ range .Workflows }}     - Search for `{{ . }}`: 
       - Issues: `repo:OWNER/REPO "tracker-id: {{ . }}" in:body type:issue`
       - Pull Requests: `repo:OWNER/REPO "tracker-id: {{ . }}" in:body type:pr`
       - Discussions: Search discussions (no GitHub search API) by browsing recent discussions in the repository
{{ end }}{{ end }}   - For each search, collect all matching URLs (issues, PRs, discussions)
   - Record metadata for each item: number, title, state (open/closed/merged), created date, updated date, content type (issue/pr/discussion)
   - Combine results from all worker searches into a single list of discovered items

3. **Query current project state** - Read the GitHub Project board
   - Retrieve all items currently on the project board
   - For each item, record: issue URL, status field value, other predefined field values
   - Create a snapshot of current board state

4. **Compare and identify gaps** - Determine what needs updating
   - Items from step 1 or 2 not on board = **new work to add**
   - Items on board with state mismatch = **status to update**
   - Items on board with missing custom fields (e.g., worker_workflow) = **fields to populate**
   - Items on board but no longer found = **check if archived/deleted**

#### Phase 2: Make Decisions (Planning)

4.5 **Deterministic planner step (required when objective/KPIs are present)**

Before choosing additions/updates, produce a small, bounded plan that is rule-based and reproducible from the discovered state:
- Output at most **3** planned actions.
- Prefer actions that are directly connected to improving the **primary** KPI.
- If signals indicate risk or uncertainty, prefer smaller, reversible actions.

Plan format (keep under 2KB):
```json
{
   "objective": "...",
   "primary_kpi": "...",
   "kpi_trends": [{"name": "...", "trend": "Improving|Flat|Regressing"}],
   "actions": [
      {"type": "add_to_project|update_status|comment", "why": "...", "target_url": "..."}
   ]
}
```

5. **Decide additions (with pacing)** - For each new item discovered:
   - Decision: Add to board? (Default: yes for all items with tracker label or worker tracker-id)
   - If `governance.max-new-items-per-run` is set, add at most that many new items
   - Prefer adding oldest (or least recently updated) missing items first
   - Determine initial status field value based on item state:
     - Open issue/PR/discussion → "Todo" status
     - Closed issue/discussion → "Done" status
     - Merged PR → "Done" status

6. **Decide updates (no downgrade)** - For each existing board item with mismatched state:
   - Decision: Update status field? (Default: yes if item state changed)
   - If `governance.do-not-downgrade-done-items` is true, do not move items from Done back to active status
   - Determine new status field value:
     - Open issue/PR/discussion → "In Progress" or "Todo"
     - Closed issue/discussion → "Done"
     - Merged PR → "Done"

6.5 **Decide field updates** - For each existing board item, check for missing custom fields:
   - If item is missing `worker_workflow` field:
     - Search item body (issue/PR/discussion) for tracker-id (e.g., `<!-- agentic-workflow: WorkflowName, tracker-id: WORKER_ID -->`)
     - If tracker-id matches a worker in `workflows`, populate `worker_workflow` field with that worker ID
   - Only update fields that exist on the project board
   - Skip items that already have all required fields populated

7. **Decide completion** - Check campaign completion criteria:
   - If all discovered items (issues/PRs/discussions) are closed/merged AND all board items are "Done" → Campaign complete
   - Otherwise → Campaign in progress

#### Phase 3: Write State (Execution)

8. **Execute additions** - Add new items to project board
   - Use `update-project` safe-output for each new item
   - Set predefined fields: `status` (required), optionally `priority`, `size`
   - If worker tracker-id is found in item body (issue/PR/discussion), populate `worker_workflow` field
   - Record outcome: success or failure with error details

9. **Execute status updates** - Update existing board items with status changes
   - Use `update-project` safe-output for each status change
   - Update only predefined fields: `status` and related metadata
   - Record outcome: success or failure with error details

9.5 **Execute field updates** - Update existing board items with missing custom fields
   - Use `update-project` safe-output for each item with missing fields
   - Populate missing fields identified in step 6.5 (e.g., `worker_workflow`)
   - Record outcome: success or failure with error details

10. **Record completion state** - If campaign is complete:
    - Mark project metadata field `campaign_status` as "completed"
    - Do NOT create new work or modify existing items
    - This is a terminal state

#### Phase 4: Report (Output)

11. **Generate status report** - Summarize execution results:
    - Total items discovered via tracker label and worker tracker-ids (by type: issues, PRs, discussions)
    - Items added to board this run (count and URLs, by type)
    - Items updated on board this run (count and status changes)
    - Items with fields populated this run (count and which fields, e.g., worker_workflow)
    - Items skipped due to governance limits (and why)
    - Current campaign metrics: open vs closed, progress percentage
    - Any failures encountered during writes
    - Campaign completion status

### Predefined Project Fields

Only these fields may be updated on the project board:

- `status` (required) - Values: "Todo", "In Progress", "Done"
- `priority` (optional) - Values: "High", "Medium", "Low"
- `size` (optional) - Values: "Small", "Medium", "Large"
- `campaign_status` (metadata) - Values: "active", "completed"

Do NOT update any other fields or create custom fields.

### Correlation Mechanism

Workers embed a tracker-id in all created assets via XML comment:
```
<!-- agentic-workflow: WorkflowName, tracker-id: WORKER_ID -->
```

The orchestrator uses this tracker-id to discover worker output by searching bodies of issues, pull requests, and discussions. This correlation is explicit and does not require workers to be aware of the campaign.

### Idempotency Guarantee

All operations must be idempotent:
- Adding an item (issue/PR/discussion) already on the board → No-op (do not duplicate)
- Updating a status that matches current value → No-op (no change recorded)
- Marking a completed campaign as completed → No-op (terminal state preserved)

Re-running the orchestrator produces consistent results regardless of how many times it executes.
