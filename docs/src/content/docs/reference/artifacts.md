---
title: Artifacts
description: Complete reference for artifact names, directory structures, and download patterns used by GitHub Agentic Workflows.
sidebar:
  order: 298
---

GitHub Agentic Workflows upload several artifacts during workflow execution. This reference documents every artifact name, its contents, and how to access the data — especially for downstream workflows that use `gh run download` directly instead of `gh aw logs`.

## Quick Reference

| Artifact Name | Constant | Type | Description |
|---------------|----------|------|-------------|
| `agent` | `constants.AgentArtifactName` | Multi-file | Unified agent job outputs (logs, safe outputs, token usage summary) |
| `activation` | `constants.ActivationArtifactName` | Multi-file | Activation job output (`aw_info.json`, `prompt.txt`, rate limits) |
| `firewall-audit-logs` | `constants.FirewallAuditArtifactName` | Multi-file | AWF firewall audit/observability logs (token usage, network policy, audit trail) |
| `detection` | `constants.DetectionArtifactName` | Single-file | Threat detection log (`detection.log`) |
| `safe-output` | `constants.SafeOutputArtifactName` | Legacy/back-compat | Historical standalone safe output artifact (`safe_output.jsonl`); in current compiled workflows this content is included in the unified `agent` artifact instead |
| `agent-output` | `constants.AgentOutputArtifactName` | Legacy/back-compat | Historical standalone agent output artifact (`agent_output.json`); in current compiled workflows this content is included in the unified `agent` artifact instead |
| `aw-info` | — | Single-file | Engine configuration (`aw_info.json`) |
| `prompt` | — | Single-file | Generated prompt (`prompt.txt`) |
| `experiment` | `constants.ExperimentArtifactName` | Multi-file | A/B experiment state (`state.json`) uploaded by the activation job when experiments are declared in the frontmatter |
| `safe-outputs-items` | `constants.SafeOutputItemsArtifactName` | Single-file | Safe output items manifest |
| `code-scanning-sarif` | `constants.SarifArtifactName` | Single-file | SARIF file for code scanning results |

## Artifact Sets

The `gh aw logs` and `gh aw audit` commands support `--artifacts` to download only specific artifact groups:

| Set Name | Artifacts Downloaded | Use Case |
|----------|---------------------|----------|
| `all` | Everything | Full analysis (default) |
| `agent` | `agent` | Agent logs and outputs |
| `activation` | `activation` | Activation data (`aw_info.json`, `prompt.txt`) |
| `firewall` | `firewall-audit-logs` | Network policy and firewall audit data |
| `mcp` | `firewall-audit-logs` | MCP gateway traffic logs |
| `detection` | `detection` | Threat detection output |
| `experiment` | `experiment` | A/B experiment state (only present when experiments are declared) |
| `github-api` | `activation`, `agent` | GitHub API rate limit logs |

```bash
# Download only firewall artifacts
gh aw logs <run-id> --artifacts firewall

# Download agent and firewall artifacts
gh aw logs <run-id> --artifacts agent --artifacts firewall

# Download everything (default)
gh aw logs <run-id>
```

## `firewall-audit-logs`

The `firewall-audit-logs` artifact is uploaded by **all firewall-enabled workflows**. It contains AWF (Agent Workflow Firewall) structured audit and observability logs.

> **⚠️ Important:** This artifact is **separate** from the `agent` artifact. Token usage data (`token-usage.jsonl`) lives here, not in the `agent` artifact.

### Directory Structure

```
firewall-audit-logs/
├── api-proxy-logs/
│   └── token-usage.jsonl        ← Token usage data (input/output/cache tokens per API request)
├── squid-logs/
│   └── access.log               ← Network policy log (domain allow/deny decisions)
├── audit.jsonl                  ← Firewall audit trail (policy matches, rule evaluations)
└── policy-manifest.json         ← Policy configuration snapshot
```

### Accessing Token Usage Data

**Recommended: Use `gh aw logs`**

```bash
# Download and analyze firewall data
gh aw logs <run-id> --artifacts firewall

# Output as JSON for scripting
gh aw logs <run-id> --artifacts firewall --json
```

**Direct download with `gh run download`:**

