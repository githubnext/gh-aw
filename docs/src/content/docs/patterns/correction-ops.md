---
title: CorrectionOps
description: Improve agentic workflows from trusted human corrections without retraining the underlying model
---

CorrectionOps is a workflow pattern that compares predictions with later human corrections.

Instead of retraining the model, CorrectionOps improves the workflow around the model. It stores predictions at decision time, compares them with later trusted human truth, and uses that evidence to update instructions, routing, thresholds, and rollout decisions.

The basic loop is simple:

1. Save what the workflow predicted
2. Collect what humans later decided
3. Use the difference to improve the workflow

Discussion labelling is a good example: a workflow applies labels, humans later correct those labels, and the system uses that correction evidence to improve future runs.

## When to Use CorrectionOps

Use CorrectionOps when you want to turn a human decision process into an agentic workflow iteratively rather than all at once.

It is a good fit when humans still make or correct the real decision, but you want the workflow to improve over time by updating instructions, routing, thresholds, or rollout state.

Typical fits include labelling and classification, routing and prioritization, moderation and approvals, and summaries or recommendations that humans later correct.

It is especially useful when the rollout path is gradual:

- start in report-only mode
- move to a shadow or other safe write target
- use later corrections to improve the workflow
- promote to direct writes only when the evidence is strong enough

## How It Works

A clean CorrectionOps setup has two long-lived surfaces and one optional temporary one. Production stays authoritative. Ops is the long-lived home for prediction, correction intake, reporting, and instruction updates. Shadow, when used, is just a safe write target during evaluation.

That means the workflows usually stay in ops. During evaluation they write to shadow. After promotion they can write directly to production without moving to a different repository. CorrectionOps is therefore broader than shadow evaluation. Shadow evaluation is one rollout shape inside CorrectionOps, not the whole pattern.

Most implementations reduce to three workflow classes: a thin relay that forwards stable facts into ops, a prediction workflow that persists snapshots and writes safely, and a compare/report/decide workflow that checks later human truth and updates the system when the evidence is strong enough.

The important rule is to keep relays, snapshot resolution, diffing, and grouping deterministic. Use the agent for semantic judgment, not for reconstructing event history or inferring provenance after the fact.

```aw wrap
---
on:
  schedule: daily
  workflow_dispatch:
  repository_dispatch:
    types: [truth-feedback]
permissions:
  contents: read
  issues: read
safe-outputs:
  create-issue:
  create-pull-request:
---

# CorrectionOps Worker

Read persisted predictions and later trusted truth, compare them deterministically, then either publish a health report or open a draft PR updating instructions.
```

CorrectionOps solves a different problem than model training. Reinforcement Learning from Human Feedback (RLHF) updates model weights from human feedback. CorrectionOps updates the workflow system around the model. In practice that usually means changing instruction files, routing rules, deterministic checks, thresholds, or rollout decisions rather than trying to retrain the engine.

In a healthy CorrectionOps loop, production truth stays authoritative, predictions are saved explicitly, corrections include provenance, and diffs are built deterministically before the agent is asked to reason about them.

CorrectionOps does not require a shadow surface, but many teams start with one. The normal progression is report-only first, then shadow evaluation when a safe write target is needed, then direct production writes once the evidence is strong enough.

## Implementing It With GitHub Actions

GitHub Actions is a strong fit because the pattern is mostly orchestration, artifact passing, and controlled writes across repositories. In practice, production events create the initial signal, a thin relay forwards that signal into ops, and the ops repo runs prediction and comparison work on schedules, manual dispatch, or forwarded events.

For most teams, the clearest starting point is three workflows: one thin relay in the source repo, one prediction workflow in ops, and one compare/report/decide workflow in ops. Split further only when the boundary is real, such as a different trigger, a different permission boundary, or a separate serialized write path.

In `gh-aw`, keep orchestration in frontmatter and step sections, use a small trusted set of GitHub Actions for plumbing, and keep policy-critical normalization, diffing, and grouping in repo-local scripts. `actions/github-script`, checkout, and artifact upload/download are usually enough.

```yaml title="prod/.github/workflows/relay-correction-signals.yml"
name: Relay Correction Signals

on:
  discussion:
    types: [created, labeled, unlabeled]

jobs:
  relay:
    runs-on: ubuntu-latest
    steps:
      - name: Forward stable facts to ops
        uses: actions/github-script@v8
        with:
          github-token: ${{ secrets.OPS_DISPATCH_TOKEN }}
          script: |
            await github.rest.repos.createDispatchEvent({
              owner: 'org',
              repo: 'ops-repo',
              event_type: context.payload.action === 'created' ? 'item-created' : 'truth-feedback',
              client_payload: {
                data: {
                  source_repository: `${context.repo.owner}/${context.repo.repo}`,
                  item_number: context.payload.discussion.number,
                  label: context.payload.label?.name || null,
                  actor: context.actor,
                  actor_type: context.actor.endsWith('[bot]') ? 'bot' : 'human',
                },
              },
            });
```

