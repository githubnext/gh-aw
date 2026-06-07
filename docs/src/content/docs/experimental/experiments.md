---
title: A/B Experiments
description: Run A/B experiments in GitHub Agentic Workflows to test prompt variants and measure the effect of different instructions across runs.
sidebar:
  order: 7
---

:::caution[Experimental]
A/B Experiments is an experimental feature.
:::

The `experiments` section of the workflow frontmatter enables statistical A/B testing by defining named experiments, each with a set of variant values. At runtime the activation job selects one variant per experiment using a balanced round-robin counter and exposes the selection to the workflow prompt.

## Declaring experiments

Add an `experiments` map to the workflow frontmatter. Each key names an experiment; the value is either a simple array of variants (bare-array form) or a rich object with additional metadata fields.

### Bare-array form

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

### Rich object form

Use the object form to attach metadata that drives automated reporting, guardrail enforcement, and lifecycle tracking:

```aw wrap
---
on:
  schedule: daily on weekdays

engine: copilot

experiments:
  prompt_style:
    variants: [concise, detailed]
    description: "Test whether a concise prompt reduces cost without quality loss"
    hypothesis: "H0: no change in aic. H1: concise reduces AIC by >=15%"
    metric: aic
    secondary_metrics: [duration_ms, discussion_word_count]
    guardrail_metrics:
      - name: success_rate
        threshold: ">=0.95"
      - name: empty_output_rate
        direction: min
        threshold: 0.0
    weight: [50, 50]
    min_samples: 25
    start_date: "2026-05-05"
    end_date: "2026-07-25"
    issue: 1234
---

Summarize the findings in a **${{ experiments.prompt_style }}** way.
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

The activation job maintains a per-variant invocation counter that is persisted according to the `storage` setting in the `experiments:` block (see [Storage Configuration](#storage-configuration) below). The variant with the lowest cumulative count is selected on each run; when multiple variants share the lowest count (including the very first run when state is empty), one is chosen at random so no variant is systematically favoured. Over N runs every variant is used approximately N/K times (K = variant count), providing basic A/B balance with no configuration.

When a `weight` array is provided, weighted-random selection is used instead of round-robin. Each variant is chosen with probability proportional to its weight (e.g. `[70, 30]` gives the first variant a 70% probability). When `start_date` or `end_date` is set and today falls outside the window, the control variant (first entry) is returned without incrementing any counter.

## Storage Configuration

The `storage` key inside the `experiments:` map controls how experiment state is persisted:

```yaml
experiments:
  storage: repo   # or: cache (default: repo)
  prompt_style: [concise, detailed]
```

| Value | Behavior |
|---|---|
| `repo` (**default**) | Commits state to a git branch named `experiments/{sanitizedWorkflowID}` (workflow ID lowercased with hyphens removed, e.g. `my-workflow` → `experiments/myworkflow`). Durable — survives cache evictions. Requires `contents: write` permission (added automatically by the compiler). |
| `cache` | Uses GitHub Actions cache (legacy). State may be evicted after 7 days of inactivity. |

When `storage: repo`, the compiler adds a `push_experiments_state` job that runs after the activation job and commits the updated `state.json` to the experiments branch.

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

### Filtering audit results by variant

Use `--experiment` and `--variant` to filter audit runs to a specific variant:

```bash
gh aw audit <run-id> --experiment prompt_style --variant concise
```

### Step summary

Each activation job writes a Markdown step summary that shows variant assignments, cumulative counts, and — when the rich object form is used — progress toward `min_samples`:

```
## 🧪 A/B Experiment Assignments

| Experiment   | Selected Variant | All Variants      | Cumulative Counts      |
| ---          | ---              | ---               | ---                    |
| prompt_style | concise          | concise, detailed | concise: 8, detailed: 7|

### 📊 Sampling Progress

prompt_style (target: 25 per variant)
  concise: ████████░░░░░░░░░░░░ 8/25 (32%)
  detailed: ███████░░░░░░░░░░░░░ 7/25 (28%)

### Experiment Details

**prompt_style**

> Test whether a concise prompt reduces cost without quality loss

**Hypothesis:** H0: no change in aic. H1: concise reduces AIC by >=15%

**Guardrail metrics:**
- `success_rate` >=0.95
- `empty_output_rate` ==0

Tracking issue: [#1234](https://github.com/owner/repo/issues/1234)
```

## Frontmatter reference

### Bare-array form

| Field | Type | Description |
|---|---|---|
| `experiments` | `object` | Map of experiment name → variant array or config object |
| `experiments.<name>` | `string[]` | Array of two or more variant strings for one experiment |

### Object form fields

| Field | Type | Required | Description |
|---|---|---|---|
| `variants` | `string[]` | ✅ | Array of two or more variant strings |
| `description` | `string` | | Human-readable explanation of what the experiment tests |
| `hypothesis` | `string` | | Null and alternative hypothesis (e.g. `"H0: no change. H1: concise reduces AIC by >=15%"`) |
| `metric` | `string` | | Primary metric to observe (e.g. `aic`, `duration_ms`) |
| `secondary_metrics` | `string[]` | | Additional metrics to track alongside the primary metric |
| `guardrail_metrics` | `object[]` | | List of guardrail objects with `name` (string), `threshold` (comparison string like `>=0.95` or bare number like `0.0`), and optional `direction` (`"min"` or `"max"`). When `threshold` is a bare number, `direction` governs the pass condition (≤ for `min`, ≥ for `max`). See [experiments-specification §4.4](./experiments-specification#44-guardrail-metrics) for full semantics. |
| `min_samples` | `integer` | | Minimum runs per variant required before statistical analysis is considered reliable. The step summary shows a progress bar toward this target. |
| `weight` | `integer[]` | | Per-variant probability weights (same length as `variants`). Enables weighted-random selection; values are relative and need not sum to 100. |
| `issue` | `integer` | | GitHub issue number that tracks this experiment's lifecycle |
| `start_date` | `string` | | ISO-8601 date (`YYYY-MM-DD`) before which the experiment is inactive. The control variant is returned before this date without incrementing any counter. |
| `end_date` | `string` | | ISO-8601 date (`YYYY-MM-DD`) after which the experiment is inactive. The control variant is returned after this date without incrementing any counter. |

