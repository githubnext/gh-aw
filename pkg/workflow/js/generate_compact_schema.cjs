// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Generates a compact schema description from JSON content
 * @param {string} content - The JSON content to analyze
 * @returns {string} Compact schema description for jq/agent
 */
function generateCompactSchema(content) {
  try {
    const parsed = JSON.parse(content);

    // Generate a compact schema based on the structure
    if (Array.isArray(parsed)) {
      if (parsed.length === 0) {
        return "[]";
      }
      // For arrays, describe the first element's structure
      const firstItem = parsed[0];
      if (typeof firstItem === "object" && firstItem !== null) {
        const keys = Object.keys(firstItem);
        return `[{${keys.join(", ")}}] (${parsed.length} items)`;
      }
      return `[${typeof firstItem}] (${parsed.length} items)`;
    } else if (typeof parsed === "object" && parsed !== null) {
      // For objects, list top-level keys
      const keys = Object.keys(parsed);
      if (keys.length > 10) {
        return `{${keys.slice(0, 10).join(", ")}, ...} (${keys.length} keys)`;
      }
      return `{${keys.join(", ")}}`;
    }

    return `${typeof parsed}`;
  } catch {
    // If not valid JSON, return generic description
    return "text content";
  }
}

module.exports = {
  generateCompactSchema,
};
