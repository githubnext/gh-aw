---
title: "Getting Started"
description: "Quick start guide for creating and launching agentic campaigns"
---

This guide is the shortest path from “we want a campaign” to a working dashboard.

> [!WARNING]
> **GitHub Agentic Workflows** is a *research demonstrator* in early development and may change significantly.
> Using [agentic workflows](/gh-aw/reference/glossary/#agentic-workflow) (AI-powered workflows that can make autonomous decisions) means giving AI [agents](/gh-aw/reference/glossary/#agent) (autonomous AI systems) the ability to make decisions and take actions in your repository. This requires careful attention to security considerations and human supervision.
> Review all outputs carefully and use time-limited trials to evaluate effectiveness for your team.

## Quick start (5 steps)

1. Create a GitHub Project board (manual, one-time) and copy its URL.
2. Add `.github/workflows/<id>.campaign.md` in a PR.
3. Run `gh aw compile`.
4. Run the generated orchestrator workflow from the Actions tab.
5. Apply the tracker label to issues/PRs you want tracked.

## 1) Create the dashboard (GitHub Project)

In GitHub: your org → **Projects** → **New project**. Start with a **Table** view, add a **Board** view grouped by `Status`, and optionally a **Roadmap** view for timelines.

Recommended custom fields (see [Project Management](/gh-aw/guides/campaigns/project-management/)):

- **Status** (Single select): Todo, In Progress, Blocked, Done
- **Worker/Workflow** (Single select): Names of your worker workflows
- **Priority** (Single select): High, Medium, Low
- **Start Date** / **End Date** (Date): For roadmap views

Copy the Project URL (e.g., `https://github.com/orgs/myorg/projects/42`).

## 2) Create the campaign spec

Create `.github/workflows/<id>.campaign.md` with frontmatter like:

```yaml
id: framework-upgrade
version: "v1"
name: "Framework Upgrade"

project-url: "https://github.com/orgs/ORG/projects/1"
tracker-label: "campaign:framework-upgrade"

objective: "Upgrade all services to Framework vNext with zero downtime."
kpis:
  - id: services_upgraded
    name: "Services upgraded"
    primary: true
    direction: "increase"
    target: 50

workflows:
  - framework-upgrade
```

You can add governance and repo-memory wiring later; start with a working loop.

## 3) Compile

Run:

```bash
gh aw compile
```

This validates the spec. When the spec has meaningful details (tracker label, workflows, memory paths, or a metrics glob), `compile` also generates an orchestrator and compiles it to `.campaign.lock.yml`.

The orchestrator workflow consists of:

1. **Discovery precomputation step**: Queries GitHub for candidate items and writes a normalized manifest
2. **Agent coordination job**: Reads the manifest and updates the project board

**Note:** A `.campaign.g.md` file is generated locally as a debug artifact to help you understand the orchestrator structure, but this file is not committed to git—only the compiled `.campaign.lock.yml` is tracked.

## 4) Run the orchestrator

Trigger the orchestrator workflow from GitHub Actions. Its job is to keep the dashboard in sync:

1. **Discovery precomputation**: Queries GitHub for items with the tracker label and writes a manifest
2. **Agent coordination**: Reads the manifest, determines what needs updating, and updates the project board
3. **Reporting**: Reports counts of items discovered, processed, and deferred

## 5) Add work items

Apply the tracker label (for example `campaign:framework-upgrade`) to issues/PRs you want tracked. The orchestrator will pick them up on the next run.

**Important: Campaign item protection**

Items with campaign labels (`campaign:*`) are automatically protected from other automated workflows:

- **Automatic exclusion**: Workflows like `issue-monster` skip issues with campaign labels
- **Controlled by orchestrator**: Only the campaign orchestrator manages campaign items
- **Manual opt-out**: Use labels like `no-bot` or `no-campaign` to exclude items from all automation

This ensures your campaign items remain under the control of the campaign orchestrator and aren't interfered with by other workflows.

## Optional: repo-memory for durable state

Enable repo-memory for campaigns using this layout: `memory/campaigns/<campaign-id>/cursor.json` and `memory/campaigns/<campaign-id>/metrics/<date>.json`. Campaign writes must include a cursor and at least one metrics snapshot.

## Automated campaign creation

For a more streamlined experience, you can use the automated campaign creation flow. Create an issue and apply the `create-agentic-campaign` label to trigger the campaign generator.

### How it works (Two-Phase Flow)

The campaign creation process uses an optimized two-phase architecture:

**Phase 1 - Campaign Generator Workflow** (~30 seconds):
1. Automatically triggered when you apply the `create-agentic-campaign` label to an issue
2. Creates a GitHub Project board for your campaign
3. Discovers relevant workflows from the local repository and the [agentics collection](https://github.com/githubnext/agentics)
4. Generates the complete campaign specification (`.github/workflows/<id>.campaign.md`)
5. Writes the campaign file to the repository
6. Updates the issue with campaign details and project board link

**Phase 2 - Compilation** (~1-2 minutes):
1. Automatically assigns a Copilot Coding Agent to compile the campaign
2. Runs `gh aw compile <campaign-id>` to generate the orchestrator
3. Creates a pull request with all campaign files:
   - `.github/workflows/<id>.campaign.md` (specification)
   - `.github/workflows/<id>.campaign.g.md` (debug artifact, not tracked in git)
   - `.github/workflows/<id>.campaign.lock.yml` (compiled workflow)

**Why two phases?** The `gh aw compile` command requires the gh-aw CLI binary, which is only available in Copilot Coding Agent sessions. GitHub Actions runners cannot compile campaigns directly.

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

The campaign generator automatically discovers and suggests workflows from three sources:

- **Agentic workflows**: AI-powered workflows (`.md` files) catalogued in `.github/workflow-catalog.yml` organized by category
- **Regular GitHub Actions workflows**: Standard automation workflows (`.yml` files, excluding `.lock.yml`) discovered dynamically by scanning `.github/workflows/` - assessed for AI enhancement potential
- **Agentics collection**: 17 reusable workflows from [githubnext/agentics](https://github.com/githubnext/agentics):
  - **Triage & Analysis**: issue-triage, ci-doctor, repo-ask, daily-accessibility-review, q-workflow-optimizer
  - **Research & Planning**: weekly-research, daily-team-status, daily-plan, plan-command
  - **Coding & Development**: daily-progress, daily-dependency-updater, update-docs, pr-fix, daily-adhoc-qa, daily-test-coverage-improver, daily-performance-improver

The generator uses a two-tier discovery approach:
1. **Static catalog** (`.github/workflow-catalog.yml`): Agentic workflows and external collections organized by category for fast lookup
2. **Dynamic scanning**: Regular `.yml` workflows (excluding `.lock.yml` compiled files) scanned at runtime to assess AI enhancement opportunities

This hybrid approach ensures:
- **Fast**: Catalog lookup for agentic workflows (<1 second vs 2-3 minutes of scanning)
- **Comprehensive**: Dynamic discovery includes all regular workflows without manual catalog maintenance
- **Flexible**: New regular workflows are automatically discovered without updating the catalog

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
