---
description: Daily report on all active A/B experiments across agentic workflows — which experiments are running, variant distribution, recent run history, and progress toward conclusions
on:
  schedule:
    - cron: "daily around 9:00"
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
  actions: read
  discussions: read
tracker-id: daily-experiment-report
engine: copilot
strict: true
timeout-minutes: 30

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
    - "find .github/workflows -maxdepth 1 -name '*.md' -type f"
    - "grep -rl 'experiments:' .github/workflows/*.md"
    - "grep -rn 'experiments:' .github/workflows/*.md"
    - "cat .github/workflows/"
    - "grep -A"
    - "grep -B"
    - "grep"
    - "awk"
    - "sed"
    - "head"
    - "cat"
    - "echo"
    - "date"
    - "jq"
    - "python3"
    - "find"
    - "wc"
    - "sort"
    - "uniq"

safe-outputs:
  mentions: false
  allowed-github-references: []
  max-bot-mentions: 1
  create-discussion:
    expires: 2d
    category: "audits"
    title-prefix: "[experiments] "
    max: 1
    close-older-discussions: true

imports:
  - shared/reporting.md

features:
  copilot-requests: true
---

{{#runtime-import? .github/shared-instructions.md}}

# Daily Experiment Report

You are an **experiment analyst** generating a daily status report on all active A/B experiments running across agentic workflows in ${{ github.repository }}.

## Mission

Discover all workflows with `experiments:` sections, collect their recent run history, compute variant distribution, and generate a discussion that helps the team understand where experiments stand today.

## Phase 1: Discover Active Experiments

### Step 1.1 — Find Workflows with Experiments

```bash
# Find all workflow .md files with an experiments: section
grep -rl 'experiments:' .github/workflows/*.md 2>/dev/null | grep -v '/shared/' | sort
```

If no workflows have experiments, skip to Phase 4 and post a brief "No Active Experiments" notice.

### Step 1.2 — Extract Experiment Definitions

For each workflow file found above, read the `experiments:` block from its frontmatter:

```bash
# Show experiments block for each workflow (replace WORKFLOW with each file path)
grep -A 20 '^experiments:' WORKFLOW | head -30
```

For each workflow, record:
- **Workflow file**: path and basename (e.g., `smoke-copilot`)
- **Experiment names**: each key under `experiments:`
- **Variants**: the list of variant strings for each experiment (e.g., `[yes, no]`)

### Step 1.3 — Read Workflow Descriptions

For each workflow with experiments, also capture its `description:` field so the report has context:

```bash
grep -m 1 '^description:' .github/workflows/WORKFLOW.md
```

## Phase 2: Collect Run History

For each workflow that has experiments, fetch its recent run history using the GitHub API. Check the last **20 runs** for each workflow.

Use the `gh` CLI via `cli-proxy`. Replace `WORKFLOW_BASENAME` with the bare workflow name without extension (e.g., for `.github/workflows/smoke-copilot.md`, use `smoke-copilot`):

```bash
gh run list --workflow="WORKFLOW_BASENAME.lock.yml" --limit 20 --json databaseId,conclusion,createdAt,displayTitle,durationMS,status
```

For each run, note:
- Run ID, conclusion (success/failure/cancelled), timestamp, duration

## Phase 3: Analyze Variant Distribution

For each experiment, try to determine variant assignments from recent successful run step summaries. The `pick_experiment.cjs` script appends a Markdown step summary table during each run showing which variant was assigned and the cumulative counts per variant.

Step summaries are stored in GitHub Actions and are not directly accessible via `gh run view --log` (which returns raw job logs only). Use the GitHub API to retrieve step summary data for a specific run's jobs:

```bash
# Get job IDs for a run (replace RUN_ID with a successful run ID)
gh api repos/${{ github.repository }}/actions/runs/RUN_ID/jobs --jq '.jobs[] | {id, name}'
```

If step summaries are not accessible via the available tools, estimate variant distribution based on successful run count divided by the number of variants (the round-robin algorithm guarantees near-equal distribution). Clearly note in the report when figures are estimates vs. confirmed.

**Important**: Only count **successful** runs when estimating variant distribution — cancelled or failed runs may not have reached the experiment assignment step.

## Phase 4: Generate Discussion Report

Create a GitHub Discussion with a comprehensive experiment status report.

**Title format**: `[experiments] Active Experiment Status — YYYY-MM-DD`

### Report Body Structure

Use this exact structure:

```markdown
### 🧪 Experiment Status Overview

[2–3 sentence summary: How many experiments are active, across how many workflows, and whether there are any noteworthy patterns or experiments close to completion.]

### Active Experiments

[One subsection per workflow that has experiments. Use #### for each workflow name.]

#### `<workflow-basename>`

> <workflow description in one line>

| Experiment | Variants | Runs (last 20) | Successful | Est. per Variant |
|-----------|----------|----------------|------------|-----------------|
| `<name>` | `<v1>`, `<v2>` | N | N | ~N each |

**Status**: [Brief assessment — e.g., "Early stage (N runs), collecting data" / "Approaching significance (~N more runs needed)" / "Sufficient data for analysis"]

<details>
<summary>Recent Run History</summary>

| Run | Date | Conclusion | Duration |
|-----|------|------------|----------|
| [§RUN_ID](https://github.com/${{ github.repository }}/actions/runs/RUN_ID) | DATE | ✅/❌/⚠️ | Xs |

</details>

---

[Repeat for each workflow with experiments]

### 📊 Summary Table

| Workflow | Experiment | Variants | Total Runs | Recommendation |
|---------|-----------|---------|-----------|----------------|
| ... | ... | ... | ... | ... |

### 🔮 Recommendations

[Actionable list of 2–5 recommendations. Examples:
- Which experiments have enough data to analyze and promote a winner
- Which experiments are inactive and could be removed
- Which workflows lack experiments and would benefit from one (refer to ab-testing-advisor)]
```

### Formatting Rules

- Use `###` for top-level sections, `####` for per-workflow sections
- Wrap run tables in `<details>` blocks to keep the report scannable
- Emoji are encouraged for scannability (🧪 ✅ ❌ ⚠️ 📊 🔮)
- Link every run ID as `[§ID](url)` — do NOT use plain `#123` references (they trigger backlinks)
- If a workflow had no successful runs in the last 20, note it as **inactive**
- If there are zero experiments across the entire repo, post a short notice celebrating the clean slate and recommending `gh aw run ab-testing-advisor` to generate experiment ideas

## Guidelines

- Be concise but specific — include real numbers from the data
- Do not mention `@usernames` in the report body (they trigger notifications)
- Do not reference issue/PR numbers with `#` (they create backlinks)
- Format all dates as `YYYY-MM-DD`
- This report is informational — never recommend changes that require human approval before taking effect

{{#runtime-import shared/noop-reminder.md}}
