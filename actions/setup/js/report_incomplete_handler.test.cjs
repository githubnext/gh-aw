import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

describe("report_incomplete_handler.cjs handler", () => {
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
    const { main } = await import("./report_incomplete_handler.cjs");

    // Create handler with default config
    handler = await main({});
  });

  afterEach(() => {
    delete global.core;
    delete global.require;
    vi.clearAllMocks();
  });

  describe("Message Processing", () => {
    it("should process valid report_incomplete message with reason", async () => {
      const message = {
        type: "report_incomplete",
        reason: "MCP server crashed on pull_request_read call",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(result.reason).toBe("MCP server crashed on pull_request_read call");
      expect(result.details).toBeNull();
      expect(result.timestamp).toBeDefined();
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("MCP server crashed on pull_request_read call"));
    });

    it("should process report_incomplete message with reason and details", async () => {
      const message = {
        type: "report_incomplete",
        reason: "GitHub authentication failed",
        details: "GH_TOKEN is not set, direct API returned 404 for private repo",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(result.reason).toBe("GitHub authentication failed");
      expect(result.details).toBe("GH_TOKEN is not set, direct API returned 404 for private repo");
      expect(result.timestamp).toBeDefined();
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("GH_TOKEN is not set"));
    });

    it("should reject message missing reason field", async () => {
      const message = {
        type: "report_incomplete",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(false);
      expect(result.error).toBe("Missing required field: reason");
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("missing or invalid 'reason' field"));
    });

    it("should reject message with empty reason string", async () => {
      const message = {
        type: "report_incomplete",
        reason: "",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(false);
      expect(result.error).toBe("Missing required field: reason");
    });

    it("should reject message with whitespace-only reason", async () => {
      const message = {
        type: "report_incomplete",
        reason: "   ",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(false);
      expect(result.error).toBe("Missing required field: reason");
    });

    it("should reject message with non-string reason field", async () => {
      const message = {
        type: "report_incomplete",
        reason: 42,
      };

      const result = await handler(message, {});

      expect(result.success).toBe(false);
      expect(result.error).toBe("Missing required field: reason");
    });

    it("should handle reason with special characters", async () => {
      const message = {
        type: "report_incomplete",
        reason: "Tool <gh> & \"api\" returned 'error': status 404",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(result.reason).toBe("Tool <gh> & \"api\" returned 'error': status 404");
    });

    it("should handle message without details field gracefully", async () => {
      const message = {
        type: "report_incomplete",
        reason: "Required tool unavailable",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(result.details).toBeNull();
      // Should not log details line when details is absent
      expect(mockCore.info).not.toHaveBeenCalledWith(expect.stringContaining("Details:"));
    });
  });

  describe("Max Count Limit", () => {
    it("should respect max count limit", async () => {
      // Create handler with max count of 2
      const limitedHandler = await (await import("./report_incomplete_handler.cjs")).main({ max: 2 });

      const msg = { reason: "Tool failed" };

      const result1 = await limitedHandler(msg, {});
      const result2 = await limitedHandler(msg, {});
      const result3 = await limitedHandler(msg, {});

      expect(result1.success).toBe(true);
      expect(result2.success).toBe(true);
      expect(result3.success).toBe(false);
      expect(result3.error).toContain("Max count of 2 reached");
    });

    it("should allow unlimited messages when max is 0", async () => {
      const unlimitedHandler = await (await import("./report_incomplete_handler.cjs")).main({ max: 0 });

      for (let i = 0; i < 5; i++) {
        const result = await unlimitedHandler({ reason: `Failure ${i}` }, {});
        expect(result.success).toBe(true);
      }
    });
  });

  describe("Timestamp", () => {
    it("should add timestamp to results", async () => {
      const message = { reason: "Infrastructure failure" };
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
