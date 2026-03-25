// @ts-check
/**
 * run_apm_unpack.cjs
 *
 * Standalone entry-point for apm_unpack.cjs used in CI integration tests.
 * Sets up lightweight CJS-compatible shims for the @actions/* globals expected
 * by apm_unpack.cjs, then calls main().
 *
 * The @actions/core v3+ package is ESM-only and cannot be loaded via require().
 * The shims below reproduce the subset of the API that apm_unpack.cjs uses:
 *   core.info / core.warning / core.error / core.setFailed / core.setOutput
 *   exec.exec(cmd, args, options)
 *
 * Environment variables (consumed by apm_unpack.main):
 *   APM_BUNDLE_DIR  – directory containing the *.tar.gz bundle
 *   OUTPUT_DIR      – destination directory for deployed files
 *
 * Usage:
 *   node actions/setup/js/run_apm_unpack.cjs
 */

"use strict";

const { spawnSync } = require("child_process");
const { setupGlobals } = require("./setup_globals.cjs");
const { main } = require("./apm_unpack.cjs");

// Minimal shim for @actions/core — only the methods used by apm_unpack.cjs.
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

// Minimal shim for @actions/exec — only exec() is used by apm_unpack.cjs.
const exec = {
  exec: async (cmd, args = [], opts = {}) => {
    const result = spawnSync(cmd, args, { stdio: "inherit", ...opts });
    if (result.status !== 0) {
      throw new Error(`Command failed: ${cmd} ${args.join(" ")} (exit ${result.status})`);
    }
    return result.status;
  },
};

// Wire shims into globals so apm_unpack.cjs can use them.
// Passing empty objects for github (GraphQL client) and context (event payload)
// because apm_unpack does not use GitHub API or event metadata.
setupGlobals(
  core, // logging, outputs, inputs
  {}, // @actions/github – not used by apm_unpack
  {}, // GitHub Actions event context – not used by apm_unpack
  exec, // runs `tar -xzf`
  {} // @actions/io    – not used by apm_unpack
);

main().catch(err => {
  console.error(`::error::${err.message}`);
  process.exit(1);
});
