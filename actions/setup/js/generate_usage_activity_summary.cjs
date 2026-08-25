#!/usr/bin/env node

// This script aggregates usage activity data from various log sources and generates
// a compact summary.json file for the usage artifact.
// usage-activity-summary/v1 structure:
//   firewall: total/allowed/blocked request counters
//   session: aggregate Copilot session event counters
//   gateway: total/failed tool-call counters with per-server breakdown
//   safe_outputs: total item count and per-type breakdown from safe-output-items manifest
//   experiments: A/B experiment variant assignments for the current run
//   working_set: cumulative input-token traffic relative to peak invocation input

const fs = require("fs");
const path = require("path");
const { readExperimentAssignments } = require("./experiment_helpers.cjs");
const { calculateWorkingSetFromJSONL } = require("./working_set_metrics.cjs");

require("./shim.cjs");

const SQUID_STATUS_INDEX = 6;
const SQUID_DECISION_INDEX = 7;
const SQUID_DOMAIN_INDEX = 2;
const SQUID_DEST_INDEX = 3;
const SQUID_CLIENT_INDEX = 1;
const LOCALHOST_CLIENT_PREFIX = "::1:";
const PLACEHOLDER_DOMAIN_KEY = "-";
const PLACEHOLDER_DEST_KEY = "-:-";
const ERROR_DOMAIN_PREFIX = "error:";
const AGENT_TOKEN_USAGE_PATH = "/tmp/gh-aw/usage/agent/token_usage.jsonl";

function findFiles(rootDir, shouldIncludeFile, maxDepth = Number.POSITIVE_INFINITY, currentDepth = 0) {
  if (!fs.existsSync(rootDir)) {
    return [];
  }

  const files = [];
  let entries;
  try {
    entries = fs.readdirSync(rootDir, { withFileTypes: true });
  } catch {
    return [];
  }

  for (const entry of entries) {
    const entryPath = path.join(rootDir, entry.name);
    if (entry.isDirectory()) {
      if (currentDepth < maxDepth) {
        files.push(...findFiles(entryPath, shouldIncludeFile, maxDepth, currentDepth + 1));
      }
    } else if (entry.isFile() && shouldIncludeFile(entry)) {
      files.push(entryPath);
    }
  }
  return files;
}

function findPrefixedDirectories(parentDir, prefix) {
  if (!fs.existsSync(parentDir)) {
    return [];
  }
  let entries;
  try {
    entries = fs.readdirSync(parentDir, { withFileTypes: true });
  } catch {
    return [];
  }
  return entries.filter(entry => entry.isDirectory() && entry.name.startsWith(prefix)).map(entry => path.join(parentDir, entry.name));
}

/**
 * @param {string} [tokenUsagePath]
 * @returns {{ workingSet: ReturnType<typeof calculateWorkingSetFromJSONL>["workingSet"], ignoredRecords: number }}
 */
function parseWorkingSetMetrics(tokenUsagePath = AGENT_TOKEN_USAGE_PATH) {
  if (!fs.existsSync(tokenUsagePath)) {
    return calculateWorkingSetFromJSONL("");
  }
  try {
    return calculateWorkingSetFromJSONL(fs.readFileSync(tokenUsagePath, "utf-8"));
  } catch (err) {
    throw new Error(`Failed to read working-set token usage from ${tokenUsagePath}: ${String(err)}`, { cause: err });
  }
}

/**
 * Check if a Squid decision indicates an allowed request
 */
function isAllowedDecision(decision) {
  // Squid decision tokens appear in multiple formats (for example
  // TCP_TUNNEL:HIER_DIRECT and TCP_MISS/200), so normalize on the leading verb.
  const base = decision.trim().toUpperCase().split(/[/:]/)[0];
  return ["TCP_TUNNEL", "TCP_HIT", "TCP_MISS"].includes(base);
}

/**
 * Resolve the domain key used in aggregate firewall stats.
 *
 * @param {string} domain
 * @param {string} dest
 * @returns {string}
 */
