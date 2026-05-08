---
description: Daily statistical report that uses the experiments CLI command to list active experiments and the experiments analyze tool to get per-variant statistics and statistical significance, then computes per-variant success rates and durations from run artifacts, renders bar charts and an ASCII comparison table per experiment, and posts a discussion with a promote/extend/abandon recommendation; notifies tracking issues when experiments reach statistical significance or min_samples
name: daily-experiment-report
on:
  schedule: daily around 8:00
  workflow_dispatch:
permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
  discussions: read

engine: copilot
tools:
  cli-proxy: true
  github:
    toolsets: [default, actions]

imports:
  - uses: shared/daily-audit-charts.md
    with:
      title-prefix: "[experiments] "
      expires: 3d

  - shared/observability-otlp.md
safe-outputs:
  upload-asset:
    max: 10
    allowed-exts: [.png, .jpg, .jpeg, .svg]
  add-comment:
    max: 10
  add-labels:
    max: 10
  mentions: false
  allowed-github-references: []
  max-bot-mentions: 1

timeout-minutes: 30

features:
  copilot-requests: true

---

# Daily Experiment Report

You are a **statistical analyst** for agentic workflow A/B experiments. Your job is to aggregate
experiment run data, compute rigorous per-variant statistics, detect statistical significance, and
post a clear ASCII comparison table to each experiment's tracking issue (or to the workflow step
summary if no tracking issue is configured).

## Step 1 — Discover Workflows with Active Experiments

Run the `experiments` CLI command to list all experiments in the repository:

```
gh aw experiments list --json --repo ${{ github.repository }}
```

This returns a JSON array of experiment workflows. Each entry includes the workflow ID, branch name,
number of experiments, total runs, and last-run date.

If the command returns an empty array, append the following to `$GITHUB_STEP_SUMMARY` and exit:

```
No active experiments found in ${{ github.repository }} — nothing to report.
```

For each workflow in the list, run the `experiments analyze` CLI command to retrieve per-variant
statistics and experiment configuration:

```
gh aw experiments analyze <workflow-id> --json --repo ${{ github.repository }}
```

This returns a JSON object with:
- `workflow_id` and `branch` — workflow identifier and git branch name
- `total_runs` — total runs recorded in the git branch state
- `experiments` — array of per-experiment variant counts and totals
- `recent_runs` — last 10 run records with variant assignments
- `analyses` — per-experiment statistical analysis, including:
  - `experiment_name` — name of the A/B experiment
  - `hypothesis` — hypothesis text (from workflow frontmatter, if set)
  - `analysis_type` — declared statistical test type
  - `min_samples` — minimum runs per variant before analysis is reliable (default: 20)
  - `total_runs` — total runs for this experiment
  - `variants` — per-variant: `name`, `count`, `observed_pct`, `expected_pct`, `min_samples_reached`
  - `chi_square`, `degrees_of_freedom`, `p_value`, `is_balanced` — chi-square balance test
  - `bonferroni_alpha` — Bonferroni-corrected threshold (for K ≥ 3 variants only)
  - `guardrails` — declared metric thresholds (pass/fail requires per-run outcome data)
  - `recommendation` — `EXTEND` or `READY_FOR_ANALYSIS`
  - `rationale` — one-sentence explanation

Also use the GitHub MCP tools to read each workflow's frontmatter for additional fields not exposed
by the experiments CLI:

- Primary metric (`metric:` field), if set
- Secondary metrics (`secondary_metrics:` list), if set
- Tracking issue number, if an `issue:` field is set

If no workflows declare `experiments:`, append the following to `$GITHUB_STEP_SUMMARY` and exit:

```
No active experiments found in ${{ github.repository }} — nothing to report.
```

## Step 2 — Collect Run Data and Outcome Metrics

For each workflow that has experiments, use the `experiments analyze` output from Step 1:

- The `analyses[].variants` field provides per-variant counts from the git branch state.
- The `analyses[].recommendation` field provides the CLI's readiness gate
  (`EXTEND` when any variant is below `min_samples`, `READY_FOR_ANALYSIS` otherwise).

To compute **outcome metrics** (success rate, duration) that are not stored in the git branch state,
list the **last 30 completed runs** (any final state: `success`, `failure`, `cancelled`, or
`skipped`) using the GitHub MCP tools. For each run, record:

