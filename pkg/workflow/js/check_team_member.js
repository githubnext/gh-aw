async function setCheckTeamMemberCancelled(message) {
    try {
        await github.rest.actions.cancelWorkflowRun({
            owner: context.repo.owner,
            repo: context.repo.repo,
            run_id: context.runId,
        });
        core.info(`Cancellation requested for this workflow run: ${message}`);
    }
    catch (error) {
        const errorMessage = error instanceof Error ? error.message : String(error);
        core.warning(`Failed to cancel workflow run: ${errorMessage}`);
        core.setFailed(message);
    }
}
async function checkTeamMemberMain() {
    const actor = context.actor;
    const { owner, repo } = context.repo;
    try {
        core.info(`Checking if user '${actor}' is admin or maintainer of ${owner}/${repo}`);
        const repoPermission = await github.rest.repos.getCollaboratorPermissionLevel({
            owner: owner,
            repo: repo,
            username: actor,
        });
        const permission = repoPermission.data.permission;
        core.info(`Repository permission level: ${permission}`);
        if (permission === "admin" || permission === "maintain") {
            core.info(`User has ${permission} access to repository`);
            core.setOutput("is_team_member", "true");
            return;
        }
    }
    catch (repoError) {
        const errorMessage = repoError instanceof Error ? repoError.message : String(repoError);
        core.warning(`Repository permission check failed: ${errorMessage}`);
    }
    core.warning(`Access denied: Only authorized team members can trigger this workflow. User '${actor}' is not authorized.`);
    await setCheckTeamMemberCancelled(`Access denied: User '${actor}' is not authorized for this workflow`);
    core.setOutput("is_team_member", "false");
}
(async () => {
    await checkTeamMemberMain();
})();