function getFirewallDomainKey(domain, dest) {
  // Squid can emit either "-" or "-:-" for missing destination fields, so both
  // placeholders are treated as invalid destination keys.
  if (domain !== PLACEHOLDER_DOMAIN_KEY) {
    return domain;
  }
  if (!isPlaceholderFirewallField(dest)) {
    return dest;
  }
  return PLACEHOLDER_DOMAIN_KEY;
}

/**
 * @param {string} value
 * @returns {boolean}
 */
function isPlaceholderFirewallField(value) {
  return value === PLACEHOLDER_DEST_KEY || value === PLACEHOLDER_DOMAIN_KEY;
}

/**
 * @param {string} domain
 * @returns {boolean}
 */
function isValidDomainKey(domain) {
  return domain !== PLACEHOLDER_DOMAIN_KEY && !domain.startsWith(ERROR_DOMAIN_PREFIX);
}

/**
 * @param {string} client
 * @param {string} domain
 * @param {string} dest
 * @returns {boolean}
 */
function isInternalFirewallErrorEntry(client, domain, dest) {
  return client.startsWith(LOCALHOST_CLIENT_PREFIX) && domain === PLACEHOLDER_DOMAIN_KEY && isPlaceholderFirewallField(dest);
}

/**
 * Parse firewall logs and aggregate request counts
 */
function parseFirewallLogs() {
  const firewall = {
    total_requests: 0,
    allowed_requests: 0,
    blocked_requests: 0,
    allowed_domains: new Set(),
    blocked_domains: new Set(),
    requests_by_domain: {},
  };

  const firewallLogDirs = [
    "/tmp/gh-aw/sandbox/firewall/logs",
    "/tmp/gh-aw/threat-detection/sandbox/firewall/logs",
    ...findPrefixedDirectories("/tmp/gh-aw", "squid-logs-"),
    ...findPrefixedDirectories("/tmp/gh-aw/threat-detection", "squid-logs-"),
  ];

  for (const logDir of firewallLogDirs) {
    for (const logPath of findFiles(logDir, entry => entry.name.endsWith(".log"))) {
      try {
        const content = fs.readFileSync(logPath, "utf-8");
        const lines = content.split("\n");

        for (const raw of lines) {
          const line = raw.trim();
          if (!line || line.startsWith("#")) {
            continue;
          }

          const parts = line.split(/\s+/);
          if (parts.length < 8) {
            continue;
          }

          // Skip non-Squid diagnostic lines (WARNING:, DNS, Accepting, etc.) by
          // validating that the first field is a numeric Unix timestamp.
          if (!/^\d+(\.\d+)?$/.test(parts[0])) {
            continue;
          }

          const domain = parts[SQUID_DOMAIN_INDEX];
          const dest = parts[SQUID_DEST_INDEX];
          const client = parts[SQUID_CLIENT_INDEX] || "";
          const isInternalErrorEntry = isInternalFirewallErrorEntry(client, domain, dest);
          if (isInternalErrorEntry) {
            continue;
          }

          // Domain key resolution intentionally considers both domain and dest
          // because Squid may leave domain unset while dest remains usable.
          const domainKey = getFirewallDomainKey(domain, dest);
          // Keep total/allowed/blocked counters aligned with per-domain buckets by
          // excluding unresolved placeholder/error keys from both representations.
          if (!isValidDomainKey(domainKey)) {
            continue;
          }

          firewall.total_requests += 1;

          // Squid access log columns (0-based):
          // 0=timestamp 1=client 2=domain 3=dest 4=proto 5=method
          // 6=status 7=decision 8=url 9=user-agent
          // Keep indices named for easier maintenance if format changes.
          const status = parts[SQUID_STATUS_INDEX];
          const decision = parts[SQUID_DECISION_INDEX];

          let allowed = false;
          const code = parseInt(status, 10);
          if (!Number.isNaN(code) && [200, 206, 304].includes(code)) {
            allowed = true;
          }

          if (!allowed && isAllowedDecision(decision)) {
            allowed = true;
          }

          if (!firewall.requests_by_domain[domainKey]) {
            firewall.requests_by_domain[domainKey] = { allowed: 0, blocked: 0 };
          }

          if (allowed) {
            firewall.allowed_requests += 1;
            firewall.requests_by_domain[domainKey].allowed += 1;
            firewall.allowed_domains.add(domainKey);
          } else {
            firewall.blocked_requests += 1;
            firewall.requests_by_domain[domainKey].blocked += 1;
            firewall.blocked_domains.add(domainKey);
          }
        }
      } catch (err) {
        // Skip files that can't be read
        continue;
      }
    }
  }

  if (firewall.total_requests === 0) {
    return null;
  }

  const requestsByDomain = {};
  for (const [domain, stats] of Object.entries(firewall.requests_by_domain)) {
    if (!isValidDomainKey(domain)) {
      continue;
    }
    requestsByDomain[domain] = stats;
  }

  return {
    total_requests: firewall.total_requests,
    allowed_requests: firewall.allowed_requests,
    blocked_requests: firewall.blocked_requests,
    allowed_domains: Array.from(firewall.allowed_domains).filter(isValidDomainKey).sort(),
    blocked_domains: Array.from(firewall.blocked_domains).filter(isValidDomainKey).sort(),
    requests_by_domain: requestsByDomain,
  };
}

