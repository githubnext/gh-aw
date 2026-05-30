---
title: OTel Observability Specification
version: 0.3.0
status: Working Draft
date: 2026-05-19
last_updated: 2026-05-22
editors:
  - GitHub gh-aw Team
---

# OTel Observability Specification

This specification defines the normative OpenTelemetry and OTLP observability contract for GitHub Agentic Workflows (`gh-aw`). It covers workflow frontmatter configuration, normalization into runtime environment variables, MCP gateway propagation, local telemetry mirrors, and minimum implementation and test obligations.

This document is the repository-level source of truth for `observability.otlp` behavior in `gh-aw`. Informative documentation such as the published OpenTelemetry reference page may explain usage patterns, but the normative behavior belongs here.

## Abstract

GitHub Agentic Workflows emits distributed tracing data using OpenTelemetry concepts and OTLP-compatible exporters. That behavior spans compiler-time schema validation, workflow environment injection, JavaScript runtime helpers, MCP gateway trace propagation, and fallback local JSONL mirrors.

Without a single normative contract, these layers drift easily: frontmatter may accept shapes that runtime code does not honor, runtime code may emit variables not described elsewhere, and gateway integration may accidentally expose credentials or lose trace context.

This specification defines the required behavior for the current `gh-aw` OTel observability model so that compiler, runtime, tests, and future changes stay synchronized.

## Status of This Document

This is a Working Draft specification. It may be revised as `gh-aw` observability evolves, especially around multi-endpoint fan-out, helper APIs, and artifact-level telemetry reconciliation.

Changes to `observability.otlp`, OTLP environment injection, MCP gateway tracing, or the telemetry mirror contract SHOULD update this specification in the same change set.

## Table of Contents

