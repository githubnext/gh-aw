async function main() {
  const actor = context.actor;
  const { owner, repo } = context.repo;
  const requiredPermissions = process.env.REQUIRED_PERMISSIONS?.split(",") || [
    "admin",
    "maintain",
  ];

  // Check if the actor has the required repository permissions
  try {
    console.log(
      `Checking if user '${actor}' has required permissions for ${owner}/${repo}`
    );
    console.log(`Required permissions: ${requiredPermissions.join(", ")}`);

    const repoPermission =
      await github.rest.repos.getCollaboratorPermissionLevel({
        owner: owner,
        repo: repo,
        username: actor,
      });

    const permission = repoPermission.data.permission;
    console.log(`Repository permission level: ${permission}`);

    // Check if user has one of the required permission levels
    for (const requiredPerm of requiredPermissions) {
      if (
        permission === requiredPerm ||
        (requiredPerm === "maintainer" && permission === "maintain")
      ) {
        console.log(`✅ User has ${permission} access to repository`);
        core.setOutput("is_team_member", "true");
        return;
      }
    }

    console.log(
      `User permission '${permission}' does not meet requirements: ${requiredPermissions.join(", ")}`
    );
  } catch (repoError) {
    const errorMessage =
      repoError instanceof Error ? repoError.message : String(repoError);
    core.warning(`Repository permission check failed: ${errorMessage}`);
  }

  // Fail the job directly when permission check fails
  core.setFailed(
    `❌ Access denied: Only authorized users can trigger this workflow. User '${actor}' is not authorized. Required permissions: ${requiredPermissions.join(", ")}`
  );
}
await main();
