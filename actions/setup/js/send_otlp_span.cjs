// @ts-check
/// <reference types="@actions/github-script" />

const { randomBytes } = require("crypto");
const fs = require("fs");
const { buildWorkflowCallId } = require("./aw_context.cjs");
const path = require("path");
const { nowMs } = require("./performance_now.cjs");
const { buildWorkflowRunUrl } = require("./workflow_metadata_helpers.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { readExperimentAssignments, EXPERIMENT_ASSIGNMENTS_PATH } = require("./experiment_helpers.cjs");

/**
 * send_otlp_span.cjs
 *
 * Sends a single OTLP (OpenTelemetry Protocol) trace span to the configured
 * HTTP/JSON endpoint.  Used by actions/setup to instrument each job execution
 * with basic telemetry.
 *
 * Design constraints:
 * - No-op when OTEL_EXPORTER_OTLP_ENDPOINT is not set (zero overhead).
 * - Errors are non-fatal: export failures must never break the workflow.
 * - No third-party dependencies: uses only Node built-ins + native fetch.
 */

// ---------------------------------------------------------------------------
// OTel GenAI engine-to-system mapping
// ---------------------------------------------------------------------------

/**
 * Maps gh-aw internal engine IDs to the OTel GenAI semantic-convention
 * `gen_ai.system` values expected by Grafana, Datadog, Honeycomb, and Sentry.
 * Unknown engines fall back to the engine ID as-is.
 *
 * Uses Object.create(null) to avoid prototype-pollution risks from keys like
 * "constructor" or "__proto__" returning unexpected non-string values.
 * @type {Record<string, string>}
 */
const ENGINE_TO_SYSTEM_MAP = Object.assign(Object.create(null), {
  copilot: "github_models",
  claude: "anthropic",
  codex: "openai",
  gemini: "google_vertex_ai",
});

// ---------------------------------------------------------------------------
// Low-level helpers
// ---------------------------------------------------------------------------

/**
 * Generate a random 16-byte trace ID encoded as a 32-character hex string.
 * @returns {string}
 */
function generateTraceId() {
  return randomBytes(16).toString("hex");
}

/**
 * Generate a random 8-byte span ID encoded as a 16-character hex string.
 * @returns {string}
 */
function generateSpanId() {
  return randomBytes(8).toString("hex");
}

/**
 * Convert a Unix timestamp in milliseconds to a nanosecond string suitable for
 * OTLP's `startTimeUnixNano` / `endTimeUnixNano` fields.
 *
 * BigInt arithmetic avoids floating-point precision loss for large timestamps.
 *
 * @param {number} ms - milliseconds since Unix epoch
 * @returns {string} nanoseconds since Unix epoch as a decimal string
 */
function toNanoString(ms) {
  return (BigInt(Math.floor(ms)) * 1_000_000n).toString();
}

/**
 * Build a single OTLP attribute object in the key-value format expected by the
 * OTLP/HTTP JSON wire format.
 *
 * @param {string} key
 * @param {string | number | boolean} value
 * @returns {{ key: string, value: object }}
 */
function buildAttr(key, value) {
  if (typeof value === "boolean") {
    return { key, value: { boolValue: value } };
  }
  if (typeof value === "number") {
    return { key, value: { intValue: value } };
  }
  return { key, value: { stringValue: String(value) } };
}

/**
 * Build the workflow-call identifier for the current run when enough GitHub
 * context is available.
 *
 * @param {string} runId
 * @param {string} runAttempt
 * @param {string} [workflowRef]
 * @returns {string}
 */
function buildCurrentWorkflowCallId(runId, runAttempt, workflowRef = process.env.GH_AW_CURRENT_WORKFLOW_REF || process.env.GITHUB_WORKFLOW_REF || "") {
  return buildWorkflowCallId(runId, runAttempt, workflowRef);
}

/**
 * Parse setup-time aw_context passed via environment before aw_info.json exists.
 *
 * @param {string | undefined} raw
 * @returns {Record<string, unknown>}
 */
function parseSetupAwContext(raw) {
  if (typeof raw !== "string" || raw.trim() === "") {
    return {};
  }
  try {
    const parsed = JSON.parse(raw);
    return parsed && typeof parsed === "object" && !Array.isArray(parsed) ? parsed : {};
  } catch {
    return {};
  }
}

/**
 * @param {unknown} value
 * @returns {string}
 */
function readContextString(value) {
  return typeof value === "string" ? value.trim() : "";
}

/**
 * Resolve live episode correlation attributes directly from runtime context.
 *
 * Prefer the canonical lineage fields propagated in aw_context: episode_id for
 * the full automation session, hop_id for the current workflow invocation, and
 * parent_hop_id for the immediate caller. Legacy workflow_call_id is accepted
 * only as a compatibility fallback when the canonical fields are absent. For
 * standalone runs we fall back to the current run's run_id-run_attempt pair so
 * every live span is still queryable as a bounded execution unit.
 *
 * @param {object} awInfo
 * @param {string} runId
 * @param {string} runAttempt
 * @returns {Array<{key: string, value: object}>}
 */
function buildEpisodeAttributesFromContext(awInfo, runId, runAttempt) {
  const currentHopId = buildCurrentWorkflowCallId(runId, runAttempt);
  const inheritedHopId = readContextString(awInfo.context?.hop_id) || readContextString(awInfo.context?.workflow_call_id);
  const episodeId = readContextString(awInfo.context?.episode_id) || inheritedHopId || currentHopId;
  const parentHopId = readContextString(awInfo.context?.parent_hop_id) || (inheritedHopId && inheritedHopId !== currentHopId ? inheritedHopId : "");
  const originEvent = readContextString(awInfo.context?.origin_event) || readContextString(awInfo.context?.event_type);
  const rootRepo = readContextString(awInfo.context?.root_repo) || readContextString(awInfo.context?.repo);
  const rootWorkflowId = readContextString(awInfo.context?.root_workflow_id) || readContextString(awInfo.context?.workflow_id);

  if (!episodeId) {
    return [];
  }

  const attributes = [buildAttr("gh-aw.episode.id", episodeId), buildAttr("gh-aw.episode.kind", parentHopId ? "workflow_call" : "run")];

  if (currentHopId) {
    attributes.push(buildAttr("gh-aw.hop.id", currentHopId));
    attributes.push(buildAttr("gh-aw.workflow_call.id", currentHopId));
  }
  if (parentHopId) {
    attributes.push(buildAttr("gh-aw.hop.parent_id", parentHopId));
    attributes.push(buildAttr("gh-aw.workflow_call.parent_id", parentHopId));
  }
  if (originEvent) {
    attributes.push(buildAttr("gh-aw.origin.event", originEvent));
  }
  if (rootRepo) {
    attributes.push(buildAttr("gh-aw.root.repo", rootRepo));
  }
  if (rootWorkflowId) {
    attributes.push(buildAttr("gh-aw.root.workflow_id", rootWorkflowId));
  }

  return attributes;
}

// ---------------------------------------------------------------------------
// OTLP SpanKind constants
// ---------------------------------------------------------------------------

/** OTLP SpanKind: span represents an internal operation (default for job lifecycle spans). */
const SPAN_KIND_INTERNAL = 1;
/** OTLP SpanKind: span covers server-side handling of a remote network request. */
const SPAN_KIND_SERVER = 2;
/** OTLP SpanKind: span represents an outbound remote call. */
const SPAN_KIND_CLIENT = 3;
/** OTLP SpanKind: span represents a message producer (e.g. message queue publish). */
const SPAN_KIND_PRODUCER = 4;
/** OTLP SpanKind: span represents a message consumer (e.g. message queue subscriber). */
const SPAN_KIND_CONSUMER = 5;

// ---------------------------------------------------------------------------
// OTLP payload builder
// ---------------------------------------------------------------------------

/**
 * @typedef {Object} OTLPSpanOptions
 * @property {string} traceId           - 32-char hex trace ID
 * @property {string} spanId            - 16-char hex span ID
 * @property {string} [parentSpanId]    - 16-char hex parent span ID; omitted for root spans
 * @property {string} spanName          - Human-readable span name
 * @property {number} startMs           - Span start time (ms since epoch)
 * @property {number} endMs             - Span end time (ms since epoch)
 * @property {string} serviceName       - Value for the service.name resource attribute
 * @property {string} [scopeVersion]    - gh-aw version string (e.g. from GH_AW_INFO_VERSION)
 * @property {Array<{key: string, value: object}>} attributes - Span attributes
 * @property {Array<{key: string, value: object}>} [resourceAttributes] - Extra resource attributes (e.g. github.repository, github.run_id)
 * @property {number} [statusCode]      - OTLP status code: 0=UNSET, 1=OK, 2=ERROR (defaults to 1)
 * @property {string} [statusMessage]   - Human-readable status message (included when statusCode is 2)
 * @property {number} [kind]            - OTLP SpanKind: use SPAN_KIND_* constants. Defaults to SPAN_KIND_INTERNAL (1).
 * @property {Array<{timeUnixNano: string, name: string, attributes: Array<{key: string, value: object}>}>} [events] - Span events following the OTel events spec (e.g. exception events).
 */

/**
 * @typedef {Object} OTLPSpanRecordOptions
 * @property {string} traceId
 * @property {string} spanId
 * @property {string} [parentSpanId]
 * @property {string} spanName
 * @property {number} startMs
 * @property {number} endMs
 * @property {Array<{key: string, value: object}>} attributes
 * @property {number} [statusCode]
 * @property {string} [statusMessage]
 * @property {number} [kind]
 * @property {Array<{timeUnixNano: string, name: string, attributes: Array<{key: string, value: object}>}>} [events]
 */

/**
 * Build the OTLP span object nested under `scopeSpans[].spans[]`.
 *
 * @param {OTLPSpanRecordOptions} opts
 * @returns {object}
 */
function buildOTLPSpan({ traceId, spanId, parentSpanId, spanName, startMs, endMs, attributes, statusCode, statusMessage, kind = SPAN_KIND_INTERNAL, events }) {
  const code = typeof statusCode === "number" ? statusCode : 1; // STATUS_CODE_OK
  /** @type {{ code: number, message?: string }} */
  const status = { code };
  if (statusMessage) {
    status.message = statusMessage;
  }
  return {
    traceId,
    spanId,
    ...(parentSpanId ? { parentSpanId } : {}),
    name: spanName,
    kind,
    startTimeUnixNano: toNanoString(startMs),
    endTimeUnixNano: toNanoString(endMs),
    status,
    attributes,
    ...(events && events.length > 0 ? { events } : {}),
  };
}

/**
 * Build resource attributes for an OTLP traces payload.
 *
 * @param {string} serviceName
 * @param {string | undefined} scopeVersion
 * @param {Array<{key: string, value: object}> | undefined} resourceAttributes
 * @returns {Array<{key: string, value: object}>}
 */
function buildOTLPResourceAttributes(serviceName, scopeVersion, resourceAttributes) {
  const baseResourceAttrs = [buildAttr("service.name", serviceName)];
  if (scopeVersion && scopeVersion !== "unknown") {
    baseResourceAttrs.push(buildAttr("service.version", scopeVersion));
  }
  return resourceAttributes ? [...baseResourceAttrs, ...resourceAttributes] : baseResourceAttrs;
}

/**
 * Build the standard GitHub Actions resource attributes shared by all OTLP spans
 * (setup, conclusion, and tool spans).  Centralises the attribute list so that
 * future additions propagate to every span type automatically.
 *
 * @param {{
 *   repository: string,
 *   runId: string,
 *   eventName?: string,
 *   ref?: string,
 *   refName?: string,
 *   headRef?: string,
 *   sha?: string,
 *   workflowRef?: string,
 *   staged: boolean,
 *   runAttempt?: string,
 * }} ctx
 * @returns {Array<{key: string, value: object}>}
 */
function buildGitHubActionsResourceAttributes({ repository, runId, eventName = "", ref = "", refName = "", headRef = "", sha = "", workflowRef = "", staged, runAttempt = "1" }) {
  const resourceAttributes = [buildAttr("github.repository", repository), buildAttr("github.run_id", runId), buildAttr("github.run_attempt", runAttempt)];
  if (repository && runId && repository.includes("/")) {
    const [owner, repo] = repository.split("/");
    resourceAttributes.push(buildAttr("github.actions.run_url", buildWorkflowRunUrl({ runId }, { owner, repo })));
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
  return resourceAttributes;
}

/**
 * Wrap one or more OTLP span objects in a single traces payload.
 *
 * @param {{
 *   serviceName: string,
 *   scopeVersion?: string,
 *   resourceAttributes?: Array<{key: string, value: object}>,
 *   spans: object[]
 * }} opts
 * @returns {object}
 */
function buildOTLPBatchPayload({ serviceName, scopeVersion, resourceAttributes, spans }) {
  return {
    resourceSpans: [
      {
        resource: {
          attributes: buildOTLPResourceAttributes(serviceName, scopeVersion, resourceAttributes),
        },
        scopeSpans: [
          {
            scope: { name: "gh-aw", version: scopeVersion || "unknown" },
            spans,
          },
        ],
      },
    ],
  };
}

/**
 * Split a large span set into chunked OTLP payloads so high-volume exporters
 * can amortize HTTP request overhead without creating oversized requests.
 *
 * @param {{
 *   serviceName: string,
 *   scopeVersion?: string,
 *   resourceAttributes?: Array<{key: string, value: object}>,
 *   spans: object[],
 *   maxSpansPerPayload?: number
 * }} opts
 * @returns {object[]}
 */
function buildOTLPBatchPayloads({ serviceName, scopeVersion, resourceAttributes, spans, maxSpansPerPayload = 100 }) {
  const normalizedMax = Number.isInteger(maxSpansPerPayload) && maxSpansPerPayload > 0 ? maxSpansPerPayload : 100;
  const payloads = [];
  for (let index = 0; index < spans.length; index += normalizedMax) {
    payloads.push(
      buildOTLPBatchPayload({
        serviceName,
        scopeVersion,
        resourceAttributes,
        spans: spans.slice(index, index + normalizedMax),
      })
    );
  }
  return payloads;
}

/**
 * Build an OTLP/HTTP JSON traces payload wrapping a single span.
 *
 * @param {OTLPSpanOptions} opts
 * @returns {object} - Ready to be serialised as JSON and POSTed to `/v1/traces`
 */
function buildOTLPPayload({ traceId, spanId, parentSpanId, spanName, startMs, endMs, serviceName, scopeVersion, attributes, resourceAttributes, statusCode, statusMessage, kind = SPAN_KIND_INTERNAL, events }) {
  return buildOTLPBatchPayload({
    serviceName,
    scopeVersion,
    resourceAttributes,
    spans: [buildOTLPSpan({ traceId, spanId, parentSpanId, spanName, startMs, endMs, attributes, statusCode, statusMessage, kind, events })],
  });
}

// ---------------------------------------------------------------------------
// Local JSONL mirror
// ---------------------------------------------------------------------------

/**
 * Path to the OTLP telemetry mirror file.
 * Every OTLP span payload is also appended here as a JSON line so that it can
 * be inspected via GitHub Actions artifacts without needing a live collector.
 * @type {string}
 */
const OTEL_JSONL_PATH = "/tmp/gh-aw/otel.jsonl";

/**
 * Append an OTLP payload as a single JSON line to the local telemetry mirror
 * file.  Creates the `/tmp/gh-aw` directory if it does not already exist.
 * Errors are silently swallowed — mirror failures must never break the workflow.
 *
 * @param {object} payload - OTLP traces payload
 * @returns {void}
 */
function appendToOTLPJSONL(payload) {
  try {
    fs.mkdirSync("/tmp/gh-aw", { recursive: true });
    fs.appendFileSync(OTEL_JSONL_PATH, JSON.stringify(payload) + "\n");
  } catch {
    // Mirror failures are non-fatal; do not propagate.
  }
}

// ---------------------------------------------------------------------------
// Experiment assignments
// ---------------------------------------------------------------------------
// readExperimentAssignments and EXPERIMENT_ASSIGNMENTS_PATH are imported from
// experiment_helpers.cjs above.

/**
 * Build OTLP span attributes for the active experiment assignments.
 *
 * Adds one `gh-aw.experiment.<name>` attribute per experiment (carrying the
 * selected variant string) and a single `gh-aw.experiments` attribute with a
 * compact JSON string of only the valid emitted assignments (key-sorted for
 * determinism), which enables simple substring searches in backends that do
 * not support per-attribute filtering.
 *
 * Invalid assignments (non-string or empty-string variants) are skipped for
 * both the per-experiment attributes and the aggregated JSON.
 *
 * Returns an empty array when no assignments are available.
 *
 * @param {Record<string, string> | null} assignments
 * @returns {Array<{key: string, value: object}>}
 */
function buildExperimentAttributes(assignments) {
  if (!assignments || typeof assignments !== "object") return [];
  const names = Object.keys(assignments).sort();
  if (names.length === 0) return [];
  const attrs = [];
  /** @type {Record<string, string>} */
  const validAssignments = {};
  for (const name of names) {
    const variant = assignments[name];
    if (typeof variant === "string" && variant) {
      attrs.push(buildAttr(`gh-aw.experiment.${name}`, variant));
      validAssignments[name] = variant;
    }
  }
  if (attrs.length > 0) {
    attrs.push(buildAttr("gh-aw.experiments", JSON.stringify(validAssignments)));
  }
  return attrs;
}

// ---------------------------------------------------------------------------
// HTTP transport
// ---------------------------------------------------------------------------

/**
 * Parse an `OTEL_EXPORTER_OTLP_HEADERS` value into a plain object suitable for
 * merging into a `Headers` / `fetch` `headers` option.
 *
 * The value follows the OpenTelemetry specification:
 *   key=value[,key=value...]
 * where individual keys and values may be percent-encoded.
 * Empty pairs (from leading/trailing/consecutive commas) are silently skipped.
 *
 * @param {string} raw - Raw header string (e.g. "Authorization=Bearer tok,X-Tenant=acme")
 * @returns {Record<string, string>} Parsed headers object
 */
function parseOTLPHeaders(raw) {
  if (!raw || !raw.trim()) return {};
  /** @type {Record<string, string>} */
  const result = {};
  for (const pair of raw.split(",")) {
    const eqIdx = pair.indexOf("=");
    if (eqIdx <= 0) continue; // skip malformed pairs (no =) or empty keys (= at start)
    // Decode before trimming so percent-encoded whitespace (%20) at edges is preserved correctly.
    const key = decodeURIComponent(pair.slice(0, eqIdx)).trim();
    const value = decodeURIComponent(pair.slice(eqIdx + 1)).trim();
    if (key) result[key] = value;
  }
  return result;
}

/**
 * Regular expression matching attribute key fragments that indicate the value
 * is sensitive and should be redacted before the payload is sent over the
 * wire.  The pattern is case-insensitive.  Word-boundary anchors (`\b`) are
 * used for `key` so that generic infrastructure keys like `sort_key` or
 * `cache_key` (where "key" is preceded by an underscore, a word character)
 * are **not** over-redacted, while dot-separated forms like `app.key` and
 * standalone `key` attributes are still caught.
 * @type {RegExp}
 */
const SENSITIVE_ATTR_KEY_RE = /token|secret|password|passwd|\bkey\b|auth|credential|api[_-]?key|access[_-]?key/i;

/**
 * Maximum length (in characters) allowed for a string attribute value.
 * Values that exceed this limit are truncated to avoid sending unexpectedly
 * large payloads to the OTLP collector.
 * @type {number}
 */
const MAX_ATTR_VALUE_LENGTH = 1024;

/**
 * Redaction placeholder substituted for sensitive attribute values.
 * @type {string}
 */
const REDACTED = "[REDACTED]";

/**
 * Sanitize an array of OTLP key-value attributes in-place (shallowly cloned).
 *
 * For each attribute:
 * - If the key matches {@link SENSITIVE_ATTR_KEY_RE} the string value is
 *   replaced with {@link REDACTED}.
 * - String values longer than {@link MAX_ATTR_VALUE_LENGTH} are truncated.
 *
 * @param {Array<{key: string, value: object}>} attrs
 * @returns {Array<{key: string, value: object}>}
 */
function sanitizeAttrs(attrs) {
  if (!Array.isArray(attrs)) return attrs;
  return attrs.map(attr => {
    if (!attr || typeof attr.key !== "string") return attr;
    const isSensitive = SENSITIVE_ATTR_KEY_RE.test(attr.key);
    const val = attr.value;
    if (typeof val !== "object" || val === null) return attr;
    if (isSensitive && "stringValue" in val) {
      return { key: attr.key, value: { stringValue: REDACTED } };
    }
    if (!isSensitive && "stringValue" in val && typeof val.stringValue === "string" && val.stringValue.length > MAX_ATTR_VALUE_LENGTH) {
      return { key: attr.key, value: { stringValue: val.stringValue.slice(0, MAX_ATTR_VALUE_LENGTH) } };
    }
    return attr;
  });
}

/**
 * Sanitize an OTLP traces payload before sending it over the wire.
 *
 * Walks the `resourceSpans[].resource.attributes`,
 * `resourceSpans[].scopeSpans[].spans[].attributes`, and
 * `resourceSpans[].scopeSpans[].spans[].events[].attributes` arrays and applies
 * {@link sanitizeAttrs} to each, redacting values for sensitive keys and
 * truncating excessively long string values.
 *
 * The original payload object is not mutated; a shallow-clone is returned.
 *
 * @param {object} payload - OTLP traces payload produced by {@link buildOTLPPayload}
 * @returns {object} Sanitized payload suitable for serialisation
 */
function sanitizeOTLPPayload(payload) {
  if (!payload || !Array.isArray(payload.resourceSpans)) return payload;
  return {
    ...payload,
    resourceSpans: payload.resourceSpans.map(rs => ({
      ...rs,
      resource: rs.resource ? { ...rs.resource, attributes: sanitizeAttrs(rs.resource.attributes) } : rs.resource,
      scopeSpans: Array.isArray(rs.scopeSpans)
        ? rs.scopeSpans.map(ss => ({
            ...ss,
            spans: Array.isArray(ss.spans)
              ? ss.spans.map(span => ({
                  ...span,
                  attributes: sanitizeAttrs(span.attributes),
                  events: Array.isArray(span.events) ? span.events.map(ev => ({ ...ev, attributes: sanitizeAttrs(ev.attributes) })) : span.events,
                }))
              : ss.spans,
          }))
        : rs.scopeSpans,
    })),
  };
}

// ---------------------------------------------------------------------------
// Multi-endpoint support
// ---------------------------------------------------------------------------

/**
 * @typedef {Object} OTLPEndpointEntry
 * @property {string} url      - OTLP base URL (e.g. https://traces.example.com:4317)
 * @property {string} [headers] - Per-endpoint headers in "key=value,key=value" format
 */

/**
 * Resolve the list of configured OTLP endpoints for the current run.
 *
 * Reads `GH_AW_OTLP_ENDPOINTS` (JSON-encoded array produced by the gh-aw
 * compiler for all endpoint configurations, including single-endpoint setups).
 * Returns an empty array when no endpoint is configured, so callers can skip
 * the export step without additional checks.
 *
 * @returns {OTLPEndpointEntry[]}
 */
function parseOTLPEndpoints() {
  const raw = process.env.GH_AW_OTLP_ENDPOINTS || "";
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw);
    if (Array.isArray(parsed) && parsed.length > 0) {
      /** @type {OTLPEndpointEntry[]} */
      const valid = parsed
        .filter(e => e && typeof e.url === "string" && e.url.trim() !== "")
        .map(e => ({
          url: e.url,
          ...(typeof e.headers === "string" && e.headers ? { headers: e.headers } : {}),
        }));
      return valid;
    }
  } catch {
    // Invalid JSON — no endpoints available.
  }
  return [];
}

