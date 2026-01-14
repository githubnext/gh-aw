---
title: Project management
description: Use GitHub Projects with roadmap views and custom date fields for campaign tracking
---

GitHub Projects offers powerful visualization and tracking capabilities for agentic campaigns. This guide covers view configurations, custom fields, and filtering strategies to maximize campaign visibility and control.

## Recommended custom fields for campaigns

The campaign generator automatically creates these custom fields when you create a new campaign. No manual setup is required.

| Field | Type | Values | Purpose |
|-------|------|--------|---------|
| **Worker/Workflow** | Single select | Workflow names (e.g., "migration-worker") | Track which agentic workflow owns each item; enables swimlane grouping |
| **Priority** | Single select | High, Medium, Low | Filter and sort items by urgency |
| **Status** | Single select | Todo, In Progress, Blocked, Done, Closed | Track work state (default in templates) |
| **Start Date** | Date | Auto-populated from `createdAt` | Timeline visualization (required for Roadmap) |
| **End Date** | Date | Auto-populated from `closedAt` | Timeline visualization (required for Roadmap) |
| **Effort** | Single select | Small (1-3 days), Medium (1 week), Large (2+ weeks) | Capacity planning |

**Optional fields** (add manually if needed):
- **Team** (Single select): Frontend, Backend, DevOps, Documentation - Team ownership
- **Repo** (Single select): Repository names - Cross-repo campaign tracking (Note: Do not use "Repository" - it conflicts with GitHub's built-in REPOSITORY type)

### Cross-repository and cross-organization campaigns

For campaigns spanning multiple repositories:

1. Create the GitHub Project at the organization level
2. Add a "Repo" single-select field with repository names (do not use "Repository" as the field name)
3. Configure the orchestrator to discover items using the campaign tracker label
4. Use "Slice by Repo" or group by Repo in Roadmap views

Example orchestrator configuration:
```yaml
update-project:
  project: "https://github.com/orgs/myorg/projects/42"
  item_url: "https://github.com/myorg/repo-a/issues/123"
  fields:
    status: "In Progress"
    repo: "repo-a"
    worker_workflow: "migration-worker"
    priority: "High"
```

### Adding additional custom fields

The campaign generator creates standard fields automatically. To add additional custom fields: Open your project board, click the **+** button, select the field type, name it (use Title Case), add option values for single-select fields, and save. Orchestrator workflows can then populate these fields using the `fields:` parameter in `update-project` safe outputs.

## Using project roadmap views with custom date fields

GitHub Projects [Roadmap view](https://docs.github.com/en/issues/planning-and-tracking-with-projects/customizing-views-in-your-project/customizing-the-roadmap-layout) visualizes work items along a timeline. The campaign generator automatically creates `Start Date` and `End Date` fields and a Roadmap view. Orchestrator workflows automatically populate these date fields.

### Automatic timestamp population

`update-project` automatically populates `Start Date` from issue `createdAt` and `End Date` from `closedAt` (ISO format: YYYY-MM-DD). Override by explicitly setting date values in the `fields:` parameter. Orchestrators can calculate end dates based on issue size and priority (e.g., small: 3 days, medium: 1 week, large: 2 weeks).

```yaml
update-project:
  project: "https://github.com/orgs/myorg/projects/42"
  item_url: "https://github.com/myorg/myrepo/issues/123"
  fields:
    status: "In Progress"
    priority: "High"
    start_date: "2025-12-19"
    end_date: "2025-12-26"
```

**Limitations**: Date fields don't auto-update; orchestrators must explicitly update them. Additional custom fields beyond the standard set must be created manually in the GitHub UI before workflows can update them. Field names are case-sensitive.

## Roadmap view swimlanes for workers

Roadmap views support grouping by custom fields to create "swimlanes." Grouping by **Worker/Workflow** shows dedicated swimlanes for each agentic workflow, revealing workload distribution and bottlenecks.

**Setup**: The campaign generator creates the "Worker/Workflow" field and Roadmap view automatically. In the Roadmap view, select **Group by** → **Worker/Workflow**. The roadmap displays horizontal swimlanes:

```
┌─────────────────────────────────────────────────────┐
│ migration-worker                                     │
│ [Issue #123]──────[Issue #124]─────[Issue #125]    │
├─────────────────────────────────────────────────────┤
│ daily-doc-updater                                    │
│ [Issue #126]───────[Issue #127]                     │
├─────────────────────────────────────────────────────┤
│ unbloat-docs                                         │
│ [Issue #128]──[Issue #129]─────[Issue #130]        │
└─────────────────────────────────────────────────────┘
    Jan 2026     Feb 2026     Mar 2026     Apr 2026
```

Swimlanes provide visual workload distribution, bottleneck identification, capacity planning, timeline coordination, and progress tracking. Orchestrators automatically populate the Worker/Workflow field using the workflow ID:

```yaml
update-project:
  project: "https://github.com/orgs/myorg/projects/42"
  item_url: "https://github.com/myorg/myrepo/issues/123"
  fields:
    status: "In Progress"
    worker_workflow: "migration-worker"
    priority: "High"
```

Worker workflows remain campaign-agnostic; orchestrators handle all campaign coordination. Roadmap views can also group by Priority, Team, Status, Effort, or Repository.

## Task view with "Slice by" filtering

GitHub Projects Table views support "Slice by" filtering, which shows all unique values for a field and lets you click to instantly filter items. Supports multiple fields simultaneously and updates dynamically.

**Setup**: The campaign generator creates a "Task Tracker" table view automatically. In the view, click the **Filter** icon or press `/`, then enable "Slice by" panels for Worker/Workflow, Priority, Status, or Effort.

```
┌────────────────────────────────────────────────┐
│ Slice by Worker/Workflow                       │
│ ☑ migration-worker (15)                        │
│ ☑ daily-doc-updater (8)                        │
│ ☐ unbloat-docs (12)                            │
├────────────────────────────────────────────────┤
│ Slice by Priority                              │
│ ☑ High (5)                                     │
│ ☐ Medium (18)                                  │
│ ☐ Low (12)                                     │
└────────────────────────────────────────────────┘
```

Benefits include focused views without creating separate saved views, quick comparisons, drill-down investigation, dynamic updates, and no label management overhead. Orchestrators should consistently populate fields for effective slicing:

```yaml
update-project:
  project: "https://github.com/orgs/myorg/projects/42"
  item_url: "https://github.com/myorg/myrepo/issues/123"
  fields:
    worker_workflow: "migration-worker"
    priority: "High"
    status: "In Progress"
    team: "Backend"
    effort: "Medium"
```

## Labeling strategies for campaign organization

Labels remain valuable for cross-project queries and GitHub-wide searches.

**Recommended labels**:
- **Campaign tracker** (required): `campaign:<campaign-id>` for orchestrator discovery
- **Work type** (recommended): `type:bug`, `type:feature`, `type:refactor` to categorize work nature

| Use Case | Recommended Approach |
|----------|---------------------|
| Campaign identification | Label (`campaign:<id>`) |
| Workflow/worker tracking | Custom field |
| Priority/Status/Team/Effort | Custom field |
| Work type (bug/feature) | Label |

Workers apply labels when creating issues:

```yaml
safe-outputs:
  create-issue:
    labels:
      - "campaign:migration-q1"
      - "type:refactor"
```

## View configuration examples

The campaign generator creates three views automatically:
1. **Campaign Roadmap** (Roadmap view) - Timeline visualization
2. **Task Tracker** (Table view) - Detailed tracking with filtering
3. **Progress Board** (Board view) - Kanban-style progress tracking

### Creating views programmatically

Views can be created programmatically using the `update-project` safe output with `operation: "create_view"`. This is useful for custom campaign setups or automating project board configuration.

**Example: Create a roadmap view**
```yaml
safe-outputs:
  update-project:
    github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
```

```javascript
// In agent workflow
update_project({
  project: "https://github.com/orgs/myorg/projects/42",
  operation: "create_view",
  view: {
    name: "Q1 Sprint Timeline",
    layout: "roadmap",
    filter: "is:issue is:open"
  }
});
```

**View layouts:**
- `table` — List view with customizable columns
- `board` — Kanban-style cards grouped by field
- `roadmap` — Timeline with date-based swimlanes

**Common filters:**
- `is:issue,is:pull_request` — Show both issues and PRs
- `label:campaign:migration-q1` — Filter by campaign label
- `status:"In Progress"` — Show items with specific status

See [Safe Outputs Reference](/gh-aw/reference/safe-outputs/#creating-project-views) for complete documentation on view creation parameters and examples.

**Customization tips:**

**Multi-Workflow Campaign**: Use Roadmap grouped by Worker/Workflow for timeline distribution, Task Tracker sliced by Priority+Status for urgent items, Progress Board grouped by Status for progress tracking.

**Single-Workflow Campaign**: Use Task Tracker sliced by Priority sorted by Effort for prioritization, Campaign Roadmap grouped by Effort for timeline balance.

**Cross-Team Campaign** (with optional Team field): Use Roadmap grouped by Team for cross-team coordination, Task Tracker sliced by Status (Blocked) for identifying blockers.

## Project status updates

Campaign orchestrators automatically create project status updates with every run, providing stakeholders with real-time campaign progress summaries. Status updates appear in the project's Updates tab and provide a historical record of campaign execution.

### Automatic status update creation

The orchestrator creates one status update per run containing:

- **Campaign Summary**: Tasks completed, in progress, and blocked
- **Key Findings**: Important discoveries from the current run
- **Trends & Velocity**: Progress metrics and completion rates
- **Next Steps**: Remaining work and action items
- **Status Indicator**: Current campaign health (ON_TRACK, AT_RISK, OFF_TRACK, COMPLETE)

### Status update fields

| Field | Type | Description |
|-------|------|-------------|
| **project** | URL | GitHub project URL (automatically set by orchestrator) |
| **body** | Markdown | Campaign summary with findings, trends, and next steps |
| **status** | Enum | Current health: `ON_TRACK`, `AT_RISK`, `OFF_TRACK`, `COMPLETE` |
| **start_date** | Date | Run start date (YYYY-MM-DD format) |
| **target_date** | Date | Projected completion or next milestone date |

### Example status update

```yaml
create-project-status-update:
  project: "https://github.com/orgs/myorg/projects/73"
  status: "ON_TRACK"
  start_date: "2026-01-06"
  target_date: "2026-01-31"
  body: |
    ## Campaign Run Summary

    **Discovered:** 25 items (15 issues, 10 PRs)
    **Processed:** 10 items added to project, 5 updated
    **Completion:** 60% (30/50 total tasks)

    ### Key Findings
    - Documentation coverage improved to 88%
    - 3 critical accessibility issues identified
    - Worker velocity: 1.2 items/day

    ### Trends
    - Velocity stable at 8-10 items/week
    - Blocked items decreased from 5 to 2
    - On track for end-of-month completion

    ### Next Steps
    - Continue processing remaining 15 items
    - Address 2 blocked items in next run
    - Target 95% documentation coverage by end of month
```

### Status indicators

Choose appropriate status based on campaign progress:

- **ON_TRACK**: Campaign is progressing as planned, meeting velocity targets
- **AT_RISK**: Potential issues identified (blocked items, slower velocity, dependencies)
- **OFF_TRACK**: Campaign behind schedule, requires intervention or re-planning
- **COMPLETE**: All campaign objectives met, no further work needed

### Viewing status updates

Status updates appear in:
1. **Project Updates Tab**: Click the "Updates" tab in your project to see all status updates
2. **Project Overview**: Recent status update displayed on project home page
3. **Timeline**: Status updates shown chronologically with other project activity

### Configuration

Campaign orchestrators automatically configure status update creation:

```yaml
safe-outputs:
  create-project-status-update:
    max: 1  # One status update per orchestrator run
    github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
```

The orchestrator uses the same GitHub token configured for `update-project` operations. This token must have Projects: Read+Write permissions.

## Summary

Effective campaign project management combines custom fields (Worker/Workflow, Priority, Status, dates) for rich filtering, Roadmap views with swimlanes for work distribution visualization, Task views with "Slice by" for dynamic filtering, automatic status updates for stakeholder communication, strategic labeling for campaign tracking, and multiple saved views tailored to monitoring, planning, and reporting needs.
