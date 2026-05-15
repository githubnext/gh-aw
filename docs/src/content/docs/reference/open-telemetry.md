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
| `OTEL_EXPORTER_OTLP_HEADERS` | Comma-separated `key=value` headers for the first endpoint (when headers are configured). |
| `OTEL_SERVICE_NAME` | Always `gh-aw`. |
| `GH_AW_OTLP_ENDPOINTS` | JSON array of all endpoint entries, used by gh-aw JavaScript span exporters for fan-out. |
| `GH_AW_OTLP_IF_MISSING` | Set to `warn` or `ignore` when `observability.otlp.if-missing` is configured. |
| `COPILOT_OTEL_FILE_EXPORTER_PATH` | Path for Copilot CLI span output (`/tmp/gh-aw/copilot-otel.jsonl`). |

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

## Trace files and artifacts

When observability is enabled, trace data is also mirrored
to local JSONL files and uploaded in the `agent` artifact:

- `otel.jsonl` for spans emitted by gh-aw JavaScript helpers
- `copilot-otel.jsonl` for spans emitted by Copilot CLI

See [Artifacts](/gh-aw/reference/artifacts/) for artifact
download details.

## Custom spans from shared imports

Shared agentic workflow imports can emit their own OTLP spans alongside the built-in gh-aw telemetry. This lets third-party tools — APM agents, data pipeline steps, custom scanners — attach their own measurements to the same distributed trace that gh-aw creates for each workflow run.

### Quick start

The `otlp.cjs` helper provides a minimal, stable API. Use it in any `steps:` entry of a shared import:

```yaml wrap title=".github/workflows/shared/my-tool.md"
---
# My Tool — shared import that instruments its own telemetry

steps:
  - name: My Tool — do work and record telemetry
    id: my-tool-run
    uses: actions/github-script@v8
    with:
      script: |
        const otlp = require('/tmp/gh-aw/actions/otlp.cjs');

        const startMs = Date.now();
        // ── do your tool's work here ──────────────────────────────────────
        // const result = await myTool.run();
        // ─────────────────────────────────────────────────────────────────
        const endMs = Date.now();

        await otlp.logSpan('my-tool', {
          'my-tool.version':         '1.2.3',
          'my-tool.items_processed': 42,
          'my-tool.result':          'success',
        }, { startMs, endMs });
---

My tool has run and its telemetry span will appear in the same distributed trace as the workflow run.
```

Import the shared file in any workflow alongside the OTLP configuration:

```yaml wrap title=".github/workflows/my-workflow.md"
---
on:
  schedule: daily
engine: copilot
imports:
  - shared/observability-otlp.md   # sets the OTLP endpoint + auth headers
  - shared/my-tool.md              # runs my-tool and records its span
---

# Daily Report

Run the daily report using my-tool results.
```

### `logSpan` API

```javascript
const otlp = require('/tmp/gh-aw/actions/otlp.cjs');

await otlp.logSpan(toolName, attributes, options);
```

| Parameter | Type | Description |
|-----------|------|-------------|
| `toolName` | `string` | Logical name for the tool (e.g. `"my-scanner"`). Used as `service.name` and as the span name prefix `<toolName>.run`. |
| `attributes` | `Record<string, string \| number \| boolean>` | Domain-specific attributes emitted on the span. All env plumbing is handled automatically. |
| `options.startMs` | `number` | Span start time (ms since epoch). Defaults to `Date.now()`. |
| `options.endMs` | `number` | Span end time (ms since epoch). Defaults to `Date.now()`. |
| `options.isError` | `boolean` | When `true`, sets the span status to `ERROR`. |
| `options.errorMessage` | `string` | Human-readable status message included when `isError` is `true`. |
| `options.traceId` | `string` | Override trace ID. Defaults to `GITHUB_AW_OTEL_TRACE_ID`. |
| `options.parentSpanId` | `string` | Override parent span ID. Defaults to `GITHUB_AW_OTEL_PARENT_SPAN_ID`. |
| `options.endpoint` | `string` | Override OTLP endpoint. Defaults to `OTEL_EXPORTER_OTLP_ENDPOINT`. |

`logSpan` is non-fatal and never throws. Export failures are surfaced as `console.warn`. When `GITHUB_AW_OTEL_TRACE_ID` is missing or invalid, the call returns silently — no warning, no side-effects.

#### Recording an error span

