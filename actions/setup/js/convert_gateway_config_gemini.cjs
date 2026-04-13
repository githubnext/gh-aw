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

const fs = require("fs");
const path = require("path");

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
  const workspace = process.env.GITHUB_WORKSPACE;

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
  if (!workspace) {
    core.error("ERROR: GITHUB_WORKSPACE environment variable is required");
    process.exit(1);
  }

  core.info("Converting gateway configuration to Gemini format...");
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
    // Remove "type" field — Gemini uses transport auto-detection from url/httpUrl
    delete entry.type;
    // Fix the URL to use the correct domain
    if (typeof entry.url === "string") {
      entry.url = rewriteUrl(entry.url, urlPrefix);
    }
    result[name] = entry;
  }

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

  const serverCount = Object.keys(result).length;
  const totalCount = Object.keys(servers).length;
  const filteredCount = totalCount - serverCount;
  core.info(`Servers: ${serverCount} included, ${filteredCount} filtered (CLI-mounted)`);

  // Create .gemini directory in the workspace (project-level settings)
  const settingsDir = path.join(workspace, ".gemini");
  const settingsFile = path.join(settingsDir, "settings.json");
  fs.mkdirSync(settingsDir, { recursive: true });

  // Write with owner-only permissions (0o600) to protect the gateway bearer token.
  // settings.json contains the bearer token for the MCP gateway; an attacker
  // who reads it could bypass the --allowed-tools constraint by issuing raw
  // JSON-RPC calls directly to the gateway.
  fs.writeFileSync(settingsFile, output, { mode: 0o600 });

  core.info(`Gemini configuration written to ${settingsFile}`);
  core.info("");
  core.info("Converted configuration:");
  core.info(output);
}

main();

module.exports = { rewriteUrl };
