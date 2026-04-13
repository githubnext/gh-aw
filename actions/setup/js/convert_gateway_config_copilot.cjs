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

const fs = require("fs");
const path = require("path");

const OUTPUT_PATH = "/home/runner/.copilot/mcp-config.json";

/**
 * Rewrite a gateway URL to use the configured domain and port.
 * Replaces http://<anything>/mcp/ with http://<domain>:<port>/mcp/.
 *
 * @param {string} url - Original URL from gateway output
 * @param {string} urlPrefix - Target URL prefix (e.g., http://host.docker.internal:80)
 * @returns {string} Rewritten URL
 */
function rewriteUrl(url, urlPrefix) {
  return url.replace(/^http:\/\/[^/]+\/mcp\//, `${urlPrefix}/mcp/`);
}

function main() {
  const gatewayOutput = process.env.MCP_GATEWAY_OUTPUT;
  const domain = process.env.MCP_GATEWAY_DOMAIN;
  const port = process.env.MCP_GATEWAY_PORT;

  if (!gatewayOutput) {
    core.error("ERROR: MCP_GATEWAY_OUTPUT environment variable is required");
    process.exit(1);
  }
  if (!fs.existsSync(gatewayOutput)) {
    core.error(`ERROR: Gateway output file not found: ${gatewayOutput}`);
    process.exit(1);
  }
  if (!domain) {
    core.error("ERROR: MCP_GATEWAY_DOMAIN environment variable is required");
    process.exit(1);
  }
  if (!port) {
    core.error("ERROR: MCP_GATEWAY_PORT environment variable is required");
    process.exit(1);
  }

  core.info("Converting gateway configuration to Copilot format...");
  core.info(`Input: ${gatewayOutput}`);
  core.info(`Target domain: ${domain}:${port}`);

  const urlPrefix = `http://${domain}:${port}`;

  /** @type {Set<string>} */
  const cliServers = new Set(JSON.parse(process.env.GH_AW_MCP_CLI_SERVERS || "[]"));
  if (cliServers.size > 0) {
    core.info(`CLI-mounted servers to filter: ${[...cliServers].join(", ")}`);
  }

  /** @type {Record<string, unknown>} */
  const config = JSON.parse(fs.readFileSync(gatewayOutput, "utf8"));
  const rawServers = config.mcpServers;
  const servers =
    /** @type {Record<string, Record<string, unknown>>} */
    rawServers && typeof rawServers === "object" && !Array.isArray(rawServers) ? rawServers : {};

  /** @type {Record<string, Record<string, unknown>>} */
  const result = {};
  for (const [name, value] of Object.entries(servers)) {
    if (cliServers.has(name)) continue;
    const entry = { ...value };
    // Add tools field if not present
    if (!entry.tools) {
      entry.tools = ["*"];
    }
    // Fix the URL to use the correct domain
    if (typeof entry.url === "string") {
      entry.url = rewriteUrl(entry.url, urlPrefix);
    }
    result[name] = entry;
  }

  const output = JSON.stringify({ mcpServers: result }, null, 2);

  const serverCount = Object.keys(result).length;
  const totalCount = Object.keys(servers).length;
  const filteredCount = totalCount - serverCount;
  core.info(`Servers: ${serverCount} included, ${filteredCount} filtered (CLI-mounted)`);

  // Ensure output directory exists
  fs.mkdirSync(path.dirname(OUTPUT_PATH), { recursive: true });

  // Write with owner-only permissions (0o600) to protect the gateway bearer token.
  // An attacker who reads mcp-config.json could bypass --allowed-tools by issuing
  // raw JSON-RPC calls directly to the gateway.
  fs.writeFileSync(OUTPUT_PATH, output, { mode: 0o600 });

  core.info(`Copilot configuration written to ${OUTPUT_PATH}`);
  core.info("");
  core.info("Converted configuration:");
  core.info(output);
}

main();

module.exports = { rewriteUrl };