```javascript
await otlp.logSpan('my-scanner', {
  'my-scanner.items_scanned': 100,
}, { isError: true, errorMessage: 'database connection timed out' });
```

### Attribute naming recommendations

- Use `your-tool.` as a prefix for tool-specific attributes (e.g. `my-tool.items_processed`).
- Use [OpenTelemetry semantic conventions](https://opentelemetry.io/docs/specs/semconv/) for cross-cutting concerns (e.g. `db.system`, `http.response.status_code`).
- Avoid attribute names containing `token`, `secret`, `password`, `key`, or `auth` — the helpers automatically redact matching attribute values before sending.

### Security

Attribute values are sanitized automatically before the payload is exported or mirrored:

- **Redacts** the value of any attribute whose key matches `token`, `secret`, `password`, `passwd`, `key`, `auth`, `credential`, `api-key`, or `access-key` (case-insensitive), replacing it with `[REDACTED]`.
- **Truncates** string values longer than 1,024 characters.

Sanitization is applied to both the over-the-wire OTLP export and the local JSONL debug mirror, so you do not need to call it yourself.

### Debugging without a live collector

Every span emitted by `logSpan` is always appended as a sanitized JSON line to `/tmp/gh-aw/otel.jsonl`, even when `OTEL_EXPORTER_OTLP_ENDPOINT` is not set. When OTLP is configured, Copilot CLI's own spans are written to `/tmp/gh-aw/copilot-otel.jsonl` and automatically forwarded to configured endpoints at the end of the run. Both files are included in the `agent` artifact when OTLP is enabled, so you can inspect spans after the run:

```bash
# Download agent artifacts for a run
gh aw logs <run-id> --artifacts agent

# Inspect spans emitted by your tool
cat otel.jsonl | jq 'select(.resourceSpans[].scopeSpans[].spans[].name | startswith("my-tool"))'

# Inspect Copilot CLI spans
cat copilot-otel.jsonl | jq '.resourceSpans'
```

### Advanced: low-level API

For full control — multiple linked spans, custom resource attributes, or span events — use the underlying helpers from `send_otlp_span.cjs` directly. The key environment variables set by the `actions/setup` step are:

| Variable | Description |
|----------|-------------|
| `GITHUB_AW_OTEL_TRACE_ID` | 32-char hex trace ID shared by all spans in this run. |
| `GITHUB_AW_OTEL_PARENT_SPAN_ID` | 16-char hex span ID of the job setup span; use as `parentSpanId` to nest spans under it. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP collector base URL. |
| `OTEL_EXPORTER_OTLP_HEADERS` | Comma-separated `key=value` authentication headers. |

```javascript
const {
  buildAttr, buildOTLPPayload, sendOTLPSpan,
  generateSpanId, SPAN_KIND_CLIENT,
} = require('/tmp/gh-aw/actions/send_otlp_span.cjs');

const traceId      = process.env.GITHUB_AW_OTEL_TRACE_ID;
const parentSpanId = process.env.GITHUB_AW_OTEL_PARENT_SPAN_ID;
const endpoint     = process.env.OTEL_EXPORTER_OTLP_ENDPOINT;

const setupSpanId = generateSpanId();
const querySpanId = generateSpanId();

// Parent span for the overall operation
await sendOTLPSpan(endpoint, buildOTLPPayload({
  traceId, spanId: setupSpanId, parentSpanId,
  spanName: 'my-tool.setup', startMs: t0, endMs: t1,
  serviceName: 'my-tool', kind: SPAN_KIND_CLIENT,
  attributes: [buildAttr('my-tool.phase', 'setup')],
  resourceAttributes: [buildAttr('my-tool.version', '1.2.3')],
}));

// Child span nested under the parent span above
await sendOTLPSpan(endpoint, buildOTLPPayload({
  traceId, spanId: querySpanId, parentSpanId: setupSpanId,
  spanName: 'my-tool.query', startMs: t1, endMs: t2,
  serviceName: 'my-tool', kind: SPAN_KIND_CLIENT,
  attributes: [buildAttr('my-tool.query.rows', 1234)],
}));
```

## Related documentation

- [Frontmatter](/gh-aw/reference/frontmatter/)
- [Network](/gh-aw/reference/network/)
- [Artifacts](/gh-aw/reference/artifacts/)
- [Audit](/gh-aw/reference/audit/)
- [Imports](/gh-aw/reference/imports/) — how shared workflow imports work
