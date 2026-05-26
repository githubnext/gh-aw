/**
 * @fileoverview Shared helpers for commit SHA normalization.
 */

const GIT_COMMIT_SHA_PATTERN = /^[0-9a-fA-F]{7,40}$/;

/**
 * Normalize and validate a git commit SHA.
 *
 * @param {unknown} value
 * @returns {string}
 */
function normalizeCommitSHA(value) {
  const normalized = String(value ?? "").trim();
  return GIT_COMMIT_SHA_PATTERN.test(normalized) ? normalized : "";
}

module.exports = {
  normalizeCommitSHA,
};
