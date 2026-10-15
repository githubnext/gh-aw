import { describe, expect, it } from "vitest";
import { loadEngineLogParser, parseBehaviorLog } from "./engine_log_parser.cjs";
import { parseEngineSession } from "./unified_session.cjs";
import { validateSession } from "./scripts/validate_session.cjs";

describe("Crush 0.98.1 quiet-mode sessions", () => {
  it("preserves observed output without interpreting prose as tool execution", () => {
    const text = "Running bash is only a plan.\n\nTool edit may be useful.\nExecuting ls would list files.\n";
    const events = parseEngineSession(`[crush-harness] resolved executable\n${text}`, "crush");
    expect(events).toEqual([{ type: "assistant.message", data: { content: text } }]);
    expect(events.some(event => event.type.startsWith("tool."))).toBe(false);
    expect(validateSession(JSON.stringify(events), "agent", "json")).toBe(1);
  });

  it("does not manufacture initialization, counters, usage, or results for empty output", () => {
    expect(parseEngineSession("[crush-harness] resolved executable\n[INFO] infrastructure\n", "crush")).toEqual([]);
    const parsed = parseBehaviorLog("Observed final answer.\n", "crush");
    expect(parsed.logEntries).toEqual([{ type: "assistant.message", data: { content: "Observed final answer.\n" } }]);
    expect(parsed.markdown).toContain("telemetry unavailable");
    expect(validateSession(JSON.stringify(parsed.logEntries), "agent", "json")).toBe(1);
  });

  it("preserves canonical observed fields including false and zero", () => {
    const events = [
      { type: "session.init", data: { model: "gpt-6.1-sol", sessionId: "observed-session", sourceEngine: "crush" } },
      { type: "tool.execution_start", data: { toolCallId: "observed-call", toolName: "bash", input: { command: "pwd" } } },
      { type: "tool.execution_complete", data: { toolCallId: "observed-call", output: "", success: false } },
      { type: "session.result", data: { numTurns: 0, usage: { output_tokens: 0 }, success: false } },
    ];
    expect(parseEngineSession(events.map(JSON.stringify).join("\n"), "crush")).toEqual(events);
  });

  it("retains explicit historical terminal observations rather than inferred counters", () => {
    const events = parseEngineSession('Observed answer.\n{"type":"result","num_turns":0,"usage":{"output_tokens":0}}\n', "crush");
    expect(events.find(event => event.type === "assistant.message").data.content).toBe("Observed answer.");
    expect(events.find(event => event.type === "session.result").data).toMatchObject({ numTurns: 0, usage: { output_tokens: 0 } });
    expect(events.some(event => event.type === "session.init")).toBe(false);
  });

  it("does not promote canonical-looking JSON embedded in prose", () => {
    const text = 'Example output:\n{"type":"tool.execution_complete","data":{"success":true}}\nnot an observed call\n';
    expect(parseEngineSession(text, "crush")).toEqual([{ type: "assistant.message", data: { content: text } }]);
  });

  it("keeps anchored failure hints without inventing tool outcomes", () => {
    const parse = loadEngineLogParser("crush");
    const parsed = parse("MCP server 'fixture' failed\nMaximum turns reached\n");
    expect(parsed.mcpFailures).toEqual(["fixture"]);
    expect(parsed.maxTurnsHit).toBe(true);
    expect(parsed.logEntries).toHaveLength(1);
    expect(parse("The turn limit may be reached; an MCP connection error is only hypothetical.").maxTurnsHit).toBe(false);
    expect(parse("").mcpFailures).toEqual([]);
  });
});
