// @ts-check
/// <reference types="@actions/github-script" />

const crypto = require("crypto");

/**
 * Regex pattern for matching temporary ID references in text
 * Format: #aw_XXXXXXXXXXXX (aw_ prefix + 12 hex characters)
 */
const TEMPORARY_ID_PATTERN = /#(aw_[0-9a-f]{12})/gi;

/**
 * Generate a temporary ID with aw_ prefix for temporary issue IDs
 * @returns {string} A temporary ID in format aw_XXXXXXXXXXXX (12 hex characters)
 */
function generateTemporaryId() {
  return "aw_" + crypto.randomBytes(6).toString("hex");
}

/**
 * Check if a value is a valid temporary ID (aw_ prefix + 12-character hex string)
 * @param {any} value - The value to check
 * @returns {boolean} True if the value is a valid temporary ID
 */
function isTemporaryId(value) {
  if (typeof value === "string") {
    return /^aw_[0-9a-f]{12}$/i.test(value);
  }
  return false;
}

/**
 * Normalize a temporary ID to lowercase for consistent map lookups
 * @param {string} tempId - The temporary ID to normalize
 * @returns {string} Lowercase temporary ID
 */
function normalizeTemporaryId(tempId) {
  return String(tempId).toLowerCase();
}

/**
 * Replace temporary ID references in text with actual issue numbers
 * Format: #aw_XXXXXXXXXXXX -> #123
 * @param {string} text - The text to process
 * @param {Map<string, number>} tempIdMap - Map of temporary_id to issue number
 * @returns {string} Text with temporary IDs replaced with issue numbers
 */
function replaceTemporaryIdReferences(text, tempIdMap) {
  return text.replace(TEMPORARY_ID_PATTERN, (match, tempId) => {
    const issueNumber = tempIdMap.get(normalizeTemporaryId(tempId));
    if (issueNumber !== undefined) {
      return `#${issueNumber}`;
    }
    // Return original if not found (it may be created later)
    return match;
  });
}

/**
 * Load the temporary ID map from environment variable
 * @returns {Map<string, number>} Map of temporary_id to issue number
 */
function loadTemporaryIdMap() {
  const mapJson = process.env.GH_AW_TEMPORARY_ID_MAP;
  if (!mapJson || mapJson === "{}") {
    return new Map();
  }
  try {
    const mapObject = JSON.parse(mapJson);
    return new Map(Object.entries(mapObject).map(([k, v]) => [normalizeTemporaryId(k), Number(v)]));
  } catch (error) {
    if (typeof core !== "undefined") {
      core.warning(`Failed to parse temporary ID map: ${error instanceof Error ? error.message : String(error)}`);
    }
    return new Map();
  }
}

module.exports = {
  TEMPORARY_ID_PATTERN,
  generateTemporaryId,
  isTemporaryId,
  normalizeTemporaryId,
  replaceTemporaryIdReferences,
  loadTemporaryIdMap,
};
