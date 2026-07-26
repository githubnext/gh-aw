// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Count total UTF-8 bytes for added lines in a unified diff.
 * Added lines start with "+" and exclude file header lines ("+++").
 *
 * @param {string} patchContent
 * @returns {number}
 */
function getAddedPatchSizeBytesFromDiff(patchContent) {
  return patchContent
    .split("\n")
    .filter(line => line.startsWith("+") && !line.startsWith("+++"))
    .reduce((sum, line) => sum + Buffer.byteLength(line + "\n", "utf8"), 0);
}

/**
 * Compute staged patch additions size from git diff --cached.
 *
 * @param {Object} options
 * @param {(args: string[], opts?: Record<string, any>) => string} options.execGitSyncFn
 * @param {string} [options.cwd]
 * @returns {number}
 */
function getStagedPatchAdditionsSizeBytes({ execGitSyncFn, cwd }) {
  const patchContent = execGitSyncFn(["diff", "--cached"], { stdio: "pipe", cwd });
  return getAddedPatchSizeBytesFromDiff(patchContent);
}

module.exports = {
  getAddedPatchSizeBytesFromDiff,
  getStagedPatchAdditionsSizeBytes,
};
