---
title: Conclusion Data Specification
description: Data model for the compact conclusion-job activity summary in GitHub Agentic Workflows
sidebar:
  order: 1356
---

# Conclusion Data Specification

**Version**: 1.0.0

**Status**: Draft

**Latest Version**: [conclusion-data-specification](/gh-aw/specs/conclusion-data-specification/)

**Editor**: GitHub Agentic Workflows Team

## Abstract

This specification defines the JSON data file produced by the conclusion job at `usage/activity/summary.json`. The file provides compact workflow-run activity data to consumers without requiring the full agent artifact.

## Status of This Document

This document is a Draft specification maintained by the GitHub Agentic Workflows project. Feedback should be filed as a GitHub issue against the `github/gh-aw` repository.

## 1. Conformance

The key words "MUST", "MUST NOT", "REQUIRED", "SHOULD", "SHOULD NOT", and "MAY" are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

A conforming producer creates the file according to this specification. A conforming consumer accepts all required fields, ignores unknown additive fields, and handles omitted optional sections.

## 2. File contract

The file:

- MUST be UTF-8 encoded JSON;
- MUST contain exactly one top-level JSON object;
- MUST use `usage-activity-summary/v1` as its `schema` value;
- SHOULD be stored at `usage/activity/summary.json` in the `usage` artifact.

The top-level object has this shape:

| Field | Type | Arity | Description |
|---|---|---:|---|
| `schema` | string | 1 | Schema identifier. |
| `firewall` | object | 0..1 | Firewall request summary. |
| `session` | object | 0..1 | Agent session event summary. |
| `gateway` | object | 0..1 | MCP gateway activity summary. |
| `integrity` | object | 0..1 | Integrity-filter activity summary. |
| `safe_outputs` | object | 0..1 | Safe-output item summary. |
| `experiments` | object | 0..1 | Experiment assignments. |
| `working_set` | object | 1 | Working-set rebuild metrics. |

Optional sections MAY be omitted when their source data is unavailable.

## 3. Firewall data

The `firewall` object contains aggregate counters and domain records:

| Field | Type | Arity | Description |
|---|---|---:|---|
| `total_requests` | non-negative integer | 1 | Count of classified requests. |
| `allowed_requests` | non-negative integer | 1 | Count of accepted requests. |
| `blocked_requests` | non-negative integer | 1 | Count of rejected requests. |
| `allowed_domains` | array of strings | 1 | Unique domains with at least one accepted request. |
| `blocked_domains` | array of strings | 1 | Unique domains with at least one rejected request. |
| `requests_by_domain` | object | 1 | Compatibility map from domain to `allowed` and `blocked` counts. |
| `domains` | array of domain records | 1 | Normalized per-domain acceptance records. |

The producer MUST omit the `firewall` object when no valid firewall requests are available.

### 3.1 Domain record

Each `domains` entry has this shape:

| Field | Type | Arity | Description |
|---|---|---:|---|
| `domain` | string | 1 | Observed destination domain, including its port when present in the firewall log. |
| `accepted` | boolean | 1 | `true` for accepted requests and `false` for blocked requests. |
| `arity` | positive integer | 1 | Number of requests for this domain and acceptance state. |

A domain with both accepted and blocked requests MUST produce two records. A producer MUST NOT emit a record with an `arity` of zero. Records MUST be ordered lexicographically by `domain`, with the accepted record before the blocked record for the same domain.

Placeholder domains such as `-`, `-:-`, and values beginning with `error:` MUST NOT appear in domain records. When the domain field is unavailable but the destination address is valid, the producer MUST use the destination address as `domain`.

### 3.2 Consistency requirements

For a conforming `firewall` object:

- the sum of all `domains[].arity` values MUST equal `total_requests`;
- the sum of `arity` where `accepted` is `true` MUST equal `allowed_requests`;
- the sum of `arity` where `accepted` is `false` MUST equal `blocked_requests`;
- each domain in `allowed_domains` MUST have an accepted domain record;
- each domain in `blocked_domains` MUST have a blocked domain record;
- `total_requests` MUST equal `allowed_requests + blocked_requests`.

## 4. Example

```json
{
  "schema": "usage-activity-summary/v1",
  "firewall": {
    "total_requests": 4,
    "allowed_requests": 3,
    "blocked_requests": 1,
    "allowed_domains": ["api.github.com:443", "mixed.example.com:443"],
    "blocked_domains": ["mixed.example.com:443"],
    "requests_by_domain": {
      "api.github.com:443": { "allowed": 2, "blocked": 0 },
      "mixed.example.com:443": { "allowed": 1, "blocked": 1 }
    },
    "domains": [
      { "domain": "api.github.com:443", "accepted": true, "arity": 2 },
      { "domain": "mixed.example.com:443", "accepted": true, "arity": 1 },
      { "domain": "mixed.example.com:443", "accepted": false, "arity": 1 }
    ]
  }
}
```

