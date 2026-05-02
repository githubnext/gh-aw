import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { existsSync, readFileSync, rmSync } from "node:fs";
import { createSharedState } from "../../src/types.js";
import { createObservabilityExtension, CONTEXT_PROVENANCE_PATH } from "../../src/extensions/observability.js";
import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";
import type { ImportEntry } from "../../src/types.js";

type EventHandler = (event: unknown, ctx?: unknown) => void;

function buildMockPi(): {
  pi: ExtensionAPI;
  handlers: Record<string, EventHandler>;
} {
  const handlers: Record<string, EventHandler> = {};
  const pi = {
    on: vi.fn((event: string, handler: unknown) => {
      handlers[event] = handler as EventHandler;
    }),
    sendUserMessage: vi.fn(),
  } as unknown as ExtensionAPI;
  return { pi, handlers };
}

describe("createObservabilityExtension", () => {
  let stderrLines: string[];
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  let stderrSpy: any;

  beforeEach(() => {
    stderrLines = [];
    stderrSpy = vi.spyOn(process.stderr, "write").mockImplementation((msg: unknown) => {
      stderrLines.push(String(msg));
      return true;
    });
    // Clean up provenance file before each test
    try {
      rmSync(CONTEXT_PROVENANCE_PATH, { force: true });
    } catch {
      // ignore
    }
  });

  afterEach(() => {
    stderrSpy.mockRestore();
    delete process.env["GITHUB_STEP_SUMMARY"];
  });

  it("registers all required event handlers", () => {
    const state = createSharedState("claude-sonnet-4.6");
    const { pi, handlers } = buildMockPi();
    createObservabilityExtension([], undefined, state, "prompt text")(pi);

    expect(handlers["agent_start"]).toBeDefined();
    expect(handlers["model_select"]).toBeDefined();
    expect(handlers["turn_end"]).toBeDefined();
    expect(handlers["tool_execution_end"]).toBeDefined();
    expect(handlers["agent_end"]).toBeDefined();
  });

  it("emits session_start JSONL on agent_start", () => {
    const state = createSharedState("claude-sonnet-4.6");
    const { pi, handlers } = buildMockPi();
    createObservabilityExtension([], undefined, state, "prompt")(pi);

    handlers["agent_start"]?.({});

    const jsonLines = stderrLines.filter((l) => l.trimStart().startsWith("{"));
    const sessionStart = jsonLines
      .map((l) => { try { return JSON.parse(l) as Record<string, unknown>; } catch { return null; } })
      .find((r) => r?.["event"] === "session_start");
    expect(sessionStart).toBeDefined();
    expect(sessionStart?.["model"]).toBe("claude-sonnet-4.6");
  });

  it("emits turn_end JSONL with token info", () => {
    const state = createSharedState("sonnet");
    state.turnCount = 1;
    state.cumulativeTokens = 1500;
    state.cumulativeCostUsd = 0.005;

    const { pi, handlers } = buildMockPi();
    createObservabilityExtension([], undefined, state, "")(pi);

    handlers["turn_end"]?.({
      message: {
        role: "assistant",
        usage: { input: 1000, output: 500, totalTokens: 1500 },
      },
    });

    const jsonLines = stderrLines.filter((l) => l.trimStart().startsWith("{"));
    const turnEnd = jsonLines
      .map((l) => { try { return JSON.parse(l) as Record<string, unknown>; } catch { return null; } })
      .find((r) => r?.["event"] === "turn_end");
    expect(turnEnd).toBeDefined();
    expect(turnEnd?.["input_tokens"]).toBe(1000);
    expect(turnEnd?.["output_tokens"]).toBe(500);
  });

  it("emits a human-readable per-turn line to stderr", () => {
    const state = createSharedState("sonnet");
    state.turnCount = 2;
    state.cumulativeTokens = 2000;

    const { pi, handlers } = buildMockPi();
    createObservabilityExtension([], undefined, state, "")(pi);

    handlers["turn_end"]?.({
      message: { role: "assistant", usage: { input: 800, output: 200 } },
    });

    const humanLine = stderrLines.find((l) => l.startsWith("> **Turn"));
    expect(humanLine).toBeDefined();
    expect(humanLine).toContain("Turn 2");
  });

  it("writes context provenance file on agent_end", () => {
    const imports: ImportEntry[] = [
      { path: "skills/foo/SKILL.md", content: "Skill content" },
    ];
    const state = createSharedState("sonnet");
    const { pi, handlers } = buildMockPi();
    createObservabilityExtension(imports, undefined, state, "Prompt body")(pi);

    handlers["agent_start"]?.({});
    handlers["agent_end"]?.({});

    expect(existsSync(CONTEXT_PROVENANCE_PATH)).toBe(true);
    const content = readFileSync(CONTEXT_PROVENANCE_PATH, "utf8");
    const records = content
      .trim()
      .split("\n")
      .filter(Boolean)
      .map((l) => JSON.parse(l) as Record<string, unknown>);

    expect(records.some((r) => r["source"] === "import" && r["path"] === "skills/foo/SKILL.md")).toBe(true);
    expect(records.some((r) => r["source"] === "prompt")).toBe(true);
  });

  it("writes GitHub step summary when GITHUB_STEP_SUMMARY is set", () => {
    const summaryPath = `/tmp/aw-test-summary-${Date.now()}.md`;
    process.env["GITHUB_STEP_SUMMARY"] = summaryPath;

    const state = createSharedState("claude-sonnet-4.6");
    const { pi, handlers } = buildMockPi();
    createObservabilityExtension([], undefined, state, "prompt")(pi);

    handlers["agent_start"]?.({});
    handlers["agent_end"]?.({});

    expect(existsSync(summaryPath)).toBe(true);
    const content = readFileSync(summaryPath, "utf8");
    expect(content).toContain("AW Harness Run");
    expect(content).toContain("EXPERIMENTAL");
    expect(content).toContain("claude-sonnet-4.6");

    rmSync(summaryPath, { force: true });
  });

  it("updates model from model_select event", () => {
    const state = createSharedState("initial-model");
    const { pi, handlers } = buildMockPi();
    createObservabilityExtension([], undefined, state, "")(pi);

    handlers["model_select"]?.({ model: { id: "gpt-4-updated" } });

    expect(state.model).toBe("gpt-4-updated");
  });
});
