---
title: "Project Management"
description: "Use GitHub Projects with roadmap views and custom date fields for campaign tracking"
---

GitHub Projects offers powerful visualization and tracking capabilities for agentic campaigns. This guide covers view configurations, custom fields, and filtering strategies to maximize campaign visibility and control.

## Recommended Custom Fields for Campaigns

Before configuring views, set up custom fields in the GitHub Projects UI for filtering and grouping:

| Field | Type | Values | Purpose |
|-------|------|--------|---------|
| **Worker/Workflow** | Single select | Workflow names (e.g., "migration-worker") | Track which agentic workflow owns each item; enables swimlane grouping |
| **Priority** | Single select | High, Medium, Low (or P0-P3) | Filter and sort items by urgency |
| **Status** | Single select | Todo, In Progress, Blocked, Done, Closed | Track work state (default in templates) |
| **Start Date** | Date | Auto-populated from `createdAt` | Timeline visualization (required for Roadmap) |
| **End Date** | Date | Auto-populated from `closedAt` | Timeline visualization (required for Roadmap) |
| **Effort** (optional) | Single select | Small (1-3d), Medium (1w), Large (2w+) | Capacity planning |
| **Team** (optional) | Single select | Frontend, Backend, DevOps, Documentation | Team ownership |
| **Repo** (optional) | Single select | Repository names | Cross-repo campaign tracking (Note: Do not use "Repository" - it conflicts with GitHub's built-in REPOSITORY type) |

### Cross-Repository and Cross-Organization Campaigns

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

### Setting Up Custom Fields

To add custom fields: Open your project board, click the **+** button, select the field type, name it (use Title Case), add option values for single-select fields, and save. Orchestrator workflows can then populate these fields using the `fields:` parameter in `update-project` safe outputs.

## Using Project Roadmap Views with Custom Date Fields

GitHub Projects [Roadmap view](https://docs.github.com/en/issues/planning-and-tracking-with-projects/customizing-views-in-your-project/customizing-the-roadmap-layout) visualizes work items along a timeline. Create `Start Date` and `End Date` fields (type: Date), then create a Roadmap view and configure it to use these fields. Orchestrator workflows can automatically populate them.

### Automatic Timestamp Population

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

**Limitations**: Custom fields must be created manually in the GitHub UI before workflows can update them. Field names are case-sensitive. Date fields don't auto-update; orchestrators must explicitly update them.

## Roadmap View Swimlanes for Workers

Roadmap views support grouping by custom fields to create "swimlanes." Grouping by **Worker/Workflow** shows dedicated swimlanes for each agentic workflow, revealing workload distribution and bottlenecks.

**Setup**: Create a "Worker/Workflow" single-select field, then in Roadmap view select **Group by** → **Worker/Workflow**. The roadmap displays horizontal swimlanes:

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

## Task View with "Slice by" Filtering

GitHub Projects Table views support "Slice by" filtering, which shows all unique values for a field and lets you click to instantly filter items. Supports multiple fields simultaneously and updates dynamically.

**Setup**: In Table view, click the **Filter** icon or press `/`, then enable "Slice by" panels for Worker/Workflow, Priority, Status, Team, or Effort.

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

## Labeling Strategies for Campaign Organization

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

## View Configuration Examples

**Multi-Workflow Campaign**: Use Roadmap grouped by Worker/Workflow for timeline distribution, Table sliced by Priority+Status for urgent items, Board grouped by Status for progress tracking.

**Single-Workflow Campaign**: Use Table sliced by Priority sorted by Effort for prioritization, Roadmap grouped by Effort for timeline balance.

**Cross-Team Campaign**: Use Roadmap grouped by Team for cross-team coordination, Table sliced by Status (Blocked) for identifying blockers.

## Summary

Effective campaign project management combines custom fields (Worker/Workflow, Priority, Status, dates) for rich filtering, Roadmap views with swimlanes for work distribution visualization, Task views with "Slice by" for dynamic filtering, strategic labeling for campaign tracking, and multiple saved views tailored to monitoring, planning, and reporting needs.
