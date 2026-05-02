import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { createSharedState } from "../../src/types.js";
import { createSteeringExtension } from "../../src/extensions/steering.js";
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

describe("createSteeringExtension", () => {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  let stderrSpy: any;

  beforeEach(() => {
    stderrSpy = vi.spyOn(process.stderr, "write").mockImplementation(() => true);
  });

  afterEach(() => {
    stderrSpy.mockRestore();
    vi.useRealTimers();
  });

  const defaultSteering = {
    timeWarningMinutes: 5,
    timeCriticalMinutes: 2,
    budgetWarnPercent: 75,
    budgetCriticalPercent: 90,
  };

  it("registers agent_start and turn_end handlers", () => {
    const state = createSharedState("sonnet");
    const { pi, handlers } = buildMockPi();
    createSteeringExtension(60, defaultSteering, state)(pi);
    expect(handlers["agent_start"]).toBeDefined();
    expect(handlers["turn_end"]).toBeDefined();
  });

  it("sets sessionStartMs on agent_start", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-01-01T00:00:00Z"));

    const state = createSharedState("sonnet");
    const { pi, handlers } = buildMockPi();
    createSteeringExtension(60, defaultSteering, state)(pi);

    handlers["agent_start"]?.({});
    expect(state.sessionStartMs).toBe(new Date("2026-01-01T00:00:00Z").getTime());
  });

  it("sends a time warning when remaining time is below timeWarningMinutes", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-01-01T00:00:00Z"));

    const state = createSharedState("sonnet");
    const { pi, handlers, sentMessages } = buildMockPi();
    // Timeout = 60 min, warning at 5 min remaining
    createSteeringExtension(60, defaultSteering, state)(pi);

    handlers["agent_start"]?.({});
    // Advance time to 56 minutes in → 4 minutes remaining (< 5 min warning)
    vi.advanceTimersByTime(56 * 60 * 1000);

    handlers["turn_end"]?.({});

    expect(sentMessages).toHaveLength(1);
    expect(sentMessages[0]?.content).toMatch(/minute|remaining/i);
    expect(sentMessages[0]?.options?.deliverAs).toBe("steer");
  });

  it("sends a critical message when remaining time is below timeCriticalMinutes", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-01-01T00:00:00Z"));

    const state = createSharedState("sonnet");
    const { pi, handlers, sentMessages } = buildMockPi();
    createSteeringExtension(60, defaultSteering, state)(pi);

    handlers["agent_start"]?.({});
    // Advance 59 minutes → 1 minute remaining (< 2 min critical)
    vi.advanceTimersByTime(59 * 60 * 1000);

    handlers["turn_end"]?.({});

    expect(sentMessages.some((m) => m.content.toUpperCase().includes("CRITICAL"))).toBe(true);
  });

  it("does nothing when time is not yet near the warning threshold", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-01-01T00:00:00Z"));

    const state = createSharedState("sonnet");
    const { pi, handlers, sentMessages } = buildMockPi();
    createSteeringExtension(60, defaultSteering, state)(pi);

    handlers["agent_start"]?.({});
    // Only 10 minutes elapsed → plenty of time remaining
    vi.advanceTimersByTime(10 * 60 * 1000);

    handlers["turn_end"]?.({});

    expect(sentMessages).toHaveLength(0);
  });

  it("does not send time warning when sessionStartMs is 0 (agent_start not yet fired)", () => {
    const state = createSharedState("sonnet");
    // Do NOT fire agent_start
    const { pi, handlers, sentMessages } = buildMockPi();
    createSteeringExtension(60, defaultSteering, state)(pi);

    handlers["turn_end"]?.({});

    expect(sentMessages).toHaveLength(0);
  });

  it("sends warning only once even across multiple turns", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-01-01T00:00:00Z"));

    const state = createSharedState("sonnet");
    const { pi, handlers, sentMessages } = buildMockPi();
    createSteeringExtension(60, defaultSteering, state)(pi);

    handlers["agent_start"]?.({});
    vi.advanceTimersByTime(56 * 60 * 1000); // into warning zone

    handlers["turn_end"]?.({});
    handlers["turn_end"]?.({});
    handlers["turn_end"]?.({});

    const warnMessages = sentMessages.filter((m) => !m.content.toUpperCase().includes("CRITICAL"));
    expect(warnMessages).toHaveLength(1);
  });
});
