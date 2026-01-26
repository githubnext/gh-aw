---
title: Creating Campaigns
description: How to create agentic campaigns using issue-based workflow or CLI commands
banner:
  content: '<strong>Do not use.</strong> Campaigns are still incomplete and may produce unreliable or unintended results.'
---

There are two ways to create a campaign:

## Recommended: Issue-based creation (with custom agent)

The easiest way is to use the automated campaign creation workflow:

1. **Create an issue** describing your campaign goal
2. **Apply the label** `create-agentic-campaign`
3. **Wait for the custom agent** to generate:
   - GitHub Project board with required fields and views
   - Campaign spec file (`.campaign.md`)
   - Pull request with the campaign configuration
4. **Review and merge** the PR to activate the campaign

The custom agent (campaign designer) analyzes your issue description, discovers relevant workflows, and generates a complete campaign configuration ready for review.

See [Getting started](/gh-aw/guides/campaigns/getting-started/) for a detailed walkthrough.

## Advanced: CLI-based creation

For advanced users who prefer manual control:

```bash
# Create campaign spec and GitHub Project
gh aw campaign new my-campaign-id --project --owner @me

# Or for organizations
gh aw campaign new my-campaign-id --project --owner myorg
```

This scaffolds the campaign spec and creates a Project board, but you'll need to manually configure all fields, add worker workflows, and test thoroughly.

See [CLI commands](/gh-aw/guides/campaigns/cli-commands/) for complete command reference.

## Example: Security Alert Campaign

Here's what a campaign spec looks like after creation:

**Issue description** you provide:

> Burn down all open code security alerts, prioritizing file-write alerts first
> and batching up to 3 related alerts/PR with a brief fix rationale comment.

**Generated [campaign spec](/gh-aw/guides/campaigns/specs/)**:

```yaml
---
id: security-alert-burndown
name: "Security Alert Burndown"
description: "Drive the code security alerts backlog to zero"

# GitHub Project for tracking
project-url: "https://github.com/orgs/ORG/projects/1"
tracker-label: "campaign:security-alert-burndown"

# Strategic goals
objective: "Reduce open code security alerts to zero without breaking CI."
kpis:
  - id: open_alerts
    name: "Open alerts"
    priority: primary
    direction: "decrease"
    target: 0

# Worker workflows to dispatch
workflows:
  - security-alert-fix

# Governance and pacing
governance:
  max-project-updates-per-run: 10
  max-comments-per-run: 10
---
```

The spec compiles into a campaign orchestrator workflow (`.campaign.lock.yml`) that GitHub Actions executes on schedule. The orchestrator [dispatches workers, tracks outputs, updates the Project board, and reports progress](/gh-aw/guides/campaigns/lifecycle/).

## Next steps

- [Getting started](/gh-aw/guides/campaigns/getting-started/) – step-by-step tutorial
- [Campaign specs](/gh-aw/guides/campaigns/specs/) – configuration reference
- [Campaign Lifecycle](/gh-aw/guides/campaigns/lifecycle/) – execution model
- [CLI commands](/gh-aw/guides/campaigns/cli-commands/) – command reference
