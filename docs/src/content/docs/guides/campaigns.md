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

When the spec includes orchestration, the tooling generates `.github/workflows/<id>.campaign.g.md` and compiles it into a locked `.lock.yml` workflow. The spec defines what success means (objective), how progress is measured (KPIs, with exactly one marked primary), where progress is shown (GitHub Project URL), what participates (workflows), and what is tracked (the label applied to issues and pull requests, commonly `campaign:<id>`).

## How it works

Most campaigns follow the same shape. The GitHub Project is the human-facing status view. The orchestrator workflow discovers tracked items from the workers and updates the Project. Worker workflows do the real work, such as opening pull requests or applying fixes but they stay campaign-agnostic. If you want cross-run discovery of worker-created assets, workers can include a `tracker-id` marker which the orchestrator can search for.

## Memory

Campaigns become repeatable when they also write durable state to repo-memory (a git branch used for snapshots). The recommended layout is `memory/campaigns/<campaign-id>/cursor.json` for the checkpoint (treated as an opaque JSON object) and `memory/campaigns/<campaign-id>/metrics/<date>.json` for append-only metrics snapshots.

Campaign tooling enforces this durability contract at push time: when a campaign writes repo-memory, it must include a cursor and at least one metrics snapshot.

## Next steps

- [Getting started](/gh-aw/guides/campaigns/getting-started/) – create a campaign quickly
- [Campaign specs](/gh-aw/guides/campaigns/specs/) – spec fields (objective/KPIs, governance, memory)
- [Project management](/gh-aw/guides/campaigns/project-management/) – project board setup tips
- [CLI commands](/gh-aw/guides/campaigns/cli-commands/) – CLI reference
