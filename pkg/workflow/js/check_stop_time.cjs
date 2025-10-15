async function main() {
  const stopTime = process.env.GITHUB_AW_STOP_TIME;
  const workflowName = process.env.GITHUB_AW_WORKFLOW_NAME;

  if (!stopTime) {
    core.warning("❌ Configuration error: GITHUB_AW_STOP_TIME not specified.");
    core.setOutput("stop_time_ok", "true"); // Default to allow if not configured
    core.setOutput("result", "config_error");
    return;
  }

  if (!workflowName) {
    core.warning("❌ Configuration error: GITHUB_AW_WORKFLOW_NAME not specified.");
    core.setOutput("stop_time_ok", "true"); // Default to allow if not configured
    core.setOutput("result", "config_error");
    return;
  }

  core.info(`Checking stop-time limit: ${stopTime}`);

  // Parse the stop time (format: "YYYY-MM-DD HH:MM:SS")
  const stopTimeDate = new Date(stopTime);

  if (isNaN(stopTimeDate.getTime())) {
    core.warning(`⚠️ Invalid stop-time format: ${stopTime}. Expected format: YYYY-MM-DD HH:MM:SS`);
    core.setOutput("stop_time_ok", "true"); // Default to allow if invalid format
    core.setOutput("result", "invalid_format");
    return;
  }

  const currentTime = new Date();
  core.info(`Current time: ${currentTime.toISOString()}`);
  core.info(`Stop time: ${stopTimeDate.toISOString()}`);

  if (currentTime >= stopTimeDate) {
    core.warning(`⏰ Stop time reached. Attempting to disable workflow to prevent cost overrun.`);

    try {
      // Disable the workflow using GitHub API
      const { owner, repo } = context.repo;

      // Get all workflows to find the one with matching name
      const { data: workflows } = await github.rest.actions.listRepoWorkflows({
        owner,
        repo,
      });

      const workflow = workflows.workflows.find(w => w.name === workflowName);

      if (workflow) {
        await github.rest.actions.disableWorkflow({
          owner,
          repo,
          workflow_id: workflow.id,
        });
        core.info(`✅ Workflow '${workflowName}' disabled. No future runs will be triggered.`);
      } else {
        core.warning(`⚠️ Could not find workflow '${workflowName}' to disable.`);
      }
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      core.warning(`⚠️ Failed to disable workflow: ${errorMessage}`);
    }

    core.setOutput("stop_time_ok", "false");
    core.setOutput("result", "stop_time_reached");
    core.setFailed("Stop time reached. Workflow execution stopped to prevent cost overrun.");
    return;
  }

  core.info("✅ All safety checks passed. Proceeding with agentic tool execution.");
  core.setOutput("stop_time_ok", "true");
  core.setOutput("result", "ok");
}
await main();
