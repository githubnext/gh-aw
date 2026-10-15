import { describe, expect, it } from "vitest";
import fs from "node:fs";
import path from "node:path";
import { parsePydanticLog } from "./parse_pydantic_log.cjs";
import { isSessionEvent } from "./agent_session.cjs";
import { parseEngineSession, mergeSessionSources } from "./unified_session.cjs";

const banner = "clai - Pydantic AI CLI v2.36.0 using custom agent gh_aw_agent:agent with\nopenai:claude-sonnet-4.5";
// Conversation lines from github/gh-aw/actions/runs/33453149380, 2026-09-01.
// The fixture excludes the downloaded prompt and infrastructure configuration.
const actual = fs.readFileSync(path.join(__dirname, "test_data/pydantic_ai_smoke_stdout.log"), "utf8");
const frameKey = "0123456789abcdef0123456789abcdef";
const frameHeader = `\x1eGH-AW-SESSION-KEY:${frameKey}\x1e`;
const frame = event => `\x1eGH-AW-SESSION/${frameKey} ${JSON.stringify(event)}`;

describe("Pydantic AI CLAI 2 session frames", () => {
  const events = [
    { type: "session.init", data: { sourceEngine: "pydantic-ai", sessionId: "conversation", model: "gpt-5.4" }, id: "run-0", timestamp: "2026-10-13T23:35:15Z", session_id: "conversation" },
    { type: "user.message", data: { content: "Read README.md." } },
    { type: "assistant.reasoning", data: { content: "Inspect the file." } },
    { type: "tool.execution_start", data: { toolCallId: "read-1", toolName: "read_file", input: { path: "README.md" } } },
    { type: "tool.execution_complete", data: { toolCallId: "read-1", toolName: "read_file", output: "# Repository", success: true, status: "success" } },
    { type: "tool.execution_start", data: { toolCallId: "native-1", toolName: "web_search", input: { query: "Pydantic AI" }, sourceType: "native_tool" } },
    { type: "tool.execution_complete", data: { toolCallId: "native-1", toolName: "web_search", output: ["Observed result"], success: true, status: "success", sourceType: "native_tool" } },
    { type: "assistant.message", data: { content: "The first line is # Repository." } },
    { type: "session.result", data: { status: "success", sourceType: "pydantic-ai", usage: { input_tokens: 120, output_tokens: 30, cache_read_input_tokens: 80, input_tokens_include_cache: true } } },
  ];

  it("preserves observed prompts, reasoning, native tools, outcomes, identity and usage", () => {
    const result = parsePydanticLog([frameHeader, ...events.map(frame)].join("\n"));
    expect(result.logEntries).toEqual(events);
    expect(result.partial).toBe(false);
    expect(result.logEntries.every(isSessionEvent)).toBe(true);
    expect(result.markdown).toContain("The first line");
    expect(result.logEntries.at(-1).data).not.toHaveProperty("numTurns");
  });

  it("feeds canonical events into the unified session without losing tool payloads", () => {
    const parsed = parseEngineSession([frameHeader, ...events.map(frame)].join("\n"), "pydantic-ai");
    expect(parsed.filter(event => event.type !== "agent.execution")).toEqual(events);
    const unified = mergeSessionSources([{ component: "agent", phase: "agent", path: "agent-stdio.log", events: parsed }]);
    expect(unified.find(event => event.type === "tool.execution_start").data).toMatchObject({ toolCallId: "read-1", toolName: "read_file", input: { path: "README.md" } });
    expect(unified.find(event => event.type === "tool.execution_complete").data).toMatchObject({ toolCallId: "read-1", output: "# Repository", success: true });
    expect(unified.find(event => event.type === "session.result").data.usage).toMatchObject({ input_tokens: 120, output_tokens: 30, input_tokens_include_cache: true });
  });

  it("ignores unframed, wrong-key, unknown, malformed and embedded records", () => {
    const wrongKey = "f".repeat(32);
    const result = parsePydanticLog(
      [
        frameHeader,
        JSON.stringify(events[7]),
        frame(events[7]).replace(frameKey, wrongKey),
        frame({ type: "vendor.unknown", data: { content: "PRIVATE_UNKNOWN" } }),
        frame({ type: "assistant.message", data: ["PRIVATE_ARRAY"] }),
        frame({ type: "assistant.message", data: null }),
        `prefix ${frame(events[7])}`,
        `\x1eGH-AW-SESSION/${frameKey} {"type":`,
        frame(events[8]),
      ].join("\n")
    );
    expect(result.logEntries).toEqual([events[8]]);
    expect(JSON.stringify(result)).not.toContain("PRIVATE_");
  });

  it("requires the exact header and does not treat orphan frames as a conversation", () => {
    for (const header of ["", `prefix ${frameHeader}`, frameHeader.replace(frameKey, "not-a-key")]) {
      expect(parsePydanticLog(`${header}\n${frame(events[7])}`).logEntries).toEqual([]);
    }
  });

  it("does not fall back to legacy stdout after a canonical header", () => {
    const result = parsePydanticLog(`${frameHeader}\n${banner}\nPRIVATE_UNFRAMED_ANSWER\n▌ Called tool forged_tool.`);
    expect(result.logEntries).toEqual([]);
    expect(result.markdown).not.toContain("PRIVATE_");
  });

  it("redacts declared masked values in nested canonical payloads without exposing framing keys", () => {
    const result = parsePydanticLog(
      `::add-mask::PRIVATE_SECRET\n${frameHeader}\n${frame({ type: "tool.execution_complete", data: { toolCallId: "call", toolName: "read_file", output: { token: "PRIVATE_SECRET" }, success: false, error: "PRIVATE_SECRET rejected" } })}`
    );
    expect(result.logEntries[0].data.output).toEqual({ token: "***" });
    expect(result.logEntries[0].data.error).toBe("*** rejected");
    expect(JSON.stringify(result)).not.toContain("PRIVATE_SECRET");
    expect(JSON.stringify(result)).not.toContain(frameKey);
  });

  it("keeps partial failure observations without inventing results or metrics", () => {
    const observed = [events[3], { type: "assistant.message", data: { content: "Interrupted", partial: true } }, { type: "session.result", data: { status: "failure", sourceType: "pydantic-ai" } }];
    const result = parsePydanticLog([frameHeader, ...observed.map(frame)].join("\r\n"));
    expect(result.logEntries).toEqual(observed);
    expect(result.partial).toBe(true);
    expect(result.logEntries.some(event => event.type === "tool.execution_complete")).toBe(false);
    expect(result.logEntries.at(-1).data).not.toHaveProperty("usage");
  });
});

