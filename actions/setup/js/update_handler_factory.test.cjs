import { describe, it, expect, beforeEach, vi } from "vitest";

// Import the factory function
let factoryModule;

// Mock the global objects that GitHub Actions provides
const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  notice: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
};

const mockGithub = {
  rest: {
    issues: {
      update: vi.fn(),
    },
  },
};

const mockContext = {
  eventName: "issues",
  repo: {
    owner: "testowner",
    repo: "testrepo",
  },
  serverUrl: "https://github.com",
  runId: 12345,
  payload: {
    issue: {
      number: 42,
    },
  },
};

// Set up global mocks
global.core = mockCore;
global.github = mockGithub;
global.context = mockContext;

describe("update_handler_factory.cjs", () => {
  beforeEach(async () => {
    // Reset all mocks before each test
    vi.clearAllMocks();
    vi.resetModules();

    // Import the module fresh for each test
    factoryModule = await import("./update_handler_factory.cjs");
  });

  describe("createUpdateHandlerFactory", () => {
    it("should create a handler factory with default configuration", async () => {
      const mockResolveItemNumber = vi.fn().mockReturnValue({ success: true, number: 42 });
      const mockBuildUpdateData = vi.fn().mockReturnValue({ success: true, data: { title: "Test" } });
      const mockExecuteUpdate = vi.fn().mockResolvedValue({ html_url: "https://example.com", title: "Test" });
      const mockFormatSuccessResult = vi.fn().mockReturnValue({ success: true, number: 42 });

      const handlerFactory = factoryModule.createUpdateHandlerFactory({
        itemType: "update_test",
        itemTypeName: "test item",
        supportsPR: false,
        resolveItemNumber: mockResolveItemNumber,
        buildUpdateData: mockBuildUpdateData,
        executeUpdate: mockExecuteUpdate,
        formatSuccessResult: mockFormatSuccessResult,
      });

      // Create handler with default config
      const handler = await handlerFactory({});

      // Execute handler
      const result = await handler({ title: "Test" });

      // Verify default configuration was logged
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("max=10"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("target=triggering"));

      // Verify handler was successful
      expect(result.success).toBe(true);
      expect(mockResolveItemNumber).toHaveBeenCalled();
      expect(mockBuildUpdateData).toHaveBeenCalled();
      expect(mockExecuteUpdate).toHaveBeenCalled();
      expect(mockFormatSuccessResult).toHaveBeenCalled();
    });

    it("should respect custom max count configuration", async () => {
      const mockResolveItemNumber = vi.fn().mockReturnValue({ success: true, number: 42 });
      const mockBuildUpdateData = vi.fn().mockReturnValue({ success: true, data: { title: "Test" } });
      const mockExecuteUpdate = vi.fn().mockResolvedValue({ html_url: "https://example.com" });
      const mockFormatSuccessResult = vi.fn().mockReturnValue({ success: true });

      const handlerFactory = factoryModule.createUpdateHandlerFactory({
        itemType: "update_test",
        itemTypeName: "test item",
        supportsPR: false,
        resolveItemNumber: mockResolveItemNumber,
        buildUpdateData: mockBuildUpdateData,
        executeUpdate: mockExecuteUpdate,
        formatSuccessResult: mockFormatSuccessResult,
      });

      // Create handler with max=2
      const handler = await handlerFactory({ max: 2 });

      // Process 2 messages (should succeed)
      const result1 = await handler({ title: "Test 1" });
      expect(result1.success).toBe(true);

      const result2 = await handler({ title: "Test 2" });
      expect(result2.success).toBe(true);

      // Third message should be rejected due to max count
      const result3 = await handler({ title: "Test 3" });
      expect(result3.success).toBe(false);
      expect(result3.error).toContain("Max count of 2 reached");
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("max count of 2 reached"));
    });

    it("should handle resolution errors gracefully", async () => {
      const mockResolveItemNumber = vi.fn().mockReturnValue({
        success: false,
        error: "Resolution failed",
      });
      const mockBuildUpdateData = vi.fn();
      const mockExecuteUpdate = vi.fn();
      const mockFormatSuccessResult = vi.fn();

      const handlerFactory = factoryModule.createUpdateHandlerFactory({
        itemType: "update_test",
        itemTypeName: "test item",
        supportsPR: false,
        resolveItemNumber: mockResolveItemNumber,
        buildUpdateData: mockBuildUpdateData,
        executeUpdate: mockExecuteUpdate,
        formatSuccessResult: mockFormatSuccessResult,
      });

      const handler = await handlerFactory({});
      const result = await handler({ title: "Test" });

      expect(result.success).toBe(false);
      expect(result.error).toBe("Resolution failed");
      expect(mockCore.warning).toHaveBeenCalledWith("Resolution failed");
      // Should not proceed to build/execute
      expect(mockBuildUpdateData).not.toHaveBeenCalled();
      expect(mockExecuteUpdate).not.toHaveBeenCalled();
    });

    it("should handle build errors gracefully", async () => {
      const mockResolveItemNumber = vi.fn().mockReturnValue({ success: true, number: 42 });
      const mockBuildUpdateData = vi.fn().mockReturnValue({
        success: false,
        error: "No fields to update",
      });
      const mockExecuteUpdate = vi.fn();
      const mockFormatSuccessResult = vi.fn();

      const handlerFactory = factoryModule.createUpdateHandlerFactory({
        itemType: "update_test",
        itemTypeName: "test item",
        supportsPR: false,
        resolveItemNumber: mockResolveItemNumber,
        buildUpdateData: mockBuildUpdateData,
        executeUpdate: mockExecuteUpdate,
        formatSuccessResult: mockFormatSuccessResult,
      });

      const handler = await handlerFactory({});
      const result = await handler({ title: "Test" });

      expect(result.success).toBe(false);
      expect(result.error).toBe("No fields to update");
      expect(mockCore.warning).toHaveBeenCalledWith("No fields to update");
      // Should not proceed to execute
      expect(mockExecuteUpdate).not.toHaveBeenCalled();
    });

    it("should handle empty update data as no-op", async () => {
      const mockResolveItemNumber = vi.fn().mockReturnValue({ success: true, number: 42 });
      const mockBuildUpdateData = vi.fn().mockReturnValue({ success: true, data: {} });
      const mockExecuteUpdate = vi.fn();
      const mockFormatSuccessResult = vi.fn();

      const handlerFactory = factoryModule.createUpdateHandlerFactory({
        itemType: "update_test",
        itemTypeName: "test item",
        supportsPR: false,
        resolveItemNumber: mockResolveItemNumber,
        buildUpdateData: mockBuildUpdateData,
        executeUpdate: mockExecuteUpdate,
        formatSuccessResult: mockFormatSuccessResult,
      });

      const handler = await handlerFactory({});
      const result = await handler({ title: "Test" });

      expect(result.success).toBe(true);
      expect(result.skipped).toBe(true);
      expect(result.reason).toBe("No update fields provided");
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No update fields provided"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("treating as no-op"));
      // Should not proceed to execute
      expect(mockExecuteUpdate).not.toHaveBeenCalled();
    });

    it("should ignore internal fields starting with underscore", async () => {
      const mockResolveItemNumber = vi.fn().mockReturnValue({ success: true, number: 42 });
      const mockBuildUpdateData = vi.fn().mockReturnValue({
        success: true,
        data: { _internal: "value", title: "Test" },
      });
      const mockExecuteUpdate = vi.fn().mockResolvedValue({ html_url: "https://example.com" });
      const mockFormatSuccessResult = vi.fn().mockReturnValue({ success: true });

      const handlerFactory = factoryModule.createUpdateHandlerFactory({
        itemType: "update_test",
        itemTypeName: "test item",
        supportsPR: false,
        resolveItemNumber: mockResolveItemNumber,
        buildUpdateData: mockBuildUpdateData,
        executeUpdate: mockExecuteUpdate,
        formatSuccessResult: mockFormatSuccessResult,
      });

      const handler = await handlerFactory({});
      const result = await handler({ title: "Test" });

      expect(result.success).toBe(true);
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining('["title"]'));
    });

    it("should NOT skip when _rawBody is present (body updates)", async () => {
      const mockResolveItemNumber = vi.fn().mockReturnValue({ success: true, number: 42 });
      const mockBuildUpdateData = vi.fn().mockReturnValue({
        success: true,
        data: { _rawBody: "New body content", _operation: "append" },
      });
      const mockExecuteUpdate = vi.fn().mockResolvedValue({ html_url: "https://example.com" });
      const mockFormatSuccessResult = vi.fn().mockReturnValue({ success: true });

      const handlerFactory = factoryModule.createUpdateHandlerFactory({
        itemType: "update_issue",
        itemTypeName: "issue",
        supportsPR: false,
        resolveItemNumber: mockResolveItemNumber,
        buildUpdateData: mockBuildUpdateData,
        executeUpdate: mockExecuteUpdate,
        formatSuccessResult: mockFormatSuccessResult,
      });

      const handler = await handlerFactory({});
      const result = await handler({ body: "New body content" });

      // Should NOT skip - _rawBody indicates a body update
      expect(result.success).toBe(true);
      expect(result.skipped).toBeUndefined();
      // Should proceed to execute the update
      expect(mockExecuteUpdate).toHaveBeenCalled();
    });

    it("should handle execution errors gracefully", async () => {
      const mockResolveItemNumber = vi.fn().mockReturnValue({ success: true, number: 42 });
      const mockBuildUpdateData = vi.fn().mockReturnValue({ success: true, data: { title: "Test" } });
      const mockExecuteUpdate = vi.fn().mockRejectedValue(new Error("API Error"));
      const mockFormatSuccessResult = vi.fn();

      const handlerFactory = factoryModule.createUpdateHandlerFactory({
        itemType: "update_test",
        itemTypeName: "test item",
        supportsPR: false,
        resolveItemNumber: mockResolveItemNumber,
        buildUpdateData: mockBuildUpdateData,
        executeUpdate: mockExecuteUpdate,
        formatSuccessResult: mockFormatSuccessResult,
      });

      const handler = await handlerFactory({});
      const result = await handler({ title: "Test" });

      expect(result.success).toBe(false);
      expect(result.error).toBe("API Error");
      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Failed to update test item"));
    });

    it("should pass additional config to log message", async () => {
      const mockResolveItemNumber = vi.fn().mockReturnValue({ success: true, number: 42 });
      const mockBuildUpdateData = vi.fn().mockReturnValue({ success: true, data: { title: "Test" } });
      const mockExecuteUpdate = vi.fn().mockResolvedValue({ html_url: "https://example.com" });
      const mockFormatSuccessResult = vi.fn().mockReturnValue({ success: true });

      const handlerFactory = factoryModule.createUpdateHandlerFactory({
        itemType: "update_test",
        itemTypeName: "test item",
        supportsPR: false,
        resolveItemNumber: mockResolveItemNumber,
        buildUpdateData: mockBuildUpdateData,
        executeUpdate: mockExecuteUpdate,
        formatSuccessResult: mockFormatSuccessResult,
        additionalConfig: {
          allow_title: true,
          allow_body: true,
        },
      });

      // Create handler with additional config
      const handler = await handlerFactory({ allow_title: false, allow_body: true });

      // Verify additional config items in log
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("allow_title=false"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("allow_body=true"));
    });

    it("should track processed count across multiple calls", async () => {
      const mockResolveItemNumber = vi.fn().mockReturnValue({ success: true, number: 42 });
      const mockBuildUpdateData = vi.fn().mockReturnValue({ success: true, data: { title: "Test" } });
      const mockExecuteUpdate = vi.fn().mockResolvedValue({ html_url: "https://example.com" });
      const mockFormatSuccessResult = vi.fn().mockReturnValue({ success: true });

      const handlerFactory = factoryModule.createUpdateHandlerFactory({
        itemType: "update_test",
        itemTypeName: "test item",
        supportsPR: false,
        resolveItemNumber: mockResolveItemNumber,
        buildUpdateData: mockBuildUpdateData,
        executeUpdate: mockExecuteUpdate,
        formatSuccessResult: mockFormatSuccessResult,
      });

      const handler = await handlerFactory({ max: 3 });

      // First call should succeed
      const result1 = await handler({ title: "Test 1" });
      expect(result1.success).toBe(true);

      // Second call should succeed
      const result2 = await handler({ title: "Test 2" });
      expect(result2.success).toBe(true);

      // Third call should succeed
      const result3 = await handler({ title: "Test 3" });
      expect(result3.success).toBe(true);

      // Fourth call should fail due to max count
      const result4 = await handler({ title: "Test 4" });
      expect(result4.success).toBe(false);
      expect(result4.error).toContain("Max count of 3 reached");
    });
  });
});
