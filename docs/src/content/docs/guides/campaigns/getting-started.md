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

## Start an agentic campaign with GitHub Issue Forms

This repo also includes a "🚀 Start an Agentic Campaign" issue form. Use it when you want to capture intent first and let an agent scaffold the spec in a PR.

### Creating the Campaign Issue

**Recommended:** Create from your project board (Open board → "Add item" → "Create new issue" → Select "🚀 Start an Agentic Campaign" template). The project will be automatically assigned.

**Alternative:** Create from Issues → New Issue → Select "🚀 Start an Agentic Campaign". Remember to assign the project manually before submitting.

Submitting the form creates a campaign issue (your campaign hub), validates the project board, generates the campaign spec (`.github/workflows/<id>.campaign.md`) in a PR, and configures tracking. Manage your campaign from the issue—workflow files are implementation details.

### Benefits

Using issue forms captures intent over syntax, validates required fields, lowers the barrier to entry, provides a traceable command center, automates spec generation, and handles project assignment automatically.
