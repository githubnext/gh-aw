// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Compute net UTF-8 bytes added by a unified diff.
 *
 * "Net added bytes" = (bytes in added lines) − (bytes in deleted lines), clamped to zero.
 *
 * Using the net value instead of raw additions means that files which are
 * completely rewritten with similar-sized content (e.g. a JSON object whose
 * keys are regenerated) contribute only their actual growth to the patch-size
 * budget rather than their entire new content.  This avoids the false positive
 * where the tool reports "entire source code size" for a rewrite that barely
 * changes the logical payload.
 *
 * Rules:
 *   - Only lines inside diff hunks (after the first "@@" line) are examined.
 *   - Lines that start with "+++" (file header) are excluded because they appear
 *     before the first "@@" and are never inside a hunk.
 *   - A new "diff " line resets the hunk state for the next file.
 *
 * @param {string} patchContent - Output of `git diff` (unified diff format)
 * @returns {number} Net added bytes (≥ 0)
 */
function getPatchDiffSizeBytes(patchContent) {
  let inHunk = false;
  let additions = 0;
  let deletions = 0;
  for (const line of patchContent.split("\n")) {
    if (line.startsWith("@@")) {
      inHunk = true;
    } else if (line.startsWith("diff ")) {
      inHunk = false;
    }
    if (inHunk) {
      if (line.startsWith("+")) {
        additions += Buffer.byteLength(line + "\n", "utf8");
      } else if (line.startsWith("-")) {
        deletions += Buffer.byteLength(line + "\n", "utf8");
      }
    }
  }
  return Math.max(0, additions - deletions);
}

/**
 * Compute the net patch diff size for staged changes (git diff --cached).
 *
 * Returns the net bytes added by the staged diff: additions minus deletions,
 * clamped to zero.  This is the "diff size" used to enforce max-patch-size on
 * repo-memory pushes.
 *
 * @param {Object} options
 * @param {(args: string[], opts?: Record<string, any>) => string} options.execGitSyncFn
 * @param {string} [options.cwd]
 * @returns {number}
 */
function getStagedPatchDiffSizeBytes({ execGitSyncFn, cwd }) {
  const patchContent = execGitSyncFn(["diff", "--cached"], { stdio: "pipe", cwd });
  return getPatchDiffSizeBytes(patchContent);
}

module.exports = {
  getPatchDiffSizeBytes,
  getStagedPatchDiffSizeBytes,
};
