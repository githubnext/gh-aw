---
description: Daily statistical report that aggregates experiment-state artifacts across recent runs, computes per-variant statistics (mean, variance, 95% CI, success rate), detects significance via Welch t-test or two-proportion z-test (p < 0.05), renders an ASCII comparison table per experiment, and posts it to each experiment's tracking issue or as a workflow step summary with a promote/extend/abandon recommendation
name: daily-experiment-report
on:
  schedule:
    - cron: "0 8 * * *"   # 08:00 UTC daily
  workflow_dispatch:
permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read

engine: copilot
tools:
  github:
    toolsets: [default, actions]

safe-outputs:
  add-comment:
    max: 20
    hide-older-comments: true
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

Use the GitHub MCP tools to list all workflows in `${{ github.repository }}`. For each workflow,
read its frontmatter and identify those that declare an `experiments:` section. Collect:

- Workflow file name (e.g. `daily-report.lock.yml`)
- Each experiment name (e.g. `prompt_style`)
- Variants (e.g. `[concise, detailed]`)
- Tracking issue number, if an `issue:` field is set for that experiment

If no workflows declare `experiments:`, append the following to `$GITHUB_STEP_SUMMARY` and exit:

```
No active experiments found in ${{ github.repository }} — nothing to report.
```

## Step 2 — Collect Run Data

For each workflow that has experiments, list the **last 30 completed runs** (any final state:
`success`, `failure`, `cancelled`, or `skipped`) using the GitHub MCP tools. For each run, record:

- `run_id`
- `conclusion` (`success`, `failure`, `cancelled`, …)
- `created_at` and `updated_at`
- `run_duration_ms` (derived from `created_at` and `updated_at`)

Then download the `experiment` artifact (`state.json`) from every run that has one. The state file
has the cumulative counts for every variant up to and including that run:

```json
{
  "counts": {
    "<experiment_name>": {
      "<variant>": <cumulative_count>
    }
  }
}
```

By comparing the cumulative counts between consecutive runs (oldest → newest), infer which variant
was assigned to each run: the variant whose count increased by exactly 1 from the previous snapshot
is the variant used on that run.

**Edge cases for variant inference:**
- **Missing artifact**: If a run has no experiment artifact, skip it and treat the count sequence as
  having a gap — do not attempt to infer assignment from the next available snapshot.
- **Zero increases**: If no variant count changed between two consecutive snapshots (e.g., cancelled
  run before the experiment step), record the variant as `unknown` and exclude that run from
  statistical calculations.
- **Multiple increases**: If more than one variant count increased (e.g., two runs completed between
  downloaded snapshots), record both runs as `ambiguous` and exclude them from calculations.
  Note the number of ambiguous runs in the report.

Build a per-run record for every run that had an experiment artifact:

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

For each experiment and each variant, compute the following statistics over all collected runs:

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

## Step 5 — Render ASCII Comparison Table

For each experiment, produce an ASCII table inside a fenced code block:

```
Experiment : <experiment_name>
Workflow   : <workflow_file_name>
Window     : last 30 runs  |  Analysed: <count> runs with artifacts

+------------------+------+----------+----------------+--------------------+-----------+
| Variant          |  n   | Succ %   | Mean dur (s)   | 95% CI (s)         |  p-value  |
+------------------+------+----------+----------------+--------------------+-----------+
| <control>        |  ##  |  ##.#%   |    ###.#       | [###.# , ###.#]    |  (ref)    |
| <variant_B>      |  ##  |  ##.#%   |    ###.#       | [###.# , ###.#]    |  0.0XX *  |
+------------------+------+----------+----------------+--------------------+-----------+
Significance: * p<0.05   ** p<0.01   *** p<0.001
p-value is two-tailed, compared against the control (first) variant.

Recommendation: <PROMOTE | EXTEND | ABANDON>
Rationale     : <one sentence>
```

**Recommendation rules** (evaluated for the best-performing non-control variant):

| Condition                                                      | Decision       |
|----------------------------------------------------------------|----------------|
| p < 0.05 AND variant improves success rate vs. control        | **PROMOTE**    |
| p ≥ 0.05 AND n < 20 per variant (more data needed)            | **EXTEND**     |
| p ≥ 0.05 AND n ≥ 20 per variant (no detectable effect)        | **ABANDON**    |
| p < 0.05 AND variant degrades success rate vs. control        | **ABANDON**    |
| Any variant has n < 5 (insufficient data)                     | **EXTEND** (note insufficient data) |

> **Note on statistical power:** With n < 20 per variant, tests have low power to detect small
> effects. Even a non-significant result at this sample size does not rule out a meaningful
> difference — use the **EXTEND** recommendation to gather more data before drawing conclusions.

## Step 6 — Post Results

For each experiment:

1. **If `issue:` is set in the experiment's frontmatter**, post the ASCII table as a new comment on
   that issue using the `add-comment` safe output. Begin the comment with:
   `### 🧪 Experiment Report — YYYY-MM-DD`

2. **If no tracking issue is configured**, append the ASCII table to `$GITHUB_STEP_SUMMARY`.

After processing all experiments, print a one-line summary to `$GITHUB_STEP_SUMMARY`:

```
Daily experiment report: N experiments analysed, M reached significance (p < 0.05).
```
