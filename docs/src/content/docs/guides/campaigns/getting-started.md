---
title: Getting started
description: Quick start guide for creating and launching agentic campaigns
---

This guide is the shortest path from “we want a campaign” to a working dashboard.

## Best practices

Before creating your first campaign, keep these core principles in mind:

- **Start small**: One clear goal per campaign (e.g., "Upgrade Node.js to v20")
- **Reuse workflows**: Search existing workflows before creating new ones
- **Minimal permissions**: Grant only necessary permissions (issues/draft PRs, not merges)
- **Standardized outputs**: Use consistent patterns for issues, PRs, and comments
- **Escalate when uncertain**: Create issues requesting human review for risky decisions

## Quick start (manual setup)

1. Create a campaign specification file `.github/workflows/<id>.campaign.md` in a PR
2. Run `gh aw compile`
3. Run the generated orchestrator workflow from the Actions tab

> [!NOTE]
> The campaign generator automatically creates the GitHub Project board with custom fields (Worker/Workflow, Priority, Status, Start/End Date, Effort) and three views (Campaign Roadmap, Task Tracker, Progress Board).

## 1) Create the campaign spec

Create `.github/workflows/<id>.campaign.md` with frontmatter like:

```yaml
id: framework-upgrade
version: "v1"
name: "Framework Upgrade"

# Project board URL will be generated automatically
tracker-label: "campaign:framework-upgrade"

objective: "Upgrade all services to Framework vNext with zero downtime."
kpis:
  - id: services_upgraded
    name: "Services upgraded"
    priority: primary
    direction: "increase"
    baseline: 0
    target: 50
    time-window-days: 30

workflows:
  - framework-scanner
  - framework-upgrader

# Governance (start conservative for first campaign)
governance:
  max-new-items-per-run: 10
  max-project-updates-per-run: 10
  max-comments-per-run: 5
```

**Note:** The campaign generator will automatically create a GitHub Project board with the project URL if not provided. You can also specify an existing project URL using `project-url: "https://github.com/orgs/ORG/projects/1"`.

## 2) Compile

Run:

```bash
gh aw compile
```

This validates the spec. When the spec has meaningful details (tracker label, workflows, memory paths, or a metrics glob), `compile` also generates an orchestrator and compiles it to `.campaign.lock.yml`.

The orchestrator workflow consists of:

1. **Discovery precomputation step**: Queries GitHub for candidate items and writes a normalized manifest
2. **Agent coordination job**: Reads the manifest and updates the project board

**Note:** A `.campaign.g.md` file is generated locally as a debug artifact to help you understand the orchestrator structure, but this file is not committed to git—only the compiled `.campaign.lock.yml` is tracked.

## 3) Run the orchestrator

Trigger the orchestrator workflow from GitHub Actions. Its job is to keep the dashboard in sync:

1. **Discovery precomputation**: Queries GitHub for items with the tracker label and writes a manifest
2. **Agent coordination**: Reads the manifest, determines what needs updating, and updates the project board
3. **Reporting**: Reports counts of items discovered, processed, and deferred

## Adding work items

Apply the tracker label (for example `campaign:framework-upgrade`) to issues/PRs you want tracked. The orchestrator will pick them up on the next run.

> [!IMPORTANT]
> **Campaign item protection:** Items with campaign labels (`campaign:*`) are automatically protected from other automated workflows. This ensures only the campaign orchestrator manages campaign items, preventing conflicts with workflows like `issue-monster`.

## Optional: repo-memory for durable state

Enable repo-memory for campaigns using this layout:
- `memory/campaigns/<campaign-id>/cursor.json`  
- `memory/campaigns/<campaign-id>/metrics/<date>.json`

Campaign writes must include a cursor and at least one metrics snapshot.

> [!TIP]
> Repo-memory enables incremental discovery (campaigns resume where they left off) and historical metrics tracking for retrospectives.

## Automated campaign creation

For a more streamlined experience, you can use the automated campaign creation flow. Create an issue and apply the `create-agentic-campaign` label to trigger the campaign generator.

### How it works (Two-Phase Flow)

The campaign creation process uses an optimized two-phase architecture:

