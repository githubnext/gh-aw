---
title: Audit Commands
description: Reference for the gh aw audit commands — single-run analysis, behavioral diff, and cross-run security reports.
sidebar:
  order: 297
---

The `gh aw audit` commands download workflow run artifacts and logs, analyze MCP tool usage and network behavior, and produce structured reports suited for security reviews, debugging, and feeding to AI agents.

## `gh aw audit <run-id-or-url>`

Audit a single workflow run and generate a detailed Markdown report.

**Arguments:**

| Argument | Description |
|----------|-------------|
| `<run-id-or-url>` | A numeric run ID, GitHub Actions run URL, job URL, or job URL with step anchor |

**Accepted input formats:**

- Numeric run ID: `1234567890`
- Run URL: `https://github.com/owner/repo/actions/runs/1234567890`
- Job URL: `https://github.com/owner/repo/actions/runs/1234567890/job/9876543210`
- Job URL with step: `https://github.com/owner/repo/actions/runs/1234567890/job/9876543210#step:7:1`
- Short run URL: `https://github.com/owner/repo/runs/1234567890`
- GitHub Enterprise URLs using the same formats above

When a job URL is provided without a step anchor, the command extracts the output of the first failing step. When a step anchor is included, it extracts that specific step.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `-o, --output <dir>` | `./logs` | Directory to write downloaded artifacts and report files |
| `--json` | off | Output report as JSON to stdout |
| `--parse` | off | Run JavaScript parsers on agent and firewall logs, writing `log.md` and `firewall.md` |
| `--repo <owner/repo>` | auto | Specify repository when the run ID is not from a URL |
| `--verbose` | off | Print detailed progress information |

**Examples:**

```bash
gh aw audit 1234567890
gh aw audit https://github.com/owner/repo/actions/runs/1234567890
gh aw audit 1234567890 --parse
gh aw audit 1234567890 --json
gh aw audit 1234567890 -o ./audit-reports
gh aw audit 1234567890 --repo owner/repo
```

**Report sections** (rendered in Markdown or JSON): Overview, Comparison, Task/Domain, Behavior Fingerprint, Agentic Assessments, Metrics, Key Findings, Recommendations, Observability Insights, Performance Metrics, Engine Config, Prompt Analysis, Session Analysis, Safe Output Summary, MCP Server Health, Jobs, Downloaded Files, Missing Tools, Missing Data, Noops, MCP Failures, Firewall Analysis, Policy Analysis, Redacted Domains, Errors, Warnings, Tool Usage, MCP Tool Usage, Created Items.

## `gh aw audit diff <base-run-id> <comparison-run-id> [<comparison-run-id>...]`

Compare behavior between workflow runs. Detects policy regressions, new unauthorized domains, behavioral drift, and changes in MCP tool usage or run metrics.

**Arguments:**

| Argument | Description |
|----------|-------------|
| `<base-run-id>` | Numeric run ID for the baseline run |
| `<comparison-run-id>` | Numeric run ID for the comparison run |
| `[<comparison-run-id>...]` | Additional run IDs to compare against the same base |

The base run is downloaded once and reused when multiple comparison runs are provided. Self-comparisons and duplicate run IDs are rejected.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--format <fmt>` | `pretty` | Output format: `pretty` or `markdown` |
| `--json` | off | Output diff as JSON |
| `--repo <owner/repo>` | auto | Specify repository |
| `-o, --output <dir>` | `./logs` | Directory for downloaded artifacts |
| `--verbose` | off | Print detailed progress |

The diff output includes:
- New and removed network domains
- Domain status changes (allowed ↔ denied)
- Volume changes (request count changes above a 100% threshold)
- Anomaly flags (new denied domains, previously-denied domains now allowed)
- MCP tool invocation changes (new/removed tools, call count and error count diffs)
- Run metrics comparison (token usage, duration, turns)
- Token usage breakdown: input tokens, output tokens, cache read/write tokens, effective tokens, total API requests, and cache efficiency per run

**Output behavior with multiple comparisons:**
- `--json` outputs a single object for one comparison, or an array for multiple
- `--format pretty` and `--format markdown` separate multiple diffs with dividers

**Examples:**

```bash
gh aw audit diff 12345 12346
gh aw audit diff 12345 12346 12347 12348
gh aw audit diff 12345 12346 --format markdown
gh aw audit diff 12345 12346 --json
gh aw audit diff 12345 12346 --repo owner/repo
```

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
| `--verbose` | off | Print detailed progress |

The report output includes an executive summary, domain inventory, metrics trends, MCP server health, and per-run breakdown. It detects cross-run anomalies such as domain access spikes, elevated MCP error rates, and connection rate changes.

**Examples:**

```bash
gh aw logs --format markdown
gh aw logs daily-repo-status --format markdown --count 10
gh aw logs agent-task --format markdown --last 5 --json
gh aw logs --format pretty
gh aw logs --format markdown --repo owner/repo --count 10
```

## Related Documentation

- [Cost Management](/gh-aw/reference/cost-management/) — Track token usage and inference spend
- [Effective Tokens Specification](/gh-aw/reference/effective-tokens-specification/) — How effective tokens are computed
- [Network](/gh-aw/reference/network/) — Firewall and domain allow/deny configuration
- [MCP Gateway](/gh-aw/reference/mcp-gateway/) — MCP server health and debugging
- [CLI Commands](/gh-aw/setup/cli/) — Full CLI reference
