// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Utilities shared by git patch generation and validation code.
 *
 * This module intentionally has no side effects and no coupling to the
 * patch-generation orchestration in generate_git_patch.cjs. Each helper is
 * pure/stateless or performs a well-defined local filesystem operation, which
 * keeps the surface small, easy to test against a real git repo, and reusable
 * by other safe-output handlers (e.g. bundle transport, create_pull_request
 * fallback paths).
 */

const fs = require("fs");

const { getErrorMessage } = require("./error_helpers.cjs");
const { execGitSync } = require("./git_helpers.cjs");

/**
 * Debug logging helper - logs to stderr when DEBUG env var matches
 * @param {string} message - Debug message to log
 */
function debugLog(message) {
  const debug = process.env.DEBUG || "";
  if (debug === "*" || debug.includes("generate_git_patch") || debug.includes("patch")) {
    console.error(`[git_patch_utils] ${message}`);
  }
}

/**
 * Sanitize a string for use as a patch filename component.
 * Replaces path separators and special characters with dashes.
 * @param {string} value - The value to sanitize
 * @param {string} fallback - Fallback value when input is empty or nullish
 * @returns {string} The sanitized string safe for use in a filename
 */
function sanitizeForFilename(value, fallback) {
  if (!value) return fallback;
  return value
    .replace(/[/\\:*?"<>|]/g, "-")
    .replace(/-{2,}/g, "-")
    .replace(/^-|-$/g, "")
    .toLowerCase();
}

/**
 * Sanitize a branch name for use as a patch filename
 * @param {string} branchName - The branch name to sanitize
 * @returns {string} The sanitized branch name safe for use in a filename
 */
function sanitizeBranchNameForPatch(branchName) {
  return sanitizeForFilename(branchName, "unknown");
}

/**
 * Sanitize a repo slug for use in a filename
 * @param {string} repoSlug - The repo slug (owner/repo)
 * @returns {string} The sanitized slug safe for use in a filename
 */
function sanitizeRepoSlugForPatch(repoSlug) {
  return sanitizeForFilename(repoSlug, "");
}

/**
 * Get the patch file path for a given branch name
 * @param {string} branchName - The branch name
 * @returns {string} The full patch file path
 */
function getPatchPath(branchName) {
  const sanitized = sanitizeBranchNameForPatch(branchName);
  return `/tmp/gh-aw/aw-${sanitized}.patch`;
}

/**
 * Get the patch file path for a given branch name and repo slug
 * Used for multi-repo scenarios to prevent patch file collisions
 * @param {string} branchName - The branch name
 * @param {string} repoSlug - The repository slug (owner/repo)
 * @returns {string} The full patch file path including repo disambiguation
 */
function getPatchPathForRepo(branchName, repoSlug) {
  const sanitizedBranch = sanitizeBranchNameForPatch(branchName);
  const sanitizedRepo = sanitizeRepoSlugForPatch(repoSlug);
  return `/tmp/gh-aw/aw-${sanitizedRepo}-${sanitizedBranch}.patch`;
}

/**
 * Builds the pathspec arguments to exclude specific files from a git command.
 * Produces ["--", ":(exclude)pattern1", ":(exclude)pattern2", ...] or [] when
 * the input is empty/unset. These are passed after a "--" so git treats them
 * as pathspecs, not revisions.
 *
 * @param {string[] | undefined | null} excludedFiles - Glob patterns to exclude
 * @returns {string[]} Arguments to append to a git format-patch or git diff call
 */
function buildExcludePathspecs(excludedFiles) {
  if (!Array.isArray(excludedFiles) || excludedFiles.length === 0) {
    return [];
  }
  return ["--", ...excludedFiles.map(p => `:(exclude)${p}`)];
}

/**
 * Compute the net diff size in bytes between two refs in the given git repo.
 *
 * This is the value that should be compared against `max_patch_size` in
 * push_to_pull_request_branch: it reflects how much the PR branch will
 * actually change as a result of the push, independent of how the patch or
 * bundle transport encodes the commit history.
 *
 * Implementation note: we use `git diff --binary --output=<tmpfile>` rather
 * than buffering the diff through execGitSync's stdout. That keeps memory
 * usage O(1) regardless of the diff size (we just stat the file) and avoids
 * hitting the execGitSync maxBuffer on large binary diffs. The temp file is
 * removed in `finally` on success, failure, and stat failure alike.
 *
 * @param {Object} args - Arguments
 * @param {string} args.baseRef - Base ref (commit SHA, branch, or ref)
 * @param {string} args.headRef - Head ref (commit SHA, branch, or ref)
 * @param {string} args.cwd - Working directory containing the git repo
 * @param {string} args.tmpPath - Absolute path to the temp diff file (will be
 *   written and removed by this function)
 * @param {string[]} [args.excludedFiles] - Glob patterns to exclude
 * @returns {number | null} The net diff size in bytes, or null on failure
 */
function computeIncrementalDiffSize({ baseRef, headRef, cwd, tmpPath, excludedFiles }) {
  if (!baseRef || !headRef || !cwd || !tmpPath) {
    return null;
  }
  const excludeArgs = buildExcludePathspecs(excludedFiles);
  let diffSize = null;
  try {
    execGitSync(["diff", "--binary", `--output=${tmpPath}`, `${baseRef}..${headRef}`, ...excludeArgs], { cwd });
    if (fs.existsSync(tmpPath)) {
      diffSize = fs.statSync(tmpPath).size;
      debugLog(`Computed incremental net diffSize=${diffSize} bytes (baseRef=${baseRef}..${headRef})`);
    }
  } catch (diffErr) {
    debugLog(`Failed to compute incremental net diffSize - ${getErrorMessage(diffErr)}`);
  } finally {
    // Best-effort cleanup of the temp diff file; we only needed its size.
    try {
      if (fs.existsSync(tmpPath)) {
        fs.unlinkSync(tmpPath);
      }
    } catch {
      // Cleanup failure is non-fatal.
    }
  }
  return diffSize;
}

module.exports = {
  sanitizeForFilename,
  sanitizeBranchNameForPatch,
  sanitizeRepoSlugForPatch,
  getPatchPath,
  getPatchPathForRepo,
  buildExcludePathspecs,
  computeIncrementalDiffSize,
};
