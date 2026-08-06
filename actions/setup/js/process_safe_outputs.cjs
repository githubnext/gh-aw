// @ts-check
/// <reference types="@actions/github-script" />

const safeOutputHandlerManager = require("./safe_output_handler_manager.cjs");

/**
 * Run safe-output handlers.
 *
 * Process stdout/stderr logs are intentionally NOT captured to disk.
 * They could contain sensitive information from handler execution and
 * must never be packaged into artifacts.
 */
async function main() {
  await safeOutputHandlerManager.main();
}

module.exports = { main };