/**
 * Send an OTLP payload to all configured endpoints concurrently.
 *
 * Uses `Promise.allSettled` so a failure on one endpoint never prevents
 * delivery to the others.  The local JSONL mirror is written once by the
 * caller before invoking this function (pass `skipJSONL: true`).
 *
 * @param {OTLPEndpointEntry[]} endpoints  - Resolved endpoint list from {@link parseOTLPEndpoints}
 * @param {object} payload                 - Serialisable OTLP JSON object
 * @param {{ maxRetries?: number, baseDelayMs?: number, skipJSONL?: boolean }} [opts]
 * @returns {Promise<void>}
 */
async function sendOTLPToAllEndpoints(endpoints, payload, opts = {}) {
  if (endpoints.length === 0) return;
  await Promise.allSettled(
    endpoints.map(ep =>
      sendOTLPSpan(ep.url, payload, {
        ...opts,
        // Pass per-endpoint headers so each collector receives only its own
        // credentials (not the merged set from a different endpoint).
        headersOverride: ep.headers !== undefined ? ep.headers : "",
      })
    )
  );
}

/**
 * POST an OTLP traces payload to `{endpoint}/v1/traces` with automatic retries.
 *
 * Failures are surfaced as `console.warn` messages and never thrown; OTLP
 * export failures must not break the workflow.  Uses exponential back-off
 * between attempts (100 ms, 200 ms) so the three total attempts finish in
 * well under a second in the typical success case.
 *
 * Reads `OTEL_EXPORTER_OTLP_HEADERS` from the environment and merges any
 * configured headers into every request, unless `headersOverride` is provided
 * (used for per-endpoint headers in the multi-endpoint case).
 *
 * @param {string} endpoint  - OTLP base URL (e.g. https://traces.example.com:4317)
 * @param {object} payload   - Serialisable OTLP JSON object
 * @param {{ maxRetries?: number, baseDelayMs?: number, skipJSONL?: boolean, headersOverride?: string }} [opts]
 * @returns {Promise<void>}
 */
