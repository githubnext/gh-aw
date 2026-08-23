---
title: Graders
description: Deterministic execution and operational value metrics
---

Graders compute deterministic metrics without LLM calls. Built-in and custom inline graders inspect post-agent execution traces. The reserved `value` grader evaluates operational repository outcomes under a frozen function and explicit evidence cutoff. Results are persisted in the agent artifact for downstream tools.

For normative requirements, see the [Graders Specification](/gh-aw/specs/graders-specification/).

:::caution[Experimental]
Graders are an experimental feature.
:::

## Quick start

```yaml
graders: {}
```

An empty map enables all built-in graders with default settings. Omitting the `graders` field entirely disables grading (no step is emitted).

## Built-in graders

| ID | Description | Value |
|---|---|---|
| `tool-success-rate` | Fraction of tool calls that succeeded | 0–1 |
| `tool-failure-count` | Number of failed tool calls | integer |
| `retries` | Count of retry events in MCP gateway logs | integer |
| `loops` | Consecutive identical tool calls (same name + args) | integer |
| `trajectory-efficiency` | Unique tool names / total tool calls | 0–1 |
| `execution-step-count` | Total LLM request count | integer |
| `execution-duration` | Total execution duration (ms) | integer |
| `working-set-rebuild-factor` | Cumulative input tokens / peak invocation input tokens | ≥1 |
| `context-growth` | Total tokens / first-request tokens | ≥1 |
| `artifact-production` | Count of outputs in agent_output.json | integer |

## Selective configuration

Disable a specific built-in:

```yaml
graders:
  loops:
    enabled: false
```

## Custom inline graders

Add a trusted inline JavaScript expression that receives the preprocessed `trace` object:

```yaml
graders:
  bash-calls:
    script: "return trace.toolCalls.filter(t => t.name === 'bash').length"
```

Custom scripts must return a value and stay within 4096 characters (no `require`, `import`, `fetch`, `eval`, or `process.exit`).

## Operational value grader

Configure the reserved `value` grader with a repository-relative Bash function:

```aw wrap
graders:
  value:
    function: .github/graders/daily-file-diet-value.sh
```

The compiler freezes the function bytes and records their SHA-256 digest. The function returns absolute operational attainment in `[0,1]` for the run's assigned case. A frozen baseline is optional metadata; when present, gh-aw derives `deltaFromBaseline` without changing the primary value.

Each result records the complete run subject, operational case, evidence time, maturity, and provenance. Value functions may query the repositories declared by their frozen evidence contract. They receive the workflow token through `GH_TOKEN` but do not receive workflow secrets.

Use the `aw-value` skill to design and verify a value function.

### Regrade a historical run

```bash
gh aw graders value 123456789 \
  --evidence-at 2026-08-30T12:00:00.000Z \
  --json
```

The command downloads the original grader artifact and reuses its case, run subject, and frozen function. The archived function must match the digest recorded by both the original manifest and result. Regrading emits a new observation identified by `(runId, functionDigest, evidenceAt)` and never modifies the original artifact. Use `--repo [HOST/]OWNER/REPO` to target another repository.

## Output files

| File | Description |
|---|---|
| `grader_manifest.json` | Which graders were configured and their enabled state |
| `grader_results.json` | Normalized values, status, implementation identity, and value observations |
| `value_function.sh` | Exact frozen value function used for initial grading and historical replay |

Both files are included in the unified `agent` artifact.

## Execution

The graders step runs as an `if: always()` post-agent step in the existing agent job, after log parsing and before the unified artifact upload. It uses a single preprocessing pass over trace files shared by all graders.
