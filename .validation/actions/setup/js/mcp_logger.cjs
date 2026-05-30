// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * MCP Logger Utility
 *
 * This module provides logger creation utilities for MCP servers.
 * It creates logger objects with debug and debugError methods that write
 * timestamped messages to stderr.
 *
 * Usage:
 *   const { createLogger } = require("./mcp_logger.cjs");
 *   const logger = createLogger("my-server");
 *   logger.debug("Server started");
 *   logger.debugError("Error: ", new Error("Something went wrong"));
 */

/**
 * Create a logger object with debug and debugError methods
 * @param {string} serverName - Name to include in log messages
 * @returns {Object} Logger object with debug and debugError methods
 */
function createLogger(serverName) {
  const logger = {
    /**
     * Log a debug message to stderr with timestamp
     * @param {string} msg - Message to log
     */
    debug: msg => {
      const timestamp = new Date().toISOString();
      process.stderr.write(`[${timestamp}] [${serverName}] ${msg}\n`);
    },

    /**
     * Log an error with optional stack trace
     * @param {string} prefix - Prefix for the error message
     * @param {Error|string|any} error - Error object or message
     */
    debugError: (prefix, error) => {
      const errorMessage = getErrorMessage(error);
      logger.debug(`${prefix}${errorMessage}`);
      if (error instanceof Error && error.stack) {
        logger.debug(`${prefix}Stack trace: ${error.stack}`);
      }
    },
  };

  return logger;
}

module.exports = {
  createLogger,
};
