---
title: "Project Management"
description: "Use GitHub Projects with roadmap views and custom date fields for campaign tracking"
---

GitHub Projects offers powerful visualization and tracking capabilities for agentic campaigns. This guide covers roadmap views and custom date fields for timeline management.

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
