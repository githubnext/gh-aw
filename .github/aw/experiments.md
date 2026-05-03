---
description: Guide for setting up A/B testing experiments in agentic workflows — syntax, design principles, dimensions to test, how to measure results, and anti-patterns.
---

# A/B Testing Experiments in Agentic Workflows

Consult this file when you want to measure the impact of a prompt change, a new skill, a model switch, or any other workflow variation. The `experiments:` frontmatter field wires the entire test-and-measure loop into gh-aw with no external tooling required.

---

## How Experiments Work

Each workflow run goes through this lifecycle:

1. **Restore** — the activation job loads the experiment state JSON from the configured storage (git branch by default, or GitHub Actions cache when `storage: cache`).
2. **Pick** — `pick_experiment.cjs` selects a variant for each declared experiment using a balanced round-robin counter. The variant with the lowest invocation count so far is chosen; ties are broken by variant array order, producing deterministic balanced assignment across runs.
3. **Save** — the updated counter state is written back to the configured storage.
4. **Upload** — the state file is uploaded as a workflow artifact named `experiment` (retained 30 days) so you can audit per-run assignments.
5. **Inject** — the selected variant is available in the workflow prompt as `${{ experiments.<name> }}` and in `{{#if experiments.<name> }}` handlebars blocks.

**Key properties**:
- Every run receives exactly one variant assignment per declared experiment.
- Assignment persists across runs automatically; no setup is required beyond the `experiments:` field.
- Multiple experiments can run simultaneously — each is independently balanced.
- No sampling or percentage-based routing: every run participates.

---

## Basic Syntax

```yaml
---
on:
  schedule: daily on weekdays
engine: copilot
experiments:
  prompt_style: [concise, detailed]
---

{{#if experiments.prompt_style == "concise" }}
Summarise the findings in ≤ 5 bullets.
{{#else}}
Provide a detailed analysis with reasoning for each finding.
{{#endif}}
```

### Naming Rules

- Experiment names must match `[a-zA-Z_][a-zA-Z0-9_]*` (identifier style).
- Use **lowercase with underscores**: `prompt_style`, not `PromptStyle` or `prompt-style`.
- Names that do not match the pattern are silently skipped at compile time.

### Variant Rules

- Each experiment must declare **at least 2 variants**.
- Variant values are plain strings — keep them lowercase and descriptive (`concise`, `detailed`, `yes`, `no`, `step_by_step`).
- Up to ~10 variants are practical; beyond that, the required sample size per variant grows quickly.

---

## Object Form (Weighted Variants and Date Gating)

The `experiments:` field also accepts an object form when you need non-uniform split probabilities, automatic deactivation after a date, or machine-readable governance metadata:

```yaml
experiments:
  prompt_style:
    variants: [concise, detailed, step_by_step]
    weight: [2, 1, 1]           # 50% concise, 25% detailed, 25% step_by_step
    description: "Verbosity A/B test"
    metric: "effective_tokens"
    issue: "42"
    start_date: "2026-05-01"
    end_date: "2026-06-01"
```

**Fields:**

- `variants:` - Array of variant strings (required, ≥ 2 entries). Same constraints as bare-array form.
- `weight:` - Array of non-negative integers, same length as `variants`. When set, weighted-random selection replaces round-robin. Weights of `[2, 1, 1]` mean 50/25/25 split. When all weights are zero, the first (control) variant is always returned. Omit to keep the default round-robin behavior.
- `start_date:` - ISO-8601 date (`YYYY-MM-DD`). Before this date the control variant is returned and counters are not incremented. Useful for pre-scheduling an experiment.
- `end_date:` - ISO-8601 date (`YYYY-MM-DD`). After this date the control variant is returned automatically. No manual intervention needed to wind down an experiment.
- `description:` - Human-readable experiment description for governance tooling (no runtime effect).
- `metric:` - Primary metric name for governance tooling (no runtime effect).
- `issue:` - Linked tracking issue number for governance tooling (no runtime effect).

**Bare array and object forms can be mixed** in the same `experiments:` map — each experiment is independent.

---

## Storage Configuration

The `storage` key inside the `experiments:` map controls how experiment state is persisted across runs:

```yaml
experiments:
  storage: repo   # or: cache
  prompt_style: [concise, detailed]
```

