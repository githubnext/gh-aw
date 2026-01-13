import { describe, it, expect, beforeEach, vi } from "vitest";

describe("missing_tool.cjs handler", () => {
  let mockCore, handler;

  beforeEach(async () => {
    // Mock core
    mockCore = {
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
    };
    global.core = mockCore;

    // Mock require for error_helpers
    global.require = vi.fn().mockImplementation(module => {
      if ("./error_helpers.cjs" === module) {
        return { getErrorMessage: error => (error instanceof Error ? error.message : String(error)) };
      }
      throw new Error(`Module not found: ${module}`);
    });

    // Load the handler module
    const { main } = await import("./missing_tool.cjs");

    // Create handler with default config
    handler = await main({});
  });

  afterEach(() => {
    delete global.core;
    delete global.require;
    vi.clearAllMocks();
  });

  describe("Message Processing", () => {
    it("should process valid missing_tool message", async () => {
      const message = {
        type: "missing_tool",
        tool: "docker",
        reason: "Need containerization support",
        alternatives: "Use VM or manual setup",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(result.tool).toBe("docker");
      expect(result.reason).toBe("Need containerization support");
      expect(result.alternatives).toBe("Use VM or manual setup");
      expect(result.timestamp).toBeDefined();
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("docker"));
    });

    it("should process message without alternatives", async () => {
      const message = {
        type: "missing_tool",
        tool: "kubectl",
        reason: "Kubernetes cluster management required",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(result.tool).toBe("kubectl");
      expect(result.reason).toBe("Kubernetes cluster management required");
      expect(result.alternatives).toBeNull();
    });

    it("should process message without tool field (general limitation)", async () => {
      const message = {
        type: "missing_tool",
        reason: "Cannot access external APIs due to network restrictions",
        alternatives: "User can manually fetch data and provide it",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(result.tool).toBeNull();
      expect(result.reason).toBe("Cannot access external APIs due to network restrictions");
      expect(result.alternatives).toBe("User can manually fetch data and provide it");
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("missing functionality/limitation"));
    });

    it("should reject message missing reason field", async () => {
      const message = {
        type: "missing_tool",
        tool: "some-tool",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(false);
      expect(result.error).toBe("Missing required field: reason");
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("missing 'reason' field"));
    });
  });

  describe("Max Count Limit", () => {
    it("should respect max count limit", async () => {
      // Create handler with max count of 2
      const limitedHandler = await (await import("./missing_tool.cjs")).main({ max: 2 });

      const message1 = { tool: "tool1", reason: "reason1" };
      const message2 = { tool: "tool2", reason: "reason2" };
      const message3 = { tool: "tool3", reason: "reason3" };

      const result1 = await limitedHandler(message1, {});
      const result2 = await limitedHandler(message2, {});
      const result3 = await limitedHandler(message3, {});

      expect(result1.success).toBe(true);
      expect(result2.success).toBe(true);
      expect(result3.success).toBe(false);
      expect(result3.error).toContain("Max count of 2 reached");
    });

    it("should allow unlimited messages when max is 0", async () => {
      // Create handler with max count of 0 (unlimited)
      const unlimitedHandler = await (await import("./missing_tool.cjs")).main({ max: 0 });

      // Process multiple messages
      for (let i = 0; i < 5; i++) {
        const result = await unlimitedHandler({ tool: `tool${i}`, reason: `reason${i}` }, {});
        expect(result.success).toBe(true);
      }
    });
  });

  describe("Timestamp", () => {
    it("should add timestamp to results", async () => {
      const message = { tool: "test-tool", reason: "testing" };
      const beforeTime = new Date();

      const result = await handler(message, {});

      const afterTime = new Date();
      const timestamp = new Date(result.timestamp);

      expect(timestamp).toBeInstanceOf(Date);
      expect(timestamp.getTime()).toBeGreaterThanOrEqual(beforeTime.getTime());
      expect(timestamp.getTime()).toBeLessThanOrEqual(afterTime.getTime());
    });
  });
});
