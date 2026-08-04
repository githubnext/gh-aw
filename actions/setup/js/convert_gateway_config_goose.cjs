// @ts-check
"use strict";

// Ensures global.core is available when running outside github-script context
require("./shim.cjs");

/**
 * convert_gateway_config_goose.cjs
 *
 * Converts the MCP gateway's standard HTTP-based configuration to the JSON
 * format expected by the Goose CLI harness (.goose/mcp.json). Reads the
 * gateway output JSON, filters out CLI-mounted servers, sets
 * type:"streamable_http" (Goose's remote MCP extension type), rewrites URLs
 * to use the correct domain, and writes the result to
 * ${GITHUB_WORKSPACE}/.goose/mcp.json.
 *
 * The Goose CLI does not support MCP configuration files directly; instead,
 * the harness script (see .github/workflows/shared/goose.md) reads this file
 * and translates each entry into a `--with-streamable-http-extension` (for
 * HTTP servers) or `--with-extension` (for stdio servers) CLI flag, plus a
 * generated Goose config-file overlay (via GOOSE_ADDITIONAL_CONFIG_FILES) so
 * that the Authorization header required by the gateway can be attached to
 * each extension.
 *
 * Required environment variables:
 * - MCP_GATEWAY_OUTPUT: Path to gateway output configuration file
 * - MCP_GATEWAY_DOMAIN: Domain for MCP server URLs (required by loadGatewayContext)
 * - MCP_GATEWAY_HOST_DOMAIN: Host-side domain for Goose MCP URLs (e.g., localhost)
 * - MCP_GATEWAY_PORT: Port for MCP gateway (e.g., 80)
 * - GITHUB_WORKSPACE: Workspace directory for the .goose/mcp.json file
 *
 * Optional:
 * - GH_AW_MCP_CLI_SERVERS: JSON array of server names to exclude from agent config
 */

const path = require("path");
const { normalizeGatewayEntry, loadGatewayContext, logCLIFilters, filterAndTransformServers, logServerStats, writeSecureOutput } = require("./convert_gateway_config_shared.cjs");

/**
 * @param {Record<string, unknown>} entry
 * @param {string} urlPrefix
 * @returns {Record<string, unknown>}
 */
function transformGooseEntry(entry, urlPrefix) {
  return normalizeGatewayEntry(entry, urlPrefix, transformed => {
    // Goose's remote MCP extension type is "streamable_http" (per the MCP
    // Streamable HTTP specification); the gateway's default "http" type is
    // not recognized by the Goose harness.
    transformed.type = "streamable_http";
    // The MCP gateway may include a "tools" field for Copilot, but Goose's
    // MCP config format does not support that field.
    delete transformed.tools;
  });
}

function main() {
  const { gatewayOutput, port, cliServers, servers, extraEnv } = loadGatewayContext({
    extraRequiredEnv: ["GITHUB_WORKSPACE"],
  });
  const workspace = extraEnv.GITHUB_WORKSPACE;

  // Goose runs directly on the host runner (not inside a Docker container), so use
  // MCP_GATEWAY_HOST_DOMAIN (localhost) instead of MCP_GATEWAY_DOMAIN (host.docker.internal).
  // host.docker.internal does not resolve on the host runner on Linux.
  const hostDomain = process.env.MCP_GATEWAY_HOST_DOMAIN || "localhost";
  const urlPrefix = `http://${hostDomain}:${port}`;

  core.info("Converting gateway configuration to Goose format...");
  core.info(`Input: ${gatewayOutput}`);
  core.info(`Target domain: ${hostDomain}:${port}`);
  logCLIFilters(cliServers);
  const result = filterAndTransformServers(servers, cliServers, (_name, entry) => transformGooseEntry(entry, urlPrefix));

  const output = JSON.stringify({ mcpServers: result }, null, 2);

  logServerStats(servers, Object.keys(result).length);

  // Create .goose directory in the workspace (matches behaviors.mcp.config-path
  // in .github/workflows/shared/goose.md)
  const configFile = path.join(workspace, ".goose", "mcp.json");

  // Write with owner-only permissions (0o600) to protect the gateway bearer token.
  // mcp.json contains the bearer token for the MCP gateway; an attacker who
  // reads it could bypass the tool constraints by issuing raw JSON-RPC calls
  // directly to the gateway.
  writeSecureOutput(configFile, output);

  core.info(`Goose configuration written to ${configFile}`);
  core.info("");
  core.info("Converted configuration:");
  core.info(output);
}

if (require.main === module) {
  main();
}

module.exports = { transformGooseEntry, main };
