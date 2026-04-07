// @ts-check
"use strict";

/**
 * action_conclusion_otlp.cjs
 *
 * Sends a `gh-aw.<jobName>.conclusion` OTLP span (or `gh-aw.job.conclusion`
 * when no job name is configured).  Used by both:
 *
 *   - actions/setup/post.js   (dev/release/action mode)
 *   - actions/setup/clean.sh  (script mode)
 *
 * Having a single .cjs file ensures the two modes behave identically.
 *
 * Environment variables read:
 *   INPUT_JOB_NAME – job name from the `job-name` action input; when set the
 *                    span is named "gh-aw.<name>.conclusion", otherwise
 *                    "gh-aw.job.conclusion".
 *   GH_AW_AGENT_CONCLUSION        – agent job result passed from the agent job
 *                                   ("success", "failure", "timed_out", etc.);
 *                                   "failure" and "timed_out" set the span
 *                                   status to STATUS_CODE_ERROR.
 *   GITHUB_AW_OTEL_JOB_START_MS   – epoch ms written by action_setup_otlp.cjs when
 *                                   setup finished; used as the span startMs so the
 *                                   conclusion span duration covers the actual job
 *                                   execution window rather than this step's overhead.
 *   GITHUB_AW_OTEL_TRACE_ID       – parent trace ID (set by action_setup_otlp.cjs)
 *   GITHUB_AW_OTEL_PARENT_SPAN_ID – parent span ID (set by action_setup_otlp.cjs)
 *   OTEL_EXPORTER_OTLP_ENDPOINT   – OTLP endpoint (no-op when not set)
 *
 * Runtime files read (optional):
 *   /tmp/gh-aw/github_rate_limits.jsonl – GitHub API rate-limit log written by
 *                                          github_rate_limit_logger.cjs; the last
 *                                          entry is read and its fields are included
 *                                          in the span as:
 *                                            gh-aw.github.rate_limit.remaining
 *                                            gh-aw.github.rate_limit.limit
 *                                            gh-aw.github.rate_limit.used
 *                                            gh-aw.github.rate_limit.resource
 *                                            gh-aw.github.rate_limit.reset
 */

const sendOtlpSpan = require("./send_otlp_span.cjs");
const { getActionInput } = require("./action_input_utils.cjs");

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

  // Read the job-start timestamp written by action_setup_otlp so the conclusion
  // span duration covers the actual job execution window, not just this step's overhead.
  const rawJobStartMs = parseInt(process.env.GITHUB_AW_OTEL_JOB_START_MS || "0", 10);
  const startMs = rawJobStartMs > 0 ? rawJobStartMs : undefined;

  const jobName = getActionInput("JOB_NAME");
  const spanName = jobName ? `gh-aw.${jobName}.conclusion` : "gh-aw.job.conclusion";
  console.log(`[otlp] sending conclusion span "${spanName}" to ${endpoint}`);

  await sendOtlpSpan.sendJobConclusionSpan(spanName, { startMs });
  console.log(`[otlp] conclusion span sent`);
}

module.exports = { run };

// When invoked directly (node action_conclusion_otlp.cjs) from clean.sh,
// run immediately.  Non-fatal: errors are silently swallowed.
if (require.main === module) {
  run().catch(() => {});
}
