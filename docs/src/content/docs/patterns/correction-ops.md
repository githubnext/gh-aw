---
title: CorrectionOps
description: Improve agentic workflows from trusted human corrections without retraining the underlying model
sidebar:
  badge: Pattern
---

CorrectionOps is a workflow pattern that compares predictions with later human corrections.

Instead of retraining the model, CorrectionOps improves the workflow around the model. It stores predictions at decision time, compares them with later trusted human truth, and uses that evidence to update instructions, routing, thresholds, and rollout decisions.

The basic loop is simple:

1. Save what the workflow predicted
2. Collect what humans later decided
3. Use the difference to improve the workflow

Discussion labeling is a good example: a workflow applies labels, humans later correct those labels, and the system uses that correction evidence to improve future runs.

## When to Use CorrectionOps

Use CorrectionOps when you want to turn a human decision process into an agentic workflow iteratively rather than all at once.

It is a good fit when humans still make or correct the real decision, but you want the workflow to improve over time by updating instructions, routing, thresholds, or rollout state.

Typical fits include labeling and classification, routing and prioritization, moderation and approvals, and summaries or recommendations that humans later correct.

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

## Reproducible Starter Setup

This page intentionally uses generic repository and workflow names so the pattern can be reproduced without depending on any partner repository.

The simplest teachable setup uses two repositories and an optional third:

- `prod-repo`: the authoritative system where the original object and later human truth live
- `ops-repo`: the long-lived control plane for prediction, correction review, reporting, and instruction updates
- `shadow-repo`: an optional safe write target used only during rollout

The workflow layout is:

| Repository | Workflow | Role |
| --- | --- | --- |
| `prod-repo` | `relay-correction-signals.yml` | Thin deterministic relay |
| `ops-repo` | `predict-items.md` | Predict and persist snapshots |
| `ops-repo` | `review-corrections.md` | Compare, report, and decide |
| `ops-repo` | `collect-corrections.yml` | Optional deterministic truth intake |
| `shadow-repo` | `mirror-items.yml` | Optional safe-write support |

If the source event stream already contains everything needed for later comparison, skip `collect-corrections.yml`. If direct writes are too risky during rollout, add `mirror-items.yml` and point safe outputs at `shadow-repo` until the evidence is strong enough.

### 1. Thin Relay In The Source Repo

The relay only forwards stable facts and provenance into ops. It should not compute diffs, infer human intent, or decide whether the workflow was correct.

```yaml title="prod-repo/.github/workflows/relay-correction-signals.yml"
name: Relay Correction Signals

on:
  issues:
    types: [opened, labeled, unlabeled]

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
              event_type: context.payload.action === 'opened' ? 'item-created' : 'truth-feedback',
              client_payload: {
                data: {
                  source_repository: `${context.repo.owner}/${context.repo.repo}`,
                  source_type: 'issue',
                  item_number: context.payload.issue.number,
                  item_title: context.payload.issue.title,
                  item_url: context.payload.issue.html_url,
                  event_type: context.payload.action,
                  label: context.payload.label?.name || null,
                  actor: context.actor,
                  actor_type: context.actor.endsWith('[bot]') ? 'bot' : 'human',
                  occurred_at: new Date().toISOString(),
                },
              },
            });
```

### 2. Prediction Workflow In Ops

The prediction workflow consumes normalized inputs, applies the current instructions, writes through safe outputs, and persists a durable snapshot that can be compared later.

```aw wrap title="ops-repo/.github/workflows/predict-items.md"
---
name: Predict Items

on:
  schedule: daily
  workflow_dispatch:
  repository_dispatch:
    types: [item-created]

tools:
  github:
    toolsets: [issues, repos]

safe-outputs:
  create-issue:
  update-issue:
    target-repo: ${{ inputs.target-repo || 'shadow-repo' }}
---

# Predict Items

Read prepared items from `/tmp/gh-aw/agent/item-scan`, apply the current instructions, write the proposed changes through safe outputs, and append a prediction snapshot containing the source identifier, predicted action, instruction version, and timestamp.
```

### 3. Compare, Report, And Decide In Ops

The review workflow reads persisted predictions and later human truth, builds deterministic diffs first, and only then asks the agent to summarize patterns or propose instruction updates.

```aw wrap title="ops-repo/.github/workflows/review-corrections.md"
---
name: Review Corrections

on:
  schedule: weekly
  workflow_dispatch:
    inputs:
      mode:
        description: report or adaptation
        required: false
        default: report
        type: choice
        options: [report, adaptation]

safe-outputs:
  create-issue:
  create-pull-request:
---

# Review Corrections

Read `correction-diffs.json` from `/tmp/gh-aw/agent/correction-review`. In `report` mode, publish a health summary. In `adaptation` mode, open a draft PR updating the instruction file only when the grouped evidence is strong enough.
```

### 4. Optional Deterministic Collector

Add a separate collector only when the later-truth boundary deserves its own trigger, permissions, or serialized write path.

```yaml title="ops-repo/.github/workflows/collect-corrections.yml"
name: Collect Corrections

on:
  repository_dispatch:
    types: [truth-feedback]

jobs:
  collect:
    runs-on: ubuntu-latest
    steps:
      - name: Resolve authoritative truth and store correction evidence
        run: ./scripts/store-correction-evidence.sh
```

### 5. Stable Contracts To Define First

Before adding rollout logic or adaptation prompts, define four small deterministic contracts:

1. relay payload: the minimal source identity, object identity, event type, actor facts, and timestamps forwarded into ops
2. prediction snapshot: the durable record of what the workflow predicted and under which instruction version
3. correction review input: the deterministic diff artifact used by reporting and adaptation
4. write target contract: which repository receives evaluation writes before direct production writes are enabled

Discussion labeling, routing, moderation, prioritization, approvals, and summaries can all reuse this shape. The production object changes, but the CorrectionOps setup does not.

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