Most CorrectionOps systems still need both scheduled and manual entry points. A scheduled run catches drift and stale backlog. `workflow_dispatch` makes it possible to backfill one item, rerun one parent correction issue, or test a new instruction revision safely. Artifact handoff is often simpler than re-fetching everything in every step, and checkout should usually stay in ops rather than in production relays.

## Portable Starter Architecture

CorrectionOps is implementable for almost any repository that has three ingredients:

1. a production object to observe, such as issues, pull requests, discussions, labels, approvals, or comments
2. a later human action that counts as trustworthy truth
3. an operational surface, usually an ops repo, where instructions and reports can live

The minimal reusable architecture is:

- one production relay workflow
- one ops prediction workflow
- one ops compare, report, and decide workflow
- one stable snapshot schema

Many teams add a separate correction-collector workflow because the truth-ingest boundary is naturally deterministic and often triggered by `repository_dispatch`. That is a useful operational split, but it is not the simplest shape to teach first.

The repository-specific work is usually limited to how to fetch and normalize the production object, which human actions count as trusted truth, what grouped correction patterns are meaningful, and which instruction or policy files are allowed to change. That is what keeps the pattern portable across different business domains.

## Concrete Example: Discussion Labelling

Discussion labelling is one concrete CorrectionOps implementation.

- Production hosts the real discussions and later human truth.
- Ops runs the long-lived workflows.
- Shadow, when used, receives safe evaluation writes before production writes are enabled.

In that example, the shape is:

- production or shadow surface: thin relay workflows only
- ops repo: the real control loop

The concrete workflow layout looks like this.

### Production or Shadow Surface

- `community-discussion-mirror.yml`: copies production discussions into the shadow surface when a live-like write target is needed
- `label-feedback-dispatch.yml`: forwards stable discussion facts and later trusted label truth into ops

### Central Ops Repo

- `auto-labelling.md`: prediction workflow that reads prepared discussion inputs, applies safe outputs, and persists prediction snapshots
- `labelling-correction-collector.yml`: deterministic correction-intake workflow that resolves current source-of-truth state and stores correction evidence
- `discussion-labelling-ops.md`: combined compare, report, and decide workflow that either publishes health summaries or opens a draft PR updating instructions

So the simple reusable pattern is still three workflow classes, but a real multi-repo example often has five workflow files once the thin mirror and relay workflows are counted.

### Example Workflow Roles

| Workflow | Role |
| --- | --- |
| `community-discussion-mirror.yml` | Optional shadow support |
| `label-feedback-dispatch.yml` | Thin truth relay |
| `auto-labelling.md` | Predict and persist |
| `labelling-correction-collector.yml` | Deterministic truth intake |
| `discussion-labelling-ops.md` | Compare, report, and decide |

This example is already close to the elegant target: one thin relay workflow in the source or shadow surface, one prediction workflow in ops, and one compare/report/decide workflow in ops. The part that still tends to feel mechanical is the deterministic helper layer behind those workflows, not the repo split itself.

This general shape also applies to routing, moderation, prioritization, approvals, summaries, and other decisions where later human actions provide trustworthy operational truth.

## Relationship To Other Patterns

CorrectionOps overlaps with several adjacent ideas, but it solves a narrower problem.

- Shadow deployment evaluates a candidate safely on live traffic. CorrectionOps adds the correction-driven adaptation loop.
- Human-in-the-loop review adds oversight at decision time. CorrectionOps adds a durable memory of corrections and uses it to change the workflow later.
- LLMOps and AgentOps provide broader tracing, evaluation, and governance capabilities. CorrectionOps is a specific design pattern for using trusted corrections to improve production-adjacent workflows.
- RLHF updates model weights from human preference data. CorrectionOps updates the operational system around the model instead.

## Related Documentation

- [Safe Rollout](/gh-aw/organization-practices/safe-rollout/) for the optional safe-write rollout guidance inside CorrectionOps
- [SideRepoOps](/gh-aw/patterns/side-repo-ops/) for separating workflow infrastructure from the production repository
- [MultiRepoOps](/gh-aw/patterns/multi-repo-ops/) for coordinating workflows across repository boundaries
- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) for controlling write targets and protections
- [GitHub Tools](/gh-aw/reference/github-tools/) for cross-repository reads and operations
