// @ts-check

/**
 * Standardized error codes for safe-outputs handlers.
 *
 * These codes provide machine-readable prefixes for error messages,
 * enabling structured logging, monitoring dashboards, and alerting rules.
 *
 * Usage:
 *   const { ERR_VALIDATION } = require("./error_codes.cjs");
 *   throw new Error(`${ERR_VALIDATION}: Missing required field: title`);
 *   core.setFailed(`${ERR_CONFIG}: GH_AW_PROMPT environment variable is not set`);
 *
 * Error code categories:
 *   ERR_VALIDATION  - Input validation failures (missing fields, invalid format, limits exceeded)
 *   ERR_PERMISSION  - Authorization and permission check failures
 *   ERR_API         - GitHub API call failures
 *   ERR_CONFIG      - Configuration errors (missing env vars, bad setup)
 *   ERR_NOT_FOUND   - Resource not found (issues, discussions, PRs)
 *   ERR_PARSE       - Parsing failures (JSON, NDJSON, log formats)
 *   ERR_SYSTEM      - System and I/O errors (file access, git operations)
 */

/** @type {string} Input validation failures */
const ERR_VALIDATION = "ERR_VALIDATION";

/** @type {string} Authorization and permission check failures */
const ERR_PERMISSION = "ERR_PERMISSION";

/** @type {string} GitHub API call failures */
const ERR_API = "ERR_API";

/** @type {string} Configuration errors (missing env vars, bad setup) */
const ERR_CONFIG = "ERR_CONFIG";

/** @type {string} Resource not found */
const ERR_NOT_FOUND = "ERR_NOT_FOUND";

/** @type {string} Parsing failures (JSON, NDJSON, log formats) */
const ERR_PARSE = "ERR_PARSE";

/** @type {string} System and I/O errors */
const ERR_SYSTEM = "ERR_SYSTEM";

module.exports = {
  ERR_VALIDATION,
  ERR_PERMISSION,
  ERR_API,
  ERR_CONFIG,
  ERR_NOT_FOUND,
  ERR_PARSE,
  ERR_SYSTEM,
};
