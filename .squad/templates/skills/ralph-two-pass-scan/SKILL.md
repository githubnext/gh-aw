---
name: "ralph-two-pass-scan"
description: "Cuts GitHub API calls by separating lightweight list scanning from full hydration"
domain: "work-monitoring"
confidence: "high"
source: "extracted"
---

# Skill: Ralph — Two-Pass Issue Scanning
**Confidence:** high
**Domain:** work-monitoring
**Last validated:** 2026-03-24

## Context
Cuts GitHub API calls from N+1 to ~7 per round (~72% reduction) by separating list scanning from full hydration.
Addresses the scanning inefficiency described in issue #596.

## Pattern

### Pass 1 — Lightweight Scan

```
gh issue list --state open --json number,title,labels,assignees --limit 100
```

**Skip hydration if ANY of these match:**

| Condition | Skip reason |
|-----------|-------------|
| `assignees` non-empty AND no `status:needs-review` | Already owned |
| Labels contain `status:blocked` or `status:waiting-external` | Externally gated |
| Labels contain `status:done` or `status:postponed` | Closed loop |
| Title matches stale/noisy pattern (`[chore]`, `[auto]`) | Low-signal |

### Pass 2 — Selective Hydration

For each issue surviving Pass 1:

```
gh issue view <number> --json number,title,body,labels,assignees,comments,state
```

Then apply normal Ralph triage logic. Rule of thumb: hydrate ≤ 30% of scanned list. If more than 30% survive Pass 1, tighten filter rules.
