// @ts-check

/**
 * Safe Inputs Validation Helpers
 *
 * This module provides validation utilities for safe-inputs MCP server.
 */

/**
 * Validate required fields in tool arguments
 * @param {Object} args - The arguments object to validate
 * @param {Object} inputSchema - The input schema containing required fields
 * @returns {string[]} Array of missing field names (empty if all required fields are present)
 */
function validateRequiredFields(args, inputSchema) {
  const requiredFields = inputSchema && Array.isArray(inputSchema.required) ? inputSchema.required : [];

  if (!requiredFields.length) {
    return [];
  }

  const missing = requiredFields.filter(f => {
    const value = args[f];
    return value === undefined || value === null || (typeof value === "string" && value.trim() === "");
  });

  return missing;
}

module.exports = {
  validateRequiredFields,
};
