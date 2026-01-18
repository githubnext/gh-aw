// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Regex pattern to match expiration marker with checked checkbox
 * Allows flexible whitespace: - [x] expires <!-- gh-aw-expires: DATE --> on ...
 * Pattern is more resilient to spacing variations
 */
const EXPIRATION_PATTERN = /^-\s*\[x\]\s+expires\s*<!--\s*gh-aw-expires:\s*([^>]+)\s*-->/m;

/**
 * Format a Date object to human-readable string in UTC
 * @param {Date} date - Date to format
 * @returns {string} Human-readable date string (e.g., "Jan 25, 2026, 1:53 PM")
 */
function formatExpirationDate(date) {
  return date.toLocaleString("en-US", {
    dateStyle: "medium",
    timeStyle: "short",
    timeZone: "UTC",
  });
}

/**
 * Create expiration marker line with checkbox, XML comment, and human-readable date
 * @param {Date} expirationDate - Date when the item expires
 * @returns {string} Formatted expiration line
 */
function createExpirationLine(expirationDate) {
  const expirationISO = expirationDate.toISOString();
  const humanReadableDate = formatExpirationDate(expirationDate);
  return `- [x] expires <!-- gh-aw-expires: ${expirationISO} --> on ${humanReadableDate} UTC`;
}

/**
 * Extract expiration date from text body
 * @param {string} body - Text body containing expiration marker
 * @returns {Date|null} Expiration date or null if not found/invalid
 */
function extractExpirationDate(body) {
  const match = body.match(EXPIRATION_PATTERN);

  if (!match) {
    return null;
  }

  const expirationISO = match[1].trim();
  const expirationDate = new Date(expirationISO);

  // Validate the date
  if (isNaN(expirationDate.getTime())) {
    return null;
  }

  return expirationDate;
}

module.exports = {
  EXPIRATION_PATTERN,
  formatExpirationDate,
  createExpirationLine,
  extractExpirationDate,
};