1. [Purpose and Scope](#1-purpose-and-scope)
2. [Conformance](#2-conformance)
3. [Definitions](#3-definitions)
4. [Configuration Model](#4-configuration-model)
5. [Runtime Environment Contract](#5-runtime-environment-contract)
6. [Export and Gateway Integration](#6-export-and-gateway-integration)
7. [Local Mirrors and Artifacts](#7-local-mirrors-and-artifacts)
8. [Security and Privacy Requirements](#8-security-and-privacy-requirements)
9. [Trace Model](#9-trace-model)
10. [Span Attribute Contract](#10-span-attribute-contract)
    - [10.7 MCP Gateway Span Attribute Contract](#107-mcp-gateway-span-attribute-contract)
11. [Resource Attributes](#11-resource-attributes)
12. [Trace ID Propagation and Lookup](#12-trace-id-propagation-and-lookup)
13. [Implementation Mapping](#13-implementation-mapping)
14. [Compliance Testing](#14-compliance-testing)
15. [References](#15-references)
16. [Change Log](#16-change-log)

---

## 1. Purpose and Scope

### 1.1 Purpose

This specification exists to ensure that `gh-aw` observability behavior is specification-first, testable, and safe by default.

It defines what a conforming `gh-aw` implementation MUST do when a workflow declares `observability.otlp`, when runtime trace context is present, and when telemetry export partially fails.

### 1.2 Scope

This specification covers:

- the `observability.otlp` frontmatter model;
- normalization of OTLP endpoint and header forms;
- workflow-level environment variable injection for OTLP export;
- OTLP multi-endpoint fan-out metadata;
- MCP gateway OpenTelemetry configuration derived from workflow observability settings;
- runtime trace-context variables used by helper libraries and gateway wiring;
- local JSONL telemetry mirrors written under `/tmp/gh-aw/`; and
- minimum implementation mapping and conformance tests.

This specification does not cover:

- vendor-specific dashboard design in Grafana, Datadog, Sentry, or other backends;
- downstream telemetry analysis workflows;
- general OpenTelemetry semantic conventions beyond the attributes explicitly required by `gh-aw`; or
- backend-specific retention, indexing, or alerting behavior.

### 1.3 Informative Documents

The following documents are informative companions and do not override this specification:

- [docs/src/content/docs/reference/open-telemetry.md](../docs/src/content/docs/reference/open-telemetry.md)
- [docs/src/content/docs/reference/frontmatter.md](../docs/src/content/docs/reference/frontmatter.md)
- [docs/src/content/docs/reference/mcp-gateway.md](../docs/src/content/docs/reference/mcp-gateway.md)

---

## 2. Conformance

An implementation conforms to this specification if it satisfies all MUST and MUST NOT requirements in Sections 4 through 12.

The key words **MUST**, **MUST NOT**, **SHOULD**, **SHOULD NOT**, and **MAY** are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### 2.1 Conformance Classes

This specification defines three conformance levels:

| Level | Requirements |
|---|---|
| **Level 1 - Config** | Correct parsing and normalization of `observability.otlp` and workflow environment injection as defined in Sections 4 and 5. |
| **Level 2 - Runtime** | Level 1 plus MCP gateway integration and degraded-mode export behavior from Section 6. |
| **Level 3 - Complete** | Level 2 plus local mirror, artifact, trace model, span attribute contract, resource attributes, trace ID propagation, implementation-mapping, and compliance obligations in Sections 7 through 12. |

---

## 3. Definitions

| Term | Definition |
|---|---|
| **OTLP entry** | A normalized `{url, headers}` endpoint record derived from workflow frontmatter. |
| **Primary OTLP endpoint** | The first normalized OTLP entry. This endpoint is used for backward-compatible single-endpoint environment variables. |
| **Fan-out endpoint set** | The ordered list of all normalized OTLP entries. |
| **Top-level headers** | The `observability.otlp.headers` field that only applies when `endpoint` is declared as a plain string. |
| **Per-endpoint headers** | The `headers` field nested inside an object or array entry in `observability.otlp.endpoint`. |
| **If-missing mode** | The `observability.otlp.if-missing` runtime behavior selector with values `error`, `warn`, or `ignore`. |
| **Telemetry mirror** | A local NDJSON or JSONL file written under `/tmp/gh-aw/` so spans remain inspectable even when OTLP export fails or is absent. |
| **Trace context variables** | Runtime variables such as `GITHUB_AW_OTEL_TRACE_ID` and `GITHUB_AW_OTEL_PARENT_SPAN_ID` used to correlate spans across steps and jobs. |

---

## 4. Configuration Model

### 4.1 Frontmatter Declaration

1. Workflows MAY declare an `observability.otlp` object.
2. When `observability.otlp` is absent, the compiler MUST NOT inject OTLP endpoint variables or gateway OTLP configuration.
3. The `observability.otlp` object MAY contain `endpoint`, `headers`, and `if-missing` fields.

### 4.2 Endpoint Forms

The `endpoint` field MUST accept exactly these forms:

1. **String form**: a single URL string.
2. **Object form**: a single object with `url` and optional `headers`.
3. **Array form**: an ordered array of objects, each with `url` and optional `headers`.

A conforming implementation MUST normalize all accepted endpoint forms into an ordered list of OTLP entries.

If the normalized list is empty, the implementation MUST behave as though OTLP export is disabled.

### 4.3 Header Forms

1. Top-level `observability.otlp.headers` MUST apply only to the string endpoint form.
2. Object and array endpoint entries MUST carry their own headers via per-endpoint `headers` fields.
3. Header declarations MUST accept either:
   - a map of header name to string value; or
   - a comma-separated raw `key=value` string.
4. Map-form headers MUST be normalized into a deterministic comma-separated `key=value` string sorted by header name.
5. Empty header maps or empty header strings MUST normalize to the empty string.

### 4.4 Endpoint-Specific Header Rewriting

When the resolved endpoint is a Sentry endpoint, a conforming implementation MUST rewrite the header name `Authorization` to `x-sentry-auth` during OTLP header normalization.

This rewrite applies to both map-form and string-form header declarations.

### 4.5 `if-missing`

1. The `if-missing` field MAY be `error`, `warn`, or `ignore`.
2. The default behavior when the field is absent or invalid MUST be `error`.
3. Invalid `if-missing` values SHOULD be ignored with a debug or diagnostic log message.
4. The `if-missing` mode governs runtime behavior for OTLP-dependent gateway setup and MUST NOT suppress normal workflow-level OTEL environment injection.

### 4.6 Static Endpoint Allowlisting

When an OTLP endpoint URL is statically resolvable at compile time, the compiler MUST extract its hostname and append that hostname to the workflow network allowlist.

GitHub Actions expressions such as `${{ secrets.OTLP_ENDPOINT }}` are not statically resolvable and MUST NOT produce compile-time allowlist entries.

---

## 5. Runtime Environment Contract

When at least one OTLP entry exists after normalization, the workflow-level environment block MUST include the following runtime contract.

### 5.1 Required Variables

| Variable | Required behavior |
|---|---|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | MUST be set to the primary OTLP endpoint URL. |
| `OTEL_SERVICE_NAME` | MUST be `gh-aw.<sanitized-workflow-id-or-name>` when a sanitized identifier is available; otherwise `gh-aw`. |
| `GH_AW_OTLP_ENDPOINTS` | MUST contain a compact JSON array of all normalized OTLP entries. |
| `OTEL_EXPORTER_OTLP_HEADERS` | MUST be set to the primary OTLP entry headers when the primary entry has non-empty headers. |
| `GH_AW_OTLP_ALL_HEADERS` | MUST contain the comma-joined headers for all configured endpoints when more than one endpoint exists and at least one endpoint has headers. |
| `GH_AW_OTLP_IF_MISSING` | MUST be set only when `if-missing` is `warn` or `ignore`. |

### 5.2 Service Name Contract

1. The service name MUST use `WorkflowID` when available.
2. If `WorkflowID` is absent, the implementation MUST fall back to the workflow display name.
3. The service-name suffix MUST be sanitized into a backend-safe lowercase token.
4. If no usable workflow identifier exists after sanitization, the service name MUST be `gh-aw`.

### 5.3 Backward Compatibility

The primary endpoint variables `OTEL_EXPORTER_OTLP_ENDPOINT` and `OTEL_EXPORTER_OTLP_HEADERS` exist for backward compatibility and legacy consumers. A conforming implementation MUST preserve the first-entry semantics for those variables even when multiple endpoints are configured.

---

## 6. Export and Gateway Integration

### 6.1 Multi-Endpoint Fan-Out

1. A conforming implementation MUST preserve the declared endpoint order when normalizing array-form endpoint entries.
2. The fan-out endpoint set encoded in `GH_AW_OTLP_ENDPOINTS` MUST include every valid normalized endpoint.
3. Failure to export to one endpoint SHOULD NOT prevent attempts to export to remaining endpoints.

### 6.2 MCP Gateway OpenTelemetry Configuration

When OTLP export is configured for the workflow, the MCP gateway runtime configuration MUST include an `opentelemetry` object with:

- `endpoint` set from `${OTEL_EXPORTER_OTLP_ENDPOINT}`
- `traceId` set from `${GITHUB_AW_OTEL_TRACE_ID}`
- `spanId` set from `${GITHUB_AW_OTEL_PARENT_SPAN_ID}`

The gateway JSON configuration MUST NOT embed OTLP authentication headers directly.

### 6.3 Gateway Container Environment

When MCP gateway tracing is enabled, the gateway container invocation MUST receive:

- `GITHUB_AW_OTEL_TRACE_ID`
- `GITHUB_AW_OTEL_PARENT_SPAN_ID`
- `OTEL_EXPORTER_OTLP_HEADERS`

Passing `OTEL_EXPORTER_OTLP_HEADERS` through the environment is REQUIRED so credentials do not transit the stdin JSON configuration pipe.

### 6.4 Missing-Value Behavior

1. `if-missing: error` MUST treat unresolved runtime OTLP values as fatal for OTLP-dependent gateway setup.
2. `if-missing: warn` MUST emit a warning and skip gateway OTLP configuration.
3. `if-missing: ignore` MUST skip gateway OTLP configuration without warning.
4. In all modes, normal workflow-level OTEL environment injection MAY still occur when values are declared.

### 6.5 Trace Context Variables

The runtime setup layer SHOULD provide valid `GITHUB_AW_OTEL_TRACE_ID` and `GITHUB_AW_OTEL_PARENT_SPAN_ID` values to downstream helpers and gateway consumers when a valid trace and parent span exist for the job.

### 6.6 Fan-Out Operations Flow and Partial Failure Handling

For multi-endpoint OTLP export, a conforming implementation MUST execute this operations flow:

1. Normalize configured endpoints into ordered OTLP entries (Section 4) and select index `0` as primary compatibility endpoint.
2. Emit the local mirror event first (or in a non-blocking parallel path) so telemetry survives remote exporter failure.
3. Attempt export to each endpoint in declared order, recording per-endpoint status (`success`, `transient_failure`, `permanent_failure`).
4. On transient failure (`5xx`, timeout, transport reset, `429`), continue fan-out to remaining endpoints and schedule retry for failed endpoints only.
5. On permanent failure (`4xx` non-rate-limit, invalid auth format), continue fan-out to remaining endpoints and record a non-retriable error.
6. Complete the workflow step without making observability export globally fatal unless an explicit `if-missing: error` setup policy requires fail-fast behavior for missing mandatory configuration.

Partial endpoint failure MUST NOT suppress successful exports to other endpoints, and MUST remain visible through local mirror artifacts and warning telemetry.

---

## 7. Local Mirrors and Artifacts

### 7.1 Local Telemetry Mirror

1. Helper-driven span emission MUST append a JSON line to `/tmp/gh-aw/otel.jsonl` even when no OTLP endpoint is configured.
2. Helper-driven span emission MUST append a JSON line to `/tmp/gh-aw/otel.jsonl` even when OTLP export fails after retries.
3. Local mirror writes MUST occur before or independently of remote exporter success so telemetry is recoverable under degraded backend conditions.

### 7.2 Artifact Expectations

When workflow observability artifacts are collected, implementations SHOULD include local OTEL mirror files such as `otel.jsonl` and runtime-specific companion files such as `copilot-otel.jsonl` when present.

### 7.3 Non-Fatal Helper Behavior

The JavaScript OTLP helper layer SHOULD remain non-fatal:

- export failures SHOULD surface as warnings rather than hard failures; and
- missing or invalid runtime trace context SHOULD skip span emission rather than crash the workflow step.

---

## 8. Security and Privacy Requirements

1. OTLP authentication headers MUST be masked before they can appear in runner logs.
2. OTLP authentication headers MUST NOT be embedded in generated gateway JSON configuration.
3. Telemetry helper layers SHOULD redact or sanitize sensitive attribute values before writing local mirrors or sending OTLP payloads.
4. Observability failures MUST be treated as degraded-mode conditions and SHOULD NOT become workflow-fatal unless the active `if-missing` policy explicitly requires failure for setup correctness.
5. Implementations SHOULD avoid emitting raw prompt text, secrets, or credential material as span attributes.

---

## 9. Trace Model

### 9.1 Overview

gh-aw emits OpenTelemetry trace spans directly to configured OTLP-compatible vendor endpoints. gh-aw does **not** require or run an OpenTelemetry Collector. All transformation, batching, retry, endpoint selection, and authentication happens in-process before sending to the vendor OTLP endpoint.

Tracing is best-effort. Export failures MUST NOT fail the workflow.

### 9.2 Span Naming Convention

All gh-aw span names MUST follow the pattern: `gh-aw.<job-name>.<operation>`.

When no job name is available, the fallback `job` MUST be used, yielding names such as `gh-aw.job.setup`.

### 9.3 Span Hierarchy

A single trace ID is shared across all jobs in a workflow run. All setup spans share a global parent span ID so they render as siblings in OTLP backends.

```text
Single Trace: trace_id (32-char hex, shared across all jobs in a run)
├── Root Setup Parent: parent_span_id (global, shared across all jobs)
│
├── Activation Job
│   ├── gh-aw.activation.setup        (parent: root setup parent)
│   └── gh-aw.activation.conclusion   (parent: activation setup span)
│
├── Agent Job
│   ├── gh-aw.agent.setup             (parent: root setup parent)
│   ├── gh-aw.agent.conclusion         (parent: agent setup span)
│   │   └── gh-aw.agent.agent          (parent: agent conclusion span)
│   │       [dedicated AI latency measurement]
│   │
│   └── MCP Gateway (service: mcp-gateway)
│       ├── gateway.request            (parent: agent setup span)
│       │   └── mcp.tool_call          (parent: gateway.request)
│       │       └── gateway.backend.execute  (parent: mcp.tool_call)
│       ├── gateway.request            (repeated per MCP request)
│       │   └── ...
│       └── API Proxy (when DIFC active)
│           ├── proxy.difc_pipeline    (parent: gateway.request)
│           │   └── proxy.backend.forward  (parent: proxy.difc_pipeline)
│           └── ...
│
└── Other Jobs
    ├── gh-aw.<job-name>.setup         (parent: root setup parent)
    └── gh-aw.<job-name>.conclusion    (parent: job setup span)
```

### 9.4 Span Kinds

Span kind assignments MUST follow these rules:

| Span | OTLP `kind` | Rationale |
|---|---|---|
| `gh-aw.*.setup` | `SPAN_KIND_INTERNAL` (1) | Internal job lifecycle |
| `gh-aw.*.conclusion` | `SPAN_KIND_INTERNAL` (1) | Internal job lifecycle |
| `gh-aw.*.agent` | `SPAN_KIND_CLIENT` (3) | Outbound AI model request |
| `gateway.request` | `SPAN_KIND_SERVER` (2) | MCP gateway HTTP handler |
| `mcp.tool_call` | `SPAN_KIND_INTERNAL` (1) | MCP tool call lifecycle |
| `gateway.backend.execute` | `SPAN_KIND_CLIENT` (3) | Backend MCP server invocation |
| `proxy.difc_pipeline` | `SPAN_KIND_INTERNAL` (1) | API proxy DIFC evaluation |
| `proxy.backend.forward` | `SPAN_KIND_CLIENT` (3) | API proxy upstream request |

### 9.5 Span Status

Conclusion spans MUST set `status.code` based on the job outcome:

| Outcome | `status.code` |
|---|---|
| `success` | `OK` (1) |
| `failure`, `timeout`, `cancelled` | `ERROR` (2) |

### 9.6 Exception Events

When errors are present in `agent_output.json`, the conclusion span MUST emit OTel exception events:

```json
{
  "timeUnixNano": "...",
  "name": "exception",
  "attributes": [
    {"key": "exception.type", "value": {"stringValue": "gh-aw.<ErrorType>"}},
    {"key": "exception.message", "value": {"stringValue": "Error description"}}
  ]
}
```

Exception type resolution:

1. If the error message matches the format `type:message`, use `gh-aw.<type>` as the exception type.
2. Otherwise, derive the type from the run status: `gh-aw.AgentError`, `gh-aw.AgentFailed`, `gh-aw.AgentTimedOut`, or `gh-aw.AgentCancelled`.

---

## 10. Span Attribute Contract

This section defines the attributes each span type MUST or MAY carry.

### 10.1 Setup Span Attributes

**Required attributes** (MUST be present on every setup span):

| Attribute | Type | Description |
|---|---|---|
| `gh-aw.job.name` | string | Job name from action input |
| `gh-aw.workflow.name` | string | Workflow name or ID |
| `gh-aw.run.id` | string | GitHub Actions run ID |
| `gh-aw.run.attempt` | string | Run attempt number |
| `gh-aw.run.actor` | string | User or bot initiating the run |
| `gh-aw.repository` | string | `owner/repo` |
| `gh-aw.staged` | boolean | Whether this is a staging deployment |

**Conditional attributes** (MUST be present when the value is available):

| Attribute | Type | Description |
|---|---|---|
| `gen_ai.system` | string | Mapped AI system name (e.g., `github_models`, `anthropic`, `openai`) |
| `gh-aw.engine.id` | string | Raw engine identifier (`copilot`, `claude`, `codex`, `gemini`, custom) |
| `gh-aw.event_name` | string | GitHub event type |
| `gh-aw.trigger.item_type` | string | Triggering item (`issue`, `pull_request`, `discussion`, etc.) |
| `gh-aw.trigger.item_number` | string | Triggering item ID/number |
| `gh-aw.trigger.label` | string | Label on triggering item |
| `gh-aw.trigger.comment_id` | string | Comment ID on triggering item |
| `gh-aw.episode.id` | string | Episode/session ID for cross-run correlation |
| `gh-aw.episode.kind` | string | `run` or `workflow_call` |
| `gh-aw.hop.id` | string | Current workflow invocation ID |
| `gh-aw.hop.parent_id` | string | Parent workflow invocation ID |
| `gh-aw.origin.event` | string | Origin event type |
| `gh-aw.root.repo` | string | Root repository (for dispatched workflows) |
| `gh-aw.root.workflow_id` | string | Root workflow ID |
| `gh-aw.frontmatter.source` | string | Frontmatter source type |
| `gh-aw.frontmatter.emoji` | string | Frontmatter emoji |
| `gh-aw.frontmatter.body_modified` | boolean | Whether body was edited |
| `gh-aw.experiment.<name>` | string | Per-experiment variant assignment |
| `gh-aw.experiments` | string | Compact JSON of all experiment assignments |
| `gh-aw.deployment.state` | string | Deployment status |
| `gh-aw.workflow_run.conclusion` | string | Workflow-level outcome |

### 10.2 Conclusion Span Attributes

**Required attributes** (MUST be present on every conclusion span):

| Attribute | Type | Description |
|---|---|---|
| `gh-aw.workflow.name` | string | Workflow name |
| `gh-aw.run.id` | string | Run ID |
| `gh-aw.run.attempt` | string | Attempt number |
| `gh-aw.run.actor` | string | Actor |
| `gh-aw.repository` | string | Repository |
| `gh-aw.run.status` | string | Run outcome (`success`, `failure`, `timeout`, `cancelled`) |
| `gh-aw.error_count` | int | Number of errors |
| `gh-aw.warning_count` | int | Number of warnings |
| `gh-aw.action_minutes` | double | Duration in minutes |
| `gh-aw.output.item_count` | int | Safe output items produced |
| `gh-aw.otlp.export_errors` | int | Count of OTLP export failures during this run |

**Conditional attributes** (MUST be present when the value is available):

| Attribute | Type | Description |
|---|---|---|
| `gh-aw.job.name` | string | Job name |
| `gen_ai.system` | string | AI system |
| `gh-aw.engine.id` | string | Engine ID |
| `gen_ai.request.model` | string | Requested model name |
| `gh-aw.tracker.id` | string | Tracker identifier |
| `gh-aw.event_name` | string | Event type |
| `gh-aw.staged` | boolean | Staging flag |
| `gh-aw.trigger.*` | string | Trigger context (same fields as setup span) |
| `gh-aw.frontmatter.*` | string | Frontmatter metadata (same fields as setup span) |
| `gh-aw.effective_tokens` | int | Effective token count |
| `gh-aw.turns` | int | Number of agent turns |
| `gh-aw.agent.conclusion` | string | Agent job outcome |
| `gh-aw.detection.conclusion` | string | Threat detection outcome |
| `gh-aw.detection.reason` | string | Detection reasoning |
| `gh-aw.otlp.export_error_details` | string | Export failure details |
| `gh-aw.error.count` | int | Output error count |
| `gh-aw.error.messages` | string | Error messages joined by ` \| ` |
| `gh-aw.output.item_types` | string | Comma-separated types of safe output items |
| `gh-aw.github.rate_limit.remaining` | int | API rate limit remaining |
| `gh-aw.github.rate_limit.limit` | int | API rate limit total |
| `gh-aw.github.rate_limit.used` | int | API rate limit used |
| `gh-aw.github.rate_limit.resource` | string | Rate limit resource category |
| `gh-aw.github.rate_limit.reset` | string | ISO 8601 rate limit reset time |
| `gh-aw.outcome.total` | int | Total outcomes |
| `gh-aw.outcome.accepted` | int | Accepted outcomes |
| `gh-aw.outcome.rejected` | int | Rejected outcomes |
| `gh-aw.outcome.pending` | int | Pending outcomes |
| `gh-aw.outcome.ignored` | int | Ignored outcomes |
| `gh-aw.outcome.acceptance_rate` | double | Acceptance rate |
| `gh-aw.outcome.waste_rate` | double | Waste rate |

### 10.3 Agent Span Attributes

The dedicated agent span (`gh-aw.*.agent`) follows OpenTelemetry [GenAI semantic conventions](https://opentelemetry.io/docs/specs/semconv/gen-ai/).

**Required attributes** (MUST be present when available from the AI engine):

| Attribute | Type | Description |
|---|---|---|
| `gen_ai.system` | string | Mapped AI system name |
| `gen_ai.request.model` | string | Requested model |
| `gen_ai.response.model` | string | Resolved runtime model |
| `gen_ai.operation.name` | string | Always `"chat"` |
| `gen_ai.workflow.name` | string | Workflow name |
| `gen_ai.usage.input_tokens` | int | Input tokens consumed |
| `gen_ai.usage.output_tokens` | int | Output tokens generated |
| `gen_ai.usage.total_tokens` | int | Total tokens (input + output, excluding cache) |
| `gen_ai.response.finish_reasons` | string[] | Stop reasons (e.g., `["stop"]`, `["length"]`, `["timeout"]`) |

**Optional attributes** (MAY be present):

| Attribute | Type | Description |
|---|---|---|
| `gen_ai.usage.cache_read.input_tokens` | int | Cache read tokens |
| `gen_ai.usage.cache_creation.input_tokens` | int | Cache write tokens |

### 10.4 Outcome Evaluation Span Attributes

Per-item outcome evaluation spans (`gh-aw.outcome.evaluation`) are emitted by the outcome-collector workflow. Each span represents one safe output item evaluated against the GitHub API.

| Attribute | Type | Condition | Description |
|---|---|---|---|
| `gh-aw.outcome.type` | string | Required | Safe output type (e.g., `create_pull_request`, `create_issue`) |
| `gh-aw.outcome.result` | string | Required | `accepted`, `rejected`, `pending`, `ignored`, `noop` |
| `gh-aw.outcome.workflow` | string | Required | Source workflow name |
| `gh-aw.outcome.run_id` | int | Required | Source run ID |
| `gh-aw.outcome.repo` | string | Required | Repository |
| `gh-aw.outcome.url` | string | When available | URL to the created object |
| `gh-aw.outcome.detail` | string | When available | Result detail (e.g., `merged`, `closed`, `open`) |
| `gh-aw.outcome.created_at` | string | When available | Item creation timestamp |
| `gh-aw.outcome.event` | string | When available | Triggering event type |
| `gh-aw.outcome.resolution_sec` | int | When resolved | Seconds from creation to resolution |
| `gh-aw.outcome.pending_age_sec` | int | When pending | Seconds since creation |
| `gh-aw.outcome.review_comments` | int | PRs only | Number of review comments |
| `gh-aw.outcome.comments` | int | When available | Number of issue-level comments |
| `gh-aw.outcome.changed_files` | int | PRs only | Files changed |
| `gh-aw.outcome.additions` | int | PRs only | Lines added |
| `gh-aw.outcome.deletions` | int | PRs only | Lines deleted |
| `gh-aw.outcome.reactions_total` | int | When available | Total reaction count |
| `gh-aw.outcome.reactions_positive` | int | When available | Positive reactions (+1, heart, hooray, rocket) |
| `gh-aw.outcome.reactions_negative` | int | When available | Negative reactions (-1, confused) |
| `gh-aw.outcome.zero_touch` | boolean | When true | Accepted with no human review comments or issue comments |

### 10.5 Outcome Summary Span Attributes

The fleet summary span (`gh-aw.outcome.summary`) aggregates all evaluated outcomes into a single span with economics metrics.

| Attribute | Type | Description |
|---|---|---|
| `gh-aw.outcome.runs_checked` | int | Number of runs evaluated |
| `gh-aw.outcome.total` | int | Total actionable outcomes |
| `gh-aw.outcome.accepted` | int | Accepted outcomes |
| `gh-aw.outcome.rejected` | int | Rejected outcomes |
| `gh-aw.outcome.ignored` | int | Ignored outcomes |
| `gh-aw.outcome.pending` | int | Pending outcomes |
| `gh-aw.outcome.noop` | int | Noop outcomes |
| `gh-aw.outcome.acceptance_rate` | double | Accepted / (accepted + rejected) |
| `gh-aw.outcome.waste_rate` | double | Rejected / total |
| `gh-aw.outcome.noop_rate` | double | Noop / (total + noop) |
| `gh-aw.outcome.zero_touch_count` | int | Count of zero-touch accepted outcomes |
| `gh-aw.outcome.zero_touch_rate` | double | Zero-touch / accepted |
| `gh-aw.outcome.median_resolution_sec` | int | Median seconds from creation to resolution |
| `gh-aw.outcome.item_count` | int | Number of per-item spans emitted |
| `gh-aw.outcome.date` | string | Evaluation date (YYYY-MM-DD) |
| `gh-aw.outcome.events` | string | Comma-separated distinct trigger events |
| `gh-aw.outcome.workflows` | string | Comma-separated distinct workflow names |
| `gh-aw.outcome.types` | string | Comma-separated distinct outcome types |

### 10.7 MCP Gateway Span Attribute Contract

This section defines the attributes emitted by the MCP gateway (`gh-aw-mcpg`). These spans are emitted under the `mcp-gateway` service but share the workflow's trace ID (linked via `GITHUB_AW_OTEL_TRACE_ID` and `GITHUB_AW_OTEL_PARENT_SPAN_ID` passed to the gateway container per §6.3).

The canonical reference for gateway span attributes is [`gh-aw-mcpg/docs/otel-sentry.md`](https://github.com/github/gh-aw-mcpg/blob/main/docs/otel-sentry.md). This section summarizes the contract for cross-referencing with workflow-level spans.

#### 10.7.1 `gateway.request` Span

Emitted once per MCP JSON-RPC request received by the gateway.

| Attribute | Type | Description |
|---|---|---|
| `http.request.method` | string | HTTP method (always `POST`) |
| `http.response.status_code` | int | Response status code |
| `url.path` | string | Request path (e.g., `/mcp`) |
| `gen_ai.conversation.id` | string | Truncated session ID for correlation |
| `gateway.tag` | string | Handler log tag (e.g., `unified`, `routed:<backendID>`) |

#### 10.7.2 `mcp.tool_call` Span

Emitted once per tool call lifecycle. This is the key span for per-tool-call granularity.

| Attribute | Type | Description |
|---|---|---|
| `gen_ai.tool.name` | string | Tool name (e.g., `search_code`, `get_file_contents`, `create_issue`) |
| `gen_ai.agent.id` | string | Backend MCP server ID (e.g., `github`, `slack`, `playwright`) |
| `mcp.method` | string | JSON-RPC method (always `tools/call`) |
| `http.response.status_code` | int | Response status code (success or error) |

This span covers phases 0–6 of the tool call pipeline: parse, guard, route, execute, filter, respond.

#### 10.7.3 `gateway.backend.execute` Span

Emitted for each backend MCP server invocation.

| Attribute | Type | Description |
|---|---|---|
| `gen_ai.tool.name` | string | Tool name being executed |
| `gen_ai.agent.id` | string | Backend MCP server ID |
| `rate_limit.hit` | boolean | Whether a rate limit was triggered during execution |

Transport errors are recorded via `span.RecordError()` with a generic message (`"tool execution failed"`) to avoid leaking internal details.

#### 10.7.4 `proxy.difc_pipeline` Span

Emitted when the API proxy (DIFC enforcement) evaluates an outbound request.

| Attribute | Type | Description |
|---|---|---|
| `gen_ai.tool.name` | string | Tool name associated with the proxied request |
| `url.path` | string | Upstream API path |

#### 10.7.5 `proxy.backend.forward` Span

Emitted for each upstream GitHub API request forwarded through the proxy.

| Attribute | Type | Description |
|---|---|---|
| `gen_ai.tool.name` | string | Tool name associated with the forwarded request |
| `url.path` | string | Upstream API endpoint path |

---

## 11. Resource Attributes

Resource attributes are applied to all OTLP spans and describe the service and execution environment.

### 11.1 Required Resource Attributes

A conforming implementation MUST include these resource attributes on every exported span:

| Attribute | Type | Description | Example |
|---|---|---|---|
| `service.name` | string | `gh-aw.<workflow-id>` or `gh-aw` | `gh-aw.daily-report` |
| `service.version` | string | gh-aw CLI version or commit SHA | `v0.23.4` |
| `github.repository` | string | `owner/repo` | `github/gh-aw` |
| `github.run_id` | string | GitHub Actions run ID | `12345678` |
| `github.run_attempt` | string | Run attempt number | `1` |
| `github.actions.run_url` | string | URL to the run | `https://github.com/owner/repo/actions/runs/123` |

### 11.2 Conditional Resource Attributes

These resource attributes MUST be included when the corresponding value is available:

| Attribute | Type | Description |
|---|---|---|
| `github.event_name` | string | Event type (e.g., `push`, `pull_request`) |
| `github.ref` | string | Git ref (branch/tag) |
| `github.ref_name` | string | Ref name |
| `github.head_ref` | string | Head ref (for PRs) |
| `github.sha` | string | Commit SHA |
| `github.job` | string | Job name |
| `github.workflow_ref` | string | Workflow ref |
| `github.actor_id` | string | Actor ID |
| `runner.os` | string | Runner OS (`Linux`, `Windows`, `macOS`) |
| `runner.arch` | string | Runner architecture (`X64`, `ARM64`) |
| `runner.name` | string | Runner name/label |
| `runner.environment` | string | Runner environment |
| `gh-aw.awf.version` | string | Agentic Workflows Framework version |
| `gh-aw.awmg.version` | string | Agentic Workflows Manager version |
| `deployment.environment` | string | `staging` or `production` |

### 11.3 Instrumentation Scope

All gh-aw spans MUST be emitted under an instrumentation scope with:

| Field | Value |
|---|---|
| `scope.name` | `gh-aw` |
| `scope.version` | The gh-aw CLI version |

---

## 12. Trace ID Propagation and Lookup

### 12.1 Trace ID Format

The OTLP trace ID is a 32-character lowercase hexadecimal string (16 random bytes). The span ID is a 16-character lowercase hexadecimal string (8 random bytes).

Do **not** confuse the OTLP trace ID with `workflow_call_id`, which is derived from the GitHub run ID and attempt number. The OTLP trace ID is the value to search for in vendor backends (Sentry, Honeycomb, Datadog, Grafana Tempo, etc.).

### 12.2 Trace ID Resolution Order

The setup span MUST resolve the trace ID using the following priority order:

1. **Explicit option** — `options.traceId` passed to the setup function (used for activation job reuse).
2. **Action input** — `INPUT_TRACE_ID` environment variable (from `trace-id` action input, used for cross-job propagation).
3. **Parent context** — `aw_info.context.otel_trace_id` (propagated from parent workflow via `aw_context`).
4. **Generate new** — 32-character random hex string via `randomBytes(16).toString("hex")`.

The conclusion span MUST resolve the trace ID using:

1. **Job environment** — `GITHUB_AW_OTEL_TRACE_ID` (set by this job's setup step).
2. **Parent context** — `aw_info.context.otel_trace_id` (inherited from parent).
3. **Legacy fallback** — `aw_info.context.workflow_call_id` (converted to hex).
4. **Generate new** — 32-character random hex string.

### 12.3 Trace ID Storage

After generating or resolving a trace ID, the setup step MUST:

1. **Write to `$GITHUB_OUTPUT`** so downstream jobs can access:
   - `trace-id` — 32-char hex trace ID
   - `span-id` — 16-char hex setup span ID
   - `parent-span-id` — 16-char hex global parent span ID

2. **Write to `$GITHUB_ENV`** so downstream steps in the same job can access:
   - `GITHUB_AW_OTEL_TRACE_ID` — Trace ID
   - `GITHUB_AW_OTEL_PARENT_SPAN_ID` — Setup span ID (parent for conclusion span)
   - `GITHUB_AW_OTEL_JOB_START_MS` — Epoch milliseconds when setup completed

### 12.4 Cross-Job Propagation

The compiler MUST wire setup outputs through the job dependency graph so all jobs in a run share a single trace ID. Downstream jobs receive `needs.<setup-job>.outputs.trace-id` and `needs.<setup-job>.outputs.parent-span-id` as action inputs.

### 12.5 Dispatch and Composite Action Propagation

When a workflow dispatches a child workflow or composite action, parent trace context MUST be passed via `aw_context`:

- `aw_context.otel_trace_id` → child inherits parent trace ID
- `aw_context.otel_parent_span_id` → child setup span parents under parent's setup span

This context is written to `/tmp/gh-aw/aw_info.json` and propagated through action inputs.

### 12.6 Trace ID Lookup

To find a trace in an OTLP backend:

1. Locate the OTLP trace ID from the GitHub Actions job summary or the `trace-id` output.
2. Search the backend by trace ID (32-char hex string).
3. For local debugging, query the JSONL mirror:

```bash
jq '.resourceSpans[].scopeSpans[].spans[] | {name, traceId, spanId, status}' /tmp/gh-aw/otel.jsonl
```

---

## 13. Implementation Mapping

This section maps the normative behavior in this specification to the current `gh-aw` implementation. These mappings MUST be kept in sync when behavior changes.

| Section | Title | Primary implementation files |
|---|---|---|
| §4 | Configuration Model | `pkg/workflow/frontmatter_types.go`, `pkg/parser/schemas/main_workflow_schema.json`, `pkg/workflow/observability_otlp.go` |
| §5 | Runtime Environment Contract | `pkg/workflow/observability_otlp.go`, `pkg/workflow/compiler_types.go` |
| §6.1 | Multi-Endpoint Fan-Out | `pkg/workflow/observability_otlp.go`, `actions/setup/js/send_otlp_span.cjs` |
| §6.2-§6.4 | Export and Gateway Integration | `pkg/workflow/mcp_renderer.go`, `pkg/workflow/mcp_setup_generator.go`, `pkg/workflow/schemas/mcp-gateway-config.schema.json` |
| §6.5 | Trace Context Variables | `actions/setup/js/action_setup_otlp.cjs`, `actions/setup/js/aw_context.cjs` |
| §7 | Local Mirrors and Artifacts | `actions/setup/js/send_otlp_span.cjs`, `actions/setup/js/constants.cjs`, `actions/setup/post.js` |
| §8 | Security and Privacy Requirements | `pkg/workflow/observability_otlp.go`, `pkg/workflow/mcp_renderer.go`, `pkg/workflow/mcp_setup_generator.go`, `actions/setup/js/send_otlp_span.cjs` |
| §9 | Trace Model | `actions/setup/js/send_otlp_span.cjs`, `actions/setup/js/action_setup_otlp.cjs`, `actions/setup/js/action_conclusion_otlp.cjs` |
| §10 | Span Attribute Contract | `actions/setup/js/action_setup_otlp.cjs`, `actions/setup/js/action_conclusion_otlp.cjs`, `actions/setup/js/send_otlp_span.cjs`, `actions/setup/js/evaluate_outcomes.cjs`, `actions/setup/js/emit_outcome_spans.cjs` |
| §10.7 | MCP Gateway Span Attribute Contract | `gh-aw-mcpg` tracing package, `gh-aw-mcpg/docs/otel-sentry.md` |
| §11 | Resource Attributes | `actions/setup/js/action_setup_otlp.cjs`, `actions/setup/js/send_otlp_span.cjs` |
| §12 | Trace ID Propagation | `actions/setup/js/action_setup_otlp.cjs`, `actions/setup/js/aw_context.cjs`, `pkg/workflow/compiler_yaml.go` |

When behavior changes in any mapped file, this table SHOULD be updated in the same change set.

---

## 14. Compliance Testing

A conforming implementation MUST include automated coverage for the following behaviors.

| Test ID | Requirement | Expected result | Primary current tests |
|---|---|---|---|
| `T-OTEL-OBS-001` | String endpoint form | Compiler injects `OTEL_EXPORTER_OTLP_ENDPOINT` and normalizes top-level headers. | `pkg/workflow/observability_otlp_test.go` |
| `T-OTEL-OBS-002` | Object endpoint form | Compiler accepts `{url, headers}` object form and injects primary env vars. | `pkg/workflow/observability_otlp_test.go` |
| `T-OTEL-OBS-003` | Array endpoint form | Compiler preserves first endpoint as primary and injects `GH_AW_OTLP_ENDPOINTS`. | `pkg/workflow/observability_otlp_test.go`, `pkg/workflow/observability_job_summary_test.go` |
| `T-OTEL-OBS-004` | Sentry header rewrite | `Authorization` is normalized to `x-sentry-auth` for Sentry endpoints. | `pkg/workflow/observability_otlp_test.go` |
| `T-OTEL-OBS-005` | Static allowlisting | Static endpoint hostnames are appended to network allowlist. | `pkg/workflow/observability_otlp_test.go` |
| `T-OTEL-OBS-006` | Gateway JSON contract | Gateway config includes `opentelemetry.endpoint`, `traceId`, and `spanId`, but not OTLP headers. | `pkg/workflow/mcp_renderer_test.go` |
| `T-OTEL-OBS-007` | Gateway container env contract | Gateway container receives `GITHUB_AW_OTEL_TRACE_ID`, `GITHUB_AW_OTEL_PARENT_SPAN_ID`, and `OTEL_EXPORTER_OTLP_HEADERS`. | `pkg/workflow/mcp_setup_generator_test.go` |
| `T-OTEL-OBS-008` | Local mirror persistence | Helper emission writes `/tmp/gh-aw/otel.jsonl` even when OTLP export fails or is absent. | `actions/setup/js/send_otlp_span.test.cjs` |
| `T-OTEL-OBS-009` | Trace context propagation | Setup writes valid trace and parent span IDs into runtime environment. | `actions/setup/js/action_setup_otlp.test.cjs`, `actions/setup/js/otlp.test.cjs` |
| `T-OTEL-OBS-010` | Artifact inclusion | Observability artifacts include the OTEL JSONL mirror when artifact collection is enabled. | `pkg/workflow/compiled_lock_files_test.go` |
| `T-OTEL-OBS-011` | Span naming convention | All emitted span names follow `gh-aw.<job-name>.<operation>` pattern. | `actions/setup/js/send_otlp_span.test.cjs` |
| `T-OTEL-OBS-012` | Span hierarchy | Setup spans share a global parent span ID; conclusion spans parent under the setup span. | `actions/setup/js/action_setup_otlp.test.cjs`, `actions/setup/js/action_conclusion_otlp.test.cjs` |
| `T-OTEL-OBS-013` | Span attribute contract | Setup and conclusion spans contain all required attributes from §10. | `actions/setup/js/action_setup_otlp.test.cjs`, `actions/setup/js/action_conclusion_otlp.test.cjs` |
| `T-OTEL-OBS-014` | Resource attributes | All exported spans include required resource attributes from §11. | `actions/setup/js/send_otlp_span.test.cjs` |
| `T-OTEL-OBS-015` | Trace ID resolution order | Trace ID follows the priority chain: explicit option → action input → parent context → generate new. | `actions/setup/js/action_setup_otlp.test.cjs` |
| `T-OTEL-OBS-016` | Gateway span linkage | Gateway spans (`gateway.request`, `mcp.tool_call`, `gateway.backend.execute`) appear as children of the agent job's setup span in the same trace. | `gh-aw-mcpg` integration tests |
| `T-OTEL-OBS-017` | Gateway tool name attribute | Every `mcp.tool_call` span includes `gen_ai.tool.name` identifying the tool invoked. | `gh-aw-mcpg` integration tests |
| `T-OTEL-OBS-018` | Gateway rate limit attribute | `gateway.backend.execute` spans include `rate_limit.hit` when a rate limit is triggered. | `gh-aw-mcpg` integration tests |
| `T-OTEL-OBS-019` | Proxy DIFC span emission | When API proxy is active, `proxy.difc_pipeline` and `proxy.backend.forward` spans are emitted. | `gh-aw-mcpg` integration tests |

Additional tests SHOULD be added when new helper APIs, new OTLP normalization rules, or new runtime sinks become normative.

### 14.1 Runtime Conformance Workflows

The following agentic workflows provide runtime conformance validation:

| Workflow | Purpose | Coverage |
|---|---|---|
| [`smoke-otel-backends.md`](../.github/workflows/smoke-otel-backends.md) | End-to-end OTLP smoke test | Local mirror + Sentry/Grafana/Datadog visibility |
| [`daily-otel-instrumentation-advisor.md`](../.github/workflows/daily-otel-instrumentation-advisor.md) | Daily code review + live data validation | Sentry + Grafana backend data |
| [`daily-grafana-otel-instrumentation-advisor.md`](../.github/workflows/daily-grafana-otel-instrumentation-advisor.md) | Grafana-only variant | Grafana Tempo data |
| [`otlp-data-quality-validator.md`](../.github/workflows/otlp-data-quality-validator.md) | OTLP data quality validation | JSONL + vendor traces + attribute contract |

---

## 15. References

### Normative References

- **[RFC 2119]** Key words for use in RFCs to Indicate Requirement Levels
- **[OpenTelemetry]** OpenTelemetry specification and semantic conventions
- **[OTLP]** OpenTelemetry Protocol specification

### Informative References

- [docs/src/content/docs/reference/open-telemetry.md](../docs/src/content/docs/reference/open-telemetry.md)
- [docs/src/content/docs/reference/mcp-gateway.md](../docs/src/content/docs/reference/mcp-gateway.md)
- [specs/aw-harness.md](./aw-harness.md)
- [specs/safe-output-outcome-evaluation.md](./safe-output-outcome-evaluation.md)

### Runtime Conformance Workflows

- [.github/workflows/smoke-otel-backends.md](../.github/workflows/smoke-otel-backends.md) — End-to-end OTLP smoke test
- [.github/workflows/daily-otel-instrumentation-advisor.md](../.github/workflows/daily-otel-instrumentation-advisor.md) — Daily code review + live data validation
- [.github/workflows/daily-grafana-otel-instrumentation-advisor.md](../.github/workflows/daily-grafana-otel-instrumentation-advisor.md) — Grafana-only variant
- [.github/workflows/otlp-data-quality-validator.md](../.github/workflows/otlp-data-quality-validator.md) — OTLP data quality validation

---

## 16. Change Log

### Version 0.3.0 (Working Draft)

- Extended §9.3 span hierarchy to include MCP gateway and API proxy spans
- Extended §9.4 span kinds table with gateway span types (`gateway.request`, `mcp.tool_call`, `gateway.backend.execute`, `proxy.difc_pipeline`, `proxy.backend.forward`)
- Added §10.7 MCP Gateway Span Attribute Contract: attribute tables for `gateway.request`, `mcp.tool_call`, `gateway.backend.execute`, `proxy.difc_pipeline`, `proxy.backend.forward`
- Added compliance tests T-OTEL-OBS-016 through T-OTEL-OBS-019 for gateway span linkage, tool names, rate limits, and proxy spans
- Source: [gh-aw-mcpg OTEL Sentry docs](https://github.com/github/gh-aw-mcpg/blob/main/docs/otel-sentry.md)

### Version 0.2.0 (Working Draft)

- Added §9 Trace Model: span naming, hierarchy, kinds, status, exception events
- Added §10 Span Attribute Contract: required and conditional attributes for setup, conclusion, and agent spans
- Added §10.4 Outcome Evaluation Span Attributes: reactions, zero-touch, comments
- Added §10.5 Outcome Summary Span Attributes: zero-touch rate, median resolution, economics metrics
- Added §11 Resource Attributes: required and conditional resource attributes, instrumentation scope
- Added §12 Trace ID Propagation and Lookup: resolution order, storage, cross-job and dispatch propagation
- Added §14.1 Runtime Conformance Workflows
- Added compliance tests T-OTEL-OBS-011 through T-OTEL-OBS-015
- Updated implementation mapping table with §9–§12 entries
- Renumbered §9–§12 to §13–§16

### Version 0.1.0 (Working Draft)

- Initial repository-level OTel observability specification
- Defined the normative `observability.otlp` contract for compiler and runtime behavior
- Added gateway-integration, local-mirror, implementation-mapping, and conformance-test sections
