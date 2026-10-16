import { describe, expect, it } from "vitest";
import { normalizeClaudeSession } from "./claude_session.cjs";
import { normalizeEngineSessionEvents } from "./engine_log_parser.cjs";
import { normalizeUnifiedSessionEvent } from "./unified_session_payload.cjs";
import { normalizeCodexSession } from "./codex_session.cjs";
import { normalizeCopilotSession } from "./copilot_session.cjs";
import { parseAgyLog } from "./parse_agy_log.cjs";
import { parsePiLog } from "./parse_pi_log.cjs";
import { mergeSessionSources } from "./unified_session.cjs";
import { normalizeAgentSession, projectSessionResult } from "./agent_session.cjs";
import { collapseStreamedMessages } from "./agent_session_render.cjs";
import { createSessionValidator } from "./scripts/validate_session.cjs";

const unified = events => mergeSessionSources([{ component: "agent", phase: "agent", path: "agent-stdio.log", events }]);
const parse = (parser, records) => parser(records.map(record => JSON.stringify(record)).join("\n")).logEntries;
const validate = events => {
  const agentValidator = createSessionValidator("agent").event;
  const unifiedValidator = createSessionValidator("unified").event;
  for (const event of events) expect(agentValidator(event), JSON.stringify(agentValidator.errors)).toBe(true);
  for (const event of unified(events)) expect(unifiedValidator(event), JSON.stringify(unifiedValidator.errors)).toBe(true);
};

