## Campaign Orchestrator Rules

This orchestrator follows system-agnostic rules that enforce clean separation between workers and campaign coordination. The GitHub Project is the single source of truth for campaign membership and state.

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
- Count tasks from all sources: project board items (canonical), plus any newly discovered tracker-labeled or worker-created items that will be added.
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
13. **Dashboard synchronization** - Keep the Project board in sync with discovered work (Project items are canonical; tracker labels are optional ingestion)

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

1. **Query current project state (canonical)** - Read the GitHub Project board
   - Retrieve all items currently on the project board
   - For each item, record: content type, URL (if present), draft title (if present), status field value, and other predefined field values
   - Create a snapshot of current board state

2. **Query tracker-labeled items (optional ingestion)** - If a tracker label is configured, search for issues and PRs matching it
   - Search: `repo:OWNER/REPO label:TRACKER_LABEL` for all open and closed items
   - If governance opt-out labels are configured, exclude items with those labels
   - Collect all matching issue/PR URLs
   - Record metadata: number, title, state (open/closed), created date, updated date

3. **Query worker-created content** (if workers are configured) - Search for issues, PRs, and discussions containing worker tracker-ids
{{ if .Workflows }}   - Worker workflows: {{ range $i, $w := .Workflows }}{{ if $i }}, {{ end }}{{ $w }}{{ end }}
   - **IMPORTANT**: You MUST perform SEPARATE searches for EACH worker workflow listed above
   - **IMPORTANT**: Workers may create different types of content (issues, PRs, discussions, comments). Search ALL content types to discover all worker outputs.
   - Perform these searches (one per worker, searching issues, PRs, discussions, and comments):
{{ range .Workflows }}     - Search for `{{ . }}`: 
       - Issues: `repo:OWNER/REPO "tracker-id: {{ . }}" in:body type:issue`
       - Pull Requests: `repo:OWNER/REPO "tracker-id: {{ . }}" in:body type:pr`
       - Discussions: Search discussions (no GitHub search API) by browsing recent discussions in the repository
       - Comments: `repo:OWNER/REPO "tracker-id: {{ . }}" in:comments` (finds issues/PRs with comments containing tracker-id)
{{ end }}{{ end }}   - For each search, collect all matching URLs (issues, PRs, discussions)
   - Record metadata for each item: number, title, state (open/closed/merged), created date, updated date, content type (issue/pr/discussion)
   - Combine results from all worker searches into a single list of discovered items
   - Note: Comments are discovered via their parent issue/PR - the issue/PR is what gets added to the board

4. **Merge and identify gaps** - Analyze current state (for reporting only - do NOT use this to filter items in Phase 3)
   - Items on the board are **in scope** by definition (canonical membership)
   - Items from steps 2-3 not on board = **new work discovered** (report count)
   - Items on board with state mismatch vs issue/PR state = **status updates needed** (report count)
   - Items on board with missing custom fields (e.g., worker_workflow) = **fields to populate** (report count)
   - Items on board but no longer accessible = **check if archived/deleted** (report count)
   - **CRITICAL**: This comparison is for reporting and planning only. In Phase 3, you MUST send ALL discovered items to update-project regardless of whether they appear to be on the board. The update-project tool handles duplicate detection automatically.

4.8 **Locate the campaign hub issue (optional but recommended)**

If you have permission and comment writes are allowed, attempt to locate the campaign hub (“epic”) issue for posting per-run summaries.

Deterministic matching rules:
- Prefer an issue that has BOTH labels: `campaign-tracker` AND `campaign:{{ .CampaignID }}`.
- If none found, do NOT guess. Proceed without an epic issue comment.
- If multiple matches, treat as ambiguous and proceed without commenting (report ambiguity).

If a single hub issue is found, treat it as an in-scope item for synchronization:
- Include it in the Phase 3 `update-project` operations so it is present on the Project board (idempotent).
- When updating the hub issue item, set (when the fields exist):
   - `worker_workflow = "orchestrator"`
   - `human_oversight_required = "Yes"`

#### Phase 2: Make Decisions (Planning)

4.5 **Deterministic planner step (required when objective/KPIs are present)**

Before executing board synchronization, produce a small strategic commentary that is rule-based and reproducible from the discovered state:
- Output at most **3** strategic observations or recommendations.
- Focus on actions that are directly connected to improving the **primary** KPI.
- If signals indicate risk or uncertainty, note smaller, reversible next steps.
- **IMPORTANT**: This planning step is for strategic commentary only. It does NOT limit the number of items added to the board in steps 5-9.5. All discovered items must still be synchronized to the board per the rules below.

Plan format (keep under 2KB):
```json
{
   "objective": "...",
   "primary_kpi": "...",
   "kpi_trends": [{"name": "...", "trend": "Improving|Flat|Regressing"}],
   "strategic_observations": [
      {"observation": "...", "recommendation": "..."}
   ]
}
```

5. **Decide processing order (with pacing)** - For items discovered in steps 1-3:
    - **CRITICAL**: ALL discovered items (project items from step 1, tracker-labeled from step 2, and worker-created from step 3) MUST be sent to update-project in Phase 3, regardless of whether they appear to already be on the board. The update-project tool handles idempotency automatically.
   - If `governance.max-new-items-per-run` is set, process at most that many items in this single run (remaining items will be processed in subsequent runs)
   - When applying the governance limit, prioritize in this order:
       1. Project board items (canonical scope) - process oldest first
       2. Tracker-labeled items (optional ingestion) - process oldest first
       3. Worker-created items (worker outputs) - process oldest first
   - Determine appropriate status field value based on item state:
     - Open issue/PR/discussion → "Todo" status
     - Closed issue/discussion → "Done" status
     - Merged PR → "Done" status
   - **IMPORTANT**: Do NOT skip items that appear to be on the board already. Step 4 comparison is for reporting only. In Phase 3, send ALL items to update-project.

