---
title: "Agentic campaigns"
description: "Run structured, visible delegation initiatives with GitHub Agentic Workflows and GitHub Projects."
---

Agentic campaigns are bounded, goal-driven efforts where agents carry out work over time.

Agentic workflows are already capable of running continuously (scheduled, event-driven, and re-run), and many initiatives should be automated that way. An agentic campaign is the step from automation to delegation which makes this continuous work easy to see, review, and steer toward a specific goal, is the step from automation to delegation.

For example, an agentic workflow can run an agent on a schedule, decide whether a repo needs a dependency bump, and then emit a `create_pull_request` safe-output to open the PR. An agentic campaign uses that same kind of agentic workflow as a repeatable worker and adds the coordination layer: it defines the objective and KPIs, applies a tracker label, keeps a GitHub Project updated, and writes durable progress to repo-memory until the goal is met.

## When to use a campaign

Use a campaign when you need to track progress over time (days/weeks). Use an agentic workflow when you just need a single automated run with logs/artifacts and pass/fail.

- Example of an agentic workflow (single run):
  > Every time a new discussion is created, classify it and apply labels. If it fails, show an error in the run logs.

- Example of an agentic campaign (ongoing initiative):
  > Over the next two weeks, label and triage 500 existing Actions discussions to a new taxonomy, track completion with a campaign:actions-labeling label, and publish weekly progress updates (done/remaining, top failure reasons).

## What you get

You get a GitHub Project as the dashboard, a generated orchestrator workflow that keeps that dashboard in sync, and a spec file that makes the effort reviewable (objective, KPIs, governance, and wiring). The orchestrator is just another workflow; campaigns are a way of wiring workflows together around a shared goal.

## What it is in the repo

A campaign is defined by a spec file and, when needed, a generated orchestrator. The spec lives at `.github/workflows/<id>.campaign.md`. When the spec includes meaningful campaign wiring, compilation also generates `.github/workflows/<id>.campaign.g.md` and compiles it to a `.lock.yml` workflow.

The spec is the source of truth for what success means (the objective), how progress is measured (KPIs, with exactly one marked `primary`), where progress is shown (the GitHub Project URL), what participates (the workflows), and what is tracked (the label applied to issues and pull requests, commonly `campaign:<id>`).

## How it works

Most campaigns follow the same shape. The GitHub Project is the human-facing status view. The orchestrator workflow discovers tracked items and updates the Project. Worker workflows (when you use them) do the real work, such as opening pull requests or applying fixes.

Workers stay campaign-agnostic. If you want cross-run discovery of worker-created assets, workers can include a `tracker-id` marker and the orchestrator can search for it.

## Durable state (repo-memory)

Campaigns become repeatable when they also write durable state to repo-memory (a git branch used for snapshots). The recommended layout is `memory/campaigns/<campaign-id>/cursor.json` for the checkpoint (treated as an opaque JSON object) and `memory/campaigns/<campaign-id>/metrics/<date>.json` for append-only metrics snapshots.

Campaign tooling enforces this durability contract at push time: when a campaign writes repo-memory, it must include a cursor and at least one metrics snapshot.

## Next steps

- [Getting started](/gh-aw/guides/campaigns/getting-started/) – create a campaign quickly
- [Campaign specs](/gh-aw/guides/campaigns/specs/) – spec fields (objective/KPIs, governance, memory)
- [Project management](/gh-aw/guides/campaigns/project-management/) – project board setup tips
- [CLI commands](/gh-aw/guides/campaigns/cli-commands/) – CLI reference
