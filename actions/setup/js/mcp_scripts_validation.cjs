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
 * When a field declares an explicit schema maxLength, that explicit character limit is enforced here;
 * otherwise the default SM-IS-01 10KB byte limit applies.
 *
 * Scope: validates only top-level (direct) properties of the schema where `type === "string"`.
 * Nested object/array schemas are not recursively validated, consistent with the SM-IS-01
 * requirement that applies to "input parameters" (top-level tool arguments).
 *
 * @param {Object} args - The arguments object to validate
 * @param {Object} inputSchema - The input schema describing property types
 * @param {number} [maxBytes] - Maximum allowed bytes per string (defaults to MAX_STRING_INPUT_BYTES)
 * @returns {{ field: string, actualLength: number, limit: number, unit: "bytes" | "characters" }[]} Array of violations (empty if all within limit)
 */
function validateStringInputLengths(args, inputSchema, maxBytes) {
  const limit = typeof maxBytes === "number" ? maxBytes : MAX_STRING_INPUT_BYTES;
  const properties = inputSchema && inputSchema.properties ? inputSchema.properties : {};
  const violations = [];

  for (const [field, schema] of Object.entries(properties)) {
    if (schema && schema.type === "string") {
      const value = args[field];
      if (typeof value === "string") {
        if (typeof schema.maxLength === "number") {
          const characterLength = Array.from(value).length;
          if (characterLength > schema.maxLength) {
            violations.push({ field, actualLength: characterLength, limit: schema.maxLength, unit: "characters" });
          }
          continue;
        }

        const byteLength = Buffer.byteLength(value, "utf8");
        if (byteLength > limit) {
          violations.push({ field, actualLength: byteLength, limit, unit: "bytes" });
        }
      }
    }
  }

  return violations;
}

/**
 * Build actionable E006 validation message for string length violations.
 *
 * @param {string} toolName - Tool name being validated
 * @param {{ field: string, actualLength: number, limit: number, unit: "bytes" | "characters" }[]} violations - Violations returned by validateStringInputLengths
 * @returns {string} E006 message
 */
function buildStringLengthValidationError(toolName, violations) {
  const details = violations.map(v => `'${v.field}' exceeds maximum length of ${v.limit} ${v.unit} (got ${v.actualLength} ${v.unit})`).join(", ");
  return `E006: Input string parameter(s) exceed maximum length for tool '${toolName}': ${details}`;
}

/**
 * Validate that string-typed arguments meet the schema's minLength constraints.
 * Trims values before comparing (matching downstream validator behavior).
 * Only checks top-level properties with `type === "string"` and an explicit `minLength`.
 * Absent or non-string values are skipped.
 *
 * @param {Object} args - The arguments object to validate
 * @param {Object} inputSchema - The input schema describing property types and constraints
 * @returns {{ field: string, minLength: number, actualLength: number }[]} Array of violations (empty if all OK)
 */
function validateStringMinLengths(args, inputSchema) {
  const properties = inputSchema && inputSchema.properties ? inputSchema.properties : {};
  const violations = [];

  for (const [field, schema] of Object.entries(properties)) {
    if (schema && schema.type === "string" && typeof schema.minLength === "number") {
      const value = args[field];
      if (typeof value === "string") {
        const trimmedLength = value.trim().length;
        if (trimmedLength < schema.minLength) {
          violations.push({ field, minLength: schema.minLength, actualLength: trimmedLength });
        }
      }
    }
  }

  return violations;
}

module.exports = {
  validateRequiredFields,
  validateStringInputLengths,
  buildStringLengthValidationError,
  validateStringMinLengths,
  MAX_STRING_INPUT_BYTES,
};