async function sendOTLPSpan(endpoint, payload, { maxRetries = 2, baseDelayMs = 100, skipJSONL = false, headersOverride = undefined } = {}) {
  // Mirror payload locally so it survives even when the collector is unreachable.
  // Callers that already wrote the JSONL mirror pass skipJSONL: true to avoid a
  // duplicate line.
  if (!skipJSONL) {
    appendToOTLPJSONL(payload);
  }

  const url = endpoint.replace(/\/$/, "") + "/v1/traces";
  // Use headersOverride when explicitly provided (including empty string, which means
  // "this endpoint has no configured headers" in the multi-endpoint fan-out path).
  // Fall back to OTEL_EXPORTER_OTLP_HEADERS only when headersOverride is absent
  // (undefined), which is the legacy single-endpoint case.
  const rawHeaders = headersOverride !== undefined ? headersOverride : process.env.OTEL_EXPORTER_OTLP_HEADERS || "";
  const extraHeaders = parseOTLPHeaders(rawHeaders);
  const headers = { "Content-Type": "application/json", ...extraHeaders };
  for (let attempt = 0; attempt <= maxRetries; attempt++) {
    if (attempt > 0) {
      await new Promise(resolve => setTimeout(resolve, baseDelayMs * 2 ** (attempt - 1)));
    }
    try {
      const response = await fetch(url, {
        method: "POST",
        headers,
        body: JSON.stringify(sanitizeOTLPPayload(payload)),
      });
      if (response.ok) {
        return;
      }
      const msg = `HTTP ${response.status} ${response.statusText}`;
      if (attempt < maxRetries) {
        console.warn(`OTLP export attempt ${attempt + 1}/${maxRetries + 1} failed: ${msg}, retrying…`);
      } else {
        console.warn(`OTLP export failed after ${maxRetries + 1} attempts: ${msg}`);
      }
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      if (attempt < maxRetries) {
        console.warn(`OTLP export attempt ${attempt + 1}/${maxRetries + 1} error: ${msg}, retrying…`);
      } else {
        console.warn(`OTLP export error after ${maxRetries + 1} attempts: ${msg}`);
      }
    }
  }
}

