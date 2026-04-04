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
 * @returns {Promise<void>}
 */
async function run() {
  const { sendJobSetupSpan, isValidTraceId, isValidSpanId } = require(path.join(__dirname, "send_otlp_span.cjs"));

  const startMs = parseInt(process.env.SETUP_START_MS || "0", 10);
  const { traceId, spanId } = await sendJobSetupSpan({ startMs });

  // Expose trace ID as a step output for cross-job correlation.
  if (isValidTraceId(traceId) && process.env.GITHUB_OUTPUT) {
    appendFileSync(process.env.GITHUB_OUTPUT, `trace-id=${traceId}\n`);
  }

  // Propagate trace/span context to subsequent steps in this job.
  if (process.env.GITHUB_ENV) {
    if (isValidTraceId(traceId)) {
      appendFileSync(process.env.GITHUB_ENV, `GITHUB_AW_OTEL_TRACE_ID=${traceId}\n`);
    }
    if (isValidSpanId(spanId)) {
      appendFileSync(process.env.GITHUB_ENV, `GITHUB_AW_OTEL_PARENT_SPAN_ID=${spanId}\n`);
    }
  }
}

module.exports = { run };

// When invoked directly (node action_setup_otlp.cjs) from setup.sh,
// run immediately.  Non-fatal: errors are silently swallowed.
if (require.main === module) {
  run().catch(() => {});
}
