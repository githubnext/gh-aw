---
title: Emitting Custom OTLP Attributes
description: How to add custom OpenTelemetry spans and attributes from shared agentic workflows so third-party tools can upload their own telemetry data alongside built-in instrumentation.
sidebar:
  order: 19
---

Shared agentic workflow imports can emit their own OTLP spans alongside the built-in gh-aw telemetry. This lets third-party tools — APM agents, data pipeline steps, custom scanners — attach their own measurements to the same distributed trace that gh-aw creates for each workflow run.

## How the helpers become available

When a workflow runs, the `actions/setup` action copies all JavaScript helpers to `/tmp/gh-aw/actions/` before the agent job begins. Every `steps:` step in a workflow or shared import can `require()` these helpers directly, with no additional installation.

```javascript
const {
  buildAttr,
  buildOTLPPayload,
  sendOTLPSpan,
  generateSpanId,
  toNanoString,
  SPAN_KIND_INTERNAL,
} = require('/tmp/gh-aw/actions/send_otlp_span.cjs');
```

The helpers use only Node.js built-ins and native `fetch`, so they work in any GitHub Actions runner environment.

## Trace context environment variables

After the `actions/setup` step runs, it writes two environment variables to `$GITHUB_ENV` that are available to every subsequent step in the same job:

| Variable | Description |
|----------|-------------|
| `GITHUB_AW_OTEL_TRACE_ID` | The 32-char hex trace ID shared by all spans in this workflow run. Use this as the `traceId` for any span you emit so it appears in the same trace tree. |
| `GITHUB_AW_OTEL_PARENT_SPAN_ID` | The 16-char hex span ID of the `gh-aw.<jobName>.setup` span. Use this as `parentSpanId` to nest your span directly under the job setup span. |

The OTLP endpoint and headers are already resolved and exported as:

| Variable | Description |
|----------|-------------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP collector base URL (e.g. `https://traces.example.com:4318`). Empty when no endpoint is configured. |
| `OTEL_EXPORTER_OTLP_HEADERS` | Comma-separated `key=value` authentication headers. |

## Core helper API

| Function | Description |
|----------|-------------|
| `buildAttr(key, value)` | Returns a single OTLP key-value attribute. Handles `string`, `number`, and `boolean` types. |
| `buildOTLPPayload(opts)` | Assembles a complete OTLP/HTTP JSON traces payload for one span. |
| `sendOTLPSpan(endpoint, payload)` | POSTs the payload to `{endpoint}/v1/traces`. Non-fatal: failures are logged as warnings and never thrown. Writes a local JSONL mirror at `/tmp/gh-aw/otel.jsonl` regardless of whether a collector is configured. |
| `generateSpanId()` | Returns a random 16-char hex span ID. |
| `toNanoString(ms)` | Converts a millisecond timestamp to the nanosecond string format required by OTLP. |
| `SPAN_KIND_INTERNAL` | Span kind constant for an in-process operation (value: `1`). |
| `SPAN_KIND_CLIENT` | Span kind constant for an outbound call (value: `3`). |

## Emitting a custom span from `steps:`

Add a `steps:` entry to your shared import's frontmatter. Use `actions/github-script@v8` to run JavaScript and call the helpers:

```yaml wrap title=".github/workflows/shared/my-tool.md"
---
# My Tool — shared import that instruments its own telemetry

steps:
  - name: My Tool — emit OTLP span
    id: my-tool-otlp
    uses: actions/github-script@v8
    env:
      OTEL_EXPORTER_OTLP_ENDPOINT: ${{ env.OTEL_EXPORTER_OTLP_ENDPOINT }}
      GITHUB_AW_OTEL_TRACE_ID: ${{ env.GITHUB_AW_OTEL_TRACE_ID }}
      GITHUB_AW_OTEL_PARENT_SPAN_ID: ${{ env.GITHUB_AW_OTEL_PARENT_SPAN_ID }}
    with:
      script: |
        const {
          buildAttr,
          buildOTLPPayload,
          sendOTLPSpan,
          generateSpanId,
          SPAN_KIND_CLIENT,
        } = require('/tmp/gh-aw/actions/send_otlp_span.cjs');

        const startMs = Date.now();

        // ── run your tool's work here ─────────────────────────────────────
        // const result = await myTool.run();
        // ─────────────────────────────────────────────────────────────────

        const endMs = Date.now();

        const endpoint = process.env.OTEL_EXPORTER_OTLP_ENDPOINT;
        const traceId  = process.env.GITHUB_AW_OTEL_TRACE_ID;
        const parentSpanId = process.env.GITHUB_AW_OTEL_PARENT_SPAN_ID;

        if (!traceId) {
          core.warning('GITHUB_AW_OTEL_TRACE_ID is not set; skipping OTLP export');
          return;
        }

        const payload = buildOTLPPayload({
          traceId,
          spanId: generateSpanId(),
          parentSpanId,                         // nests under the job setup span
          spanName: 'my-tool.run',
          startMs,
          endMs,
          serviceName: 'my-tool',
          kind: SPAN_KIND_CLIENT,
          attributes: [
            buildAttr('my-tool.version', '1.2.3'),
            buildAttr('my-tool.items_processed', 42),
            buildAttr('my-tool.result', 'success'),
          ],
        });

        await sendOTLPSpan(endpoint, payload);
---

My tool has been configured and its telemetry span will appear in the same trace as the workflow run.
```

