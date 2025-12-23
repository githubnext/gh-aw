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

- Keep it simple: a Table view is enough.
- If you want lanes, create a Board view and group by a single-select field (commonly `Status`).

Copy the Project URL (it must be a full URL).

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

This validates the spec. When the spec has meaningful details (tracker label, workflows, memory paths, or a metrics glob), `compile` also generates an orchestrator `.campaign.g.md` and compiles it to `.lock.yml`.

## 4) Run the orchestrator

Trigger the orchestrator workflow from GitHub Actions. Its job is to keep the dashboard in sync:

- Finds tracker-labeled issues/PRs
- Adds them to the Project
- Updates fields/status
- Posts a short report

## 5) Add work items

Apply the tracker label (for example `campaign:framework-upgrade`) to issues/PRs you want tracked. The orchestrator will pick them up on the next run.

## Optional: repo-memory for durable state

If you enable repo-memory for campaigns, use a stable layout:

- `memory/campaigns/<campaign-id>/cursor.json`
- `memory/campaigns/<campaign-id>/metrics/<date>.json`

Campaign tooling enforces that a campaign repo-memory write includes a cursor and at least one metrics snapshot.

## Start an agentic campaign with GitHub Issue Forms

This repo also includes a “🚀 Start an Agentic Campaign” issue form. Use it when you want to capture intent first and let an agent scaffold the spec in a PR.

When you submit the issue form:

1. **an agentic campaign issue is created** - This becomes your campaign's central hub with the `campaign` and `campaign-tracker` labels
2. **An agent validates your project board** - Ensures the URL is accessible and properly configured
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
