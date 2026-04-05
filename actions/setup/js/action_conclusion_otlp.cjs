// @ts-check
"use strict";

/**
 * action_conclusion_otlp.cjs
 *
 * Sends a gh-aw.job.conclusion OTLP span (or a span named after the current
 * job).  Used by both:
 *
 *   - actions/setup/post.js   (dev/release/action mode)
 *   - actions/setup/clean.sh  (script mode)
 *
 * Having a single .cjs file ensures the two modes behave identically.
 *
 * Environment variables read:
 *   INPUT_JOB_NAME – job name from the `job-name` action input; when set the
 *                    span is named "gh-aw.job.<name>", otherwise
 *                    "gh-aw.job.conclusion".
 *   GH_AW_AGENT_CONCLUSION        – agent job result passed from the agent job
 *                                   ("success", "failure", "timed_out", etc.);
 *                                   "failure" and "timed_out" set the span
 *                                   status to STATUS_CODE_ERROR.
 *   GITHUB_AW_OTEL_TRACE_ID       – parent trace ID (set by action_setup_otlp.cjs)
 *   GITHUB_AW_OTEL_PARENT_SPAN_ID – parent span ID (set by action_setup_otlp.cjs)
 *   OTEL_EXPORTER_OTLP_ENDPOINT   – OTLP endpoint (no-op when not set)
 */

const path = require("path");

/**
 * Send the OTLP job-conclusion span.  Non-fatal: all errors are silently
 * swallowed.
 * @returns {Promise<void>}
 */
async function run() {
  const endpoint = process.env.OTEL_EXPORTER_OTLP_ENDPOINT;
  if (!endpoint) {
    console.log("[otlp] OTEL_EXPORTER_OTLP_ENDPOINT not set, skipping conclusion span");
    return;
  }

  const spanName = process.env.INPUT_JOB_NAME ? `gh-aw.job.${process.env.INPUT_JOB_NAME}` : "gh-aw.job.conclusion";
  console.log(`[otlp] sending conclusion span "${spanName}" to ${endpoint}`);

  const { sendJobConclusionSpan } = require(path.join(__dirname, "send_otlp_span.cjs"));
  await sendJobConclusionSpan(spanName);
  console.log(`[otlp] conclusion span sent`);
}

module.exports = { run };

// When invoked directly (node action_conclusion_otlp.cjs) from clean.sh,
// run immediately.  Non-fatal: errors are silently swallowed.
if (require.main === module) {
  run().catch(() => {});
}