describe("shared engine event mappings", () => {
  // Sanitized status shapes from live runs 37789661275 and 37864205681.
  it("maps Claude reasoning estimates and VCS status without counting estimates as measured usage", () => {
    const records = [
      { type: "system", subtype: "thinking_tokens", session_id: "session", uuid: "estimate", estimated_tokens: 12, estimated_tokens_delta: 0 },
      { type: "system", subtype: "vcs_state_changed", session_id: "session", uuid: "vcs", kind: "branch", branch: "example", cwd: "/workspace" },
      { type: "system", subtype: "future_status", session_id: "session", value: false },
    ];
    const original = structuredClone(records);
    const events = normalizeClaudeSession(records);
    expect(events.map(event => event.type)).toEqual(["session.info", "session.info", "claude.system"]);
    expect(unified(events)[0].data).toEqual({ sessionId: "session", sourceType: "thinking_tokens", estimatedTokens: 12, estimatedTokensDelta: 0 });
    expect(unified(events)[1].data).toEqual({ sessionId: "session", sourceType: "vcs_state_changed", kind: "branch", branch: "example", cwd: "/workspace" });
    expect(projectSessionResult(events)).toBeUndefined();
    expect(normalizeClaudeSession(events)).toEqual(events);
    expect(records).toEqual(original);
    validate(events);
  });

  it("keeps mapped Claude status notices out of conversational messages", () => {
    const events = normalizeClaudeSession([{ type: "system", subtype: "status", session_id: "session", message: "provider notice" }]);
    expect(events.map(event => event.type)).toEqual(["session.info"]);
    expect(unified(events)[0].data).toMatchObject({ sourceType: "status", message: "provider notice" });
    validate(events);
  });

  it("maps Claude tool progress, authentication and rate limits without inventing completions or answers", () => {
    const events = normalizeClaudeSession([
      { type: "tool_progress", session_id: "session", tool_use_id: "call", tool_name: "mcp__github__lookup", elapsed_time_seconds: 1.5 },
      { type: "tool_use_summary", summary: " observed tools\n" },
      { type: "auth_status", isAuthenticating: false, output: [] },
      { type: "rate_limit_event", rate_limit_info: { status: "allowed_warning", resetsAt: 0 } },
    ]);
    expect(events.map(event => event.type)).toEqual(["tool.execution_update", "session.info", "session.info", "session.info"]);
    expect(unified(events)[0].data).toEqual({ sessionId: "session", toolCallId: "call", toolName: "lookup", mcpServerName: "github", partial: true, sourceType: "tool_progress", durationMs: 1500 });
    expect(unified(events)[1].data.content).toBe(" observed tools\n");
    expect(unified(events)[2].data).toMatchObject({ isAuthenticating: false, output: [] });
    expect(unified(events)[3].data.rateLimitInfo).toEqual({ status: "allowed_warning", resetsAt: 0 });
    expect(projectSessionResult(events)).toBeUndefined();
    validate(events);
  });

  it("upgrades saved Claude status wrappers without replaying canonical messages or unknown events", () => {
    const records = [
      {
        type: "claude.system",
        id: "estimate",
        provenance: { component: "agent", phase: "agent", path: "agent-session.jsonl", index: 0 },
        data: { subtype: "thinking_tokens", session_id: "session", estimated_tokens: 3, estimated_tokens_delta: 0 },
      },
      { type: "assistant.message", data: { content: " once\n" } },
      { type: "claude.system", data: { subtype: "future_status", value: false } },
    ];
    const events = normalizeEngineSessionEvents(records, "claude", event => ({ ...normalizeUnifiedSessionEvent(event), provenance: event.provenance }));
    expect(events[0]).toEqual({ type: "session.info", id: "estimate", provenance: records[0].provenance, data: { sessionId: "session", sourceType: "thinking_tokens", estimatedTokens: 3, estimatedTokensDelta: 0 } });
    expect(events.slice(1)).toEqual(records.slice(1));
    expect(normalizeEngineSessionEvents(events, "claude")).toBe(events);
    expect(normalizeEngineSessionEvents(records, "copilot")).toBe(records);
    validate(events);
  });

  it("does not replay compatibility text while upgrading a saved Claude status notice", () => {
    const records = [
      { type: "claude.system", data: { subtype: "status", message: " observed status\n" } },
      { type: "assistant.message", data: { content: " observed status\n" } },
    ];
    const events = normalizeEngineSessionEvents(records, "claude");
    expect(events.map(event => event.type)).toEqual(["session.info", "assistant.message"]);
    expect(events[0].data.message).toBe(" observed status\n");
    expect(events[1]).toEqual(records[1]);
    expect(normalizeEngineSessionEvents(events, "claude")).toBe(events);
    validate(events);
  });

  it("retains Claude stream controls as standard updates through compact rendering", () => {
    const stream = event => ({ type: "stream_event", session_id: "session", event });
    const records = [
      stream({ type: "message_start", message: { id: "answer", model: "fixture-model" } }),
      stream({ type: "content_block_start", index: 0, content_block: { type: "text", text: "first" } }),
      stream({ type: "message_delta", delta: { stop_reason: "end_turn" }, usage: { output_tokens: 2 } }),
      stream({ type: "message_stop" }),
      { type: "assistant", session_id: "session", message: { id: "answer", content: [{ type: "text", text: "first\n" }] } },
    ];
    const events = normalizeClaudeSession(records);
    expect(events.some(event => event.type.startsWith("claude."))).toBe(false);
    expect(events.filter(event => event.type === "assistant.message_update").map(event => event.data.sourceType)).toEqual(["message_start", "message_delta", "message_stop"]);
    expect(
      collapseStreamedMessages(unified(events))
        .filter(event => event.type === "assistant.message")
        .map(event => event.data.content)
    ).toEqual(["first\n"]);
    expect(projectSessionResult(events).usage.output_tokens).toBe(2);
    validate(events);
  });

  it("retains argument and signature observations on core events without extra stream wrappers", () => {
    const stream = event => ({ type: "stream_event", session_id: "session", event });
    const records = [
      stream({ type: "message_start", message: { id: "message" } }),
      stream({ type: "content_block_start", index: 0, content_block: { type: "thinking", thinking: " exact\n" } }),
      { ...stream({ type: "content_block_delta", index: 0, delta: { type: "signature_delta", signature: "signature" } }), id: "signature-event", timestamp: 0 },
      stream({ type: "content_block_start", index: 1, content_block: { type: "tool_use", id: "call", name: "lookup", input: {} } }),
      { ...stream({ type: "content_block_delta", index: 1, delta: { type: "input_json_delta", partial_json: '{"flag":false}' } }), id: "arguments-event" },
      stream({ type: "content_block_stop", index: 1 }),
    ];
    const original = structuredClone(records);
    const events = normalizeClaudeSession(records);
    expect(events.some(event => event.type.startsWith("claude."))).toBe(false);
    expect(events.find(event => event.type === "assistant.reasoning").data).toMatchObject({ signature: "signature", snapshots: [records[2]] });
    expect(events.find(event => event.type === "tool.execution_start").data).toMatchObject({ input: { flag: false }, argumentText: '{"flag":false}', snapshots: [records[4], records[5]] });
    expect(normalizeClaudeSession(events)).toEqual(events);
    expect(records).toEqual(original);
    validate(events);
  });

  it("projects historical Pi and future message updates through the shared allowlist", () => {
    const event = { type: "assistant.message_update", data: { sourceType: "message_update", type: "future_delta", delta: { flag: false }, usage: { input: 3 }, contentIndex: 0 } };
    expect(normalizeUnifiedSessionEvent(event).data).toEqual({ sourceType: "message_update", type: "future_delta", delta: { flag: false }, usageSnapshot: { input: 3 }, contentIndex: 0 });
    validate([event]);
  });

  it.each(["text_delta", "thinking_delta", "text_end", "thinking_end"])("preserves essential historical Pi %s observations without admitting extension fields", type => {
    const record = { type: "pi.message_update", data: { type, contentIndex: 0, delta: "", content: false, usage: { input: 0 }, privateExtension: "native evidence" } };
    const original = structuredClone(record);
    const events = normalizeEngineSessionEvents([record], "pi");
    expect(events[0].data.privateExtension).toBe("native evidence");
    const projected = unified(events);
    expect(projected[0].data).toEqual({ sourceEngine: "pi", sourceType: "message_update", type, contentIndex: 0, delta: "", content: false, usageSnapshot: { input: 0 } });
    expect(normalizeUnifiedSessionEvent(projected[0]).data).toEqual(projected[0].data);
    expect(projectSessionResult(events)).toBeUndefined();
    expect(record).toEqual(original);
    validate(events);
  });

  it("maps Codex snapshots and todo lists to standard events while preserving partial output and history", () => {
    const records = [
      { type: "thread.started", thread_id: "thread" },
      { type: "item.started", id: "partial", timestamp: 0, item: { id: "answer", type: "agent_message", text: " partial\n" } },
      { type: "item.started", item: { id: "call", type: "command_execution", command: "exact command", status: "in_progress" } },
      { type: "item.updated", item: { id: "call", type: "command_execution", aggregated_output: "partial output\n", status: "in_progress" } },
      { type: "item.completed", item: { id: "answer", type: "agent_message", text: "final answer\n" } },
      { type: "item.completed", item: { id: "todo", type: "todo_list", items: [{ text: "check", completed: false }] } },
    ];
    const original = structuredClone(records);
    const events = normalizeCodexSession(records);
    expect(events.map(event => event.type)).toEqual(["session.init", "tool.execution_start", "tool.execution_update", "assistant.message", "session.info"]);
    expect(events[3].data.nativeSnapshots).toEqual([records[1]]);
    expect(unified(events)[2].data).toMatchObject({ toolCallId: "call", command: "exact command", output: "partial output\n", partial: true, status: "in_progress" });
    expect(unified(events)[4].data).toEqual({ sessionId: "thread", sourceType: "todo_list", content: [{ text: "check", completed: false }] });
    expect(projectSessionResult(events)).toBeUndefined();
    expect(normalizeCodexSession(events)).toEqual(events);
    expect(records).toEqual(original);
    validate(events);
  });

  it("maps content-free Codex message boundaries without inventing text", () => {
    const record = { type: "item.started", id: "event", item: { id: "message", type: "agent_message" } };
    const events = normalizeCodexSession([record]);
    expect(events).toHaveLength(1);
    expect(events[0]).toMatchObject({ type: "assistant.message_update", id: "event", data: { messageId: "message", sourceType: "item.started", partial: true } });
    expect(events[0].data).not.toHaveProperty("content");
    expect(normalizeCodexSession(events)).toEqual(events);
    validate(events);
  });

  it.each([
    ["user_message", "user.message"],
    ["reasoning", "assistant.reasoning"],
  ])("keeps content-free Codex %s boundaries on the correct speaker", (itemType, eventType) => {
    const events = normalizeCodexSession([{ type: "item.started", item: { id: "message", type: itemType } }]);
    expect(events.map(event => event.type)).toEqual([eventType]);
    expect(events[0].data).toMatchObject({ messageId: "message", partial: true });
    expect(events[0].data).not.toHaveProperty("content");
    validate(events);
  });

  it.each([false, 0, null, { type: "image", source: { data: "sanitized" } }, { type: "future_block", value: false }])("retains Pi non-text content once across message and transcript snapshots (%j)", content => {
    const message = { role: "user", id: "user", content: [content] };
    const records = [
      { type: "message_start", message },
      { type: "message_end", message },
      { type: "agent_end", messages: [message] },
    ];
    const original = structuredClone(records);
    const events = parse(parsePiLog, records);
    expect(events.filter(event => event.type === "user.message")).toHaveLength(1);
    expect(unified(events)[0].data).toEqual({ content, contentIndex: 0, messageId: "user" });
    expect(records).toEqual(original);
    validate(events);
  });

  it("reconciles Pi content snapshots when a block changes category", () => {
    const image = { type: "image", source: { data: "observed" } };
    const records = [
      { type: "message_start", message: { role: "user", id: "snapshot", content: [image] } },
      { type: "message_end", message: { role: "user", id: "snapshot", content: [{ type: "text", text: "final" }] } },
    ];
    const original = structuredClone(records);
    const events = parse(parsePiLog, records);
    const messages = events.filter(event => event.type === "user.message");
    expect(messages).toHaveLength(1);
    expect(messages[0].data).toMatchObject({ content: "final", nativeSnapshots: [{ content: image }] });
    expect(normalizeAgentSession(events)).toEqual(events);
    expect(records).toEqual(original);
    validate(events);
  });

  it("projects Codex, Pi, and future message updates through the shared allowlist", () => {
    const events = [
      ...normalizeCodexSession([{ type: "item.updated", item: { id: "codex", type: "agent_message", privateExtension: "omit" } }]),
      { type: "assistant.message_update", data: { sourceType: "message_update", delta: "Pi", privateExtension: "omit" } },
      { type: "assistant.message_update", data: { sourceType: "future_delta", delta: "future", privateExtension: "omit" } },
    ];
    const projected = unified(events);
    expect(projected.map(event => event.data)).toEqual([
      { messageId: "codex", sourceType: "item.updated", type: "agent_message", partial: true },
      { sourceType: "message_update", delta: "Pi" },
      { sourceType: "future_delta", delta: "future" },
    ]);
    const validateUnified = createSessionValidator("unified").event;
    for (const event of projected) expect(validateUnified(event), JSON.stringify(validateUnified.errors)).toBe(true);
    for (const event of projected) expect(validateUnified({ ...event, data: { ...event.data, privateExtension: "unexpected" } })).toBe(false);
  });

  it("preserves divergent Pi atomic content as snapshot history without mutating source blocks", () => {
    const observed = { type: "image", data: "observed" };
    const finalized = { type: "image", data: "finalized" };
    const records = [
      { type: "message_start", message: { role: "assistant", id: "message", content: [observed] } },
      { type: "message_end", message: { role: "assistant", id: "message", content: [finalized] } },
    ];
    const original = structuredClone(records);
    const events = parse(parsePiLog, records);
    const messages = events.filter(event => event.type === "assistant.message");
    expect(messages).toHaveLength(1);
    expect(messages[0].data).toMatchObject({ content: finalized, nativeSnapshots: [{ content: observed }] });
    expect(records).toEqual(original);
    validate(events);
  });

  it("keeps Agy step correlation, output, reasoning and cumulative usage in standard events", () => {
    const step = step_update => ({ event: "step_update", step_update: { conversation_id: "conversation", ...step_update } });
    const events = parse(parseAgyLog, [
      { event: "init", conversation_id: "conversation", init: { model: "fixture-model" } },
      step({ step_index: 0, step_type: "thinking", text_delta: " exact reasoning\n" }),
      step({ step_index: 1, step_type: "tool", state: "ACTIVE", tool_info: { name: "lookup", input: false } }),
      step({ step_index: 1, step_type: "tool", state: "DONE", tool_info: { name: "lookup", output: 0 } }),
      { event: "result", result: { conversation_id: "conversation", status: "SUCCESS", num_turns: 1, usage: { input_tokens: 3, output_tokens: 1, cache_read_tokens: 2 } } },
    ]);
    expect(events.some(event => event.type.startsWith("agy."))).toBe(false);
    expect(unified(events).find(event => event.type === "assistant.reasoning").data.content).toBe(" exact reasoning\n");
    expect(unified(events).find(event => event.type === "tool.execution_complete").data).toMatchObject({ sessionId: "conversation", stepIndex: 1, output: 0 });
    expect(projectSessionResult(events).usage).toMatchObject({ input_tokens: 3, output_tokens: 1, cache_read_input_tokens: 2 });
    validate(events);
  });

  it("retains Copilot native lifecycle and request identity with standard message projections", () => {
    const records = [
      { type: "session.start", id: "start", data: { sessionId: "session", selectedModel: "fixture-model" } },
      { type: "assistant.message_delta", id: "delta", data: { messageId: "answer", deltaContent: " exact\n" } },
      { type: "assistant.message", id: "answer-event", data: { messageId: "answer", apiCallId: "request", content: " exact\n", reasoningText: " thinking\n" } },
      { type: "vendor.future", data: { value: false } },
    ];
    const events = normalizeCopilotSession(records);
    expect(events.filter(event => event.type === "assistant.message")).toHaveLength(1);
    expect(unified(events).find(event => event.type === "assistant.message").data).toMatchObject({ messageId: "answer", apiCallId: "request", content: " exact\n" });
    expect(events.find(event => event.type === "assistant.reasoning").data.content).toBe(" thinking\n");
    expect(events.find(event => event.type === "vendor.future")).toEqual(records[3]);
    expect(normalizeCopilotSession(events)).toEqual(events);
    validate(events);
  });
});