```bash
# Download the firewall-audit-logs artifact
gh run download <run-id> -n firewall-audit-logs

# Token usage data is at:
cat firewall-audit-logs/api-proxy-logs/token-usage.jsonl

# Network access log is at:
cat firewall-audit-logs/squid-logs/access.log

# Audit trail is at:
cat firewall-audit-logs/audit.jsonl

# Policy manifest is at:
cat firewall-audit-logs/policy-manifest.json
```

### Common Mistake

Downstream workflows sometimes download `agent-artifacts` or `agent` expecting to find `token-usage.jsonl`. This will silently return no data — the token usage file is only in the `firewall-audit-logs` artifact.

```bash
# ❌ WRONG — token-usage.jsonl is NOT in the agent artifact
gh run download <run-id> -n agent
cat agent/token-usage.jsonl  # File not found!

# ✅ CORRECT — download from firewall-audit-logs
gh run download <run-id> -n firewall-audit-logs
cat firewall-audit-logs/api-proxy-logs/token-usage.jsonl
```

## `agent`

The unified `agent` artifact contains all agent job outputs.

### Contents

- Agent execution logs
- Safe output data (`agent_output.json`)
- GitHub API rate limit logs (`github_rate_limits.jsonl`)
- Token usage summary (`agent_usage.json`) — aggregated totals only; per-request data is in `firewall-audit-logs`
- `otel.jsonl` — OTLP span mirror written by gh-aw's JavaScript span exporters (only present when `observability.otlp` is configured)
- `copilot-otel.jsonl` — OTLP spans emitted by Copilot CLI (only present when `observability.otlp` is configured)

## `activation`

The `activation` artifact contains activation job outputs.

### Contents

- `aw_info.json` — Engine configuration and workflow metadata
- `prompt.txt` — The generated prompt sent to the AI agent
- `github_rate_limits.jsonl` — Rate limit data from the activation job

## `detection`

The `detection` artifact contains threat detection output.

### Contents

- `detection.log` — Threat detection analysis results

Legacy name: `threat-detection.log` (still supported for backward compatibility).

## `experiment`

The `experiment` artifact is uploaded by the **activation job** only when the workflow frontmatter declares one or more `experiments` entries. It is not present on runs without experiments.

### Contents

- `state.json` — Cumulative per-variant invocation counters used to balance A/B assignments across runs

### Accessing experiment data

```bash
# Download the experiment artifact for a specific run
gh aw audit <run-id> --artifacts experiment

# Display the A/B experiment section in the audit report
gh aw audit <run-id>
```

The `🧪 A/B Experiments` section of the audit report shows the variant chosen for the run and the cumulative counts:

```
🧪 A/B Experiments
  • style = concise (cumulative: concise:5, detailed:4)
```

See [A/B Experiments](/gh-aw/guides/experiments/) for how to declare experiments in workflow frontmatter.

## Naming Compatibility

Artifact names changed between upload-artifact v4 and v5. The `gh aw logs` and `gh aw audit` commands handle both naming schemes transparently:

| Old Name (pre-v5) | New Name (v5+) | File Inside |
|--------------------|----------------|-------------|
| `aw_info.json` | `aw-info` | `aw_info.json` |
| `safe_output.jsonl` | `safe-output` | `safe_output.jsonl` |
| `agent_output.json` | `agent-output` | `agent_output.json` |
| `prompt.txt` | `prompt` | `prompt.txt` |
| `threat-detection.log` | `detection` | `detection.log` |

Single-file artifacts are automatically flattened to root level regardless of their artifact directory name. Multi-file artifacts (`firewall-audit-logs`, `agent`, `activation`, `experiment`) retain their directory structure.

## Workflow Call Prefixes

When workflows are invoked via `workflow_call`, GitHub Actions prepends a short hash to artifact names (e.g., `abc123-firewall-audit-logs`). The CLI handles this automatically by matching artifact names that end with `-{base-name}`.

```bash
# Both of these are recognized as the firewall artifact:
# - firewall-audit-logs           (direct invocation)
# - abc123-firewall-audit-logs    (workflow_call invocation)
```

## Related Documentation

- [Audit Commands](/gh-aw/reference/audit/) — Download and analyze workflow run artifacts
- [Cost Management](/gh-aw/reference/cost-management/) — Track token usage and inference spend
- [Network](/gh-aw/reference/network/) — Firewall and domain allow/deny configuration
- [Compilation Process](/gh-aw/reference/compilation-process/) — How workflows are compiled including artifact upload steps
