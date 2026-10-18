---
title: Graders
description: Pure deterministic metrics from the unified agent session
---

Graders compute metrics without model calls, file reads, subprocesses, or
network access. They run in the **conclusion job**, after observations from
agent execution, safe outputs, experiments, and evals have been collected and
redacted in memory.

:::caution[Experimental]
Graders and their result format are experimental.
:::

## Default graders

No configuration is required. Omitting `graders` or setting `graders: {}` enables
these ten fast, linear-time built-ins:

| ID | Measurement |
|---|---|
| `tool-success-rate` | Fraction of tool calls that succeeded |
| `tool-failure-count` | Failed or unfinished tool calls |
| `retries` | Retry observations in main-agent MCP events |
| `loops` | Consecutive identical tool names and arguments |
| `trajectory-efficiency` | Unique tool names / total calls |
| `execution-step-count` | Observed model request count |
| `execution-duration` | Observed duration in milliseconds |
| `working-set-rebuild-factor` | Cumulative input tokens / peak invocation input tokens |
| `context-growth` | Total tokens / first-request tokens |
| `artifact-production` | Number of safe-output results |

Detection and eval usage do not count toward main-agent execution statistics.
Native tool calls take precedence over duplicate gateway records, and proxy
token telemetry takes precedence over aggregate usage. Missing required
evidence yields `unavailable` rather than an invented score.

## Configuration

Override thresholds, disable defaults, or enable additional registered built-ins:

```yaml
graders:
  loops:
    enabled: false
  tool-success-rate:
    threshold: 0.95
  skill-constraint-coverage:
    config:
      constraints:
        - id: uses-docs-skill
          pattern: "skill .*documentation"
```

Supported fields are `enabled`, `name`, `description`, `unit`, `direction`,
`threshold`, `min`, `max`, and `config`. Disabling every default is valid.
Only built-in IDs are accepted. Custom graders, `script`, `run`, and legacy
`operational-value` evaluators are not supported, even in disabled entries.

## Opt-in trajectory graders

These built-ins are not enabled by default:

| Family | IDs |
|---|---|
| Policy and constraints | `policy-near-miss`, `skill-constraint-coverage` |
| Exploration | `exploration-error`, `exploitation-error` |
| Recurrence | `state-revisit-probability-rep`, `recurrence-rate`, `recurrence-determinism`, `recurrence-laminarity`, `recurrence-trapping-time` |
| Sequence complexity | `event-entropy-rate`, `lempel-ziv-trajectory-complexity` |
| Provenance | `tool-output-consumption-rate`, `end-to-end-lineage-completeness`, `action-provenance-coverage` |
| Objectives | `premature-termination-gap`, `evidence-saturation-stopping-lag`, `dependency-order-violation-rate`, `objective-coverage`, `grounding-accuracy` |
| Reference comparison | `tool-wise-score`, `trajectory-ndtw`, `code-search-recall`, `code-read-precision`, `code-edit-precision` |
| Process model | `process-alignment-fitness` |

All use the same in-memory session interface. Internal projections cannot add
evidence absent from that session: analyses needing objectives, state,
references, or lineage report `unavailable` when those observations are missing.
Trajectory collections are bounded to 5,000 items for expensive analyses.
Importable fragments under `.github/workflows/shared/graders/` configure these
built-ins; they do not contain executable custom code.

## Output and replay

The `usage` artifact contains:

| File | Description |
|---|---|
| `graders/grader_manifest.json` | Effective built-ins, metadata, thresholds, and enabled states |
| `graders/grader_results.json` | Numeric or unavailable results, status, and implementation identity |
| `aw_session.jsonl` | Unified session including conclusion-provenance grader manifest and results |

The collector passes the redacted session array directly to `computeGraders`.
`createSessionLookups` provides shared event-type lookups, correlated tool calls,
and concrete accounting statistics. Graders never reread the persisted session
or source logs.

Replay one enabled built-in using the same implementation:

```bash
gh aw graders run weekly-research loops 123456789
cat aw_session.jsonl | gh aw graders run weekly-research loops
```

For normative requirements, see the [Graders Specification](/gh-aw/specs/graders-specification/).
