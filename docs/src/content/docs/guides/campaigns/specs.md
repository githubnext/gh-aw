---
title: Campaign specs
description: Define and configure agentic campaigns with spec files, tracker labels, and recommended wiring
banner:
  content: '<strong>Do not use.</strong> Campaigns are still incomplete and may produce unreliable or unintended results.'
---

Campaigns are defined as Markdown files under `.github/workflows/` with a `.campaign.md` suffix. The YAML frontmatter is the campaign “contract”; the body can contain optional narrative context.

## What this file does

The campaign spec is a reviewable configuration file that:

- names the campaign
- points to a GitHub Project for tracking
- lists the workflows the orchestrator should dispatch
- defines goals (objective + KPIs)

Most users should create specs via the [Getting started flow](/gh-aw/guides/campaigns/getting-started/).

## Minimal spec

```yaml
# .github/workflows/framework-upgrade.campaign.md
id: framework-upgrade
version: "v1"
name: "Framework Upgrade"
description: "Move services to Framework vNext"

project-url: "https://github.com/orgs/ORG/projects/1"
tracker-label: "campaign:framework-upgrade"

# Optional: Custom GitHub token for Projects v2 operations
# project-github-token: "${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}"

## Optional: Repositories this campaign can operate on
## If omitted, defaults to the repository where the campaign spec lives.
allowed-repos:
  - "myorg/service-a"
  - "myorg/service-b"

# Optional: Organizations this campaign can operate on
# allowed-orgs:
#   - "myorg"

objective: "Upgrade all services to Framework vNext with zero downtime."
kpis:
  - id: services_upgraded
    name: "Services upgraded"
    priority: primary
    direction: "increase"
    target: 50
  - id: incidents
    name: "Incidents caused"
    direction: "decrease"
    target: 0

# Required: Workflows to orchestrate
workflows:
  - framework-upgrade

state: "active"
owners:
  - "platform-team"
```

## Core fields (what they do)

### Required

- `id`: stable identifier used for file naming and reporting.
- `name`: human-friendly name.
- `project-url`: GitHub Project used for tracking.
- `objective`: one sentence describing what “done” means.
- `kpis`: measures used in status updates.
- `workflows`: workflow IDs the orchestrator can dispatch (via `workflow_dispatch`).
- `state`: lifecycle state, typically `active`.

### Optional

- `tracker-label`: label used to help discovery across runs (commonly `campaign:<id>`).
- `allowed-repos`: repositories the campaign can operate on (defaults to the repo containing the spec).
- `allowed-orgs`: organizations the campaign can operate on.
- `project-github-token`: token to use for Projects operations when `GITHUB_TOKEN` isn’t enough.

> [!IMPORTANT]
> Use `priority: primary` (not `primary: true`) to mark your primary KPI.

## Strategic goals (objective + KPIs)

Use `objective` and `kpis` to define what “done” means and how progress should be reported.

- `objective`: a one-sentence definition of success.
- `kpis`: a small set of measures shown in status updates.

## KPIs (recommended shape)

Keep KPIs small and crisp:

- Use 1 primary KPI + a few supporting KPIs.
- Use `direction: increase|decrease|maintain` to describe the desired trend.
- Use `target` when there is a clear threshold.

If you define `kpis`, also define `objective` (and vice versa). It keeps the spec reviewable and makes reports consistent.

## Unified tracking (GitHub Project)

Use `project-url` to point the campaign at a GitHub Project board for tracking.

- `project-url`: the Project URL (for example: `https://github.com/orgs/ORG/projects/1`).
- `project-github-token` (optional): a token to use for Projects operations when `GITHUB_TOKEN` isn’t enough.

Project updates are applied by the orchestrator using safe outputs; see [Update Project](/gh-aw/reference/safe-outputs/#project-board-updates-update-project).

## Worker workflows

Use `workflows` to list the dispatchable workflows (“workers”) the orchestrator can trigger via `workflow_dispatch`.

For worker requirements and dispatch behavior, see [Dispatching worker workflows](/gh-aw/guides/campaigns/flow/#dispatching-worker-workflows).

## Governance (pacing)

Use `governance` to cap how much the orchestrator updates per run.

## Governance (pacing & safety)

Use `governance` to keep orchestration predictable and reviewable:

```yaml
governance:
  max-new-items-per-run: 10
  max-discovery-items-per-run: 100
  max-discovery-pages-per-run: 5
  opt-out-labels: ["campaign:skip"]
  do-not-downgrade-done-items: true
  max-project-updates-per-run: 50
  max-comments-per-run: 10
```

> [!TIP]
> Start conservative with low limits (e.g., `max-project-updates-per-run: 10`) for your first campaign, then increase as you gain confidence.

### Common fields

- `max-project-updates-per-run`: cap Project updates per run (default is conservative).
- `max-comments-per-run`: cap comments per run.
- `do-not-downgrade-done-items`: prevents moving items backward.

## Next

- See [Flow & lifecycle](/gh-aw/guides/campaigns/flow/) for what happens each run.
- See [CLI commands](/gh-aw/guides/campaigns/cli-commands/) to validate and inspect campaigns.
