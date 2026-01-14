---
title: Getting started
description: Quick start guide for creating and launching agentic campaigns
---

This guide is the shortest path from “we want a campaign” to a working dashboard.

## Best practices

Before creating your first campaign, keep these core principles in mind:

- **Start small**: One clear goal per campaign (e.g., "Upgrade Node.js to v20")
- **Start passive**: Use passive mode first to observe behavior and build trust—this is essentially ProjectOps with campaign tracking
- **Understand the progression**: ProjectOps (simple) → Passive Campaign (structured tracking) → Active Campaign (autonomous execution)
- **Reuse workflows**: Search existing workflows before creating new ones
- **Minimal permissions**: Grant only necessary permissions (issues/draft PRs, not merges)
- **Standardized outputs**: Use consistent patterns for issues, PRs, and comments
- **Escalate when uncertain**: Create issues requesting human review for risky decisions

## Quick start (3 steps)

1. Create a campaign specification file `.github/workflows/<id>.campaign.md` in a PR.
2. Run `gh aw compile`.
3. Run the generated orchestrator workflow from the Actions tab.

The campaign generator automatically creates:
- GitHub Project board with custom fields (Worker/Workflow, Priority, Status, Start/End Date, Effort)
- Three views (Campaign Roadmap, Task Tracker, Progress Board)
- Campaign orchestrator workflow

## 1) Create the campaign spec

Create `.github/workflows/<id>.campaign.md` with frontmatter like:

**For your first campaign** (passive mode - recommended):

This creates a **passive campaign**, which is essentially [ProjectOps](/gh-aw/examples/issue-pr-events/projectops/) with campaign structure. The orchestrator tracks work created by independent workflows on a project board and reports progress toward objectives.

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
  - framework-upgrade  # Use an existing workflow (runs independently)

# Governance (conservative defaults for first campaign)
governance:
  max-new-items-per-run: 5
  max-project-updates-per-run: 5
  max-comments-per-run: 3
```

**Note:** The campaign generator will automatically create a GitHub Project board with the project URL if not provided. You can also specify an existing project URL using `project-url: "https://github.com/orgs/ORG/projects/1"`.

**For experienced users** (active mode - advanced):

This creates an **active campaign** with true orchestration. The campaign actively executes workflows, creates missing ones, and drives progress autonomously.

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
    priority: primary
    direction: "increase"
    baseline: 0
    target: 50
    time-window-days: 30

workflows:
  - framework-scanner
  - framework-upgrader

# Enable active execution (ADVANCED - only after passive campaign experience)
execute-workflows: true

# Governance (still start conservative even in active mode)
governance:
  max-new-items-per-run: 10
  max-project-updates-per-run: 10
  max-comments-per-run: 5
```

**Key differences:**
- **Passive mode** (default): ProjectOps-based tracking—discovers and tracks work created by existing workflows (safer, simpler)
- **Active mode** (`execute-workflows: true`): True campaign orchestration—executes workflows, creates missing ones (powerful but complex)

**Recommendation:** Start with passive mode to understand campaign patterns. This gives you ProjectOps benefits (project board updates, tracking) with campaign structure (objectives, KPIs, governance) before moving to autonomous execution.

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

**Important: Campaign item protection**

Items with campaign labels (`campaign:*`) are automatically protected from other automated workflows:

- **Automatic exclusion**: Workflows like `issue-monster` skip issues with campaign labels
- **Controlled by orchestrator**: Only the campaign orchestrator manages campaign items
- **Manual opt-out**: Use labels like `no-bot` or `no-campaign` to exclude items from all automation

This ensures your campaign items remain under the control of the campaign orchestrator and aren't interfered with by other workflows.

## Migrating from passive to active mode

Once you've successfully run a passive campaign for 1-2 weeks and understand how it works, you can enable active execution:

**Prerequisites before enabling active mode:**
1. ✅ You've run at least 2-3 passive campaign runs successfully
2. ✅ You understand how the orchestrator coordinates work
3. ✅ You've reviewed the project board and it's tracking items correctly
4. ✅ You have clear governance rules and conservative limits set

**Migration steps:**

1. **Update your campaign spec** to add `execute-workflows: true`:
   ```yaml
   execute-workflows: true  # Enable active execution
   
   governance:
     max-new-items-per-run: 10  # Start conservative
     max-project-updates-per-run: 10
     max-comments-per-run: 5
   ```

2. **Recompile** the campaign: `gh aw compile <campaign-id>`

3. **Test with a manual run** before scheduling:
   - Trigger the workflow manually from GitHub Actions
   - Watch the run logs carefully
   - Verify it behaves as expected

4. **Monitor closely** for the first few runs:
   - Check that workflows execute correctly
   - Review any new workflows it creates
   - Ensure governance limits are appropriate

5. **Adjust governance** based on observed behavior:
   - Increase limits if runs are too conservative
   - Decrease limits if runs are too aggressive
   - Add opt-out labels if needed

**Rollback if needed:**
- Remove `execute-workflows: true` from spec
- Recompile: `gh aw compile <campaign-id>`
- Campaign reverts to passive mode

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
