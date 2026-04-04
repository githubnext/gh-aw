// @ts-check
/// <reference types="@actions/github-script" />

const { randomBytes } = require("crypto");

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

// ---------------------------------------------------------------------------
// OTLP payload builder
// ---------------------------------------------------------------------------

/**
 * @typedef {Object} OTLPSpanOptions
 * @property {string} traceId       - 32-char hex trace ID
 * @property {string} spanId        - 16-char hex span ID
 * @property {string} spanName      - Human-readable span name
 * @property {number} startMs       - Span start time (ms since epoch)
 * @property {number} endMs         - Span end time (ms since epoch)
 * @property {string} serviceName   - Value for the service.name resource attribute
 * @property {Array<{key: string, value: object}>} attributes - Span attributes
 */

/**
 * Build an OTLP/HTTP JSON traces payload wrapping a single span.
 *
 * @param {OTLPSpanOptions} opts
 * @returns {object} - Ready to be serialised as JSON and POSTed to `/v1/traces`
 */
function buildOTLPPayload({ traceId, spanId, spanName, startMs, endMs, serviceName, attributes }) {
  return {
    resourceSpans: [
      {
        resource: {
          attributes: [buildAttr("service.name", serviceName)],
        },
        scopeSpans: [
          {
            scope: { name: "gh-aw.setup", version: "1.0.0" },
            spans: [
              {
                traceId,
                spanId,
                name: spanName,
                kind: 2, // SPAN_KIND_SERVER
                startTimeUnixNano: toNanoString(startMs),
                endTimeUnixNano: toNanoString(endMs),
                status: { code: 1 }, // STATUS_CODE_OK
                attributes,
              },
            ],
          },
        ],
      },
    ],
  };
}

// ---------------------------------------------------------------------------
// HTTP transport
// ---------------------------------------------------------------------------

/**
 * POST an OTLP traces payload to `{endpoint}/v1/traces`.
 *
 * @param {string} endpoint - OTLP base URL (e.g. https://traces.example.com:4317)
 * @param {object} payload  - Serialisable OTLP JSON object
 * @returns {Promise<void>}
 * @throws {Error} when the server returns a non-2xx status
 */
async function sendOTLPSpan(endpoint, payload) {
  const url = endpoint.replace(/\/$/, "") + "/v1/traces";
  const response = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  if (!response.ok) {
    throw new Error(`OTLP export failed: HTTP ${response.status} ${response.statusText}`);
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
 * @typedef {Object} SendJobSetupSpanOptions
 * @property {number} [startMs]  - Override for the span start time (ms).  Defaults to `Date.now()`.
 * @property {string} [traceId] - Existing trace ID to reuse for cross-job correlation.
 *   When omitted the value is taken from the `INPUT_TRACE_ID` environment variable (the
 *   `trace-id` action input); if that is also absent a new random trace ID is generated.
 *   Pass the `trace-id` output of the activation job setup step to correlate all
 *   subsequent job spans under the same trace.
 */

/**
 * Send a `gh-aw.job.setup` span to the configured OTLP endpoint.
 *
 * This is designed to be called from `actions/setup/index.js` immediately after
 * the setup script completes.  It always returns the trace ID so callers can
 * expose it as an action output for cross-job correlation — even when
 * `OTEL_EXPORTER_OTLP_ENDPOINT` is not set (no span is sent in that case).
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
 * @param {SendJobSetupSpanOptions} [options]
 * @returns {Promise<string>} The trace ID used for the span (generated or passed in).
 */
async function sendJobSetupSpan(options = {}) {
  // Resolve the trace ID before the early-return so it is always available as
  // an action output regardless of whether OTLP is configured.
  // Priority: options.traceId > INPUT_TRACE_ID env var > newly generated ID.
  // Invalid (wrong length, non-hex) values are silently discarded.

  // Validate options.traceId if supplied; callers may pass raw user input.
  const optionsTraceId = options.traceId && isValidTraceId(options.traceId) ? options.traceId : "";

  // Normalise INPUT_TRACE_ID to lowercase before validating: OTLP requires lowercase
  // hex, but trace IDs pasted from external tools may use uppercase characters.
  const rawInputTraceId = (process.env.INPUT_TRACE_ID || "").trim().toLowerCase();
  const inputTraceId = isValidTraceId(rawInputTraceId) ? rawInputTraceId : "";

  const traceId = optionsTraceId || inputTraceId || generateTraceId();

  const endpoint = process.env.OTEL_EXPORTER_OTLP_ENDPOINT || "";
  if (!endpoint) {
    return traceId;
  }

  const startMs = options.startMs ?? Date.now();
  const endMs = Date.now();

  const serviceName = process.env.OTEL_SERVICE_NAME || "gh-aw";
  const jobName = process.env.INPUT_JOB_NAME || "";
  const workflowName = process.env.GH_AW_INFO_WORKFLOW_NAME || process.env.GITHUB_WORKFLOW || "";
  const engineId = process.env.GH_AW_INFO_ENGINE_ID || "";
  const runId = process.env.GITHUB_RUN_ID || "";
  const actor = process.env.GITHUB_ACTOR || "";
  const repository = process.env.GITHUB_REPOSITORY || "";

  const attributes = [buildAttr("gh-aw.job.name", jobName), buildAttr("gh-aw.workflow.name", workflowName), buildAttr("gh-aw.run.id", runId), buildAttr("gh-aw.run.actor", actor), buildAttr("gh-aw.repository", repository)];

  if (engineId) {
    attributes.push(buildAttr("gh-aw.engine.id", engineId));
  }

  const payload = buildOTLPPayload({
    traceId,
    spanId: generateSpanId(),
    spanName: "gh-aw.job.setup",
    startMs,
    endMs,
    serviceName,
    attributes,
  });

  await sendOTLPSpan(endpoint, payload);
  return traceId;
}

module.exports = {
  isValidTraceId,
  generateTraceId,
  generateSpanId,
  toNanoString,
  buildAttr,
  buildOTLPPayload,
  sendOTLPSpan,
  sendJobSetupSpan,
};
