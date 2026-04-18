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
 * correct domain, and writes the result to /home/runner/.copilot/mcp-config.json.
 *
 * Required environment variables:
 * - MCP_GATEWAY_OUTPUT: Path to gateway output configuration file
 * - MCP_GATEWAY_DOMAIN: Domain for MCP server URLs (e.g., host.docker.internal)
 * - MCP_GATEWAY_PORT: Port for MCP gateway (e.g., 80)
 *
 * Optional:
 * - GH_AW_MCP_CLI_SERVERS: JSON array of server names to exclude from agent config
 */

const { rewriteUrl, loadGatewayContext, logCLIFilters, filterAndTransformServers, logServerStats, writeSecureOutput } = require("./convert_gateway_config_shared.cjs");

const OUTPUT_PATH = "/home/runner/.copilot/mcp-config.json";

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
  writeSecureOutput(OUTPUT_PATH, output);

  core.info(`Copilot configuration written to ${OUTPUT_PATH}`);
  core.info("");
  core.info("Converted configuration:");
  core.info(output);
}

if (require.main === module) {
  main();
}

module.exports = { rewriteUrl, transformCopilotEntry, main };
