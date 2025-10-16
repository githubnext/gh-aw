async function main() {
  const stopTime = process.env.GITHUB_AW_STOP_TIME;
  const workflowName = process.env.GITHUB_AW_WORKFLOW_NAME;

  if (!stopTime) {
    core.setFailed("Configuration error: GITHUB_AW_STOP_TIME not specified.");
    return;
  }

  if (!workflowName) {
    core.setFailed("Configuration error: GITHUB_AW_WORKFLOW_NAME not specified.");
    return;
  }

  core.info(`Checking stop-time limit: ${stopTime}`);

  // Parse the stop time (format: "YYYY-MM-DD HH:MM:SS")
  const stopTimeDate = new Date(stopTime);

  if (isNaN(stopTimeDate.getTime())) {
    core.setFailed(`Invalid stop-time format: ${stopTime}. Expected format: YYYY-MM-DD HH:MM:SS`);
    return;
  }

  const currentTime = new Date();
  core.info(`Current time: ${currentTime.toISOString()}`);
  core.info(`Stop time: ${stopTimeDate.toISOString()}`);

  if (currentTime >= stopTimeDate) {
    core.warning(`⏰ Stop time reached. Workflow execution will be prevented by activation job.`);
    core.setOutput("stop_time_ok", "false");
    return;
  }

  core.setOutput("stop_time_ok", "true");
}
await main();
