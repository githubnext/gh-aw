#!/usr/bin/env node

// This script aggregates usage activity data from various log sources and generates
// a compact summary.json file for the usage artifact.
// usage-activity-summary/v1 structure:
//   firewall: total/allowed/blocked request counters
//   session: aggregate Copilot session event counters
//   gateway: total/failed tool-call counters with per-server breakdown

const fs = require("fs");
const { globSync } = require("node:fs");
const path = require("path");

const SQUID_STATUS_INDEX = 6;
const SQUID_DECISION_INDEX = 7;

/**
 * Check if a Squid decision indicates an allowed request
 */
function isAllowedDecision(decision) {
  const base = decision.split("/")[0].trim().toUpperCase();
  return ["TCP_TUNNEL", "TCP_HIT", "TCP_MISS"].includes(base);
}

/**
 * Parse firewall logs and aggregate request counts
 */
function parseFirewallLogs() {
  const firewall = { total_requests: 0, allowed_requests: 0, blocked_requests: 0 };

  const firewallPaths = ["/tmp/gh-aw/sandbox/firewall/logs/*.log", "/tmp/gh-aw/threat-detection/sandbox/firewall/logs/*.log", "/tmp/gh-aw/squid-logs-*/*.log", "/tmp/gh-aw/threat-detection/squid-logs-*/*.log"];

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

          firewall.total_requests += 1;

          // Squid access log columns (0-based):
          // 0=timestamp 1=client 2=domain 3=dest 4=proto 5=method
          // 6=status 7=decision 8=url 9=user-agent
          // Keep indices named for easier maintenance if format changes.
          const status = parts[SQUID_STATUS_INDEX];
          const decision = parts[SQUID_DECISION_INDEX];

          let allowed = false;
          const code = parseInt(status, 10);
          if (!isNaN(code) && [200, 206, 304].includes(code)) {
            allowed = true;
          }

          if (!allowed && isAllowedDecision(decision)) {
            allowed = true;
          }

          if (allowed) {
            firewall.allowed_requests += 1;
          } else {
            firewall.blocked_requests += 1;
          }
        }
      } catch (err) {
        // Skip files that can't be read
        continue;
      }
    }
  }

  return firewall.total_requests > 0 ? firewall : null;
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

  // Write summary to file
  const outputPath = "/tmp/gh-aw/usage/activity/summary.json";
  fs.writeFileSync(outputPath, JSON.stringify(summary, null, 2), "utf-8");
  console.log(outputPath);
}

// Run main function
if (require.main === module) {
  main();
}

module.exports = { parseFirewallLogs, parseSessionLogs, parseGatewayLogs };