| Value | Behaviour | When to use |
|---|---|---|
| `repo` (**default**) | Commits `state.json` to a git branch named `experiments/{sanitizedWorkflowID}` (workflow ID lowercased with hyphens removed, e.g. `my-workflow` → `experiments/myworkflow`) after each run. State survives cache evictions. | Recommended for all experiments — experiment data is valuable. |
| `cache` | Uses GitHub Actions cache (legacy behaviour). State may be evicted after 7 days of inactivity. | Only when `contents: write` cannot be granted to the workflow. |

**Key differences:**

- **`repo` storage** adds a `push_experiments_state` job that runs after the activation job. This job commits the updated state to a branch like `experiments/myworkflow` using `contents: write` permission. The state is durable and survives long periods without workflow runs.
- **`cache` storage** is the original behaviour. No extra job or permission is required, but state can be evicted after 7 days of GitHub Actions cache inactivity.

> The branch is created automatically on first run (as an orphan branch containing only `state.json` and `assignments.json`).

---

## Referencing the Active Variant

The selected variant is injected into the prompt in two ways:

### 1 — Conditional blocks (most common)

```markdown
{{#if experiments.tone == "formal" }}
Use formal, professional language throughout the report.
{{#else}}
Use a friendly, conversational tone.
{{#endif}}
```

### 2 — Direct interpolation

```markdown
Use `${{ experiments.tone }}` tone when writing the issue body.
```

Both forms are resolved before the agent receives the prompt. The agent always sees the resolved text, never the raw expression.

---

## Designing a Good Experiment

A well-designed experiment has:

1. **One dimension** changed at a time — isolate the variable to attribute differences to the right cause.
2. **A falsifiable hypothesis** — state what you expect and what would disprove it.
3. **A primary metric** that is measurable from workflow run data (artifacts, outputs, duration, token counts).
4. **Guardrail metrics** — things that must not degrade (e.g., crash rate, empty-output rate, run success rate).
5. **A sample size estimate** — calculate how many runs per variant are needed before drawing conclusions.

### Hypothesis template

```
Changing <dimension> from <baseline> to <variant> will improve <primary metric>
by at least <minimum detectable effect> without degrading <guardrail metric>.
```

### Sample size rule of thumb

For a two-proportion test with 80% power and α = 0.05:
- Detecting a **5 pp** effect needs ~600 runs per variant.
- Detecting a **10 pp** effect needs ~160 runs per variant.
- Detecting a **20 pp** effect needs ~40 runs per variant.

For daily workflows (~1 run/day per variant), a 10 pp effect takes roughly 160 days per variant. Prefer experiments on **high-frequency workflows** (hourly, multiple times per day) so you reach statistical significance faster.

---

## Dimensions Worth Experimenting On

### Prompt Design

```yaml
experiments:
  prompt_style: [concise, detailed]
  reasoning_depth: [shallow, deep]
  output_format: [bullets, prose, table]
  tone: [formal, casual]
```

Use `{{#if experiments.prompt_style }}` to swap the corresponding instructions in the prompt body.

**Typical metrics**: output quality score (human-rated), effective token count, action success rate, output length.

### Engine & Model

```yaml
experiments:
  engine_variant: [copilot, claude]
```

Then use a `{{#if experiments.engine_variant == "claude" }}` block *or* simply point to different engine configurations in separate compiled workflows.

> ⚠️ **Engine experiments require separate compiled files** if the engine changes the `engine:` frontmatter key. You cannot switch the engine mid-run from a single workflow file. Instead, create two workflow files (baseline + variant), run them in parallel, and compare their run metrics.

**Typical metrics**: run cost (token usage), run duration, task completion rate, error rate.

### Tool Configuration

```yaml
experiments:
  tool_scope: [narrow, broad]
```

```markdown
{{#if experiments.tool_scope == "narrow" }}
Only use the `issues` and `pull_requests` toolsets.
{{#else}}
Use any available GitHub MCP tools.
{{#endif}}
```

**Typical metrics**: number of tool calls, run duration, output accuracy.

### Skill Usage

```yaml
experiments:
  skill_hint: [enabled, disabled]
```

```markdown
{{#if experiments.skill_hint == "enabled" }}
Check `skills/` for SKILL.md files relevant to this task and apply their guidance.
{{#endif}}
```

**Typical metrics**: output quality, context token consumption, run duration.

### Timeout & Pacing

```yaml
experiments:
  timeout: [short, long]
```

