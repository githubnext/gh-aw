---
title: Auditing Workflows
description: Reference for the gh aw audit commands — single-run analysis, behavioral diff, and cross-run security reports.
sidebar:
  order: 297
---

The `gh aw audit` commands download workflow run artifacts and logs, analyze MCP tool usage and network behavior, and produce structured reports suited for security reviews, debugging, and feeding to AI agents.

## `gh aw audit <run-id-or-url> [<run-id-or-url>...]`

Audit one or more workflow runs. When a single run is provided, a detailed Markdown report is generated. When two or more runs are provided, the first is used as the base (reference) run and the remaining runs are compared against it, producing a diff report.

**Arguments:**

| Argument | Description |
|----------|-------------|
| `<run-id-or-url>` | A numeric run ID, GitHub Actions run URL, job URL, or job URL with step anchor |
| `[<run-id-or-url>...]` | Additional run IDs or URLs to compare against the first (diff mode) |

**Accepted input formats (per argument):**

- Numeric run ID: `1234567890`
- Run URL: `https://github.com/owner/repo/actions/runs/1234567890`
- Job URL: `https://github.com/owner/repo/actions/runs/1234567890/job/9876543210`
- Job URL with step: `https://github.com/owner/repo/actions/runs/1234567890/job/9876543210#step:7:1`
- Short run URL: `https://github.com/owner/repo/runs/1234567890`
- GitHub Enterprise URLs using the same formats above

When a job URL is provided without a step anchor (single-run mode), the command extracts the output of the first failing step. When a step anchor is included, it extracts that specific step.

In diff mode, job URLs and step-anchored URLs are accepted for any argument — the job/step specificity is silently normalized to the parent run ID, so it is always a run-level diff.

Self-comparisons and duplicate run IDs are rejected when using diff mode.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `-o, --output <dir>` | `./logs` | Directory to write downloaded artifacts and report files |
| `--json` | off | Output report as JSON to stdout |
| `--parse` | off | Run JavaScript parsers on agent and firewall logs, writing `log.md` and `firewall.md` (single-run only) |
| `--repo <owner/repo>` | auto | Specify repository when the run ID is not from a URL |
| `--stdin` | off | Read run IDs or URLs from stdin (one per line) instead of positional arguments |
| `--verbose` | off | Print detailed progress information |
| `--format <fmt>` | `pretty` | Diff output format: `pretty` or `markdown` (multi-run only) |

**Single-run examples:**

```bash
gh aw audit 1234567890
gh aw audit https://github.com/owner/repo/actions/runs/1234567890
gh aw audit 1234567890 --parse
gh aw audit 1234567890 --json
gh aw audit 1234567890 -o ./audit-reports
gh aw audit 1234567890 --repo owner/repo
```

**Stdin mode:**

Use `--stdin` to pass run IDs or URLs from a file or pipeline. This is mutually exclusive with positional arguments. Blank lines and lines starting with `#` are ignored. When passing bare numeric IDs (without embedded repo context), `--repo owner/repo` is required.

```bash
echo "1234567890" | gh aw audit --stdin
echo -e "1234567890\n9876543210" | gh aw audit --stdin   # diff mode: first is base
cat run-ids.txt | gh aw audit --stdin
cat run-ids.txt | gh aw audit --stdin --repo owner/repo  # required for bare numeric IDs
```

**Multi-run diff examples:**

```bash
gh aw audit 12345 12346                        # Compare two runs
gh aw audit 12345 12346 12347 12348            # Compare base against 3 runs
gh aw audit 12345 12346 --format markdown      # Markdown output for PR comments
gh aw audit 12345 12346 --json                 # JSON for CI integration
gh aw audit 12345 12346 --repo owner/repo      # Specify repository
```

**Single-run report sections** (rendered in Markdown or JSON): Overview, Comparison, Task/Domain, Behavior Fingerprint, Agentic Assessments, Metrics, Key Findings, Recommendations, Observability Insights, Performance Metrics, Engine Config, Prompt Analysis, Session Analysis, Safe Output Summary, MCP Server Health, Jobs, Downloaded Files, Missing Tools, Missing Data, Noops, MCP Failures, Firewall Analysis, Policy Analysis, Redacted Domains, Errors, Warnings, Tool Usage, MCP Tool Usage, Created Items.