## 5. Compatibility

The `domains` field is additive to `usage-activity-summary/v1`. Consumers MUST ignore unknown fields. Producers MUST preserve the existing aggregate counters, domain arrays, and `requests_by_domain` compatibility map while emitting `domains`.

Future incompatible changes require a new schema identifier.

## 6. Other activity sections

### 6.1 Session

The optional `session` object contains non-negative integer counters:

| Field | Arity |
|---|---:|
| `total_events` | 1 |
| `session_starts` | 1 |
| `session_shutdowns` | 1 |
| `turns` | 1 |
| `assistant_messages` | 1 |
| `reasoning_events` | 1 |
| `tool_execution_starts` | 1 |
| `tool_execution_completes` | 1 |
| `failed_tool_executions` | 1 |

### 6.2 Gateway

The optional `gateway` object contains these non-negative numeric fields:

| Field | Type | Arity |
|---|---|---:|
| `total_calls` | integer | 1 |
| `failed_calls` | integer | 1 |
| `total_input_size` | integer | 1 |
| `total_output_size` | integer | 1 |
| `max_input_size` | integer | 1 |
| `max_output_size` | integer | 1 |
| `total_duration_ms` | number | 1 |
| `max_duration_ms` | number | 1 |
| `servers` | array of server records | 1 |
| `tools` | array of tool records | 1 |

A server record contains `server_name`, `request_count`, `tool_call_count`, `failed_calls`, `total_input_size`, `total_output_size`, `total_duration_ms`, and `avg_duration_ms`. A tool record contains `server_name`, `tool_name`, `call_count`, `failed_calls`, `total_input_size`, `total_output_size`, `max_input_size`, `max_output_size`, `total_duration_ms`, `avg_duration_ms`, and `max_duration_ms`. Names are strings; all other fields are non-negative numbers.

### 6.3 Integrity

The optional `integrity` object contains:

| Field | Type | Arity |
|---|---|---:|
| `total_filtered` | non-negative integer | 1 |
| `filtered_server_counts` | string-to-integer object | 1 |
| `filtered_tool_counts` | string-to-integer object | 1 |
| `filtered_reason_counts` | string-to-integer object | 1 |

### 6.4 Safe outputs

The optional `safe_outputs` object contains `total_items`, a non-negative integer, and `items_by_type`, an object mapping each safe-output type to a positive integer count. A present but empty manifest MUST produce `total_items: 0` and an empty `items_by_type` object.

### 6.5 Experiments

The optional `experiments` object contains one required `assignments` object. Each property maps an experiment name to its selected string variant.

### 6.6 Working set

The required `working_set` object contains:

| Field | Type | Arity |
|---|---|---:|
| `measurement_state` | string enum | 1 |
| `rebuild_factor` | non-negative number | 0..1 |
| `cumulative_input_tokens` | non-negative number | 1 |
| `peak_input_tokens` | non-negative number | 1 |
| `rebuild_excess_tokens` | non-negative number | 1 |
| `invocations` | non-negative integer | 1 |

`measurement_state` MUST be `measured`, `partial`, or `unavailable`. `rebuild_factor` MUST be present for measured or partial data and MUST be omitted when unavailable.

## 7. Security and privacy

The producer MUST NOT include request paths, query strings, credentials, headers, or payloads in firewall domain records. Consumers SHOULD treat domain values as potentially sensitive workflow metadata and avoid exposing them outside the artifact's existing access boundary.

## 8. Compliance tests

| ID | Requirement |
|---|---|
| T-CD-001 | A valid allowed request produces a record with `accepted: true` and `arity: 1`. |
| T-CD-002 | A valid blocked request produces a record with `accepted: false` and `arity: 1`. |
| T-CD-003 | Repeated requests with the same domain and acceptance state are represented by one record whose `arity` equals the request count. |
| T-CD-004 | A domain observed in both states produces two records in deterministic order. |
| T-CD-005 | Placeholder and internal diagnostic entries do not produce domain records. |
| T-CD-006 | Domain-record arities reconcile with the aggregate firewall counters. |

## 9. References

- [Artifacts reference](/gh-aw/reference/artifacts/)
- [OpenTelemetry Observability Specification](https://github.com/github/gh-aw/blob/main/specs/otel-observability-spec.md)
- [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119)

## 10. Change log

### 1.0.0

- Defined the conclusion data file and its top-level sections.
- Defined normalized firewall records containing domain, acceptance state, and arity.
