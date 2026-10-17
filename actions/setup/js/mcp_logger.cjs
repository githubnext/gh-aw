// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * MCP Logger Utility
 *
 * This module provides logger creation utilities for MCP servers.
 * It creates logger objects with debug and debugError methods that write
 * timestamped messages to stderr.
 * Debug messages use the server name as their DEBUG namespace. Diagnostics and
 * error summaries remain visible independently of DEBUG.
 *
 * Usage:
 *   const { createLogger } = require("./mcp_logger.cjs");
 *   const logger = createLogger("my-server");
 *   logger.debug("Server started");
 *   logger.debugError("Error: ", new Error("Something went wrong"));
 */

/**
 * Match comma- or whitespace-separated DEBUG patterns, with exclusions winning.
 * @param {string} namespace
 * @returns {boolean}
 */
function debugEnabled(namespace) {
  let enabled = false;
  for (const token of (process.env.DEBUG || "").split(/[\s,]+/).filter(Boolean)) {
    const excluded = token.startsWith("-");
    const pattern = excluded ? token.slice(1) : token;
    const expression = pattern
      .split("*")
      .map(part => part.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"))
      .join(".*");
    if (!new RegExp("^" + expression + "$").test(namespace)) continue;
    if (excluded) return false;
    enabled = true;
  }
  return enabled;
}

/**
 * Create a logger with filtered debug output and unconditional diagnostics
 * @param {string} serverName - Name to include in log messages
 * @param {(message: string) => string} [redactLogMessage] - Optional diagnostic payload redactor
 * @returns {Object} Logger object with debug, diagnostic and debugError methods
 */
function createLogger(serverName, redactLogMessage) {
  const logger = {
    /**
     * Log a debug message to stderr with timestamp
     * @param {string} msg - Message to log
     */
    debug: msg => {
      if (!debugEnabled(serverName)) return;
      logger.diagnostic(msg);
    },

    /**
     * Log a required diagnostic regardless of DEBUG.
     * @param {string} msg - Message to log
     */
    diagnostic: msg => {
      const timestamp = new Date().toISOString();
      const safeMessage = redactLogMessage ? redactLogMessage(msg) : msg;
      process.stderr.write(`[${timestamp}] [${serverName}] ${safeMessage}\n`);
    },

    /**
     * Log an error summary unconditionally and its stack trace when DEBUG matches
     * @param {string} prefix - Prefix for the error message
     * @param {Error|string|any} error - Error object or message
     */
    debugError: (prefix, error) => {
      const errorMessage = getErrorMessage(error);
      logger.diagnostic(`${prefix}${errorMessage}`);
      if (error instanceof Error && error.stack) {
        logger.debug(`${prefix}Stack trace: ${error.stack}`);
      }
    },
  };

  return logger;
}

module.exports = {
  createLogger,
  debugEnabled,
};
