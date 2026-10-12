// @ts-check

const fs = require("fs");
const path = require("path");
const { writeSessionArtifact } = require("./session_artifact.cjs");
const { lstatGuard } = require("./symlink_guard.cjs");
const { ERR_SYSTEM, ERR_VALIDATION } = require("./error_codes.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Use an allowlist, not credential-pattern matching: URLs, commands, arguments,
 * environment, headers, policies, and unknown extensions can all carry secrets.
 * @param {any} data
 * @returns {import("./types/unified_session").McpConfigurationData}
 */
function normalizeMcpConfigurationData(data) {
  if (!Array.isArray(data?.servers)) {
    throw new Error(`${ERR_VALIDATION}: MCP configuration snapshot must contain a servers array`);
  }
  return {
    servers: data.servers.flatMap(server => {
      if (!server || typeof server !== "object" || typeof server.serverName !== "string") return [];
      const transport = server.transport === "local" ? "stdio" : server.transport;
      return [
        {
          serverName: server.serverName,
          ...(["stdio", "http", "sse", "streamable-http"].includes(transport) ? { transport } : {}),
          ...(typeof server.required === "boolean" ? { required: server.required } : {}),
          ...(Array.isArray(server.tools) ? { tools: server.tools.filter(tool => typeof tool === "string") } : {}),
        },
      ];
    }),
  };
}

/**
 * Check every existing path component, not only the leaf directory.
 * @param {string} directory
 * @returns {boolean}
 */
function hasSymlinkPathComponent(directory) {
  const absolutePath = path.resolve(directory);
  const { root } = path.parse(absolutePath);
  let current = root;
  for (const component of absolutePath.slice(root.length).split(path.sep).filter(Boolean)) {
    current = path.join(current, component);
    try {
      if (lstatGuard(current) === null) return true;
    } catch (error) {
      if (error && typeof error === "object" && "code" in error && error.code === "ENOENT") return false;
      throw error;
    }
  }
  return false;
}

/**
 * Persist only the sanitized inventory before attempting gateway startup.
 * @param {string} logDirectory
 * @param {Record<string, unknown>} config
 */
function writeMcpConfiguration(logDirectory, config) {
  const servers = config.mcpServers;
  if (!servers || typeof servers !== "object" || Array.isArray(servers)) {
    throw new Error(`${ERR_VALIDATION}: MCP configuration must contain an mcpServers object`);
  }
  const data = normalizeMcpConfigurationData({
    servers: Object.entries(servers).map(([serverName, server]) => ({
      serverName,
      transport: server?.type ?? (typeof server?.command === "string" || typeof server?.container === "string" ? "stdio" : undefined),
      required: server?.required,
      tools: server?.tools,
    })),
  });
  if (hasSymlinkPathComponent(logDirectory)) {
    throw new Error(`${ERR_VALIDATION}: MCP configuration log directory must not be a symlink`);
  }
  try {
    fs.mkdirSync(logDirectory, { recursive: true });
  } catch (error) {
    throw new Error(`${ERR_SYSTEM}: Failed to create MCP configuration log directory: ${getErrorMessage(error)}`, { cause: error });
  }
  if (hasSymlinkPathComponent(logDirectory) || !lstatGuard(logDirectory)?.isDirectory()) {
    throw new Error(`${ERR_VALIDATION}: MCP configuration log directory must not be a symlink`);
  }
  writeSessionArtifact(path.join(logDirectory, "configuration.jsonl"), [{ type: "mcp.configuration", timestamp: new Date().toISOString(), data: { ...data } }]);
}

module.exports = { normalizeMcpConfigurationData, writeMcpConfiguration };
