---
title: "Campaign Specs"
description: "Define and configure agentic campaigns with spec files, tracker labels, and recommended wiring"
---

Campaigns are defined as Markdown files under `.github/workflows/` with a `.campaign.md` suffix. The YAML frontmatter is the campaign “contract”; the body can contain optional narrative context.

## What a campaign is (in gh-aw)

In GitHub Agentic Workflows, a campaign is not “a special kind of workflow.” The `.campaign.md` file is a specification: a reviewable contract that wires together agentic workflows around a shared initiative (a GitHub Project dashboard as the canonical source of membership, optional tracker label for ingestion, and optional durable state).

In a typical setup:

- Worker workflows do the work. They run an agent and use safe-outputs (for example `create_pull_request`, `add_comment`, or `update_issues`) for write operations.
- A generated orchestrator workflow keeps the campaign coherent over time. It discovers items from the project board (optionally using tracker labels), updates the Project board, and produces ongoing progress reporting.
- Repo-memory (optional) makes the campaign repeatable. It lets you store a cursor checkpoint and append-only metrics snapshots so each run can pick up where the last one left off.

**Note:** The `.campaign.g.md` file is a local debug artifact generated during compilation to help developers review the orchestrator structure. It is not committed to git (it's in `.gitignore`). Only the source `.campaign.md` and the compiled `.campaign.lock.yml` are version controlled.

## Minimal spec

```yaml
# .github/workflows/framework-upgrade.campaign.md
id: framework-upgrade
version: "v1"
name: "Framework Upgrade"
description: "Move services to Framework vNext"

project-url: "https://github.com/orgs/ORG/projects/1"
tracker-label: "campaign:framework-upgrade"

objective: "Upgrade all services to Framework vNext with zero downtime."
kpis:
  - id: services_upgraded
    name: "Services upgraded"
    primary: true
    direction: "increase"
    target: 50
  - id: incidents
    name: "Incidents caused"
    direction: "decrease"
    target: 0

workflows:
  - framework-upgrade

state: "active"
owners:
  - "platform-team"
```

## Core fields (what they do)

- `id`: stable identifier used for file naming, reporting, and (if used) repo-memory paths.
- `project-url`: the GitHub Project that acts as the campaign dashboard and canonical source of campaign membership.
- `tracker-label` (optional): an ingestion hint label that helps discover issues and pull requests created by workers (commonly `campaign:<id>`). When provided, the orchestrator can discover work across runs. The project board remains the canonical source of truth.
- `objective`: a single sentence describing what “done” means.
- `kpis`: the measures you use to report progress (exactly one should be marked `primary`).
- `workflows`: the participating workflow IDs. These refer to workflows in the repo (commonly `.github/workflows/<workflow-id>.md`), and they can be scheduled, event-driven, or long-running.

## KPIs (recommended shape)

Keep KPIs small and crisp:

- Use 1 primary KPI + a few supporting KPIs.
- Use `direction: increase|decrease|maintain` to describe the desired trend.
- Use `target` when there is a clear threshold.

If you define `kpis`, also define `objective` (and vice versa). It keeps the spec reviewable and makes reports consistent.

## Durable state (repo-memory)

If you use repo-memory for campaigns, standardize on one layout so runs are comparable:

- `memory/campaigns/<campaign-id>/cursor.json` - Checkpoint for discovery
- `memory/campaigns/<campaign-id>/metrics/<date>.json` - Daily metrics snapshots

### Cursor File

Opaque JSON object maintained by orchestrator. Contains:
- `last_updated_at`: Timestamp of most recent item processed
- `last_item_id`: Identifier for resumption
- Additional campaign-specific state

**Do not manually edit cursor files.** The orchestrator manages them automatically.

### Metrics Snapshots

One JSON file per run, named with UTC date: `YYYY-MM-DD.json`

**Required Fields (must always be present):**
```json
{
  "campaign_id": "campaign-id",
  "date": "2026-01-05",
  "tasks_total": 50,
  "tasks_completed": 25
}
```

**Optional Fields (include when available):**
```json
{
  "tasks_in_progress": 15,
  "tasks_blocked": 3,
  "velocity_per_day": 3.5,
  "estimated_completion": "2026-02-15"
}
```

**Field Descriptions:**
- `campaign_id`: Campaign identifier (must match spec)
- `date`: UTC date in YYYY-MM-DD format
- `tasks_total`: Total number of tasks in scope (≥ 0)
- `tasks_completed`: Completed task count (≥ 0, ≤ tasks_total)
- `tasks_in_progress`: Currently active tasks (optional)
- `tasks_blocked`: Tasks awaiting resolution (optional)
- `velocity_per_day`: Average completion rate (optional)
- `estimated_completion`: Projected date YYYY-MM-DD (optional)

Typical wiring in the spec:

```yaml
memory-paths:
  - "memory/campaigns/framework-upgrade/cursor.json"
  - "memory/campaigns/framework-upgrade/metrics/*.json"
metrics-glob: "memory/campaigns/framework-upgrade/metrics/*.json"
cursor-glob: "memory/campaigns/framework-upgrade/cursor.json"
```

**Validation:** Campaign tooling enforces that repo-memory writes include a cursor and at least one metrics snapshot with all required fields.

## Governance (pacing & safety)

Use `governance` to keep orchestration predictable and reviewable:

```yaml
governance:
  # Discovery budgets (controls precomputed discovery phase)
  max-discovery-items-per-run: 100      # Max items to discover per run
  max-discovery-pages-per-run: 5        # Max API pages to fetch per run
  
  # Write budgets (controls orchestrator execution)
  max-new-items-per-run: 10             # Max new items to add to project
  max-project-updates-per-run: 50       # Max total project updates
  max-comments-per-run: 10              # Max comments to post
  
  # Behavior controls
  opt-out-labels: ["campaign:skip"]     # Items to exclude
  do-not-downgrade-done-items: true     # Prevent Done → other transitions
```

### Budget Guidelines

**Discovery Budgets:**
- **Purpose:** Control API usage during precomputed discovery phase
- **`max-discovery-items-per-run`:** Total items to discover (default: 100)
- **`max-discovery-pages-per-run`:** API pagination limit (default: 5-10 pages)
- **When to increase:** Large campaigns with many tracked items
- **When to decrease:** API rate limit concerns

**Write Budgets:**
- **Purpose:** Control project updates and comment activity
- **`max-new-items-per-run`:** Prevent overwhelming project board
- **`max-project-updates-per-run`:** Total field updates allowed
- **`max-comments-per-run`:** Limit comment activity

Discovery budgets affect what's available; write budgets control actual updates. Set discovery budgets higher than write budgets to maintain a processing backlog.

## Discovery System

Campaign orchestrators use a two-phase discovery approach for efficiency and determinism:

### Phase 0: Precomputed Discovery

Before the agent executes, a JavaScript-based discovery step:
1. Loads cursor from repo-memory (if exists)
2. Searches for worker outputs using tracker labels
3. Applies pagination budgets from governance configuration
4. Normalizes discovered items with consistent metadata
5. Writes discovery manifest: `./.gh-aw/campaign.discovery.json`
6. Updates cursor in repo-memory for next run

### Discovery Manifest Schema

```json
{
  "schema_version": "v1",
  "campaign_id": "campaign-id",
  "generated_at": "2026-01-05T14:00:00Z",
  "discovery": {
    "total_items": 45,
    "cursor": {
      "last_updated_at": "2026-01-05T13:30:00Z"
    }
  },
  "summary": {
    "needs_add_count": 5,
    "needs_update_count": 10,
    "open_count": 25,
    "closed_count": 20
  },
  "items": [
    {
      "url": "https://github.com/org/repo/issues/123",
      "content_type": "issue",
      "number": 123,
      "repo": "org/repo",
      "created_at": "2026-01-01T10:00:00Z",
      "updated_at": "2026-01-05T12:00:00Z",
      "state": "open",
      "title": "Example issue"
    }
  ]
}
```

### Phase 1+: Agent Processing

The agent reads the precomputed manifest instead of performing GitHub searches:
1. Reads `./.gh-aw/campaign.discovery.json`
2. Processes normalized items deterministically
3. Makes decisions based on explicit GitHub state
4. Executes writes according to governance budgets

**Benefits:**
- **Deterministic:** Same inputs always produce same outputs
- **Efficient:** Reduces redundant GitHub API calls
- **Traceable:** Discovery logic separated from agent decisions
- **Resumable:** Cursor enables incremental processing across runs

## Compilation and orchestrators

`gh aw compile` validates campaign specs. When the spec has meaningful details (tracker label, workflows, memory paths, or a metrics glob), it also generates an orchestrator and compiles it to `.campaign.lock.yml`.

During compilation, a `.campaign.g.md` file is generated locally as a debug artifact to help developers understand the orchestrator structure, but this file is not committed to git—only the compiled `.campaign.lock.yml` is tracked.

See [Agentic campaign specs and orchestrators](/gh-aw/setup/cli/#compile).
