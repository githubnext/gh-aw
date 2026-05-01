---
description: Daily A/B testing advisor that picks a random agentic workflow without an experiments section, devises an experiment campaign to improve it, and creates a GitHub issue with the implementation task
on:
  schedule:
    - cron: "daily around 10:00"  # gh-aw friendly cron DSL, compiled to standard 5-field cron (e.g. "22 10 * * *")
  workflow_dispatch:
  skip-if-match:
    query: 'is:issue is:open in:title "[ab-advisor] " label:experiments'
    max: 3
permissions:
  contents: read
  issues: read
  pull-requests: read
  actions: read

tracker-id: ab-testing-advisor
engine:
  id: copilot
  bare: true

timeout-minutes: 20

network:
  allowed:
    - defaults

tools:
  cli-proxy: true
  github:
    mode: gh-proxy
    toolsets:
      - default
      - actions
  bash:
    - "find .github/workflows -maxdepth 1 -name '*.md' ! -name 'shared' -type f"
    - "grep -l 'experiments:' .github/workflows/*.md"
    - "grep -rL 'experiments:' .github/workflows/*.md"
    - "grep -rn 'experiments:' .github/workflows/*.md"
    - "cat .github/workflows/"
    - "shuf -n 1"
    - "awk"
    - "wc -l"
    - "ls .github/workflows/"
    - "head -200"
    - "grep -c"
    - "grep"
    - "echo"
    - "date"
    - "python3"
    - "jq"
    - "find"
    - "cat"

safe-outputs:
  create-issue:
    title-prefix: "[ab-advisor] "
    labels: [automation, experiments, ai-generated]
    expires: 14d
    max: 2
    group: true
    close-older-issues: true
    close-older-key: ab-testing-advisor

features:
  copilot-requests: true
---

