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
      addLabels: vi.fn(),
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

describe("safe_output_handler_manager.cjs", () => {
  let tempFilePath;
  let tempDir;
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
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "handler-manager-test-"));

    // Dynamically import the module (fresh for each test)
    handlerManagerModule = await import("./safe_output_handler_manager.cjs");
  });

  afterEach(() => {
    if (tempDir && fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
  });

  describe("loadHandlerConfig", () => {
    it("should return failure when no config is provided", () => {
      delete process.env.GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG;
      const result = handlerManagerModule.loadHandlerConfig();
      
      expect(result.success).toBe(false);
      expect(mockCore.info).toHaveBeenCalledWith("No GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG environment variable found");
    });

    it("should parse valid JSON config", () => {
      const config = {
        create_issue: { max: 5, expires: 7 },
        add_comment: { max: 1 },
      };
      process.env.GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG = JSON.stringify(config);
      
      const result = handlerManagerModule.loadHandlerConfig();
      
      expect(result.success).toBe(true);
      expect(result.config).toEqual(config);
    });

    it("should handle invalid JSON gracefully", () => {
      process.env.GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG = "invalid json{";
      
      const result = handlerManagerModule.loadHandlerConfig();
      
      expect(result.success).toBe(false);
      expect(result.error).toContain("Error parsing handler configuration JSON");
    });
  });

  describe("main", () => {
    it("should skip when no agent output is provided", async () => {
      delete process.env.GH_AW_AGENT_OUTPUT;
      
      await handlerManagerModule.main();
      
      expect(mockCore.info).toHaveBeenCalledWith("Starting safe output handler manager");
      expect(mockCore.info).toHaveBeenCalledWith("No agent output to process");
    });

    it("should skip when agent output is empty", async () => {
      setAgentOutput({ items: [] });
      
      await handlerManagerModule.main();
      
      expect(mockCore.info).toHaveBeenCalledWith("No items found in agent output");
    });

    it("should skip when no handler config is provided", async () => {
      setAgentOutput({ items: [{ type: "create_issue", title: "Test" }] });
      delete process.env.GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG;
      
      await handlerManagerModule.main();
      
      expect(mockCore.info).toHaveBeenCalledWith("No handler configuration found");
    });

    it("should identify handler types from items", async () => {
      setAgentOutput({
        items: [
          { type: "create_issue", title: "Test 1" },
          { type: "add_comment", body: "Test comment" },
          { type: "create_issue", title: "Test 2" },
        ],
      });
      process.env.GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG = JSON.stringify({
        create_issue: {},
        add_comment: {},
      });
      
      await handlerManagerModule.main();
      
      // Should identify unique handler types
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringMatching(/Handler types needed:.*create_issue.*add_comment/));
    });

    it("should set staged mode when environment variable is true", async () => {
      setAgentOutput({ items: [{ type: "create_issue", title: "Test" }] });
      process.env.GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG = JSON.stringify({ create_issue: {} });
      process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";
      
      await handlerManagerModule.main();
      
      expect(mockCore.info).toHaveBeenCalledWith("Running in staged mode");
    });

    it("should warn when handler cannot be loaded", async () => {
      setAgentOutput({ items: [{ type: "nonexistent_handler", data: "test" }] });
      process.env.GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG = JSON.stringify({ nonexistent_handler: {} });
      
      await handlerManagerModule.main();
      
      // Should error about failing to load handler
      expect(mockCore.error).toHaveBeenCalledWith(expect.stringMatching(/Failed to load handler nonexistent_handler/));
    });
  });
});
