// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Count total UTF-8 bytes for added lines in a unified diff.
 * Added lines start with "+" and exclude file header lines ("+++").
 * Uses hunk-state tracking so content lines that begin with "++" are
 * counted correctly (they appear as "+++" inside a hunk but are not headers).
 *
 * @param {string} patchContent
 * @returns {number}
 */
function getAddedPatchSizeBytesFromDiff(patchContent) {
  let inHunk = false;
  let total = 0;
  for (const line of patchContent.split("\n")) {
    if (line.startsWith("@@")) {
      inHunk = true;
    } else if (line.startsWith("diff ")) {
      inHunk = false;
    }
    if (inHunk && line.startsWith("+")) {
      total += Buffer.byteLength(line + "\n", "utf8");
    }
  }
  return total;
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
