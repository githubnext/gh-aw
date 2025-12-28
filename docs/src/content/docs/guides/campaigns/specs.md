---
title: "Campaign Specs"
description: "Define and configure agentic campaigns with spec files, tracker labels, and recommended wiring"
---

Campaigns are defined as Markdown files under `.github/workflows/` with a `.campaign.md` suffix. The YAML frontmatter is the campaign “contract”; the body can contain optional narrative context.

## What a campaign is (in gh-aw)

In GitHub Agentic Workflows, a campaign is not “a special kind of workflow.” The `.campaign.md` file is a specification: a reviewable contract that wires together agentic workflows around a shared initiative (a tracker label, a GitHub Project dashboard, and optional durable state).

In a typical setup:

- Worker workflows do the work. They run an agent and use safe-outputs (for example `create_pull_request`, `add_comment`, or `update_issues`) for write operations.
- A generated orchestrator workflow keeps the campaign coherent over time. It discovers items tagged with your tracker label, updates the Project board, and produces ongoing progress reporting.
- Repo-memory (optional) makes the campaign repeatable. It lets you store a cursor checkpoint and append-only metrics snapshots so each run can pick up where the last one left off.

### Mental model (ASCII)

```
  .github/workflows/<id>.campaign.md
  (specification / contract)
      |
      |  gh aw compile
      v
  .github/workflows/<id>.campaign.g.md  ->  <id>.campaign.lock.yml
  (generated orchestrator source)           (compiled workflow)
      |
      |  discovers items via tracker-label (e.g. campaign:<id>)
      |  updates Project dashboard
      |  reads/writes repo-memory (cursor + metrics)
      v
  +---------------------------+
  | Orchestrator workflow     |
  +---------------------------+
    |                  |
    | triggers/coordinates |
    v                  v
  +----------------+   +----------------+
  | Worker workflow |   | Worker workflow |
  | (agent +        |   | (agent +        |
  | safe-outputs)   |   | safe-outputs)   |
  +----------------+   +----------------+
    |
    | creates/updates Issues/PRs with tracker-label
    v
  GitHub Project board  <---  "campaign dashboard"

  repo-memory branch:
  memory/campaigns/<id>/cursor.json
  memory/campaigns/<id>/metrics/<date>.json
```

Editable diagram (draw.io): `docs/src/content/docs/guides/campaigns/agentic-campaign.drawio`

This is why campaigns feel like “delegation over time”: you are defining success, scope, and reporting, not just describing a single run.

## Minimal spec

```yaml
# .github/workflows/framework-upgrade.campaign.md
id: framework-upgrade
version: "v1"
name: "Framework Upgrade"
description: "Move services to Framework vNext"

project-url: "https://github.com/orgs/ORG/projects/1"
tracker-label: "campaign:framework-upgrade"

objective: "Upgrade all services to Framework vNext with zero downtime."
kpis:
  - id: services_upgraded
    name: "Services upgraded"
    primary: true
    direction: "increase"
    target: 50
  - id: incidents
    name: "Incidents caused"
    direction: "decrease"
    target: 0

workflows:
  - framework-upgrade

state: "active"
owners:
  - "platform-team"
```

## Core fields (what they do)

- `id`: stable identifier used for file naming, reporting, and (if used) repo-memory paths.
- `project-url`: the GitHub Project that acts as the campaign dashboard.
- `tracker-label`: the label applied to issues and pull requests that belong to the campaign (commonly `campaign:<id>`). This is the key that lets the orchestrator discover work across runs.
- `objective`: a single sentence describing what “done” means.
- `kpis`: the measures you use to report progress (exactly one should be marked `primary`).
- `workflows`: the participating workflow IDs. These refer to workflows in the repo (commonly `.github/workflows/<workflow-id>.md`), and they can be scheduled, event-driven, or long-running.

## KPIs (recommended shape)

Keep KPIs small and crisp:

- Use 1 primary KPI + a few supporting KPIs.
- Use `direction: increase|decrease|maintain` to describe the desired trend.
- Use `target` when there is a clear threshold.

If you define `kpis`, also define `objective` (and vice versa). It keeps the spec reviewable and makes reports consistent.

## Durable state (repo-memory)

If you use repo-memory for campaigns, standardize on one layout so runs are comparable:

- `memory/campaigns/<campaign-id>/cursor.json`
- `memory/campaigns/<campaign-id>/metrics/<date>.json`

Typical wiring in the spec:

```yaml
memory-paths:
  - "memory/campaigns/framework-upgrade/cursor.json"
metrics-glob: "memory/campaigns/framework-upgrade/metrics/*.json"
```

Campaign tooling enforces the durability contract at push time: a campaign repo-memory write must include a cursor and at least one metrics snapshot.

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

## Compilation and orchestrators

`gh aw compile` validates campaign specs. When the spec has meaningful details (tracker label, workflows, memory paths, or a metrics glob), it also generates an orchestrator `.github/workflows/<id>.campaign.g.md` and compiles it to `.lock.yml`.

See [Agentic campaign specs and orchestrators](/gh-aw/setup/cli/#compile).
