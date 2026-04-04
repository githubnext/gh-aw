---
title: Cost Management
description: Understand and control the cost of running GitHub Agentic Workflows, including Actions minutes, inference billing, and strategies to reduce spend.
sidebar:
  order: 296
---

The cost of running an agentic workflow is the sum of two components: **GitHub Actions minutes** consumed by the workflow jobs, and **inference costs** charged by the AI provider for each agent run.

## Cost Components

### GitHub Actions Minutes

Every workflow job consumes Actions compute time billed at standard [GitHub Actions pricing](https://docs.github.com/en/billing/managing-billing-for-your-products/managing-billing-for-github-actions/about-billing-for-github-actions). A typical agentic workflow run includes at least two jobs:

| Job | Purpose | Typical duration |
|-----|---------|-----------------|
| Pre-activation / detection | Validates the trigger, runs membership checks, evaluates `skip-if-match` conditions | 10–30 seconds |
| Agent | Runs the AI engine and executes tools | 1–15 minutes |

Each job also incurs approximately 1.5 minutes of runner setup overhead on top of its execution time.

### Inference Costs

The agent job invokes an AI engine (Copilot, Claude, Codex, or a custom engine) to process the prompt and call tools. Inference is billed by the provider:

- **GitHub Copilot CLI** (`copilot` engine): Usage is billed as premium requests against the GitHub account that owns the [`COPILOT_GITHUB_TOKEN`](/gh-aw/reference/auth/#copilot_github_token). A typical workflow run uses 1–2 premium requests. See [GitHub Copilot billing](https://docs.github.com/en/copilot/about-github-copilot/subscription-plans-for-github-copilot).
- **Claude** (`claude` engine): Billed per token to the Anthropic account associated with [`ANTHROPIC_API_KEY`](/gh-aw/reference/auth/#anthropic_api_key).
- **Codex** (`codex` engine): Billed per token to the OpenAI account associated with [`OPENAI_API_KEY`](/gh-aw/reference/auth/#openai_api_key).

> [!NOTE]
> For Copilot, inference is charged to the individual account owning `COPILOT_GITHUB_TOKEN`, not to the repository or organization running the workflow. Use a dedicated service account and monitor its premium request usage to track spend per workflow.

## Monitoring Costs with `gh aw logs`

The `gh aw logs` command downloads workflow run data and surfaces per-run metrics including elapsed duration, token usage, and estimated inference cost. Use it to see exactly what your workflows are consuming before deciding what to optimize.

For a deep dive into a single run's token usage, tool calls, and inference spend, use `gh aw audit <run-id>`. The **Metrics** and **Performance Metrics** sections of the audit report show token counts, effective tokens, turn counts, and estimated cost in one place — useful for diagnosing why a specific run was expensive. For cost trends across multiple runs, use `gh aw logs --format markdown [workflow]` to generate a cross-run report with metrics trends and anomaly detection.

### View recent run durations

```bash
# Overview table for all agentic workflows (last 10 runs)
gh aw logs

# Narrow to a single workflow
gh aw logs issue-triage-agent

# Last 30 days for Copilot workflows
gh aw logs --engine copilot --start-date -30d
```

The overview table includes a **Duration** column showing elapsed wall-clock time per run. Because GitHub Actions bills compute time by the minute (rounded up per job), duration is the primary indicator of Actions spend.

### Export metrics as JSON

Use `--json` to get structured output suitable for scripting or trend analysis:

```bash
# Write JSON to a file for further processing
gh aw logs --start-date -1w --json > /tmp/logs.json

# List per-run duration, tokens, and cost across all workflows
gh aw logs --start-date -30d --json | \
  jq '.runs[] | {workflow: .workflow_name, duration: .duration, cost: .estimated_cost}'

# Total cost grouped by workflow over the past 30 days
gh aw logs --start-date -30d --json | \
  jq '[.runs[]] | group_by(.workflow_name) |
  map({workflow: .[0].workflow_name, runs: length, total_cost: (map(.estimated_cost) | add // 0)})'
```

The JSON output includes `duration`, `token_usage`, `estimated_cost`, `workflow_name`, and `agent` (the engine ID) for each run under `.runs[]`.

For orchestrated workflows, the same JSON also includes deterministic lineage under `.episodes[]` and `.edges[]`. The episode rollups expose aggregate fields such as `total_runs`, `total_tokens`, `total_estimated_cost`, `risky_node_count`, and `suggested_route`, which are more useful than raw per-run metrics when one logical job spans multiple workflow runs.

```bash
# List episode-level cost and risk data over the past 30 days
gh aw logs --start-date -30d --json | \
  jq '.episodes[] | {episode: .episode_id, workflow: .primary_workflow, runs: .total_runs, cost: .total_estimated_cost, risky_nodes: .risky_node_count}'
```

### Use inside a workflow agent

The `agentic-workflows` MCP tool exposes the same `logs` operation so that a workflow agent can collect cost data programmatically. Add `tools: agentic-workflows:` to any workflow that needs to read run metrics:

```aw wrap
description: Weekly Actions minutes cost report
on: weekly
permissions:
  actions: read
engine: copilot
tools:
  agentic-workflows:
```

The agent then calls the `logs` tool with `start_date: "-7d"` to retrieve duration and cost data for all recent runs, enabling automated reporting or optimization.

## Trigger Frequency and Cost Risk

The primary cost lever for most workflows is how often they run. Some events are inherently high-frequency:

| Trigger type | Risk | Notes |
|-------------|------|-------|
| `push` | High | Every commit to any matching branch fires the workflow |
| `pull_request` | Medium–High | Fires on open, sync, re-open, label, and other subtypes |
| `issues` | Medium–High | Fires on open, close, label, edit, and other subtypes |
| `check_run`, `check_suite` | High | Can fire many times per push in busy repositories |
| `issue_comment`, `pull_request_review_comment` | Medium | Scales with comment activity |
| `schedule` | Low–Predictable | Fires at a fixed cadence; easy to budget |
| `workflow_dispatch` | Low | Human-initiated; naturally rate-limited |

> [!CAUTION]
> Attaching an agentic workflow to `push`, `check_run`, or `check_suite` in an active repository can generate hundreds of runs per day. Start with `schedule` or `workflow_dispatch` while evaluating cost, then move to event-based triggers with safeguards in place.

## Reducing Cost

### Use Deterministic Checks to Skip the Agent

The most effective cost reduction is skipping the agent job entirely when it is not needed. The `skip-if-match` and `skip-if-no-match` conditions run during the low-cost pre-activation job and cancel the workflow before the agent starts:

```aw wrap
on:
  issues:
    types: [opened]
  skip-if-match: 'label:duplicate OR label:wont-fix'
```

```aw wrap
on:
  issues:
    types: [labeled]
  skip-if-no-match: 'label:needs-triage'
```

Use these to filter out noise before incurring inference costs. See [Triggers](/gh-aw/reference/triggers/) for the full syntax.

### Choose a Cheaper Model

The `engine.model` field selects the AI model. Smaller or faster models cost significantly less per token while still handling many routine tasks:

```aw wrap
engine:
  id: copilot
  model: gpt-4.1-mini
```

```aw wrap
engine:
  id: claude
  model: claude-haiku-4-5
```

Reserve frontier models (GPT-5, Claude Sonnet, etc.) for complex tasks. Use lighter models for triage, labeling, summarization, and other structured outputs.

### Limit Context Size

Inference cost scales with the size of the prompt sent to the model. Reduce context by:

- Writing focused prompts that include only necessary information.
- Avoiding whole-file reads when only a few lines are relevant.
- Capping the number of search results or list items fetched by tools.
- Using `imports` to compose a smaller subset of prompt sections at runtime.

### Rate Limiting and Concurrency

Use `rate-limit` to cap how many times a user can trigger the workflow in a given window, and rely on concurrency controls to serialize runs rather than letting them pile up:

```aw wrap
rate-limit:
  max: 3
  window: 60  # 3 runs per hour per user
```

See [Rate Limiting Controls](/gh-aw/reference/rate-limiting-controls/) and [Concurrency](/gh-aw/reference/concurrency/) for details.

### Use Schedules for Predictable Budgets

Scheduled workflows fire at a fixed cadence, making cost easy to estimate and cap:

```aw wrap
schedule: daily on weekdays
```

One scheduled run per weekday = five agent invocations per week. See [Schedule Syntax](/gh-aw/reference/schedule-syntax/) for the full fuzzy schedule syntax.

## Agentic Cost Optimization

Agentic workflows can inspect and optimize other agentic workflows automatically. A scheduled meta-agent reads aggregate run data through the `agentic-workflows` MCP tool, identifies expensive or inefficient workflows, and applies changes — closing the optimization loop without manual intervention.

### How It Works

The `agentic-workflows` tool exposes the same operations as the CLI (`logs`, `audit`, `status`) to any workflow agent. A meta-agent can:

1. Fetch aggregate cost and token data with the `logs` tool (equivalent to `gh aw logs`).
2. Deep-dive into individual runs with the `audit` tool (equivalent to `gh aw audit <run-id>`).
3. Propose or directly apply frontmatter changes (cheaper model, tighter `skip-if-match`, lower `rate-limit`) via a pull request.

### What to Optimize Automatically

| Signal | Automatic action |
|--------|-----------------|
| High token count per run | Switch to a smaller model (`gpt-4.1-mini`, `claude-haiku-4-5`) |
| Frequent runs with no safe-output produced | Add or tighten `skip-if-match` |
| Long queue times due to concurrency | Lower `rate-limit.max` or add a `concurrency` group |
| Workflow running too often | Change trigger to `schedule` or add `workflow_dispatch` |

> [!NOTE]
> The `agentic-workflows` tool requires `actions: read` permission and is configured under the `tools:` frontmatter key. See [GH-AW as an MCP Server](/gh-aw/reference/gh-aw-as-mcp-server/) for available operations.

## Common Scenario Estimates

These are rough estimates to help with budgeting. Actual costs vary by prompt size, tool usage, model, and provider pricing.

| Scenario | Frequency | Actions minutes/month | Inference/month |
|----------|-----------|----------------------|-----------------|
| Weekly digest (schedule, 1 repo) | 4×/month | ~1 min | ~4–8 premium requests (Copilot) |
| Issue triage (issues opened, 20/month) | 20×/month | ~10 min | ~20–40 premium requests |
| PR review on every push (busy repo, 100 pushes/month) | 100×/month | ~100 min | ~100–200 premium requests |
| On-demand via slash command | User-controlled | Varies | Varies |

> [!TIP]
> Use `gh aw audit <run-id>` to deep-dive into token usage and cost for a single run. Use `gh aw logs --format markdown [workflow]` to analyze cost trends across multiple runs. Create separate `COPILOT_GITHUB_TOKEN` service accounts per repository or team to attribute spend by workflow.

## Related Documentation

- [Audit Commands](/gh-aw/reference/audit/) - Single-run analysis, diff, and cross-run reporting
- [Effective Tokens Specification](/gh-aw/reference/effective-tokens-specification/) - How effective token counts are computed
- [Triggers](/gh-aw/reference/triggers/) - Configuring workflow triggers and skip conditions
- [Rate Limiting Controls](/gh-aw/reference/rate-limiting-controls/) - Preventing runaway workflows
- [Concurrency](/gh-aw/reference/concurrency/) - Serializing workflow execution
- [AI Engines](/gh-aw/reference/engines/) - Engine and model configuration
- [Schedule Syntax](/gh-aw/reference/schedule-syntax/) - Cron schedule format
- [GH-AW as an MCP Server](/gh-aw/reference/gh-aw-as-mcp-server/) - `agentic-workflows` tool for self-inspection
- [FAQ](/gh-aw/reference/faq/) - Common questions including cost and billing
