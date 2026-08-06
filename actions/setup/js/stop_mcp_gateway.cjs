// @ts-check
/// <reference types="@actions/github-script" />

const path = require("path");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Emits only the MCP gateway container's stderr stream. Gateway stdout contains
 * the generated client configuration and must never be written to the Actions log.
 *
 * @param {typeof exec} execApi
 * @param {typeof core} coreApi
 */
async function printGatewayStderr(execApi, coreApi) {
  const result = await execApi.getExecOutput("docker", ["logs", "awmg-mcpg"], {
    ignoreReturnCode: true,
    silent: true,
  });

  const stderr = result.stderr.trim();
  if (stderr) {
    coreApi.info(`MCP Gateway stderr:\n${stderr}`);
  } else if (result.exitCode === 0) {
    coreApi.info("MCP Gateway produced no stderr output.");
  } else {
    coreApi.info("MCP Gateway stderr is unavailable.");
  }
}

async function main() {
  await printGatewayStderr(exec, core);

  const runnerTemp = process.env.RUNNER_TEMP;
  if (!runnerTemp) {
    throw new Error("RUNNER_TEMP environment variable is not set");
  }

  await exec.exec("bash", [path.join(runnerTemp, "gh-aw/actions/stop_mcp_gateway.sh"), process.env.GATEWAY_PID || ""]);
}

module.exports = {
  main,
  printGatewayStderr,
};

if (require.main === module) {
  require("./shim.cjs");
  main().catch(error => {
    core.setFailed(getErrorMessage(error));
  });
}
