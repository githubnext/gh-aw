// @ts-check
/// <reference types="@actions/github-script" />

// Ensures global.core is available when running outside github-script context
require("./shim.cjs");
const fs = require("fs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { ERR_SYSTEM } = require("./error_codes.cjs");

/**
 * Substitutes `__KEY__` placeholders in a file with values from the substitutions map.
 * Undefined/null values are treated as empty strings.
 *
 * @param {{ file: string, substitutions: Record<string, string | null | undefined> }} params
 * @returns {Promise<string>}
 */
const substitutePlaceholders = async ({ file, substitutions }) => {
  // Validate parameters
  if (!file) {
    throw new Error("file parameter is required");
  }
  if (!substitutions || typeof substitutions !== "object") {
    throw new Error("substitutions parameter must be an object");
  }

  core.info(`[substitutePlaceholders] ${file} (${Object.keys(substitutions).length} substitution(s))`);

  // Read the file
  let content;
  try {
    content = fs.readFileSync(file, "utf8");
  } catch (error) {
    const errorMessage = getErrorMessage(error);
    throw new Error(`${ERR_SYSTEM}: Failed to read file ${file}: ${errorMessage}`);
  }

  // Perform substitutions
  for (const [key, value] of Object.entries(substitutions)) {
    const placeholder = `__${key}__`;
    // Convert undefined/null to empty string to avoid leaving "undefined" or "null" in the output
    const safeValue = value == null ? "" : value;
    content = content.split(placeholder).join(safeValue);
  }

  // Write back to the file
  try {
    fs.writeFileSync(file, content, "utf8");
  } catch (error) {
    const errorMessage = getErrorMessage(error);
    throw new Error(`${ERR_SYSTEM}: Failed to write file ${file}: ${errorMessage}`);
  }

  return `Successfully substituted ${Object.keys(substitutions).length} placeholder(s) in ${file}`;
};

module.exports = substitutePlaceholders;
