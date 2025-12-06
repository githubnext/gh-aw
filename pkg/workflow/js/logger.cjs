// @ts-check

/**
 * Logger utilities for MCP servers
 *
 * This module provides simple logger creation functions for use in MCP servers
 * and related modules. It supports timestamped messages and error logging.
 */

/**
 * @typedef {Object} Logger
 * @property {Function} debug - Debug logging function
 * @property {Function} [debugError] - Debug logging function for errors
 */

/**
 * Create a simple logger with timestamp and name prefix
 * @param {string} name - Name to use in log prefix (e.g., "safe-inputs-startup")
 * @returns {Logger} Logger instance with debug and debugError methods
 */
function createLogger(name) {
  const logger = {
    debug: msg => {
      const timestamp = new Date().toISOString();
      process.stderr.write(`[${timestamp}] [${name}] ${msg}\n`);
    },
    debugError: (prefix, error) => {
      const errorMessage = error instanceof Error ? error.message : String(error);
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
