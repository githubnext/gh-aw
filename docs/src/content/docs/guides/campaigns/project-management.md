---
title: "Project Management"
description: "Use GitHub Projects with roadmap views and custom date fields for campaign tracking"
---

GitHub Projects offers powerful visualization and tracking capabilities for agentic campaigns. This guide covers view configurations, custom fields, and filtering strategies to maximize campaign visibility and control.

## Recommended Custom Fields for Campaigns

Before configuring views, set up custom fields that provide valuable filtering and grouping capabilities:

### Essential Campaign Fields

**One-time manual setup** (in the GitHub Projects UI):

1. **Worker/Workflow** (Single select)
   - Values: Names of your worker workflows (e.g., "migration-worker", "daily-doc-updater", "unbloat-docs")
   - Purpose: Track which agentic workflow created or owns each item
   - Enables swimlane grouping in Roadmap view and filtering in Task view

2. **Priority** (Single select)
   - Values: High, Medium, Low, or P0, P1, P2, P3
   - Purpose: Filter and sort items by urgency
   - Enables "Slice by Priority" filtering

3. **Status** (Single select)
   - Values: Todo, In Progress, Blocked, Done, Closed
   - Purpose: Track work state across the campaign
   - Default field in most project templates

4. **Start Date** (Date)
   - Purpose: When work begins (auto-populated from issue `createdAt`)
   - Required for Roadmap timeline visualization

5. **End Date** (Date)
   - Purpose: When work completes (auto-populated from issue `closedAt`)
   - Required for Roadmap timeline visualization

6. **Effort** (Single select - optional)
   - Values: Small (1-3 days), Medium (1 week), Large (2+ weeks)
   - Purpose: Estimate work size for capacity planning

7. **Team** (Single select - optional)
   - Values: Frontend, Backend, DevOps, Documentation, etc.
   - Purpose: Track which team or area owns the work

8. **Repository** (Single select - optional, for cross-repository campaigns)
   - Values: Repository names (e.g., "gh-aw", "docs-site", "api-server")
   - Purpose: Track which repository an item belongs to
   - Enables filtering and grouping by repository in multi-repo campaigns

### Cross-Repository and Cross-Organization Campaigns

For campaigns that span multiple repositories or organizations:

1. **Use Organization-level Projects**: Create the GitHub Project at the organization level to track items across all repositories
2. **Add Repository field**: Create a "Repository" single-select field with values for each repository in scope
3. **Configure orchestrator**: Ensure the orchestrator can discover items across repositories using the campaign tracker label
4. **Filter by repository**: Use "Slice by Repository" to focus on specific repositories
5. **Group by repository**: In Roadmap views, group by Repository to see timeline per repository

**Example cross-repo configuration**:
```yaml
update-project:
  project: "https://github.com/orgs/myorg/projects/42"
  item_url: "https://github.com/myorg/repo-a/issues/123"
  fields:
    status: "In Progress"
    repository: "repo-a"  # Track which repo this item belongs to
    worker_workflow: "migration-worker"
    priority: "High"
```

**Benefits**:
- **Multi-repo visibility**: See all campaign work across repositories in one view
- **Repository-specific filtering**: Slice by repository to focus on specific codebases
- **Cross-repo coordination**: Identify dependencies between repositories
- **Workload distribution**: Balance work across repositories and teams

### Setting Up Custom Fields

To add custom fields to your project:

1. Open your campaign's Project board
2. Click the **+** button in the header row
3. Select field type (Single select, Date, Number, Text, Iteration)
4. Name the field (use Title Case for consistency)
5. For single-select fields, add option values
6. Click "Save" to create the field

Once these fields exist, orchestrator workflows can automatically populate them when adding or updating project items using the `fields:` parameter in `update-project` safe outputs.

## Using Project Roadmap Views with Custom Date Fields

