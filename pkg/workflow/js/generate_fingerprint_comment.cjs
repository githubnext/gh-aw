// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Generate fingerprint HTML comment for tracking assets across workflows
 * @param {string} fingerprint - Fingerprint identifier (empty string if not set)
 * @returns {string} HTML comment with fingerprint or empty string
 */
function generateFingerprintComment(fingerprint) {
  if (!fingerprint) {
    return "";
  }
  return `\n\n<!-- fingerprint: ${fingerprint} -->`;
}

module.exports = {
  generateFingerprintComment,
};