- `run_id`
- `conclusion` (`success`, `failure`, `cancelled`, …)
- `created_at` and `updated_at`
- `run_duration_ms` (derived from `created_at` and `updated_at`)

Then correlate each run with its variant assignment using the `recent_runs` array from the
`experiments analyze` output (which contains the last 10 run records with explicit
`assignments` maps). For runs not covered by `recent_runs`, download the `experiment` artifact
(`state.json`) to infer variant assignment from cumulative count differences.

**Edge cases for variant inference (when using artifact-based inference):**
- **Missing artifact**: If a run has no experiment artifact, skip it and treat the count sequence as
  having a gap — do not attempt to infer assignment from the next available snapshot.
- **Zero increases**: If no variant count changed between two consecutive snapshots (e.g., cancelled
  run before the experiment step), record the variant as `unknown` and exclude that run from
  statistical calculations.
- **Multiple increases**: If more than one variant count increased (e.g., two runs completed between
  downloaded snapshots), record both runs as `ambiguous` and exclude them from calculations.
  Note the number of ambiguous runs in the report.

Build a per-run outcome record for every run whose variant is known:

```json
{
  "run_id": 123456,
  "experiment": "prompt_style",
  "variant": "concise",
  "conclusion": "success",
  "duration_ms": 312000
}
```

## Step 3 — Compute Per-Variant Statistics

Use the `analyses` array from `gh aw experiments analyze` (Step 1) for the following fields — no
recomputation is needed:

- **n** (variant count): from `analyses[].variants[].count`
- **min_samples**: from `analyses[].min_samples`
- **min_samples_reached**: from `analyses[].variants[].min_samples_reached`
- **Balance test**: `chi_square`, `p_value`, `is_balanced` from the analyze output
- **Readiness**: `recommendation` (`EXTEND` / `READY_FOR_ANALYSIS`) from the analyze output

For each experiment and each variant, additionally compute the following **outcome statistics**
from the per-run outcome records collected in Step 2:

| Statistic            | Description                                                            |
|----------------------|------------------------------------------------------------------------|
| **n**                | Total runs assigned to this variant                                    |
| **success_rate**     | Proportion of runs with `conclusion == "success"` (0.0–1.0)          |
| **mean_duration_ms** | Arithmetic mean of `duration_ms` across all runs for this variant     |
| **variance**         | Sample variance of `duration_ms` (Bessel-corrected, requires n ≥ 2)  |
| **std_dev**          | Square root of variance                                                |
| **ci_95_lower**      | Lower bound of 95% CI for mean duration                               |
| **ci_95_upper**      | Upper bound of 95% CI for mean duration                               |

95% CI formula (t-distribution with n − 1 degrees of freedom):

```
CI = mean ± t(0.975, n-1) × (std_dev / sqrt(n))
```

For precise t-critical values use `scipy.stats.t.ppf(0.975, df=n-1)` if Python is available.
Fallback approximations: n=2 → 12.706, n=3 → 4.303, n=4 → 3.182, n=5 → 2.776,
n=10 → 2.262, n=15 → 2.131, n=20 → 2.093, n=30 → 2.045, n=60 → 2.000, n=∞ → 1.960.
For unlisted values interpolate linearly between the two nearest entries.

**Edge cases for variance:**
- If n < 2 for a variant, variance and CI cannot be computed — show `N/A` in those columns and
  exclude that variant from the Welch t-test comparison.

### Guardrail Metric Evaluation

For each experiment that declares `guardrail_metrics:`, evaluate each threshold against the
current data and record a pass/fail status:

| Guardrail        | How to evaluate                                                   |
|------------------|-------------------------------------------------------------------|
| `>=0.95`         | Compute the metric value for this run window; check if ≥ 0.95   |
| `==0`            | Check if the metric equals exactly 0 for all variants            |
| `<=X`            | Check if the metric does not exceed X                             |

For `success_rate` guardrails: use the computed `success_rate` per variant.
For `empty_output_rate` and other binary metrics: infer from run conclusions where applicable.

If **any guardrail fails for any variant**, mark the experiment as `GUARDRAIL_FAILED` and use
`ABANDON` as the recommendation regardless of the primary metric significance.

