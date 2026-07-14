---
title: A/B Experiments
description: Run A/B experiments in GitHub Agentic Workflows to test prompt variants and measure the effect of different instructions across runs.
sidebar:
  order: 7
---

:::caution[Experimental]
A/B Experiments is an experimental feature.
:::

Use the `experiments` frontmatter section to compare workflow variants across
repeated runs. Each experiment declares a name and a set of variants. On every
run, the activation job picks one variant and exposes it to the prompt.

Experiments work best when you test one workflow choice at a time, such as:

- prompt wording
- model selection
- whether to delegate to a sub-agent
- which subskill (inline skill) to invoke

## Declaring experiments

Add an `experiments` map to the workflow frontmatter. Each key names an
experiment. The value is either a simple array of variants (bare-array form) or
a rich object with additional metadata fields.

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

Use the object form when you want built-in reporting and experiment metadata:

```aw wrap
---
on:
  schedule: daily on weekdays

engine: copilot

evals:
  - id: focused
    question: Is the change focused and actionable?

experiments:
  prompt_style:
    variants: [concise, detailed]
    description: "Test whether a concise prompt reduces cost without quality loss"
    hypothesis: "H0: no change in aic. H1: concise reduces AIC by >=15%"
    metric: eval:focused
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

When `evals` are configured, `metric` can reference an eval question ID using
`eval:<id>` (for example `eval:focused`) or `evals.<id>`.

> [!NOTE]
> Experiment names must be valid identifiers: start with a letter or
> underscore, followed by letters, digits, or underscores. For example, use
> `style` or `feature_1`. Names that do not match this pattern are ignored.

## Using variants in the prompt

Reference a variant with `${{ experiments.<name> }}`. At runtime, gh-aw
replaces the expression with the selected variant string, such as `concise`.

Use the `{{#if experiments.<name> }}` block syntax for conditional prompt
sections. A variant value of `no` is treated as falsy, which makes yes/no
experiments easy to express:

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

## Common experiment ideas

Most experiments compare a single decision in the workflow. The examples below
show common patterns.

### Try different prompt styles

```aw wrap
---
experiments:
  style: [concise, detailed]
---

Summarize this issue in a **${{ experiments.style }}** way.
```

### Try different models

Model experiments are useful when you want to compare speed, cost, and output
quality. gh-aw model aliases such as `small` and `large` are often a good place
to start. See [Model Aliases](/gh-aw/reference/model-tables/).

```aw wrap
---
engine:
  id: copilot
  model: ${{ experiments.model }}

experiments:
  model: [small, large]
---

Review the issue and recommend the next action.
```

### Try using a sub-agent

This pattern compares a direct prompt with a delegated sub-agent flow.

```aw wrap
---
experiments:
  use_summarizer: [yes, no]
---

{{#if experiments.use_summarizer }}
Use the `file-summarizer` sub-agent to summarize `README.md`, then continue.
{{/if}}

Write a short project overview for maintainers.

## agent: `file-summarizer`
---
model: small
description: Summarizes a file in a few sentences
---
Read the given file and return a concise summary.
```

See [Inline Sub-Agents](/gh-aw/reference/inline-sub-agents/) for the full
syntax.

### Try different subskills

This pattern compares two reusable instruction blocks, sometimes called
subskills, without changing the main workflow prompt.

```aw wrap
---
experiments:
  triage_skill: [triage-fast, triage-deep]
---

Use the `${{ experiments.triage_skill }}` skill to classify this issue.

## skill: `triage-fast`
---
description: Fast issue triage
---
Classify the issue and suggest the smallest next step.

## skill: `triage-deep`
---
description: Detailed issue triage
---
Classify the issue, identify missing context, and recommend a fuller follow-up
plan.
```

## Statistical balancing

The activation job tracks how often each variant has been selected. The counter
is stored using the `storage` setting in the `experiments:` block. By default,
gh-aw chooses the least-used variant on each run. If multiple variants are tied,
including on the first run, one of them is chosen at random. Over time, this
keeps usage roughly balanced across variants.

When you provide a `weight` array, gh-aw uses weighted random selection instead
of least-used selection. For example, `[70, 30]` gives the first variant a 70%
selection probability. If `start_date` or `end_date` is set and the current
date falls outside that range, gh-aw returns the control variant (the first
entry) without incrementing any counter.

## Storage Configuration

The `storage` key inside the `experiments:` map controls where experiment state
is persisted:

```yaml
experiments:
  storage: repo   # or: cache (default: repo)
  prompt_style: [concise, detailed]
```

| Value | Behavior |
|---|---|
| `repo` (**default**) | Commits state to a git branch named `experiments/{sanitizedWorkflowID}` (workflow ID lowercased with hyphens removed, e.g. `my-workflow` → `experiments/myworkflow`). Durable — survives cache evictions. Requires `contents: write` permission (added automatically by the compiler). |
| `cache` | Uses GitHub Actions cache (legacy). State may be evicted after 7 days of inactivity. |

When `storage: repo`, the compiler adds a `push_experiments_state` job after the
activation job and commits the updated `state.json` to the experiments branch.

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

Each activation job writes a Markdown step summary that shows the selected
variants, cumulative counts, and, when you use the object form, progress toward
`min_samples`:

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
| `guardrail_metrics` | `object[]` | | List of guardrail objects with `name` (string), `threshold` (comparison string like `>=0.95` or bare number like `0.0`), and optional `direction` (`"min"` or `"max"`). When `threshold` is a bare number, `direction` governs the pass condition (≤ for `min`, ≥ for `max`). See [experiments-specification §4.4](/gh-aw/experimental/experiments-specification/#44-guardrail-metrics) for full semantics. |
| `min_samples` | `integer` | | Minimum runs per variant required before statistical analysis is considered reliable. The step summary shows a progress bar toward this target. |
| `weight` | `integer[]` | | Per-variant probability weights (same length as `variants`). Enables weighted-random selection; values are relative and need not sum to 100. |
| `issue` | `integer` | | GitHub issue number that tracks this experiment's lifecycle |
| `start_date` | `string` | | ISO-8601 date (`YYYY-MM-DD`) before which the experiment is inactive. The control variant is returned before this date without incrementing any counter. |
| `end_date` | `string` | | ISO-8601 date (`YYYY-MM-DD`) after which the experiment is inactive. The control variant is returned after this date without incrementing any counter. |