// ---------------------------------------------------------------------------
// High-level: job setup span
// ---------------------------------------------------------------------------

/**
 * Regular expression that matches a valid OTLP trace ID: 32 lowercase hex characters.
 * @type {RegExp}
 */
const TRACE_ID_RE = /^[0-9a-f]{32}$/;

/**
 * Validate that a string is a well-formed OTLP trace ID (32 lowercase hex chars).
 * @param {string} id
 * @returns {boolean}
 */
function isValidTraceId(id) {
  return TRACE_ID_RE.test(id);
}

/**
 * Regular expression that matches a valid OTLP span ID: 16 lowercase hex characters.
 * @type {RegExp}
 */
const SPAN_ID_RE = /^[0-9a-f]{16}$/;

/**
 * Validate that a string is a well-formed OTLP span ID (16 lowercase hex chars).
 * @param {string} id
 * @returns {boolean}
 */
function isValidSpanId(id) {
  return SPAN_ID_RE.test(id);
}

/**
 * @typedef {Object} SendJobSetupSpanOptions
 * @property {number} [startMs]  - Override for the span start time (ms).  Defaults to `Date.now()`.
 * @property {string} [traceId] - Existing trace ID to reuse for cross-job correlation.
 *   When omitted the value is taken from the `INPUT_TRACE_ID` environment variable (the
 *   `trace-id` action input); if that is also absent the `otel_trace_id` field from
 *   `aw_info.context` is used (propagated from the parent workflow via `aw_context`);
 *   and if none of those are set a new random trace ID is generated.
 *   Pass the `trace-id` output of the activation job setup step to correlate all
 *   subsequent job spans under the same trace.
 */

/**
 * Send a `gh-aw.<jobName>.setup` span (or `gh-aw.job.setup` when no job name
 * is configured) to the configured OTLP endpoint.
 *
 * This is designed to be called from `actions/setup/index.js` immediately after
 * the setup script completes.  It always returns `{ traceId, spanId }` so callers
 * can expose the trace ID as an action output and write both values to `$GITHUB_ENV`
 * for downstream step correlation — even when `OTEL_EXPORTER_OTLP_ENDPOINT` is not
 * set (no span is sent in that case).
 * Errors are swallowed so the workflow is never broken by tracing failures.
 *
 * Environment variables consumed:
 * - `OTEL_EXPORTER_OTLP_ENDPOINT` – collector endpoint (required to send anything)
 * - `OTEL_SERVICE_NAME`            – service name (defaults to "gh-aw")
 * - `INPUT_JOB_NAME`               – job name passed via the `job-name` action input
 * - `INPUT_TRACE_ID`               – optional trace ID passed via the `trace-id` action input
 * - `GH_AW_INFO_WORKFLOW_NAME`     – workflow name injected by the gh-aw compiler
 * - `GH_AW_INFO_ENGINE_ID`         – engine ID injected by the gh-aw compiler
 * - `GITHUB_RUN_ID`                – GitHub Actions run ID
 * - `GITHUB_ACTOR`                 – GitHub Actions actor (user / bot)
 * - `GITHUB_REPOSITORY`            – `owner/repo` string
 *
 * Runtime files read (optional):
 * - `/tmp/gh-aw/aw_info.json` – when present, `context.otel_trace_id` is used as a fallback
 *   trace ID so that dispatched child workflows share the parent's OTLP trace;
 *   `context.otel_parent_span_id` is used as the parent span ID so the child's setup span
 *   is properly nested under the parent's setup span in the trace hierarchy; and
 *   `context.item_type`, `context.item_number`, `context.trigger_label`, and `context.comment_id`
 *   are emitted as `gh-aw.trigger.item_type`, `gh-aw.trigger.item_number`, `gh-aw.trigger.label`,
 *   and `gh-aw.trigger.comment_id` attributes so every span can be linked back to the GitHub item
 *   (and specific comment) that triggered the workflow
 *
 * @param {SendJobSetupSpanOptions} [options]
 * @returns {Promise<{ traceId: string, spanId: string }>} The trace and span IDs used.
 */
