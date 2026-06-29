// @ts-check
import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { createRequire } from "module";
import * as fs from "fs";
import * as os from "os";
import * as path from "path";

const req = createRequire(import.meta.url);
const {
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
  SOURCE_GATEWAY,
  SOURCE_FIREWALL,
  SOURCE_AGENT,
  KIND_TOOL_CALL,
  KIND_DIFC_FILTERED,
  KIND_GUARD_BLOCKED,
  KIND_NET_ALLOWED,
  KIND_NET_BLOCKED,
  KIND_AGENT_TURN,
  KIND_AGENT_TOOL_START,
  KIND_AGENT_TOOL_DONE,
} = req("./unified_timeline.cjs");

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

/** @type {string} */
let tmpDir;

beforeEach(() => {
  tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "timeline-test-"));
});

afterEach(() => {
  fs.rmSync(tmpDir, { recursive: true, force: true });
});

/**
 * Writes a JSONL file to the given path.
 * @param {string} filePath
 * @param {object[]} objects
 */
function writeJsonl(filePath, objects) {
  const dir = path.dirname(filePath);
  fs.mkdirSync(dir, { recursive: true });
  fs.writeFileSync(filePath, objects.map(o => JSON.stringify(o)).join("\n") + "\n", "utf8");
}

// ---------------------------------------------------------------------------
// parseJsonl
// ---------------------------------------------------------------------------

describe("parseJsonl", () => {
  it("parses valid JSON objects", () => {
    const content = '{"a":1}\n{"b":2}\n';
    const result = parseJsonl(content);
    expect(result).toHaveLength(2);
    expect(result[0]).toEqual({ a: 1 });
  });

  it("skips blank lines", () => {
    const content = '\n{"a":1}\n\n{"b":2}\n';
    expect(parseJsonl(content)).toHaveLength(2);
  });

  it("skips lines not starting with {", () => {
    const content = 'not json\n{"a":1}\n[1,2,3]\n';
    expect(parseJsonl(content)).toHaveLength(1);
  });

  it("skips malformed JSON", () => {
    const content = '{bad json}\n{"a":1}\n';
    expect(parseJsonl(content)).toHaveLength(1);
  });

  it("returns empty array for empty string", () => {
    expect(parseJsonl("")).toHaveLength(0);
  });
});

// ---------------------------------------------------------------------------
// parseTimestamp
// ---------------------------------------------------------------------------

describe("parseTimestamp", () => {
  it("parses RFC3339 string", () => {
    const d = parseTimestamp("2024-01-15T10:00:00Z");
    expect(d).toBeInstanceOf(Date);
    expect(d.getUTCHours()).toBe(10);
  });

  it("parses Unix float64 seconds (firewall format)", () => {
    const d = parseTimestamp(1705312800.5);
    expect(d).toBeInstanceOf(Date);
    expect(d.getTime()).toBe(Math.round(1705312800.5 * 1000));
  });

  it("returns null for null", () => {
    expect(parseTimestamp(null)).toBeNull();
  });

  it("returns null for undefined", () => {
    expect(parseTimestamp(undefined)).toBeNull();
  });

  it("returns null for invalid string", () => {
    expect(parseTimestamp("not-a-date")).toBeNull();
  });
});

// ---------------------------------------------------------------------------
// formatTime
// ---------------------------------------------------------------------------

describe("formatTime", () => {
  it("formats a UTC date as HH:MM:SS.mmm", () => {
    const d = new Date("2024-01-15T10:05:03.042Z");
    expect(formatTime(d)).toBe("10:05:03.042");
  });

  it("zero-pads hours, minutes, seconds, and milliseconds", () => {
    const d = new Date("2024-01-15T01:02:03.004Z");
    expect(formatTime(d)).toBe("01:02:03.004");
  });
});

// ---------------------------------------------------------------------------
// truncate
// ---------------------------------------------------------------------------

