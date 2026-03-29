// @ts-check
/**
 * run_apm_install.cjs
 *
 * Standalone entry-point for apm_install.cjs used in CI integration tests
 * and local development. Sets up lightweight CJS-compatible shims for the
 * @actions/* globals expected by apm_install.cjs, then calls main().
 *
 * Environment variables (consumed by apm_install.main):
 *   GITHUB_APM_PAT   – GitHub token (falls back to GITHUB_TOKEN)
 *   APM_PACKAGES     – JSON array of package slugs
 *   APM_WORKSPACE    – workspace directory for downloaded files + lockfile
 *
 * Usage:
 *   node actions/setup/js/run_apm_install.cjs
 */

"use strict";

const { setupGlobals } = require("./setup_globals.cjs");
const { main } = require("./apm_install.cjs");

// Minimal shim for @actions/core — only the methods used by apm_install.cjs.
const core = {
  info: msg => console.log(msg),
  warning: msg => console.warn(`::warning::${msg}`),
  error: msg => console.error(`::error::${msg}`),
  setFailed: msg => {
    console.error(`::error::${msg}`);
    process.exitCode = 1;
  },
  setOutput: (name, value) => console.log(`::set-output name=${name}::${value}`),
};

// Wire shims into globals so apm_install.cjs can use them.
setupGlobals(
  core, // logging
  {}, // @actions/github — not used directly (apm_install creates its own Octokit)
  {}, // GitHub Actions event context — not used
  {}, // @actions/exec — not used
  {} // @actions/io — not used
);

main().catch(err => {
  console.error(`::error::${err.message}`);
  process.exit(1);
});