## Step 4 — Detect Statistical Significance (p < 0.05)

Compare each variant against the first (control) variant using the appropriate test:

**Success rate — two-proportion z-test:**

```
p1 = successes_ctrl / n_ctrl
p2 = successes_var  / n_var
p_pool = (successes_ctrl + successes_var) / (n_ctrl + n_var)
z = (p1 - p2) / sqrt(p_pool × (1 − p_pool) × (1/n_ctrl + 1/n_var))
```

Convert z to a two-tailed p-value using: p ≈ 2 × (1 − Φ(|z|)).
For precise p-values use `scipy.stats.norm.sf(abs(z)) * 2` if Python is available.
Fallback CDF approximations: Φ(1.282)=0.90, Φ(1.645)=0.95, Φ(1.960)=0.975,
Φ(2.326)=0.99, Φ(2.576)=0.995. Interpolate linearly for intermediate z-values.

**Duration — Welch's t-test:**

```
t  = (mean_A − mean_B) / sqrt(var_A/n_A + var_B/n_B)
df = (var_A/n_A + var_B/n_B)^2 / ((var_A/n_A)^2/(n_A−1) + (var_B/n_B)^2/(n_B−1))
```

For precise p-values use `scipy.stats.t.sf(abs(t), df=df) * 2` if Python is available.

**Zero-variance edge case:** If all runs for a variant share the same duration (variance = 0), the
Welch t-test cannot be applied — show `N/A` for p-value and note "zero variance" in the table.

The significance threshold is **p < 0.05**.

**`min_samples` gate:** Use the `recommendation` field from `gh aw experiments analyze` to
determine readiness: when the CLI returns `EXTEND` for an experiment, always use `EXTEND` as the
recommendation (regardless of p-value) and show the per-variant progress toward `min_samples`
from `analyses[].variants[].min_samples_reached`. Only proceed with `PROMOTE` or `ABANDON` when
the CLI returns `READY_FOR_ANALYSIS`.

## Step 5 — Generate Bar Charts

For each experiment, generate two bar charts using Python (libraries and directories are already set
up by the imported `shared/trending-charts-simple.md` environment):

### Chart A — Success Rate by Variant

```python
#!/usr/bin/env python3
import json, os
import matplotlib.pyplot as plt
import matplotlib.patches as mpatches
import numpy as np

# Load per-run data written in step 2 (replace with actual data)
# variants: list of variant names
# success_rates: matching list of 0.0-1.0 success rates
# ns: matching list of sample sizes

fig, ax = plt.subplots(figsize=(10, 6), dpi=150)
colors = plt.cm.Set2(np.linspace(0, 1, len(variants)))
bars = ax.bar(variants, [r * 100 for r in success_rates], color=colors, edgecolor='white', linewidth=1.5)

# Annotate each bar with n and percentage
for bar, n, rate in zip(bars, ns, success_rates):
    ax.text(bar.get_x() + bar.get_width() / 2, bar.get_height() + 0.8,
            f'{rate*100:.1f}%\n(n={n})', ha='center', va='bottom', fontsize=11, fontweight='bold')

ax.axhline(y=success_rates[0] * 100, color='grey', linestyle='--', linewidth=1.2, label='Control baseline')
ax.set_ylim(0, 115)
ax.set_xlabel('Variant', fontsize=13)
ax.set_ylabel('Success Rate (%)', fontsize=13)
ax.set_title(f'Experiment: {experiment_name} — Success Rate by Variant', fontsize=14, fontweight='bold')
ax.legend(fontsize=11)
ax.grid(axis='y', alpha=0.4)
plt.tight_layout()
plt.savefig(f'/tmp/gh-aw/python/charts/{experiment_name}_success_rate.png',
            dpi=150, bbox_inches='tight', facecolor='white')
plt.close()
```

### Chart B — Mean Duration by Variant (with 95% CI error bars)

