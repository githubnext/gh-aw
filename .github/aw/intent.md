---
description: Design-time guidance for deriving an outcome-oriented workflow intent and using it to select, implement, and evaluate an agentic workflow.
---

# Intent-Driven Workflow Design

Start every workflow design by extracting a concise, implementation-independent outcome. Persist that canonical outcome in the top-level `intent:` frontmatter field; use the richer analysis below only while designing the workflow.

```yaml
intent: Reduce maintainer effort spent identifying recurring actionable CI regressions without generating duplicate work.
```

An intent describes the outcome for an actor and subject, not a trigger, tool, schedule, or write action. It must remain valid if the implementation changes from an immediate issue to a weekly digest.

## Derive an IntentSpec

Before selecting implementation details, derive this transient model:

```text
IntentSpec
- intent: concise canonical outcome
- actors and subject: who benefits and what is affected
- activation conditions: facts that make the intent relevant
- required context: evidence needed to make a decision
- required effects: observable results that satisfy the outcome
- noop conditions: inverse cases that must not create attention or writes
- success conditions: how the design avoids attention cost while producing value
- uncertainties: policy or evidence gaps that need a conservative default or clarification
```

Do not serialize this structure in workflow frontmatter. The `intent:` value is the only persisted part. Do not duplicate executable configuration such as `on:`, `tools:`, `permissions:`, `safe-outputs:`, or schedules in it.

For an explicit, narrow request, infer the obvious intent and keep this pass lightweight. For an underspecified request or one asking what to automate, collect bounded repository evidence first, then propose evidence-backed candidate intents with their evidence, feasibility, expected value, risk, and uncertainties. Do not perform a broad survey when it cannot materially change a clear request.

## Design from the IntentSpec

Use the model to derive implementation rather than mapping the request directly to a trigger:

1. Compare plausible architectures against intent coverage, timeliness, attention cost, safety, boundedness, determinism, state requirements, implementation complexity, and available evidence.
2. Select the trigger, schedule, data collection, tools, permissions, safe outputs, and deduplication strategy that satisfy the selected architecture.
3. Put the activation conditions, required effects, evidence threshold, and no-op conditions in the prompt body. Require `noop` when a counter-case applies or evidence is insufficient.
4. Make duplicate detection, filters, output caps, and previous-result strategy enforce the same conditions where configuration can do so.

For example, an intent to surface actionable CI regressions can require completed relevant CI, actionable and novel evidence, and sufficient diagnostics. Known flakes, infrastructure failures, already-tracked regressions, closed pull requests, and insufficient evidence are counter-cases. An immediate incident, PR comment, daily digest, and weekly trend report are alternative architectures; choose the one that best meets the intent without unnecessary attention cost.

## Derive Evals from Intent

Create representative positive scenarios from required effects and adversarial scenarios from no-op conditions. Turn each scenario into one observable, falsifiable YES/NO evaluation question about the agent output:

- a novel, sufficiently evidenced actionable case produces the intended visible result;
- a duplicate or known benign case produces no visible write;
- an uncertain case investigates when appropriate but does not write.

Keep eval questions binary and output-observable. Do not ask a judge whether the intent itself is good or whether the agent made sufficient effort.

## Preserve Intent on Updates

Read the existing `intent:` before changing an existing workflow. Preserve it for an implementation-only change, including a trigger or output-channel redesign. Reconsider and update it only when the request materially expands, contracts, or otherwise changes the outcome. When it changes, re-derive conditions, architecture, prompt behavior, and evals.
