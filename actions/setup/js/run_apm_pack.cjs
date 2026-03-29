// @ts-check
/**
 * run_apm_pack.cjs
 *
 * Standalone entry-point for apm_pack.cjs used in CI integration tests and
 * local development. Sets up lightweight CJS-compatible shims for the
 * @actions/* globals expected by apm_pack.cjs (which imports apm_unpack.cjs),
 * then calls main().
 *
 * The @actions/core v3+ package is ESM-only and cannot be loaded via require().
 * The shims below reproduce the subset of the API used by apm_pack.cjs:
 *   core.info / core.warning / core.error / core.setFailed / core.setOutput
 *   exec.exec(cmd, args, options)
 *
 * Environment variables (consumed by apm_pack.main):
 *   APM_WORKSPACE     – project root with apm.lock.yaml and installed files
 *   APM_BUNDLE_OUTPUT – directory where the bundle archive is written
 *   APM_TARGET        – pack target (claude, copilot/vscode, cursor, opencode, all)
 *
 * Usage:
 *   node actions/setup/js/run_apm_pack.cjs
 */

"use strict";

const { spawnSync } = require("child_process");
const { setupGlobals } = require("./setup_globals.cjs");
const { main } = require("./apm_pack.cjs");

// Minimal shim for @actions/core — only the methods used by apm_pack.cjs.
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

// Minimal shim for @actions/exec — only exec() is used by apm_pack.cjs.
const exec = {
  exec: async (cmd, args = [], opts = {}) => {
    const result = spawnSync(cmd, args, { stdio: "inherit", ...opts });
    if (result.status !== 0) {
      throw new Error(`Command failed: ${cmd} ${args.join(" ")} (exit ${result.status})`);
    }
    return result.status;
  },
};

// Wire shims into globals so apm_pack.cjs (and the imported apm_unpack.cjs) can use them.
setupGlobals(
  core, // logging, outputs, inputs
  {}, // @actions/github – not used by apm_pack
  {}, // GitHub Actions event context – not used by apm_pack
  exec, // runs `tar -czf`
  {} // @actions/io    – not used by apm_pack
);

main().catch(err => {
  console.error(`::error::${err.message}`);
  process.exit(1);
});