async function sendJobSetupSpan(options = {}) {
  // Resolve the trace ID before the early-return so it is always available as
  // an action output regardless of whether OTLP is configured.
  // Priority: options.traceId > INPUT_TRACE_ID > aw_info.context.otel_trace_id > newly generated ID.
  // Invalid (wrong length, non-hex) values are silently discarded.

  // Validate options.traceId if supplied; callers may pass raw user input.
  const optionsTraceId = options.traceId && isValidTraceId(options.traceId) ? options.traceId : "";

  // Normalize INPUT_TRACE_ID to lowercase before validating: OTLP requires lowercase
  // hex, but trace IDs pasted from external tools may use uppercase characters.
  // Also handle INPUT_TRACE-ID (with hyphen) in case the runner preserves the original
  // input name hyphen instead of converting it to an underscore.
  const rawInputTraceId = (process.env.INPUT_TRACE_ID || process.env["INPUT_TRACE-ID"] || "").trim().toLowerCase();
  const inputTraceId = isValidTraceId(rawInputTraceId) ? rawInputTraceId : "";

  // When this job was dispatched by a parent workflow, the parent's trace ID is
  // propagated via aw_context.otel_trace_id → aw_info.context.otel_trace_id so that
  // composite-action spans share a single trace with their caller.
  const awInfo = readJSONIfExists("/tmp/gh-aw/aw_info.json") || {};
  const setupAwContext = parseSetupAwContext(process.env.GH_AW_SETUP_AW_CONTEXT);
  if ((!awInfo.context || typeof awInfo.context !== "object") && Object.keys(setupAwContext).length > 0) {
    awInfo.context = setupAwContext;
  }
  const rawContextTraceId = typeof awInfo.context?.otel_trace_id === "string" ? awInfo.context.otel_trace_id.trim().toLowerCase() : "";
  const contextTraceId = isValidTraceId(rawContextTraceId) ? rawContextTraceId : "";
  // When this job was dispatched by a parent workflow, the parent's setup span ID is
  // propagated via aw_context.otel_parent_span_id → aw_info.context.otel_parent_span_id so
  // that the child's setup span is nested under the parent's setup span in the trace.
  const rawContextParentSpanId = typeof awInfo.context?.otel_parent_span_id === "string" ? awInfo.context.otel_parent_span_id.trim().toLowerCase() : "";
  const contextParentSpanId = isValidSpanId(rawContextParentSpanId) ? rawContextParentSpanId : "";
  const staged = awInfo.staged === true || process.env.GH_AW_INFO_STAGED === "true";
  const itemType = typeof awInfo.context?.item_type === "string" ? awInfo.context.item_type : "";
  const itemNumber = typeof awInfo.context?.item_number === "string" ? awInfo.context.item_number : "";
  const triggerLabel = typeof awInfo.context?.trigger_label === "string" ? awInfo.context.trigger_label : "";
  const commentId = typeof awInfo.context?.comment_id === "string" ? awInfo.context.comment_id : "";

  const traceId = optionsTraceId || inputTraceId || contextTraceId || generateTraceId();

  // Always generate a span ID so it can be written to GITHUB_ENV as
  // GITHUB_AW_OTEL_PARENT_SPAN_ID even when OTLP is not configured, allowing downstream
  // scripts to establish the correct parent span context.
  const spanId = generateSpanId();

  // Build the full payload unconditionally so the JSONL mirror is always written,
  // enabling artifact-based debugging even without a live OTLP collector.
  const startMs = options.startMs ?? nowMs();
  const endMs = nowMs();

  const serviceName = process.env.OTEL_SERVICE_NAME || "gh-aw";
  const jobName = process.env.INPUT_JOB_NAME || "";
  const workflowName = process.env.GH_AW_INFO_WORKFLOW_NAME || process.env.GH_AW_SETUP_WORKFLOW_NAME || process.env.GITHUB_WORKFLOW || "";
  const engineId = process.env.GH_AW_INFO_ENGINE_ID || "";
  const runId = process.env.GITHUB_RUN_ID || "";
  const runAttempt = process.env.GITHUB_RUN_ATTEMPT || "1";
  const actor = process.env.GITHUB_ACTOR || "";
  const repository = process.env.GITHUB_REPOSITORY || "";
  const eventName = process.env.GITHUB_EVENT_NAME || "";
  const ref = process.env.GITHUB_REF || "";
  const refName = process.env.GITHUB_REF_NAME || "";
  const headRef = process.env.GITHUB_HEAD_REF || "";
  const sha = process.env.GITHUB_SHA || "";
  const workflowRef = process.env.GH_AW_CURRENT_WORKFLOW_REF || process.env.GITHUB_WORKFLOW_REF || "";

  const attributes = [
    buildAttr("gh-aw.job.name", jobName),
    buildAttr("gh-aw.workflow.name", workflowName),
    buildAttr("gh-aw.run.id", runId),
    buildAttr("gh-aw.run.attempt", runAttempt),
    buildAttr("gh-aw.run.actor", actor),
    buildAttr("gh-aw.repository", repository),
  ];

  if (engineId) {
    attributes.push(buildAttr("gh-aw.engine.id", engineId));
  }
  if (eventName) {
    attributes.push(buildAttr("gh-aw.event_name", eventName));
  }
  // Deployment state: prefer the env var (set from github.event.deployment_status.state
  // in the compiled workflow), fall back to aw_context propagation via awInfo.
  const deploymentStateSetup =
    process.env.GH_AW_GITHUB_EVENT_DEPLOYMENT_STATUS_STATE || (typeof awInfo.deployment_state === "string" ? awInfo.deployment_state : "") || (typeof awInfo.context?.deployment_state === "string" ? awInfo.context.deployment_state : "");
  if (deploymentStateSetup) {
    attributes.push(buildAttr("gh-aw.deployment.state", deploymentStateSetup));
  }
  // Workflow run conclusion: from aw_info or aw_context propagation.
  const workflowRunConclusion = (typeof awInfo.workflow_run_conclusion === "string" ? awInfo.workflow_run_conclusion : "") || (typeof awInfo.context?.workflow_run_conclusion === "string" ? awInfo.context.workflow_run_conclusion : "");
  if (workflowRunConclusion) {
    attributes.push(buildAttr("gh-aw.workflow_run.conclusion", workflowRunConclusion));
  }
  attributes.push(buildAttr("gh-aw.staged", staged));
  if (itemType) attributes.push(buildAttr("gh-aw.trigger.item_type", itemType));
  if (itemNumber) attributes.push(buildAttr("gh-aw.trigger.item_number", itemNumber));
  if (triggerLabel) attributes.push(buildAttr("gh-aw.trigger.label", triggerLabel));
  if (commentId) attributes.push(buildAttr("gh-aw.trigger.comment_id", commentId));

  // Include experiment assignments so each span can be correlated with the
  // A/B variant selected for this run (written by pick_experiment.cjs).
  const experimentAssignments = readExperimentAssignments();
  attributes.push(...buildExperimentAttributes(experimentAssignments));
  attributes.push(...buildEpisodeAttributesFromContext(awInfo, runId, runAttempt));

  const resourceAttributes = buildGitHubActionsResourceAttributes({ repository, runId, eventName, ref, refName, headRef, sha, workflowRef, staged, runAttempt });

  const payload = buildOTLPPayload({
    traceId,
    spanId,
    ...(contextParentSpanId ? { parentSpanId: contextParentSpanId } : {}),
    spanName: jobName ? `gh-aw.${jobName}.setup` : "gh-aw.job.setup",
    startMs,
    endMs,
    serviceName,
    scopeVersion: process.env.GH_AW_INFO_VERSION || "unknown",
    attributes,
    resourceAttributes,
  });

  // Always mirror to JSONL — the artifact is useful even without a live collector.
  appendToOTLPJSONL(payload);

  const endpoints = parseOTLPEndpoints();
  if (endpoints.length === 0) {
    return { traceId, spanId };
  }

  // Pass skipJSONL: true so sendOTLPToAllEndpoints/sendOTLPSpan don't double-write the mirror.
  await sendOTLPToAllEndpoints(endpoints, payload, { skipJSONL: true });
  return { traceId, spanId };
}

