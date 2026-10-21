// @ts-check
"use strict";

const fs = require("fs");
const { isIP } = require("net");

/**
 * Translate AWF domain rules into Copilot URL patterns. AWF includes subdomains
 * and both protocols unless restricted; Copilot requires separate approvals.
 * @param {unknown} domains
 * @param {{block?: boolean}} options
 * @returns {string[]}
 */
function domainURLPatterns(domains, { block = false } = {}) {
  if (domains === undefined) return [];
  if (!Array.isArray(domains)) throw new Error("AWF network domains must be an array");
  const patterns = new Set();
  for (const domain of domains) {
    if (typeof domain !== "string" || !domain.trim()) throw new Error("AWF network domain must be a non-empty string");
    let authority = domain.trim().toLowerCase().replace(/\/$/, "");
    const protocols = [];
    if (authority.startsWith("https://")) {
      protocols.push("https");
      authority = authority.slice(8);
    } else if (authority.startsWith("http://")) {
      protocols.push("http");
      authority = authority.slice(7);
    } else {
      protocols.push("http", "https");
    }
    // A global wildcard must never become an unrestricted URL approval.
    if (authority === "*") {
      if (block) throw new Error("Global AWF domain blocks cannot be translated to scoped Copilot URL permissions");
      continue;
    }
    let host = authority;
    let port = "";
    if (host.startsWith("[")) {
      const match = /^(\[[^\]]+\])(?::([0-9]+))?$/.exec(host);
      if (!match || isIP(match[1].slice(1, -1)) !== 6) throw new Error("Invalid AWF IPv6 domain rule");
      host = match[1];
      port = match[2] || "";
    } else if (isIP(host) === 6) {
      host = `[${host}]`;
    } else {
      const match = /^([^:]+)(?::([0-9]+))?$/.exec(host);
      if (!match) throw new Error("Invalid AWF domain rule");
      host = match[1].replace(/^\*\./, "").replace(/\.$/, "");
      port = match[2] || "";
      if (!host.split(".").every(label => /^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$/.test(label))) {
        throw new Error("Invalid AWF domain hostname");
      }
    }
    if (port && (Number(port) < 1 || Number(port) > 65535)) throw new Error("Invalid AWF domain port");
    const hosts = host.startsWith("[") || isIP(host) ? [host] : [host, `*.${host}`];
    for (const protocol of protocols) {
      for (const hostname of hosts) {
        patterns.add(`${protocol}://${hostname}${port ? `:${port}` : ""}`);
      }
    }
  }
  return [...patterns];
}

/**
 * The compiler copies the finalized config after expanding network_allowed.
 * Read it at startup, not at compilation, without changing shell permissions.
 * @param {string[]} args
 * @param {{env?: NodeJS.ProcessEnv, sdkMode?: boolean, configPath: string}} options
 * @returns {string[]}
 */
function addCopilotURLPermissions(args, { env = process.env, sdkMode = false, configPath }) {
  if (sdkMode || env.AWF_REFLECT_ENABLED !== "1") return args;
  let config;
  try {
    config = JSON.parse(fs.readFileSync(configPath, "utf8"));
  } catch {
    throw new Error("Cannot read AWF config for Copilot URL approvals; check the generated awf-config.json");
  }
  if (!config || typeof config !== "object" || Array.isArray(config)) throw new Error("Invalid AWF config for Copilot URL approvals");
  const network = config.network;
  if (network === undefined) return args;
  if (!network || typeof network !== "object" || Array.isArray(network)) throw new Error("Invalid AWF network config for Copilot URL approvals");
  const permissions = [];
  for (const pattern of domainURLPatterns(network.allowDomains)) permissions.push("--allow-url", pattern);
  for (const pattern of domainURLPatterns(network.blockDomains, { block: true })) permissions.push("--deny-url", pattern);
  // Keep any explicit CLI argument delimiter in its original role.
  const delimiter = args.indexOf("--");
  const split = delimiter < 0 ? args.length : delimiter;
  return [...args.slice(0, split), ...permissions, ...args.slice(split)];
}

module.exports = { domainURLPatterns, addCopilotURLPermissions };