**Phase 1 - Campaign Generator Workflow** (~30 seconds):
1. Automatically triggered when you apply the `create-agentic-campaign` label to an issue
2. Creates a GitHub Project board for your campaign with custom fields and views
3. Discovers relevant workflows from the local repository and the [agentics collection](https://github.com/githubnext/agentics)
4. Generates the complete campaign specification (`.github/workflows/<id>.campaign.md`)
5. Writes the campaign file to the repository
6. Updates the issue with campaign details and project board link

> [!NOTE]
> The campaign generator discovers workflows dynamically by scanning `.github/workflows/*.md` files (agentic workflows) and `.github/workflows/*.yml` files (regular GitHub Actions), plus 17 reusable workflows from the agentics collection.

**Phase 2 - Compilation** (~1-2 minutes):
1. Automatically assigns a Copilot Coding Agent to compile the campaign
2. Runs `gh aw compile <campaign-id>` to generate the orchestrator
3. Creates a pull request with all campaign files:
   - `.github/workflows/<id>.campaign.md` (specification)
   - `.github/workflows/<id>.campaign.lock.yml` (compiled workflow)

> [!NOTE]
> A `.campaign.g.md` file is generated locally as a debug artifact, but this file is not committed to git—only the compiled `.campaign.lock.yml` is tracked.

**Why two phases?** The `gh aw compile` command requires the gh-aw CLI binary, which is only available in Copilot Coding Agent sessions. GitHub Actions runners cannot compile campaigns directly.

> [!TIP]
> This two-phase architecture is 60% faster than the previous single-phase flow (2-3 minutes vs. 5-10 minutes).

### Creating a Campaign

**Option 1: Simple issue creation**
1. Go to Issues → New Issue
2. Set a descriptive title for your campaign (e.g., "Upgrade all services to Node.js 20")
3. In the issue body, describe your campaign goal, scope, and requirements
4. Apply the `create-agentic-campaign` label to the issue
5. The campaign generator will automatically trigger

**Option 2: Using issue forms (if configured)**
1. Go to Issues → New Issue → Select "🚀 Start an Agentic Campaign" template
2. Fill in the form fields
3. The issue form will automatically apply the `create-agentic-campaign` label
4. Submit the issue

### Workflow Discovery

The campaign generator automatically discovers and suggests workflows by dynamically scanning the repository:

- **Agentic workflows**: AI-powered workflows (`.md` files) discovered by scanning `.github/workflows/*.md` and parsing frontmatter to extract descriptions, triggers, and safe-outputs
- **Regular GitHub Actions workflows**: Standard automation workflows (`.yml` files, excluding `.lock.yml`) discovered by scanning `.github/workflows/*.yml` - assessed for AI enhancement potential
- **Agentics collection**: 17 reusable workflows from [githubnext/agentics](https://github.com/githubnext/agentics):
  - **Triage & Analysis**: issue-triage, ci-doctor, repo-ask, daily-accessibility-review, q-workflow-optimizer
  - **Research & Planning**: weekly-research, daily-team-status, daily-plan, plan-command
  - **Coding & Development**: daily-progress, daily-dependency-updater, update-docs, pr-fix, daily-adhoc-qa, daily-test-coverage-improver, daily-performance-improver

The generator uses fully dynamic discovery:
1. **Agentic workflows**: Scans `.github/workflows/*.md` files and parses frontmatter to understand each workflow's purpose
2. **Regular workflows**: Scans `.github/workflows/*.yml` (excluding `.lock.yml` compiled files) to assess AI enhancement opportunities
3. **External collections**: References known collections like agentics for additional workflow suggestions

This dynamic approach ensures:
- **Always up-to-date**: All workflows discovered automatically without manual catalog maintenance
- **Comprehensive**: Finds all workflow files in the repository
- **Flexible**: New workflows are discovered immediately without configuration changes
- **Accurate**: Reads actual workflow definitions rather than relying on static metadata

### What you get

After the two-phase process completes (typically 2-3 minutes total):

1. **Campaign specification file** - Complete `.campaign.md` with your objectives, KPIs, and workflow configuration
2. **GitHub Project board** - Automatic dashboard for tracking campaign progress
3. **Compiled orchestrator** - Ready-to-run `.campaign.lock.yml` workflow
4. **Pull request** - All files ready for review and merge
5. **Issue updates** - Your original issue is updated with campaign details and links

### Benefits

- **Fast**: 60% faster than the previous flow (5-10 min → 2-3 min)
- **Comprehensive**: Discovers both local and external workflows automatically
- **Transparent**: Issue updates provide real-time status throughout creation
- **Deterministic**: Workflow catalog enables consistent, fast discovery
- **Intelligent**: AI-powered workflow matching based on your campaign goals
- **Single source of truth**: All campaign design logic consolidated in one place