describe("truncate", () => {
  it("returns the string unchanged when shorter than maxLen", () => {
    expect(truncate("hello", 10)).toBe("hello");
  });

  it("returns the string unchanged when equal to maxLen", () => {
    expect(truncate("hello", 5)).toBe("hello");
  });

  it("truncates and appends ellipsis when longer than maxLen", () => {
    const result = truncate("hello world", 8);
    expect(result).toHaveLength(8);
    expect(result.endsWith("…")).toBe(true);
  });

  it("returns empty string for empty input", () => {
    expect(truncate("", 10)).toBe("");
  });
});

// ---------------------------------------------------------------------------
// sourceLabel / eventIcon / kindLabel
// ---------------------------------------------------------------------------

describe("sourceLabel", () => {
  it.each([
    [SOURCE_GATEWAY, "GW"],
    [SOURCE_FIREWALL, "FW"],
    [SOURCE_AGENT, "AG"],
  ])("maps %s → %s", (source, expected) => {
    expect(sourceLabel(source)).toBe(expected);
  });

  it("returns first two chars uppercased for unknown sources longer than 1 char", () => {
    expect(sourceLabel("custom-source")).toBe("CU");
  });

  it("returns uppercased single char for 1-char unknown sources", () => {
    expect(sourceLabel("x")).toBe("X");
  });
});

describe("eventIcon", () => {
  const allKinds = [KIND_TOOL_CALL, KIND_DIFC_FILTERED, KIND_GUARD_BLOCKED, KIND_NET_ALLOWED, KIND_NET_BLOCKED, KIND_AGENT_TURN, KIND_AGENT_TOOL_START, KIND_AGENT_TOOL_DONE];

  it.each(allKinds)("returns non-default icon for %s", kind => {
    const icon = eventIcon(kind);
    expect(icon).not.toBe("·");
    expect(icon.length).toBeGreaterThan(0);
  });

  it("returns default dot for unknown kind", () => {
    expect(eventIcon("unknown_kind")).toBe("·");
  });
});

describe("kindLabel", () => {
  it.each([
    [KIND_TOOL_CALL, "tool_call"],
    [KIND_DIFC_FILTERED, "difc_filtered"],
    [KIND_GUARD_BLOCKED, "guard_blocked"],
    [KIND_NET_ALLOWED, "net_allowed"],
    [KIND_NET_BLOCKED, "net_blocked"],
    [KIND_AGENT_TURN, "agent_turn"],
    [KIND_AGENT_TOOL_START, "tool_start"],
    [KIND_AGENT_TOOL_DONE, "tool_done"],
  ])("maps %s → %s", (kind, expected) => {
    expect(kindLabel(kind)).toBe(expected);
  });

  it("returns the kind string unchanged for unknown kinds", () => {
    expect(kindLabel("custom_event")).toBe("custom_event");
  });
});

// ---------------------------------------------------------------------------
// collectGatewayEvents
// ---------------------------------------------------------------------------

