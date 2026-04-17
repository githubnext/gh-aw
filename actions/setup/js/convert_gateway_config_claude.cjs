// @ts-check
"use strict";

// Ensures global.core is available when running outside github-script context
require("./shim.cjs");

/**
 * convert_gateway_config_claude.cjs
 *
 * Converts the MCP gateway's standard HTTP-based configuration to the JSON
 * format expected by Claude. Reads the gateway output JSON, filters out
 * CLI-mounted servers, sets type:"http", rewrites URLs to use the correct
 * domain, and writes the result to ${RUNNER_TEMP}/gh-aw/mcp-config/mcp-servers.json.
 *
 * Required environment variables:
 * - MCP_GATEWAY_OUTPUT: Path to gateway output configuration file
 * - MCP_GATEWAY_DOMAIN: Domain for MCP server URLs (e.g., host.docker.internal)
 * - MCP_GATEWAY_PORT: Port for MCP gateway (e.g., 80)
 * - RUNNER_TEMP: GitHub Actions runner temp directory
 *
 * Optional:
 * - GH_AW_MCP_CLI_SERVERS: JSON array of server names to exclude from agent config
 */

const path = require("path");
const { rewriteUrl, loadGatewayContext, logCLIFilters, filterAndTransformServers, logServerStats, writeSecureOutput } = require("./convert_gateway_config_shared.cjs");

const OUTPUT_PATH = path.join(process.env.RUNNER_TEMP || "/tmp", "gh-aw/mcp-config/mcp-servers.json");

/**
 * @param {Record<string, unknown>} entry
 * @param {string} urlPrefix
 * @returns {Record<string, unknown>}
 */
function transformClaudeEntry(entry, urlPrefix) {
  const transformed = { ...entry };
  // Claude uses "type": "http" for HTTP-based MCP servers
  transformed.type = "http";
  // Fix the URL to use the correct domain
  if (typeof transformed.url === "string") {
    transformed.url = rewriteUrl(transformed.url, urlPrefix);
  }
  return transformed;
}

function main() {
  const { gatewayOutput, domain, port, urlPrefix, cliServers, servers } = loadGatewayContext();

  core.info("Converting gateway configuration to Claude format...");
  core.info(`Input: ${gatewayOutput}`);
  core.info(`Target domain: ${domain}:${port}`);
  logCLIFilters(cliServers);
  const result = filterAndTransformServers(servers, cliServers, (_name, entry) => transformClaudeEntry(entry, urlPrefix));

  const output = JSON.stringify({ mcpServers: result }, null, 2);
  logServerStats(servers, Object.keys(result).length);

  // Write with owner-only permissions (0o600) to protect the gateway bearer token.
  // An attacker who reads mcp-servers.json could bypass --allowed-tools by issuing
  // raw JSON-RPC calls directly to the gateway.
  writeSecureOutput(OUTPUT_PATH, output);

  core.info(`Claude configuration written to ${OUTPUT_PATH}`);
  core.info("");
  core.info("Converted configuration:");
  core.info(output);
}

if (require.main === module) {
  main();
}

module.exports = { rewriteUrl, transformClaudeEntry, main };
