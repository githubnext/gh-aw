// @ts-check
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Validate that a target branch is safe for agent pushes.
 * Blocks pushes to default/protected branches and fails closed on unexpected
 * branch protection lookup errors.
 *
 * @param {import("@actions/github-script").AsyncFunctionArguments["github"]} githubClient
 * @param {string} owner
 * @param {string} repo
 * @param {string} branchName
 * @param {boolean} checkBranchProtection
 * @returns {Promise<{ success: true } | { success: false, error: string }>}
 */
async function validatePushBranchSafety(githubClient, owner, repo, branchName, checkBranchProtection) {
  // Check whether the branch is the repository default branch
  let defaultBranch = null;
  try {
    const { data: repoData } = await githubClient.rest.repos.get({
      owner,
      repo,
    });
    defaultBranch = repoData.default_branch;
  } catch (repoError) {
    core.warning(`Could not check repository default branch: ${getErrorMessage(repoError)}`);
  }

  if (defaultBranch && branchName === defaultBranch) {
    const msg = `Cannot push to branch "${branchName}": this is the repository's default branch. Agents must not push directly to the default branch.`;
    core.error(msg);
    return { success: false, error: msg };
  }

  if (checkBranchProtection) {
    // Check whether the branch has protection rules
    let isBranchProtected = false;
    try {
      await githubClient.rest.repos.getBranchProtection({
        owner,
        repo,
        branch: branchName,
      });
      // Successful response means branch protection rules exist
      isBranchProtected = true;
    } catch (protectionError) {
      const protectionStatus = protectionError && typeof protectionError === "object" && "status" in protectionError ? protectionError.status : undefined;
      if (protectionStatus === 404) {
        // 404 means no protection rules – safe to proceed
        core.info(`Branch "${branchName}" has no protection rules`);
      } else if (protectionStatus === 403) {
        // 403 means the token lacks permission to read branch protection rules.
        // The GitHub platform will still enforce branch protection at push time,
        // so warn and allow the push to proceed.
        core.warning(`Could not check branch protection rules for "${branchName}" (insufficient permissions): ${getErrorMessage(protectionError)}`);
      } else {
        // Unexpected errors (5xx, network failures, etc.) – fail closed to
        // avoid bypassing branch protection due to transient API issues.
        const msg = `Cannot verify branch protection rules for "${branchName}": ${getErrorMessage(protectionError)}. Push blocked to prevent accidental writes to protected branches.`;
        core.error(msg);
        return { success: false, error: msg };
      }
    }

    if (isBranchProtected) {
      const msg = `Cannot push to branch "${branchName}": this branch has protection rules. Agents must not push directly to protected branches.`;
      core.error(msg);
      return { success: false, error: msg };
    }
  } else {
    core.info(`Branch protection pre-flight check disabled for "${branchName}" by config (check-branch-protection: false)`);
  }

  return { success: true };
}

module.exports = { validatePushBranchSafety };
