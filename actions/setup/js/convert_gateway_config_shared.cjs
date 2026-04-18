// @ts-check
"use strict";

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

/**
 * @param {string} name
 * @returns {string}
 */
function requireEnvVar(name) {
  const value = process.env[name];
  if (!value) {
    core.error(`ERROR: ${name} environment variable is required`);
    process.exit(1);
  }
  return value;
}

/**
 * @param {{ extraRequiredEnv?: string[] }} [options]
 * @returns {{
 *   gatewayOutput: string;
 *   domain: string;
 *   port: string;
 *   urlPrefix: string;
 *   cliServers: Set<string>;
 *   servers: Record<string, Record<string, unknown>>;
 *   extraEnv: Record<string, string>;
 * }}
 */
function loadGatewayContext(options = {}) {
  const extraRequiredEnv = options.extraRequiredEnv || [];
  const gatewayOutput = requireEnvVar("MCP_GATEWAY_OUTPUT");
  if (!fs.existsSync(gatewayOutput)) {
    core.error(`ERROR: Gateway output file not found: ${gatewayOutput}`);
    process.exit(1);
  }

  const domain = requireEnvVar("MCP_GATEWAY_DOMAIN");
  const port = requireEnvVar("MCP_GATEWAY_PORT");

  /** @type {Record<string, string>} */
  const extraEnv = {};
  for (const envVar of extraRequiredEnv) {
    extraEnv[envVar] = requireEnvVar(envVar);
  }

  /** @type {Set<string>} */
  const cliServers = new Set(JSON.parse(process.env.GH_AW_MCP_CLI_SERVERS || "[]"));

  /** @type {Record<string, unknown>} */
  const config = JSON.parse(fs.readFileSync(gatewayOutput, "utf8"));
  const rawServers = config.mcpServers;
  /** @type {Record<string, Record<string, unknown>>} */
  let servers = {};
  if (rawServers && typeof rawServers === "object" && !Array.isArray(rawServers)) {
    servers = Object.fromEntries(Object.entries(rawServers));
  }

  return {
    gatewayOutput,
    domain,
    port,
    urlPrefix: `http://${domain}:${port}`,
    cliServers,
    servers,
    extraEnv,
  };
}

/**
 * @param {Set<string>} cliServers
 */
function logCLIFilters(cliServers) {
  if (cliServers.size > 0) {
    core.info(`CLI-mounted servers to filter: ${[...cliServers].join(", ")}`);
  }
}

/**
 * @param {Record<string, Record<string, unknown>>} servers
 * @param {Set<string>} cliServers
 * @param {(name: string, value: Record<string, unknown>) => Record<string, unknown>} transformServer
 * @returns {Record<string, Record<string, unknown>>}
 */
function filterAndTransformServers(servers, cliServers, transformServer) {
  /** @type {Record<string, Record<string, unknown>>} */
  const result = {};
  for (const [name, value] of Object.entries(servers)) {
    if (cliServers.has(name)) continue;
    const entry = { ...value };
    result[name] = transformServer(name, entry);
  }
  return result;
}

/**
 * @param {Record<string, Record<string, unknown>>} servers
 * @param {number} includedCount
 */
function logServerStats(servers, includedCount) {
  const totalCount = Object.keys(servers).length;
  const filteredCount = totalCount - includedCount;
  core.info(`Servers: ${includedCount} included, ${filteredCount} filtered (CLI-mounted)`);
}

/**
 * @param {string} outputPath
 * @param {string} output
 */
function writeSecureOutput(outputPath, output) {
  fs.mkdirSync(path.dirname(outputPath), { recursive: true });
  fs.writeFileSync(outputPath, output, { mode: 0o600 });
  fs.chmodSync(outputPath, 0o600);
}

module.exports = {
  rewriteUrl,
  loadGatewayContext,
  logCLIFilters,
  filterAndTransformServers,
  logServerStats,
  writeSecureOutput,
};