GitHub Projects offers a [Roadmap view](https://docs.github.com/en/issues/planning-and-tracking-with-projects/customizing-views-in-your-project/customizing-the-roadmap-layout) that visualizes work items along a timeline. To use this view with campaigns, you need to add custom date fields to track when work items start and end.

### Setting Up Custom Date Fields

**One-time manual setup** (in the GitHub Projects UI):

1. Open your campaign's Project board
2. Click the **+** button in the header row to add a new field
3. Create a **Date** field named `Start Date`
4. Create another **Date** field named `End Date`
5. Create a **Roadmap** view from the view dropdown
6. Configure the roadmap to use your date fields

Once these fields exist, orchestrator workflows can automatically populate them when adding or updating project items.

### Automatic Timestamp Population

`update-project` automatically populates date fields from issue and pull request timestamps. If your project has `Start Date` and `End Date` fields, the system queries `createdAt` and `closedAt` timestamps, converts them to ISO format (YYYY-MM-DD), and populates the fields unless you provide explicit values.

**Example:** When adding issue #123 (created on 2025-12-15, closed on 2025-12-18) to a project board with "Start Date" and "End Date" fields:

```yaml
update-project:
  project: "https://github.com/orgs/myorg/projects/42"
  content_number: 123
  content_type: "issue"
  # No fields specified - dates will be auto-populated!
```

Result:
- `Start Date` → `2025-12-15` (from createdAt)
- `End Date` → `2025-12-18` (from closedAt)

**Override automatic timestamps:** You can still explicitly set date values if needed:

```yaml
update-project:
  project: "https://github.com/orgs/myorg/projects/42"
  content_number: 123
  content_type: "issue"
  fields:
    start_date: "2025-12-10"  # Overrides automatic timestamp
    end_date: "2025-12-20"    # Overrides automatic timestamp
```

### Orchestrator Configuration for Date Fields

Configure orchestrators to set date fields by modifying instructions or using the `fields` parameter in `update-project` outputs. Calculate end dates based on issue size and priority (e.g., small: 3 days, medium: 1 week, large: 2 weeks).

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

### Best Practices for Campaign Date Fields

Use field names like `Start Date`, `End Date`, or `Target Date`. For new issues, set start date to creation date and calculate end date from estimated effort. Keep original start dates for in-progress work, updating only end dates as needed. For completed work, update end dates to actual completion.

Roadmap views help visualize timelines chronologically, identify overlapping work, plan capacity, and track progress against planned dates.

### Example: Campaign with Roadmap Tracking

```yaml
# .github/workflows/migration-q1.campaign.md
id: migration-q1
name: "Q1 Migration Campaign"
project-url: "https://github.com/orgs/myorg/projects/15"
workflows:
  - migration-worker
tracker-label: "campaign:migration-q1"
```

The orchestrator queries issues with the tracker ID, adds them to the project board, sets status based on state, populates start/end dates from creation timestamps and size labels, and generates timeline reports.

### Limitations

Custom fields must be created manually in the GitHub UI before workflows can update them. Field names are case-sensitive and dates must use ISO 8601 format (YYYY-MM-DD). Date fields don't auto-update; orchestrators must explicitly update them. Roadmap views require manual configuration.

## Roadmap View Swimlanes for Workers

GitHub Projects Roadmap views support grouping by custom fields, creating visual "swimlanes" that separate work by category. For campaigns, grouping by **Worker/Workflow** creates dedicated swimlanes for each agentic workflow, making it easy to see workload distribution and identify bottlenecks.

### Setting Up Workflow Swimlanes

**Prerequisites**: Create a "Worker/Workflow" single-select field with values matching your campaign's workflow names.

**Configuration**:

1. Open your campaign's Project board
2. Select or create a **Roadmap** view
3. Click the view options menu (three dots)
4. Select **Group by** → **Worker/Workflow**
5. Configure date fields: **Start Date** and **End Date**

**Result**: The roadmap displays horizontal swimlanes, one per workflow:

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

### Benefits of Workflow Swimlanes

- **Visual workload distribution**: Instantly see which workflows are handling the most work
- **Bottleneck identification**: Spot workflows with overlapping or stalled items
- **Capacity planning**: Balance work across workflows to avoid overloading any single worker
- **Timeline coordination**: Ensure workflows don't create conflicting or dependent items at the same time
- **Progress tracking**: Monitor each workflow's contribution to campaign objectives

### Populating Worker/Workflow Fields

Orchestrator workflows should automatically set the Worker/Workflow field when adding items to the project board:

```yaml
update-project:
  project: "https://github.com/orgs/myorg/projects/42"
  item_url: "https://github.com/myorg/myrepo/issues/123"
  fields:
    status: "In Progress"
    worker_workflow: "migration-worker"  # Identifies the creating workflow
    priority: "High"
```

**Best practice**: Use the workflow's ID (from `.github/workflows/<id>.md`) as the Worker/Workflow field value for consistency.

**Important**: Worker workflows themselves remain **campaign-agnostic** and do not need to know about custom fields or campaigns. The orchestrator discovers which worker created an item (via tracker-id in issue body) and populates the Worker/Workflow field accordingly. Workers continue to execute their tasks independently, and all campaign coordination happens in the orchestrator.

### Alternative Groupings

Roadmap views can group by any single-select field. Other useful groupings for campaigns:

- **Priority**: Swimlanes for High/Medium/Low priority work
- **Team**: Swimlanes for different teams (Frontend, Backend, DevOps)
- **Status**: Swimlanes for Todo, In Progress, Blocked, Done
- **Effort**: Swimlanes for Small, Medium, Large work items
- **Repository**: Swimlanes per repository (for cross-repo campaigns)

Experiment with different groupings to find the visualization that best supports your campaign's needs.

## Task View with "Slice by" Filtering

GitHub Projects Table views (also called Task views) support powerful filtering through the "Slice by" feature. This allows you to segment your campaign board by any field, creating dynamic filtered views without manually managing labels or searches.

### What is "Slice by"?

"Slice by" is GitHub Projects' filtering mechanism that:
- Shows all unique values for a selected field (e.g., all Priority values: High, Medium, Low)
- Lets you click a value to instantly filter the view to show only items with that value
- Supports filtering by multiple fields simultaneously
- Updates dynamically as items and fields change

### Setting Up "Slice by" for Campaigns

**Configuration**:

1. Open your campaign's Project board
2. Select or create a **Table** (or **Board**) view
3. In the view toolbar, click the **Filter** icon or press `/` 
4. Enable "Slice by" panels for relevant fields:
   - **Worker/Workflow**: Filter by which workflow created the item
   - **Priority**: Filter by urgency level
   - **Status**: Filter by work state
   - **Team**: Filter by owning team
   - **Effort**: Filter by estimated size

**Example view with multiple slices**:

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

Clicking "migration-worker" and "High" filters the table to show only high-priority items from the migration-worker workflow.

### Benefits of "Slice by" for Campaigns

- **Focused views**: Filter to specific workflows, priorities, or teams without creating separate saved views
- **Quick comparisons**: Compare workload across workflows or priority distribution
- **Drill-down investigation**: Start with all items, then slice by workflow → priority → status to identify specific issues
- **Dynamic updates**: Filters automatically reflect new items and field changes
- **No label management**: Avoid creating and maintaining dozens of labels for different filter combinations

### Recommended Slice Configurations

**For orchestrator workflows** (monitoring campaign health):
- Slice by Worker/Workflow + Status to see each workflow's progress
- Slice by Priority + Status to identify high-priority blocked items

**For capacity planning**:
- Slice by Worker/Workflow + Effort to see workload distribution
- Slice by Team + Status to track team-specific progress

**For risk management**:
- Slice by Status (show only "Blocked")
- Slice by Priority (show only "High") + Status (show "In Progress" or "Blocked")

### Populating Fields for Effective Slicing

To maximize the value of "Slice by", orchestrators should consistently populate custom fields:

```yaml
update-project:
  project: "https://github.com/orgs/myorg/projects/42"
  item_url: "https://github.com/myorg/myrepo/issues/123"
  fields:
    worker_workflow: "migration-worker"  # Enables slicing by workflow
    priority: "High"                      # Enables slicing by priority
    status: "In Progress"                 # Enables slicing by status
    team: "Backend"                       # Enables slicing by team
    effort: "Medium"                      # Enables slicing by effort size
```

**Best practice**: Establish field naming conventions and valid values in your campaign spec documentation so all workflows populate fields consistently.

## Labeling Strategies for Campaign Organization

While custom fields provide rich filtering and grouping capabilities, labels remain valuable for campaign organization, especially for cross-project queries and GitHub-wide searches.

### Recommended Label Strategy

**Campaign tracker label** (required):
- Format: `campaign:<campaign-id>`
- Example: `campaign:migration-q1`, `campaign:docs-quality-maintenance-project67`
- Purpose: Primary identifier for all campaign items; enables orchestrator discovery

**Worker/workflow labels** (optional, use custom fields instead):
- Format: `workflow:<workflow-id>`
- Example: `workflow:migration-worker`, `workflow:daily-doc-updater`
- Purpose: Alternative to Worker/Workflow custom field if you prefer labels
- Note: Custom fields provide better filtering; use labels only if needed for GitHub-wide searches

**Priority labels** (optional, use custom fields instead):
- Format: `priority:high`, `priority:medium`, `priority:low`
- Purpose: Alternative to Priority custom field
- Note: Custom fields are preferred for project board filtering

**Work type labels** (recommended):
- Format: `type:bug`, `type:feature`, `type:refactor`, `type:documentation`
- Purpose: Categorize the nature of work beyond workflow ownership
- Benefit: Helps identify patterns (e.g., "most items from workflow X are bugs")

### Label vs Custom Field Decision Guide

| Use Case | Recommended Approach | Reason |
|----------|---------------------|---------|
| Campaign identification | Label (`campaign:<id>`) | Required for orchestrator discovery; works across GitHub |
| Workflow/worker tracking | Custom field | Better filtering in project views; supports swimlanes |
| Priority management | Custom field | Native support for sorting and "Slice by" |
| Status tracking | Custom field | Native project board integration |
| Work type (bug/feature) | Label | Helps with repository-wide queries |
| Team ownership | Custom field | Better for capacity planning and swimlanes |
| Effort estimation | Custom field | Enables roadmap timeline calculations |

**Best practice**: Use the campaign tracker label as the primary identifier, then rely on custom fields for project board organization. Add labels only when you need repository-wide or cross-project queries.

### Applying Labels in Workflows

Workers can apply labels when creating issues or pull requests using the `labels:` parameter in safe outputs:

```yaml
safe-outputs:
  create-issue:
    labels:
      - "campaign:migration-q1"        # Campaign tracker (required)
      - "type:refactor"                 # Work type
      - "automated"                     # Indicates AI-generated
```

Orchestrators typically don't need to add labels since they work with existing items that already have the campaign tracker label applied by workers.

## View Configuration Examples

### Example 1: Multi-Workflow Documentation Campaign

**Campaign**: Documentation quality maintenance with 6 worker workflows

**Recommended views**:

1. **Roadmap: By Workflow** (swimlanes)
   - Group by: Worker/Workflow
   - Shows: Timeline with one swimlane per workflow
   - Use: Identify workload distribution and timeline conflicts

2. **Table: High Priority Items**
   - Slice by: Priority (show only "High")
   - Slice by: Status (show "Todo" and "In Progress")
   - Use: Focus on urgent work requiring immediate attention

3. **Board: Status Kanban**
   - Group by: Status
   - Columns: Todo, In Progress, Blocked, Done
   - Use: Track overall campaign progress

4. **Table: All Items**
   - No filters or grouping
   - Use: Comprehensive view for orchestrator reporting

### Example 2: Single-Workflow Refactoring Campaign

**Campaign**: Go file size reduction with 1 worker workflow

**Recommended views**:

1. **Table: By Priority**
   - Slice by: Priority
   - Sort by: Effort (descending)
   - Use: Prioritize large, high-priority files first

2. **Roadmap: Timeline**
   - Group by: Effort (Small, Medium, Large)
   - Shows: Work distribution over time by size
   - Use: Balance small quick wins with large refactors

3. **Board: Status Tracking**
   - Group by: Status
   - Use: Simple kanban for single-workflow progress

### Example 3: Cross-Team Migration Campaign

**Campaign**: Service migration with multiple teams involved

**Recommended views**:

1. **Roadmap: By Team** (swimlanes)
   - Group by: Team
   - Shows: Timeline with one swimlane per team
   - Use: Coordinate cross-team dependencies

2. **Table: Blocked Items**
   - Slice by: Status (show only "Blocked")
   - Group by: Worker/Workflow
   - Use: Identify blockers and their sources

3. **Table: Team Workload**
   - Slice by: Team
   - Slice by: Effort
   - Use: Balance work across teams

## Summary

Effective campaign project management combines:

1. **Custom fields** for rich filtering and grouping (Worker/Workflow, Priority, Status, dates)
2. **Roadmap views with swimlanes** for visualizing work distribution across workflows
3. **Task views with "Slice by"** for dynamic filtering without label proliferation
4. **Strategic labeling** for campaign tracking and work type categorization
5. **Multiple saved views** tailored to different use cases (monitoring, planning, reporting)

By leveraging GitHub Projects' advanced features, campaigns gain powerful visibility into agentic workflow coordination, workload distribution, and progress tracking—enabling effective delegation and steering of AI-driven initiatives.