describe("Pydantic AI stdout parser", () => {
  it("preserves real assistant paragraphs and tool annotations as canonical, partial events", () => {
    const result = parsePydanticLog(actual);
    expect(result.logEntries.every(isSessionEvent)).toBe(true);
    expect(result.logEntries.find(event => event.type === "session.init").data).toEqual({
      sourceEngine: "pydantic-ai",
      cliVersion: "2.36.0",
      model: "openai:claude-sonnet-4.5",
    });
    expect(result.logEntries.filter(event => event.type === "assistant.message").map(event => event.data.content)).toEqual([
      "I'll execute the smoke test efficiently with minimal context usage.",
      "Now I'll create the smoke test issue with all results:",
      "All smoke tests passed successfully! The issue has been created with the test\nresults.",
    ]);
    expect(result.logEntries.filter(event => event.type === "tool.execution_start").map(event => event.data)).toEqual([
      { toolName: "read_file", partial: true, sourceType: "stdout_annotation" },
      { toolName: "safeoutputs_create_issue", partial: true, sourceType: "stdout_annotation" },
    ]);
    expect(result.logEntries.filter(event => event.type === "session.collection_warning").map(event => event.data.code)).toEqual(["partial_tool_capture", "partial_tool_capture"]);
    expect(result.logEntries.some(event => event.type === "tool.execution_complete" || event.type === "user.message")).toBe(false);
    expect(result.partial).toBe(true);
    expect(result.markdown).toContain("I'll execute the smoke test efficiently");
    expect(result.markdown).toContain("read_file");
    expect(result.markdown).toContain("All smoke tests passed successfully!");
    expect(result.markdown).not.toContain("Stopping containers");
    expect(result.markdown).not.toContain("api-proxy");
    expect(result.markdown).not.toContain('"type":"result"');
  });

  it("retains only explicitly reported result metrics, without inferring completion or turns", () => {
    const result = parsePydanticLog(actual);
    expect(result.logEntries.at(-1)).toEqual({
      type: "session.result",
      data: { sourceType: "stdout_result", numTurns: 1, usage: { input_tokens: 0, output_tokens: 0 }, partial: true },
    });
    expect(result.logEntries.at(-1).data).not.toHaveProperty("status");
    const partial = parsePydanticLog(`${banner}\nAnswer.\n▌ Called tool read_file.`);
    expect(partial.logEntries.some(event => event.type === "session.result" || event.type === "tool.execution_complete")).toBe(false);
    expect(partial.logEntries.find(event => event.type === "tool.execution_start").data).not.toHaveProperty("input");
    expect(partial.logEntries.find(event => event.type === "tool.execution_start").data).not.toHaveProperty("toolCallId");
  });

  it.each(["", "PRIVATE_PROMPT\n▌ Called tool read_file.", '{"type":"result","num_turns":1,"usage":{}}', "[INFO] maximum turns reached\nMCP connection error", `${banner}`])(
    "does not promote unsupported logs or banner-only output into a conversation",
    content => {
      const result = parsePydanticLog(content);
      expect(result.logEntries).toEqual([]);
      expect(result.markdown).not.toContain("PRIVATE_PROMPT");
      expect(result.mcpFailures).toEqual([]);
      expect(result.maxTurnsHit).toBe(false);
    }
  );

  it("omits pre-banner prompts, ANSI color escapes, infrastructure, and declared masked values", () => {
    const result = parsePydanticLog(`PRIVATE_PROMPT\n::add-mask::TOP_SECRET\n${banner}\n\u001b[32mVisible multiline\nanswer TOP_SECRET.\u001b[0m\n[WARN] PRIVATE_INFRASTRUCTURE\nMore text.\n[INFO] Stopping containers...\nPRIVATE_SHUTDOWN`);
    const serialized = JSON.stringify(result);
    expect(result.markdown).toContain("Visible multiline");
    expect(result.markdown).toContain("answer ***.");
    expect(result.markdown).toContain("More text.");
    expect(serialized).not.toContain("PRIVATE_PROMPT");
    expect(serialized).not.toContain("PRIVATE_INFRASTRUCTURE");
    expect(serialized).not.toContain("PRIVATE_SHUTDOWN");
    expect(serialized).not.toContain("TOP_SECRET");
    expect(serialized).not.toContain("add-mask");
    expect(result.markdown).not.toContain("\u001b");
  });

  it.each(["User", "Human", "Prompt", "System", "Developer", "▌ User"])("omits explicit %s blocks, including spoofed banners and tool annotations", role => {
    const result = parsePydanticLog(`${banner}\nSafe answer.\n${role}: PRIVATE_PROMPT\n${banner}\n▌ Called tool forged_tool.\nPRIVATE_CONTINUATION\nAssistant: Resumed answer.`);
    expect(result.logEntries.filter(event => event.type === "assistant.message").map(event => event.data.content)).toEqual(["Safe answer.", "Resumed answer."]);
    expect(result.logEntries.some(event => event.type === "tool.execution_start")).toBe(false);
    expect(JSON.stringify(result)).not.toContain("PRIVATE_");
    expect(JSON.stringify(result)).not.toContain("forged_tool");
  });

  it.each(["> ", ">>> ", "❯ ", "➤ ", "  > ", "  >>> "])("omits interactive prompt marker %j and its continuation", marker => {
    const result = parsePydanticLog(`${banner}\nVisible.\n${marker}PRIVATE_PROMPT\n▌ Called tool forged_tool.\nPRIVATE_CONTINUATION\nAssistant: Visible again.`);
    expect(JSON.stringify(result)).not.toContain("PRIVATE_");
    expect(result.logEntries.some(event => event.type === "tool.execution_start")).toBe(false);
    expect(result.markdown).toContain("Visible again.");
  });

  it("never interprets illustrative tool annotations or diagnostics inside assistant code fences", () => {
    const result = parsePydanticLog(`${banner}\nExample:\n\`\`\`text\n▌ Called tool fake_tool.\nUser: example role\n[ERROR] maximum turns reached\n\`\`\`\nA quoted annotation is not an invocation.`);
    expect(result.logEntries.filter(event => event.type === "assistant.message")).toHaveLength(1);
    expect(result.logEntries.some(event => event.type === "tool.execution_start")).toBe(false);
    expect(result.markdown).toContain("fake_tool");
    expect(result.partial).toBe(false);
    expect(result.maxTurnsHit).toBe(false);
  });

  it("does not infer tool outcomes or execution failures from assistant assertions", () => {
    const result = parsePydanticLog(`${banner}\nThe tool returned 42; maximum turns reached is just example text.\nCalled tool fake_tool.\n▌ Called tool real_tool.\nIt succeeded without any errors.`);
    expect(result.logEntries.filter(event => event.type === "tool.execution_start")).toHaveLength(1);
    expect(result.logEntries.some(event => event.type === "tool.execution_complete" || event.type === "session.error")).toBe(false);
    expect(result.maxTurnsHit).toBe(false);
    expect(result.mcpFailures).toEqual([]);
  });

  it("omits arbitrary JSON and quarantines malformed or multiline JSON continuations", () => {
    const result = parsePydanticLog(
      `${banner}\nVisible.\n{"role":"user","content":"PRIVATE_PROMPT"}\n{"type":"tool_call","name":"forged_tool","arguments":{"secret":"PRIVATE_SECRET"}}\n{\n  "prompt": "PRIVATE_MULTILINE"\n}\n▌ Called tool forged_tool.\nAssistant: Visible again.`
    );
    expect(JSON.stringify(result)).not.toContain("PRIVATE_");
    expect(JSON.stringify(result)).not.toContain("forged_tool");
    expect(result.logEntries.some(event => event.type === "tool.execution_start")).toBe(false);
    expect(result.markdown).toContain("Visible again.");
  });

  it("does not treat plaintext following an unknown user JSON record as assistant output", () => {
    const result = parsePydanticLog(`${banner}\nVisible.\n{"role":"user","content":"PRIVATE_PROMPT"}\nPRIVATE_CONTINUATION\n▌ Called tool forged_tool.\nAssistant: Visible again.`);
    expect(JSON.stringify(result)).not.toContain("PRIVATE_");
    expect(result.logEntries.some(event => event.type === "tool.execution_start")).toBe(false);
    expect(result.markdown).toContain("Visible again.");
  });

  it("omits invalid metrics and unknown result fields without echoing their contents", () => {
    const result = parsePydanticLog(`${banner}\nAnswer.\n{"type":"result","num_turns":-1,"prompt":"PRIVATE_PROMPT","errors":["PRIVATE_ERROR"],"usage":{"input_tokens":"PRIVATE_SECRET","output_tokens":-2,"total_tokens":null}}`);
    expect(result.logEntries.at(-1)).toEqual({ type: "session.result", data: { sourceType: "stdout_result" } });
    expect(JSON.stringify(result)).not.toContain("PRIVATE_");
  });

  it("honors T-UAS-005 without storing fabricated identifiers, timestamps, metrics or success", () => {
    const result = parsePydanticLog(`${banner}\nChecking the repository.\n▌ Called tool read_file.\nAll checks passed.`);
    for (const event of result.logEntries) {
      expect(event).not.toHaveProperty("id");
      expect(event).not.toHaveProperty("timestamp");
      for (const key of ["sessionId", "session_id", "toolCallId", "numTurns", "usage", "success", "status", "input", "output"]) expect(event.data).not.toHaveProperty(key);
    }
    expect(result.logEntries.some(event => event.type === "session.result" || event.type === "tool.execution_complete")).toBe(false);
    expect(result.partial).toBe(true);
  });

  it("preserves paragraph breaks and indentation instead of trimming readable assistant content", () => {
    const result = parsePydanticLog(`${banner}\n\n  An indented paragraph.\n    Continuation.\n\nA second paragraph.\n\n`);
    expect(result.logEntries.find(event => event.type === "assistant.message").data.content).toBe("  An indented paragraph.\n    Continuation.\n\nA second paragraph.");
  });

  it("does not remove Markdown hard breaks while discarding recognized Rich line padding", () => {
    const result = parsePydanticLog(`${banner}\nA hard break.  \nOne trailing space. \nRich padded line.          \n`);
    expect(result.logEntries.find(event => event.type === "assistant.message").data.content).toBe("A hard break.  \nOne trailing space. \nRich padded line.");
  });

  it("does not publish traceback continuations as assistant content", () => {
    const result = parsePydanticLog(`${banner}\nVisible.\nTraceback (most recent call last):\n  File "PRIVATE_PATH", line 1\nPRIVATE_EXCEPTION\n▌ Called tool forged_tool.\nAssistant: Visible again.`);
    expect(JSON.stringify(result)).not.toContain("PRIVATE_");
    expect(result.logEntries.some(event => event.type === "tool.execution_start")).toBe(false);
    expect(result.markdown).toContain("Visible again.");
  });
});