/**
 * Parse Copilot session event logs and aggregate counters
 */
function parseSessionLogs(sessionLogDirs = ["/tmp/gh-aw/sandbox/agent/logs/copilot-session-state", "/tmp/gh-aw/threat-detection/sandbox/agent/logs/copilot-session-state"]) {
  const session = {
    total_events: 0,
    session_starts: 0,
    session_shutdowns: 0,
    turns: 0,
    assistant_messages: 0,
    reasoning_events: 0,
    tool_execution_starts: 0,
    tool_execution_completes: 0,
    failed_tool_executions: 0,
  };

  for (const logDir of sessionLogDirs) {
    for (const eventsPath of findFiles(logDir, entry => entry.name === "events.jsonl", 1)) {
      try {
        const content = fs.readFileSync(eventsPath, "utf-8");
        const lines = content.split("\n");

        for (const raw of lines) {
          const line = raw.trim();
          if (!line || !line.startsWith("{")) {
            continue;
          }

          let entry;
          try {
            entry = JSON.parse(line);
          } catch {
            continue;
          }

          const eventType = String(entry.type || "")
            .trim()
            .toLowerCase();
          session.total_events += 1;

          if (eventType === "session.start") {
            session.session_starts += 1;
          } else if (eventType === "session.shutdown") {
            session.session_shutdowns += 1;
          } else if (eventType === "user.message") {
            session.turns += 1;
          } else if (eventType === "assistant.message") {
            session.assistant_messages += 1;
          }
          // Copilot session logs use both reasoning and assistant.reasoning
          // across CLI/runtime versions, so count both as reasoning events.
          else if (eventType === "reasoning" || eventType === "assistant.reasoning") {
            session.reasoning_events += 1;
          } else if (eventType === "tool.execution_start") {
            session.tool_execution_starts += 1;
          } else if (eventType === "tool.execution_complete") {
            session.tool_execution_completes += 1;
            const data = entry.data || {};
            const success = typeof data === "object" ? data.success !== false : true;
            if (!success) {
              session.failed_tool_executions += 1;
            }
          }
        }
      } catch (err) {
        // Skip files that can't be read
        continue;
      }
    }
  }

  return session.total_events > 0 ? session : null;
}

/**
 * Parse MCP gateway logs and aggregate tool call counts
 */