// ---------------------------------------------------------------------------
// Utilities for conclusion span
// ---------------------------------------------------------------------------

/**
 * Safely read and parse a JSON file.  Returns `null` on any error (missing
 * file, invalid JSON, permission denied, etc.).
 *
 * @param {string} filePath - Absolute path to the JSON file
 * @returns {object | null}
 */
function readJSONIfExists(filePath) {
  try {
    return JSON.parse(fs.readFileSync(filePath, "utf8"));
  } catch {
    return null;
  }
}

/**
 * Path to the GitHub rate-limit JSONL log file.
 * Mirrors GITHUB_RATE_LIMITS_JSONL_PATH from constants.cjs without introducing
 * a runtime require() dependency on that module.
 * @type {string}
 */
const GITHUB_RATE_LIMITS_JSONL_PATH = "/tmp/gh-aw/github_rate_limits.jsonl";

/**
 * @typedef {Object} RateLimitEntry
 * @property {string} [resource]   - GitHub rate-limit resource category (e.g. "core", "graphql")
 * @property {number} [limit]      - Total request quota for the window
 * @property {number} [remaining]  - Requests remaining in the current window
 * @property {number} [used]       - Requests consumed in the current window
 * @property {string} [reset]      - ISO 8601 timestamp when the window resets
 * @property {string} [operation]  - API operation that produced this entry
 */

/**
 * Read the last entry from the GitHub rate-limit JSONL log file.
 * Returns the parsed entry or `null` when the file is absent, empty, or
 * contains no valid JSON lines.  Errors are silently swallowed — this is
 * an observability enrichment and must never break the workflow.
 *
 * @returns {RateLimitEntry | null}
 */
function readLastRateLimitEntry() {
  try {
    const content = fs.readFileSync(GITHUB_RATE_LIMITS_JSONL_PATH, "utf8");
    const lines = content.split("\n").filter(l => l.trim() !== "");
    if (lines.length === 0) return null;
    return JSON.parse(lines[lines.length - 1]);
  } catch {
    return null;
  }
}

// ---------------------------------------------------------------------------
// High-level: job conclusion span
// ---------------------------------------------------------------------------

/**
 * Send a conclusion span for a job to the configured OTLP endpoint.  Called
 * from the action post step so it runs at the end of every job that uses the
 * setup action.  The span carries workflow metadata read from `aw_info.json`
 * and the effective token count from `GH_AW_EFFECTIVE_TOKENS`.
 *
 * The span payload is always built and mirrored to the local JSONL file so
 * that it can be inspected via GitHub Actions artifacts without needing a live
 * collector.  The HTTP export to the OTLP endpoint is skipped when
 * `OTEL_EXPORTER_OTLP_ENDPOINT` is not set.  All errors are surfaced as
 * `console.warn` messages and never re-thrown.
 *
 * Environment variables consumed:
 * - `OTEL_EXPORTER_OTLP_ENDPOINT`  – collector endpoint
 * - `OTEL_SERVICE_NAME`             – service name (defaults to "gh-aw")
 * - `GH_AW_EFFECTIVE_TOKENS`        – total effective token count for the run
 * - `GH_AW_AGENT_CONCLUSION`        – agent job result ("success", "failure", "timed_out",
 *                                     "cancelled", "skipped"); when "failure" or "timed_out"
 *                                     the span status is set to STATUS_CODE_ERROR (2)
 * - `GH_AW_DETECTION_CONCLUSION`   – threat-detection scan outcome ("success", "warning",
 *                                     "failure", "skipped"); emitted as
 *                                     `gh-aw.detection.conclusion` when present
 * - `GH_AW_DETECTION_REASON`       – machine-readable reason for the detection conclusion
 *                                     (e.g. "threat_detected", "agent_failure"); emitted as
 *                                     `gh-aw.detection.reason` when present
 * - `INPUT_JOB_NAME`               – job name; set automatically by GitHub Actions from the
 *                                     `job-name` action input
 * - `GITHUB_AW_OTEL_TRACE_ID`      – trace ID written to GITHUB_ENV by the setup step;
 *                                     enables 1-trace-per-run when present
 * - `GITHUB_AW_OTEL_PARENT_SPAN_ID` – setup span ID written to GITHUB_ENV by the setup step;
 *                                     links this span as a child of the job setup span
 * - `GITHUB_RUN_ID`                 – GitHub Actions run ID
 * - `GITHUB_ACTOR`                  – GitHub Actions actor
 * - `GITHUB_REPOSITORY`             – `owner/repo` string
 *
 * Runtime files read:
 * - `/tmp/gh-aw/aw_info.json`    – workflow/engine metadata written by the agent job;
 *                                   `context.item_type`, `context.item_number`, and
 *                                   `context.trigger_label` are emitted as
 *                                   `gh-aw.trigger.item_type`, `gh-aw.trigger.item_number`,
 *                                   and `gh-aw.trigger.label` attributes so every span can
 *                                   be linked back to the GitHub item that triggered the workflow
 * - `/tmp/gh-aw/agent_usage.json` – per-type token breakdown written by parse_token_usage.cjs;
 *                                    provides `input_tokens`, `output_tokens`,
 *                                    `cache_read_tokens`, and `cache_write_tokens` counters
 *
 * @param {string} spanName - OTLP span name (e.g. `"gh-aw.job.conclusion"`)
 * @param {{ startMs?: number }} [options]
 * @returns {Promise<void>}
 */
