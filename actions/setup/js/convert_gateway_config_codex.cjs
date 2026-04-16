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
 * Security: The safeoutputs write-sink server is given a direct connection (bypassing
 * the gateway) using an env var reference for the Authorization header rather than
 * the shared MCP_GATEWAY_API_KEY. This ensures that a bash subprocess reading the
 * gateway bearer token cannot use it to reach the safeoutputs write-sink, which
 * would bypass the declared read-only permission ceiling.
 *
 * Required environment variables:
 * - MCP_GATEWAY_OUTPUT: Path to gateway output configuration file
 * - MCP_GATEWAY_DOMAIN: Domain for MCP server URLs (e.g., host.docker.internal)
 * - MCP_GATEWAY_PORT: Port for MCP gateway (e.g., 80)
 * - RUNNER_TEMP: GitHub Actions runner temp directory
 *
 * Optional:
 * - GH_AW_MCP_CLI_SERVERS: JSON array of server names to exclude from agent config
 * - GH_AW_SAFE_OUTPUTS_PORT: Port for the safe-outputs HTTP server (default: 3001)
 */

const fs = require("fs");
const path = require("path");

const OUTPUT_PATH = path.join(process.env.RUNNER_TEMP || "/tmp", "gh-aw/mcp-config/config.toml");

/** Server ID for the safe-outputs write-sink MCP server. */
const SAFEOUTPUTS_SERVER_ID = "safeoutputs";

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

  // Build the TOML output
  let toml = '[history]\npersistence = "none"\n\n';

  for (const [name, value] of Object.entries(servers)) {
    if (cliServers.has(name)) continue;

    if (name === SAFEOUTPUTS_SERVER_ID) {
      // The safeoutputs write-sink uses a direct connection with an env var reference
      // for the Authorization header instead of the shared MCP gateway API key.
      // This ensures that a bash subprocess reading the gateway bearer token from
      // config.toml cannot use that token to reach the safeoutputs write-sink,
      // which would bypass the declared read-only permission ceiling.
      //
      // The ${...} reference is written literally; Codex expands it at connection time
      // from the process environment. The LD_PRELOAD one-shot token library prevents
      // bash subprocesses from reading GH_AW_SAFE_OUTPUTS_API_KEY from the environment
      // after the agent process has consumed it on startup.
      const safeOutputsPort = process.env.GH_AW_SAFE_OUTPUTS_PORT || "3001";
      const directUrl = `http://${resolvedDomain}:${safeOutputsPort}`;
      toml += `[mcp_servers.${name}]\n`;
      toml += `url = "${directUrl}"\n`;
      // Literal env var reference — NOT expanded here. Codex resolves it at runtime.
      toml += `http_headers = { Authorization = "\${GH_AW_SAFE_OUTPUTS_API_KEY}" }\n`;
      toml += "\n";
      core.info(`safeoutputs: using direct connection at port ${safeOutputsPort} with env var reference`);
      continue;
    }

    const url = `${urlPrefix}/mcp/${name}`;
    const headers = /** @type {Record<string, string>} */ value.headers || {};
    const authKey = headers.Authorization || "";
    toml += `[mcp_servers.${name}]\n`;
    toml += `url = "${url}"\n`;
    toml += `http_headers = { Authorization = "${authKey}" }\n`;
    toml += "\n";
  }

  const includedCount = Object.keys(servers).length - [...Object.keys(servers)].filter(k => cliServers.has(k)).length;
  const filteredCount = Object.keys(servers).length - includedCount;
  core.info(`Servers: ${includedCount} included, ${filteredCount} filtered (CLI-mounted)`);

  // Ensure output directory exists
  fs.mkdirSync(path.dirname(OUTPUT_PATH), { recursive: true });

  // Write with owner-only permissions (0o600) to protect the gateway bearer token.
  // An attacker who reads config.toml could issue raw JSON-RPC calls directly
  // to the gateway.
  fs.writeFileSync(OUTPUT_PATH, toml, { mode: 0o600 });

  core.info(`Codex configuration written to ${OUTPUT_PATH}`);
  core.info("");
  core.info("Converted configuration:");
  core.info(toml);
}

main();

module.exports = {};
