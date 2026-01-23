---
title: Getting started
description: Quick start guide for creating and launching agentic campaigns
banner:
  content: '<strong>Do not use.</strong> Campaigns are still incomplete and may produce unreliable or unintended results.'
---

Create your first campaign using the automated creation flow. The flow generates a Project board, campaign spec, and orchestrator workflow based on an issue description.

## Prerequisites

- Repository with GitHub Agentic Workflows installed
- `create-agentic-campaign` label configured in your repository
- Write access to create issues and merge pull requests
- GitHub Actions enabled

## Create a campaign

1. **Create an issue** describing your campaign goal and scope
2. **Apply the label** `create-agentic-campaign` to the issue
3. **Wait for automation** - A pull request appears within a few minutes
4. **Review the PR** - Verify the generated Project, spec, and orchestrator
5. **Merge the PR** when ready
6. **Run the orchestrator** from the Actions tab to start the campaign

## Generated files

The pull request creates three components:

**Project board** - GitHub Project for tracking campaign progress with custom fields and views.

**Campaign spec** - Configuration file at `.github/workflows/<id>.campaign.md` defining goals, workers, and governance.

**Orchestrator workflow** - Compiled workflow at `.github/workflows/<id>.campaign.lock.yml` that executes the campaign logic.

## Campaign execution

The orchestrator runs on the configured schedule (daily by default):

1. Dispatches worker workflows via `workflow_dispatch` (if configured)
2. Discovers issues and pull requests created by workers
3. Updates the Project board with new items
4. Posts a status update summarizing progress

See [Campaign lifecycle](/gh-aw/guides/campaigns/lifecycle/) for execution details.

## Best practices

Start with focused scope:
- Define one clear objective
- Include 1-3 worker workflows maximum
- Set conservative governance limits (e.g., 10 project updates per run)

Configure worker triggers:
- Workers should accept `workflow_dispatch` only
- Remove cron schedules, push, and pull_request triggers
- Let the orchestrator control execution timing

See [Campaign specs](/gh-aw/guides/campaigns/specs/) for configuration options.