Import it in any workflow:

```yaml wrap title=".github/workflows/my-workflow.md"
---
on:
  schedule: daily
engine: copilot
imports:
  - shared/observability-otlp.md   # sets OTLP endpoint + headers
  - shared/my-tool.md              # installs my-tool and emits its span
---

# Daily Report

Run the daily report using my-tool results.
```

## Adding resource attributes

Pass `resourceAttributes` to `buildOTLPPayload` to add attributes at the resource (process) level. These are indexed separately from span attributes and appear under the resource in most OTLP backends:

```javascript
const payload = buildOTLPPayload({
  traceId,
  spanId: generateSpanId(),
  parentSpanId,
  spanName: 'my-tool.run',
  startMs,
  endMs,
  serviceName: 'my-tool',
  attributes: [
    buildAttr('my-tool.items_processed', 42),
  ],
  resourceAttributes: [
    buildAttr('my-tool.version', '1.2.3'),
    buildAttr('deployment.environment', 'production'),
  ],
});
```

## Emitting multiple spans

Call `sendOTLPSpan` multiple times — once per logical operation. Assign a unique span ID to each call. Link related spans under the same parent to build a trace tree:

```javascript
const setupSpanId = generateSpanId();
const querySpanId = generateSpanId();

// Parent span: overall operation
await sendOTLPSpan(endpoint, buildOTLPPayload({
  traceId, spanId: setupSpanId, parentSpanId,
  spanName: 'my-tool.setup', startMs: t0, endMs: t1,
  serviceName: 'my-tool',
  attributes: [buildAttr('my-tool.phase', 'setup')],
}));

// Child span: sub-operation nested under the parent span above
await sendOTLPSpan(endpoint, buildOTLPPayload({
  traceId, spanId: querySpanId, parentSpanId: setupSpanId,
  spanName: 'my-tool.query', startMs: t1, endMs: t2,
  serviceName: 'my-tool',
  attributes: [buildAttr('my-tool.query.rows', 1234)],
}));
```

## Attribute naming recommendations

Follow existing gh-aw conventions to ensure your attributes are easy to find and filter in dashboards:

- Use `your-tool.` as a prefix for tool-specific attributes (e.g. `my-tool.items_processed`).
- Use [OpenTelemetry semantic conventions](https://opentelemetry.io/docs/specs/semconv/) for cross-cutting concerns (e.g. `db.system`, `http.response.status_code`, `gen_ai.usage.input_tokens`).
- Avoid attribute names containing `token`, `secret`, `password`, `key`, or `auth` — the helpers automatically redact the values of matching attributes before sending (see [Security](#security)).

## Security

All attribute values pass through `sanitizeOTLPPayload` before the payload is sent over the wire. This function:

- **Redacts** the string value of any attribute whose key matches `token`, `secret`, `password`, `passwd`, `key`, `auth`, `credential`, `api-key`, or `access-key` (case-insensitive), replacing it with `[REDACTED]`.
- **Truncates** string values longer than 1,024 characters.

Sanitization is automatic — you do not need to call it yourself. `sendOTLPSpan` applies it internally before every HTTP request.

## Debugging without a live collector

`sendOTLPSpan` always appends every payload as a JSON line to `/tmp/gh-aw/otel.jsonl`, even when `OTEL_EXPORTER_OTLP_ENDPOINT` is not set. This file is included in the `firewall-audit-logs` artifact so you can inspect spans locally after the run:

```bash
# Download logs for a run
gh aw logs <run-id> --artifacts firewall

# Inspect spans emitted by your tool
cat otel.jsonl | jq 'select(.resourceSpans[].scopeSpans[].spans[].name | startswith("my-tool"))'
```

## Related documentation

- [Observability (`observability:`)](/gh-aw/reference/frontmatter/#observability-observability) — configure the OTLP endpoint and headers
- [Imports](/gh-aw/reference/imports/) — how shared workflow imports work
- [Deterministic Agentic Patterns](/gh-aw/guides/deterministic-agentic-patterns/) — adding custom `steps:` to workflows
- [Artifacts](/gh-aw/reference/artifacts/) — downloading the `otel.jsonl` mirror and other artifacts
