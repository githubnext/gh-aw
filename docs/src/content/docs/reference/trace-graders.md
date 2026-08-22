---
title: Trace Graders
description: Deterministic metrics computed from agent execution traces
---

Trace graders compute deterministic metrics from post-agent execution trace files (token usage, MCP gateway logs, agent output) without LLM calls or network access. Results are persisted in the agent artifact for downstream consumption by detection jobs and reporting tools.

:::caution[Experimental]
Trace graders are an experimental feature.
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
    script: "trace.toolCalls.filter(t => t.name === 'bash').length"
```

Custom scripts must be pure expressions (≤4 KB, no `require`, `import`, `fetch`, `eval`, or `process.exit`).

## Output files

| File | Description |
|---|---|
| `grader_manifest.json` | Which graders were configured and their enabled state |
| `grader_results.json` | Normalized metric values with trace summary |

Both files are included in the unified `agent` artifact.

## Execution

The graders step runs as an `if: always()` post-agent step in the existing agent job, after log parsing and before the unified artifact upload. It uses a single preprocessing pass over trace files shared by all graders.
