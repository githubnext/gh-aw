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
- defines where to discover worker-created items
- lists the workflows the orchestrator should dispatch
- defines goals (objective + KPIs)

Most users should create specs via the [Getting started flow](/gh-aw/guides/campaigns/getting-started/).

## Complete spec example

This example shows a complete, working campaign spec with all commonly-used fields:

```yaml
# .github/workflows/framework-upgrade.campaign.md
---
id: framework-upgrade
version: "v1"
name: "Framework Upgrade"
description: "Move services to Framework vNext"

project-url: "https://github.com/orgs/ORG/projects/1"
tracker-label: "campaign:framework-upgrade"

# Discovery: Where to find worker-created issues/PRs
discovery-repos:
  - "myorg/service-a"
  - "myorg/service-b"
# Or use discovery-orgs for organization-wide discovery:
# discovery-orgs:
#   - "myorg"

# Optional: Custom GitHub token for Projects v2 operations
# project-github-token: "${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}"

# Optional: Restrict which repos this campaign can operate on
# If omitted, defaults to the repository where the campaign spec lives.
# allowed-repos:
#   - "myorg/service-a"
#   - "myorg/service-b"
# allowed-orgs:
#   - "myorg"

objective: "Upgrade all services to Framework vNext with zero downtime."
kpis:
  - id: services_upgraded
    name: "Services upgraded"
    priority: primary
    unit: count
    baseline: 0
    target: 50
    time-window-days: 30
    direction: "increase"
  - id: incidents
    name: "Incidents caused"
    priority: supporting
    unit: count
    baseline: 5
    target: 0
    time-window-days: 30
    direction: "decrease"

workflows:
  - framework-upgrade

state: "active"
owners:
  - "platform-team"
---
```

## Core fields (what they do)

### Required

- `id`: Stable identifier used for file naming and reporting (lowercase letters, digits, hyphens only).
- `name`: Human-friendly name for the campaign.
- `project-url`: GitHub Project URL used for tracking.

### Required for discovery

When your campaign uses `workflows` or `tracker-label`, you must specify where to discover worker-created items:

- `discovery-repos`: List of repositories (in `owner/repo` format) where worker workflows create issues/PRs.
- `discovery-orgs`: List of GitHub organizations where worker workflows operate (searches all repos in those orgs).

At least one of `discovery-repos` or `discovery-orgs` is required when using workflows or tracker labels.

### Commonly used

- `objective`: One sentence describing what success means for this campaign.
- `kpis`: List of 1-3 KPIs used to measure progress toward the objective.
- `workflows`: Workflow IDs the orchestrator can dispatch via `workflow_dispatch`.
- `tracker-label`: Label used to discover worker-created issues/PRs (commonly `campaign:<id>`).
- `state`: Lifecycle state (`planned`, `active`, `paused`, `completed`, or `archived`).

### Optional

- `allowed-repos`: Repositories this campaign can operate on (defaults to the repo containing the spec).
- `allowed-orgs`: Organizations this campaign can operate on.
- `project-github-token`: Token expression for Projects v2 operations (e.g., `${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}`).
- `description`: Brief description of the campaign.
- `version`: Spec version (defaults to `v1`).
- `owners`: Primary human owners for this campaign.
- `governance`: Pacing and opt-out policies (see Governance section below).

> [!IMPORTANT]
> - Use `priority: primary` (not `primary: true`) to mark your primary KPI.
> - The `discovery-*` fields control WHERE to search for worker outputs.
> - The `allowed-*` fields control WHERE the campaign can operate.

## Strategic goals (objective + KPIs)

Use `objective` and `kpis` to define what “done” means and how progress should be reported.

- `objective`: a one-sentence definition of success.
- `kpis`: a small set of measures shown in status updates.

## KPIs (recommended shape)

Each KPI requires these fields:

- `name`: Human-readable KPI name.
- `baseline`: Starting value.
- `target`: Goal value.
- `time-window-days`: Rolling window for measurement (e.g., 7, 14, 30 days).

Optional fields:

- `id`: Stable identifier (lowercase letters, digits, hyphens).
- `priority`: `primary` or `supporting` (exactly one KPI should be primary).
- `unit`: Unit of measurement (e.g., `count`, `percent`, `days`).
- `direction`: `increase` or `decrease` (describes improvement direction).
- `source`: Signal source (`ci`, `pull_requests`, `code_security`, or `custom`).

Keep KPIs small and crisp:

- Use 1 primary KPI + up to 2 supporting KPIs (maximum 3 total).
- When you define `kpis`, also define `objective` (and vice versa).

## Unified tracking (GitHub Project)

Use `project-url` to point the campaign at a GitHub Project board for tracking.

- `project-url`: the Project URL (for example: `https://github.com/orgs/ORG/projects/1`).
- `project-github-token` (optional): a token to use for Projects operations when `GITHUB_TOKEN` isn’t enough.

Project updates are applied by the orchestrator using safe outputs; see [Update Project](/gh-aw/reference/safe-outputs/#project-board-updates-update-project).

## Worker workflows

Use `workflows` to list the dispatchable workflows (“workers”) the orchestrator can trigger via `workflow_dispatch`.

For worker requirements and dispatch behavior, see [Dispatching worker workflows](/gh-aw/guides/campaigns/lifecycle/#dispatching-worker-workflows).

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

- See [Campaign lifecycle](/gh-aw/guides/campaigns/lifecycle/) for what happens each run.
- See [CLI commands](/gh-aw/guides/campaigns/cli-commands/) to validate and inspect campaigns.
