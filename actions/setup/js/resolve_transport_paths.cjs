// @ts-check

/** @type {typeof import("fs")} */
const fs = require("fs");
const { getPatchPathForBranch, getPatchPathForBranchInRepo } = require("./git_patch_utils.cjs");
const { getBundlePathForBranch, getBundlePathForBranchInRepo } = require("./generate_git_bundle.cjs");

/**
 * Re-derive patch and bundle file paths for a safe-output message from the
 * validated `branch` (and optional `repo`) fields.
 *
 * The MCP server sets `patch_path` and `bundle_path` on entries it appends to
 * the safe-outputs NDJSON, but the validation step in collect_ndjson_output
 * unconditionally strips these infrastructure fields from every entry as a
 * defense against agents forging file paths via raw NDJSON. By the time the
 * privileged handler runs, any `patch_path`/`bundle_path` still on the
 * message would be agent-controlled, so we ignore them and always re-derive
 * from `branch` (and `repo`).
 *
 * Sanitization in getPatchPathForBranch / getBundlePathForBranch constrains
 * the result to /tmp/gh-aw/aw-<sanitized>.{patch,bundle}, so a malicious
 * `branch` cannot escape that prefix.
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