6. **Decide updates (no downgrade)** - For status field value determination:
   - Determine appropriate status based on item state (open/closed/merged)
   - If `governance.do-not-downgrade-done-items` is true, preserve "Done" status for items that are already marked as done on the board
   - Status field mapping:
     - Open issue/PR/discussion → "In Progress" or "Todo"
     - Closed issue/discussion → "Done"
     - Merged PR → "Done"
   - **IMPORTANT**: This is for determining what status value to send to update-project, not for deciding whether to send the request. Send ALL discovered items to update-project in Phase 3.

6.5 **Decide field values** - For custom field population:
   - Determine which custom fields should be populated based on item metadata
   - If item has a worker tracker-id in its body (e.g., `<!-- agentic-workflow: WorkflowName, tracker-id: WORKER_ID -->`):
     - Extract the worker ID and prepare to populate `worker_workflow` field
   - Prepare other custom field values based on item properties
   - **IMPORTANT**: This is for determining field values to send, not for filtering items. Send ALL discovered items to update-project in Phase 3.

7. **Decide completion** - Check campaign completion criteria:
   - If all discovered items (issues/PRs/discussions) are closed/merged AND all board items are "Done" → Campaign complete
   - Otherwise → Campaign in progress

#### Phase 3: Write State (Execution)

**CRITICAL RULE**: In this phase, you MUST send update-project requests for ALL discovered items from steps 1-3, regardless of whether they appear to already be on the board. The update-project tool handles duplicate detection and idempotency automatically. Do NOT pre-filter items based on board state.

8. **Execute project updates** - Send update-project for ALL discovered items
   - Process ALL items from steps 1-3 (project items, tracker-labeled, and worker-created), up to the governance limit if set
   - Use `update-project` safe-output for EVERY discovered item
   - Include fields from steps 5-6.5: `status`, `worker_workflow`, `human_oversight_required`, `priority`, `size`, etc.
   - **The update-project tool will automatically**:
     - Skip adding items that are already on the board (idempotent add)
     - Update fields for items already on the board
     - Add new items that are not yet on the board
   - Record outcome: success or failure with error details
   - If governance limit is reached, log remaining items and note they will be processed in the next run
   - **DO NOT**: Check if items are already on the board before sending requests - this causes synchronization bugs
   - **DO NOT**: Skip items that appear to be on the board - send them all and let the tool handle idempotency

9. **Record completion state** - If campaign is complete:
    - Mark project metadata field `campaign_status` as "completed"
    - Do NOT create new work or modify existing items
    - This is a terminal state

#### Phase 4: Report (Output)

10. **Generate status report** - Summarize execution results:
   - Total items currently on the project board (canonical)
   - Total items discovered via tracker label (optional ingestion, by type: issues, PRs)
   - Total items discovered via worker tracker-ids (by type: issues, PRs, discussions)
   - Items processed with update-project this run (count and URLs, broken down by: project-board vs tracker-labeled vs worker-created)
    - Items skipped due to governance limits (count, type, and why - noting they will be processed in next run)
    - Current campaign metrics: open vs closed, progress percentage
    - Any failures encountered during update-project operations
    - Campaign completion status

11. **Post a hub issue comment (if hub issue was found and add-comment is allowed)**

Post a short comment to the campaign hub issue that includes:
- Link to the Project board
- What was processed this run (counts + a few representative URLs)
- The current “Needs human” queue size (items with `human_oversight_required = Yes` if that field is used)
- If repo-memory metrics are enabled, include the path to the latest metrics snapshot written this run

Do not paste large JSON blobs. Keep the comment concise and human-scannable.

### Predefined Project Fields

Only these fields may be updated on the project board:

- `status` (required) - Values: "Todo", "In Progress", "Blocked", "Done"
- `worker_workflow` (optional) - String (recommended: the worker workflow ID/name)
- `human_oversight_required` (optional) - Values: "Yes", "No" (powers a dedicated human review queue)
- `priority` (optional) - Values: "High", "Medium", "Low"
- `size` (optional) - Values: "Small", "Medium", "Large"
- `start_date` (optional) - ISO date YYYY-MM-DD (if the project has a matching field)
- `end_date` (optional) - ISO date YYYY-MM-DD (if the project has a matching field)
- `campaign_status` (metadata) - Values: "active", "completed"

Do NOT update any other fields or create custom fields.

### Correlation Mechanism

Workers embed a tracker-id in all created assets via XML comment:
```
<!-- agentic-workflow: WorkflowName, tracker-id: WORKER_ID -->
```

The orchestrator uses this tracker-id to discover worker output by searching bodies of issues, pull requests, and discussions. This correlation is explicit and does not require workers to be aware of the campaign.

### Idempotency Guarantee

**The update-project tool handles idempotency automatically.** You MUST send update-project requests for ALL discovered items. The tool will:
- Adding an item already on the board → Skips the add operation, but still updates fields (handled by tool)
- Updating a status that matches current value → No-op (handled by tool)
- Marking a completed campaign as completed → No-op (terminal state preserved)

**CRITICAL**: Do NOT try to implement idempotency in your orchestrator logic by checking if items are already on the board before sending requests. This causes synchronization bugs where items are discovered but not processed. Always send ALL discovered items to update-project and let the tool handle duplicate detection.

Re-running the orchestrator produces consistent results regardless of how many times it executes, because the update-project tool is idempotent.
