---
title: OTel Observability Specification
version: 0.1.0
status: Working Draft
date: 2026-05-19
last_updated: 2026-05-19
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
9. [Implementation Mapping](#9-implementation-mapping)
10. [Compliance Testing](#10-compliance-testing)
11. [References](#11-references)
12. [Change Log](#12-change-log)

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

An implementation conforms to this specification if it satisfies all MUST and MUST NOT requirements in Sections 4 through 10.

The key words **MUST**, **MUST NOT**, **SHOULD**, **SHOULD NOT**, and **MAY** are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### 2.1 Conformance Classes

This specification defines three conformance levels:

| Level | Requirements |
|---|---|
| **Level 1 - Config** | Correct parsing and normalization of `observability.otlp` and workflow environment injection as defined in Sections 4 and 5. |
| **Level 2 - Runtime** | Level 1 plus MCP gateway integration and degraded-mode export behavior from Section 6. |
| **Level 3 - Complete** | Level 2 plus local mirror, artifact, implementation-mapping, and compliance obligations in Sections 7 through 10. |

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

## 9. Implementation Mapping

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

When behavior changes in any mapped file, this table SHOULD be updated in the same change set.

---

## 10. Compliance Testing

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

Additional tests SHOULD be added when new helper APIs, new OTLP normalization rules, or new runtime sinks become normative.

---

## 11. References

### Normative References

- **[RFC 2119]** Key words for use in RFCs to Indicate Requirement Levels
- **[OpenTelemetry]** OpenTelemetry specification and semantic conventions
- **[OTLP]** OpenTelemetry Protocol specification

### Informative References

- [docs/src/content/docs/reference/open-telemetry.md](../docs/src/content/docs/reference/open-telemetry.md)
- [docs/src/content/docs/reference/mcp-gateway.md](../docs/src/content/docs/reference/mcp-gateway.md)
- [specs/aw-harness.md](./aw-harness.md)
- [specs/safe-output-outcome-evaluation.md](./safe-output-outcome-evaluation.md)

---

## 12. Change Log

### Version 0.1.0 (Working Draft)

- Initial repository-level OTel observability specification
- Defined the normative `observability.otlp` contract for compiler and runtime behavior
- Added gateway-integration, local-mirror, implementation-mapping, and conformance-test sections