describe("collectGatewayEvents", () => {
  it("returns empty array when neither file exists", () => {
    expect(collectGatewayEvents({ gatewayJsonlPath: "/nonexistent", rpcMessagesPath: "/nonexistent2" })).toEqual([]);
  });

  it("parses tool_call events from gateway.jsonl", () => {
    const gwPath = path.join(tmpDir, "gateway.jsonl");
    writeJsonl(gwPath, [
      {
        timestamp: "2024-01-15T10:00:02.000Z",
        event: "tool_call",
        server_name: "my-server",
        tool_name: "get_file",
        duration: 100,
      },
    ]);
    const events = collectGatewayEvents({ gatewayJsonlPath: gwPath, rpcMessagesPath: "/nonexistent" });
    expect(events).toHaveLength(1);
    expect(events[0].kind).toBe(KIND_TOOL_CALL);
    expect(events[0].source).toBe(SOURCE_GATEWAY);
    expect(events[0].detail).toContain("my-server/get_file");
    expect(events[0].status).toContain("success");
    expect(events[0].status).toContain("100ms");
  });

  it("parses DIFC_FILTERED events from gateway.jsonl", () => {
    const gwPath = path.join(tmpDir, "gateway.jsonl");
    writeJsonl(gwPath, [
      {
        timestamp: "2024-01-15T10:00:01Z",
        type: "DIFC_FILTERED",
        server_id: "srv-1",
        tool_name: "post_comment",
        reason: "secrecy violation",
      },
    ]);
    const events = collectGatewayEvents({ gatewayJsonlPath: gwPath, rpcMessagesPath: "/nonexistent" });
    expect(events).toHaveLength(1);
    expect(events[0].kind).toBe(KIND_DIFC_FILTERED);
    expect(events[0].status).toBe("secrecy violation");
  });

  it("parses GUARD_POLICY_BLOCKED events from gateway.jsonl", () => {
    const gwPath = path.join(tmpDir, "gateway.jsonl");
    writeJsonl(gwPath, [
      {
        timestamp: "2024-01-15T10:00:03Z",
        type: "GUARD_POLICY_BLOCKED",
        server_name: "code-srv",
        tool_name: "run_shell",
        reason: "policy: no shell",
      },
    ]);
    const events = collectGatewayEvents({ gatewayJsonlPath: gwPath, rpcMessagesPath: "/nonexistent" });
    expect(events).toHaveLength(1);
    expect(events[0].kind).toBe(KIND_GUARD_BLOCKED);
  });

  it("falls back to rpc-messages.jsonl when gateway.jsonl is absent", () => {
    const rpcPath = path.join(tmpDir, "rpc-messages.jsonl");
    writeJsonl(rpcPath, [
      {
        timestamp: "2024-01-15T10:00:01Z",
        direction: "OUT",
        type: "REQUEST",
        method: "tools/call",
      },
    ]);
    const events = collectGatewayEvents({ gatewayJsonlPath: "/nonexistent", rpcMessagesPath: rpcPath });
    expect(events).toHaveLength(1);
    expect(events[0].kind).toBe(KIND_TOOL_CALL);
  });

  it("skips entries with missing or invalid timestamps", () => {
    const gwPath = path.join(tmpDir, "gateway.jsonl");
    writeJsonl(gwPath, [
      { event: "tool_call", server_name: "srv", tool_name: "t" },
      { timestamp: "not-a-date", event: "tool_call", server_name: "srv", tool_name: "t" },
      { timestamp: "2024-01-15T10:00:02Z", event: "tool_call", server_name: "srv", tool_name: "t2" },
    ]);
    const events = collectGatewayEvents({ gatewayJsonlPath: gwPath, rpcMessagesPath: "/nonexistent" });
    expect(events).toHaveLength(1);
    expect(events[0].detail).toContain("t2");
  });

  it("produces error status when entry has error field", () => {
    const gwPath = path.join(tmpDir, "gateway.jsonl");
    writeJsonl(gwPath, [
      {
        timestamp: "2024-01-15T10:00:02Z",
        event: "tool_call",
        server_name: "srv",
        tool_name: "failing_tool",
        error: "timeout",
      },
    ]);
    const events = collectGatewayEvents({ gatewayJsonlPath: gwPath, rpcMessagesPath: "/nonexistent" });
    expect(events).toHaveLength(1);
    expect(events[0].status).toBe("error");
  });

  it("parses DIFC_FILTERED events from rpc-messages.jsonl", () => {
    const rpcPath = path.join(tmpDir, "rpc-messages.jsonl");
    writeJsonl(rpcPath, [
      {
        timestamp: "2024-01-15T10:00:01Z",
        type: "DIFC_FILTERED",
        server_id: "rpc-srv",
        tool_name: "secret_tool",
        reason: "secrecy",
      },
    ]);
    const events = collectGatewayEvents({ gatewayJsonlPath: "/nonexistent", rpcMessagesPath: rpcPath });
    expect(events).toHaveLength(1);
    expect(events[0].kind).toBe(KIND_DIFC_FILTERED);
    expect(events[0].detail).toContain("rpc-srv/secret_tool");
    expect(events[0].status).toBe("secrecy");
  });

  it("parses gateway rpc_call without server name", () => {
    const gwPath = path.join(tmpDir, "gateway.jsonl");
    writeJsonl(gwPath, [
      {
        timestamp: "2024-01-15T10:00:02Z",
        event: "rpc_call",
        tool_name: "standalone_tool",
      },
    ]);
    const events = collectGatewayEvents({ gatewayJsonlPath: gwPath, rpcMessagesPath: "/nonexistent" });
    expect(events).toHaveLength(1);
    expect(events[0].detail).toBe("standalone_tool");
  });
});