async function sendJobConclusionSpan(spanName, options = {}) {
  const startMs = options.startMs ?? nowMs();

  // Read workflow metadata from aw_info.json (written by the agent job setup step).
  const awInfo = readJSONIfExists("/tmp/gh-aw/aw_info.json") || {};

  // Effective token count is surfaced by the agent job and passed to downstream jobs
  // via the GH_AW_EFFECTIVE_TOKENS environment variable.
  const rawET = process.env.GH_AW_EFFECTIVE_TOKENS || "";
  const effectiveTokens = rawET ? parseInt(rawET, 10) : NaN;

  const serviceName = process.env.OTEL_SERVICE_NAME || "gh-aw";
  const version = awInfo.agent_version || awInfo.version || process.env.GH_AW_INFO_VERSION || "unknown";

  // Prefer GITHUB_AW_OTEL_TRACE_ID (written to GITHUB_ENV by this job's setup step) so
  // all spans in the same job share one trace.  Fall back to aw_context.otel_trace_id
  // for cross-job correlation, then try the legacy workflow_call_id fallback.
  const envTraceId = (process.env.GITHUB_AW_OTEL_TRACE_ID || "").trim().toLowerCase();
  const inheritedTraceId = readContextString(awInfo.context?.otel_trace_id).toLowerCase();
  const awTraceId = typeof awInfo.context?.workflow_call_id === "string" ? awInfo.context.workflow_call_id.replace(/-/g, "") : "";
  let traceId = generateTraceId();
  if (isValidTraceId(envTraceId)) {
    traceId = envTraceId;
  } else if (isValidTraceId(inheritedTraceId)) {
    traceId = inheritedTraceId;
  } else if (awTraceId && isValidTraceId(awTraceId)) {
    traceId = awTraceId;
  }

  // Use GITHUB_AW_OTEL_PARENT_SPAN_ID (written to GITHUB_ENV by this job's setup step) so
  // conclusion spans are linked as children of the setup span (1 parent span per job).
  const rawParentSpanId = (process.env.GITHUB_AW_OTEL_PARENT_SPAN_ID || "").trim().toLowerCase();
  const parentSpanId = isValidSpanId(rawParentSpanId) ? rawParentSpanId : "";

  const workflowName = awInfo.workflow_name || process.env.GH_AW_INFO_WORKFLOW_NAME || process.env.GITHUB_WORKFLOW || "";
  const engineId = awInfo.engine_id || "";
  const model = awInfo.model || "";
  const staged = awInfo.staged === true;
  const itemType = typeof awInfo.context?.item_type === "string" ? awInfo.context.item_type : "";
  const itemNumber = typeof awInfo.context?.item_number === "string" ? awInfo.context.item_number : "";
  const triggerLabel = typeof awInfo.context?.trigger_label === "string" ? awInfo.context.trigger_label : "";
  const commentId = typeof awInfo.context?.comment_id === "string" ? awInfo.context.comment_id : "";
  const jobName = process.env.INPUT_JOB_NAME || "";
  const runId = process.env.GITHUB_RUN_ID || "";
  const runAttempt = awInfo.run_attempt || process.env.GITHUB_RUN_ATTEMPT || "1";
  const actor = process.env.GITHUB_ACTOR || "";
  const repository = process.env.GITHUB_REPOSITORY || "";
  const eventName = process.env.GITHUB_EVENT_NAME || "";
  const ref = process.env.GITHUB_REF || "";
  const refName = process.env.GITHUB_REF_NAME || "";
  const headRef = process.env.GITHUB_HEAD_REF || "";
  const sha = process.env.GITHUB_SHA || "";
  const workflowRef = process.env.GITHUB_WORKFLOW_REF || "";

  // Agent conclusion is passed to downstream jobs via GH_AW_AGENT_CONCLUSION.
  // Values: "success", "failure", "timed_out", "cancelled", "skipped".
  const agentConclusion = process.env.GH_AW_AGENT_CONCLUSION || "";

  // Detection conclusion and reason are injected from needs.detection.outputs.*
  // when threat detection is enabled in the compiled workflow.
  const detectionConclusion = process.env.GH_AW_DETECTION_CONCLUSION || "";
  const detectionReason = process.env.GH_AW_DETECTION_REASON || "";

  // Mark the span as an error when the agent job failed, timed out, or was cancelled.
  const isAgentFailure = agentConclusion === "failure" || agentConclusion === "timed_out";
  const isAgentCancelled = agentConclusion === "cancelled";
  const isAgentNonOK = isAgentFailure || isAgentCancelled;
  // STATUS_CODE_ERROR = 2, STATUS_CODE_OK = 1
  const statusCode = isAgentNonOK ? 2 : 1;
  let statusMessage;
  if (isAgentFailure) {
    statusMessage = `agent ${agentConclusion}`;
  } else if (isAgentCancelled) {
    statusMessage = "agent cancelled";
  }

  // Always read agent_output.json so output metrics are available on all outcomes.
  const agentOutput = readJSONIfExists("/tmp/gh-aw/agent_output.json") || {};
  const outputErrors = Array.isArray(agentOutput.errors) ? agentOutput.errors : [];
  const outputItems = Array.isArray(agentOutput.items) ? agentOutput.items : [];
  const errorMessages = outputErrors.map(getErrorMessage).filter(Boolean).slice(0, 5);

  if (isAgentFailure && errorMessages.length > 0) {
    statusMessage = `agent ${agentConclusion}: ${errorMessages[0]}`.slice(0, 256);
  }

  const attributes = [buildAttr("gh-aw.workflow.name", workflowName), buildAttr("gh-aw.run.id", runId), buildAttr("gh-aw.run.attempt", runAttempt), buildAttr("gh-aw.run.actor", actor), buildAttr("gh-aw.repository", repository)];

  if (jobName) attributes.push(buildAttr("gh-aw.job.name", jobName));
  if (engineId) attributes.push(buildAttr("gh-aw.engine.id", engineId));
  if (model) attributes.push(buildAttr("gen_ai.request.model", model));
  if (eventName) attributes.push(buildAttr("gh-aw.event_name", eventName));
  // Deployment state: prefer the env var (set from github.event.deployment_status.state
  // in the compiled workflow), fall back to aw_info.deployment_state or aw_context propagation.
  const deploymentStateConclusion =
    process.env.GH_AW_GITHUB_EVENT_DEPLOYMENT_STATUS_STATE || (typeof awInfo.deployment_state === "string" ? awInfo.deployment_state : "") || (typeof awInfo.context?.deployment_state === "string" ? awInfo.context.deployment_state : "");
  if (deploymentStateConclusion) {
    attributes.push(buildAttr("gh-aw.deployment.state", deploymentStateConclusion));
  }
  // Workflow run conclusion: from aw_info or aw_context propagation.
  const workflowRunConclusion = (typeof awInfo.workflow_run_conclusion === "string" ? awInfo.workflow_run_conclusion : "") || (typeof awInfo.context?.workflow_run_conclusion === "string" ? awInfo.context.workflow_run_conclusion : "");
  if (workflowRunConclusion) {
    attributes.push(buildAttr("gh-aw.workflow_run.conclusion", workflowRunConclusion));
  }
  attributes.push(buildAttr("gh-aw.staged", staged));
  if (itemType) attributes.push(buildAttr("gh-aw.trigger.item_type", itemType));
  if (itemNumber) attributes.push(buildAttr("gh-aw.trigger.item_number", itemNumber));
  if (triggerLabel) attributes.push(buildAttr("gh-aw.trigger.label", triggerLabel));
  if (commentId) attributes.push(buildAttr("gh-aw.trigger.comment_id", commentId));
  attributes.push(...buildEpisodeAttributesFromContext(awInfo, runId, runAttempt));
  if (!isNaN(effectiveTokens) && effectiveTokens > 0) {
    attributes.push(buildAttr("gh-aw.effective_tokens", effectiveTokens));
  }

  if (agentConclusion) {
    attributes.push(buildAttr("gh-aw.agent.conclusion", agentConclusion));
  }
  if (detectionConclusion) {
    attributes.push(buildAttr("gh-aw.detection.conclusion", detectionConclusion));
  }
  if (detectionReason) {
    attributes.push(buildAttr("gh-aw.detection.reason", detectionReason));
  }
  if (errorMessages.length > 0) {
    attributes.push(buildAttr("gh-aw.error.count", outputErrors.length));
    attributes.push(buildAttr("gh-aw.error.messages", errorMessages.join(" | ")));
  }
  attributes.push(buildAttr("gh-aw.output.item_count", outputItems.length));
  const rawItemTypes = outputItems.map(i => (i && typeof i.type === "string" ? i.type : "")).filter(Boolean);
  const itemTypes = [...new Set(rawItemTypes)].sort();
  if (itemTypes.length > 0) {
    attributes.push(buildAttr("gh-aw.output.item_types", itemTypes.join(",")));
  }

  // Enrich span with the most recent GitHub API rate-limit snapshot for post-run
  // observability.  Reads the last entry from github_rate_limits.jsonl so that
  // rate-limit headroom at conclusion time is visible in the OTLP span without
  // requiring a live collector to parse the artifact separately.
  const lastRateLimit = readLastRateLimitEntry();
  if (lastRateLimit) {
    if (typeof lastRateLimit.remaining === "number") {
      attributes.push(buildAttr("gh-aw.github.rate_limit.remaining", lastRateLimit.remaining));
    }
    if (typeof lastRateLimit.limit === "number") {
      attributes.push(buildAttr("gh-aw.github.rate_limit.limit", lastRateLimit.limit));
    }
    if (typeof lastRateLimit.used === "number") {
      attributes.push(buildAttr("gh-aw.github.rate_limit.used", lastRateLimit.used));
    }
    if (lastRateLimit.resource) {
      attributes.push(buildAttr("gh-aw.github.rate_limit.resource", String(lastRateLimit.resource)));
    }
    if (lastRateLimit.reset) {
      attributes.push(buildAttr("gh-aw.github.rate_limit.reset", String(lastRateLimit.reset)));
    }
  }

  // Include experiment assignments so each span can be correlated with the
  // A/B variant selected for this run (written by pick_experiment.cjs).
  const conclusionExperimentAssignments = readExperimentAssignments();
  attributes.push(...buildExperimentAttributes(conclusionExperimentAssignments));

  const resourceAttributes = buildGitHubActionsResourceAttributes({ repository, runId, eventName, ref, refName, headRef, sha, workflowRef, staged, runAttempt });
  // OpenTelemetry semantic convention for exceptions.  Each event has
  // name="exception" with "exception.type" and "exception.message" attributes,
  // making individual errors queryable and classifiable in backends like
  // Grafana Tempo, Honeycomb, and Datadog.
  const buildSpanEvents = eventTimeMs => {
    if (outputErrors.length === 0) {
      return [];
    }
    const errorTimeNano = toNanoString(eventTimeMs);
    return outputErrors
      .map(getErrorMessage)
      .filter(Boolean)
      .map(msg => {
        // Extract colon-prefixed type when available ("push_to_pull_request_branch:...")
        const colonIdx = msg.indexOf(":");
        const prefix = msg.slice(0, colonIdx);
        const hasValidPrefix = colonIdx > 0 && colonIdx < 64 && /^[a-z_][a-z0-9_.]*$/i.test(prefix);
        const exceptionType = hasValidPrefix ? `gh-aw.${prefix.toLowerCase()}` : "gh-aw.AgentError";
        const exceptionMessage = (hasValidPrefix ? msg.slice(colonIdx + 1).trim() : msg).slice(0, MAX_ATTR_VALUE_LENGTH);
        return {
          timeUnixNano: errorTimeNano,
          name: "exception",
          attributes: [buildAttr("exception.type", exceptionType), buildAttr("exception.message", exceptionMessage)],
        };
      });
  };

  const spanEvents = buildSpanEvents(nowMs());

  const agentStartMs = options.startMs;
  let agentEndMs = null;
  try {
    agentEndMs = fs.statSync("/tmp/gh-aw/agent_output.json").mtimeMs;
  } catch {
    // agent_output.json may be absent for agent failures and cancellations,
    // including timed-out or manually-cancelled runs where the process was
    // killed before writing output. Fall back to nowMs() so we still emit
    // the dedicated agent span for these cases.
    if ((isAgentFailure || isAgentCancelled) && jobName === "agent" && typeof agentStartMs === "number" && agentStartMs > 0) {
      agentEndMs = nowMs();
    }
  }

  // Read agent token-usage counters and build per-category breakdown attributes.
  // These are attached exclusively to the dedicated agent span (when one is emitted)
  // to avoid double-counting in backends that sum gen_ai.usage.* across all spans.
  // When no agent span is emitted the attributes fall through to the conclusion span
  // so a single query is still sufficient for observability.
  const agentUsage = readJSONIfExists("/tmp/gh-aw/agent_usage.json") || {};
  const usageAttrs = [];
  if (typeof agentUsage.input_tokens === "number" && agentUsage.input_tokens > 0) {
    usageAttrs.push(buildAttr("gen_ai.usage.input_tokens", agentUsage.input_tokens));
  }
  if (typeof agentUsage.output_tokens === "number" && agentUsage.output_tokens > 0) {
    usageAttrs.push(buildAttr("gen_ai.usage.output_tokens", agentUsage.output_tokens));
  }
  if (typeof agentUsage.cache_read_tokens === "number" && agentUsage.cache_read_tokens > 0) {
    usageAttrs.push(buildAttr("gen_ai.usage.cache_read.input_tokens", agentUsage.cache_read_tokens));
  }
  if (typeof agentUsage.cache_write_tokens === "number" && agentUsage.cache_write_tokens > 0) {
    usageAttrs.push(buildAttr("gen_ai.usage.cache_creation.input_tokens", agentUsage.cache_write_tokens));
  }

  const endpoints = parseOTLPEndpoints();
  const conclusionSpanId = generateSpanId();
  let agentSpanEmitted = false;
  if (jobName === "agent" && typeof agentStartMs === "number" && agentStartMs > 0 && typeof agentEndMs === "number" && agentEndMs > agentStartMs) {
    agentSpanEmitted = true;
    const agentSpanEvents = buildSpanEvents(agentEndMs);

    // Build OTel GenAI semantic convention attributes for the dedicated agent span.
    // These follow the OpenTelemetry GenAI specification and enable out-of-the-box
    // LLM dashboards in Grafana, Datadog, and Honeycomb without custom mappings.
    // Token-usage attributes are included here (and only here) to prevent
    // double-counting with the conclusion span.
    const agentAttributes = [...attributes, ...usageAttrs];
    // gen_ai.operation.name is Required by the OTel GenAI spec for inference spans.
    // All gh-aw agent executions are chat-style LLM completions.
    agentAttributes.push(buildAttr("gen_ai.operation.name", "chat"));
    // gen_ai.request.model is already present in agentAttributes via the spread above
    // (added to attributes at the top of this function); do not push again.
    // gen_ai.system is the OTel GenAI standard attribute for the LLM system/provider.
    // Map the gh-aw internal engine ID to the standardized value so backends can apply
    // native GenAI dashboard detection. The original engine ID is preserved in gh-aw.engine.
    if (engineId) {
      const genAiSystem = ENGINE_TO_SYSTEM_MAP[engineId] || engineId;
      agentAttributes.push(buildAttr("gen_ai.system", genAiSystem));
      agentAttributes.push(buildAttr("gh-aw.engine", engineId));
    }
    // gen_ai.workflow.name identifies the agentic workflow, matching the OTel spec example
    // use-cases (e.g. "multi_agent_rag", "customer_support_pipeline").
    if (workflowName) agentAttributes.push(buildAttr("gen_ai.workflow.name", workflowName));

    const agentPayload = buildOTLPPayload({
      traceId,
      spanId: generateSpanId(),
      parentSpanId: conclusionSpanId,
      spanName: jobName ? `gh-aw.${jobName}.agent` : "gh-aw.job.agent",
      startMs: agentStartMs,
      endMs: agentEndMs,
      serviceName,
      scopeVersion: version,
      attributes: agentAttributes,
      resourceAttributes,
      statusCode,
      statusMessage,
      events: agentSpanEvents,
      kind: SPAN_KIND_CLIENT,
    });
    appendToOTLPJSONL(agentPayload);
    if (endpoints.length > 0) {
      await sendOTLPToAllEndpoints(endpoints, agentPayload, { skipJSONL: true });
    }
  }

  // When no agent span was emitted, attach token-usage attributes to the conclusion span
  // so a single query remains sufficient for observability (no join required).
  if (!agentSpanEmitted) {
    attributes.push(...usageAttrs);
  }

  const payload = buildOTLPPayload({
    traceId,
    spanId: conclusionSpanId,
    ...(parentSpanId ? { parentSpanId } : {}),
    spanName,
    startMs,
    endMs: nowMs(),
    serviceName,
    scopeVersion: version,
    attributes,
    resourceAttributes,
    statusCode,
    statusMessage,
    events: spanEvents,
  });

  // Always mirror to JSONL — the artifact is useful even without a live collector.
  appendToOTLPJSONL(payload);

  if (endpoints.length === 0) {
    return;
  }

  // Pass skipJSONL: true so sendOTLPToAllEndpoints/sendOTLPSpan don't double-write the mirror.
  await sendOTLPToAllEndpoints(endpoints, payload, { skipJSONL: true });
}

module.exports = {
  SPAN_KIND_INTERNAL,
  SPAN_KIND_SERVER,
  SPAN_KIND_CLIENT,
  SPAN_KIND_PRODUCER,
  SPAN_KIND_CONSUMER,
  isValidTraceId,
  isValidSpanId,
  generateTraceId,
  generateSpanId,
  toNanoString,
  buildAttr,
  buildGitHubActionsResourceAttributes,
  buildOTLPSpan,
  buildOTLPBatchPayload,
  buildOTLPBatchPayloads,
  buildOTLPPayload,
  sanitizeOTLPPayload,
  parseOTLPHeaders,
  parseOTLPEndpoints,
  sendOTLPSpan,
  sendOTLPToAllEndpoints,
  readJSONIfExists,
  readLastRateLimitEntry,
  buildCurrentWorkflowCallId,
  buildEpisodeAttributesFromContext,
  GITHUB_RATE_LIMITS_JSONL_PATH,
  sendJobSetupSpan,
  sendJobConclusionSpan,
  OTEL_JSONL_PATH,
  appendToOTLPJSONL,
  buildExperimentAttributes,
};