```python
fig, ax = plt.subplots(figsize=(10, 6), dpi=150)
# ci_lower, ci_upper: lists of CI bounds in seconds
yerr_lower = [mean - lo for mean, lo in zip(mean_durations_s, ci_lower_s)]
yerr_upper = [hi - mean for mean, hi in zip(mean_durations_s, ci_upper_s)]
colors = plt.cm.Set2(np.linspace(0, 1, len(variants)))
bars = ax.bar(variants, mean_durations_s, yerr=[yerr_lower, yerr_upper],
              color=colors, edgecolor='white', linewidth=1.5,
              capsize=8, error_kw={'linewidth': 2, 'ecolor': 'dimgray'})

for bar, mean, n in zip(bars, mean_durations_s, ns):
    ax.text(bar.get_x() + bar.get_width() / 2, bar.get_height() + max(yerr_upper) * 0.05,
            f'{mean:.0f}s\n(n={n})', ha='center', va='bottom', fontsize=11, fontweight='bold')

ax.axhline(y=mean_durations_s[0], color='grey', linestyle='--', linewidth=1.2, label='Control baseline')
ax.set_xlabel('Variant', fontsize=13)
ax.set_ylabel('Mean Duration (s)', fontsize=13)
ax.set_title(f'Experiment: {experiment_name} — Mean Duration by Variant (95% CI)', fontsize=14, fontweight='bold')
ax.legend(fontsize=11)
ax.grid(axis='y', alpha=0.4)
plt.tight_layout()
plt.savefig(f'/tmp/gh-aw/python/charts/{experiment_name}_duration.png',
            dpi=150, bbox_inches='tight', facecolor='white')
plt.close()
```

After saving each chart, upload it using the `upload_asset` safe-output tool and store the returned
asset URLs — they will be embedded in the discussion body.

## Step 6 — Render ASCII Comparison Table

For each experiment, produce an ASCII table inside a fenced code block:

```
Experiment : <experiment_name>
Workflow   : <workflow_file_name>
Hypothesis : <hypothesis text if declared, else "(not specified)">
Window     : last 30 runs  |  Analysed: <count> runs with artifacts
min_samples: <min_samples> per variant

+------------------+------+----------+----------------+--------------------+-----------+---------------+
| Variant          |  n   | Succ %   | Mean dur (s)   | 95% CI (s)         |  p-value  | min_samples   |
+------------------+------+----------+----------------+--------------------+-----------+---------------+
| <control>        |  ##  |  ##.#%   |    ###.#       | [###.# , ###.#]    |  (ref)    | ##/## (##%)   |
| <variant_B>      |  ##  |  ##.#%   |    ###.#       | [###.# , ###.#]    |  0.0XX *  | ##/## (##%)   |
+------------------+------+----------+----------------+--------------------+-----------+---------------+
Significance: * p<0.05   ** p<0.01   *** p<0.001
p-value is two-tailed, compared against the control (first) variant.

Guardrails:
  success_rate >=0.95 : PASS (control=0.97, variant_B=0.96)
  empty_output_rate ==0 : FAIL (variant_B=0.02) ← ABORT
  
For multi-variant experiments show pass/fail per variant per guardrail:
  success_rate >=0.95 : control=PASS(0.97), variant_B=FAIL(0.92), variant_C=PASS(0.96)

Recommendation: <PROMOTE | EXTEND | ABANDON>
Rationale     : <one sentence>
```

**Recommendation rules** (evaluated for the best-performing non-control variant):

| Condition                                                                              | Decision       |
|----------------------------------------------------------------------------------------|----------------|
| Any guardrail metric fails for any variant                                             | **ABANDON**    |
| p < 0.05 AND all variants have n ≥ min_samples AND variant improves success rate      | **PROMOTE**    |
| p < 0.05 AND variant degrades success rate vs. control                                | **ABANDON**    |
| p ≥ 0.05 AND any variant has n < min_samples (more data needed)                       | **EXTEND**     |
| p ≥ 0.05 AND all variants have n ≥ min_samples (no detectable effect)                 | **ABANDON**    |
| Any variant has n < 5 (insufficient data)                                              | **EXTEND** (note insufficient data) |

> **Note on statistical power:** Until all variants reach `min_samples`, tests have low power to
> detect small effects. Use **EXTEND** to gather more data before drawing conclusions.

## Step 7 — Post Discussion

Create a single GitHub Discussion containing all experiments using the `create-discussion`
safe output. The `shared/daily-audit-charts.md` import configures the discussion with
title-prefix `[experiments]`, category `audits`, and automatic cleanup of older discussions.

