import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { createRepairExtension } from "../../src/extensions/repair.js";
import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";

type EventHandler = (event: unknown, ctx?: unknown) => void;

function buildMockPi(): {
  pi: ExtensionAPI;
  handlers: Record<string, EventHandler>;
  sentMessages: Array<{ content: string; options: { deliverAs?: string } }>;
} {
  const handlers: Record<string, EventHandler> = {};
  const sentMessages: Array<{ content: string; options: { deliverAs?: string } }> = [];

  const pi = {
    on: vi.fn((event: string, handler: unknown) => {
      handlers[event] = handler as EventHandler;
    }),
    sendUserMessage: vi.fn((content: unknown, options: unknown) => {
      sentMessages.push({
        content: content as string,
        options: (options ?? {}) as { deliverAs?: string },
      });
    }),
  } as unknown as ExtensionAPI;

  return { pi, handlers, sentMessages };
}

describe("createRepairExtension", () => {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  let stderrSpy: any;

  beforeEach(() => {
    stderrSpy = vi.spyOn(process.stderr, "write").mockImplementation(() => true);
  });

  afterEach(() => {
    stderrSpy.mockRestore();
  });

  it("registers tool_result and agent_end handlers", () => {
    const { pi, handlers } = buildMockPi();
    createRepairExtension()(pi);
    expect(handlers["tool_result"]).toBeDefined();
    expect(handlers["agent_end"]).toBeDefined();
  });

  it("emits a JSONL repair event for corrupted tool results", () => {
    const { pi, handlers } = buildMockPi();
    createRepairExtension()(pi);

    const writtenLines: string[] = [];
    stderrSpy.mockImplementation((msg: unknown) => {
      writtenLines.push(String(msg));
      return true;
    });

    handlers["tool_result"]?.({
      toolName: "bash",
      content: "tool_calls[0].type is null",
    });

    const jsonLines = writtenLines.filter((l) => l.trimStart().startsWith("{"));
    expect(jsonLines.some((l) => l.includes("corrupted_tool_result_detected"))).toBe(true);
  });

  it("does not emit repair event for clean tool results", () => {
    const { pi, handlers } = buildMockPi();
    createRepairExtension()(pi);

    const writtenLines: string[] = [];
    stderrSpy.mockImplementation((msg: unknown) => {
      writtenLines.push(String(msg));
      return true;
    });

    handlers["tool_result"]?.({
      toolName: "read",
      content: "file contents here",
    });

    const jsonLines = writtenLines.filter((l) => l.trimStart().startsWith("{"));
    expect(jsonLines.some((l) => l.includes("repair"))).toBe(false);
  });

  it("injects a follow-up message for recoverable errors on agent_end", () => {
    const { pi, handlers, sentMessages } = buildMockPi();
    createRepairExtension()(pi);

    handlers["agent_end"]?.({ error: new Error("rate limit exceeded") });

    expect(sentMessages).toHaveLength(1);
    expect(sentMessages[0]?.options?.deliverAs).toBe("followUp");
    expect(sentMessages[0]?.content).toMatch(/transient error|continue/i);
  });

  it("does not inject a follow-up for non-recoverable errors", () => {
    const { pi, handlers, sentMessages } = buildMockPi();
    createRepairExtension()(pi);

    handlers["agent_end"]?.({ error: new Error("permission denied") });

    expect(sentMessages).toHaveLength(0);
  });

  it("does nothing on agent_end without an error", () => {
    const { pi, handlers, sentMessages } = buildMockPi();
    createRepairExtension()(pi);

    handlers["agent_end"]?.({});

    expect(sentMessages).toHaveLength(0);
  });

  it("treats timeout errors as recoverable", () => {
    const { pi, handlers, sentMessages } = buildMockPi();
    createRepairExtension()(pi);

    handlers["agent_end"]?.({ error: new Error("request timeout") });

    expect(sentMessages).toHaveLength(1);
  });

  it("treats ECONNRESET as recoverable", () => {
    const { pi, handlers, sentMessages } = buildMockPi();
    createRepairExtension()(pi);

    handlers["agent_end"]?.({ error: new Error("ECONNRESET: connection reset") });

    expect(sentMessages).toHaveLength(1);
  });

  it("treats overloaded errors as recoverable", () => {
    const { pi, handlers, sentMessages } = buildMockPi();
    createRepairExtension()(pi);

    handlers["agent_end"]?.({ error: new Error("API overloaded, please retry") });

    expect(sentMessages).toHaveLength(1);
  });
});