// ---------------------------------------------------------------------------
// collectFirewallEvents
// ---------------------------------------------------------------------------

describe("collectFirewallEvents", () => {
  it("returns empty array when file does not exist", () => {
    expect(collectFirewallEvents({ auditJsonlPath: "/nonexistent" })).toEqual([]);
  });

  it("parses allowed network events (unix float ts)", () => {
    const auditPath = path.join(tmpDir, "audit.jsonl");
    writeJsonl(auditPath, [
      {
        ts: 1705312800.123,
        host: "api.example.com:443",
        method: "CONNECT",
        status: 200,
        decision: "TCP_TUNNEL:HIER_DIRECT",
      },
    ]);
    const events = collectFirewallEvents({ auditJsonlPath: auditPath });
    expect(events).toHaveLength(1);
    expect(events[0].kind).toBe(KIND_NET_ALLOWED);
    expect(events[0].source).toBe(SOURCE_FIREWALL);
    expect(events[0].detail).toContain("api.example.com:443");
  });

  it("classifies events with blocked decision as net_blocked", () => {
    const auditPath = path.join(tmpDir, "audit.jsonl");
    writeJsonl(auditPath, [
      {
        ts: 1705312800.0,
        host: "malicious.example.com",
        decision: "TCP_DENIED:HIER_DIRECT",
      },
    ]);
    const events = collectFirewallEvents({ auditJsonlPath: auditPath });
    expect(events).toHaveLength(1);
    expect(events[0].kind).toBe(KIND_NET_BLOCKED);
  });

  it("classifies entries with 4xx/5xx HTTP status as net_blocked", () => {
    const auditPath = path.join(tmpDir, "audit.jsonl");
    writeJsonl(auditPath, [{ ts: 1705312800.0, host: "blocked.example.com", status: 403 }]);
    const events = collectFirewallEvents({ auditJsonlPath: auditPath });
    expect(events[0].kind).toBe(KIND_NET_BLOCKED);
  });

  it("skips entries with missing ts", () => {
    const auditPath = path.join(tmpDir, "audit.jsonl");
    writeJsonl(auditPath, [{ host: "no-ts.example.com" }, { ts: 1705312800.0, host: "ok.example.com" }]);
    const events = collectFirewallEvents({ auditJsonlPath: auditPath });
    expect(events).toHaveLength(1);
  });

  it("skips benign Squid operational entries (error:transaction-end-before-headers)", () => {
    const auditPath = path.join(tmpDir, "audit.jsonl");
    writeJsonl(auditPath, [
      { ts: 1705312800.0, host: "squid.internal", url: "error:transaction-end-before-headers" },
      { ts: 1705312801.0, host: "ok.example.com" },
    ]);
    const events = collectFirewallEvents({ auditJsonlPath: auditPath });
    expect(events).toHaveLength(1);
    expect(events[0].detail).toContain("ok.example.com");
  });

  it("skips entries with empty or dash host", () => {
    const auditPath = path.join(tmpDir, "audit.jsonl");
    writeJsonl(auditPath, [
      { ts: 1705312800.0, host: "" },
      { ts: 1705312801.0, host: "-" },
      { ts: 1705312802.0, host: "ok.example.com" },
    ]);
    const events = collectFirewallEvents({ auditJsonlPath: auditPath });
    expect(events).toHaveLength(1);
    expect(events[0].detail).toContain("ok.example.com");
  });
});

// ---------------------------------------------------------------------------
// findEventsJsonlFile
// ---------------------------------------------------------------------------

