// @ts-check
"use strict";

/**
 * update_network_allowed.cjs
 *
 * Updates the AWF config file's network.allowDomains list based on the
 * GH_AW_WORKFLOW_CALL_NETWORK_ALLOWED environment variable.
 *
 * The variable contains a comma-separated list of ecosystem tokens (e.g. "npm,pip")
 * or raw domain names. Each token is expanded to its known set of domains using the
 * ecosystem map embedded via the GH_AW_ECOSYSTEM_MAP_JSON environment variable.
 * Unknown tokens are treated as raw domain names.
 *
 * Environment variables:
 *   RUNNER_TEMP                       - GitHub Actions runner temp directory
 *   GH_AW_WORKFLOW_CALL_NETWORK_ALLOWED - Comma-separated allowed tokens/domains
 *   GH_AW_ECOSYSTEM_MAP_JSON          - JSON object mapping ecosystem names to domain arrays
 *
 * Exit codes:
 *   0 — Success (including when no tokens are specified)
 *   1 — Fatal error (missing RUNNER_TEMP, unreadable/invalid config file, write failure)
 */

const fs = require("fs");
const path = require("path");

const NETWORK_ALLOWED_ENV_VAR = "GH_AW_WORKFLOW_CALL_NETWORK_ALLOWED";

/**
 * @returns {Promise<void>}
 */
async function main() {
  const runnerTemp = process.env.RUNNER_TEMP;
  if (!runnerTemp) {
    process.stderr.write("RUNNER_TEMP is not set\n");
    process.exit(1);
  }

  const configPath = path.join(runnerTemp, "gh-aw", "awf-config.json");

  /** @type {Record<string, unknown>} */
  let config;
  try {
    config = JSON.parse(fs.readFileSync(configPath, "utf8"));
  } catch (/** @type {unknown} */ err) {
    const e = /** @type {NodeJS.ErrnoException} */ err;
    if (e.code === "ENOENT") {
      process.stderr.write(`Missing AWF config file at ${configPath}\n`);
    } else if (err instanceof SyntaxError) {
      process.stderr.write(`Invalid AWF config JSON at ${configPath}: ${e.message}\n`);
    } else {
      process.stderr.write(`Failed to read AWF config file at ${configPath}: ${e.message}\n`);
    }
    process.exit(1);
  }

  const networkAllowed = process.env[NETWORK_ALLOWED_ENV_VAR] || "";
  const tokens = networkAllowed
    .split(",")
    .map(t => t.trim())
    .filter(t => t.length > 0);

  if (tokens.length > 0) {
    const ecosystemMapJSON = process.env.GH_AW_ECOSYSTEM_MAP_JSON;
    if (!ecosystemMapJSON) {
      process.stderr.write("GH_AW_ECOSYSTEM_MAP_JSON is not set\n");
      process.exit(1);
    }

    /** @type {Record<string, string[]>} */
    const ecosystemMap = JSON.parse(ecosystemMapJSON);

    if (!config.network || typeof config.network !== "object") {
      config.network = {};
    }
    const network = /** @type {Record<string, unknown>} */ config.network;
    if (!Array.isArray(network.allowDomains)) {
      network.allowDomains = [];
    }
    const allowDomains = /** @type {string[]} */ network.allowDomains;
    const seen = new Set(allowDomains);

    for (const token of tokens) {
      const domains = ecosystemMap[token] || [token];
      for (const domain of domains) {
        if (!seen.has(domain)) {
          allowDomains.push(domain);
          seen.add(domain);
        }
      }
    }
  }

  try {
    fs.writeFileSync(configPath, JSON.stringify(config) + "\n");
  } catch (/** @type {unknown} */ err) {
    const e = /** @type {NodeJS.ErrnoException} */ err;
    process.stderr.write(`Failed to write AWF config file at ${configPath}: ${e.message}\n`);
    process.exit(1);
  }
}

module.exports = { main };

if (require.main === module) {
  main().catch((/** @type {unknown} */ err) => {
    const e = /** @type {Error} */ err;
    process.stderr.write(`Error: ${e.message}\n`);
    process.exit(1);
  });
}
