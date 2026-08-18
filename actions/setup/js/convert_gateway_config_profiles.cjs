// @ts-check
"use strict";

const path = require("path");
const { normalizeGatewayEntry } = require("./convert_gateway_config_shared.cjs");

/**
 * Resolves the Copilot CLI MCP config output path from the runtime $HOME.
 *
 * @returns {string}
 */
function resolveCopilotConfigOutputPath() {
  const home = process.env.HOME;
  if (!home) {
    throw new Error("HOME environment variable is not set; cannot locate Copilot CLI config directory");
  }
  return path.join(home, ".copilot", "mcp-config.json");
}

/**
 * @param {Record<string, unknown>} entry
 * @param {string} urlPrefix
 * @returns {Record<string, unknown>}
 */
function transformCopilotEntry(entry, urlPrefix) {
  return normalizeGatewayEntry(entry, urlPrefix, transformed => {
    if (!transformed.tools) {
      transformed.tools = ["*"];
    }
  });
}

/**
 * @param {Record<string, unknown>} entry
 * @param {string} urlPrefix
 * @returns {Record<string, unknown>}
 */
function transformClaudeEntry(entry, urlPrefix) {
  return normalizeGatewayEntry(entry, urlPrefix, transformed => {
    transformed.type = "http";
    delete transformed.tools;
  });
}

/**
 * @param {Record<string, unknown>} entry
 * @param {string} urlPrefix
 * @returns {Record<string, unknown>}
 */
function transformGeminiEntry(entry, urlPrefix) {
  return normalizeGatewayEntry(entry, urlPrefix, transformed => {
    delete transformed.type;
  });
}

function getGeminiHostDomain() {
  return process.env.MCP_GATEWAY_HOST_DOMAIN || "localhost";
}

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

const gatewayConversionProfiles = {
  copilot: {
    format: "Copilot",
    engine: "Copilot",
    preRunOutputPath: resolveCopilotConfigOutputPath,
    transformEntry: transformCopilotEntry,
    serialize: servers => JSON.stringify({ mcpServers: servers }, null, 2),
    setFailedOnError: true,
  },
  codex: {
    format: "Codex TOML",
    engine: "Codex",
    preRunOutputPath: () => path.join(process.env.RUNNER_TEMP || "/tmp", "gh-aw/mcp-config/config.toml"),
    getUrlPrefix: ({ domain, port }) => {
      if (domain === "host.docker.internal") {
        return `http://172.30.0.1:${port}`;
      }
      return `http://${domain}:${port}`;
    },
    getUrlPrefixLog: ({ domain }) => (domain === "host.docker.internal" ? "Resolving host.docker.internal to gateway IP: 172.30.0.1" : undefined),
    transformServer: (_name, entry) => entry,
    serialize: (servers, _context, urlPrefix) => {
      let toml = '[history]\npersistence = "none"\n\n';
      for (const [name, value] of Object.entries(servers)) {
        toml += toCodexTomlSection(name, value, urlPrefix);
      }
      return toml;
    },
  },
  claude: {
    format: "Claude",
    engine: "Claude",
    preRunOutputPath: () => path.join(process.env.RUNNER_TEMP || "/tmp", "gh-aw/mcp-config/mcp-servers.json"),
    transformEntry: transformClaudeEntry,
    serialize: servers => JSON.stringify({ mcpServers: servers }, null, 2),
  },
  gemini: {
    format: "Gemini",
    engine: "Gemini",
    contextOptions: { extraRequiredEnv: ["GITHUB_WORKSPACE"] },
    outputPath: ({ extraEnv }) => path.join(extraEnv.GITHUB_WORKSPACE, ".gemini", "settings.json"),
    getTargetDomain: () => getGeminiHostDomain(),
    getUrlPrefix: ({ port }) => `http://${getGeminiHostDomain()}:${port}`,
    transformEntry: transformGeminiEntry,
    serialize: servers =>
      JSON.stringify(
        {
          mcpServers: servers,
          context: { includeDirectories: ["/tmp/"] },
        },
        null,
        2
      ),
  },
};

module.exports = {
  gatewayConversionProfiles,
  resolveCopilotConfigOutputPath,
  transformCopilotEntry,
  transformClaudeEntry,
  transformGeminiEntry,
  getGeminiHostDomain,
  toCodexTomlSection,
};
