import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

describe("missing_data.cjs handler", () => {
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
    const { main } = await import("./missing_data.cjs");

    // Create handler with default config
    handler = await main({});
  });

  afterEach(() => {
    delete global.core;
    delete global.require;
    vi.clearAllMocks();
  });

  describe("Message Processing", () => {
    it("should process valid missing_data message", async () => {
      const message = {
        type: "missing_data",
        data_type: "user_preferences",
        reason: "User preferences not found in database",
        context: "Needed to customize dashboard layout",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(result.data_type).toBe("user_preferences");
      expect(result.reason).toBe("User preferences not found in database");
      expect(result.context).toBe("Needed to customize dashboard layout");
      expect(result.timestamp).toBeDefined();
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("user_preferences"));
    });

    it("should process message with alternatives", async () => {
      const message = {
        type: "missing_data",
        data_type: "api_credentials",
        reason: "API credentials not configured",
        alternatives: "Could use default read-only access",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(result.alternatives).toBe("Could use default read-only access");
    });

    it("should process message without context", async () => {
      const message = {
        type: "missing_data",
        data_type: "simple_data",
        reason: "Simple reason",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(result.context).toBeNull();
      expect(result.alternatives).toBeNull();
    });

    it("should process message with only data_type", async () => {
      const message = {
        type: "missing_data",
        data_type: "some_data",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(result.data_type).toBe("some_data");
      expect(result.reason).toBeNull();
      expect(result.context).toBeNull();
      expect(result.alternatives).toBeNull();
    });

    it("should process message with only reason", async () => {
      const message = {
        type: "missing_data",
        reason: "No data type specified",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(result.data_type).toBeNull();
      expect(result.reason).toBe("No data type specified");
    });

    it("should process message with no fields (empty message)", async () => {
      const message = {
        type: "missing_data",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(result.data_type).toBeNull();
      expect(result.reason).toBeNull();
      expect(result.context).toBeNull();
      expect(result.alternatives).toBeNull();
      expect(result.timestamp).toBeDefined();
    });
  });

  describe("Max Count Limit", () => {
    it("should respect max count limit", async () => {
      // Create handler with max count of 2
      const limitedHandler = await (await import("./missing_data.cjs")).main({ max: 2 });

      // First message should succeed
      const result1 = await limitedHandler({ type: "missing_data", data_type: "data1" }, {});
      expect(result1.success).toBe(true);

      // Second message should succeed
      const result2 = await limitedHandler({ type: "missing_data", reason: "reason2" }, {});
      expect(result2.success).toBe(true);

      // Third message should fail
      const result3 = await limitedHandler({ type: "missing_data" }, {});
      expect(result3.success).toBe(false);
      expect(result3.error).toContain("Max count");
    });
  });
});
