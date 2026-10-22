// @ts-check
"use strict";

const fs = require("fs");
const path = require("path");
const { createHash } = require("crypto");

/**
 * Resolve author-configured CA files on the runner and mount just those files
 * read-only into the gateway. No CA is installed in any global trust store.
 *
 * @param {Record<string, unknown>} config
 * @param {NodeJS.ProcessEnv} [env]
 * @returns {string[]} Docker arguments to insert before the container image
 */
function prepareMCPServerCAs(config, env = process.env) {
  const servers = config.mcpServers;
  if (!servers || typeof servers !== "object") return [];
  const mounts = [];
  for (const [name, server] of Object.entries(servers)) {
    if (!server || typeof server !== "object" || !("caCertFile" in server)) continue;
    const fail = () => new Error(`MCP server '${name}': ca-cert-file must reference a readable PEM file for an HTTPS HTTP server; provision it before gateway startup`);
    if (server.type !== "http" || typeof server.url !== "string" || !server.url.startsWith("https://") || typeof server.caCertFile !== "string" || !server.caCertFile.trim()) {
      throw fail();
    }
    let source = server.caCertFile;
    for (const key of ["GITHUB_WORKSPACE", "RUNNER_TEMP"]) {
      const prefix = `\${${key}}/`;
      if (source.startsWith(prefix)) {
        if (!env[key]) throw fail();
        source = path.join(env[key], source.slice(prefix.length));
      }
    }
    if (!path.isAbsolute(source)) {
      if (!env.GITHUB_WORKSPACE) throw fail();
      source = path.resolve(env.GITHUB_WORKSPACE, source);
    }
    try {
      source = fs.realpathSync(source);
      if (!fs.statSync(source).isFile()) throw fail();
      fs.accessSync(source, fs.constants.R_OK);
    } catch {
      // Do not include filesystem errors or certificate contents in diagnostics.
      throw fail();
    }
    // Docker --mount uses commas as separators even when passed without a shell.
    if (source.includes(",") || /[\r\n]/.test(source)) throw fail();
    const target = `/etc/gh-aw/mcp-ca/${createHash("sha256").update(name).digest("hex")}.pem`;
    mounts.push("--mount", `type=bind,source=${source},target=${target},readonly`);
    server.caCertFile = target;
  }
  return mounts;
}

module.exports = { prepareMCPServerCAs };
