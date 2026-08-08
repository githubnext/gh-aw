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

/**
 * Extract the trusted base commit embedded in the generated patch artifact.
 *
 * @param {unknown} patchContent
 * @returns {string}
 */
function extractPatchBaseCommit(patchContent) {
  if (typeof patchContent !== "string") {
    return "";
  }
  const headerBlock = patchContent.split(/\r?\n\r?\n/, 1)[0];
  const match = headerBlock.match(/^X-GH-AW-Base-Commit:\s*([0-9a-fA-F]{7,40})\s*$/m);
  return normalizeCommitSHA(match?.[1]);
}

module.exports = {
  extractPatchBaseCommit,
  normalizeCommitSHA,
};
