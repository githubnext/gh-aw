// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const path = require("path");
const { getErrorMessage } = require("./error_helpers.cjs");

const MCP_STDERR_LOG_PATH = "/tmp/gh-aw/mcp-logs/stderr.log";

/**
 * Collects the MCP gateway container's stderr stream after the container has stopped
 * and writes it to the excluded log path for downstream parser consumption.
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

  try {
    fs.mkdirSync(path.dirname(MCP_STDERR_LOG_PATH), { recursive: true });
    fs.writeFileSync(MCP_STDERR_LOG_PATH, stderr, { mode: 0o600 });
    coreApi.info(`MCP Gateway stderr written to log (${stderr.length} bytes).`);
  } catch (err) {
    coreApi.info(`Failed to write MCP Gateway stderr log: ${getErrorMessage(err)}`);
  }
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

  // Collect stderr from the now-stopped container and persist it to the excluded
  // log path so downstream parsers (parse_mcp_gateway_log.cjs) can detect
  // ai_credits_rate_limit_error and unknown_model_ai_credits. The path is
  // excluded from agent artifacts to prevent credential leakage.
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
