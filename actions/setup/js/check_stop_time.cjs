// @ts-check
/// <reference types="@actions/github-script" />

const { ERR_CONFIG, ERR_VALIDATION } = require("./error_codes.cjs");
async function main() {
  const stopTime = process.env.GH_AW_STOP_TIME;
  const workflowName = process.env.GH_AW_WORKFLOW_NAME;

  if (!stopTime) {
    core.setFailed(`${ERR_CONFIG}: Configuration error: GH_AW_STOP_TIME not specified.`);
    return;
  }

  if (!workflowName) {
    core.setFailed(`${ERR_CONFIG}: Configuration error: GH_AW_WORKFLOW_NAME not specified.`);
    return;
  }

  core.info(`Checking stop-time limit: ${stopTime}`);

  // Parse the stop time (format: "YYYY-MM-DD HH:MM:SS")
  const stopTimeDate = new Date(stopTime);

  if (isNaN(stopTimeDate.getTime())) {
    core.setFailed(`${ERR_VALIDATION}: Invalid stop-time format: ${stopTime}. Expected format: YYYY-MM-DD HH:MM:SS`);
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

module.exports = { main };