**Discussion title**: `[experiments] Daily Experiment Report — YYYY-MM-DD`

### Discussion body structure

```markdown
### 🧪 Daily Experiment Report — YYYY-MM-DD

[1–2 sentence executive summary: N experiments analysed across M workflows,
 K reached significance (p < 0.05), list recommendations at a glance.]

---

#### `<experiment_name>` · `<workflow_basename>`

> **Variants**: `<v1>` vs `<v2>` · **Window**: last 30 runs · **Analysed**: N runs with artifacts
> **min_samples**: <min_samples> per variant

<hypothesis if declared>

![Success Rate Chart](<ASSET_URL_success_rate>)

![Duration Chart](<ASSET_URL_duration>)

<ASCII comparison table from Step 6 inside a ``` code block>

**Recommendation: PROMOTE / EXTEND / ABANDON** — <one sentence rationale>

---

[Repeat the section above for each experiment]

### 📊 Summary

| Experiment | Workflow | Control | Best variant | p-value | Guardrails | Recommendation |
|-----------|---------|---------|-------------|---------|-----------|----------------|
| ... | ... | ... | ... | ... | PASS/FAIL | ... |

> Analysis window: last 30 runs per workflow · Significance threshold: p < 0.05 (two-tailed)
> Run: [${{ github.run_id }}](https://github.com/${{ github.repository }}/actions/runs/${{ github.run_id }})
```

If no workflows declare `experiments:`, create the discussion with a brief notice:

```markdown
### 🧪 No Active Experiments — YYYY-MM-DD

No workflows in `${{ github.repository }}` currently declare an `experiments:` section.

Run the `ab-testing-advisor` workflow to generate experiment campaign ideas.
```

After the discussion is created, also write a one-line summary to `$GITHUB_STEP_SUMMARY`:

```
Daily experiment report: N experiments analysed, M reached significance (p < 0.05). Discussion: <url>
```

## Step 8 — Notify Tracking Issues

For each experiment that has a `issue:` field set, post a comment to that tracking issue when any
of the following conditions are met **for the first time today**:

**Condition A — All variants reached `min_samples`:**
Post a comment:
```
🧪 **Experiment `<name>` is ready for analysis!**

All variants have reached the minimum sample size of `<min_samples>` runs:
<variant>: <n>/<min_samples>
...

View the latest statistics in the [Daily Experiment Report](<discussion_url>).
```

**Condition B — Experiment reached statistical significance (p < 0.05) with all guardrails
passing:**
Post a comment:
```
📊 **Experiment `<name>` has reached statistical significance (p = <p_value>)**

Recommendation: **<PROMOTE | EXTEND | ABANDON>**
<one sentence rationale>

View the full report: [Daily Experiment Report](<discussion_url>)
```

**Condition C — A guardrail metric failed:**
Post a comment:
```
⚠️ **Guardrail violation in experiment `<name>`**

The following guardrail metric failed:
- `<metric_name>` expected `<threshold>`, got `<actual_value>` for variant `<variant>`

Recommendation: **ABANDON** — investigate immediately.
```

Use the `add-comment` safe-output tool to post comments. Skip experiments with no
`issue:` field. Do not post duplicate comments if the same condition was already reported in a
previous run today.

## Step 9 — Update Experiment Lifecycle Labels

For each experiment with a tracking `issue:` field, apply the following GitHub labels on the
tracking issue when the corresponding condition is met. Create the label first if it does not
already exist (use a neutral gray color). Labels are **additive only** — once applied they are
not removed automatically; the person concluding the experiment can remove them manually.

| Label                           | Apply when                                                                   |
|--------------------------------|------------------------------------------------------------------------------|
| `experiment:active`            | `start_date <= today <= end_date` (or no dates declared)                    |
| `experiment:ready-for-analysis`| All variants have `n >= min_samples`                                         |
| `experiment:concluded`         | Recommendation is PROMOTE or ABANDON after reaching statistical significance |

Use the `add-labels` safe-output tool to apply labels to the tracking issue.
If a label does not exist in the repository, create it with `create_label` GitHub MCP tool
before applying it, using a neutral gray color (e.g. `#808080`) and a short description.