The Metrics section includes an `ambient_context` object when available. Ambient context captures the first LLM inference footprint for the run:
- `ambient_context.input_tokens` — input tokens for the first invocation
- `ambient_context.cached_tokens` — cache-read tokens reused by the first invocation
- `ambient_context.effective_tokens` — `input_tokens + cached_tokens`

**Diff output** includes:
- New and removed network domains
- Domain status changes (allowed ↔ denied)
- Volume changes (request count changes above a 100% threshold)
- Anomaly flags (new denied domains, previously-denied domains now allowed)
- MCP tool invocation changes (new/removed tools, call count and error count diffs)
- Run metrics comparison (token usage, duration, turns)
- Token usage and spend breakdown: input tokens, output tokens, cache read/write tokens, effective tokens, AIC, total API requests, and cache efficiency per run
- Tokens per turn: effective tokens divided by turn count for each run, with the change between runs
- AIC reporting: AI Credits are shown alongside token metrics for spend tracking
- Tool call breakdown: per-tool call counts (new, removed, and changed tools) with max input/output sizes
- Bash command breakdown: aggregated call counts and max input/output sizes for each distinct bash command invoked

**Diff output behavior with multiple comparisons:**
- `--json` outputs a single object for one comparison, or an array for multiple
- `--format pretty` and `--format markdown` separate multiple diffs with dividers

## `gh aw logs --format <fmt>`

Generate a cross-run security and performance audit report across multiple recent workflow runs.
This feature is built into the `gh aw logs` command via the `--format` flag.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `[workflow]` | all workflows | Filter by workflow name or filename (positional argument) |
| `-c, --count <n>` | 10 | Number of recent runs to analyze |
| `--last <n>` | — | Alias for `--count` |
| `--format <fmt>` | — | Output format: `markdown` or `pretty` (generates cross-run audit report) |
| `--json` | off | Output cross-run report as JSON (when combined with `--format`) |
| `--repo <owner/repo>` | auto | Specify repository |
| `-o, --output <dir>` | `./logs` | Directory for downloaded artifacts |
| `--stdin` | off | Read run IDs or URLs from stdin (one per line) instead of run-discovery; content filters still apply |
| `--verbose` | off | Print detailed progress |

The report output includes an executive summary, domain inventory, metrics trends, MCP server health, and per-run breakdown. It detects cross-run anomalies such as domain access spikes, elevated MCP error rates, and connection rate changes.

For each run in detailed logs JSON output, an `ambient_context` object is included when token usage data is available. It reflects only the first LLM invocation in the run (`input_tokens`, `cached_tokens`, `effective_tokens`).

**`--stdin` mode:** Pass `--stdin` to supply an explicit list of run IDs or URLs instead of letting the command discover runs from the GitHub API. Date, count, and workflow-name filters are ignored; `--engine`, `--firewall`, `--safe-output`, and other content filters still apply. Blank lines and `#`-prefixed lines are ignored. Bare numeric IDs require `--repo owner/repo`.

```bash
cat run-ids.txt | gh aw logs --stdin
echo "1234567890" | gh aw logs --stdin --engine claude
cat run-ids.txt | gh aw logs --stdin --repo owner/repo   # required for bare numeric IDs
```

**Examples:**

```bash
gh aw logs --format markdown
gh aw logs daily-repo-status --format markdown --count 10
gh aw logs agent-task --format markdown --last 5 --json
gh aw logs --format pretty
gh aw logs --format markdown --repo owner/repo --count 10
```

## Related Documentation

- [Cost Management](/gh-aw/reference/cost-management/) — Track AIC-first spend and token usage
- [Artifacts](/gh-aw/reference/artifacts/) — Artifact names, directory structures, and token usage file locations (`token-usage.jsonl` in `firewall-audit-logs`)
- [Effective Tokens Specification](/gh-aw/reference/effective-tokens-specification/) — Legacy effective token computation details
- [Network](/gh-aw/reference/network/) — Firewall and domain allow/deny configuration
- [MCP Gateway](/gh-aw/reference/mcp-gateway/) — MCP server health and debugging
- [CLI Commands](/gh-aw/setup/cli/) — Full CLI reference

## Consuming Audit Reports in Workflows

When running locally, all three audit commands accept `--json` to write structured output to stdout. Pipe through `jq` to extract the fields a model needs.

