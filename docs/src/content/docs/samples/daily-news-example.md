---
title: Daily News Workflow
description: An automated daily status report that summarizes repository activity and posts to GitHub Discussions using the Copilot CLI engine
sidebar:
  order: 150
---

The Daily News workflow creates an automated daily status report summarizing recent repository activity. It runs every weekday morning and posts a discussion with insights, suggestions, and a haiku.

## What It Does

The workflow:
- Analyzes recent issues, pull requests, discussions, and code changes
- Identifies failed CI runs and highlights areas needing attention
- Provides thoughtful suggestions for productivity improvements
- Reports on open source community engagement
- Posts findings as a GitHub Discussion with today's date

## Example Output

The workflow creates discussions with titles like "Daily Status - 2025-10-27" containing:
- Recent activity summary
- Productivity improvement suggestions
- Community engagement highlights
- A motivating haiku
- Detailed methodology notes (searches performed, files read, commands used)

[View example output](https://github.com/githubnext/gh-aw/discussions/categories/daily-news)

## Adding to Your Repository

Add the daily news workflow to your repository:

```bash wrap
gh aw add githubnext/agentics/daily-news --pr
```

Review and merge the pull request to activate the workflow.

## Configuration

The workflow uses:
- **Engine**: GitHub Copilot CLI (default)
- **Schedule**: Weekdays at 9 AM UTC (`cron: "0 9 * * 1-5"`)
- **Permissions**: Read-only (`permissions: read-all`)
- **Tools**: web-fetch, cache-memory, bash, edit
- **Safe Outputs**: create-discussion (limited to 1 per run)

## Customizing the Schedule

Edit `.github/workflows/daily-news.md` to change when the report runs:

```yaml
on:
  schedule:
    - cron: "0 14 * * 1-5"  # 2 PM UTC, weekdays only
```

:::tip[Avoid weekends]
The workflow uses `1-5` in the cron schedule to run Monday through Friday only, avoiding weekend noise. This is a best practice for daily team status reports.
:::

## Customizing the Content

The workflow prompt can be edited to focus on specific areas:

```markdown
# Daily News

Write an upbeat, friendly summary focusing on:
- Pull request velocity and review patterns
- Test coverage trends
- Security and dependency updates
```

After editing, recompile:

```bash wrap
gh aw compile daily-news
```

## Required Secret

The workflow uses the GitHub Copilot CLI engine, which requires a `COPILOT_CLI_TOKEN` secret:

1. Create a fine-grained Personal Access Token with "Copilot Requests" permission
2. Add to repository secrets:

```bash
gh secret set COPILOT_CLI_TOKEN -a actions --body "<your-token>"
```

See [Quick Start Guide](/gh-aw/start-here/quick-start/) for detailed setup instructions.

## Network Access

The workflow uses the Tavily MCP server for web search and enables web-fetch for gathering information beyond the repository. Network access is controlled by the firewall configuration.

## Use Cases

**Team Coordination:**
- Start daily standups with automated context
- Track velocity and identify bottlenecks
- Surface issues that need attention

**Project Management:**
- Monitor community engagement patterns
- Identify trending topics in discussions
- Track open source contribution health

**Developer Experience:**
- Celebrate recent wins and merged PRs
- Provide motivational content
- Suggest productivity improvements

## Related Workflows

- [Research & Planning Workflows](/gh-aw/samples/research-planning/) - Weekly summaries and status reports
- [Triage & Analysis Workflows](/gh-aw/samples/triage-analysis/) - Workflow analysis and investigation

## Source Code

View the [daily-news.md source](https://github.com/githubnext/gh-aw/blob/main/.github/workflows/daily-news.md) on GitHub.
