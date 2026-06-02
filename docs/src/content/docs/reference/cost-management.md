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

The agent job invokes an AI engine to process the prompt and call tools. Inference is billed by the provider:

| Engine | Billed to | Unit |
|--------|-----------|------|
| `copilot` | Account owning [`COPILOT_GITHUB_TOKEN`](/gh-aw/reference/auth/#copilot_github_token) | Premium requests (1–2 per run; see [Copilot billing](https://docs.github.com/en/copilot/about-github-copilot/subscription-plans-for-github-copilot)) |
| `claude` | Anthropic account for [`ANTHROPIC_API_KEY`](/gh-aw/reference/auth/#anthropic_api_key) | Tokens |
| `codex` | OpenAI account for [`OPENAI_API_KEY`](/gh-aw/reference/auth/#openai_api_key) | Tokens |

> [!NOTE]
> For Copilot, inference is charged to the individual account owning `COPILOT_GITHUB_TOKEN`, not the repository or organization. Use a dedicated service account to track spend per workflow.

## Monitoring Costs with `gh aw logs`

The `gh aw logs` command surfaces per-run metrics — elapsed duration, token usage, and estimated inference cost — before you decide what to optimize. Use `gh aw audit <run-id>` to deep-dive into a single run's token usage, tool calls, and inference spend; its **Metrics** and **Performance Metrics** sections cover token counts, effective tokens, turn counts, and estimated cost in one place. For cost trends across multiple runs, use `gh aw logs --format markdown [workflow]` to generate a cross-run report with anomaly detection.

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

# List per-run duration and tokens across all workflows
gh aw logs --start-date -30d --json | \
  jq '.runs[] | {workflow: .workflow_name, duration: .duration, tokens: .token_usage}'

# Token usage grouped by workflow over the past 30 days
gh aw logs --start-date -30d --json | \
  jq '[.runs[]] | group_by(.workflow_name) |
  map({workflow: .[0].workflow_name, runs: length, total_tokens: (map(.token_usage) | add // 0)})'
```

Each run under `.runs[]` includes `duration`, `token_usage`, `workflow_name`, and `agent`. For orchestrated workflows, the same JSON includes deterministic lineage under `.episodes[]` and `.edges[]` — see the next section.

### Interpret Episode-Level Usage

`gh aw logs --json` emits three views of the same data: `.runs[]` (individual workflow runs), `.episodes[]` (related runs grouped into one logical execution — orchestrator, workers, `workflow_call` follow-ups, and reporting passes), and `.edges[]` (the inferred parent-child lineage). Use `.runs[]` to find which specific run was resource-heavy; use `.episodes[]` to answer "what did this job use end-to-end?". For non-orchestrated workflows, an episode collapses to a single run and the two views are equivalent.

Useful episode fields for usage analysis:

| Field | Meaning |
|-------|---------|
| `total_runs` | Workflow runs in the logical execution |
| `total_tokens` / `total_effective_tokens` | Raw and effective token aggregates; prefer `total_effective_tokens` for Copilot |
| `total_duration` | Wall-clock duration across grouped runs |
| `primary_workflow` | Main workflow label |
| `resource_heavy_node_count` | Runs flagged as resource-heavy |
| `blocked_request_count` | Aggregate blocked-network pressure |

For Copilot runs, `total_effective_tokens` is the most reliable proxy for resource usage — Copilot does not expose billing-grade cost data.

Safe-output actuation also appears in both `gh aw logs --json` (run- and repo-level) and `gh aw audit <run-id>` (under `safe_output_summary`). The relevant fields — `temporary_id_map_status`, `temporary_id_mappings`, `chained_target_count`, `chained_followup_action_count`, `delegated_temp_target_count`, `closed_temp_target_count`, and their repo-level aggregates — show how often a workflow follows up on its own outputs. When `temporary_id_map_status` is `missing` or `invalid`, chain counts fall back to `0` rather than guessing from incomplete data.

```bash
# Top 10 heaviest logical executions over the past 30 days by effective tokens
gh aw logs --start-date -30d --json | \
  jq '[.episodes[] | {episode: .episode_id, workflow: .primary_workflow, runs: .total_runs, effective_tokens: (.total_effective_tokens // 0)}]
      | sort_by(.effective_tokens) | reverse | .[:10]'
```

## Track Costs at Scale with OpenTelemetry

Use `observability.otlp` to stream run telemetry into a central
OpenTelemetry backend when one repository or one `gh aw logs`
report is no longer enough. This is the best fit for
organization-wide dashboards, alerting, and cross-repository cost
analysis.

```aw wrap
observability:
  otlp:
    endpoint: ${{ secrets.OTLP_ENDPOINT }}
    headers:
      Authorization: ${{ secrets.OTLP_TOKEN }}
```

The exported spans include workflow and model metadata such as
`gh-aw.engine.id`, `gen_ai.request.model`,
`gen_ai.usage.input_tokens`, and
`gen_ai.usage.output_tokens`. Use these attributes to group usage
by workflow, engine, model, repository, or team in the backend of
your choice.

OpenTelemetry is most useful for answering questions such as:
"Which repositories are driving the most token usage?",
"Which model change caused a cost spike?", and
"Which workflows should be moved to a smaller model or stricter
trigger policy?" See [OpenTelemetry](/gh-aw/reference/open-telemetry/)
for the full attribute reference and collector configuration.

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

Inference cost scales with prompt size. Write focused prompts, avoid whole-file reads when only a few lines matter, cap result counts in tool calls, and use `imports` to compose a smaller subset of prompt sections at runtime.

### Cap Effective Tokens per Run

Use the top-level `max-effective-tokens` frontmatter field to cap
the effective-token budget for a single workflow run. This provides
a hard stop for unusually expensive runs and a consistent cost
guardrail across all supported engines. The field accepts plain
integers or `K`/`M` suffixes such as `100M`.

```aw wrap
max-effective-tokens: 5M
```

Effective tokens are the normalized usage metric described in the
[Effective Tokens Specification](/gh-aw/reference/effective-tokens-specification/).
When the budget is approached, gh-aw emits steering warnings before
the run reaches the limit. Set a negative value only when budget
enforcement must be disabled explicitly.

### Cap Turns per Run

Use the top-level `max-turns` frontmatter field to cap the number
of chat iterations (model responses and tool calls) for a single
workflow run. Each additional turn consumes more tokens and Actions
compute time, so a turn limit bounds both runaway loops and cost.

```aw wrap
max-turns: 20
```

`max-turns` is supported across Claude, Codex, Copilot, and
Antigravity engines. When set, gh-aw exports the compiled value as
`GH_AW_MAX_TURNS` for the engine runtime — you do not need to set
`CLAUDE_CODE_MAX_TURNS` or an equivalent variable separately.

The field accepts integer literals or GitHub Actions expressions,
making it composable with `workflow_call` inputs:

```aw wrap
max-turns: ${{ inputs.max-turns || 15 }}
```

> [!NOTE]
> `engine.max-turns` is a deprecated alias for the top-level field
> and continues to compile for backward compatibility. Use
> `gh aw fix engine-max-turns-to-top-level` to migrate existing
> workflows automatically.

An enterprise-wide default can be set via the compiler process
environment variable `GH_AW_DEFAULT_MAX_TURNS`. Individual
workflows override this default by setting `max-turns` in
frontmatter.

### Cap Daily Effective Tokens per Workflow

Use `max-daily-effective-tokens` to set a 24-hour effective-token
cap for one workflow. The guardrail sums runs from the past 24 hours of the same
workflow started by the same triggering user.

```aw wrap
max-daily-effective-tokens: 15M
```

You can also configure the same threshold via environment variable
to make the guardrail configurable per environment or workflow call:

```aw wrap
env:
  GH_AW_MAX_DAILY_EFFECTIVE_TOKENS: ${{ vars.AWF_DAILY_ET_LIMIT }}
```

When the total from the past 24 hours already meets or exceeds this threshold, the activation
job warns, creates an issue, skips the agent job, and lets the
conclusion job report the failure context.

The guardrail is disabled by default when omitted. Set `-1` to disable
it explicitly. Positive values accept plain integers or `K`/`M`
suffixes such as `100M`.

> [!NOTE]
> The daily guardrail is skipped for `workflow_call`,
> `repository_dispatch`, and `workflow_dispatch` runs carrying internal
> `aw_context` dispatch metadata.

### Roll out org/repo defaults with enterprise controls

For large installations, set baseline model and token guardrails
once, then let individual workflows override only when needed:

1. Export current defaults:

```bash
gh aw env get defaults.yml --scope org --org MY_ORG
```

2. Update and apply shared defaults in batch:

```yaml
default_max_effective_tokens: "5M"
default_max_daily_effective_tokens: "15M"
default_model_copilot: "gpt-5-mini"
default_model_claude: "claude-haiku-4-5"
default_model_codex: "gpt-5.4-mini"
```

```bash
gh aw env update defaults.yml --scope org --org MY_ORG
```

`gh aw env update` shows a confirmation preview before applying changes.
Pass `--yes` to skip the prompt in automation, or `--dry-run` to preview
without changing any variables. Set a field to `null` to delete the
corresponding variable from the target scope. Unknown YAML keys are rejected,
`default_max_turns` / `default_timeout_minutes` must be positive integers, and
`default_max_effective_tokens` / `default_max_daily_effective_tokens` must be
non-zero integers (negative values disable the corresponding guardrail).

3. If you compile workflows in CI, pass compiler-read defaults into
the compiler process environment (for example via `${{ vars.* }}`):
`GH_AW_DEFAULT_MAX_EFFECTIVE_TOKENS`,
`GH_AW_DEFAULT_MAX_DAILY_EFFECTIVE_TOKENS`,
`GH_AW_DEFAULT_MAX_TURNS`,
`GH_AW_DEFAULT_TIMEOUT_MINUTES`,
`GH_AW_DEFAULT_DETECTION_MODEL`.

> [!TIP]
> `GH_AW_DEFAULT_MODEL_*` values are resolved at workflow runtime via
> `${{ vars.* }}` in compiled YAML, while timeout/max-turns/token
> defaults are read by the compiler process at compile time.

### Rate Limiting and Concurrency

Use `user-rate-limit` to cap how many times a user can trigger the workflow in a given window, and rely on concurrency controls to serialize runs rather than letting them pile up:

```aw wrap
user-rate-limit:
  max-runs-per-window: 3
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

The `agentic-workflows` MCP tool exposes the same operations as the CLI (`logs`, `audit`, `status`) to any workflow agent, so a scheduled meta-agent can inspect and optimize other agentic workflows automatically — fetching aggregate cost data, deep-diving into individual runs, and proposing frontmatter changes (cheaper model, tighter `skip-if-match`, lower `user-rate-limit`) via a pull request.

```aw wrap
description: Weekly Actions minutes cost report
on: weekly
permissions:
  actions: read
engine: copilot
tools:
  agentic-workflows:
```

### What to Optimize Automatically

| Signal | Automatic action |
|--------|-----------------|
| High token count per run | Switch to a smaller model (`gpt-4.1-mini`, `claude-haiku-4-5`) |
| High turn count per run | Set `max-turns` to cap iterations and prevent runaway loops |
| Frequent runs with no safe-output produced | Add or tighten `skip-if-match` |
| Long queue times due to concurrency | Lower `user-rate-limit.max-runs-per-window` or add a `concurrency` group |
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
> Create separate `COPILOT_GITHUB_TOKEN` service accounts per repository or team to attribute spend by workflow.

## Related Documentation

- [Audit Commands](/gh-aw/reference/audit/) - Single-run analysis, diff, and cross-run reporting
- [Artifacts](/gh-aw/reference/artifacts/) - Artifact names, directory structures, and token usage file locations
- [Effective Tokens Specification](/gh-aw/reference/effective-tokens-specification/) - How effective token counts are computed
- [OpenTelemetry](/gh-aw/reference/open-telemetry/) - Exporting workflow telemetry to centralized observability backends
- [Triggers](/gh-aw/reference/triggers/) - Configuring workflow triggers and skip conditions
- [Rate Limiting Controls](/gh-aw/reference/rate-limiting-controls/) - Preventing runaway workflows
- [Concurrency](/gh-aw/reference/concurrency/) - Serializing workflow execution
- [AI Engines](/gh-aw/reference/engines/) - Engine and model configuration
- [Compiler Enterprise Environment Controls](/gh-aw/reference/compiler-enterprise-environment-controls/) - Default model and guardrail precedence
- [Environment Variables](/gh-aw/reference/environment-variables/) - Variable scopes and compiler-managed defaults
- [Schedule Syntax](/gh-aw/reference/schedule-syntax/) - Cron schedule format
- [GH-AW as an MCP Server](/gh-aw/reference/gh-aw-as-mcp-server/) - `agentic-workflows` tool for self-inspection
- [FAQ](/gh-aw/reference/faq/) - Common questions including cost and billing
