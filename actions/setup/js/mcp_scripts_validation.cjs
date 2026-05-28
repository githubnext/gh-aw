// @ts-check

/**
 * MCP Scripts Validation Helpers
 *
 * This module provides validation utilities for mcp-scripts MCP server.
 */

/**
 * Maximum allowed byte length for any single string-typed input parameter (SM-IS-01).
 * 10 KB = 10 * 1024 bytes.
 */
const MAX_STRING_INPUT_BYTES = 10 * 1024;

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

/**
 * Validate that no string-typed input parameter exceeds the maximum allowed byte length (SM-IS-01).
 * Implementations MUST enforce a maximum input string length of at least 10KB for each
 * string-typed input parameter. Inputs exceeding the configured maximum MUST be rejected with a
 * validation error before the tool script is invoked. Implementations MUST NOT silently truncate
 * oversized inputs.
 *
 * Scope: validates only top-level (direct) properties of the schema where `type === "string"`.
 * Nested object/array schemas are not recursively validated, consistent with the SM-IS-01
 * requirement that applies to "input parameters" (top-level tool arguments).
 *
 * @param {Object} args - The arguments object to validate
 * @param {Object} inputSchema - The input schema describing property types
 * @param {number} [maxBytes] - Maximum allowed bytes per string (defaults to MAX_STRING_INPUT_BYTES)
 * @returns {{ field: string, byteLength: number }[]} Array of violations (empty if all within limit)
 */
function validateStringInputLengths(args, inputSchema, maxBytes) {
  const limit = typeof maxBytes === "number" ? maxBytes : MAX_STRING_INPUT_BYTES;
  const properties = inputSchema && inputSchema.properties ? inputSchema.properties : {};
  const violations = [];

  for (const [field, schema] of Object.entries(properties)) {
    if (schema && schema.type === "string") {
      // Skip fields with an explicit maxLength — handler-level validation enforces their limit.
      if (typeof schema.maxLength === "number") {
        continue;
      }
      const value = args[field];
      if (typeof value === "string") {
        const byteLength = Buffer.byteLength(value, "utf8");
        if (byteLength > limit) {
          violations.push({ field, byteLength });
        }
      }
    }
  }

  return violations;
}

module.exports = {
  validateRequiredFields,
  validateStringInputLengths,
  MAX_STRING_INPUT_BYTES,
};
