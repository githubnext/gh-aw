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

**NEW**: Starting in this release, `update-project` automatically populates date fields from issue and pull request timestamps:

- **Start Date** is automatically set from `createdAt` (when the issue/PR was created)
- **End Date** is automatically set from `closedAt` (when the issue/PR was closed, if applicable)

This happens automatically for campaign project boards and requires no additional configuration. The system will:

1. Query the issue or PR to fetch `createdAt` and `closedAt` timestamps
2. Convert timestamps to ISO date format (YYYY-MM-DD)
3. Populate `Start Date` and `End Date` fields if they exist in the project and aren't already set
4. Respect any manually provided date values in the `fields` parameter

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

To have orchestrators set date fields automatically, modify the orchestrator's instructions or use the `fields` parameter in `update-project` outputs.

**Example workflow instruction:**

```markdown
When adding issues to the project board, set these custom fields:
- `Start Date`: Set to the issue's creation date
- `End Date`: Set to estimated completion date based on issue size and priority
  - Small issues: 3 days from start
  - Medium issues: 1 week from start
  - Large issues: 2 weeks from start
```

**Example agent output for update-project:**

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

**Recommended field names:**
- `Start Date` or `start_date` - When work begins
- `End Date` or `end_date` - Expected or actual completion date
- `Target Date` - Optional milestone or deadline

**Date assignment strategies:**

- **For new issues**: Set `start_date` to current date, calculate `end_date` based on estimated effort
- **For in-progress work**: Keep original `start_date`, adjust `end_date` if needed
- **For completed work**: Update `end_date` to actual completion date

**Roadmap view benefits:**

- **Visual timeline**: See all campaign work laid out chronologically
- **Dependency identification**: Spot overlapping or sequential work items
- **Capacity planning**: Identify periods with too much concurrent work
- **Progress tracking**: Compare planned vs actual completion dates

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

The orchestrator can set date fields when adding issues:

```markdown
## Campaign Orchestrator

When adding discovered issues to the project board:

1. Query issues with tracker-id: "migration-worker"
2. For each issue:
   - Add to project board
   - Set `status` to "Todo" (or "Done" if closed)
   - Set `start_date` to the issue creation date
   - Set `end_date` based on labels:
     - `size:small` → 3 days from start
     - `size:medium` → 1 week from start  
     - `size:large` → 2 weeks from start
   - Set `priority` based on issue labels

Generate a report showing timeline distribution of all work items.
```

### Limitations and Considerations

- **Manual field creation**: Workflows cannot create custom fields; they must exist before workflows can update them
- **Field name matching**: Custom field names are case-sensitive; use exact names as defined in the project
- **Date format**: Use ISO 8601 format (YYYY-MM-DD) for date values
- **No automatic recalculation**: Date fields don't auto-update; orchestrators must explicitly update them
- **View configuration**: Roadmap views must be configured manually in the GitHub UI
