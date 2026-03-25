// Setup Activation Action - Main Entry Point
// Invokes setup.sh to copy activation job files to the agent environment

const { spawnSync } = require("child_process");
const path = require("path");

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
