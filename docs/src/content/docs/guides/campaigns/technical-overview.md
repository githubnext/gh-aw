---
title: Agentic campaigns
description: Run structured, goal-driven initiatives with GitHub Agentic Workflows and GitHub Projects.
---

Agentic campaigns are goal-driven initiatives that coordinate AI agents to achieve specific objectives.

They provide a simple layer for managing work: define objectives with measurable KPIs, track progress on GitHub Projects, and execute workflows to drive work forward.

> [!TIP]
> **Quick start:** Use the [automated campaign creation flow](/gh-aw/guides/campaigns/getting-started/#automated-campaign-creation) to create a complete campaign in 2-3 minutes.

## When to use a campaign

Use a campaign when you need to coordinate work toward a specific goal with measurable progress tracking.

**Examples:**
- Migration projects: "Upgrade all services to Framework vNext"
- Quality improvements: "Increase test coverage to 90%"
- Security initiatives: "Resolve all high-severity vulnerabilities"

## Campaign structure

A campaign consists of:
1. **Spec file** (`.github/workflows/<id>.campaign.md`) - Defines objectives, KPIs, and configuration
2. **Orchestrator workflow** (`.campaign.lock.yml`) - Auto-generated coordinator
3. **GitHub Project** - Dashboard showing real-time progress
4. **Worker workflows** (optional) - Execute specific tasks (or created automatically)

## How it works

The orchestrator workflow runs on a schedule (default: daily) and executes phases:

**Phase 0: Workflow Execution**
- Check if configured workflows exist
- Create and test any missing workflows
- Execute workflows sequentially
- Collect outputs from workflow runs

> [!NOTE]
> This phase only runs when workflows are configured in the campaign spec.

**Phase 1: Discovery**
- Find work items (issues, PRs) created by workers
- Use tracker labels or workflow run queries
- Build a manifest of discovered items

**Phase 2: Planning**
- Determine which items need to be added to project board
- Calculate required updates based on item state

**Phase 3: Project Updates**
- Add new items to GitHub Project board
- Update status for existing items
- Apply governance limits (e.g., max 20 updates per run)

**Phase 4: Status Reporting**
- Create project status update summarizing progress
- Report KPI trends and completion percentage
- Document next steps

## Memory (optional)

Campaigns can write durable state to repo-memory (a git branch):
- **Cursor file**: `memory/campaigns/<id>/cursor.json` - Checkpoint for incremental discovery
- **Metrics snapshots**: `memory/campaigns/<id>/metrics/<date>.json` - Append-only progress tracking

This allows campaigns to resume where they left off and track progress over time.

> [!TIP]
> Repo-memory enables incremental discovery and historical metrics tracking for retrospectives and trend analysis.

## Next steps

- [Getting started](/gh-aw/guides/campaigns/getting-started/) – create a campaign quickly
- [Campaign flow](/gh-aw/guides/campaigns/flow/) – understand the complete lifecycle and execution flow
- [Campaign specs](/gh-aw/guides/campaigns/specs/) – spec fields and configuration
- [Project management](/gh-aw/guides/campaigns/project-management/) – project board setup
- [CLI commands](/gh-aw/guides/campaigns/cli-commands/) – CLI reference
