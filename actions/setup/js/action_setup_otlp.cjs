// @ts-check
"use strict";

/**
 * action_setup_otlp.cjs
 *
 * Sends a gh-aw.job.setup OTLP span and writes the trace/span IDs to
 * GITHUB_OUTPUT and GITHUB_ENV.  Used by both:
 *
 *   - actions/setup/index.js  (dev/release/action mode)
 *   - actions/setup/setup.sh  (script mode)
 *
 * Having a single .cjs file ensures the two modes behave identically.
 *
 * Environment variables read:
 *   SETUP_START_MS  – epoch ms when setup began (set by callers)
 *   GITHUB_OUTPUT   – path to the GitHub Actions output file
 *   GITHUB_ENV      – path to the GitHub Actions env file
 *   INPUT_*         – standard GitHub Actions input env vars (read by sendJobSetupSpan)
 */

const path = require("path");
const { appendFileSync } = require("fs");

/**
 * Send the OTLP job-setup span and propagate trace context via GITHUB_OUTPUT /
 * GITHUB_ENV.  Non-fatal: all errors are silently swallowed.
 *
 * The trace-id is ALWAYS resolved and written to GITHUB_OUTPUT / GITHUB_ENV so
 * that cross-job span correlation works even when OTEL_EXPORTER_OTLP_ENDPOINT
 * is not configured.  The span itself is only sent when the endpoint is set.
 * @returns {Promise<void>}
 */
async function run() {
  const endpoint = process.env.OTEL_EXPORTER_OTLP_ENDPOINT;

  const { sendJobSetupSpan, isValidTraceId, isValidSpanId } = require(path.join(__dirname, "send_otlp_span.cjs"));

  const startMs = parseInt(process.env.SETUP_START_MS || "0", 10);

  // Explicitly read INPUT_TRACE_ID and pass it as options.traceId so the
  // activation job's trace ID is used even when process.env propagation
  // through GitHub Actions expression evaluation is unreliable.
  const inputTraceId = (process.env.INPUT_TRACE_ID || "").trim().toLowerCase();
  if (inputTraceId) {
    console.log(`[otlp] INPUT_TRACE_ID=${inputTraceId} (will reuse activation trace)`);
  } else {
    console.log("[otlp] INPUT_TRACE_ID not set, a new trace ID will be generated");
  }

  if (!endpoint) {
    console.log("[otlp] OTEL_EXPORTER_OTLP_ENDPOINT not set, skipping setup span");
  } else {
    console.log(`[otlp] sending setup span to ${endpoint}`);
  }

  const { traceId, spanId } = await sendJobSetupSpan({ startMs, traceId: inputTraceId || undefined });

  console.log(`[otlp] resolved trace-id=${traceId}`);

  if (endpoint) {
    console.log(`[otlp] setup span sent (traceId=${traceId}, spanId=${spanId})`);
  }

  // Always expose trace ID as a step output for cross-job correlation, even
  // when OTLP is not configured.  This ensures needs.*.outputs.setup-trace-id
  // is populated for downstream jobs regardless of observability configuration.
  if (isValidTraceId(traceId) && process.env.GITHUB_OUTPUT) {
    appendFileSync(process.env.GITHUB_OUTPUT, `trace-id=${traceId}\n`);
    console.log(`[otlp] trace-id=${traceId} written to GITHUB_OUTPUT`);
  }

  // Always propagate trace/span context to subsequent steps in this job so
  // that the conclusion span can find the same trace ID.
  if (process.env.GITHUB_ENV) {
    if (isValidTraceId(traceId)) {
      appendFileSync(process.env.GITHUB_ENV, `GITHUB_AW_OTEL_TRACE_ID=${traceId}\n`);
      console.log(`[otlp] GITHUB_AW_OTEL_TRACE_ID written to GITHUB_ENV`);
    }
    if (isValidSpanId(spanId)) {
      appendFileSync(process.env.GITHUB_ENV, `GITHUB_AW_OTEL_PARENT_SPAN_ID=${spanId}\n`);
      console.log(`[otlp] GITHUB_AW_OTEL_PARENT_SPAN_ID written to GITHUB_ENV`);
    }
  }
}

module.exports = { run };

// When invoked directly (node action_setup_otlp.cjs) from setup.sh,
// run immediately.  Non-fatal: errors are silently swallowed.
if (require.main === module) {
  run().catch(() => {});
}