function parseGatewayLogs() {
  const gateway = { total_calls: 0, failed_calls: 0, servers: {} };
  const gatewayPaths = [];

  const pathPairs = [
    ["/tmp/gh-aw/sandbox/agent/logs/mcp-logs/gateway.jsonl", "/tmp/gh-aw/sandbox/agent/logs/gateway.jsonl"],
    ["/tmp/gh-aw/threat-detection/sandbox/agent/logs/mcp-logs/gateway.jsonl", "/tmp/gh-aw/threat-detection/sandbox/agent/logs/gateway.jsonl"],
  ];

  for (const [modernPath, legacyPath] of pathPairs) {
    if (fs.existsSync(modernPath)) {
      gatewayPaths.push(modernPath);
    } else if (fs.existsSync(legacyPath)) {
      gatewayPaths.push(legacyPath);
    }
  }

  for (const gatewayPath of gatewayPaths) {
    if (!fs.existsSync(gatewayPath)) {
      continue;
    }

    try {
      const content = fs.readFileSync(gatewayPath, "utf-8");
      const lines = content.split("\n");

      for (const raw of lines) {
        const line = raw.trim();
        if (!line || !line.startsWith("{")) {
          continue;
        }

        let entry;
        try {
          entry = JSON.parse(line);
        } catch {
          continue;
        }

        const event = String(entry.event || "")
          .trim()
          .toLowerCase();
        if (!["tool_call", "rpc_call", "request"].includes(event)) {
          continue;
        }

        gateway.total_calls += 1;

        const status = String(entry.status || "")
          .trim()
          .toLowerCase();
        const level = String(entry.level || "")
          .trim()
          .toLowerCase();
        const errorText = String(entry.error || "").trim();
        const failed = status === "error" || errorText !== "" || level === "error";

        if (failed) {
          gateway.failed_calls += 1;
        }

        // gateway.jsonl has server_name for modern logs and server_id in
        // some compatibility/transition paths; keep fallback ordering explicit.
        const serverName = String(entry.server_name || entry.server_id || "unknown");

        if (!gateway.servers[serverName]) {
          gateway.servers[serverName] = { tool_call_count: 0, failed_calls: 0 };
        }

        gateway.servers[serverName].tool_call_count += 1;
        if (failed) {
          gateway.servers[serverName].failed_calls += 1;
        }
      }
    } catch (err) {
      // Skip files that can't be read
      continue;
    }
  }

  if (gateway.total_calls > 0) {
    return {
      total_calls: gateway.total_calls,
      failed_calls: gateway.failed_calls,
      servers: Object.entries(gateway.servers)
        .sort((a, b) => a[0].localeCompare(b[0]))
        .map(([serverName, bucket]) => ({
          server_name: serverName,
          tool_call_count: bucket.tool_call_count,
          failed_calls: bucket.failed_calls,
        })),
    };
  }

  return null;
}

/**
 * Parse the safe-output-items manifest and aggregate item counts by type.
 * Reads the JSONL file written by the safe_outputs job and downloaded into
 * the conclusion job via the safe-outputs-items artifact.
 *
 * Three distinct return states let callers distinguish artifact provenance:
 *   • returns null                          → manifest file not found
 *   • returns { total_items: 0, ... }       → manifest present but contained no loggable items
 *   • returns { total_items: N, ... }       → manifest present with N items
 *   • throws                                → manifest file exists but could not be read
 *
 * @param {string} [manifestPath] - Path to the manifest file (defaults to MANIFEST_FILE_PATH)
 * @returns {{ total_items: number, items_by_type: Record<string, number> } | null}
 */
const MANIFEST_FILE_PATH = "/tmp/gh-aw/safe-output-items.jsonl";

function parseSafeOutputsManifest(manifestPath = MANIFEST_FILE_PATH) {
  if (!fs.existsSync(manifestPath)) {
    return null;
  }

  // Let read errors propagate so the caller can distinguish "unreadable file"
  // from "file present but no items" — both previously collapsed to null.
  let content;
  try {
    content = fs.readFileSync(manifestPath, "utf-8");
  } catch (error) {
    throw new Error(`Failed to read safe output manifest ${manifestPath}`, { cause: error });
  }

  const itemsByType = {};
  let totalItems = 0;

  for (const raw of content.split("\n")) {
    const line = raw.trim();
    if (!line || !line.startsWith("{")) {
      continue;
    }

    let entry;
    try {
      entry = JSON.parse(line);
    } catch {
      continue;
    }

    const itemType = String(entry.type || "").trim();
    if (!itemType) {
      continue;
    }

    totalItems += 1;
    itemsByType[itemType] = (itemsByType[itemType] || 0) + 1;
  }

  return {
    total_items: totalItems,
    items_by_type: itemsByType,
  };
}

