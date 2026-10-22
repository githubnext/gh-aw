// @ts-check
/// <reference types="@actions/github-script" />

const { lookupCheckout } = require("./checkout_manifest.cjs");
const { getBaseBranch } = require("./get_base_branch.cjs");
const { normalizeBranchName } = require("./normalize_branch_name.cjs");
const { parseAllowedBaseBranches, isBaseBranchAllowed } = require("./create_pull_request_helpers.cjs");

function validateBaseBranch(branch) {
  if (
    typeof branch !== "string" ||
    !branch ||
    branch === "HEAD" ||
    normalizeBranchName(branch) !== branch ||
    branch.startsWith("/") ||
    branch.endsWith("/") ||
    branch.endsWith(".") ||
    branch.includes("..") ||
    branch.includes("//") ||
    branch.split("/").some(part => part.startsWith(".") || part.endsWith(".lock"))
  ) {
    throw new Error(`Invalid patch base branch: ${JSON.stringify(branch)}`);
  }
  return branch;
}

async function resolvePatchBase(repoParts, config, cwd) {
  const manifest = lookupCheckout(`${repoParts.owner}/${repoParts.repo}`);
  const configured = typeof config.base_branch === "string" ? config.base_branch.trim() : "";
  const pr = typeof context !== "undefined" ? context.payload?.pull_request : undefined;
  const prRepo = pr?.base?.repo?.full_name || (typeof context !== "undefined" && context.repo ? `${context.repo.owner}/${context.repo.repo}` : "");
  // An initial checkout of this PR's head is not a selection of a new PR base.
  const prBase = manifest?.checked_out_ref && manifest.checked_out_ref === pr?.head?.ref && prRepo.toLowerCase() === `${repoParts.owner}/${repoParts.repo}`.toLowerCase() ? pr?.base?.ref : "";
  const branch = configured || prBase || manifest?.checked_out_ref || manifest?.default_branch || (await getBaseBranch(repoParts, { preferLocalDefaultBranchMetadata: Boolean(cwd), cwd }));
  return validateBaseBranch(branch);
}

// Transported metadata is agent-controlled, not an authorization grant. In the
// apply job the original resolver may be absent: allow a different resolved base
// only when the workflow's trusted allowlist authorizes it.
function validateTransportBase(branch, trustedBase, config) {
  validateBaseBranch(branch);
  if (branch !== trustedBase && !isBaseBranchAllowed(branch, parseAllowedBaseBranches(config.allowed_base_branches))) {
    throw new Error(`Patch base branch '${branch}' is not authorized. Configure safe-outputs.create-pull-request.allowed-base-branches for runtime-selected bases.`);
  }
  return branch;
}

module.exports = { resolvePatchBase, validateBaseBranch, validateTransportBase };
