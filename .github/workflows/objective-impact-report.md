---
emoji: 📊
description: Objective impact report from closed prioritized issues.
on:
  workflow_dispatch:
permissions:
  issues: read
safe-outputs:
  create-issue:
    max: 1
---

# Objective Impact Report

## Goal

Test whether prioritized GitHub issues are a useful proxy for objective value.

Use the simplest possible model:

```text
Issue = objective
Priority or severity = value
Closed issue = delivered value
```

Do not infer complex objectives.
Do not allocate value to workflows.
Do not score labels except priority or severity.
Do not use an LLM judge.

## Scope

Analyze issues from the last 180 days.

Ignore pull requests.

## Value mapping

Use the first matching priority or severity signal from issue labels or fields.

```text
P0 / urgent / critical = 100
P1 / high              = 50
P2 / medium            = 20
P3 / low               = 5
unknown                = 1
```

Recognize common label forms case-insensitively:

```text
P0, priority:P0, priority/P0, severity:critical, critical, urgent
P1, priority:P1, priority/P1, severity:high, high
P2, priority:P2, priority/P2, severity:medium, medium
P3, priority:P3, priority/P3, severity:low, low
```

All other labels are classification only.

## Computation

Compute:

```text
Delivered Impact = sum(value of closed issues)
Remaining Impact = sum(value of open issues)
Total Impact     = Delivered Impact + Remaining Impact
Completion       = Delivered Impact / Total Impact
```

If no prioritized issues exist, use `unknown = 1` and explain that the repository needs priority or severity signals for a meaningful test.

## Report

Create one issue titled:

```text
Objective Impact Report - YYYY-MM-DD
```

The report must include:

### Summary

- Issues analyzed
- Closed issues
- Open issues
- Delivered Impact
- Remaining Impact
- Completion percentage

### Top closed issues by value

| Issue | Value signal | Value | Labels |
|---|---:|---:|---|

### Top open issues by value

| Issue | Value signal | Value | Labels |
|---|---:|---:|---|

### Interpretation

Explain whether the result looks useful for maintainers and PMs.

### Data quality

Mention whether priority/severity labels were common or missing.

## Safe output

Use only `create-issue`.

If a report for today already exists, do nothing.
