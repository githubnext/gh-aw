// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Shared utility for repository permission validation
 * Used by both check_permissions.cjs and check_membership.cjs
 */

/**
 * Parse required permissions from environment variable
 * @returns {string[]} Array of required permission levels
 */
function parseRequiredPermissions() {
  const requiredPermissionsEnv = process.env.GH_AW_REQUIRED_ROLES;
  return requiredPermissionsEnv ? requiredPermissionsEnv.split(",").filter(p => p.trim() !== "") : [];
}

/**
 * Check if user has required repository permissions
 * @param {string} actor - GitHub username to check
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {string[]} requiredPermissions - Array of required permission levels
 * @returns {Promise<{authorized: boolean, permission?: string, error?: string}>}
 */
async function checkRepositoryPermission(actor, owner, repo, requiredPermissions) {
  try {
    core.info(`Checking if user '${actor}' has required permissions for ${owner}/${repo}`);
    core.info(`Required permissions: ${requiredPermissions.join(", ")}`);

    const repoPermission = await github.rest.repos.getCollaboratorPermissionLevel({
      owner: owner,
      repo: repo,
      username: actor,
    });

    const permission = repoPermission.data.permission;
    core.info(`Repository permission level: ${permission}`);

    // Check if user has one of the required permission levels
    for (const requiredPerm of requiredPermissions) {
      if (permission === requiredPerm || (requiredPerm === "maintainer" && permission === "maintain")) {
        core.info(`✅ User has ${permission} access to repository`);
        return { authorized: true, permission: permission };
      }
    }

    core.warning(`User permission '${permission}' does not meet requirements: ${requiredPermissions.join(", ")}`);
    return { authorized: false, permission: permission };
  } catch (repoError) {
    const errorMessage = repoError instanceof Error ? repoError.message : String(repoError);
    core.warning(`Repository permission check failed: ${errorMessage}`);
    return { authorized: false, error: errorMessage };
  }
}

module.exports = {
  parseRequiredPermissions,
  checkRepositoryPermission,
};
