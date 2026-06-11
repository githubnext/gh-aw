// @ts-check
"use strict";

// Ensures global.core is available when running outside github-script context
require("./shim.cjs");

/**
 * convert_gateway_config_copilot.cjs
 *
 * Converts the MCP gateway's standard HTTP-based configuration to the format
 * expected by GitHub Copilot CLI. Reads the gateway output JSON, filters out
 * CLI-mounted servers, adds tools:["*"] if missing, rewrites URLs to use the
 * correct domain, and writes the result to $HOME/.copilot/mcp-config.json
 * (typically /home/runner/.copilot/mcp-config.json on GitHub-hosted runners,
 * but may differ on self-hosted or containerized runners where HOME varies).
 *
 * Required environment variables:
 * - MCP_GATEWAY_OUTPUT: Path to gateway output configuration file
 * - MCP_GATEWAY_DOMAIN: Domain for MCP server URLs (e.g., host.docker.internal)
 * - MCP_GATEWAY_PORT: Port for MCP gateway (e.g., 80)
 * - HOME: User home directory (standard POSIX env var inherited by the runner)
 *
 * Optional:
 * - GH_AW_MCP_CLI_SERVERS: JSON array of server names to exclude from agent config
 */

const path = require("path");
const { rewriteUrl, loadGatewayContext, logCLIFilters, filterAndTransformServers, logServerStats, writeSecureOutput } = require("./convert_gateway_config_shared.cjs");

/**
 * Resolves the Copilot CLI MCP config output path from the runtime $HOME.
 * The Copilot CLI uses ~/.copilot, which is /home/runner/.copilot on standard
 * GitHub-hosted runners (HOME=/home/runner) but may differ on self-hosted or
 * containerized runners. HOME is a standard POSIX environment variable inherited
 * from the runner's parent process and passed through to shell steps; other
 * generators (copilot_mcp.go, copilot_engine_execution.go) rely on it the same way.
 *
 * Exported for testability; throws Error rather than exiting so tests can
 * exercise the missing-HOME branch.
 *
 * @returns {string}
 */
function resolveCopilotConfigOutputPath() {
  const home = process.env.HOME;
  if (!home) {
    throw new Error("HOME environment variable is not set; cannot locate Copilot CLI config directory");
  }
  return path.join(home, ".copilot", "mcp-config.json");
}

/**
 * @param {Record<string, unknown>} entry
 * @param {string} urlPrefix
 * @returns {Record<string, unknown>}
 */
function transformCopilotEntry(entry, urlPrefix) {
  const transformed = { ...entry };
  // Add tools field if not present
  if (!transformed.tools) {
    transformed.tools = ["*"];
  }
  // Fix the URL to use the correct domain
  if (typeof transformed.url === "string") {
    transformed.url = rewriteUrl(transformed.url, urlPrefix);
  }
  return transformed;
}

function main() {
  let outputPath;
  try {
    outputPath = resolveCopilotConfigOutputPath();
  } catch (err) {
    core.error(`ERROR: ${err.message}`);
    process.exit(1);
  }

  const { gatewayOutput, domain, port, urlPrefix, cliServers, servers } = loadGatewayContext();

  core.info("Converting gateway configuration to Copilot format...");
  core.info(`Input: ${gatewayOutput}`);
  core.info(`Target domain: ${domain}:${port}`);
  logCLIFilters(cliServers);
  const result = filterAndTransformServers(servers, cliServers, (_name, entry) => transformCopilotEntry(entry, urlPrefix));

  const output = JSON.stringify({ mcpServers: result }, null, 2);
  logServerStats(servers, Object.keys(result).length);

  // Write with owner-only permissions (0o600) to protect the gateway bearer token.
  // An attacker who reads mcp-config.json could bypass --allowed-tools by issuing
  // raw JSON-RPC calls directly to the gateway.
  writeSecureOutput(outputPath, output);

  core.info(`Copilot configuration written to ${outputPath}`);
  core.info("");
  core.info("Converted configuration:");
  core.info(output);
}

if (require.main === module) {
  main();
}

module.exports = { rewriteUrl, transformCopilotEntry, resolveCopilotConfigOutputPath, main };
