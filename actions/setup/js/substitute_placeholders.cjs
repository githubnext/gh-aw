// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { ERR_SYSTEM } = require("./error_codes.cjs");

/** @param {string} msg */
const log = msg => {
  if (typeof core !== "undefined") core.info(msg);
};

/**
 * Substitutes `__KEY__` placeholders in a file with values from the substitutions map.
 * Undefined/null values are treated as empty strings.
 *
 * @param {{ file: string, substitutions: Record<string, string | null | undefined> }} params
 * @returns {Promise<string>}
 */
const substitutePlaceholders = async ({ file, substitutions }) => {
  log("========================================");
  log("[substitutePlaceholders] Starting placeholder substitution");
  log("========================================");

  // Validate parameters
  if (!file) {
    const error = new Error("file parameter is required");
    log(`[substitutePlaceholders] ERROR: ${error.message}`);
    throw error;
  }
  if (!substitutions || typeof substitutions !== "object") {
    const error = new Error("substitutions parameter must be an object");
    log(`[substitutePlaceholders] ERROR: ${error.message}`);
    throw error;
  }

  log(`[substitutePlaceholders] File: ${file}`);
  log(`[substitutePlaceholders] Substitution count: ${Object.keys(substitutions).length}`);

  // Read the file
  let content;
  try {
    log(`[substitutePlaceholders] Reading file...`);
    content = fs.readFileSync(file, "utf8");
    log(`[substitutePlaceholders] File read successfully`);
    log(`[substitutePlaceholders] Original content length: ${content.length} characters`);
    log(`[substitutePlaceholders] First 200 characters: ${content.substring(0, 200).replace(/\n/g, "\\n")}`);
  } catch (error) {
    const errorMessage = getErrorMessage(error);
    log(`[substitutePlaceholders] ERROR reading file: ${errorMessage}`);
    throw new Error(`${ERR_SYSTEM}: Failed to read file ${file}: ${errorMessage}`);
  }

  // Perform substitutions
  log("\n========================================");
  log("[substitutePlaceholders] Processing Substitutions");
  log("========================================");

  let totalReplacements = 0;
  const beforeLength = content.length;

  for (const [key, value] of Object.entries(substitutions)) {
    const placeholder = `__${key}__`;
    // Convert undefined/null to empty string to avoid leaving "undefined" or "null" in the output
    const safeValue = value == null ? "" : value;

    // Count occurrences before replacement
    const occurrences = (content.match(new RegExp(placeholder.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"), "g")) || []).length;

    if (occurrences > 0) {
      log(`[substitutePlaceholders] Replacing ${placeholder} (${occurrences} occurrence(s))`);
      log(`[substitutePlaceholders]   Value: ${safeValue.substring(0, 100)}${safeValue.length > 100 ? "..." : ""}`);
    } else {
      log(`[substitutePlaceholders] Placeholder ${placeholder} not found in content (unused)`);
    }

    content = content.split(placeholder).join(safeValue);
    totalReplacements += occurrences;
  }

  const afterLength = content.length;
  log(`[substitutePlaceholders] Substitution complete: ${totalReplacements} total replacement(s)`);
  log(`[substitutePlaceholders] Content length change: ${beforeLength} -> ${afterLength} (${afterLength > beforeLength ? "+" : ""}${afterLength - beforeLength})`);

  // Write back to the file
  try {
    log("\n========================================");
    log("[substitutePlaceholders] Writing Output");
    log("========================================");
    log(`[substitutePlaceholders] Writing processed content back to: ${file}`);
    fs.writeFileSync(file, content, "utf8");
    log(`[substitutePlaceholders] File written successfully`);
    log(`[substitutePlaceholders] Last 200 characters: ${content.substring(Math.max(0, content.length - 200)).replace(/\n/g, "\\n")}`);
    log("========================================");
    log("[substitutePlaceholders] Processing complete - SUCCESS");
    log("========================================");
  } catch (error) {
    const errorMessage = getErrorMessage(error);
    log(`[substitutePlaceholders] ERROR writing file: ${errorMessage}`);
    throw new Error(`${ERR_SYSTEM}: Failed to write file ${file}: ${errorMessage}`);
  }

  return `Successfully substituted ${Object.keys(substitutions).length} placeholder(s) in ${file}`;
};

module.exports = substitutePlaceholders;
