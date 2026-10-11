import { afterEach, beforeEach, describe, expect, it } from "vitest";
import fs from "fs";
import path from "path";
import { createRequire } from "module";

const req = createRequire(import.meta.url);
const { parseSessionLogs } = req("./generate_usage_activity_summary.cjs");
const { normalizeCopilotSession } = req("./copilot_session.cjs");
const { sessionToolSuccess } = req("./agent_session.cjs");

describe("Copilot usage summary consumer", () => {
  let root;
  beforeEach(() => {
    root = path.join(process.cwd(), `.usage-summary-fixture-${process.pid}-${Date.now()}`);
    fs.mkdirSync(root);
  });
  afterEach(() => fs.rmSync(root, { recursive: true, force: true }));

  function summarize(events, name = "native") {
    const dir = path.join(root, name);
    fs.mkdirSync(dir, { recursive: true });
    fs.writeFileSync(path.join(dir, "events.jsonl"), events.map(JSON.stringify).join("\n"));
    return parseSessionLogs([dir]);
  }

  function compareCanonical(events) {
    const original = structuredClone(events);
    const native = summarize(events);
    const canonical = summarize(normalizeCopilotSession(events), "canonical");
    expect(events).toEqual(original);
    expect(canonical).toEqual(native);
    return native;
  }

  it.each(["input", "arguments", "parameters", "args", "toolInput"])("extracts %s without capture-time enrichment", alias => {
    const result = compareCanonical([
      { type: "session.start", data: { sessionId: "root" } },
      { type: "tool.execution_start", timestamp: "2026-10-11T00:00:00Z", data: { toolCallId: "", toolName: "skill", [alias]: { skill: "documentation", private: "not retained" } } },
      { type: "tool.execution_complete", data: { toolCallId: "", success: false } },
    ]);
    expect(result).toMatchObject({
      total_events: 3,
      tool_execution_starts: 1,
      tool_execution_completes: 1,
      failed_tool_executions: 1,
      skills: { total_invocations: 1, items: [{ name: "documentation", invocation_count: 1, failed_count: 1 }] },
    });
    expect(JSON.stringify(result)).not.toContain("not retained");
  });

  it.each(["", 0, false, null])("correlates explicit falsy ID %j only once", id => {
    const result = compareCanonical([
      { type: "tool.execution_start", data: { toolCallId: id, toolName: "skill", parameters: { skill: "documentation" } } },
      { type: "tool.execution_complete", data: { toolCallId: id, success: false } },
      { type: "tool.execution_complete", data: { toolCallId: id, success: false } },
    ]);
    expect(result.tool_execution_completes).toBe(2);
    expect(result.skills.items[0].failed_count).toBe(1);
  });

  it("keeps reused root, child, session and provenance IDs independent", () => {
    const skill = (name, context = {}) => ({ type: "tool.execution_start", data: { ...context, toolCallId: "reused", toolName: "skill", args: { skill: name } } });
    const complete = (success, context = {}) => ({ type: "tool.execution_complete", data: { ...context, toolCallId: "reused", success } });
    const result = compareCanonical([
      { type: "session.start", data: { sessionId: "first" } },
      skill("first-root"),
      { type: "session.start", data: { sessionId: "child-session", agentId: "child", parentToolCallId: "parent" } },
      skill("child", { agentId: "child", parentToolCallId: "parent" }),
      { type: "session.start", data: { sessionId: "second" } },
      skill("second-root"),
      { ...skill("detector"), provenance: { component: "detection", phase: "agent", path: "detector.jsonl" } },
      complete(false, { agentId: "child", parentToolCallId: "parent" }),
      complete(true),
      complete(false, { sessionId: "first" }),
      { ...complete(false), provenance: { component: "detection", phase: "agent", path: "detector.jsonl" } },
    ]);
    expect(result.skills.items.map(({ name, failed_count }) => [name, failed_count])).toEqual([
      ["child", 1],
      ["detector", 1],
      ["first-root", 1],
      ["second-root", 0],
    ]);
  });

  it("does not pair a reused ID across event files", () => {
    const first = [{ type: "tool.execution_start", data: { toolCallId: "same", toolName: "skill", input: { skill: "first" } } }];
    const second = [
      { type: "tool.execution_start", data: { toolCallId: "same", toolName: "view" } },
      { type: "tool.execution_complete", data: { toolCallId: "same", success: false } },
    ];
    summarize(first, "a");
    summarize(second, "b");
    expect(parseSessionLogs([root]).skills.items[0].failed_count).toBe(0);
  });

  it("correlates ID-less tools conservatively and does not borrow identified starts", () => {
    const result = compareCanonical([
      { type: "tool.execution_start", data: { toolCallId: "", toolName: "skill", input: { skill: "identified" } } },
      { type: "tool.execution_start", data: { toolName: "skill", input: { skill: "anonymous" } } },
      { type: "tool.execution_start", data: { toolName: "view" } },
      { type: "tool.execution_complete", data: { success: false } },
      { type: "tool.execution_complete", data: { toolName: "view", success: true } },
      { type: "tool.execution_complete", data: { success: false } },
      { type: "tool.execution_complete", data: { success: false } },
      { type: "tool.execution_complete", data: { toolCallId: "", success: true } },
    ]);
    expect(result.skills.items.map(({ name, failed_count }) => [name, failed_count])).toEqual([
      ["anonymous", 1],
      ["identified", 0],
    ]);
  });

  it("preserves unknown outcomes and counts explicit error evidence", () => {
    const outcomes = [{}, { success: true }, { success: false }, { success: true, result: { isError: true } }, { shellExecution: { exitCode: 2 } }, { error: "failed" }];
    const events = outcomes.flatMap((outcome, index) => [
      { type: "tool.execution_start", data: { toolCallId: index, toolName: "skill", toolInput: { skill: `skill-${index}` } } },
      { type: "tool.execution_complete", data: { toolCallId: index, ...outcome } },
    ]);
    const normalized = normalizeCopilotSession(events);
    expect(normalized[1].data).not.toHaveProperty("success");
    expect(sessionToolSuccess(normalized[1].data)).toBeUndefined();
    const result = compareCanonical(events);
    expect(result.failed_tool_executions).toBe(4);
    expect(result.skills.items.map(item => item.failed_count)).toEqual([0, 0, 1, 1, 1, 1]);
  });

  it("preserves input precedence and chronological timestamps", () => {
    const result = compareCanonical([
      { type: "tool.execution_start", timestamp: "2026-10-11T00:00:03Z", data: { toolName: "skill", input: { skill: "documentation" }, parameters: { skill: "wrong" } } },
      { type: "tool.execution_complete", data: { toolName: "skill", success: true } },
      { type: "tool.execution_start", timestamp: "2026-10-11T00:00:01Z", data: { toolName: "skill", arguments: { skill: "documentation" } } },
      { type: "tool.execution_complete", data: { toolName: "skill", success: false } },
      { type: "tool.execution_start", data: { toolName: "skill", input: null, arguments: { skill: "wrong" } } },
    ]);
    expect(result.skills).toEqual({
      total_invocations: 2,
      unique_skills: 1,
      items: [{ name: "documentation", invocation_count: 2, failed_count: 1, first_timestamp: "2026-10-11T00:00:01Z", last_timestamp: "2026-10-11T00:00:03Z" }],
    });
  });
});
