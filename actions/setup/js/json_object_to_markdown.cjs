// @ts-check

/**
 * JSON Object to Markdown Converter
 *
 * Converts a plain JavaScript object to a Markdown bullet list.
 * Handles nested objects (with indentation), arrays, and primitive values.
 */

/**
 * Humanify a JSON key by replacing underscores and hyphens with spaces.
 * e.g. "engine_id" → "engine id", "awf-version" → "awf version"
 * @param {string} key - The raw object key
 * @returns {string} - Human-readable key
 */
function humanifyKey(key) {
  return key.replace(/[_-]/g, " ");
}

/**
 * Format a single value as a readable string for Markdown output.
 * @param {unknown} value - The value to format
 * @returns {string} - String representation of the value
 */
function formatValue(value) {
  if (value === null || value === undefined || value === "") {
    return "(none)";
  }
  if (Array.isArray(value)) {
    return value.length === 0 ? "(none)" : "";
  }
  if (typeof value === "object") {
    return "";
  }
  return String(value);
}

/**
 * Convert a plain JavaScript object to Markdown bullet points.
 * Nested objects and arrays are rendered as indented sub-lists.
 *
 * @param {Record<string, unknown>} obj - The object to render
 * @param {number} [depth=0] - Current indentation depth
 * @returns {string} - Markdown bullet list string
 */
function jsonObjectToMarkdown(obj, depth = 0) {
  if (!obj || typeof obj !== "object" || Array.isArray(obj)) {
    return "";
  }

  const indent = "  ".repeat(depth);
  const lines = [];

  for (const [key, value] of Object.entries(obj)) {
    const label = humanifyKey(key);
    if (Array.isArray(value)) {
      if (value.length === 0) {
        lines.push(`${indent}- **${label}**: (none)`);
      } else {
        lines.push(`${indent}- **${label}**:`);
        for (const item of value) {
          if (typeof item === "object" && item !== null) {
            lines.push(jsonObjectToMarkdown(/** @type {Record<string, unknown>} */ item, depth + 1));
          } else {
            lines.push(`${"  ".repeat(depth + 1)}- ${String(item)}`);
          }
        }
      }
    } else if (typeof value === "object" && value !== null) {
      lines.push(`${indent}- **${label}**:`);
      lines.push(jsonObjectToMarkdown(/** @type {Record<string, unknown>} */ value, depth + 1));
    } else {
      const formatted = formatValue(value);
      lines.push(`${indent}- **${label}**: ${formatted}`);
    }
  }

  return lines.join("\n");
}

module.exports = { humanifyKey, jsonObjectToMarkdown };
