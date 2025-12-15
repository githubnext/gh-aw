---
title: Daily Team Status Report
description: Automated daily status reports that gather repository activity and post upbeat team updates as discussions
---

This workflow automatically creates daily team status reports every weekday morning, gathering recent repository activity and posting engaging updates to help teams stay connected and informed.

## Use Case

- **Automated team updates**: Keep everyone informed without manual status meetings
- **Activity summaries**: Track issues, PRs, discussions, and code changes
- **Team morale**: Positive, encouraging updates that boost productivity
- **Community engagement**: Highlight contributions and suggest improvements

## Workflow Example

```aw wrap
---
timeout-minutes: 10
strict: true
on:
  schedule:
    - cron: "0 9 * * 1-5"  # 9 AM on weekdays
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
network: defaults
tools:
  github:
safe-outputs:
  create-discussion:
    expires: 3d
    category: announcements
    title-prefix: "[team-status] "
    close-older-discussions: true
---

# Daily Team Status

Create an upbeat daily status report for the team as a GitHub discussion.

## What to include

- Recent repository activity (issues, PRs, discussions, releases, code changes)
- Team productivity suggestions and improvement ideas
- Community engagement highlights
- Project investment and feature recommendations

## Style

- Be positive, encouraging, and helpful 🌟
- Use emojis moderately for engagement
- Keep it concise - adjust length based on actual activity

## Process

1. Gather recent activity from the repository
2. Create a new GitHub discussion with your findings and insights
```

## Configuration Details

### Scheduling

```yaml
on:
  schedule:
    - cron: "0 9 * * 1-5"  # Runs at 9 AM, Monday-Friday
  workflow_dispatch:        # Also allows manual triggering
```

The workflow runs automatically on weekdays at 9 AM UTC. The `workflow_dispatch` trigger allows you to run it manually for testing or on-demand updates.

### Permissions

```yaml
permissions:
  contents: read
  issues: read
  pull-requests: read
```

Read-only permissions allow the AI to gather repository data without making changes. The `create-discussion` safe-output handles creating the discussion post with appropriate permissions.

### Tools Configuration

```yaml
network: defaults
tools:
  github:
```

Enables GitHub API access through the GitHub MCP server, allowing the AI to query issues, pull requests, discussions, and other repository data.

### Safe Outputs

```yaml
safe-outputs:
  create-discussion:
    expires: 3d                      # Auto-close after 3 days
    category: announcements          # Post to announcements category
    title-prefix: "[team-status] "   # Consistent title formatting
    close-older-discussions: true    # Clean up old reports
```

The `create-discussion` safe-output creates GitHub discussions with:
- **Auto-expiration**: Reports are automatically closed after 3 days
- **Categorization**: Posts appear in the "Announcements" category
- **Title prefix**: Makes reports easy to identify
- **Cleanup**: Automatically closes older status reports to reduce clutter

## Adding This Workflow

Install from the agentics collection:

```bash
gh aw add githubnext/agentics/daily-team-status --create-pull-request
```

Or create it manually in `.github/workflows/daily-team-status.md`.

## Customization Ideas

1. **Change schedule**: Adjust the cron expression to run at different times
2. **Focus areas**: Modify the prompt to emphasize specific metrics or topics
3. **Style preferences**: Adjust tone, emoji usage, or report format
4. **Different outputs**: Use `create-issue` or `add-comment` instead of discussions
5. **Additional data**: Include security alerts, dependencies, or code quality metrics

## Prerequisites

- **GitHub Discussions** must be enabled in your repository
- **Copilot token** configured as `COPILOT_GITHUB_TOKEN` secret
- **Announcements category** exists in Discussions (or modify category in config)

## Related Examples

- [DailyOps](/gh-aw/examples/scheduled/dailyops/) - Continuous improvement through small daily changes
- [Research Planning](/gh-aw/examples/scheduled/research-planning/) - Weekly planning workflows
