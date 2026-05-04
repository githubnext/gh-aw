// @ts-check
"use strict";

/**
 * otlp.cjs
 *
 * Stable, service-level API for emitting custom OpenTelemetry spans from shared
 * agentic workflow imports.  Wraps the low-level `send_otlp_span.cjs` helpers
 * and reads all required environment variables automatically, so callers only
 * need to provide a tool name and any domain-specific attributes they want to record.
 *
 * Design goals:
 * - Minimal public surface: one primary function (`logSpan`) for the common case.
 * - Zero configuration: endpoint, trace context, and service name are resolved
 *   from the environment automatically.
 * - Non-fatal: export failures are logged as warnings and never throw.
 * - Stable: callers are isolated from internal refactors of `send_otlp_span.cjs`.
 *
 * Usage (in a `steps:` github-script step inside a shared import):
 *
 *   const otlp = require('/tmp/gh-aw/actions/otlp.cjs');
 *   const start = Date.now();
 *   // ... do work ...
 *   await otlp.logSpan('my-tool', { 'my-tool.items_processed': 42, 'my-tool.result': 'ok' }, { startMs: start });
 */

const path = require("path");

// Ensures global.core / global.context shims are available when this module
// is loaded outside the github-script runtime (e.g., in plain Node.js or the
// MCP server context where those globals are not injected automatically).
require(path.join(__dirname, "shim.cjs"));

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

/**
 * @typedef {Object} LogSpanOptions
 * @property {number}  [startMs]      - Span start time (ms since epoch). Defaults to `Date.now()`.
 * @property {number}  [endMs]        - Span end time   (ms since epoch). Defaults to `Date.now()`.
 * @property {string}  [traceId]      - Override the trace ID.  Defaults to `GITHUB_AW_OTEL_TRACE_ID`.
 * @property {string}  [parentSpanId] - Override the parent span ID.  Defaults to `GITHUB_AW_OTEL_PARENT_SPAN_ID`.
 * @property {string}  [endpoint]     - Override the OTLP endpoint.  Defaults to `OTEL_EXPORTER_OTLP_ENDPOINT`.
 * @property {boolean} [isError]      - When `true`, the span status is set to ERROR (code 2).
 * @property {string}  [errorMessage] - Human-readable status message included when `isError` is `true`.
 */

/**
 * Emit a single OTLP span for the given tool, correlated with the current
 * workflow run's distributed trace.
 *
 * All environment plumbing (endpoint, trace ID, parent span ID) is handled
 * automatically; callers only provide the tool name and their own attributes.
 *
 * Attribute values may be `string`, `number`, or `boolean`.  Keys that match
 * sensitive patterns (`token`, `secret`, `password`, etc.) are automatically
 * redacted before the payload is sent over the wire.
 *
 * @param {string} toolName
 *   Logical name for the tool being instrumented (e.g. `"my-scanner"`).
 *   Used as both the OTLP `service.name` resource attribute and as the span
 *   name prefix: `<toolName>.run`.
 *
 * @param {Record<string, string | number | boolean>} [attributes]
 *   Domain-specific span attributes emitted under the tool's own namespace.
 *   Example: `{ 'my-scanner.issues_found': 3, 'my-scanner.version': '1.2.0' }`.
 *
 * @param {LogSpanOptions} [options]
 *
 * @returns {Promise<void>}
 */
