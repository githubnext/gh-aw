// @ts-check

/**
 * Shared safeoutputs CLI helpers for all agentic harnesses.
 *
 * Provides the low-level CLI invocation helper and the higher-level emit
 * functions so that missing_tool / report_incomplete diagnostics are emitted
 * through the supported safeoutputs CLI channel rather than via direct file
 * appends.  This survives read-only teardown states (EROFS) where direct
 * appends would silently drop structured failure signals.
 */

"use strict";

const childProcess = require("child_process");

/**
 * @typedef {(toolName: string, args: Record<string, string>) => void} RunSafeOutputsCLILike
 */

/**
 * Invoke the safeoutputs CLI with named arguments.
 * @param {string} toolName
 * @param {Record<string, string>} args
 */
function runSafeOutputsCLI(toolName, args) {
  const command = process.env.GH_AW_SAFEOUTPUTS_CLI || "safeoutputs";
  /** @type {string[]} */
  const commandArgs = [toolName];
  for (const [key, value] of Object.entries(args)) {
    if (typeof value !== "string" || value.length === 0) continue;
    if (!/^[a-zA-Z0-9_-]+$/.test(key)) {
      throw new Error(`invalid safeoutputs argument key: ${key}`);
    }
    commandArgs.push(`--${key}`);
    commandArgs.push(value);
  }
  try {
    childProcess.execFileSync(command, commandArgs, { encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] });
  } catch (error) {
    const err = /** @type {{message?: string, stderr?: string | Buffer}} */ error ?? {};
    const stderr = typeof err.stderr === "string" ? err.stderr.trim() : Buffer.isBuffer(err.stderr) ? err.stderr.toString("utf8").trim() : "";
    const message = typeof err.message === "string" ? err.message : String(error);
    const keysSummary = Object.keys(args).join(", ");
    throw new Error(stderr ? `safeoutputs ${toolName}(${keysSummary}) failed: ${message}: ${stderr}` : `safeoutputs ${toolName}(${keysSummary}) failed: ${message}`);
  }
}

/**
 * Build missing_tool alternatives text with optional denied-command diagnostics.
 * @param {string} baseAlternatives
 * @param {string[]} deniedCommands
 * @returns {string}
 */
function buildMissingToolAlternatives(baseAlternatives, deniedCommands) {
  if (!Array.isArray(deniedCommands) || deniedCommands.length === 0) {
    return baseAlternatives;
  }
  const maxLength = 512;
  if (baseAlternatives.length >= maxLength) {
    return baseAlternatives.slice(0, maxLength);
  }

  const remaining = maxLength - baseAlternatives.length;
  const prefix = " Denied commands: ";
  if (remaining <= prefix.length) {
    return baseAlternatives.slice(0, maxLength);
  }

  let suffix = prefix;
  let appendedCount = 0;
  for (const command of deniedCommands) {
    const delimiter = appendedCount === 0 ? "" : " | ";
    const candidate = `${delimiter}${command}`;
    if (suffix.length + candidate.length > remaining) {
      const remainingCount = deniedCommands.length - appendedCount;
      const more = ` | ... and ${remainingCount} more`;
      if (remainingCount > 0 && suffix.length + more.length <= remaining) {
        suffix += more;
      }
      break;
    }
    suffix += candidate;
    appendedCount += 1;
  }

  return (baseAlternatives + suffix).slice(0, maxLength);
}

/**
 * Emit a structured missing_tool signal for repeated permission-denied failures.
 * @param {{
 *   safeOutputsPath?: string,
 *   runSafeOutputsCLI?: RunSafeOutputsCLILike,
 *   logger?: (message: string) => void,
 *   deniedCommands?: string[]
 * }=} options
 */
function emitMissingToolPermissionIssue(options) {
  const safeOutputsPath = options && typeof options.safeOutputsPath === "string" ? options.safeOutputsPath : process.env.GH_AW_SAFE_OUTPUTS || "";
  const runSafeOutputs = options && options.runSafeOutputsCLI ? options.runSafeOutputsCLI : runSafeOutputsCLI;
  const logger = options && options.logger ? options.logger : defaultLog;
  const deniedCommands = options && options.deniedCommands ? options.deniedCommands : [];

  if (!safeOutputsPath) {
    logger("missing_tool skipped: GH_AW_SAFE_OUTPUTS is not set");
    return;
  }
  try {
    runSafeOutputs("missing_tool", {
      tool: "tool/permission",
      reason: "missing tool/permission issue: numerous permission denied errors detected",
      alternatives: buildMissingToolAlternatives("Verify token scopes, repository permissions, and MCP/tool access configuration.", deniedCommands),
    });
    logger(`missing_tool emitted via safeoutputs CLI: ${safeOutputsPath}`);
  } catch (error) {
    const err = /** @type {Error} */ /** @type {unknown} */ error;
    logger(`missing_tool emission failed: ${err.message}`);
  }
}

/**
 * Append a structured report_incomplete signal when infrastructure failures prevent completion.
 * This allows downstream failure handling to classify transient infrastructure errors explicitly.
 * @param {string} details
 * @param {{
 *   safeOutputsPath?: string,
 *   runSafeOutputsCLI?: RunSafeOutputsCLILike,
 *   logger?: (message: string) => void
 * }=} options
 */
function emitInfrastructureIncomplete(details, options) {
  const safeOutputsPath = options && typeof options.safeOutputsPath === "string" ? options.safeOutputsPath : process.env.GH_AW_SAFE_OUTPUTS || "";
  const runSafeOutputs = options && options.runSafeOutputsCLI ? options.runSafeOutputsCLI : runSafeOutputsCLI;
  const logger = options && options.logger ? options.logger : defaultLog;

  if (!safeOutputsPath) {
    logger("report_incomplete skipped: GH_AW_SAFE_OUTPUTS is not set");
    return;
  }
  try {
    runSafeOutputs("report_incomplete", {
      reason: "infrastructure_error",
      details,
    });
    logger(`report_incomplete emitted via safeoutputs CLI: ${safeOutputsPath}`);
  } catch (error) {
    const err = /** @type {Error} */ /** @type {unknown} */ error;
    logger(`report_incomplete emission failed: ${err.message}`);
  }
}

/**
 * Default logger that writes to stderr.
 * @param {string} message
 */
function defaultLog(message) {
  process.stderr.write(`[safeoutputs-cli] ${message}\n`);
}

if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    runSafeOutputsCLI,
    buildMissingToolAlternatives,
    emitMissingToolPermissionIssue,
    emitInfrastructureIncomplete,
  };
}
