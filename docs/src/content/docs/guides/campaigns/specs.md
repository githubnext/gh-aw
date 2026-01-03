---
title: "Campaign Specs"
description: "Define and configure agentic campaigns with spec files, tracker labels, and recommended wiring"
---

Campaigns are defined as Markdown files under `.github/workflows/` with a `.campaign.md` suffix. The YAML frontmatter is the campaign “contract”; the body can contain optional narrative context.

## What a campaign is (in gh-aw)

In GitHub Agentic Workflows, a campaign is not “a special kind of workflow.” The `.campaign.md` file is a specification: a reviewable contract that wires together agentic workflows around a shared initiative (a GitHub Project dashboard as the canonical source of membership, optional tracker label for ingestion, and optional durable state).

In a typical setup:

- Worker workflows do the work. They run an agent and use safe-outputs (for example `create_pull_request`, `add_comment`, or `update_issues`) for write operations.
- A generated orchestrator workflow keeps the campaign coherent over time. It discovers items from the project board (optionally using tracker labels), updates the Project board, and produces ongoing progress reporting.
- Repo-memory (optional) makes the campaign repeatable. It lets you store a cursor checkpoint and append-only metrics snapshots so each run can pick up where the last one left off.

### Mental model

```mermaid
flowchart TB
    spec["fa:fa-file-code .github/workflows/&lt;id&gt;.campaign.md<br/><small>specification / contract<br/>(tracked in git)</small>"]
    compile["fa:fa-cogs gh aw compile"]
    debug["fa:fa-file .campaign.g.md<br/><small>debug artifact<br/>(not tracked)</small>"]
    lock["fa:fa-lock .campaign.lock.yml<br/><small>compiled workflow<br/>(tracked in git)</small>"]
    orchestrator["fa:fa-sitemap Orchestrator workflow<br/><small>discovers items from project<br/>updates Project dashboard<br/>reads/writes repo-memory</small>"]
    worker1["fa:fa-robot Worker workflow<br/><small>agent + safe-outputs</small>"]
    worker2["fa:fa-robot Worker workflow<br/><small>agent + safe-outputs</small>"]
    project["fa:fa-table GitHub Project board<br/><small>campaign dashboard</small>"]
    memory["fa:fa-code-branch repo-memory branch<br/><small>memory/campaigns/&lt;id&gt;/cursor.json<br/>memory/campaigns/&lt;id&gt;/metrics/&lt;date&gt;.json</small>"]

    spec --> compile
    compile --> debug
    compile --> lock
    lock --> orchestrator
    orchestrator -->|triggers/coordinates| worker1
    orchestrator -->|triggers/coordinates| worker2
    worker1 -->|creates/updates<br/>Issues/PRs<br/>(optional tracker-label)| project
    worker2 -->|creates/updates<br/>Issues/PRs<br/>(optional tracker-label)| project
    orchestrator -.->|reads/writes| memory
    project -.->|dashboard view| orchestrator

    %% Colors are applied via CSS for light/dark themes; keep only stroke width here.
    classDef tracked stroke-width:2px
    classDef notTracked stroke-width:2px
    classDef workflow stroke-width:2px
    classDef external stroke-width:2px

    class spec,lock tracked
    class debug notTracked
    class orchestrator,worker1,worker2 workflow
    class project,memory external
```

**Note:** The `.campaign.g.md` file is a local debug artifact generated during compilation to help developers review the orchestrator structure. It is not committed to git (it's in `.gitignore`). Only the source `.campaign.md` and the compiled `.campaign.lock.yml` are version controlled.

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
- `project-url`: the GitHub Project that acts as the campaign dashboard and canonical source of campaign membership.
- `tracker-label` (optional): an ingestion hint label that helps discover issues and pull requests created by workers (commonly `campaign:<id>`). When provided, the orchestrator can discover work across runs. The project board remains the canonical source of truth.
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

`gh aw compile` validates campaign specs. When the spec has meaningful details (tracker label, workflows, memory paths, or a metrics glob), it also generates an orchestrator and compiles it to `.campaign.lock.yml`.

During compilation, a `.campaign.g.md` file is generated locally as a debug artifact to help developers understand the orchestrator structure, but this file is not committed to git—only the compiled `.campaign.lock.yml` is tracked.

See [Agentic campaign specs and orchestrators](/gh-aw/setup/cli/#compile).