describe("findEventsJsonlFile", () => {
  it("returns null when directory does not exist", () => {
    expect(findEventsJsonlFile(path.join(tmpDir, "nonexistent"))).toBeNull();
  });

  it("finds events.jsonl under copilot-session-state/<uuid>/", () => {
    const sessionDir = path.join(tmpDir, "sandbox", "agent", "logs", "copilot-session-state", "test-uuid-1234");
    fs.mkdirSync(sessionDir, { recursive: true });
    const eventsPath = path.join(sessionDir, "events.jsonl");
    fs.writeFileSync(eventsPath, "", "utf8");
    expect(findEventsJsonlFile(tmpDir)).toBe(eventsPath);
  });
});

// ---------------------------------------------------------------------------
// collectAgentEvents
// ---------------------------------------------------------------------------

describe("collectAgentEvents", () => {
  it("returns empty array when no events.jsonl is found", () => {
    expect(collectAgentEvents({ eventsJsonlPath: "/nonexistent" })).toEqual([]);
  });

  it("parses user.message as agent_turn with incrementing turn index", () => {
    const eventsPath = path.join(tmpDir, "events.jsonl");
    writeJsonl(eventsPath, [
      { type: "user.message", id: "m1", timestamp: "2024-01-15T10:00:01Z", data: {} },
      { type: "user.message", id: "m2", timestamp: "2024-01-15T10:00:10Z", data: {} },
    ]);
    const events = collectAgentEvents({ eventsJsonlPath: eventsPath });
    expect(events).toHaveLength(2);
    expect(events[0].kind).toBe(KIND_AGENT_TURN);
    expect(events[0].detail).toBe("turn 1");
    expect(events[1].detail).toBe("turn 2");
  });

  it("parses tool.execution_start", () => {
    const eventsPath = path.join(tmpDir, "events.jsonl");
    writeJsonl(eventsPath, [
      {
        type: "tool.execution_start",
        id: "t1",
        timestamp: "2024-01-15T10:00:02Z",
        data: { toolCallId: "c1", toolName: "search_files", mcpServerName: "code-srv" },
      },
    ]);
    const events = collectAgentEvents({ eventsJsonlPath: eventsPath });
    expect(events).toHaveLength(1);
    expect(events[0].kind).toBe(KIND_AGENT_TOOL_START);
    expect(events[0].detail).toContain("code-srv/search_files");
    expect(events[0].status).toBe("");
  });

  it("parses tool.execution_complete success", () => {
    const eventsPath = path.join(tmpDir, "events.jsonl");
    writeJsonl(eventsPath, [
      {
        type: "tool.execution_complete",
        id: "t2",
        timestamp: "2024-01-15T10:00:03Z",
        data: { toolCallId: "c1", toolName: "search_files", success: true },
      },
    ]);
    const events = collectAgentEvents({ eventsJsonlPath: eventsPath });
    expect(events[0].kind).toBe(KIND_AGENT_TOOL_DONE);
    expect(events[0].status).toBe("success");
  });

  it("parses tool.execution_complete failure", () => {
    const eventsPath = path.join(tmpDir, "events.jsonl");
    writeJsonl(eventsPath, [
      {
        type: "tool.execution_complete",
        id: "t3",
        timestamp: "2024-01-15T10:00:04Z",
        data: { toolCallId: "c2", toolName: "run_cmd", success: false },
      },
    ]);
    const events = collectAgentEvents({ eventsJsonlPath: eventsPath });
    expect(events[0].status).toBe("error");
  });

  it("skips session.start and session.shutdown events", () => {
    const eventsPath = path.join(tmpDir, "events.jsonl");
    writeJsonl(eventsPath, [
      { type: "session.start", id: "s1", timestamp: "2024-01-15T10:00:00Z", data: {} },
      { type: "user.message", id: "m1", timestamp: "2024-01-15T10:00:01Z", data: {} },
      { type: "session.shutdown", id: "s2", timestamp: "2024-01-15T10:00:10Z", data: {} },
    ]);
    const events = collectAgentEvents({ eventsJsonlPath: eventsPath });
    expect(events).toHaveLength(1);
    expect(events[0].kind).toBe(KIND_AGENT_TURN);
  });

  it("uses logDir to find events.jsonl in canonical location", () => {
    const sessionDir = path.join(tmpDir, "sandbox", "agent", "logs", "copilot-session-state", "uuid-xyz");
    fs.mkdirSync(sessionDir, { recursive: true });
    const eventsPath = path.join(sessionDir, "events.jsonl");
    writeJsonl(eventsPath, [{ type: "user.message", id: "m1", timestamp: "2024-01-15T10:00:01Z", data: {} }]);
    const events = collectAgentEvents({ logDir: tmpDir });
    expect(events).toHaveLength(1);
  });

  it("uses tool name only when mcpServerName is absent", () => {
    const eventsPath = path.join(tmpDir, "events.jsonl");
    writeJsonl(eventsPath, [
      {
        type: "tool.execution_start",
        id: "t1",
        timestamp: "2024-01-15T10:00:02Z",
        data: { toolName: "standalone_tool" },
      },
    ]);
    const events = collectAgentEvents({ eventsJsonlPath: eventsPath });
    expect(events).toHaveLength(1);
    expect(events[0].detail).toBe("standalone_tool");
  });
});

