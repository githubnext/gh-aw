import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { createSharedState } from "../../src/types.js";
import { createCostTrackerExtension } from "../../src/extensions/cost-tracker.js";
import type { ExtensionAPI, ExtensionContext } from "@mariozechner/pi-coding-agent";

// ─── Minimal mock helpers ─────────────────────────────────────────────────────

type TurnEndHandler = (
  event: { message: Record<string, unknown> },
  ctx: Partial<ExtensionContext>,
) => void;

function buildMockPi(): {
  pi: ExtensionAPI;
  turnEndHandlers: TurnEndHandler[];
  sentMessages: Array<{ content: string; options: { deliverAs?: string } }>;
} {
  const turnEndHandlers: TurnEndHandler[] = [];
  const sentMessages: Array<{ content: string; options: { deliverAs?: string } }> = [];

  const pi = {
    on: vi.fn((event: string, handler: unknown) => {
      if (event === "turn_end") turnEndHandlers.push(handler as TurnEndHandler);
    }),
    sendUserMessage: vi.fn((content: unknown, options: unknown) => {
      sentMessages.push({
        content: content as string,
        options: (options ?? {}) as { deliverAs?: string },
      });
    }),
  } as unknown as ExtensionAPI;

  return { pi, turnEndHandlers, sentMessages };
}

function buildMockCtx(tokens: number | null = null): Partial<ExtensionContext> {
  return {
    getContextUsage: () =>
      tokens != null ? { tokens, contextWindow: 200_000, percent: tokens / 2000 } : undefined,
  };
}

function makeTurnEndEvent(
  role: "assistant" | "user" | "toolResult",
  usage?: { input?: number; output?: number; totalTokens?: number; cost?: { total?: number } },
): { message: Record<string, unknown> } {
  return { message: { role, usage } };
}

// ─── Tests ───────────────────────────────────────────────────────────────────

describe("createCostTrackerExtension", () => {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  let stderrSpy: any;

  beforeEach(() => {
    stderrSpy = vi.spyOn(process.stderr, "write").mockImplementation(() => true);
  });

  afterEach(() => {
    stderrSpy.mockRestore();
  });

  it("registers a turn_end handler", () => {
    const state = createSharedState("sonnet");
    const { pi } = buildMockPi();
    const factory = createCostTrackerExtension(undefined, { budgetWarnPercent: 75, budgetCriticalPercent: 90, timeWarningMinutes: 5, timeCriticalMinutes: 2 }, state);
    factory(pi);
    expect((pi.on as ReturnType<typeof vi.fn>).mock.calls.some(([e]) => e === "turn_end")).toBe(true);
  });

  it("increments turnCount on each turn_end", () => {
    const state = createSharedState("sonnet");
    const steering = { budgetWarnPercent: 75, budgetCriticalPercent: 90, timeWarningMinutes: 5, timeCriticalMinutes: 2 };
    const { pi, turnEndHandlers } = buildMockPi();
    const factory = createCostTrackerExtension(undefined, steering, state);
    factory(pi);

    const ctx = buildMockCtx();
    turnEndHandlers[0]?.(makeTurnEndEvent("assistant", { input: 100, output: 50, totalTokens: 150 }), ctx);
    expect(state.turnCount).toBe(1);
    turnEndHandlers[0]?.(makeTurnEndEvent("assistant", { input: 100, output: 50, totalTokens: 150 }), ctx);
    expect(state.turnCount).toBe(2);
  });

  it("accumulates token counts from assistant messages", () => {
    const state = createSharedState("sonnet");
    const steering = { budgetWarnPercent: 75, budgetCriticalPercent: 90, timeWarningMinutes: 5, timeCriticalMinutes: 2 };
    const { pi, turnEndHandlers } = buildMockPi();
    const factory = createCostTrackerExtension(undefined, steering, state);
    factory(pi);

    const ctx = buildMockCtx();
    turnEndHandlers[0]?.(makeTurnEndEvent("assistant", { input: 1000, output: 500, totalTokens: 1500 }), ctx);
    expect(state.cumulativeTokens).toBe(1500);
    turnEndHandlers[0]?.(makeTurnEndEvent("assistant", { input: 500, output: 200, totalTokens: 700 }), ctx);
    expect(state.cumulativeTokens).toBe(2200);
  });

  it("sends a budget warning at budgetWarnPercent", () => {
    const state = createSharedState("sonnet");
    const budget = { maxEffectiveTokens: 1000 };
    const steering = { budgetWarnPercent: 75, budgetCriticalPercent: 90, timeWarningMinutes: 5, timeCriticalMinutes: 2 };
    const { pi, turnEndHandlers, sentMessages } = buildMockPi();
    const factory = createCostTrackerExtension(budget, steering, state);
    factory(pi);

    const ctx = buildMockCtx();
    // 800 tokens = 80% > warn threshold (75%)
    turnEndHandlers[0]?.(makeTurnEndEvent("assistant", { totalTokens: 800 }), ctx);

    expect(sentMessages).toHaveLength(1);
    expect(sentMessages[0]?.content).toMatch(/75|80|budget/i);
    expect(sentMessages[0]?.options?.deliverAs).toBe("steer");
  });

  it("sends a critical message and sets budgetAborted at budgetCriticalPercent", () => {
    const state = createSharedState("sonnet");
    const budget = { maxEffectiveTokens: 1000 };
    const steering = { budgetWarnPercent: 75, budgetCriticalPercent: 90, timeWarningMinutes: 5, timeCriticalMinutes: 2 };
    const { pi, turnEndHandlers, sentMessages } = buildMockPi();
    const factory = createCostTrackerExtension(budget, steering, state);
    factory(pi);

    const ctx = buildMockCtx();
    // 950 tokens = 95% > critical threshold (90%)
    turnEndHandlers[0]?.(makeTurnEndEvent("assistant", { totalTokens: 950 }), ctx);

    expect(state.budgetAborted).toBe(true);
    expect(sentMessages.some((m) => m.content.includes("CRITICAL"))).toBe(true);
  });

  it("does not send messages when no budget is configured", () => {
    const state = createSharedState("sonnet");
    const steering = { budgetWarnPercent: 75, budgetCriticalPercent: 90, timeWarningMinutes: 5, timeCriticalMinutes: 2 };
    const { pi, turnEndHandlers, sentMessages } = buildMockPi();
    const factory = createCostTrackerExtension(undefined, steering, state);
    factory(pi);

    const ctx = buildMockCtx();
    turnEndHandlers[0]?.(makeTurnEndEvent("assistant", { totalTokens: 99_999 }), ctx);

    expect(sentMessages).toHaveLength(0);
    expect(state.budgetAborted).toBe(false);
  });

  it("sends the warning message only once even with multiple turns", () => {
    const state = createSharedState("sonnet");
    const budget = { maxEffectiveTokens: 1000 };
    const steering = { budgetWarnPercent: 75, budgetCriticalPercent: 90, timeWarningMinutes: 5, timeCriticalMinutes: 2 };
    const { pi, turnEndHandlers, sentMessages } = buildMockPi();
    const factory = createCostTrackerExtension(budget, steering, state);
    factory(pi);

    const ctx = buildMockCtx();
    // First turn: hit warn
    turnEndHandlers[0]?.(makeTurnEndEvent("assistant", { totalTokens: 800 }), ctx);
    // Second turn: still above warn, below critical
    turnEndHandlers[0]?.(makeTurnEndEvent("assistant", { totalTokens: 10 }), ctx);

    const warnMessages = sentMessages.filter((m) => !m.content.includes("CRITICAL"));
    expect(warnMessages).toHaveLength(1);
  });
});
