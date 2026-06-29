// @ts-check
/// <reference types="@actions/github-script" />

/**
 * unified_timeline.cjs - Unified event timeline for GITHUB_STEP_SUMMARY.
 *
 * Collects JSONL events from three sources:
 *   - MCP Gateway:  gateway.jsonl (or rpc-messages.jsonl as fallback)
 *   - AWF Firewall: audit.jsonl
 *   - Agent:        events.jsonl (Copilot CLI session events)
 *
 * All events are normalised to a common schema, merged into a single
 * chronologically sorted stream, and rendered as a GitHub-flavoured Markdown
 * `<details>` table suitable for inclusion in GITHUB_STEP_SUMMARY.
 *
 * Path constants mirror the values in constants.cjs and the Go package.
 */

const fs = require("fs");
const path = require("path");

// ---------------------------------------------------------------------------
// Path constants
// ---------------------------------------------------------------------------

const TMP_GH_AW = "/tmp/gh-aw";
const GATEWAY_JSONL_PATH = `${TMP_GH_AW}/mcp-logs/gateway.jsonl`;
const RPC_MESSAGES_PATH = `${TMP_GH_AW}/mcp-logs/rpc-messages.jsonl`;
const FIREWALL_AUDIT_PATH = `${TMP_GH_AW}/sandbox/firewall/audit/audit.jsonl`;

// ---------------------------------------------------------------------------
// Event-source and event-kind constants (mirror Go constants)
// ---------------------------------------------------------------------------

const SOURCE_GATEWAY = "gateway";
const SOURCE_FIREWALL = "firewall";
const SOURCE_AGENT = "agent";

const KIND_TOOL_CALL = "tool_call";
const KIND_DIFC_FILTERED = "difc_filtered";
const KIND_GUARD_BLOCKED = "guard_blocked";
const KIND_NET_ALLOWED = "net_allowed";
const KIND_NET_BLOCKED = "net_blocked";
const KIND_AGENT_TURN = "agent_turn";
const KIND_AGENT_TOOL_START = "agent_tool_start";
const KIND_AGENT_TOOL_DONE = "agent_tool_done";

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

/**
 * Parses a JSONL string into an array of objects, skipping blank/malformed lines.
 * @param {string} content
 * @returns {object[]}
 */
function parseJsonl(content) {
  const results = [];
  for (const raw of content.split("\n")) {
    const line = raw.trim();
    if (!line || !line.startsWith("{")) continue;
    try {
      results.push(JSON.parse(line));
    } catch {
      // skip malformed lines
    }
  }
  return results;
}

/**
 * Tries to parse a timestamp string as a Date.  Returns null when unparseable.
 * @param {string|number|undefined} value
 * @returns {Date|null}
 */
function parseTimestamp(value) {
  if (value == null) return null;
  if (typeof value === "number") {
    // Unix float seconds (AWF firewall audit.jsonl uses this)
    const ms = Math.round(value * 1000);
    const d = new Date(ms);
    return isNaN(d.getTime()) ? null : d;
  }
  const d = new Date(String(value));
  return isNaN(d.getTime()) ? null : d;
}

/**
 * Formats a Date as HH:MM:SS.mmm (UTC).
 * @param {Date} d
 * @returns {string}
 */
function formatTime(d) {
  const hh = String(d.getUTCHours()).padStart(2, "0");
  const mm = String(d.getUTCMinutes()).padStart(2, "0");
  const ss = String(d.getUTCSeconds()).padStart(2, "0");
  const ms = String(d.getUTCMilliseconds()).padStart(3, "0");
  return `${hh}:${mm}:${ss}.${ms}`;
}

/**
 * Truncates a string to maxLen characters, appending "…" when trimmed.
 * @param {string} s
 * @param {number} maxLen
 * @returns {string}
 */
function truncate(s, maxLen) {
  if (!s) return "";
  if (s.length <= maxLen) return s;
  return s.slice(0, maxLen - 1) + "…";
}

