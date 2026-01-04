---
title: "Agentic campaigns"
description: "Run structured, visible delegation initiatives with GitHub Agentic Workflows and GitHub Projects."
---

Agentic campaigns are bounded, goal-driven initiatives that run over time and are easy to see, review, and steer. They reuse agentic workflows as repeatable workers, but add a delegation layer: a clear objective, a set of measurable KPIs, governance, and a shared tracker so progress stays consistent across many runs.

While agentic workflows can already run continuously (scheduled, event-driven, and rerunnable), a campaign is what you use when you want that continuous work to be directed at a specific outcome and managed like a project, not just executed like automation.

For example, an agentic workflow might run nightly, decide whether a repo needs a dependency bump, and open a pull request. An agentic campaign uses the same kind of workflow as a worker and adds coordination: it defines what success looks like, tracks progress against KPIs, and maintains a single source of truth for status via a GitHub Project that automatically reflects which tracked items are new, in progress, blocked, or done. It also writes durable checkpoints and metrics so the campaign can resume safely and report progress until the goal is met.

## When to use a campaign

Use a campaign when you need to manage an initiative—scope, progress, and outcomes—across many workflow runs, not just execute automation and inspect each run’s result. Workflows execute; campaigns coordinate work toward a goal with shared tracking and a clear definition of done.

- Example of an agentic workflow (per-event automation):
  > Every time a new discussion is created, classify it and apply labels. If it fails, show an error in the run logs.

- Example of an agentic campaign (ongoing initiative):
  > Over the next two weeks, label and triage 500 existing Actions discussions to a new taxonomy, track completion with a campaign:actions-labeling label, and publish weekly progress updates (done/remaining, top failure reasons).

## Campaign structure

A campaign gives you a dashboard (GitHub Project), a coordinating orchestrator workflow that keeps it in sync, and a spec file that captures the objective, KPIs, governance, and wiring. In the repo, the spec lives at `.github/workflows/<id>.campaign.md` and is the source of truth.

When the spec includes orchestration, the tooling generates an orchestrator workflow and compiles it into a locked `.campaign.lock.yml` workflow. The spec defines what success means (objective), how progress is measured (KPIs, with exactly one marked primary), where progress is shown (GitHub Project URL), and what participates (workflows). Optionally, you can add a tracker label (commonly `campaign:<id>`) to help discover issues and PRs, but the project board remains the canonical source of campaign membership.

**Note:** During compilation, a `.campaign.g.md` file is generated locally as a debug artifact to help developers understand the orchestrator structure, but this file is not committed to git—only the source `.campaign.md` and compiled `.campaign.lock.yml` are tracked.

### Mental model

```mermaid
flowchart TB
    spec["fa:fa-file-code .github/workflows/<id>.campaign.md<br/><small>specification / contract<br/>(tracked in git)</small>"]
    compile["fa:fa-cogs gh aw compile"]
    debug["fa:fa-file .campaign.g.md<br/><small>debug artifact<br/>(not tracked)</small>"]
    lock["fa:fa-lock .campaign.lock.yml<br/><small>compiled workflow<br/>(tracked in git)</small>"]
    orchestrator["fa:fa-sitemap Orchestrator workflow<br/><small>discovers items from project<br/>updates Project dashboard<br/>reads/writes repo-memory</small>"]
    worker1["fa:fa-robot Worker workflow<br/><small>agent + safe-outputs</small>"]
    worker2["fa:fa-robot Worker workflow<br/><small>agent + safe-outputs</small>"]
    project["fa:fa-table GitHub Project board<br/><small>campaign dashboard</small>"]
    memory["fa:fa-code-branch repo-memory branch<br/><small>memory/campaigns/<id>/cursor.json<br/>memory/campaigns/<id>/metrics/<date>.json</small>"]

    spec --> compile
    compile --> debug
    compile --> lock
    lock --> orchestrator
    orchestrator -->|triggers/coordinates| worker1
    orchestrator -->|triggers/coordinates| worker2
    worker1 -->|"creates/updates<br/>Issues/PRs<br/>(optional tracker-label)"| project
    worker2 -->|"creates/updates<br/>Issues/PRs<br/>(optional tracker-label)"| project
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

## How it works

Most campaigns follow the same shape. The GitHub Project is the human-facing status view and the canonical source of campaign membership. The orchestrator workflow discovers tracked items from the workers and updates the Project. Worker workflows do the real work, such as opening pull requests or applying fixes but they stay campaign-agnostic. If you want cross-run discovery of worker-created assets, workers can include a `tracker-id` marker which the orchestrator can search for. Optionally, you can configure a tracker label (e.g., `campaign:<id>`) as an ingestion hint to help discover issues and PRs created by workers.

## Memory

Campaigns become repeatable when they also write durable state to repo-memory (a git branch used for snapshots). The recommended layout is `memory/campaigns/<campaign-id>/cursor.json` for the checkpoint (treated as an opaque JSON object) and `memory/campaigns/<campaign-id>/metrics/<date>.json` for append-only metrics snapshots.

Campaign tooling enforces this durability contract at push time: when a campaign writes repo-memory, it must include a cursor and at least one metrics snapshot.

## Next steps

- [Getting started](/gh-aw/guides/campaigns/getting-started/) – create a campaign quickly
- [Campaign specs](/gh-aw/guides/campaigns/specs/) – spec fields (objective/KPIs, governance, memory)
- [Project management](/gh-aw/guides/campaigns/project-management/) – project board setup tips
- [CLI commands](/gh-aw/guides/campaigns/cli-commands/) – CLI reference
