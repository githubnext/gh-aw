// @ts-check
/// <reference types="@actions/github-script" />

const { randomBytes } = require("crypto");
const fs = require("fs");
const { nowMs } = require("./performance_now.cjs");
const { buildWorkflowRunUrl } = require("./workflow_metadata_helpers.cjs");

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
 */

/**
 * Build an OTLP/HTTP JSON traces payload wrapping a single span.
 *
 * @param {OTLPSpanOptions} opts
 * @returns {object} - Ready to be serialised as JSON and POSTed to `/v1/traces`
 */
function buildOTLPPayload({ traceId, spanId, parentSpanId, spanName, startMs, endMs, serviceName, scopeVersion, attributes, resourceAttributes, statusCode, statusMessage, kind = SPAN_KIND_INTERNAL }) {
  const code = typeof statusCode === "number" ? statusCode : 1; // STATUS_CODE_OK
  /** @type {{ code: number, message?: string }} */
  const status = { code };
  if (statusMessage) {
    status.message = statusMessage;
  }
  const baseResourceAttrs = [buildAttr("service.name", serviceName)];
  if (scopeVersion && scopeVersion !== "unknown") {
    baseResourceAttrs.push(buildAttr("service.version", scopeVersion));
  }
  const allResourceAttrs = resourceAttributes ? [...baseResourceAttrs, ...resourceAttributes] : baseResourceAttrs;
  return {
    resourceSpans: [
      {
        resource: {
          attributes: allResourceAttrs,
        },
        scopeSpans: [
          {
            scope: { name: "gh-aw", version: scopeVersion || "unknown" },
            spans: [
              {
                traceId,
                spanId,
                ...(parentSpanId ? { parentSpanId } : {}),
                name: spanName,
                kind,
                startTimeUnixNano: toNanoString(startMs),
                endTimeUnixNano: toNanoString(endMs),
                status,
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
 * POST an OTLP traces payload to `{endpoint}/v1/traces` with automatic retries.
 *
 * Failures are surfaced as `console.warn` messages and never thrown; OTLP
 * export failures must not break the workflow.  Uses exponential back-off
 * between attempts (100 ms, 200 ms) so the three total attempts finish in
 * well under a second in the typical success case.
 *
 * Reads `OTEL_EXPORTER_OTLP_HEADERS` from the environment and merges any
 * configured headers into every request.
 *
 * @param {string} endpoint  - OTLP base URL (e.g. https://traces.example.com:4317)
 * @param {object} payload   - Serialisable OTLP JSON object
 * @param {{ maxRetries?: number, baseDelayMs?: number }} [opts]
 * @returns {Promise<void>}
 */
async function sendOTLPSpan(endpoint, payload, { maxRetries = 2, baseDelayMs = 100 } = {}) {
  // Mirror payload locally so it survives even when the collector is unreachable.
  appendToOTLPJSONL(payload);

  const url = endpoint.replace(/\/$/, "") + "/v1/traces";
  const extraHeaders = parseOTLPHeaders(process.env.OTEL_EXPORTER_OTLP_HEADERS || "");
  const headers = { "Content-Type": "application/json", ...extraHeaders };
  for (let attempt = 0; attempt <= maxRetries; attempt++) {
    if (attempt > 0) {
      await new Promise(resolve => setTimeout(resolve, baseDelayMs * 2 ** (attempt - 1)));
    }
    try {
      const response = await fetch(url, {
        method: "POST",
        headers,
        body: JSON.stringify(payload),
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
 *   trace ID so that dispatched child workflows share the parent's OTLP trace
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
  const rawContextTraceId = typeof awInfo.context?.otel_trace_id === "string" ? awInfo.context.otel_trace_id.trim().toLowerCase() : "";
  const contextTraceId = isValidTraceId(rawContextTraceId) ? rawContextTraceId : "";

  const traceId = optionsTraceId || inputTraceId || contextTraceId || generateTraceId();

  // Always generate a span ID so it can be written to GITHUB_ENV as
  // GITHUB_AW_OTEL_PARENT_SPAN_ID even when OTLP is not configured, allowing downstream
  // scripts to establish the correct parent span context.
  const spanId = generateSpanId();

  const endpoint = process.env.OTEL_EXPORTER_OTLP_ENDPOINT || "";
  if (!endpoint) {
    return { traceId, spanId };
  }

  const startMs = options.startMs ?? nowMs();
  const endMs = nowMs();

  const serviceName = process.env.OTEL_SERVICE_NAME || "gh-aw";
  const jobName = process.env.INPUT_JOB_NAME || "";
  const workflowName = process.env.GH_AW_INFO_WORKFLOW_NAME || process.env.GITHUB_WORKFLOW || "";
  const engineId = process.env.GH_AW_INFO_ENGINE_ID || "";
  const runId = process.env.GITHUB_RUN_ID || "";
  const runAttempt = process.env.GITHUB_RUN_ATTEMPT || "1";
  const actor = process.env.GITHUB_ACTOR || "";
  const repository = process.env.GITHUB_REPOSITORY || "";
  const eventName = process.env.GITHUB_EVENT_NAME || "";

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

  const resourceAttributes = [buildAttr("github.repository", repository), buildAttr("github.run_id", runId)];
  if (repository && runId) {
    const [owner, repo] = repository.split("/");
    resourceAttributes.push(buildAttr("github.actions.run_url", buildWorkflowRunUrl({ runId }, { owner, repo })));
  }
  if (eventName) {
    resourceAttributes.push(buildAttr("github.event_name", eventName));
  }

  const payload = buildOTLPPayload({
    traceId,
    spanId,
    spanName: jobName ? `gh-aw.${jobName}.setup` : "gh-aw.job.setup",
    startMs,
    endMs,
    serviceName,
    scopeVersion: process.env.GH_AW_INFO_VERSION || "unknown",
    attributes,
    resourceAttributes,
  });

  await sendOTLPSpan(endpoint, payload);
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
 * This is a no-op when `OTEL_EXPORTER_OTLP_ENDPOINT` is not set.  All errors
 * are surfaced as `console.warn` messages and never re-thrown.
 *
 * Environment variables consumed:
 * - `OTEL_EXPORTER_OTLP_ENDPOINT`  – collector endpoint
 * - `OTEL_SERVICE_NAME`             – service name (defaults to "gh-aw")
 * - `GH_AW_EFFECTIVE_TOKENS`        – total effective token count for the run
 * - `GH_AW_AGENT_CONCLUSION`        – agent job result ("success", "failure", "timed_out",
 *                                     "cancelled", "skipped"); when "failure" or "timed_out"
 *                                     the span status is set to STATUS_CODE_ERROR (2)
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
 * - `/tmp/gh-aw/aw_info.json` – workflow/engine metadata written by the agent job
 *
 * @param {string} spanName - OTLP span name (e.g. `"gh-aw.job.conclusion"`)
 * @param {{ startMs?: number }} [options]
 * @returns {Promise<void>}
 */
async function sendJobConclusionSpan(spanName, options = {}) {
  const endpoint = process.env.OTEL_EXPORTER_OTLP_ENDPOINT || "";
  if (!endpoint) {
    return;
  }

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
  // all spans in the same job share one trace.  Fall back to the workflow_call_id
  // from aw_info for cross-job correlation, then generate a fresh ID.
  const envTraceId = (process.env.GITHUB_AW_OTEL_TRACE_ID || "").trim().toLowerCase();
  const awTraceId = typeof awInfo.context?.workflow_call_id === "string" ? awInfo.context.workflow_call_id.replace(/-/g, "") : "";
  let traceId = generateTraceId();
  if (isValidTraceId(envTraceId)) {
    traceId = envTraceId;
  } else if (awTraceId && isValidTraceId(awTraceId)) {
    traceId = awTraceId;
  }

  // Use GITHUB_AW_OTEL_PARENT_SPAN_ID (written to GITHUB_ENV by this job's setup step) so
  // conclusion spans are linked as children of the setup span (1 parent span per job).
  const rawParentSpanId = (process.env.GITHUB_AW_OTEL_PARENT_SPAN_ID || "").trim().toLowerCase();
  const parentSpanId = isValidSpanId(rawParentSpanId) ? rawParentSpanId : "";

  const workflowName = awInfo.workflow_name || "";
  const engineId = awInfo.engine_id || "";
  const model = awInfo.model || "";
  const staged = awInfo.staged === true;
  const jobName = process.env.INPUT_JOB_NAME || "";
  const runId = process.env.GITHUB_RUN_ID || "";
  const runAttempt = awInfo.run_attempt || process.env.GITHUB_RUN_ATTEMPT || "1";
  const actor = process.env.GITHUB_ACTOR || "";
  const repository = process.env.GITHUB_REPOSITORY || "";
  const eventName = process.env.GITHUB_EVENT_NAME || "";

  // Agent conclusion is passed to downstream jobs via GH_AW_AGENT_CONCLUSION.
  // Values: "success", "failure", "timed_out", "cancelled", "skipped".
  const agentConclusion = process.env.GH_AW_AGENT_CONCLUSION || "";

  // Mark the span as an error when the agent job failed or timed out.
  const isAgentFailure = agentConclusion === "failure" || agentConclusion === "timed_out";
  // STATUS_CODE_ERROR = 2, STATUS_CODE_OK = 1
  const statusCode = isAgentFailure ? 2 : 1;
  let statusMessage = isAgentFailure ? `agent ${agentConclusion}` : undefined;

  // When the agent failed, read agent_output.json to surface structured error details.
  // Lazy-read: skip I/O entirely when the job succeeded or was cancelled.
  const agentOutput = isAgentFailure ? readJSONIfExists("/tmp/gh-aw/agent_output.json") || {} : {};
  const outputErrors = Array.isArray(agentOutput.errors) ? agentOutput.errors : [];
  const errorMessages = outputErrors
    .map(e => (e && typeof e.message === "string" ? e.message : String(e)))
    .filter(Boolean)
    .slice(0, 5);

  if (isAgentFailure && errorMessages.length > 0) {
    statusMessage = `agent ${agentConclusion}: ${errorMessages[0]}`.slice(0, 256);
  }

  const attributes = [buildAttr("gh-aw.workflow.name", workflowName), buildAttr("gh-aw.run.id", runId), buildAttr("gh-aw.run.attempt", runAttempt), buildAttr("gh-aw.run.actor", actor), buildAttr("gh-aw.repository", repository)];

  if (jobName) attributes.push(buildAttr("gh-aw.job.name", jobName));
  if (engineId) attributes.push(buildAttr("gh-aw.engine.id", engineId));
  if (model) attributes.push(buildAttr("gh-aw.model", model));
  attributes.push(buildAttr("gh-aw.staged", staged));
  if (!isNaN(effectiveTokens) && effectiveTokens > 0) {
    attributes.push(buildAttr("gh-aw.effective_tokens", effectiveTokens));
  }
  if (agentConclusion) {
    attributes.push(buildAttr("gh-aw.agent.conclusion", agentConclusion));
  }
  if (isAgentFailure && errorMessages.length > 0) {
    attributes.push(buildAttr("gh-aw.error.count", outputErrors.length));
    attributes.push(buildAttr("gh-aw.error.messages", errorMessages.join(" | ")));
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
  }

  const resourceAttributes = [buildAttr("github.repository", repository), buildAttr("github.run_id", runId)];
  if (repository && runId) {
    const [owner, repo] = repository.split("/");
    resourceAttributes.push(buildAttr("github.actions.run_url", buildWorkflowRunUrl({ runId }, { owner, repo })));
  }
  if (eventName) {
    resourceAttributes.push(buildAttr("github.event_name", eventName));
  }
  resourceAttributes.push(buildAttr("deployment.environment", staged ? "staging" : "production"));

  const payload = buildOTLPPayload({
    traceId,
    spanId: generateSpanId(),
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
  });

  await sendOTLPSpan(endpoint, payload);
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
  buildOTLPPayload,
  parseOTLPHeaders,
  sendOTLPSpan,
  readJSONIfExists,
  readLastRateLimitEntry,
  GITHUB_RATE_LIMITS_JSONL_PATH,
  sendJobSetupSpan,
  sendJobConclusionSpan,
  OTEL_JSONL_PATH,
  appendToOTLPJSONL,
};
