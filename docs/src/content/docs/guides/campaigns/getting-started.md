---
title: Getting started
description: Quick start guide for creating and launching agentic campaigns
---

This guide shows you how to create your first campaign using the automated creation flow.

> [!IMPORTANT]
> **Automated creation is the only supported way to create campaigns.** The automated flow handles project board creation, workflow discovery, spec generation, and compilation automatically in 2-3 minutes.

## Best practices

Before creating your first campaign, keep these core principles in mind:

- **Start small**: One clear goal per campaign (e.g., "Upgrade Node.js to v20")
- **Reuse workflows**: Search existing workflows before creating new ones
- **Minimal permissions**: Grant only necessary permissions (issues/draft PRs, not merges)
- **Standardized outputs**: Use consistent patterns for issues, PRs, and comments
- **Escalate when uncertain**: Create issues requesting human review for risky decisions

## How to create a campaign

### Step 1: Create an issue

1. Go to Issues → New Issue
2. Set a descriptive title for your campaign (e.g., "Upgrade all services to Node.js 20")
3. In the issue body, describe your campaign goal, scope, and requirements
4. Apply the `create-agentic-campaign` label to the issue

**Alternative**: If configured, use the "🚀 Start an Agentic Campaign" issue template which automatically applies the label.

### Step 2: Wait for automated creation

The campaign generator will automatically:

**Phase 1 - Campaign Generator Workflow** (~30 seconds):
1. Create a GitHub Project board with custom fields and views
2. Discover relevant workflows from your repository and the [agentics collection](https://github.com/githubnext/agentics)
3. Generate the complete campaign specification (`.github/workflows/<id>.campaign.md`)
4. Update your issue with campaign details and project board link

> [!NOTE]
> The campaign generator discovers workflows dynamically by scanning `.github/workflows/*.md` files (agentic workflows) and `.github/workflows/*.yml` files (regular GitHub Actions), plus 17 reusable workflows from the agentics collection.

**Phase 2 - Compilation** (~1-2 minutes):
1. Assign a Copilot Coding Agent to compile the campaign
2. Run `gh aw compile <campaign-id>` to generate the orchestrator
3. Create a pull request with all campaign files:
   - `.github/workflows/<id>.campaign.md` (specification)
   - `.github/workflows/<id>.campaign.lock.yml` (compiled workflow)

> [!NOTE]
> A `.campaign.g.md` file is generated locally as a debug artifact, but this file is not committed to git—only the compiled `.campaign.lock.yml` is tracked.

> [!TIP]
> This two-phase architecture is 60% faster than the previous flow (2-3 minutes vs. 5-10 minutes).

### Step 3: Review and merge

After 2-3 minutes, you'll have:

1. **Campaign specification file** - Complete `.campaign.md` with your objectives, KPIs, and workflow configuration
2. **GitHub Project board** - Automatic dashboard for tracking campaign progress
3. **Compiled orchestrator** - Ready-to-run `.campaign.lock.yml` workflow
4. **Pull request** - All files ready for review and merge
5. **Issue updates** - Your original issue is updated with campaign details and links

Review the pull request and merge it when ready.

### Step 4: Run the orchestrator

After merging, trigger the orchestrator workflow from the GitHub Actions tab. The orchestrator will:

1. **Discovery**: Find issues/PRs with the campaign tracker label
2. **Project updates**: Add new items to the project board and update existing items
3. **Status reporting**: Create project status updates with progress metrics

## Workflow discovery

The campaign generator automatically discovers and suggests workflows:

- **Agentic workflows**: AI-powered workflows (`.md` files) from `.github/workflows/*.md` with parsed frontmatter
- **Regular GitHub Actions workflows**: Standard automation (`.yml` files, excluding `.lock.yml`) assessed for AI enhancement potential
- **Agentics collection**: 17 reusable workflows from [githubnext/agentics](https://github.com/githubnext/agentics):
  - **Triage & Analysis**: issue-triage, ci-doctor, repo-ask, daily-accessibility-review, q-workflow-optimizer
  - **Research & Planning**: weekly-research, daily-team-status, daily-plan, plan-command
  - **Coding & Development**: daily-progress, daily-dependency-updater, update-docs, pr-fix, daily-adhoc-qa, daily-test-coverage-improver, daily-performance-improver

This dynamic approach ensures:
- **Always up-to-date**: All workflows discovered automatically without manual catalog maintenance
- **Comprehensive**: Finds all workflow files in the repository
- **Flexible**: New workflows are discovered immediately without configuration changes
- **Accurate**: Reads actual workflow definitions rather than relying on static metadata

## Adding work items

After your campaign is running, you can add work items by applying the campaign tracker label (e.g., `campaign:framework-upgrade`) to issues or pull requests. The orchestrator will pick them up on the next run.

> [!IMPORTANT]
> **Campaign item protection:** Items with campaign labels (`campaign:*`) are automatically protected from other automated workflows. This ensures only the campaign orchestrator manages campaign items, preventing conflicts with workflows like `issue-monster`.

## Optional: repo-memory for durable state

Campaigns can use repo-memory for durable state:
- `memory/campaigns/<campaign-id>/cursor.json` - Checkpoint for incremental discovery
- `memory/campaigns/<campaign-id>/metrics/<date>.json` - Append-only progress tracking

> [!TIP]
> Repo-memory enables incremental discovery (campaigns resume where they left off) and historical metrics tracking for retrospectives.
