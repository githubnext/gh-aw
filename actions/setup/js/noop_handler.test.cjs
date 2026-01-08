import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

describe("noop_handler.cjs handler", () => {
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
    const { main } = await import("./noop_handler.cjs");

    // Create handler with default config
    handler = await main({});
  });

  afterEach(() => {
    delete global.core;
    delete global.require;
    vi.clearAllMocks();
  });

  describe("Message Processing", () => {
    it("should process valid noop message", async () => {
      const message = {
        type: "noop",
        message: "No issues found in this review",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(result.message).toBe("No issues found in this review");
      expect(result.timestamp).toBeDefined();
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No issues found in this review"));
    });

    it("should process simple message", async () => {
      const message = {
        type: "noop",
        message: "Analysis complete, no action needed",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(result.message).toBe("Analysis complete, no action needed");
      expect(result.timestamp).toBeDefined();
    });

    it("should reject message missing message field", async () => {
      const message = {
        type: "noop",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(false);
      expect(result.error).toBe("Missing required field: message");
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("missing or invalid 'message' field"));
    });

    it("should reject message with empty string", async () => {
      const message = {
        type: "noop",
        message: "",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(false);
      expect(result.error).toBe("Missing required field: message");
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("missing or invalid 'message' field"));
    });

    it("should reject message with only whitespace", async () => {
      const message = {
        type: "noop",
        message: "   ",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(false);
      expect(result.error).toBe("Missing required field: message");
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("missing or invalid 'message' field"));
    });

    it("should reject message with non-string message field", async () => {
      const message = {
        type: "noop",
        message: 123,
      };

      const result = await handler(message, {});

      expect(result.success).toBe(false);
      expect(result.error).toBe("Missing required field: message");
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("missing or invalid 'message' field"));
    });

    it("should handle messages with special characters", async () => {
      const message = {
        type: "noop",
        message: "Analysis complete: <tag> & \"quotes\" 'apostrophes'",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(result.message).toBe("Analysis complete: <tag> & \"quotes\" 'apostrophes'");
    });

    it("should handle very long messages", async () => {
      const longMessage = "A".repeat(1000);
      const message = {
        type: "noop",
        message: longMessage,
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(result.message).toBe(longMessage);
      expect(result.message.length).toBe(1000);
    });
  });

  describe("Max Count Limit", () => {
    it("should respect max count limit", async () => {
      // Create handler with max count of 2
      const limitedHandler = await (await import("./noop_handler.cjs")).main({ max: 2 });

      const message1 = { message: "First message" };
      const message2 = { message: "Second message" };
      const message3 = { message: "Third message" };

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
      const unlimitedHandler = await (await import("./noop_handler.cjs")).main({ max: 0 });

      // Process multiple messages
      for (let i = 0; i < 5; i++) {
        const result = await unlimitedHandler({ message: `Message ${i}` }, {});
        expect(result.success).toBe(true);
      }
    });
  });

  describe("Timestamp", () => {
    it("should add timestamp to results", async () => {
      const message = { message: "Test message" };
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
