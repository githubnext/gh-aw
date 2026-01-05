---
title: "Getting Started"
description: "Quick start guide for creating and launching agentic campaigns"
---

This guide is the shortest path from “we want a campaign” to a working dashboard.

## Quick start (5 steps)

1. Create a GitHub Project board (manual, one-time) and copy its URL.
2. Add `.github/workflows/<id>.campaign.md` in a PR.
3. Run `gh aw compile`.
4. Run the generated orchestrator workflow from the Actions tab.
5. Apply the tracker label to issues/PRs you want tracked.

## 1) Create the dashboard (GitHub Project)

In GitHub: your org → **Projects** → **New project**.

**Quick start setup**:
- Start with a **Table** view (simplest option)
- Add a **Board** view grouped by `Status` for kanban-style tracking
- Consider a **Roadmap** view for timeline visualization (requires Start Date/End Date fields)

**Recommended custom fields** (see [Project Management](/gh-aw/guides/campaigns/project-management/) for details):
- **Status** (Single select): Todo, In Progress, Blocked, Done
- **Worker/Workflow** (Single select): Names of your worker workflows
- **Priority** (Single select): High, Medium, Low
- **Start Date** / **End Date** (Date): For roadmap timeline views

Copy the Project URL (it must be a full URL like `https://github.com/orgs/myorg/projects/42`).

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

## Optional: repo-memory for durable state

If you enable repo-memory for campaigns, use a stable layout:

- `memory/campaigns/<campaign-id>/cursor.json`
- `memory/campaigns/<campaign-id>/metrics/<date>.json`

Campaign tooling enforces that a campaign repo-memory write includes a cursor and at least one metrics snapshot.

## Start an agentic campaign with GitHub Issue Forms

This repo also includes a "🚀 Start an Agentic Campaign" issue form. Use it when you want to capture intent first and let an agent scaffold the spec in a PR.

### Creating the Campaign Issue

**Option A (Recommended):** Create from your project board:
1. Open your GitHub Project board
2. Click "Add item" → "Create new issue"  
3. Select this repository and choose the "🚀 Start an Agentic Campaign" template
4. The project will be automatically assigned to the issue ✅

**Option B:** Create from the repository's Issues page:
1. Navigate to Issues → New Issue
2. Select "🚀 Start an Agentic Campaign"
3. **Important:** Before submitting, scroll down and use the project selector to assign the issue to your project board

Creating the issue from the project board (Option A) is recommended as it ensures the project is automatically assigned and reduces the chance of forgetting this required step.

When you submit the issue form:

1. **an agentic campaign issue is created** - This becomes your campaign's central hub with the `campaign` and `campaign-tracker` labels
2. **An agent validates your project board** - Ensures the project assignment exists and is accessible
3. **an agentic campaign spec is generated** - Creates `.github/workflows/<id>.campaign.md` with your inputs as a PR
4. **The spec is linked to the issue** - So you can track the technical implementation
5. **Your project board is configured** - The agent sets up tracking labels and fields

You manage the agentic campaign from the issue. The generated workflow files are implementation details and should not be edited directly.

### Benefits of the issue form approach

- **Captures intent, not YAML**: Focus on what you want to accomplish, not technical syntax
- **Structured validation**: Form fields ensure required information is provided
- **Lower barrier to entry**: No need to understand campaign spec file format
- **Traceable**: Issue serves as the agentic campaign's command center with full history
- **Agent-assisted scaffolding**: Automated generation of spec files and workflows
- **Automatic project assignment**: When created from project board, the project is automatically linked
