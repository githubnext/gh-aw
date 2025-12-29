import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import fs from "fs";
import path from "path";
import os from "os";

const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  notice: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  exportVariable: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

const mockGithub = {
  rest: {
    issues: {
      create: vi.fn(),
    },
  },
};

const mockContext = {
  eventName: "issues",
  repo: { owner: "testowner", repo: "testrepo" },
  payload: { issue: { number: 123 } },
};

global.core = mockCore;
global.github = mockGithub;
global.context = mockContext;

describe("example_handler.cjs with handler_manager", () => {
  let tempDir;
  let tempFilePath;
  let exampleHandlerModule;
  let handlerManagerModule;

  const setAgentOutput = (data) => {
    tempFilePath = path.join(tempDir, `test_agent_output_${Date.now()}_${Math.random().toString(36).slice(2)}.json`);
    const content = typeof data === "string" ? data : JSON.stringify(data);
    fs.writeFileSync(tempFilePath, content);
    process.env.GH_AW_AGENT_OUTPUT = tempFilePath;
  };

  beforeEach(async () => {
    vi.clearAllMocks();
    delete process.env.GH_AW_AGENT_OUTPUT;
    delete process.env.GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG;
    delete process.env.GH_AW_SAFE_OUTPUTS_STAGED;

    // Create temp directory for test files
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "example-handler-test-"));

    // Dynamically import modules
    exampleHandlerModule = await import("./example_handler.cjs");
    handlerManagerModule = await import("./safe_output_handler_manager.cjs");
  });

  afterEach(() => {
    if (tempDir && fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
  });

  describe("Factory pattern", () => {
    it("should return a function when main() is called with config", async () => {
      const config = { allowed: ["bug", "feature"], maxCount: 5 };
      const messageProcessor = await exampleHandlerModule.main(config);
      
      expect(typeof messageProcessor).toBe("function");
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Initializing example_handler"));
    });

    it("should process a message with the returned function", async () => {
      const config = { allowed: ["bug"], maxCount: 3 };
      const messageProcessor = await exampleHandlerModule.main(config);
      
      const outputItem = {
        type: "example_handler",
        title: "Test Issue",
      };
      
      const resolvedTemporaryIds = new Map();
      const result = await messageProcessor(outputItem, resolvedTemporaryIds);
      
      expect(result).toBeTruthy();
      expect(result.title).toBe("Test Issue");
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Processing example message"));
    });

    it("should return temporary ID mapping when present", async () => {
      const config = {};
      const messageProcessor = await exampleHandlerModule.main(config);
      
      const outputItem = {
        type: "example_handler",
        title: "Test Issue",
        temporary_id: "aw_123456789abc",
      };
      
      const resolvedTemporaryIds = new Map();
      const result = await messageProcessor(outputItem, resolvedTemporaryIds);
      
      expect(result.temporaryId).toBe("aw_123456789abc");
      expect(result.repo).toBe("testowner/testrepo");
      expect(result.number).toBe(123);
    });

    it("should warn when message is invalid", async () => {
      const config = {};
      const messageProcessor = await exampleHandlerModule.main(config);
      
      const outputItem = {
        type: "example_handler",
        // Missing title
      };
      
      const resolvedTemporaryIds = new Map();
      const result = await messageProcessor(outputItem, resolvedTemporaryIds);
      
      expect(result).toBeNull();
      expect(mockCore.warning).toHaveBeenCalledWith("No title provided in example message");
    });
  });

  describe("Integration with handler manager", () => {
    it("should process example_handler messages through handler manager", async () => {
      setAgentOutput({
        items: [
          { type: "example_handler", title: "Issue 1" },
          { type: "example_handler", title: "Issue 2", temporary_id: "aw_abc123def456" },
        ],
      });
      
      process.env.GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG = JSON.stringify({
        example_handler: { allowed: ["bug"], maxCount: 5 },
      });
      
      await handlerManagerModule.main();
      
      // Verify handler was initialized
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Initializing example_handler"));
      
      // Verify messages were processed
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Processing item 1/2"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Processing item 2/2"));
      
      // Verify temporary ID was mapped
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Mapped temporary ID"));
      
      // Verify final count
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Processed 2 items successfully"));
    });

    it("should handle multiple handler types", async () => {
      setAgentOutput({
        items: [
          { type: "example_handler", title: "Issue 1" },
          { type: "another_handler", data: "test" }, // Will fail to load but shouldn't stop processing
          { type: "example_handler", title: "Issue 2" },
        ],
      });
      
      process.env.GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG = JSON.stringify({
        example_handler: { allowed: ["bug"] },
        another_handler: {},
      });
      
      await handlerManagerModule.main();
      
      // Should error when trying to load another_handler
      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Failed to load handler another_handler"));
      
      // Should warn when trying to process item with no handler
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("No handler found for type: another_handler"));
      
      // Should process the 2 example_handler messages successfully (0 errors in processing)
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Processed 2 items successfully, 0 errors"));
    });
  });
});