/**
 * Escapes pipe characters in a table cell value so they don't break Markdown tables.
 * @param {string} s
 * @returns {string}
 */
function escMd(s) {
  return String(s ?? "").replace(/\|/g, "&#124;");
}

// ---------------------------------------------------------------------------
// Source label + icon helpers (mirror Go render functions)
// ---------------------------------------------------------------------------

/**
 * @param {string} source
 * @returns {string}
 */
function sourceLabel(source) {
  switch (source) {
    case SOURCE_GATEWAY:
      return "GW";
    case SOURCE_FIREWALL:
      return "FW";
    case SOURCE_AGENT:
      return "AG";
    default:
      return source.length < 2 ? source.toUpperCase() : source.slice(0, 2).toUpperCase();
  }
}

/**
 * @param {string} kind
 * @returns {string}
 */
function eventIcon(kind) {
  switch (kind) {
    case KIND_TOOL_CALL:
      return "🔧";
    case KIND_DIFC_FILTERED:
      return "🚫";
    case KIND_GUARD_BLOCKED:
      return "🛡";
    case KIND_NET_ALLOWED:
      return "✓";
    case KIND_NET_BLOCKED:
      return "✗";
    case KIND_AGENT_TURN:
      return "💬";
    case KIND_AGENT_TOOL_START:
      return "▶";
    case KIND_AGENT_TOOL_DONE:
      return "■";
    default:
      return "·";
  }
}

/**
 * @param {string} kind
 * @returns {string}
 */
function kindLabel(kind) {
  switch (kind) {
    case KIND_TOOL_CALL:
      return "tool_call";
    case KIND_DIFC_FILTERED:
      return "difc_filtered";
    case KIND_GUARD_BLOCKED:
      return "guard_blocked";
    case KIND_NET_ALLOWED:
      return "net_allowed";
    case KIND_NET_BLOCKED:
      return "net_blocked";
    case KIND_AGENT_TURN:
      return "agent_turn";
    case KIND_AGENT_TOOL_START:
      return "tool_start";
    case KIND_AGENT_TOOL_DONE:
      return "tool_done";
    default:
      return kind;
  }
}

// ---------------------------------------------------------------------------
// Per-source collectors
// ---------------------------------------------------------------------------

/**
 * Reads gateway.jsonl (or rpc-messages.jsonl as fallback) and returns timeline events.
 * @param {{gatewayJsonlPath?: string, rpcMessagesPath?: string}} [opts]
 * @returns {{source: string, kind: string, time: Date, detail: string, status: string}[]}
 */
