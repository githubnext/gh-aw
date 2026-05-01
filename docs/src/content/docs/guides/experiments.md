---
title: A/B Experiments
description: Run A/B experiments in GitHub Agentic Workflows to test prompt variants and measure the effect of different instructions across runs.
sidebar:
  order: 7
---

The `experiments` section of the workflow frontmatter enables statistical A/B testing by defining named experiments, each with a set of variant values. At runtime the activation job selects one variant per experiment using a balanced round-robin counter and exposes the selection to the workflow prompt.

## Declaring experiments

Add an `experiments` map to the workflow frontmatter. Each key names an experiment; the value is an array of two or more variant strings.

```aw wrap
---
on:
  issues:
    types: [opened]
engine: copilot

experiments:
  style: [concise, detailed]
---

Summarize this issue in a **${{ experiments.style }}** way.
```

> [!NOTE]
> Experiment names must be valid identifiers: start with a letter or underscore, followed by letters, digits, or underscores (e.g. `style`, `feature_1`). Names that do not match this pattern are ignored.

## Using variants in the prompt

Reference a variant with `${{ experiments.<name> }}`. At runtime this is substituted with the selected variant string (e.g. `concise`).

Use the `{{#if experiments.<name> }}` block syntax for conditional prompt sections. A variant value of `no` is treated as falsy, enabling yes/no flag experiments:

```aw wrap
---
experiments:
  caveman: [yes, no]
---

{{#if experiments.caveman }}
Talk like a caveman in all your responses. Me test. You run.
{{/if}}

Address the issue described above.
```

## Statistical balancing

The activation job maintains a per-variant invocation counter in an `actions/cache` entry keyed by workflow ID. The variant with the lowest cumulative count is selected on each run; ties are broken by variant order. Over N runs every variant is used approximately N/K times (K = variant count), providing basic A/B balance with no configuration.

The counter persists across workflow runs via the GitHub Actions cache. A fresh repository starts from zero counts.

## Accessing assignments downstream

Each experiment exposes its selected variant as an activation job output:

| Expression | Description |
|---|---|
| `needs.activation.outputs.<name>` | Selected variant for experiment `<name>` |
| `needs.activation.outputs.experiments` | All assignments as a JSON object |

Use these expressions in downstream jobs defined in the `jobs:` frontmatter section.

## Analyzing results

The activation job uploads the counter state as an `experiment` artifact. Download and inspect it with the `gh aw` CLI:

```bash
# Download the experiment artifact for a specific run
gh aw audit <run-id> --artifacts experiment

# Display experiment assignments in the audit report
gh aw audit <run-id>
```

The `🧪 A/B Experiments` section of the audit report shows the variant chosen on the most recent run and the cumulative counts across all runs:

```
🧪 A/B Experiments
  • caveman = yes (cumulative: no:4, yes:5)
  • style = concise (cumulative: concise:5, detailed:4)
```

## Frontmatter reference

| Field | Type | Description |
|---|---|---|
| `experiments` | `object` | Map of experiment name → variant array |
| `experiments.<name>` | `string[]` | Array of two or more variant strings for one experiment |
