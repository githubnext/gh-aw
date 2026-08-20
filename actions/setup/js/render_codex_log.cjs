// @ts-check
/// <reference types="@actions/github-script" />

/**
 * render_codex_log — Renders Codex CLI's internal log output to the step log
 * when the Codex CLI execution step failed.
 *
 * Codex CLI ships its own tracing/diagnostic output (controlled by RUST_LOG) to
 * files under $CODEX_HOME/logs rather than to stdout/stderr, so a bare non-zero
 * exit code with no console output can still have a diagnosable error recorded
 * there. This script locates the most recently modified `*.log` file under
 * $CODEX_HOME/logs and renders it via the same `renderLogFromFile` helper used
 * for the threat-detection log, reusing its secret redaction and
 * `::stop-commands::` workflow-command-safe wrapping instead of duplicating
 * that logic in shell.
 *
 * This is a no-op when the Codex CLI execution step did not fail, or when no
 * log files are found.
 */

"use strict";

const fs = require("fs");
const path = require("path");
const { renderLogFromFile } = require("./render_detection_log.cjs");

/**
 * Finds the most recently modified `*.log` file under `dir`, recursing into
 * subdirectories.
 *
 * @param {string} dir
 * @returns {string | undefined}
 */
function findMostRecentLogFile(dir) {
  /** @type {{ path: string, mtimeMs: number }[]} */
  const logFiles = [];

  /** @param {string} current */
  function walk(current) {
    let entries;
    try {
      entries = fs.readdirSync(current, { withFileTypes: true });
    } catch {
      return;
    }
    for (const entry of entries) {
      const entryPath = path.join(current, entry.name);
      if (entry.isDirectory()) {
        walk(entryPath);
      } else if (entry.isFile() && entry.name.endsWith(".log")) {
        try {
          logFiles.push({ path: entryPath, mtimeMs: fs.statSync(entryPath).mtimeMs });
        } catch {
          // Ignore files that disappear or fail to stat between readdir and stat.
        }
      }
    }
  }

  walk(dir);
  if (logFiles.length === 0) {
    return undefined;
  }

  logFiles.sort((a, b) => b.mtimeMs - a.mtimeMs);
  return logFiles[0].path;
}

/**
 * Renders the most recently modified Codex internal log file to the step log
 * when the Codex CLI execution step failed.
 *
 * @returns {Promise<void>}
 */
async function main() {
  const outcome = process.env.GH_AW_AGENTIC_EXECUTION_OUTCOME;
  if (outcome !== "failure") {
    core.info("Codex CLI execution outcome was '" + outcome + "', skipping internal log render");
    return;
  }

  const codexHome = process.env.CODEX_HOME;
  if (!codexHome) {
    core.info("CODEX_HOME not set, skipping Codex internal log render");
    return;
  }

  const logsDir = path.join(codexHome, "logs");
  const logFile = findMostRecentLogFile(logsDir);
  if (!logFile) {
    core.info("No Codex internal log files found under " + logsDir);
    return;
  }

  await renderLogFromFile(logFile, "Codex internal logs (" + logsDir + ")");
}

module.exports = { main, findMostRecentLogFile };
