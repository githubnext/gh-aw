import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

describe("pi_steering_extension.cjs", () => {
  let piSteeringExtension, loadSteeringConfig;
  let originalEnv;
  let stderrOutput;

  beforeEach(async () => {
    originalEnv = { ...process.env };

    // Capture stderr writes
    stderrOutput = [];
    vi.spyOn(process.stderr, "write").mockImplementation(msg => {
      stderrOutput.push(String(msg));
      return true;
    });

    const module = await import("./pi_steering_extension.cjs?" + Date.now());
    piSteeringExtension = module.default;
    loadSteeringConfig = module.loadSteeringConfig;
  });

  afterEach(() => {
    process.env = originalEnv;
    vi.restoreAllMocks();
  });

  // ---------------------------------------------------------------------------
  // loadSteeringConfig
  // ---------------------------------------------------------------------------
  describe("loadSteeringConfig", () => {
    it("should return default values when no env vars are set", () => {
      delete process.env.GH_AW_TIMEOUT_MINUTES;
      delete process.env.GH_AW_STEERING_TIME_WARNING_MINUTES;
      delete process.env.GH_AW_STEERING_TIME_CRITICAL_MINUTES;

      const config = loadSteeringConfig();

      expect(config.timeoutMinutes).toBe(30);
      expect(config.timeWarningMinutes).toBe(5);
      expect(config.timeCriticalMinutes).toBe(2);
    });

    it("should read timeout from GH_AW_TIMEOUT_MINUTES", () => {
      process.env.GH_AW_TIMEOUT_MINUTES = "45";

      const config = loadSteeringConfig();

      expect(config.timeoutMinutes).toBe(45);
    });

    it("should read warning threshold from GH_AW_STEERING_TIME_WARNING_MINUTES", () => {
      process.env.GH_AW_STEERING_TIME_WARNING_MINUTES = "10";

      const config = loadSteeringConfig();

      expect(config.timeWarningMinutes).toBe(10);
    });

    it("should read critical threshold from GH_AW_STEERING_TIME_CRITICAL_MINUTES", () => {
      process.env.GH_AW_STEERING_TIME_CRITICAL_MINUTES = "3";

      const config = loadSteeringConfig();

      expect(config.timeCriticalMinutes).toBe(3);
    });

    it("should fall back to defaults for non-numeric env var values", () => {
      process.env.GH_AW_TIMEOUT_MINUTES = "not-a-number";
      process.env.GH_AW_STEERING_TIME_WARNING_MINUTES = "";
      process.env.GH_AW_STEERING_TIME_CRITICAL_MINUTES = "abc";

      const config = loadSteeringConfig();

      expect(config.timeoutMinutes).toBe(30);
      expect(config.timeWarningMinutes).toBe(5);
      expect(config.timeCriticalMinutes).toBe(2);
    });

    it("should support fractional minute values", () => {
      process.env.GH_AW_STEERING_TIME_WARNING_MINUTES = "2.5";

      const config = loadSteeringConfig();

      expect(config.timeWarningMinutes).toBe(2.5);
    });
  });

  // ---------------------------------------------------------------------------
  // piSteeringExtension — event handler registration
  // ---------------------------------------------------------------------------
  describe("piSteeringExtension registration", () => {
    it("should register agent_start and turn_end handlers", () => {
      const handlers = {};
      const mockPi = {
        on: vi.fn((event, handler) => {
          handlers[event] = handler;
        }),
      };

      piSteeringExtension(mockPi);

      expect(mockPi.on).toHaveBeenCalledWith("agent_start", expect.any(Function));
      expect(mockPi.on).toHaveBeenCalledWith("turn_end", expect.any(Function));
    });
  });

  // ---------------------------------------------------------------------------
  // piSteeringExtension — agent_start handler
  // ---------------------------------------------------------------------------
  describe("agent_start handler", () => {
    it("should log session start to stderr", async () => {
      const handlers = {};
      const mockPi = {
        on: vi.fn((event, handler) => {
          handlers[event] = handler;
        }),
      };

      piSteeringExtension(mockPi);
      await handlers["agent_start"]();

      expect(stderrOutput.some(line => line.includes("[gh-aw/steering] Session started"))).toBe(true);
    });
  });

  // ---------------------------------------------------------------------------
  // piSteeringExtension — turn_end handler: no steer before any threshold
  // ---------------------------------------------------------------------------
  describe("turn_end handler", () => {
    it("should not steer when plenty of time remains", async () => {
      process.env.GH_AW_TIMEOUT_MINUTES = "30";
      process.env.GH_AW_STEERING_TIME_WARNING_MINUTES = "5";
      process.env.GH_AW_STEERING_TIME_CRITICAL_MINUTES = "2";

      const handlers = {};
      const steerCalls = [];
      const mockPi = { on: vi.fn((e, h) => (handlers[e] = h)) };
      const mockCtx = { agent: { steer: vi.fn(msg => steerCalls.push(msg)) } };

      piSteeringExtension(mockPi);

      // Start the session
      await handlers["agent_start"]();

      // Fake only 1 minute elapsed — 29 minutes remaining, well above thresholds
      vi.setSystemTime(Date.now() + 60 * 1000);
      await handlers["turn_end"]({}, mockCtx);

      expect(steerCalls).toHaveLength(0);
    });

    it("should inject a warning message when time remaining falls below warning threshold", async () => {
      process.env.GH_AW_TIMEOUT_MINUTES = "10";
      process.env.GH_AW_STEERING_TIME_WARNING_MINUTES = "5";
      process.env.GH_AW_STEERING_TIME_CRITICAL_MINUTES = "2";

      const handlers = {};
      const steerCalls = [];
      const mockPi = { on: vi.fn((e, h) => (handlers[e] = h)) };
      const mockCtx = { agent: { steer: vi.fn(msg => steerCalls.push(msg)) } };

      piSteeringExtension(mockPi);
      await handlers["agent_start"]();

      // 6 minutes elapsed → 4 minutes remaining (below 5-min warning threshold)
      vi.setSystemTime(Date.now() + 6 * 60 * 1000);
      await handlers["turn_end"]({}, mockCtx);

      expect(steerCalls).toHaveLength(1);
      expect(steerCalls[0].content).toContain("⚠️");
      expect(steerCalls[0].content).toContain("minute");
      expect(steerCalls[0].role).toBe("user");
      expect(typeof steerCalls[0].timestamp).toBe("number");
    });

    it("should inject a critical message when time remaining falls below critical threshold", async () => {
      process.env.GH_AW_TIMEOUT_MINUTES = "10";
      process.env.GH_AW_STEERING_TIME_WARNING_MINUTES = "5";
      process.env.GH_AW_STEERING_TIME_CRITICAL_MINUTES = "2";

      const handlers = {};
      const steerCalls = [];
      const mockPi = { on: vi.fn((e, h) => (handlers[e] = h)) };
      const mockCtx = { agent: { steer: vi.fn(msg => steerCalls.push(msg)) } };

      piSteeringExtension(mockPi);
      await handlers["agent_start"]();

      // 9 minutes elapsed → 1 minute remaining (below 2-min critical threshold)
      vi.setSystemTime(Date.now() + 9 * 60 * 1000);
      await handlers["turn_end"]({}, mockCtx);

      expect(steerCalls).toHaveLength(1);
      expect(steerCalls[0].content).toContain("CRITICAL");
      expect(steerCalls[0].role).toBe("user");
    });

    it("should only inject the warning message once even across multiple turns", async () => {
      process.env.GH_AW_TIMEOUT_MINUTES = "10";
      process.env.GH_AW_STEERING_TIME_WARNING_MINUTES = "5";
      process.env.GH_AW_STEERING_TIME_CRITICAL_MINUTES = "2";

      const handlers = {};
      const steerCalls = [];
      const mockPi = { on: vi.fn((e, h) => (handlers[e] = h)) };
      const mockCtx = { agent: { steer: vi.fn(msg => steerCalls.push(msg)) } };

      piSteeringExtension(mockPi);
      await handlers["agent_start"]();

      // First turn at 4min remaining — triggers warning
      vi.setSystemTime(Date.now() + 6 * 60 * 1000);
      await handlers["turn_end"]({}, mockCtx);

      // Second turn at 3.5min remaining — should NOT fire again
      vi.setSystemTime(Date.now() + 30 * 1000);
      await handlers["turn_end"]({}, mockCtx);

      expect(steerCalls).toHaveLength(1);
    });

    it("should only inject the critical message once even across multiple turns", async () => {
      process.env.GH_AW_TIMEOUT_MINUTES = "10";
      process.env.GH_AW_STEERING_TIME_WARNING_MINUTES = "5";
      process.env.GH_AW_STEERING_TIME_CRITICAL_MINUTES = "2";

      const handlers = {};
      const steerCalls = [];
      const mockPi = { on: vi.fn((e, h) => (handlers[e] = h)) };
      const mockCtx = { agent: { steer: vi.fn(msg => steerCalls.push(msg)) } };

      piSteeringExtension(mockPi);
      await handlers["agent_start"]();

      // First turn — critical threshold crossed
      vi.setSystemTime(Date.now() + 9 * 60 * 1000);
      await handlers["turn_end"]({}, mockCtx);

      // Second turn — still critical, should NOT fire again
      vi.setSystemTime(Date.now() + 30 * 1000);
      await handlers["turn_end"]({}, mockCtx);

      expect(steerCalls).toHaveLength(1);
    });

    it("should not steer when agent_start has not fired yet", async () => {
      process.env.GH_AW_TIMEOUT_MINUTES = "10";

      const handlers = {};
      const steerCalls = [];
      const mockPi = { on: vi.fn((e, h) => (handlers[e] = h)) };
      const mockCtx = { agent: { steer: vi.fn(msg => steerCalls.push(msg)) } };

      piSteeringExtension(mockPi);

      // Call turn_end WITHOUT calling agent_start first
      vi.setSystemTime(Date.now() + 20 * 60 * 1000);
      await handlers["turn_end"]({}, mockCtx);

      expect(steerCalls).toHaveLength(0);
    });

    it("should log to stderr when injecting warning message", async () => {
      process.env.GH_AW_TIMEOUT_MINUTES = "10";
      process.env.GH_AW_STEERING_TIME_WARNING_MINUTES = "5";
      process.env.GH_AW_STEERING_TIME_CRITICAL_MINUTES = "2";

      const handlers = {};
      const mockPi = { on: vi.fn((e, h) => (handlers[e] = h)) };
      const mockCtx = { agent: { steer: vi.fn() } };

      piSteeringExtension(mockPi);
      await handlers["agent_start"]();

      vi.setSystemTime(Date.now() + 6 * 60 * 1000);
      await handlers["turn_end"]({}, mockCtx);

      expect(stderrOutput.some(line => line.includes("[gh-aw/steering] WARNING"))).toBe(true);
    });

    it("should log to stderr when injecting critical message", async () => {
      process.env.GH_AW_TIMEOUT_MINUTES = "10";
      process.env.GH_AW_STEERING_TIME_WARNING_MINUTES = "5";
      process.env.GH_AW_STEERING_TIME_CRITICAL_MINUTES = "2";

      const handlers = {};
      const mockPi = { on: vi.fn((e, h) => (handlers[e] = h)) };
      const mockCtx = { agent: { steer: vi.fn() } };

      piSteeringExtension(mockPi);
      await handlers["agent_start"]();

      vi.setSystemTime(Date.now() + 9 * 60 * 1000);
      await handlers["turn_end"]({}, mockCtx);

      expect(stderrOutput.some(line => line.includes("[gh-aw/steering] CRITICAL"))).toBe(true);
    });

    it("should inject warning before critical when time drops through both thresholds", async () => {
      process.env.GH_AW_TIMEOUT_MINUTES = "10";
      process.env.GH_AW_STEERING_TIME_WARNING_MINUTES = "5";
      process.env.GH_AW_STEERING_TIME_CRITICAL_MINUTES = "2";

      const handlers = {};
      const steerCalls = [];
      const mockPi = { on: vi.fn((e, h) => (handlers[e] = h)) };
      const mockCtx = { agent: { steer: vi.fn(msg => steerCalls.push(msg)) } };

      piSteeringExtension(mockPi);
      await handlers["agent_start"]();

      // Turn 1: 4 min remaining → warning
      vi.setSystemTime(Date.now() + 6 * 60 * 1000);
      await handlers["turn_end"]({}, mockCtx);

      // Turn 2: 1 min remaining → critical
      vi.setSystemTime(Date.now() + 3 * 60 * 1000);
      await handlers["turn_end"]({}, mockCtx);

      expect(steerCalls).toHaveLength(2);
      expect(steerCalls[0].content).not.toContain("CRITICAL");
      expect(steerCalls[1].content).toContain("CRITICAL");
    });
  });
});
