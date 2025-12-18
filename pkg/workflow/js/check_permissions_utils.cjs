function parseRequiredPermissions() {
  const requiredPermissionsEnv = process.env.GH_AW_REQUIRED_ROLES;
  return requiredPermissionsEnv ? requiredPermissionsEnv.split(",").filter(p => "" !== p.trim()) : [];
}
function parseAllowedBots() {
  const allowedBotsEnv = process.env.GH_AW_ALLOWED_BOTS;
  return allowedBotsEnv ? allowedBotsEnv.split(",").filter(b => "" !== b.trim()) : [];
}
async function checkBotStatus(actor, owner, repo) {
  try {
    if (!actor.endsWith("[bot]")) return { isBot: !1, isActive: !1 };
    core.info(`Checking if bot '${actor}' is active on ${owner}/${repo}`);
    try {
      const botPermission = await github.rest.repos.getCollaboratorPermissionLevel({ owner, repo, username: actor });
      return (core.info(`Bot '${actor}' is active with permission level: ${botPermission.data.permission}`), { isBot: !0, isActive: !0 });
    } catch (botError) {
      if ("object" == typeof botError && null !== botError && "status" in botError && 404 === botError.status) return (core.warning(`Bot '${actor}' is not active/installed on ${owner}/${repo}`), { isBot: !0, isActive: !1 });
      const errorMessage = botError instanceof Error ? botError.message : String(botError);
      return (core.warning(`Failed to check bot status: ${errorMessage}`), { isBot: !0, isActive: !1, error: errorMessage });
    }
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    return (core.warning(`Error checking bot status: ${errorMessage}`), { isBot: !1, isActive: !1, error: errorMessage });
  }
}
async function checkRepositoryPermission(actor, owner, repo, requiredPermissions) {
  try {
    (core.info(`Checking if user '${actor}' has required permissions for ${owner}/${repo}`), core.info(`Required permissions: ${requiredPermissions.join(", ")}`));
    const permission = (await github.rest.repos.getCollaboratorPermissionLevel({ owner, repo, username: actor })).data.permission;
    core.info(`Repository permission level: ${permission}`);
    for (const requiredPerm of requiredPermissions)
      if (permission === requiredPerm || ("maintainer" === requiredPerm && "maintain" === permission)) return (core.info(`✅ User has ${permission} access to repository`), { authorized: !0, permission });
    return (core.warning(`User permission '${permission}' does not meet requirements: ${requiredPermissions.join(", ")}`), { authorized: !1, permission });
  } catch (repoError) {
    const errorMessage = repoError instanceof Error ? repoError.message : String(repoError);
    return (core.warning(`Repository permission check failed: ${errorMessage}`), { authorized: !1, error: errorMessage });
  }
}
module.exports = { parseRequiredPermissions, parseAllowedBots, checkRepositoryPermission, checkBotStatus };
