// @ts-check
// @safe-outputs-exempt SEC-005: pure local path-derivation utility; no cross-repo API calls. Target-repo allowlist enforcement is handled upstream in API-calling handlers.

/** @type {typeof import("fs")} */
const fs = require("fs");
const { getPatchPathForBranch, getPatchPathForBranchInRepo } = require("./git_patch_utils.cjs");
const { getBundlePathForBranch, getBundlePathForBranchInRepo } = require("./generate_git_bundle.cjs");
const { TransportIntegrityError } = require("./transport_integrity.cjs");

/**
 * Derive patch and bundle file paths for a safe-output message from the
 * validated branch and qualified target repository.
 *
 * Paths are always re-derived from `branch` so that even a malicious agent
 * cannot point the privileged handler at a file outside the canonical
 * `/tmp/gh-aw/aw-<sanitized-branch>.{patch,bundle}` prefix. Sanitization in
 * `getPatchPathForBranch` / `getBundlePathForBranch` enforces that prefix.
 *
 * @param {Object} message - The safe-output message
 * @param {string} [qualifiedRepo] - Authorized, fully qualified target repo slug
 * @returns {{patchPath: string|undefined, bundlePath: string|undefined}}
 */
function resolveTransportPaths(message, qualifiedRepo) {
  const branch = message.branch;
  if (!branch) {
    return { patchPath: undefined, bundlePath: undefined };
  }
  const repoCandidates = qualifiedRepo ? [qualifiedRepo, null] : [null];
  const candidates = repoCandidates
    .map(repo => {
      const patch = repo ? getPatchPathForBranchInRepo(branch, repo) : getPatchPathForBranch(branch);
      const bundle = repo ? getBundlePathForBranchInRepo(branch, repo) : getBundlePathForBranch(branch);
      return {
        patchPath: fs.existsSync(patch) ? patch : undefined,
        bundlePath: fs.existsSync(bundle) ? bundle : undefined,
      };
    })
    .filter(candidate => candidate.patchPath || candidate.bundlePath);
  // Patch and bundle must belong to one namespace, never independently selected
  // from qualified and legacy artifacts left by separate producer invocations.
  if (candidates.length > 1) {
    throw new TransportIntegrityError(branch, undefined, "resolve", "ambiguous transport candidates");
  }
  return candidates[0] || { patchPath: undefined, bundlePath: undefined };
}

module.exports = { resolveTransportPaths };
