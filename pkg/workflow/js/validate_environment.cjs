// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Validate Environment Variables Module
 *
 * This module provides utilities for validating required environment variables
 * in safe output job scripts. It ensures that necessary variables are present
 * before execution, providing clear error messages when variables are missing.
 *
 * Usage:
 *   const { validateEnvironment } = require("./validate_environment.cjs");
 *   validateEnvironment(['GH_AW_WORKFLOW_ID', 'GITHUB_TOKEN']);
 */

/**
 * Validate that required environment variables are present
 *
 * Checks for the presence of required environment variables and throws a
 * descriptive error if any are missing. This provides clear feedback to users
 * about what configuration is needed.
 *
 * @param {string[]} required - Array of required environment variable names
 * @throws {Error} If any required environment variables are missing
 *
 * @example
 * // Check for a single required variable
 * validateEnvironment(['GH_AW_WORKFLOW_ID']);
 *
 * @example
 * // Check for multiple required variables
 * validateEnvironment(['GH_AW_WORKFLOW_ID', 'GH_AW_BASE_BRANCH']);
 */
function validateEnvironment(required) {
  if (!Array.isArray(required) || required.length === 0) {
    return;
  }

  const missing = required.filter(varName => {
    const value = process.env[varName];
    return value === undefined || value === null || value.trim() === "";
  });

  if (missing.length > 0) {
    const errorMessage = [
      `Missing required environment variable${missing.length > 1 ? "s" : ""}: ${missing.join(", ")}`,
      "",
      "Please ensure these are set in the safe_outputs job configuration.",
    ].join("\n");

    throw new Error(errorMessage);
  }
}

module.exports = {
  validateEnvironment,
};
