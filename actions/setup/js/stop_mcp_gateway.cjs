// @ts-check
/// <reference types="@actions/github-script" />

const path = require("path");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Emits the MCP gateway container's stderr stream via core.debug after shutdown.
 * Gateway stdout contains the generated client configuration and must never be
 * written to the Actions log or persisted in artifacts.
 *
 * @param {typeof exec} execApi
 * @param {typeof core} coreApi
 */
async function collectGatewayStderr(execApi, coreApi) {
  const result = await execApi.getExecOutput("docker", ["logs", "awmg-mcpg"], {
    ignoreReturnCode: true,
    silent: true,
  });

  const stderr = result.stderr.trim();
  if (result.exitCode !== 0 && !stderr) {
    coreApi.info("MCP Gateway stderr is unavailable.");
    return;
  }

  coreApi.debug(`MCP Gateway stderr:\n${stderr}`);
}

/**
 * Removes the stopped gateway container from Docker.
 * Called after logs have been collected.
 *
 * @param {typeof exec} execApi
 * @param {typeof core} coreApi
 */
async function removeGatewayContainer(execApi, coreApi) {
  const result = await execApi.getExecOutput("docker", ["rm", "awmg-mcpg"], {
    ignoreReturnCode: true,
    silent: true,
  });
  if (result.exitCode !== 0) {
    coreApi.info("MCP Gateway container already removed or not found.");
  }
}

async function main() {
  const runnerTemp = process.env.RUNNER_TEMP;
  if (!runnerTemp) {
    throw new Error("RUNNER_TEMP environment variable is not set");
  }

  // Stop the gateway first so that all shutdown-time stderr is captured by Docker.
  // The stop script's EXIT trap issues `docker stop` (not `docker rm`), so the
  // container remains available for log collection after the script exits.
  await exec.exec("bash", [path.join(runnerTemp, "gh-aw/actions/stop_mcp_gateway.sh"), process.env.GATEWAY_PID || ""]);

  // Collect stderr from the now-stopped container and emit via core.debug.
  await collectGatewayStderr(exec, core);

  // Explicitly remove the container now that logs have been harvested.
  await removeGatewayContainer(exec, core);
}

module.exports = {
  main,
  collectGatewayStderr,
  removeGatewayContainer,
};

if (require.main === module) {
  require("./shim.cjs");
  main().catch(error => {
    core.setFailed(getErrorMessage(error));
  });
}