// ---------------------------------------------------------------------------
// collectUnifiedTimelineEvents
// ---------------------------------------------------------------------------

describe("collectUnifiedTimelineEvents", () => {
  it("returns empty array when no files exist", () => {
    const opts = {
      gatewayJsonlPath: "/nonexistent",
      rpcMessagesPath: "/nonexistent2",
      auditJsonlPath: "/nonexistent3",
      eventsJsonlPath: "/nonexistent4",
    };
    expect(collectUnifiedTimelineEvents(opts)).toHaveLength(0);
  });

  it("merges and sorts events from all three sources", () => {
    // Gateway at t+2s
    const gwPath = path.join(tmpDir, "gateway.jsonl");
    writeJsonl(gwPath, [{ timestamp: "2024-01-15T10:00:02Z", event: "tool_call", server_name: "srv", tool_name: "get_file" }]);

    // Firewall at t+3s
    const auditPath = path.join(tmpDir, "audit.jsonl");
    writeJsonl(auditPath, [{ ts: new Date("2024-01-15T10:00:03Z").getTime() / 1000, host: "api.example.com", decision: "TCP_TUNNEL:HIER_DIRECT" }]);

    // Agent at t+1s
    const eventsPath = path.join(tmpDir, "events.jsonl");
    writeJsonl(eventsPath, [{ type: "user.message", id: "m1", timestamp: "2024-01-15T10:00:01Z", data: {} }]);

    const events = collectUnifiedTimelineEvents({
      gatewayJsonlPath: gwPath,
      rpcMessagesPath: "/nonexistent",
      auditJsonlPath: auditPath,
      eventsJsonlPath: eventsPath,
    });

    expect(events).toHaveLength(3);
    // Should be sorted: agent(t+1) < gateway(t+2) < firewall(t+3)
    expect(events[0].source).toBe(SOURCE_AGENT);
    expect(events[1].source).toBe(SOURCE_GATEWAY);
    expect(events[2].source).toBe(SOURCE_FIREWALL);
  });
});

// ---------------------------------------------------------------------------
// buildUnifiedTimelineMarkdown
// ---------------------------------------------------------------------------