/**
 * Parse A/B experiment assignments for the current run.
 * Reads the assignments.json file written by pick_experiment.cjs.
 * Returns null when no experiments are active for this run.
 *
 * @returns {{ assignments: Record<string, string> } | null}
 */
function parseExperimentsData() {
  const assignments = readExperimentAssignments();
  if (!assignments || Object.keys(assignments).length === 0) {
    return null;
  }
  return { assignments };
}

/**
 * Main function to generate usage activity summary
 */
function main() {
  const summary = { schema: "usage-activity-summary/v1" };

  // Parse firewall logs
  const firewall = parseFirewallLogs();
  if (firewall) {
    summary.firewall = firewall;
  }

  // Parse session logs
  const session = parseSessionLogs();
  if (session) {
    summary.session = session;
  }

  // Parse gateway logs
  const gateway = parseGatewayLogs();
  if (gateway) {
    summary.gateway = gateway;
  }

  // Parse safe outputs manifest.
  // parseSafeOutputsManifest() has three distinct outcomes that drive the three
  // states downstream consumers need to distinguish:
  //   • safe_outputs absent        → manifest not found (artifact download failed or job never ran)
  //   • safe_outputs.total_items == 0 → manifest present, no items logged
  //   • safe_outputs.total_items > 0  → manifest present with N items
  // A read error is kept separate: it logs a warning but omits safe_outputs so
  // the consumer cannot mistake a broken artifact for a legitimately empty one.
  try {
    const safeOutputs = parseSafeOutputsManifest();
    if (safeOutputs === null) {
      core.info(`safe-output-items manifest not found at ${MANIFEST_FILE_PATH} — safe-outputs-items artifact may not have been downloaded`);
    } else {
      summary.safe_outputs = safeOutputs;
      if (safeOutputs.total_items === 0) {
        core.info(`safe-output-items manifest: 0 item(s) logged (file present but contained no loggable items)`);
      } else {
        core.info(`safe-output-items manifest: ${safeOutputs.total_items} item(s) logged (types: ${Object.keys(safeOutputs.items_by_type).join(", ")})`);
      }
    }
  } catch (err) {
    core.warning(`safe-output-items manifest could not be read from ${MANIFEST_FILE_PATH}: ${String(err)} — safe_outputs omitted from summary`);
  }

  // Include A/B experiment assignments so the CLI can read them from the usage artifact.
  const experiments = parseExperimentsData();
  if (experiments) {
    summary.experiments = experiments;
  }

  // Compute run-level Working-Set Rebuild Factor after the agent token-usage
  // file has been copied into the compact usage artifact.
  try {
    const { workingSet, ignoredRecords } = parseWorkingSetMetrics();
    summary.working_set = workingSet;
    if (ignoredRecords > 0) {
      core.warning(`Working-set rebuild measurement ignored ${ignoredRecords} malformed or unsupported token-usage record(s)`);
    }
  } catch (err) {
    summary.working_set = calculateWorkingSetFromJSONL("").workingSet;
    core.warning(`Working-set rebuild measurement unavailable: ${String(err)}`);
  }

  // Write summary to file
  const outputPath = "/tmp/gh-aw/usage/activity/summary.json";
  try {
    fs.writeFileSync(outputPath, JSON.stringify(summary, null, 2), "utf-8");
  } catch (err) {
    throw new Error(`Failed to write file ${outputPath}: ${String(err)}`, { cause: err });
  }
  core.info(outputPath);
}

// Run main function
if (require.main === module) {
  main();
}

module.exports = {
  parseFirewallLogs,
  parseSessionLogs,
  parseGatewayLogs,
  parseSafeOutputsManifest,
  parseExperimentsData,
  calculateWorkingSetFromJSONL,
  parseWorkingSetMetrics,
  AGENT_TOKEN_USAGE_PATH,
  MANIFEST_FILE_PATH,
};