| Command | Use case |
| --------- | ---------- |
| `gh aw audit <run-id> --json` | Single run — `key_findings`, `recommendations`, `metrics` |
| `gh aw logs [workflow] --last 10 --json` | Trend analysis — `per_run_breakdown`, `domain_inventory` |
| `gh aw audit <id1> <id2> --json` | Before/after — `run_metrics_diff`, `firewall_diff` |

Inside GitHub Actions workflows, agents access these commands through the `agentic-workflows` MCP tool rather than calling the CLI directly.

### Posting findings as a PR comment

```aw wrap
---
description: Post audit findings as a PR comment after each agent run

on:
  workflow_run:
    workflows: ['my-workflow']
    types: [completed]

engine: copilot

tools:
  github:
    toolsets: [pull_requests]
  agentic-workflows:

permissions:
  contents: read
  actions: read
  pull-requests: write
---

# Summarize Audit Findings

Use the `agentic-workflows` MCP tool `audit` with run ID ${{ github.event.workflow_run.id }}, identify the pull request that triggered it, and post a comment summarizing key findings and blocked domains. Highlight issues with severity `high` or `critical`. If there are no findings, post a brief "no issues found" comment.
```

### Detecting regressions with diff

```aw wrap
---
description: Detect regressions between two workflow runs

on:
  workflow_dispatch:
    inputs:
      base_run_id:
        description: 'Baseline run ID'
        required: true
      current_run_id:
        description: 'Current run ID to compare'
        required: true

engine: copilot

tools:
  github:
    toolsets: [issues]
  agentic-workflows:

permissions:
  contents: read
  actions: read
  issues: write
---

# Regression Detection

Use the `agentic-workflows` MCP tool `audit` with run IDs ${{ inputs.base_run_id }} and ${{ inputs.current_run_id }} to compare the two runs. Check for new blocked domains, increased MCP error rates, cost increase > 20%, or token usage increase > 50%. If regressions are found, open a GitHub issue with a table from `run_metrics_diff`, affected domains from `firewall_diff`, and affected MCP tools from `mcp_tools_diff`.
```

### Filing issues from audit findings

```aw wrap
---
description: File GitHub issues for high-severity audit findings

on:
  workflow_run:
    workflows: ['my-workflow']
    types: [completed]

engine: copilot

tools:
  github:
    toolsets: [issues]
  agentic-workflows:

permissions:
  contents: read
  actions: read
  issues: write
---

# Auto-File Issues for Critical Findings

Use the `agentic-workflows` MCP tool `audit` with run ID ${{ github.event.workflow_run.id }}. Filter `key_findings` for severity `high` or `critical`. For each finding without a matching open issue, create one with the finding title, description, impact, and recommendations, labelled `audit-finding`. If no critical findings, call the `noop` safe output tool.
```

### Weekly audit monitoring agent

```aw wrap
---
description: Weekly audit digest with trend analysis

on:
  schedule: weekly

engine: copilot

tools:
  github:
    toolsets: [discussions]
  agentic-workflows:
  cache-memory:
    key: audit-monitoring-trends

permissions:
  contents: read
  actions: read
  discussions: write
---

# Weekly Audit Monitoring Digest

1. Use the `agentic-workflows` MCP tool `logs` with parameters `workflow: my-workflow, last: 10` and read `/tmp/gh-aw/cache-memory/audit-trends.json` as the previous baseline.
2. Detect: cost spikes (`cost_spike: true` in `per_run_breakdown`), new denied domains in `domain_inventory`, MCP servers with `error_rate > 0.10` or `unreliable: true`, and week-over-week changes in `error_trend.runs_with_errors`.
3. Create a GitHub discussion "Audit Digest — [YYYY-MM-DD]" with an executive summary, anomalies table, and MCP health table.
4. Update `/tmp/gh-aw/cache-memory/audit-trends.json` with rolling averages (cost, tokens, error count, deny rate), keeping only the last 30 days.
```

Top-level fields (`key_findings`, `recommendations`, `metrics`, `firewall_analysis`, `mcp_tool_usage`) are stable; nested sub-fields may be extended but are not removed without deprecation. Add `--parse` to populate `behavior_fingerprint` and `agentic_assessments`. Cross-run JSON can be large — extract only the slices your model needs.
