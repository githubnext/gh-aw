async function main() {
  const actor = context.actor;
  const { owner, repo } = context.repo;

  // Check if the actor has repository access (admin, maintain permissions)
  try {
    console.log(
      `Checking if user '${actor}' is admin or maintainer of ${owner}/${repo}`
    );

    const repoPermission =
      await github.rest.repos.getCollaboratorPermissionLevel({
        owner: owner,
        repo: repo,
        username: actor,
      });

    const permission = repoPermission.data.permission;
    console.log(`Repository permission level: ${permission}`);

    if (permission === "admin" || permission === "maintain") {
      console.log(`User has ${permission} access to repository`);
      core.setOutput("is_team_member", "true");
      return;
    }
  } catch (repoError) {
    const errorMessage =
      repoError instanceof Error ? repoError.message : String(repoError);
    core.warning(`Repository permission check failed: ${errorMessage}`);
  }

  core.setOutput("is_team_member", "false");
}
await main();
