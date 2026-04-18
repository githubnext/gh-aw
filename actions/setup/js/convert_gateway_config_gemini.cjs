// @ts-check
"use strict";

// Ensures global.core is available when running outside github-script context
require("./shim.cjs");

/**
 * convert_gateway_config_gemini.cjs
 *
 * Converts the MCP gateway's standard HTTP-based configuration to the JSON
 * format expected by Gemini CLI (.gemini/settings.json). Reads the gateway
 * output JSON, filters out CLI-mounted servers, removes the "type" field
 * (Gemini uses transport auto-detection), rewrites URLs to use the correct
 * domain, and adds /tmp/ to context.includeDirectories.
 *
 * Gemini CLI reads MCP server configuration from settings.json files:
 * - Global: ~/.gemini/settings.json
 * - Project: .gemini/settings.json (used here)
 *
 * See: https://geminicli.com/docs/tools/mcp-server/
 *
 * Required environment variables:
 * - MCP_GATEWAY_OUTPUT: Path to gateway output configuration file
 * - MCP_GATEWAY_DOMAIN: Domain for MCP server URLs (e.g., host.docker.internal)
 * - MCP_GATEWAY_PORT: Port for MCP gateway (e.g., 80)
 * - GITHUB_WORKSPACE: Workspace directory for project-level settings
 *
 * Optional:
 * - GH_AW_MCP_CLI_SERVERS: JSON array of server names to exclude from agent config
 */

const path = require("path");
const { rewriteUrl, loadGatewayContext, logCLIFilters, filterAndTransformServers, logServerStats, writeSecureOutput } = require("./convert_gateway_config_shared.cjs");

/**
 * @param {Record<string, unknown>} entry
 * @param {string} urlPrefix
 * @returns {Record<string, unknown>}
 */
function transformGeminiEntry(entry, urlPrefix) {
  const transformed = { ...entry };
  // Remove "type" field — Gemini uses transport auto-detection from url/httpUrl
  delete transformed.type;
  // Fix the URL to use the correct domain
  if (typeof transformed.url === "string") {
    transformed.url = rewriteUrl(transformed.url, urlPrefix);
  }
  return transformed;
}

function main() {
  const { gatewayOutput, domain, port, urlPrefix, cliServers, servers, extraEnv } = loadGatewayContext({
    extraRequiredEnv: ["GITHUB_WORKSPACE"],
  });
  const workspace = extraEnv.GITHUB_WORKSPACE;

  core.info("Converting gateway configuration to Gemini format...");
  core.info(`Input: ${gatewayOutput}`);
  core.info(`Target domain: ${domain}:${port}`);
  logCLIFilters(cliServers);
  const result = filterAndTransformServers(servers, cliServers, (_name, entry) => transformGeminiEntry(entry, urlPrefix));

  // Build settings with mcpServers and context.includeDirectories
  // Allow Gemini CLI to read/write files from /tmp/ (e.g. MCP payload files,
  // cache-memory, agent outputs)
  const settings = {
    mcpServers: result,
    context: {
      includeDirectories: ["/tmp/"],
    },
  };

  const output = JSON.stringify(settings, null, 2);

  logServerStats(servers, Object.keys(result).length);

  // Create .gemini directory in the workspace (project-level settings)
  const settingsFile = path.join(workspace, ".gemini", "settings.json");

  // Write with owner-only permissions (0o600) to protect the gateway bearer token.
  // settings.json contains the bearer token for the MCP gateway; an attacker
  // who reads it could bypass the --allowed-tools constraint by issuing raw
  // JSON-RPC calls directly to the gateway.
  writeSecureOutput(settingsFile, output);

  core.info(`Gemini configuration written to ${settingsFile}`);
  core.info("");
  core.info("Converted configuration:");
  core.info(output);
}

if (require.main === module) {
  main();
}

module.exports = { rewriteUrl, transformGeminiEntry, main };
