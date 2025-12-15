// @ts-check
/**
 * Slimmed-down sanitization for incoming text (compute_text)
 * This version does NOT include mention filtering - all @mentions are escaped
 */

const { sanitizeContent: fullSanitizeContent, writeRedactedDomainsLog } = require("./sanitize_content.cjs");

/**
 * Sanitizes incoming text content without selective mention filtering
 * All @mentions are escaped to prevent unintended notifications
 * 
 * This is a wrapper around the full sanitizeContent that explicitly
 * does NOT pass allowedAliases, ensuring all mentions are neutralized.
 * 
 * @param {string} content - The content to sanitize
 * @param {number} [maxLength] - Maximum length of content (default: 524288)
 * @returns {string} The sanitized content with all mentions escaped
 */
function sanitizeIncomingText(content, maxLength) {
  // Call sanitizeContent without allowedAliases option
  // This ensures all @mentions are neutralized
  return fullSanitizeContent(content, maxLength);
}

module.exports = {
  sanitizeIncomingText,
  writeRedactedDomainsLog,
};
