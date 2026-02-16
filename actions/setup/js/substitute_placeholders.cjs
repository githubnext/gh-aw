const fs = require("fs");
const { getErrorMessage } = require("./error_helpers.cjs");

const substitutePlaceholders = async ({ file, substitutions }) => {
  if (typeof core !== "undefined") {
    core.info("========================================");
    core.info("[substitutePlaceholders] Starting placeholder substitution");
    core.info("========================================");
  }

  // Validate parameters
  if (!file) {
    const error = new Error("file parameter is required");
    if (typeof core !== "undefined") {
      core.info(`[substitutePlaceholders] ERROR: ${error.message}`);
    }
    throw error;
  }
  if (!substitutions || "object" != typeof substitutions) {
    const error = new Error("substitutions parameter must be an object");
    if (typeof core !== "undefined") {
      core.info(`[substitutePlaceholders] ERROR: ${error.message}`);
    }
    throw error;
  }

  if (typeof core !== "undefined") {
    core.info(`[substitutePlaceholders] File: ${file}`);
    core.info(`[substitutePlaceholders] Substitution count: ${Object.keys(substitutions).length}`);
  }

  // Read the file
  let content;
  try {
    if (typeof core !== "undefined") {
      core.info(`[substitutePlaceholders] Reading file...`);
    }
    content = fs.readFileSync(file, "utf8");
    if (typeof core !== "undefined") {
      core.info(`[substitutePlaceholders] File read successfully`);
      core.info(`[substitutePlaceholders] Original content length: ${content.length} characters`);
      core.info(`[substitutePlaceholders] First 200 characters: ${content.substring(0, 200).replace(/\n/g, "\\n")}`);
    }
  } catch (error) {
    const errorMessage = getErrorMessage(error);
    if (typeof core !== "undefined") {
      core.info(`[substitutePlaceholders] ERROR reading file: ${errorMessage}`);
    }
    throw new Error(`Failed to read file ${file}: ${errorMessage}`);
  }

  // Perform substitutions
  if (typeof core !== "undefined") {
    core.info("\n========================================");
    core.info("[substitutePlaceholders] Processing Substitutions");
    core.info("========================================");
  }

  let totalReplacements = 0;
  const beforeLength = content.length;

  for (const [key, value] of Object.entries(substitutions)) {
    const placeholder = `__${key}__`;
    // Convert undefined/null to empty string to avoid leaving "undefined" or "null" in the output
    const safeValue = value === undefined || value === null ? "" : value;

    // Count occurrences before replacement
    const occurrences = (content.match(new RegExp(placeholder.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"), "g")) || []).length;

    if (typeof core !== "undefined") {
      if (occurrences > 0) {
        core.info(`[substitutePlaceholders] Replacing ${placeholder} (${occurrences} occurrence(s))`);
        core.info(`[substitutePlaceholders]   Value: ${safeValue.substring(0, 100)}${safeValue.length > 100 ? "..." : ""}`);
      } else {
        core.info(`[substitutePlaceholders] Placeholder ${placeholder} not found in content (unused)`);
      }
    }

    content = content.split(placeholder).join(safeValue);
    totalReplacements += occurrences;
  }

  const afterLength = content.length;

  if (typeof core !== "undefined") {
    core.info(`[substitutePlaceholders] Substitution complete: ${totalReplacements} total replacement(s)`);
    core.info(`[substitutePlaceholders] Content length change: ${beforeLength} -> ${afterLength} (${afterLength > beforeLength ? "+" : ""}${afterLength - beforeLength})`);
  }

  // Write back to the file
  try {
    if (typeof core !== "undefined") {
      core.info("\n========================================");
      core.info("[substitutePlaceholders] Writing Output");
      core.info("========================================");
      core.info(`[substitutePlaceholders] Writing processed content back to: ${file}`);
    }
    fs.writeFileSync(file, content, "utf8");
    if (typeof core !== "undefined") {
      core.info(`[substitutePlaceholders] File written successfully`);
      core.info(`[substitutePlaceholders] Last 200 characters: ${content.substring(Math.max(0, content.length - 200)).replace(/\n/g, "\\n")}`);
      core.info("========================================");
      core.info("[substitutePlaceholders] Processing complete - SUCCESS");
      core.info("========================================");
    }
  } catch (error) {
    const errorMessage = getErrorMessage(error);
    if (typeof core !== "undefined") {
      core.info(`[substitutePlaceholders] ERROR writing file: ${errorMessage}`);
    }
    throw new Error(`Failed to write file ${file}: ${errorMessage}`);
  }

  return `Successfully substituted ${Object.keys(substitutions).length} placeholder(s) in ${file}`;
};
module.exports = substitutePlaceholders;
