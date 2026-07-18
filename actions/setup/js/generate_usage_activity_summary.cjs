#!/usr/bin/env node

// This script aggregates usage activity data from various log sources and generates
// a compact summary.json file for the usage artifact.
// usage-activity-summary/v1 structure:
//   firewall: total/allowed/blocked request counters
//   session: aggregate Copilot session event counters
//   gateway: total/failed tool-call counters with per-server breakdown
//   safe_outputs: total item count and per-type breakdown from safe-output-items manifest

const fs = require("fs");
const { globSync } = require("node:fs");
const path = require("path");

const SQUID_STATUS_INDEX = 6;
const SQUID_DECISION_INDEX = 7;
const SQUID_DOMAIN_INDEX = 2;
const SQUID_DEST_INDEX = 3;
const SQUID_CLIENT_INDEX = 1;
const LOCALHOST_CLIENT_PREFIX = "::1:";
const PLACEHOLDER_DOMAIN_KEY = "-";
const PLACEHOLDER_DEST_KEY = "-:-";
const ERROR_DOMAIN_PREFIX = "error:";

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

  // The sandbox firewall logs may be emitted in nested directories (for example,
  // api-proxy-logs/*.log), so these patterns are intentionally recursive.
  const firewallPaths = ["/tmp/gh-aw/sandbox/firewall/logs/**/*.log", "/tmp/gh-aw/threat-detection/sandbox/firewall/logs/**/*.log", "/tmp/gh-aw/squid-logs-*/**/*.log", "/tmp/gh-aw/threat-detection/squid-logs-*/**/*.log"];

  for (const pattern of firewallPaths) {
    const files = globSync(pattern);
    for (const logPath of files) {
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
function parseSessionLogs() {
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

  const sessionPaths = ["/tmp/gh-aw/sandbox/agent/logs/copilot-session-state/*/events.jsonl", "/tmp/gh-aw/threat-detection/sandbox/agent/logs/copilot-session-state/*/events.jsonl"];

  for (const pattern of sessionPaths) {
    const files = globSync(pattern);
    for (const eventsPath of files) {
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
 * @param {string} [manifestPath] - Path to the manifest file (defaults to MANIFEST_FILE_PATH)
 * @returns {{ total_items: number, items_by_type: Record<string, number> } | null}
 */
const MANIFEST_FILE_PATH = "/tmp/gh-aw/safe-output-items.jsonl";

function parseSafeOutputsManifest(manifestPath = MANIFEST_FILE_PATH) {
  if (!fs.existsSync(manifestPath)) {
    return null;
  }

  let content;
  try {
    content = fs.readFileSync(manifestPath, "utf-8");
  } catch (err) {
    return null;
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

  if (totalItems === 0) {
    return null;
  }

  return {
    total_items: totalItems,
    items_by_type: itemsByType,
  };
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

  // Parse safe outputs manifest
  const safeOutputs = parseSafeOutputsManifest();
  if (safeOutputs) {
    summary.safe_outputs = safeOutputs;
  }

  // Write summary to file
  const outputPath = "/tmp/gh-aw/usage/activity/summary.json";
  try {
    fs.writeFileSync(outputPath, JSON.stringify(summary, null, 2), "utf-8");
  } catch (err) {
    throw new Error(`Failed to write file ${outputPath}: ${String(err)}`, { cause: err });
  }
  console.log(outputPath);
}

// Run main function
if (require.main === module) {
  main();
}

module.exports = { parseFirewallLogs, parseSessionLogs, parseGatewayLogs, parseSafeOutputsManifest, MANIFEST_FILE_PATH };