describe("buildUnifiedTimelineMarkdown", () => {
  it("returns empty string for empty events", () => {
    expect(buildUnifiedTimelineMarkdown([])).toBe("");
  });

  it("returns empty string for null/undefined", () => {
    expect(buildUnifiedTimelineMarkdown(null)).toBe("");
  });

  it("wraps output in a <details> block", () => {
    const events = [{ source: SOURCE_GATEWAY, kind: KIND_TOOL_CALL, time: new Date("2024-01-15T10:00:02Z"), detail: "srv/get_file", status: "success" }];
    const md = buildUnifiedTimelineMarkdown(events);
    expect(md).toContain("<details>");
    expect(md).toContain("</details>");
  });

  it("includes all three source labels in summary when present", () => {
    const t = new Date("2024-01-15T10:00:00Z");
    const events = [
      { source: SOURCE_GATEWAY, kind: KIND_TOOL_CALL, time: t, detail: "srv/t", status: "success" },
      { source: SOURCE_FIREWALL, kind: KIND_NET_ALLOWED, time: new Date(t.getTime() + 1000), detail: "host", status: "200" },
      { source: SOURCE_AGENT, kind: KIND_AGENT_TURN, time: new Date(t.getTime() + 2000), detail: "turn 1", status: "" },
    ];
    const md = buildUnifiedTimelineMarkdown(events);
    expect(md).toContain("GW");
    expect(md).toContain("FW");
    expect(md).toContain("AG");
  });

  it("contains a Markdown table header row", () => {
    const events = [{ source: SOURCE_AGENT, kind: KIND_AGENT_TURN, time: new Date("2024-01-15T10:00:01Z"), detail: "turn 1", status: "" }];
    const md = buildUnifiedTimelineMarkdown(events);
    expect(md).toContain("| Time | Src | Kind | Detail | Status |");
  });

  it("shows summary counts for all three sources", () => {
    const t = new Date("2024-01-15T10:00:00Z");
    const events = [
      { source: SOURCE_AGENT, kind: KIND_AGENT_TURN, time: t, detail: "turn 1", status: "" },
      { source: SOURCE_AGENT, kind: KIND_AGENT_TOOL_START, time: new Date(t.getTime() + 100), detail: "srv/tool", status: "" },
      { source: SOURCE_AGENT, kind: KIND_AGENT_TOOL_DONE, time: new Date(t.getTime() + 200), detail: "srv/tool", status: "success" },
      { source: SOURCE_FIREWALL, kind: KIND_NET_BLOCKED, time: new Date(t.getTime() + 300), detail: "bad.host", status: "blocked" },
    ];
    const md = buildUnifiedTimelineMarkdown(events);
    expect(md).toContain("turns=1");
    expect(md).toContain("tool_start=1");
    expect(md).toContain("tool_done=1");
    expect(md).toContain("blocked=1");
  });

  it("escapes pipe characters in detail/status to avoid breaking Markdown tables", () => {
    const events = [{ source: SOURCE_GATEWAY, kind: KIND_TOOL_CALL, time: new Date("2024-01-15T10:00:00Z"), detail: "srv|tool", status: "ok|extra" }];
    const md = buildUnifiedTimelineMarkdown(events);
    // After escaping, literal | inside cells should not appear
    const rows = md.split("\n").filter(l => l.startsWith("| 10:"));
    expect(rows).toHaveLength(1);
    expect(rows[0]).not.toContain("srv|tool");
    expect(rows[0]).toContain("&#124;");
  });

  it("counts gateway tool_calls in stats line", () => {
    const events = [{ source: SOURCE_GATEWAY, kind: KIND_TOOL_CALL, time: new Date("2024-01-15T10:00:02Z"), detail: "srv/tool", status: "success" }];
    const md = buildUnifiedTimelineMarkdown(events);
    expect(md).toContain("tool_calls=1");
    expect(md).toContain("difc_filtered=0");
  });
});

// ---------------------------------------------------------------------------
// generateUnifiedTimelineSummary (integration)
// ---------------------------------------------------------------------------

describe("generateUnifiedTimelineSummary", () => {
  it("returns empty string when all files are missing", () => {
    const result = generateUnifiedTimelineSummary({
      gatewayJsonlPath: "/nonexistent",
      rpcMessagesPath: "/nonexistent2",
      auditJsonlPath: "/nonexistent3",
      eventsJsonlPath: "/nonexistent4",
    });
    expect(result).toBe("");
  });

  it("renders a non-empty summary when at least one source has events", () => {
    const eventsPath = path.join(tmpDir, "events.jsonl");
    writeJsonl(eventsPath, [{ type: "user.message", id: "m1", timestamp: "2024-01-15T10:00:01Z", data: {} }]);
    const result = generateUnifiedTimelineSummary({
      gatewayJsonlPath: "/nonexistent",
      rpcMessagesPath: "/nonexistent2",
      auditJsonlPath: "/nonexistent3",
      eventsJsonlPath: eventsPath,
    });
    expect(result).not.toBe("");
    expect(result).toContain("agent_turn");
  });
});