function collectGatewayEvents(opts = {}) {
  const gwPath = opts.gatewayJsonlPath ?? GATEWAY_JSONL_PATH;
  const rpcPath = opts.rpcMessagesPath ?? RPC_MESSAGES_PATH;

  let content = "";
  let isRpc = false;

  if (fs.existsSync(gwPath)) {
    content = fs.readFileSync(gwPath, "utf8");
  } else if (fs.existsSync(rpcPath)) {
    content = fs.readFileSync(rpcPath, "utf8");
    isRpc = true;
  } else {
    return [];
  }

  const events = [];
  for (const entry of parseJsonl(content)) {
    const time = parseTimestamp(entry.timestamp);
    if (!time) continue;

    if (isRpc) {
      // rpc-messages.jsonl: DIFC_FILTERED and REQUEST/OUT are the timeline events
      if (entry.type === "DIFC_FILTERED") {
        events.push({
          source: SOURCE_GATEWAY,
          kind: KIND_DIFC_FILTERED,
          time,
          detail: truncate(`${entry.server_id ?? ""}/${entry.tool_name ?? ""}`.replace(/^\//, ""), 48),
          status: entry.reason ?? "",
        });
      } else if (entry.direction === "OUT" && entry.type === "REQUEST") {
        events.push({
          source: SOURCE_GATEWAY,
          kind: KIND_TOOL_CALL,
          time,
          detail: truncate(entry.method ?? "", 48),
          status: "",
        });
      }
    } else {
      // gateway.jsonl: structured entries with type/event fields
      if (entry.type === "DIFC_FILTERED") {
        const server = entry.server_id ?? entry.server_name ?? "";
        const tool = entry.tool_name ?? "";
        events.push({
          source: SOURCE_GATEWAY,
          kind: KIND_DIFC_FILTERED,
          time,
          detail: truncate(server ? `${server}/${tool}` : tool, 48),
          status: entry.reason ?? "",
        });
      } else if (entry.type === "GUARD_POLICY_BLOCKED") {
        const server = entry.server_id ?? entry.server_name ?? "";
        const tool = entry.tool_name ?? "";
        events.push({
          source: SOURCE_GATEWAY,
          kind: KIND_GUARD_BLOCKED,
          time,
          detail: truncate(server ? `${server}/${tool}` : tool, 48),
          status: entry.reason ?? entry.message ?? "",
        });
      } else if (["tool_call", "rpc_call", "request"].includes(entry.event)) {
        const server = entry.server_name ?? "";
        const tool = entry.tool_name ?? "";
        const dur = entry.duration ? `${Math.round(entry.duration)}ms` : "";
        const statusStr = entry.status || (entry.error || entry.level === "error" ? "error" : "success");
        events.push({
          source: SOURCE_GATEWAY,
          kind: KIND_TOOL_CALL,
          time,
          detail: truncate(server ? `${server}/${tool}` : tool, 48),
          status: dur ? `${statusStr} (${dur})` : statusStr,
        });
      }
    }
  }

  return events;
}

/**
 * Reads audit.jsonl from the AWF firewall and returns timeline events.
 * @param {{auditJsonlPath?: string}} [opts]
 * @returns {{source: string, kind: string, time: Date, detail: string, status: string}[]}
 */
function collectFirewallEvents(opts = {}) {
  const auditPath = opts.auditJsonlPath ?? FIREWALL_AUDIT_PATH;

  if (!fs.existsSync(auditPath)) return [];

  const content = fs.readFileSync(auditPath, "utf8");
  const events = [];

  for (const entry of parseJsonl(content)) {
    // Firewall audit.jsonl uses Unix float64 `ts` field
    const time = parseTimestamp(entry.ts);
    if (!time) continue;

    // Skip benign Squid operational entries (mirrors Go auditEntryToTimelineEvent filter).
    // The audit.jsonl field name is "url" (lowercase), matching Go's json:"url" tag.
    const url = entry.url ?? "";
    if (url === "error:transaction-end-before-headers") continue;

    const host = entry.host ?? entry.domain ?? "";
    // Skip entries with no host information (mirrors Go auditEntryToTimelineEvent filter).
    if (!host || host === "-") continue;

    const method = entry.method ?? "";
    const status = entry.status ?? entry.http_status ?? "";

    // Decision field: "TCP_TUNNEL:HIER_DIRECT" or "TCP_DENIED:..." etc.
    const decision = entry.decision ?? entry.squid_request_status ?? "";
    const blocked = /denied|blocked|reject/i.test(decision) || (typeof status === "number" && status >= 400 && status < 600);

    const detail = truncate([host, method].filter(Boolean).join(" "), 48);
    const statusStr = status ? String(status) : blocked ? "blocked" : "allowed";

    events.push({
      source: SOURCE_FIREWALL,
      kind: blocked ? KIND_NET_BLOCKED : KIND_NET_ALLOWED,
      time,
      detail,
      status: statusStr,
    });
  }

  return events;
}

/**
 * Searches logDir recursively for an events.jsonl file, mirroring findEventsJSONLFile in Go.
 * @param {string} logDir
 * @returns {string|null}
 */
function findEventsJsonlFile(logDir) {
  const sessionStateDir = path.join(logDir, "sandbox", "agent", "logs", "copilot-session-state");
  if (!fs.existsSync(sessionStateDir)) return null;

  // Search one level deep: each entry is a UUID-named session directory
  try {
    const entries = fs.readdirSync(sessionStateDir, { withFileTypes: true });
    for (const entry of entries) {
      if (!entry.isDirectory()) continue;
      const candidate = path.join(sessionStateDir, entry.name, "events.jsonl");
      if (fs.existsSync(candidate)) return candidate;
    }
  } catch {
    // ignore read errors
  }
  return null;
}

/**
 * Reads events.jsonl from the agent session directory and returns timeline events.
 * @param {{eventsJsonlPath?: string, logDir?: string}} [opts]
 * @returns {{source: string, kind: string, time: Date, detail: string, status: string}[]}
 */
function collectAgentEvents(opts = {}) {
  let eventsPath = opts.eventsJsonlPath;
  if (!eventsPath && opts.logDir) {
    eventsPath = findEventsJsonlFile(opts.logDir) ?? undefined;
  }
  if (!eventsPath) {
    // Default: search the canonical TMP_GH_AW directory
    eventsPath = findEventsJsonlFile(TMP_GH_AW) ?? undefined;
  }
  if (!eventsPath || !fs.existsSync(eventsPath)) return [];

  const content = fs.readFileSync(eventsPath, "utf8");
  const events = [];
  let turnIndex = 0;

  for (const entry of parseJsonl(content)) {
    const time = parseTimestamp(entry.timestamp);
    if (!time) continue;

    switch (entry.type) {
      case "user.message":
        turnIndex++;
        events.push({
          source: SOURCE_AGENT,
          kind: KIND_AGENT_TURN,
          time,
          detail: `turn ${turnIndex}`,
          status: "",
        });
        break;

      case "tool.execution_start": {
        const server = entry.data?.mcpServerName ?? "";
        const tool = entry.data?.toolName ?? "";
        events.push({
          source: SOURCE_AGENT,
          kind: KIND_AGENT_TOOL_START,
          time,
          detail: truncate(server ? `${server}/${tool}` : tool, 48),
          status: "",
        });
        break;
      }

      case "tool.execution_complete": {
        const server = entry.data?.mcpServerName ?? "";
        const tool = entry.data?.toolName ?? "";
        const success = entry.data?.success;
        events.push({
          source: SOURCE_AGENT,
          kind: KIND_AGENT_TOOL_DONE,
          time,
          detail: truncate(server ? `${server}/${tool}` : tool, 48),
          status: success ? "success" : "error",
        });
        break;
      }

      default:
        break;
    }
  }

  return events;
}

// ---------------------------------------------------------------------------
// Merge + sort
// ---------------------------------------------------------------------------

/**
 * Collects events from all three sources, merges them, and sorts by time ascending.
 * @param {{gatewayJsonlPath?: string, rpcMessagesPath?: string, auditJsonlPath?: string, eventsJsonlPath?: string, logDir?: string}} [opts]
 * @returns {{source: string, kind: string, time: Date, detail: string, status: string}[]}
 */
function collectUnifiedTimelineEvents(opts = {}) {
  const gateway = collectGatewayEvents(opts);
  const firewall = collectFirewallEvents(opts);
  const agent = collectAgentEvents(opts);

  const all = [...gateway, ...firewall, ...agent];
  all.sort((a, b) => a.time.getTime() - b.time.getTime());
  return all;
}

// ---------------------------------------------------------------------------
// Markdown renderer
// ---------------------------------------------------------------------------

/**
 * Renders a unified timeline event list as a GitHub-flavoured Markdown
 * `<details>` table.  Returns an empty string when events is empty.
 * @param {{source: string, kind: string, time: Date, detail: string, status: string}[]} events
 * @returns {string}
 */
function buildUnifiedTimelineMarkdown(events) {
  if (!events || events.length === 0) return "";

  // Tally events per source and per kind in a single pass.
  // NOTE: byKind is a global tally. The gateway / firewall / agent stats
  // lines each assume their KIND_* constants do not appear in other sources.
  // If you add a new kind, ensure it is unique across all collectors.
  /** @type {Record<string, number>} */
  const bySource = {};
  /** @type {Record<string, number>} */
  const byKind = {};
  for (const { source, kind } of events) {
    bySource[source] = (bySource[source] ?? 0) + 1;
    byKind[kind] = (byKind[kind] ?? 0) + 1;
  }

  const gwCount = bySource[SOURCE_GATEWAY] ?? 0;
  const fwCount = bySource[SOURCE_FIREWALL] ?? 0;
  const agCount = bySource[SOURCE_AGENT] ?? 0;

  const summaryParts = [`${events.length} events`, ...(gwCount > 0 ? [`GW:${gwCount}`] : []), ...(fwCount > 0 ? [`FW:${fwCount}`] : []), ...(agCount > 0 ? [`AG:${agCount}`] : [])];

  const lines = [];

  lines.push(`<details>`);
  lines.push(`<summary>🕒 Unified Event Timeline — ${summaryParts.join(" · ")}</summary>`);
  lines.push(``);
  lines.push(`**Total Events:** ${events.length}`);
  if (gwCount > 0) {
    lines.push(`**Gateway (GW):** ${gwCount} — tool_calls=${byKind[KIND_TOOL_CALL] ?? 0}, difc_filtered=${byKind[KIND_DIFC_FILTERED] ?? 0}, guard_blocked=${byKind[KIND_GUARD_BLOCKED] ?? 0}`);
  }
  if (fwCount > 0) {
    lines.push(`**Firewall (FW):** ${fwCount} — allowed=${byKind[KIND_NET_ALLOWED] ?? 0}, blocked=${byKind[KIND_NET_BLOCKED] ?? 0}`);
  }
  if (agCount > 0) {
    lines.push(`**Agent (AG):** ${agCount} — turns=${byKind[KIND_AGENT_TURN] ?? 0}, tool_start=${byKind[KIND_AGENT_TOOL_START] ?? 0}, tool_done=${byKind[KIND_AGENT_TOOL_DONE] ?? 0}`);
  }
  lines.push(``);
  lines.push(`| Time | Src | Kind | Detail | Status |`);
  lines.push(`| --- | --- | --- | --- | --- |`);

  for (const evt of events) {
    const time = formatTime(evt.time);
    const src = sourceLabel(evt.source);
    const kind = `${eventIcon(evt.kind)} ${kindLabel(evt.kind)}`;
    const detail = escMd(evt.detail);
    const status = escMd(evt.status);
    lines.push(`| ${time} | ${src} | ${kind} | ${detail} | ${status} |`);
  }

  lines.push(``);
  lines.push(`</details>`);
  lines.push(``);

  return lines.join("\n");
}

/**
 * Collects events from all sources and returns the rendered Markdown.
 * Returns an empty string when no events are found.
 * @param {{gatewayJsonlPath?: string, rpcMessagesPath?: string, auditJsonlPath?: string, eventsJsonlPath?: string, logDir?: string}} [opts]
 * @returns {string}
 */
function generateUnifiedTimelineSummary(opts = {}) {
  const events = collectUnifiedTimelineEvents(opts);
  return buildUnifiedTimelineMarkdown(events);
}

// ---------------------------------------------------------------------------
// Module exports
// ---------------------------------------------------------------------------

if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    parseJsonl,
    parseTimestamp,
    formatTime,
    truncate,
    sourceLabel,
    eventIcon,
    kindLabel,
    collectGatewayEvents,
    collectFirewallEvents,
    collectAgentEvents,
    findEventsJsonlFile,
    collectUnifiedTimelineEvents,
    buildUnifiedTimelineMarkdown,
    generateUnifiedTimelineSummary,
    // Source constants
    SOURCE_GATEWAY,
    SOURCE_FIREWALL,
    SOURCE_AGENT,
    // Kind constants
    KIND_TOOL_CALL,
    KIND_DIFC_FILTERED,
    KIND_GUARD_BLOCKED,
    KIND_NET_ALLOWED,
    KIND_NET_BLOCKED,
    KIND_AGENT_TURN,
    KIND_AGENT_TOOL_START,
    KIND_AGENT_TOOL_DONE,
  };
}
