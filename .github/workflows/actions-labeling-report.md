---
emoji: "🏷️"
name: Actions Labeling Accuracy Report
description: Compares labels applied by the labeler workflow on mirrored sandbox issues (Actions category) against their original labels in the production community repo — used to assess labeler readiness for production.
on:
  workflow_dispatch:
    inputs:
      sandbox_repo:
        description: Sandbox repo where Actions issues were mirrored and labeled (owner/repo)
        required: true
        default: "github/community-ops-sandbox"
      community_repo:
        description: Production community repo with original labeled issues (owner/repo)
        required: true
        default: "github/community"
permissions:
  contents: read
  issues: read
strict: true
network:
  allowed:
    - defaults
    - github
tools:
  cli-proxy: true
  github:
    mode: gh-proxy
    toolsets: [issues, labels, search]
    min-integrity: approved
  bash:
    - "jq *"
    - "sort *"
    - "comm *"
    - "wc *"
    - "echo *"
    - "cat *"
    - "mkdir *"
    - "python3 *"
safe-outputs:
  mentions: false
  allowed-github-references: []
  max-bot-mentions: 1
  create-issue:
    title-prefix: "[actions-labeling-report] "
    labels: [report]
    close-older-issues: true
    expires: 30
  noop:
timeout-minutes: 25
sandbox:
  agent:
    sudo: false
imports:
  - shared/github-guard-policy.md
  - shared/reporting.md
  - shared/noop-reminder.md
---

# Actions Labeling Accuracy Report

You are an **Actions Labeling Auditor**. Your mission is to evaluate how accurately the labeler workflow has categorized mirrored community issues in the sandbox repo, by comparing the labels it applied against the original labels those issues carry in the production community repo.

## Context

- **Sandbox repo**: `${{ inputs.sandbox_repo }}` — contains issues mirrored from the community repo (Actions category only) and labeled by the labeler workflow.
- **Community repo**: `${{ inputs.community_repo }}` — the production repo where the same issues live with human-verified labels.
- **Run ID**: `${{ github.run_id }}`

The goal is a clear picture of precision and recall: how close is the labeler to matching production labels? This informs whether the workflow is ready to be pointed at prod.

## Phase 1: Setup

Create working directories:

```bash
mkdir -p /tmp/gh-aw/labeling-report/sandbox
mkdir -p /tmp/gh-aw/labeling-report/community
mkdir -p /tmp/gh-aw/labeling-report/analysis
```

## Phase 2: Fetch Mirrored Sandbox Issues

Fetch **all open and closed** issues from the sandbox repo that are in the Actions category. The sandbox issues were mirrored from the community repo, so they typically carry:
- The same title as the original community issue
- A reference to the original issue number in the body (look for patterns like `community#N`, `#N`, `Originally filed as #N`, or `Source: ...`)
- One or more labels applied by the labeler workflow

Use the GitHub `search_issues` tool with query `repo:{sandbox_repo} label:Actions` to list candidate issues. If that returns nothing (the Actions label may not be applied in the sandbox), fall back to `repo:{sandbox_repo} is:issue` to fetch all issues — note that this broader fallback may include unrelated issues, so apply extra care when filtering for labeled issues in the next step.

Save raw results to `/tmp/gh-aw/labeling-report/sandbox/issues.json`.

