// @ts-check
"use strict";

// Ensures global.core is available when running outside github-script context
require("./shim.cjs");

/**
 * convert_gateway_config_codex.cjs
 *
 * Converts the MCP gateway's standard HTTP-based configuration to the TOML
 * format expected by Codex. Reads the gateway output JSON, filters out
 * CLI-mounted servers, resolves host.docker.internal to 172.30.0.1 for Rust
 * DNS compatibility, and writes the result to ${RUNNER_TEMP}/gh-aw/mcp-config/config.toml.
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
const { loadGatewayContext, logCLIFilters, filterAndTransformServers, logServerStats, writeSecureOutput } = require("./convert_gateway_config_shared.cjs");

const OUTPUT_PATH = path.join(process.env.RUNNER_TEMP || "/tmp", "gh-aw/mcp-config/config.toml");

/**
 * @param {string} name
 * @param {Record<string, unknown>} value
 * @param {string} urlPrefix
 * @returns {string}
 */
function toCodexTomlSection(name, value, urlPrefix) {
  const url = `${urlPrefix}/mcp/${name}`;
  const rawHeaders = value.headers;
  /** @type {Record<string, string>} */
  const headers = rawHeaders && typeof rawHeaders === "object" && !Array.isArray(rawHeaders) ? Object.fromEntries(Object.entries(rawHeaders).filter(([, headerValue]) => typeof headerValue === "string")) : {};
  const authKey = headers.Authorization || "";
  let section = `[mcp_servers.${name}]\n`;
  section += `url = "${url}"\n`;
  section += `http_headers = { Authorization = "${authKey}" }\n`;
  section += "\n";
  return section;
}

function main() {
  const { gatewayOutput, domain, port, cliServers, servers } = loadGatewayContext();

  core.info("Converting gateway configuration to Codex TOML format...");
  core.info(`Input: ${gatewayOutput}`);
  core.info(`Target domain: ${domain}:${port}`);

  // For host.docker.internal, resolve to the gateway IP to avoid DNS resolution
  // issues in Rust
  let resolvedDomain = domain;
  if (domain === "host.docker.internal") {
    // AWF network gateway IP is always 172.30.0.1
    resolvedDomain = "172.30.0.1";
    core.info(`Resolving host.docker.internal to gateway IP: ${resolvedDomain}`);
  }

  const urlPrefix = `http://${resolvedDomain}:${port}`;
  logCLIFilters(cliServers);
  const filteredServers = filterAndTransformServers(servers, cliServers, (_name, entry) => entry);

  // Build the TOML output
  let toml = '[history]\npersistence = "none"\n\n';

  for (const [name, value] of Object.entries(filteredServers)) {
    toml += toCodexTomlSection(name, value, urlPrefix);
  }

  logServerStats(servers, Object.keys(filteredServers).length);

  // Write with owner-only permissions (0o600) to protect the gateway bearer token.
  // An attacker who reads config.toml could issue raw JSON-RPC calls directly
  // to the gateway.
  writeSecureOutput(OUTPUT_PATH, toml);

  core.info(`Codex configuration written to ${OUTPUT_PATH}`);
  core.info("");
  core.info("Converted configuration:");
  core.info(toml);
}

if (require.main === module) {
  main();
}

module.exports = { toCodexTomlSection, main };
