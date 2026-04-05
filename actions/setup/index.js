// Setup Activation Action - Main Entry Point
// Invokes setup.sh to copy activation job files to the agent environment

const { spawnSync } = require("child_process");
const path = require("path");

// Record start time for the OTLP span before any setup work begins.
const setupStartMs = Date.now();

// GitHub Actions sets INPUT_* env vars for JavaScript actions by converting
// input names to uppercase and replacing hyphens with underscores. Explicitly
// normalize inputs with hyphens in their names because some runner versions
// preserve the original hyphen instead of converting it to an underscore.
const safeOutputCustomTokens =
  process.env["INPUT_SAFE_OUTPUT_CUSTOM_TOKENS"] ||
  process.env["INPUT_SAFE-OUTPUT-CUSTOM-TOKENS"] ||
  "false";

// Normalize trace-id input: handle both INPUT_TRACE_ID (underscore, standard)
// and INPUT_TRACE-ID (hyphen, used by some runner versions).
const inputTraceId =
  process.env["INPUT_TRACE_ID"] ||
  process.env["INPUT_TRACE-ID"] ||
  "";

const result = spawnSync(path.join(__dirname, "setup.sh"), [], {
  stdio: "inherit",
  env: Object.assign({}, process.env, {
    INPUT_SAFE_OUTPUT_CUSTOM_TOKENS: safeOutputCustomTokens,
    INPUT_TRACE_ID: inputTraceId,
    // Tell setup.sh to skip the OTLP span: in action mode index.js sends it
    // after setup.sh returns so that the startMs captured here is used.
    GH_AW_SKIP_SETUP_OTLP: "1",
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
// Delegates to action_setup_otlp.cjs so that script mode (setup.sh) and
// dev/release mode share the same implementation.
// Explicitly set INPUT_TRACE_ID (normalized above) so action_setup_otlp.cjs
// always reads the underscore form regardless of runner version.
// The IIFE keeps the event loop alive until the fetch completes.
// Errors are swallowed: trace export failures must never break the workflow.
(async () => {
  try {
    process.env.SETUP_START_MS = String(setupStartMs);
    process.env.INPUT_TRACE_ID = inputTraceId;
    const { run } = require(path.join(__dirname, "js", "action_setup_otlp.cjs"));
    await run();
  } catch {
    // Non-fatal: silently ignore any OTLP export or output-write errors.
  }
})();
