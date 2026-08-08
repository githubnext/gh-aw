// @ts-check
/// <reference types="@actions/github-script" />

/**
 * render_log_to_stdout — Reusable helper for emitting log content to stdout
 * wrapped in GitHub Actions group + stop-commands macros with secret masking.
 *
 * Wraps `content` in:
 *   ::group::<groupTitle>
 *   ::stop-commands::<token>
 *   <content>
 *   ::<token>::
 *   ::endgroup::
 *
 * MCP gateway tokens are masked at the runner level before the content is
 * emitted so that subsequent output in the same step is also redacted.
 */

"use strict";

const crypto = require("crypto");
const { extractMCPGatewayTokens, MCP_GATEWAY_CONFIG_PATHS } = require("./redact_secrets.cjs");
const { maskSecret } = require("./actions_secret_masking.cjs");

/**
 * Masks MCP gateway tokens at the runner level and then writes `content` to
 * stdout wrapped in GitHub Actions group + stop-commands macros.
 *
 * The stop-commands token is generated with `crypto` so that nested pairs do
 * not interfere.
 *
 * @param {string} groupTitle - Label shown in the collapsible group header.
 * @param {string} content    - Already-redacted text to emit.
 */
function renderLogToStdout(groupTitle, content) {
  // Mask MCP gateway tokens at the runner level so the runner's own masking
  // pass will also replace them in subsequent output within this step.
  const gatewayTokens = extractMCPGatewayTokens(MCP_GATEWAY_CONFIG_PATHS);
  for (const token of gatewayTokens) {
    maskSecret(token);
  }

  // Use a cryptographically random token so nested stop-commands pairs do not
  // interfere.
  const stopToken = "render-" + crypto.randomBytes(12).toString("hex");

  process.stdout.write("::group::" + groupTitle + "\n");
  process.stdout.write("::stop-commands::" + stopToken + "\n");
  try {
    process.stdout.write(content);
    if (!content.endsWith("\n")) {
      process.stdout.write("\n");
    }
  } finally {
    process.stdout.write("::" + stopToken + "::\n");
    process.stdout.write("::endgroup::\n");
  }
}

module.exports = { renderLogToStdout };
