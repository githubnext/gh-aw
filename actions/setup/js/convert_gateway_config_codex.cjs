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
const { runGatewayConversion } = require("./convert_gateway_config_shared.cjs");

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
  return runGatewayConversion({
    format: "Codex TOML",
    engine: "Codex",
    outputPath: OUTPUT_PATH,
    getUrlPrefix: ({ domain, port }) => {
      if (domain === "host.docker.internal") {
        core.info("Resolving host.docker.internal to gateway IP: 172.30.0.1");
        return `http://172.30.0.1:${port}`;
      }
      return `http://${domain}:${port}`;
    },
    transformServer: (_name, entry) => entry,
    serialize: (servers, _context, urlPrefix) => {
      let toml = '[history]\npersistence = "none"\n\n';
      for (const [name, value] of Object.entries(servers)) {
        toml += toCodexTomlSection(name, value, urlPrefix);
      }
      return toml;
    },
  });
}

if (require.main === module) {
  main();
}

module.exports = { toCodexTomlSection, main };