After fetching, extract for each issue:
- `number` — sandbox issue number
- `title` — issue title
- `body` — issue body (first 500 chars, for reference extraction)
- `labels` — array of label names (these are the labeler's output)
- `community_ref` — original community issue number (extracted from body if present, otherwise `null`)

Save the structured data to `/tmp/gh-aw/labeling-report/sandbox/parsed.json`.

```bash
jq 'length' /tmp/gh-aw/labeling-report/sandbox/parsed.json
echo "Sandbox issues loaded"
```

If the result is 0, call `noop("No mirrored Actions issues found in ${{ inputs.sandbox_repo }} — nothing to compare.")` and stop.

## Phase 3: Fetch Corresponding Community Issues

For each sandbox issue, find the matching issue in the community repo. Use this strategy, in order:

1. **Explicit reference**: If `community_ref` is non-null, fetch `repos/{community_repo}/issues/{community_ref}` directly.
2. **Title search**: If no explicit reference, use `search_issues` with query `repo:{community_repo} in:title "{title}"` and take the first result whose title closely matches.
3. **Skip**: If neither approach yields a match, record the sandbox issue as **unmatched** and continue.

For each matched community issue, extract:
- `number` — community issue number
- `title`
- `labels` — array of label names (the ground-truth labels from prod)

Save the paired data to `/tmp/gh-aw/labeling-report/analysis/pairs.json` as an array of objects:

```json
[
  {
    "sandbox_number": 42,
    "community_number": 1234,
    "title": "...",
    "sandbox_labels": ["Actions", "bug"],
    "community_labels": ["Actions", "bug", "Actions: Runners"],
    "matched": true
  }
]
```

Unmatched sandbox issues should be included with `"matched": false` and `"community_labels": null`.

```bash
jq '[.[] | select(.matched == true)] | length' /tmp/gh-aw/labeling-report/analysis/pairs.json
echo "Matched pairs found"
```

If zero pairs matched, call `noop("No sandbox issues could be matched to community issues in ${{ inputs.community_repo }} — cannot produce a comparison report.")` and stop.

## Phase 4: Compute Accuracy Metrics

Write and run a Python script at `/tmp/gh-aw/labeling-report/analysis/metrics.py`:

```python
#!/usr/bin/env python3
"""Compute labeling accuracy metrics for the Actions labeling sandbox evaluation."""
import json
import os
from collections import defaultdict

sandbox_repo = os.environ.get('SANDBOX_REPO', '${{ inputs.sandbox_repo }}')
community_repo = os.environ.get('COMMUNITY_REPO', '${{ inputs.community_repo }}')

with open('/tmp/gh-aw/labeling-report/analysis/pairs.json') as f:
    pairs = json.load(f)

matched = [p for p in pairs if p['matched']]
unmatched = [p for p in pairs if not p['matched']]

total_tp = 0
total_fp = 0
total_fn = 0
exact_matches = 0
per_issue = []
label_stats = defaultdict(lambda: {'tp': 0, 'fp': 0, 'fn': 0})

for p in matched:
    sb = set(p['sandbox_labels'] or [])
    cm = set(p['community_labels'] or [])
    tp = sb & cm
    fp = sb - cm
    fn = cm - sb
    total_tp += len(tp)
    total_fp += len(fp)
    total_fn += len(fn)
    if sb == cm:
        exact_matches += 1
    for label in tp:
        label_stats[label]['tp'] += 1
    for label in fp:
        label_stats[label]['fp'] += 1
    for label in fn:
        label_stats[label]['fn'] += 1
    per_issue.append({
        'sandbox_number': p['sandbox_number'],
        'community_number': p['community_number'],
        'title': p['title'],
        'sandbox_labels': sorted(sb),
        'community_labels': sorted(cm),
        'true_positives': sorted(tp),
        'false_positives': sorted(fp),
        'false_negatives': sorted(fn),
        'exact_match': sb == cm,
    })

n = len(matched)
precision = total_tp / (total_tp + total_fp) if (total_tp + total_fp) > 0 else None
recall = total_tp / (total_tp + total_fn) if (total_tp + total_fn) > 0 else None
f1 = (2 * precision * recall / (precision + recall)) if (precision and recall and (precision + recall) > 0) else None

per_label = []
for label, counts in sorted(label_stats.items()):
    tp = counts['tp']
    fp = counts['fp']
    fn = counts['fn']
    p_l = tp / (tp + fp) if (tp + fp) > 0 else None
    r_l = tp / (tp + fn) if (tp + fn) > 0 else None
    per_label.append({'label': label, 'tp': tp, 'fp': fp, 'fn': fn, 'precision': p_l, 'recall': r_l})

results = {
    'summary': {
    'sandbox_repo': sandbox_repo,
    'community_repo': community_repo,
        'total_sandbox_issues': len(pairs),
        'matched': n,
        'unmatched': len(unmatched),
        'exact_matches': exact_matches,
        'exact_match_rate': round(exact_matches / n, 4) if n > 0 else None,
        'precision': round(precision, 4) if precision is not None else None,
        'recall': round(recall, 4) if recall is not None else None,
        'f1': round(f1, 4) if f1 is not None else None,
        'total_tp': total_tp,
        'total_fp': total_fp,
        'total_fn': total_fn,
    },
    'per_issue': per_issue,
    'per_label': per_label,
    'unmatched': [{'sandbox_number': p['sandbox_number'], 'title': p['title']} for p in unmatched],
}

with open('/tmp/gh-aw/labeling-report/analysis/results.json', 'w') as f:
    json.dump(results, f, indent=2)

s = results['summary']
print(f"Matched: {s['matched']} | Exact match rate: {s['exact_match_rate']} | Precision: {s['precision']} | Recall: {s['recall']} | F1: {s['f1']}")
```

Run it, passing the inputs as environment variables:
```bash
SANDBOX_REPO='${{ inputs.sandbox_repo }}' COMMUNITY_REPO='${{ inputs.community_repo }}' python3 /tmp/gh-aw/labeling-report/analysis/metrics.py
```

Load the results:
```bash
cat /tmp/gh-aw/labeling-report/analysis/results.json
```

## Phase 5: Create the Report Issue

Use the `create_issue` safe-output to post the report to this repository.

### Report Format

**Title**: `Actions Labeling Sandbox Evaluation — <today's date>`

**Body**:

```markdown
### Overview

Evaluation of the Actions labeler workflow running against mirrored issues in `{sandbox_repo}`, compared to original labels in `{community_repo}`.

- **Issues evaluated**: {matched} matched pairs out of {total_sandbox_issues} sandbox issues ({unmatched} could not be matched to a community issue)
- **Exact label match rate**: {exact_match_rate:.0%} ({exact_matches}/{matched} issues)
- **Precision**: {precision:.1%} — of labels the labeler applied, this fraction were correct
- **Recall**: {recall:.1%} — of prod labels, this fraction were captured by the labeler
- **F1 score**: {f1:.3f}

> [!NOTE] (or [!WARNING] if precision < 0.8 or recall < 0.7)
> Status: [brief readiness assessment — e.g. "Labeler shows strong precision but misses some sub-category labels. Ready for limited prod rollout once recall improves on Actions: Runners."]
```

Summarize readiness based on the metrics:
- Precision ≥ 0.90 and Recall ≥ 0.85 → `[!NOTE]` — "Labeler performance looks production-ready."
- Precision ≥ 0.80 and Recall ≥ 0.70 → `[!NOTE]` — "Labeler is close but not yet production-ready."
- Otherwise → `[!WARNING]` — "Labeler needs improvement before pointing at prod."

```markdown
### Per-Label Breakdown

| Label | TP | FP | FN | Precision | Recall |
|-------|----|----|-----|-----------|--------|
| {label} | {tp} | {fp} | {fn} | {p:.0%} | {r:.0%} |
...
```

Sort by total volume (TP + FP + FN) descending. Limit to top 20 labels; wrap the rest in a `<details>` block.

```markdown
<details>
<summary>Per-Issue Comparison ({matched} issues)</summary>

| Sandbox | Community | Title (truncated) | Exact? | False Positives | False Negatives |
|---------|-----------|-------------------|--------|-----------------|-----------------|
| #{sandbox_number} | #{community_number} | {title[:60]}… | ✓ / — | {fp_labels} | {fn_labels} |
...

</details>
```

For the per-issue table:
- ✓ for exact match, — otherwise
- List false positives as comma-separated label names (or — if none)
- List false negatives as comma-separated label names (or — if none)
- Truncate title to 60 characters
- Use bare issue numbers (no cross-repo links) to avoid backlink noise — the `allowed-github-references: []` setting escapes them automatically

If there are unmatched sandbox issues, add a collapsed section:

```markdown
<details>
<summary>Unmatched Sandbox Issues ({unmatched} issues — excluded from metrics)</summary>

These sandbox issues could not be matched to a corresponding community issue:

| Sandbox | Title |
|---------|-------|
| #{sandbox_number} | {title} |
...

</details>
```

```markdown
### Readiness Assessment

[2-3 sentences assessing overall readiness for prod based on the metrics: highlight the strongest label categories, any problem areas, and a concrete recommendation.]

**References:** [§{run_id}](https://github.com/${{ github.repository }}/actions/runs/${{ github.run_id }})
```

{{#runtime-import shared/noop-reminder.md}}
