// Setup Activation Action - Main Entry Point
// Invokes setup.sh to copy activation job files to the agent environment

const { spawnSync } = require("child_process");
const path = require("path");

// Record start time for the OTLP span before any setup work begins.
const setupStartMs = Date.now();

// GitHub Actions sets INPUT_* env vars for JavaScript actions by converting
// input names to uppercase and replacing hyphens with underscores. Explicitly
// normalise the safe-output-custom-tokens input to ensure setup.sh finds it.
const safeOutputCustomTokens =
  process.env["INPUT_SAFE_OUTPUT_CUSTOM_TOKENS"] ||
  process.env["INPUT_SAFE-OUTPUT-CUSTOM-TOKENS"] ||
  "false";

const result = spawnSync(path.join(__dirname, "setup.sh"), [], {
  stdio: "inherit",
  env: Object.assign({}, process.env, {
    INPUT_SAFE_OUTPUT_CUSTOM_TOKENS: safeOutputCustomTokens,
  }),
});

if (result.error) {
  console.error(`Failed to run setup.sh: ${result.error.message}`);
  process.exit(1);
}

if (result.status !== 0) {
  process.exit(result.status ?? 1);
}

// Send a gh-aw.job.setup span to the OTLP endpoint when configured.
// The IIFE returns a Promise that keeps the Node.js event loop alive until
// the fetch request completes, so the span is delivered before the process
// exits naturally.  Errors are swallowed: trace export failures must never
// break the workflow.
(async () => {
  try {
    const { appendFileSync } = require("fs");
    const { isValidTraceId, isValidSpanId, sendJobSetupSpan } = require(path.join(__dirname, "js", "send_otlp_span.cjs"));
    const { traceId, spanId } = await sendJobSetupSpan({ startMs: setupStartMs });
    // Expose the trace ID as an action output so downstream jobs can reference it
    // via `steps.<id>.outputs.trace-id` for cross-job trace correlation.
    if (isValidTraceId(traceId) && process.env.GITHUB_OUTPUT) {
      appendFileSync(process.env.GITHUB_OUTPUT, `trace-id=${traceId}\n`);
    }
    // Write both the trace ID and setup span ID to GITHUB_ENV so all subsequent
    // steps in this job automatically inherit the parent trace context:
    //   GH_AW_TRACE_ID       – shared trace ID (1 trace per run)
    //   GH_AW_PARENT_SPAN_ID – setup span ID used as parent (1 parent span per job)
    if (process.env.GITHUB_ENV) {
      if (isValidTraceId(traceId)) {
        appendFileSync(process.env.GITHUB_ENV, `GH_AW_TRACE_ID=${traceId}\n`);
      }
      if (isValidSpanId(spanId)) {
        appendFileSync(process.env.GITHUB_ENV, `GH_AW_PARENT_SPAN_ID=${spanId}\n`);
      }
    }
  } catch {
    // Non-fatal: silently ignore any OTLP export or output-write errors.
  }
})();
