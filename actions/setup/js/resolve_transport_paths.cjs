// @ts-check
// @safe-outputs-exempt SEC-005: pure local path-derivation utility; no cross-repo API calls. Target-repo allowlist enforcement is handled upstream in API-calling handlers.

/** @type {typeof import("fs")} */
const fs = require("fs");
const { getPatchPathForBranch, getPatchPathForBranchInRepo } = require("./git_patch_utils.cjs");
const { getBundlePathForBranch, getBundlePathForBranchInRepo } = require("./generate_git_bundle.cjs");

/**
 * Derive patch and bundle file paths for a safe-output message from the
 * validated `branch` (and optional `repo`) fields.
 *
 * Paths are always re-derived from `branch` so that even a malicious agent
 * cannot point the privileged handler at a file outside the canonical
 * `/tmp/gh-aw/aw-<sanitized-branch>.{patch,bundle}` prefix. Sanitization in
 * `getPatchPathForBranch` / `getBundlePathForBranch` enforces that prefix.
 *
 * @param {Object} message - The safe-output message
 * @param {string} [defaultTargetRepo] - Default target repo slug used as a fallback
 *   candidate for multi-repo path computation
 * @returns {{patchPath: string|undefined, bundlePath: string|undefined}}
 */
function resolveTransportPaths(message, defaultTargetRepo) {
  const branch = message.branch;
  if (!branch) {
    return { patchPath: undefined, bundlePath: undefined };
  }
  /** @type {(string|null)[]} */
  const repoCandidates = [];
  if (message.repo) repoCandidates.push(message.repo);
  if (defaultTargetRepo && defaultTargetRepo !== message.repo) repoCandidates.push(defaultTargetRepo);
  repoCandidates.push(null);
  let patchPath;
  let bundlePath;
  for (const repo of repoCandidates) {
    const p = repo ? getPatchPathForBranchInRepo(branch, repo) : getPatchPathForBranch(branch);
    if (fs.existsSync(p)) {
      patchPath = p;
      break;
    }
  }
  for (const repo of repoCandidates) {
    const p = repo ? getBundlePathForBranchInRepo(branch, repo) : getBundlePathForBranch(branch);
    if (fs.existsSync(p)) {
      bundlePath = p;
      break;
    }
  }
  return { patchPath, bundlePath };
}

module.exports = { resolveTransportPaths };
