---
title: CLI commands
description: Command reference for campaign management
banner:
  content: '<strong>Do not use.</strong> Campaigns are still incomplete and may produce unreliable or unintended results.'
---

The GitHub Agentic Workflows CLI provides commands for listing, inspecting, validating, and managing campaigns.

:::note
Use the automated creation flow to create campaigns. These commands are for managing existing campaigns. See [Getting started](/gh-aw/guides/campaigns/getting-started/).
:::

## Campaign commands

```bash
gh aw campaign                    # List all campaigns
gh aw campaign security           # Filter by ID or name
gh aw campaign --json             # JSON output

gh aw campaign status             # Status for all campaigns
gh aw campaign status incident    # Filter status by ID or name
gh aw campaign status --json      # JSON status output

gh aw campaign new my-campaign    # Scaffold new spec (advanced)
gh aw campaign validate           # Validate all specs
gh aw campaign validate --no-strict  # Report without failing

gh aw campaign create-project     # Create GitHub Project V2
  --owner myorg                   # Owner (org or user)
  --title "My Project"            # Project title
  --org                           # Owner is an organization
  --view "name:layout[:filter]"   # Add view (repeatable)
  --field "name:type[:options]"   # Add custom field (repeatable)
```

## List campaigns

View all campaign specs in `.github/workflows/*.campaign.md`:

```bash
gh aw campaign
```

Output shows campaign ID, name, state, and file path.

### Filter by name or ID

```bash
gh aw campaign security
```

Shows campaigns containing "security" in ID or name.

### JSON output

```bash
gh aw campaign --json
```

Returns structured data for scripting and automation.

## Check campaign status

View live status from project boards:

```bash
gh aw campaign status
```

Shows active campaigns with project board statistics, progress, and health indicators.

### Filter status

```bash
gh aw campaign status incident
```

Shows status for campaigns matching "incident".

### JSON status

```bash
gh aw campaign status --json
```

Returns structured status data including metrics, KPIs, and item counts.

## Validate campaigns

Check all campaign specs for configuration errors:

```bash
gh aw campaign validate
```

Validates:
- Required fields present
- Valid YAML syntax
- Proper KPI configuration
- Discovery scope configured
- Project URLs valid
- Workflow references exist

Exit code 1 indicates validation failures.

### Non-failing validation

```bash
gh aw campaign validate --no-strict
```

Reports problems without failing. Useful for CI pipelines during development.

## Create new campaign (advanced)

:::caution
Advanced users only. Most users should use the [automated creation flow](/gh-aw/guides/campaigns/getting-started/).
:::

Scaffold a new campaign spec:

```bash
gh aw campaign new my-campaign-id
```

Creates `.github/workflows/my-campaign-id.campaign.md` with basic structure. You must:
1. Configure all required fields
2. Set up project board manually (or use `create-project` command)
3. Compile the spec with `gh aw compile`
4. Test thoroughly before running

The automated flow handles all this for you.

## Create GitHub Project

Create a GitHub Project V2 for campaign tracking:

```bash
gh aw campaign create-project --owner myorg --title "Security Q1 2025" --org
```

This command creates a project and optionally configures views and custom fields:

### Basic project creation

```bash
gh aw campaign create-project --owner myorg --title "Campaign Tracker" --org
```

### With views

```bash
gh aw campaign create-project --owner myorg --title "Campaign Board" --org \
  --view "Progress:board:is:open" \
  --view "All Items:table" \
  --view "Timeline:roadmap"
```

View format: `name:layout[:filter]`
- **name**: View name (required)
- **layout**: `board`, `table`, or `roadmap` (required)
- **filter**: Optional GitHub search filter (e.g., `is:issue is:pr`)

### With custom fields

```bash
gh aw campaign create-project --owner myorg --title "Task Tracker" --org \
  --field "Priority:SINGLE_SELECT:High,Medium,Low" \
  --field "Size:SINGLE_SELECT:Small,Medium,Large" \
  --field "Start Date:DATE" \
  --field "Campaign Id:TEXT"
```

Field format: `name:type[:options]`
- **name**: Field name (required)
- **type**: `TEXT`, `DATE`, `SINGLE_SELECT`, or `NUMBER` (required)
- **options**: Comma-separated options for `SINGLE_SELECT` fields

### Complete example

```bash
gh aw campaign create-project --owner myorg --title "Security Campaign" --org \
  --view "Board:board:is:issue is:pr" \
  --view "Timeline:roadmap" \
  --field "Priority:SINGLE_SELECT:High,Medium,Low" \
  --field "Size:SINGLE_SELECT:Small,Medium,Large" \
  --field "Start Date:DATE" \
  --field "End Date:DATE" \
  --field "Campaign Id:TEXT" \
  --field "Worker Workflow:TEXT"
```

The command outputs the project URL, which you can use in your campaign spec's `project-url` field.

## Common workflows

### Check campaign health

```bash
# Quick health check
gh aw campaign status

# Detailed inspection of specific campaign
gh aw campaign status security-audit --json | jq '.campaigns[0]'
```

### Pre-commit validation

```bash
# In CI or pre-commit hook
gh aw campaign validate --no-strict
```

### Find inactive campaigns

```bash
# List campaigns with their states
gh aw campaign --json | jq '.campaigns[] | {id, state}'
```

### Monitor campaign progress

```bash
# Watch campaign status (requires watch/jq)
watch -n 300 'gh aw campaign status my-campaign'
```

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Validation error or command failed |
| 2 | Invalid arguments |

## Further reading

- [Campaign specs](/gh-aw/guides/campaigns/specs/) - Configuration format
- [Getting started](/gh-aw/guides/campaigns/getting-started/) - Create your first campaign
- [Campaign lifecycle](/gh-aw/guides/campaigns/lifecycle/) - Execution model
