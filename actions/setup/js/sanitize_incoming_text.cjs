// @ts-check
/**
 * Slimmed-down sanitization for incoming text (compute_text)
 * This version does NOT include mention filtering - all @mentions are escaped
 */

const { sanitizeContentCore, writeRedactedDomainsLog } = require("./sanitize_content_core.cjs");

/**
 * Sanitizes incoming text content without selective mention filtering
 * All @mentions are escaped to prevent unintended notifications
 *
 * Uses the core sanitization functions directly to minimize bundle size.
 *
 * @param {string} content - The content to sanitize
 * @param {number} [maxLength] - Maximum length of content (default: 524288)
 * @returns {string} The sanitized content with all mentions escaped
 */
function sanitizeIncomingText(content, maxLength) {
  // Call core sanitization which neutralizes all mentions
  return sanitizeContentCore(content, maxLength);
}

module.exports = {
  sanitizeIncomingText,
  writeRedactedDomainsLog,
};