Pair with a conditional step that sets the effective timeout, or use two compiled workflow files with different `timeout-minutes:` values.

---

## Minimal Working Example

```markdown
---
description: Daily PR summary — A/B test concise vs. detailed output
on:
  schedule: daily on weekdays
engine: copilot
permissions:
  pull-requests: read
tools:
  github:
    toolsets: [pull_requests]
safe-outputs:
  create-discussion:
    title-prefix: "[pr-summary] "
    close-older-discussions: true
timeout-minutes: 15
experiments:
  output_style: [concise, detailed]
---

Summarise the pull requests merged in ${{ github.repository }} today.

{{#if experiments.output_style == "concise" }}
Write a maximum of 5 bullet points. Each bullet is one sentence.
{{#else}}
Write a structured report with sections for: new features, bug fixes, refactors,
and documentation changes. Include a one-paragraph executive summary at the top.
{{#endif}}

Include links to each PR. Use ${{ github.server_url }}/${{ github.repository }}/pull/<number> format.
```

Compile and deploy:

```bash
gh aw compile pr-summary
```

The first run picks `concise` (lowest count = 0 for both), the second picks `detailed`, and so on, alternating until one variant is statistically better.

---

## Reading Experiment Results

The state file is available as a workflow artifact named `experiment` on every run. Download it to inspect cumulative counts:

```bash
# List recent runs for a workflow
gh run list --workflow="my-workflow.lock.yml" --limit 20 --json databaseId,conclusion,createdAt

# Download the experiment artifact from a specific run
gh run download <run-id> --name "experiment" --dir /tmp/exp-results

cat /tmp/exp-results/state.json
```

The `state.json` format:

```json
{
  "counts": {
    "output_style": {
      "concise": 12,
      "detailed": 12
    }
  }
}
```

To compare variant outcomes, collect the artifact from every run and join the variant assignment to whatever metric you are tracking (e.g., output length, discussion reaction count, manual quality rating).

The `ab-testing-advisor` workflow (`.github/workflows/ab-testing-advisor.md`) automates the experiment design step — it picks a random workflow without experiments and creates a GitHub issue with a ready-to-implement experiment campaign.

---

## Multiple Simultaneous Experiments

You can run several experiments at once. Each is assigned independently:

```yaml
experiments:
  prompt_style: [concise, detailed]
  emoji_density: [heavy, minimal]
  skill_hint: [enabled, disabled]
```

All three variants are independently balanced. The prompt receives all three active values simultaneously.

> ⚠️ **Interaction effects** — when two experiments are both active, differences in the primary metric could be caused by either variable or their interaction. Limit simultaneous experiments to 2–3 and analyse them separately unless you have enough runs to do a factorial analysis.

---

## Lifecycle of an Experiment

1. **Design** — write hypothesis, pick dimension, define primary + guardrail metrics.
2. **Instrument** — add `experiments:` to frontmatter and `{{#if experiments.<name> }}` blocks to the prompt.
3. **Compile** — `gh aw compile <workflow-name>` to regenerate the lock file.
4. **Run** — let the workflow accumulate runs. Check the step summary in each run's activation job to confirm the variant assignment.
5. **Analyse** — once the minimum sample size per variant is reached, compare metric distributions across variants.
6. **Conclude** — promote the winning variant by rewriting the baseline prompt and removing the `experiments:` field. Run `gh aw compile` to finalize.

---

## Anti-Patterns

- ❌ **Do not test multiple dimensions in a single experiment name** — if you change both the tone and the output length together, you cannot tell which change caused the improvement.
- ❌ **Do not remove the `experiments:` field before the sample size is reached** — this resets the state on the next run and invalidates accumulated counts.
- ❌ **Do not interpret early results** — with fewer than ~20 runs per variant, chance variation dominates. Wait for statistical significance before drawing conclusions.
- ❌ **Do not use experiments for feature flags** — use the `features:` frontmatter field for deterministic on/off switches that are not under statistical test.
- ❌ **Do not run engine experiments from a single workflow file** — engine switches require a different `engine:` frontmatter value, which means a separate compiled file. Use two parallel workflow files and compare their GitHub Actions run metrics instead.
- ❌ **Do not nest `{{#if experiments.<name> }}` inside `{{#runtime-import? }}` blocks** — expression evaluation order is not guaranteed across import boundaries; keep experiment conditionals in the top-level workflow body.
