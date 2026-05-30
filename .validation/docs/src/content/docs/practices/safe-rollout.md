---
title: Safe Rollout
description: Move from report-only or staged behavior to direct production writes with evidence and control.
---

Safe rollout is the practice of increasing workflow autonomy in steps instead of enabling direct production writes immediately.

The main question is not whether a workflow is useful, but whether it is trusted enough to act on the live system. In practice, teams usually move through a ladder: report-only first, then staged behavior, then a more realistic safe-write technique if needed, and finally direct production writes.

## Rollout Ladder

The usual progression is:

1. Start in report-only mode.
2. Enable `staged` behavior when proposed writes need to be previewed.
3. Use shadow evaluation when preview mode is not enough and the real write path needs to be exercised safely.
4. Promote the same workflow to direct production writes.

`staged` and shadow evaluation are not interchangeable. Staged mode is sufficient when the question is what the workflow would do. Shadow evaluation is needed when the question is whether the real write path behaves correctly on a safe non-production target.

## When Staged Is Enough

Use staged mode when the main risk is decision quality rather than operational behavior.

It is usually enough when maintainers only need to review proposed actions, compare alternatives, or inspect whether the workflow's judgment is reasonable before any write is allowed.

## When Shadow Evaluation Is Needed

Use shadow evaluation when staged mode is too weak because the real write path itself needs validation.

This is a good fit when:

- the workflow must update real target objects to prove the behavior is correct
- concurrency, deduplication, or serialization needs to be tested on a live-like surface
- maintainers need to inspect the actual produced state, not only proposed intent
- cross-repository writes, permissions, or dispatch boundaries need to be exercised safely

Shadow evaluation is one technique inside safe rollout, not a separate top-level pattern.

## Design Rules

### Production truth stays authoritative

Do not let the evaluation surface become the new source of truth. Production events and later trusted human actions should remain authoritative.

### Prediction snapshots should be explicit

If later comparison matters, persist what the workflow predicted at decision time. Do not reconstruct predictions from logs.

### Correction evidence needs provenance

Not every later edit should count as trustworthy truth. Record provenance such as actor type, manual versus automated source, trust status, and origin repository role.

### Evaluation surfaces should remain disposable

Keep the shadow target thin. It should support measurement and rollout, not become a second long-lived control plane.

## Example Shape

The common repository split is:

- production repository: emits live events and contains authoritative later human truth
- ops repository: persists predictions, collects corrections, publishes reports, and updates instructions
- shadow repository: temporary non-production write target during rollout

That shape is often useful, but it is still rollout guidance rather than a primary pattern.

## Related Documentation

- [MultiRepoOps](/gh-aw/patterns/multi-repo-ops/)
- [MultiRepoOps](/gh-aw/patterns/multi-repo-ops/)
- [Staged Mode](/gh-aw/reference/staged-mode/)
- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/)
