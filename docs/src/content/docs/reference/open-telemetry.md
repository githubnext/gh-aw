---
title: OpenTelemetry
description: Reference for OpenTelemetry observability in GitHub Agentic Workflows, including OTLP configuration, runtime variables, span attributes, and trace artifacts.
sidebar:
  order: 205
---

GitHub Agentic Workflows can export distributed traces to
OpenTelemetry Protocol (OTLP) compatible backends.

Use this page as the canonical reference for observability
configuration and runtime behavior.

## Configure `observability.otlp`

Set `observability.otlp` in workflow frontmatter:

```yaml wrap
observability:
  otlp:
    endpoint: ${{ secrets.OTLP_ENDPOINT }}
    headers:
      Authorization: ${{ secrets.OTLP_TOKEN }}
      X-Tenant: my-org
```

### Fields

| Field | Type | Description |
| --- | --- | --- |
| `observability.otlp.endpoint` | string, object, or array | OTLP/HTTP collector endpoint URL. Accepts a plain URL string, a single `{url, headers}` object, or an array of `{url, headers}` objects for concurrent fan-out to multiple collectors. When a static URL is provided, its hostname is automatically added to the network firewall allowlist. |
| `observability.otlp.headers` | map or string | HTTP headers sent with every OTLP export request. Only applies when `endpoint` is a plain string; object and array endpoint entries carry their own per-endpoint headers. |
| `observability.otlp.if-missing` | string (`error`, `warn`, `ignore`) | Controls behavior when OTLP endpoint/header values resolve to empty values at runtime. `error` (default) fails startup. `warn` logs a warning and skips MCP gateway OTLP configuration. `ignore` skips MCP gateway OTLP configuration without warning. This setting affects MCP gateway setup only. |

### Endpoint forms

The `endpoint` field accepts three forms.

String form (backward-compatible):

```yaml wrap
observability:
  otlp:
    endpoint: ${{ secrets.OTLP_ENDPOINT }}
    headers:
      Authorization: ${{ secrets.OTLP_TOKEN }}
```

Object form (single endpoint with per-endpoint headers):

```yaml wrap
observability:
  otlp:
    endpoint:
      url: ${{ secrets.OTLP_ENDPOINT }}
      headers:
        Authorization: ${{ secrets.OTLP_TOKEN }}
        X-Tenant: acme
```

Array form (concurrent fan-out to multiple endpoints):

```yaml wrap
observability:
  otlp:
    endpoint:
      - url: ${{ secrets.OTLP_ENDPOINT_PRIMARY }}
        headers:
          Authorization: ${{ secrets.OTLP_TOKEN_PRIMARY }}
      - url: ${{ secrets.OTLP_ENDPOINT_BACKUP }}
        headers:
          Authorization: ${{ secrets.OTLP_TOKEN_BACKUP }}
```

If one endpoint fails in array mode, export still continues
for the remaining endpoints.

### Header forms

The `headers` field applies to the string endpoint form and
accepts either a map or a comma-separated string.

Map form:

```yaml wrap
observability:
  otlp:
    endpoint: ${{ secrets.OTLP_ENDPOINT }}
    headers:
      Authorization: ${{ secrets.OTLP_TOKEN }}
      X-Tenant: acme
```

String form:

```yaml wrap
observability:
  otlp:
    endpoint: ${{ secrets.OTLP_ENDPOINT }}
    headers: "Authorization=${{ secrets.OTLP_TOKEN }},X-Tenant=acme"
```

## Runtime environment variables

When `observability.otlp` is configured, gh-aw injects:

| Variable | Description |
| --- | --- |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP collector URL for the first configured endpoint (backward compatibility). |
| `OTEL_EXPORTER_OTLP_HEADERS` | Comma-separated `key=value` headers for the first endpoint (when headers are configured). |
| `OTEL_SERVICE_NAME` | Always `gh-aw`. |
| `GH_AW_OTLP_ENDPOINTS` | JSON array of all endpoint entries, used by gh-aw JavaScript span exporters for fan-out. |
| `GH_AW_OTLP_IF_MISSING` | Set to `warn` or `ignore` when `observability.otlp.if-missing` is configured. |
| `COPILOT_OTEL_FILE_EXPORTER_PATH` | Path for Copilot CLI span output (`/tmp/gh-aw/copilot-otel.jsonl`). |

> [!NOTE]
> `GH_AW_OTLP_ENDPOINTS` is the primary variable for
> gh-aw JavaScript span exporters.
> `OTEL_EXPORTER_OTLP_ENDPOINT` is retained for backward
> compatibility.

## Agent span attributes

The agent span (`gh-aw.agent.agent`) uses OpenTelemetry
[GenAI semantic conventions](https://opentelemetry.io/docs/specs/semconv/gen-ai/)
and is emitted as `SPAN_KIND_CLIENT`.

| Attribute | Description |
| --- | --- |
| `gen_ai.request.model` | Model name used for inference |
| `gen_ai.operation.name` | Always `"chat"` |
| `gen_ai.system` | Standardized OTel system name (for example, `github_models`, `anthropic`, `openai`, `google_vertex_ai`) |
| `gh-aw.engine` | Raw gh-aw engine identifier (for example, `copilot`, `claude`, `codex`, `gemini`) |
| `gen_ai.workflow.name` | Workflow name |
| `gen_ai.usage.input_tokens` | Total input tokens consumed |
| `gen_ai.usage.output_tokens` | Total output tokens produced |
| `gen_ai.usage.cache_read.input_tokens` | Cache-read tokens reused |
| `gen_ai.usage.cache_creation.input_tokens` | Cache-creation tokens written |
| `gen_ai.response.finish_reasons` | Array containing the agent stop reason |

> [!NOTE]
> Prior to v0.70, the agent span used private `gh-aw.*`
> attributes and `SPAN_KIND_INTERNAL`.
> Update dashboards and alerts that still use old names.

> [!NOTE]
> Prior to v0.76, the engine was emitted as
> `gen_ai.provider.name` with raw gh-aw IDs.
> It is now emitted as `gen_ai.system`, while raw IDs are
> preserved in `gh-aw.engine`.

## Trace files and artifacts

When observability is enabled, trace data is also mirrored
to local JSONL files and uploaded in the `agent` artifact:

- `otel.jsonl` for spans emitted by gh-aw JavaScript helpers
- `copilot-otel.jsonl` for spans emitted by Copilot CLI

See [Artifacts](/gh-aw/reference/artifacts/) for artifact
download details.

## Custom spans from shared imports

To emit custom spans from imported shared workflows, use:
[Telemetry](/gh-aw/guides/telemetry/).

## Related documentation

- [Frontmatter](/gh-aw/reference/frontmatter/)
- [Network](/gh-aw/reference/network/)
- [Artifacts](/gh-aw/reference/artifacts/)
- [Audit](/gh-aw/reference/audit/)
