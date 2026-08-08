// @ts-check
/// <reference types="@actions/github-script" />

/**
 * render_detection_log — Detection log renderer for the detection job.
 *
 * Reads the detection engine log file and pipes it to stdout wrapped in GitHub
 * Actions group (`::group::` / `::endgroup::`) and stop-commands
 * (`::stop-commands::` / `::<token>::`) macros so that:
 *   - the output is folded into a collapsible section in the Actions log UI, and
 *   - any workflow-command-shaped lines inside the log are not interpreted by
 *     the runner (preventing command injection from agent-controlled content).
 *
 * Secret redaction is applied before the content is written so that credential-
 * shaped strings are replaced with `***REDACTED***` and MCP gateway tokens are
 * masked via `::add-mask::` at the runner level.
 *
 * This script is intended to run AFTER redact_secrets so that secrets are
 * redacted from the file on disk before this helper reads and re-emits them.
 * The in-line redaction here is a defence-in-depth layer for the stdout copy.
 */

"use strict";

const fs = require("fs");
const { redactBuiltInPatterns } = require("./redact_secrets.cjs");
const { renderLogToStdout } = require("./render_log_to_stdout.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

/** Default path to the detection engine log file. */
const DETECTION_LOG_PATH = "/tmp/gh-aw/threat-detection/detection.log";

/**
 * Renders the detection log file to stdout wrapped in GitHub Actions macros.
 *
 * The output sequence is:
 *   ::group::Detection Log
 *   ::stop-commands::<token>
 *   <redacted log content>
 *   ::<token>::
 *   ::endgroup::
 *
 * @param {string} [logPath] - Path to the log file; defaults to DETECTION_LOG_PATH.
 * @returns {Promise<void>}
 */
async function main(logPath) {
  const filePath = logPath || DETECTION_LOG_PATH;

  if (!fs.existsSync(filePath)) {
    core.info("Detection log not found, skipping render: " + filePath);
    return;
  }

  let stat;
  try {
    stat = fs.statSync(filePath);
  } catch (error) {
    core.warning("Failed to stat detection log: " + getErrorMessage(error));
    return;
  }

  if (stat.size === 0) {
    core.info("Detection log is empty, skipping render");
    return;
  }

  let content;
  try {
    content = fs.readFileSync(filePath, "utf8");
  } catch (error) {
    core.warning("Failed to read detection log: " + getErrorMessage(error));
    return;
  }

  // Apply in-line redaction of built-in credential patterns before emitting.
  const { content: redacted } = redactBuiltInPatterns(content);

  renderLogToStdout("Detection Log", redacted);

  core.info("Detection log rendered (" + stat.size + " bytes)");
}

module.exports = { main, DETECTION_LOG_PATH };
