import { describe, expect, it } from "vitest";
import { parsePiLog } from "./parse_pi_log.cjs";
import { computePiV3Stats } from "./pi_session.cjs";
import { release130 } from "./fixtures/pi_ci_stream.cjs";
import { selectSessionResult, sessionTokenTotal } from "./agent_session.cjs";
import { mergeSessionSources } from "./unified_session.cjs";
import { createSessionValidator } from "./scripts/validate_session.cjs";

const parse = records => parsePiLog(records.map(record => JSON.stringify(record)).join("\n"));

describe("Pi v1.3.0 session contract", () => {
  it("preserves reported reasoning as an output subset without counting repeated snapshots", () => {
    const original = structuredClone(release130);
    const events = parse(release130).logEntries;
    const result = selectSessionResult(events);
    expect(result).toMatchObject({
      numTurns: 1,
      totalCostUsd: 0.1,
      usage: { input_tokens: 10, output_tokens: 5, reasoning_output_tokens: 3, cache_read_input_tokens: 2, total_tokens: 17 },
    });
    expect(sessionTokenTotal(result.usage)).toBe(17);
    const unified = mergeSessionSources([{ component: "agent", phase: "agent", path: "pi-streaming.jsonl", events }]);
    expect(unified.find(event => event.type === "session.result").data.usage).toMatchObject({ reasoningOutputTokens: 3, outputTokens: 5, totalTokens: 17 });
    expect(unified.find(event => event.type === "session.reasoning_effort_changed").data.level).toBe("max");
    const validate = createSessionValidator("unified").event;
    for (const event of unified) expect(validate(event), JSON.stringify(validate.errors)).toBe(true);
    expect(release130).toEqual(original);
  });

  it("uses the served model while retaining requested-model and effort metadata in the canonical trace", () => {
    const events = parse(release130).logEntries;
    const message = events.find(event => event.type === "assistant.message");
    expect(message.data).toMatchObject({
      model: "served-model",
      messageMetadata: { model: "configured-model", responseModel: "served-model", thinkingLevel: "max", providerThinkingLevel: "max" },
    });
    const unified = mergeSessionSources([{ component: "agent", phase: "agent", path: "pi-streaming.jsonl", events }]);
    expect(unified.find(event => event.type === "assistant.message").data.model).toBe("served-model");
  });

  it.each([undefined, 0, 2, -1, 0.5])("retains only a reported valid reasoning count (%s)", reasoning => {
    const stats = computePiV3Stats([{ type: "message_end", message: { role: "assistant", content: [], usage: { input: 10, output: 3, reasoning } } }]);
    if (Number.isSafeInteger(reasoning) && reasoning >= 0) expect(stats.usage.reasoning_output_tokens).toBe(reasoning);
    else expect(stats.usage).not.toHaveProperty("reasoning_output_tokens");
    expect(sessionTokenTotal(stats.usage)).toBe(13);
  });

  it("sums reasoning from distinct messages and auxiliary inference once", () => {
    const first = { role: "assistant", responseId: "first", content: [], usage: { input: 10, output: 5, reasoning: 3 } };
    const second = { role: "assistant", responseId: "second", content: [], usage: { input: 2, output: 2, reasoning: 1 } };
    const tool = { role: "toolResult", toolCallId: "classifier", content: [], usage: { input: 1, output: 2, reasoning: 1 } };
    const usage = { id: "warming", type: "usage", usage: { input: 2, output: 0, reasoning: 0 } };
    const stats = computePiV3Stats([
      { type: "message_end", message: first },
      { type: "turn_end", message: first, toolResults: [] },
      { type: "message_end", message: tool },
      { type: "message_end", message: second },
      { type: "turn_end", message: second, toolResults: [tool] },
      { type: "compaction_end", result: { usage: { input: 3, output: 4, reasoning: 2 } } },
      { type: "entry_appended", entry: usage },
      usage,
      { type: "agent_end", messages: [first, tool, second] },
    ]);
    expect(stats).toMatchObject({ turns: 2, usage: { input_tokens: 18, output_tokens: 13, reasoning_output_tokens: 7 } });
    expect(sessionTokenTotal(stats.usage)).toBe(31);
  });

  it("allows an authoritative session usage snapshot to reset reasoning to zero", () => {
    const records = [...release130, { type: "agent_end", messages: [], usage: { reasoning: 0 } }];
    expect(selectSessionResult(parse(records).logEntries).usage).toMatchObject({ reasoning_output_tokens: 0, output_tokens: 5 });
  });

  it("keeps child reasoning and served-model evidence scoped outside parent accounting", () => {
    const message = release130.find(record => record.type === "message_end").message;
    const records = [
      ...release130,
      { type: "gh_aw_subagent_dispatch", agent: "reader", invocation_id: "child" },
      { type: "gh_aw_subagent_event", agent: "reader", invocation_id: "child", event: { type: "message_end", message } },
      { type: "gh_aw_subagent_event", agent: "reader", invocation_id: "child", event: { type: "turn_end", message } },
    ];
    const events = parse(records).logEntries;
    expect(events.filter(event => event.type === "subagent.request")).toMatchObject([{ agentId: "child", data: { model: "served-model", reasoningOutputTokens: 3, outputTokens: 5 } }]);
    expect(selectSessionResult(events).usage.reasoning_output_tokens).toBe(3);
    const unified = mergeSessionSources([{ component: "agent", phase: "agent", path: "pi-streaming.jsonl", events }]);
    expect(unified.find(event => event.type === "subagent.request").data.reasoningOutputTokens).toBe(3);
  });

  it("preserves the served model on provider failures", () => {
    const message = { ...release130.find(record => record.type === "message_end").message, content: [], stopReason: "error", errorMessage: "Synthetic provider failure" };
    const events = parse([{ type: "message_end", message }]).logEntries;
    expect(events.find(event => event.type === "session.error").data).toMatchObject({ model: "served-model", error: message.errorMessage });
  });

  // https://github.com/earendil-works/pi/blob/v1.3.0/packages/coding-agent/src/modes/rpc/rpc-mode.ts
  it.each([false, true])("reports RPC extension errors even without a session header (%s)", header => {
    const failure = { type: "extension_error", extensionPath: "/sanitized/extension.cjs", event: "tool_call", error: "Synthetic extension failure" };
    const records = [...(header ? [release130[0]] : []), failure];
    const events = parse(records).logEntries;
    expect(events.find(event => event.type === "session.error")).toMatchObject({
      extensionPath: failure.extensionPath,
      event: "tool_call",
      data: { error: failure.error, sourceType: "extension_error" },
    });
    expect(selectSessionResult(events)).toMatchObject({ errors: [failure.error] });
    const unified = mergeSessionSources([{ component: "agent", phase: "agent", path: "pi-streaming.jsonl", events }]);
    expect(unified.find(event => event.type === "session.error").data).toMatchObject({ error: failure.error, sourceType: "extension_error" });
  });
});
