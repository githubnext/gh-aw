---
title: CorrectionOps
description: Improve agentic workflows from trusted human corrections without retraining the underlying model
---

:::caution[Experimental]
CorrectionOps is an experimental pattern. The guidance and workflow shape on this page may change as the pattern is tested in more real-world workflows.
:::

CorrectionOps is a workflow pattern that compares predictions with later human corrections.

Instead of retraining the model, CorrectionOps improves the workflow around the model. It stores predictions at decision time, compares them with later trusted human truth, and uses that evidence to update instructions, routing, thresholds, and rollout decisions.

The basic loop is simple:

1. Save what the workflow predicted
2. Collect what humans later decided
3. Use the difference to improve the workflow

## When to Use CorrectionOps

Use CorrectionOps when you want to turn a human decision process into an agentic workflow iteratively rather than all at once.

It is a good fit when humans still make or correct the real decision, but you want the workflow to improve over time by updating instructions, routing, thresholds, or rollout state.

Typical fits include labeling and classification, routing and prioritization, moderation and approvals, and summaries or recommendations that humans later correct.

It is especially useful when the rollout path is gradual:

- Start with `staged: true`
- Keep evaluation and reporting in Ops
- Use later corrections to improve the workflow
- Promote to direct writes only when the evidence is strong enough

## How It Works

A clean CorrectionOps setup has two long-lived surfaces. Production stays authoritative. Ops is the long-lived home for prediction, correction intake, reporting, instruction updates, and rollout control.

That means the workflows usually stay in Ops. Early on they report, compare, and adapt from Ops without writing back to production. After promotion they can write directly to production.

Most implementations reduce to three workflow classes: a thin relay that forwards stable facts into ops, a prediction workflow that persists snapshots and writes safely, and a compare/report/decide workflow that checks later human truth and updates the system when the evidence is strong enough.

The important rule is to keep relays, snapshot resolution, diffing, and grouping deterministic. Use the agent for semantic judgment, not for reconstructing event history or inferring provenance after the fact.

## Example: Issue Labeling

```mermaid
flowchart TB
  subgraph ProductionRepo[Production Repo]
    A[Issue or item in production]
    D[Later human correction in production]
    B[Thin relay]
  end

  subgraph OpsRepo[Ops Repo]
    C[Store prediction snapshot]
    E[Collect correction evidence]
    F[Build deterministic diff]
    G[Publish report or open instruction PR]
    H[Make rollout decision]
  end

  A -->|item-created event| B
  B --> C
  D -->|truth-feedback event| E
  C --> F
  E --> F
  F --> G
  G --> H
  H -.->|improves next run| A
```

In this shape, production stays authoritative. Ops records the original prediction, collects later human corrections, builds the diff, and decides whether the workflow should stay staged, update its instructions, or graduate to direct writes.

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

CorrectionOps solves a different problem than model training. Reinforcement Learning from Human Feedback (RLHF) updates model weights from human feedback. CorrectionOps updates the workflow system *around* the model. In practice that usually means changing instruction files, routing rules, deterministic checks, thresholds, or rollout decisions rather than trying to retrain the engine.

In a healthy CorrectionOps loop, production truth stays authoritative, predictions are saved explicitly, corrections include provenance, and diffs are built deterministically before the agent is asked to reason about them.

CorrectionOps does not require a separate evaluation repository. The normal progression is to start with `staged: true`, then use ops-managed adaptation and gated review, then enable direct production writes once the evidence is strong enough.

### Full Workflow Pieces

If you want the explicit workflow split, the same example usually breaks into four pieces.

#### 1. Relay In The Source Repo

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

#### 2. Prediction Workflow In Ops

The prediction workflow consumes normalized inputs, applies the current instructions, and persists a durable snapshot that can be compared later.

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
---

# Predict Items

Read prepared items from `/tmp/gh-aw/agent/item-scan`, apply the current instructions, write review artifacts through safe outputs in Ops, and append a prediction snapshot containing the source identifier, predicted action, instruction version, and timestamp.
```

#### 3. Compare, Report, And Decide In Ops

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

#### 4. Optional Deterministic Collector

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

### Stable Contracts To Define First

Before adding rollout logic or adaptation prompts, define four small deterministic contracts:

1. relay payload: the minimal source identity, object identity, event type, actor facts, and timestamps forwarded into ops
2. prediction snapshot: the durable record of what the workflow predicted and under which instruction version
3. correction review input: the deterministic diff artifact used by reporting and adaptation
4. rollout gate contract: what evidence or approvals are required before direct production writes are enabled

Discussion labeling, routing, moderation, prioritization, approvals, and summaries can all reuse this shape. The production object changes, but the CorrectionOps setup does not.

## Related Documentation

- [Staged Mode](/gh-aw/reference/staged-mode/) for the optional safe-write rollout guidance inside CorrectionOps
- [SideRepoOps](/gh-aw/patterns/side-repo-ops/) for separating workflow infrastructure from the production repository
- [MultiRepoOps](/gh-aw/patterns/multi-repo-ops/) for coordinating workflows across repository boundaries
- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) for controlling write targets and protections
- [GitHub Tools](/gh-aw/reference/github-tools/) for cross-repository reads and operations
