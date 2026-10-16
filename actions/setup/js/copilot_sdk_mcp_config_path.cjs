// @ts-check
"use strict";

const path = require("node:path");

/**
 * Keep SDK registration separate from the user-managed Copilot CLI catalog.
 * RUNNER_TEMP/gh-aw is already mounted or staged into each agent sandbox.
 *
 * @param {string | undefined} [runnerTemp]
 * @returns {string}
 */
function resolveCopilotSDKMCPConfigPath(runnerTemp = process.env.RUNNER_TEMP) {
  if (!runnerTemp) {
    throw new Error("RUNNER_TEMP is required to locate the generated Copilot SDK MCP configuration");
  }
  return path.join(runnerTemp, "gh-aw", "mcp-config", "copilot-sdk.json");
}

module.exports = { resolveCopilotSDKMCPConfigPath };