async function logSpan(toolName, attributes = {}, options = {}) {
  try {
    const { buildAttr, buildOTLPPayload, parseOTLPEndpoints, sendOTLPToAllEndpoints, sanitizeOTLPPayload, appendToOTLPJSONL, generateSpanId, isValidTraceId, isValidSpanId, SPAN_KIND_CLIENT } = require(
      path.join(__dirname, "send_otlp_span.cjs")
    );

    const now = Date.now();
    const startMs = options.startMs ?? now;
    const endMs = options.endMs ?? now;

    const traceId = options.traceId ?? process.env.GITHUB_AW_OTEL_TRACE_ID ?? "";
    const parentSpanId = options.parentSpanId ?? process.env.GITHUB_AW_OTEL_PARENT_SPAN_ID ?? "";

    if (!isValidTraceId(traceId)) {
      return;
    }

    const spanAttrs = Object.entries(attributes).map(([k, v]) => buildAttr(k, v));

    // Build resource attributes matching what job setup and conclusion spans carry
    // so that tool spans are visible to resource-level filtering in Grafana etc.
    const repository = process.env.GITHUB_REPOSITORY || "";
    const runId = process.env.GITHUB_RUN_ID || "";
    const eventName = process.env.GITHUB_EVENT_NAME || "";
    const ref = process.env.GITHUB_REF || "";
    const refName = process.env.GITHUB_REF_NAME || "";
    const headRef = process.env.GITHUB_HEAD_REF || "";
    const sha = process.env.GITHUB_SHA || "";
    const workflowRef = process.env.GH_AW_CURRENT_WORKFLOW_REF || process.env.GITHUB_WORKFLOW_REF || "";
    const staged = process.env.GH_AW_INFO_STAGED === "true";
    const scopeVersion = process.env.GH_AW_INFO_VERSION || "unknown";

    const resourceAttributes = [buildAttr("github.repository", repository), buildAttr("github.run_id", runId)];
    if (repository && runId && repository.includes("/")) {
      const [owner, repo] = repository.split("/");
      const serverUrl = process.env.GITHUB_SERVER_URL || "https://github.com";
      resourceAttributes.push(buildAttr("github.actions.run_url", `${serverUrl}/${owner}/${repo}/actions/runs/${runId}`));
    }
    if (eventName) {
      resourceAttributes.push(buildAttr("github.event_name", eventName));
    }
    if (ref) {
      resourceAttributes.push(buildAttr("github.ref", ref));
    }
    if (refName) {
      resourceAttributes.push(buildAttr("github.ref_name", refName));
    }
    if (headRef) {
      resourceAttributes.push(buildAttr("github.head_ref", headRef));
    }
    if (sha) {
      resourceAttributes.push(buildAttr("github.sha", sha));
    }
    if (workflowRef) {
      resourceAttributes.push(buildAttr("github.workflow_ref", workflowRef));
    }
    resourceAttributes.push(buildAttr("deployment.environment", staged ? "staging" : "production"));

    const payload = buildOTLPPayload({
      traceId,
      spanId: generateSpanId(),
      ...(isValidSpanId(parentSpanId) ? { parentSpanId } : {}),
      spanName: `${toolName}.run`,
      startMs,
      endMs,
      serviceName: toolName,
      scopeVersion,
      kind: SPAN_KIND_CLIENT,
      attributes: spanAttrs,
      resourceAttributes,
      statusCode: options.isError ? 2 : 1,
      ...(options.isError && options.errorMessage ? { statusMessage: options.errorMessage } : {}),
    });

    // Sanitize before mirroring so that the local JSONL debug file never
    // contains secrets, just like the over-the-wire export.
    appendToOTLPJSONL(sanitizeOTLPPayload(payload));

    // When an endpoint override is provided use the legacy single-endpoint path;
    // otherwise fan out to all configured endpoints concurrently.
    if (options.endpoint) {
      const { sendOTLPSpan } = require(path.join(__dirname, "send_otlp_span.cjs"));
      await sendOTLPSpan(options.endpoint, payload, { skipJSONL: true });
    } else {
      const endpoints = parseOTLPEndpoints();
      if (endpoints.length > 0) {
        await sendOTLPToAllEndpoints(endpoints, payload, { skipJSONL: true });
      }
    }
  } catch (err) {
    // Export failures must never break the workflow.
    console.warn(`[otlp] ${toolName}: failed to emit span: ${err instanceof Error ? err.message : String(err)}`);
  }
}

module.exports = { logSpan };