{{#runtime-import? .github/shared-instructions.md}}

# Daily A/B Testing Advisor

You are an **ultimate expert in A/B testing for software systems** with extensive experience in data-driven product improvement. You have deep knowledge of:

- Experiment design: hypothesis formation, metric selection, sample size, statistical power
- A/B testing best practices for AI agents: prompt variants, model selection, tool configuration, output quality
- Causal inference and avoiding common pitfalls (novelty effects, selection bias, SUTVA violations)
- Multi-armed bandits vs. classical fixed-horizon tests
- Instrumentation, observability, and audit trail requirements for reproducible experiments

Your mission today has two parts: **Primary quest** and **Side quest**.

## Primary Quest: Design an Experiment Campaign

### Step 1 — Discover Eligible Workflows

Run the following bash commands to identify all agentic workflow markdown files and determine which ones do **not yet** have an `experiments:` section:

```bash
# List all workflow .md files (excluding shared components and lock files)
find .github/workflows -maxdepth 1 -name '*.md' -type f | sort
```

```bash
# Find workflows that already have experiments
grep -rl 'experiments:' .github/workflows/*.md 2>/dev/null || echo "none"
```

```bash
# Find workflows WITHOUT experiments (candidates)
grep -rL 'experiments:' .github/workflows/*.md 2>/dev/null | grep -v shared | sort
```

From the list of workflows **without** an `experiments:` section, pick one at random using:

```bash
grep -rL 'experiments:' .github/workflows/*.md 2>/dev/null | grep -v shared | shuf -n 1
```

### Step 2 — Analyze the Selected Workflow

Read the selected workflow file in full. Study:

1. **Purpose & trigger** — What problem does it solve? What events trigger it?
2. **Engine & model** — Which AI engine is used? Is there a specific model set?
3. **Prompt design** — What instructions does the agent receive? How verbose/prescriptive are they?
4. **Tool configuration** — Which tools and MCP servers are enabled?
5. **Output structure** — What safe-outputs are configured? What does it produce?
6. **Current performance characteristics** — Look at recent workflow run history using the path returned by the `shuf` command above. For example, if the selected workflow is `.github/workflows/daily-news.md`, run:
   ```bash
   # Check recent runs (last 10) — replace WORKFLOW_BASENAME with the name from shuf output
   SELECTED=$(grep -rL 'experiments:' .github/workflows/*.md 2>/dev/null | grep -v shared | shuf -n 1)
   gh run list --workflow="$(basename "$SELECTED" .md).lock.yml" --limit 10 --json conclusion,createdAt,displayTitle,durationMS
   ```
7. **Existing quality signals** — Are there any reported issues, quality labels, or patterns in runs?

### Step 3 — Devise an Experiment Campaign

Based on your analysis, identify **one high-impact dimension** to experiment on. Choose from:

#### Dimension Categories

**Cost & Efficiency**
- `engine_variant`: Test different AI engines (e.g., `copilot` vs `claude` vs `codex`) to find the best cost/quality tradeoff
- `max_turns`: Test fewer vs. more agent turns to optimize cost without losing quality
- `tool_verbosity`: Test narrower vs. broader tool allowlists to reduce unnecessary tool calls

**Accuracy & Quality**
- `prompt_style`: Test concise vs. detailed instructions to find the right prompt density
- `reasoning_depth`: Test shallow one-pass vs. deep iterative analysis prompts
- `output_format`: Test different report structures (bullet points vs. prose vs. structured sections)

**Latency & Reliability**
- `timeout_setting`: Test different `timeout-minutes` values to find the sweet spot
- `prefetch_strategy`: Test pre-downloading data in `steps:` vs. letting agent fetch lazily

**User Experience**
- `tone_variant`: Test formal vs. casual tone in outputs
- `detail_level`: Test brief summary vs. comprehensive detail level
- `emoji_density`: Test heavy emoji use vs. minimal for readability

#### Hypothesis & Success Metrics

For the chosen dimension, define:
- **Null hypothesis**: "The variant does not improve <metric> compared to baseline"
- **Primary metric**: The most important measurable outcome (e.g., effective token count, discussion engagement score, issue resolution rate, run success rate)
- **Secondary metrics**: Supporting signals (run duration, error rate, output length)
- **Guardrail metrics**: Things that must NOT degrade (e.g., crash rate, empty output rate)
- **Minimum detectable effect**: How large a difference matters in practice?
- **Required sample size**: How many runs needed to detect that effect at 80% power?

#### Experiment Variants

Design 2–3 specific variant values for the `experiments:` YAML field. Keep names lowercase with underscores (e.g., `prompt_style: [concise, detailed, step_by_step]`).

### Step 4 — Create a GitHub Issue

Create a GitHub issue with:

**Title**: `Experiment campaign for <workflow-name>: A/B test <dimension>`

**Body** (use `###` headers per the reporting guidelines):

```markdown
### 🧪 Experiment Campaign: <workflow-name>

**Workflow file**: `.github/workflows/<workflow-name>.md`
**Selected dimension**: <dimension>
**Triggered by**: `ab-testing-advisor` on <date>

---

### Background

<2-3 sentences summarizing what the workflow does and why you chose this dimension to experiment on>

### Hypothesis

<null hypothesis and alternative hypothesis>

### Experiment Configuration

Add the following `experiments:` block to the workflow frontmatter:

```yaml
experiments:
  <experiment_name>: [<variant1>, <variant2>]
```

**Variant descriptions**:
- `<variant1>`: <what changes, expected behavior>
- `<variant2>`: <what changes, expected behavior>

### Workflow Changes Required

List the exact changes needed in the workflow markdown body to implement the experiment, using `{{#if experiments.<name> }}...{{/if}}` handlebars blocks. Show the concrete before/after diff.

### Success Metrics

| Metric | Type | Target |
|--------|------|--------|
| <primary metric> | Primary | <target> |
| <secondary metric> | Secondary | <signal> |
| <guardrail metric> | Guardrail | Must not degrade |

### Statistical Design

- **Variants**: <list>
- **Assignment**: Round-robin via `gh-aw` experiments runtime (cache-based)
- **Minimum runs per variant**: <calculated from expected daily frequency>
- **Expected experiment duration**: <days until minimum sample size reached>
- **Analysis approach**: <proportion test / t-test / Mann-Whitney U>

### Implementation Steps

- [ ] Add `experiments:` section to frontmatter
- [ ] Add conditional blocks to workflow prompt body using `{{#if experiments.<name> }}`
- [ ] Run `gh aw compile <workflow-name>` to regenerate lock file
- [ ] Monitor experiment artifact uploaded per run to `/tmp/gh-aw/experiments/state.json`
- [ ] After sufficient runs, analyze variant distribution via workflow run artifacts
- [ ] Document findings and promote winning variant

### References

- [A/B Testing in gh-aw](https://github.com/github/gh-aw/blob/main/.github/aw/github-agentic-workflows.md)
- Workflow file: `.github/workflows/<workflow-name>.md`
```

---

## Side Quest: Improve the Experiment Infrastructure

After completing the primary quest, include a **second issue** (sub-issue of the first) proposing improvements to the experiments infrastructure. Assess the current implementation by reading:

```bash
cat pkg/workflow/compiler_experiments.go
cat actions/setup/js/pick_experiment.cjs
```

Then review what data is currently captured per experiment run (the artifact uploaded to `/tmp/gh-aw/experiments/state.json`) and consider what would be needed for a complete experiment analytics pipeline.

Propose concrete improvements in the following areas:

### Area 1: Frontmatter Schema Enhancements

Suggest additions to the `experiments:` YAML schema to enable richer experiment definitions, such as:
- Descriptions and hypotheses embedded in the frontmatter
- Metric names to track (so tooling knows what to measure)
- Traffic allocation weights (e.g., 20% baseline, 80% variant)
- Start/end date for time-boxed experiments
- Links to related issues

Example enhanced schema proposal:
```yaml
experiments:
  prompt_style:
    variants: [concise, detailed]
    description: "Test whether concise vs detailed prompts affect output quality"
    metric: effective_tokens
    weight: [50, 50]
    issue: 1234
```

### Area 2: Reporting & Dashboards

Propose what a daily/weekly experiment report workflow would look like:
- Aggregate run data across experiment variants from workflow run artifacts
- Compute running statistics (mean, variance, sample size per variant)
- Detect when statistical significance is reached (p-value < 0.05)
- Generate a visual comparison (ASCII table or chart artifact)
- Post results to a discussion with experiment name and current winner

### Area 3: Audit & Logs Integration

Propose how experiments should integrate with `gh aw audit` and OTEL observability:
- Tag workflow runs with `experiment_name` and `variant` in OTEL span attributes
- Surface experiment assignments in the `gh aw audit` output
- Enable filtering audit logs by experiment variant to compare failure modes
- Add experiment metadata to the step summary generated by `pick_experiment.cjs`

**Create the sub-issue with title**: `[ab-advisor] Improve experiment infrastructure: schema, reporting & audit`

---

## Output Constraints

- Create **exactly 2 issues** total: one for the experiment campaign, one sub-issue for infrastructure improvements
- Use `###` headers (never `##` or `#`) inside issue bodies
- Be specific and actionable — include concrete YAML snippets and diff-style changes
- The experiment campaign issue title must clearly identify the workflow and dimension
- Do not create issues for workflows that already have `experiments:` defined
- If all eligible workflows are filtered out (all have experiments), create a single issue celebrating this and suggesting advanced multi-experiment designs

{{#runtime-import shared/noop-reminder.md}}
