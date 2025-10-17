import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";

describe("missing_tool.cjs", () => {
  let mockCore;
  let missingToolScript;
  let originalConsole;
  let tempFilePath;

  // Helper function to set agent output via file
  const setAgentOutput = data => {
    tempFilePath = path.join("/tmp", `test_agent_output_${Date.now()}_${Math.random().toString(36).slice(2)}.json`);
    const content = typeof data === "string" ? data : JSON.stringify(data);
    fs.writeFileSync(tempFilePath, content);
    process.env.GITHUB_AW_AGENT_OUTPUT = tempFilePath;
  };

  beforeEach(() => {
    // Save original console before mocking
    originalConsole = global.console;

    // Mock console methods
    global.console = {
      log: vi.fn(),
      error: vi.fn(),
    };

    // Mock core actions methods
    mockCore = {
      // Core logging functions
      debug: vi.fn(),
      info: vi.fn(),
      notice: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),

      // Core workflow functions
      setFailed: vi.fn(),
      setOutput: vi.fn(),
      exportVariable: vi.fn(),
      setSecret: vi.fn(),

      // Input/state functions (less commonly used but included for completeness)
      getInput: vi.fn(),
      getBooleanInput: vi.fn(),
      getMultilineInput: vi.fn(),
      getState: vi.fn(),
      saveState: vi.fn(),

      // Group functions
      startGroup: vi.fn(),
      endGroup: vi.fn(),
      group: vi.fn(),

      // Other utility functions
      addPath: vi.fn(),
      setCommandEcho: vi.fn(),
      isDebug: vi.fn().mockReturnValue(false),
      getIDToken: vi.fn(),
      toPlatformPath: vi.fn(),
      toPosixPath: vi.fn(),
      toWin32Path: vi.fn(),

      // Summary object with chainable methods
      summary: {
        addRaw: vi.fn().mockReturnThis(),
        write: vi.fn().mockResolvedValue(),
      },
    };
    global.core = mockCore;

    // Mock require
    global.require = vi.fn().mockImplementation(module => {
      if (module === "fs") {
        return fs;
      }
      if (module === "@actions/core") {
        return mockCore;
      }
      throw new Error(`Module not found: ${module}`);
    });

    // Read the script file
    const scriptPath = path.join(__dirname, "missing_tool.cjs");
    missingToolScript = fs.readFileSync(scriptPath, "utf8");
  });

  afterEach(() => {
    // Clean up temporary file
    if (tempFilePath && require("fs").existsSync(tempFilePath)) {
      require("fs").unlinkSync(tempFilePath);
      tempFilePath = undefined;
    }

    // Clean up environment variables
    delete process.env.GITHUB_AW_AGENT_OUTPUT;
    delete process.env.GITHUB_AW_MISSING_TOOL_MAX;

    // Restore original console
    global.console = originalConsole;

    // Clean up globals
    delete global.core;
    delete global.require;
  });

  const runScript = async () => {
    const scriptFunction = new Function(missingToolScript);
    return scriptFunction();
  };

  describe("JSON Array Input Format", () => {
    it("should parse JSON array with missing-tool entries", async () => {
      const testData = {
        items: [
          {
            type: "missing_tool",
            tool: "docker",
            reason: "Need containerization support",
            alternatives: "Use VM or manual setup",
          },
          {
            type: "missing_tool",
            tool: "kubectl",
            reason: "Kubernetes cluster management required",
          },
        ],
        errors: [],
      };

      setAgentOutput(testData);

      await runScript();

      expect(mockCore.setOutput).toHaveBeenCalledWith("total_count", "2");
      const toolsReportedCall = mockCore.setOutput.mock.calls.find(call => call[0] === "tools_reported");
      expect(toolsReportedCall).toBeDefined();

      const reportedTools = JSON.parse(toolsReportedCall[1]);
      expect(reportedTools).toHaveLength(2);
      expect(reportedTools[0].tool).toBe("docker");
      expect(reportedTools[0].reason).toBe("Need containerization support");
      expect(reportedTools[0].alternatives).toBe("Use VM or manual setup");
      expect(reportedTools[1].tool).toBe("kubectl");
      expect(reportedTools[1].reason).toBe("Kubernetes cluster management required");
      expect(reportedTools[1].alternatives).toBe(null);
    });

    it("should filter out non-missing-tool entries", async () => {
      const testData = {
        items: [
          {
            type: "missing_tool",
            tool: "docker",
            reason: "Need containerization",
          },
          {
            type: "other-type",
            data: "should be ignored",
          },
          {
            type: "missing_tool",
            tool: "kubectl",
            reason: "Need k8s support",
          },
        ],
        errors: [],
      };

      setAgentOutput(testData);

      await runScript();

      expect(mockCore.setOutput).toHaveBeenCalledWith("total_count", "2");
      const toolsReportedCall = mockCore.setOutput.mock.calls.find(call => call[0] === "tools_reported");
      const reportedTools = JSON.parse(toolsReportedCall[1]);
      expect(reportedTools).toHaveLength(2);
      expect(reportedTools[0].tool).toBe("docker");
      expect(reportedTools[1].tool).toBe("kubectl");
    });
  });

  describe("Validation", () => {
    it("should skip entries missing tool field", async () => {
      const testData = {
        items: [
          {
            type: "missing_tool",
            reason: "No tool specified",
          },
          {
            type: "missing_tool",
            tool: "valid-tool",
            reason: "This should work",
          },
        ],
        errors: [],
      };

      setAgentOutput(testData);

      await runScript();

      expect(mockCore.setOutput).toHaveBeenCalledWith("total_count", "1");
      expect(mockCore.warning).toHaveBeenCalledWith(`missing-tool entry missing 'tool' field: ${JSON.stringify(testData.items[0])}`);
    });

    it("should skip entries missing reason field", async () => {
      const testData = {
        items: [
          {
            type: "missing_tool",
            tool: "some-tool",
          },
          {
            type: "missing_tool",
            tool: "valid-tool",
            reason: "This should work",
          },
        ],
        errors: [],
      };

      setAgentOutput(testData);

      await runScript();

      expect(mockCore.setOutput).toHaveBeenCalledWith("total_count", "1");
      expect(mockCore.warning).toHaveBeenCalledWith(`missing-tool entry missing 'reason' field: ${JSON.stringify(testData.items[0])}`);
    });
  });

  describe("Max Reports Limit", () => {
    it("should respect max reports limit", async () => {
      const testData = {
        items: [
          { type: "missing_tool", tool: "tool1", reason: "reason1" },
          { type: "missing_tool", tool: "tool2", reason: "reason2" },
          { type: "missing_tool", tool: "tool3", reason: "reason3" },
          { type: "missing_tool", tool: "tool4", reason: "reason4" },
        ],
        errors: [],
      };

      setAgentOutput(testData);
      process.env.GITHUB_AW_MISSING_TOOL_MAX = "2";

      await runScript();

      expect(mockCore.setOutput).toHaveBeenCalledWith("total_count", "2");
      expect(mockCore.info).toHaveBeenCalledWith("Reached maximum number of missing tool reports (2)");

      const toolsReportedCall = mockCore.setOutput.mock.calls.find(call => call[0] === "tools_reported");
      const reportedTools = JSON.parse(toolsReportedCall[1]);
      expect(reportedTools).toHaveLength(2);
      expect(reportedTools[0].tool).toBe("tool1");
      expect(reportedTools[1].tool).toBe("tool2");
    });

    it("should work without max limit", async () => {
      const testData = {
        items: [
          { type: "missing_tool", tool: "tool1", reason: "reason1" },
          { type: "missing_tool", tool: "tool2", reason: "reason2" },
          { type: "missing_tool", tool: "tool3", reason: "reason3" },
        ],
        errors: [],
      };

      setAgentOutput(testData);
      // No GITHUB_AW_MISSING_TOOL_MAX set

      await runScript();

      expect(mockCore.setOutput).toHaveBeenCalledWith("total_count", "3");
    });
  });

  describe("Edge Cases", () => {
    it("should handle empty agent output", async () => {
      setAgentOutput("");

      await runScript();

      expect(mockCore.setOutput).toHaveBeenCalledWith("total_count", "0");
      expect(mockCore.info).toHaveBeenCalledWith("No agent output to process");
    });

    it("should handle agent output with empty items array", async () => {
      const testData = {
        items: [],
        errors: [],
      };

      setAgentOutput(testData);

      await runScript();

      expect(mockCore.setOutput).toHaveBeenCalledWith("total_count", "0");
      expect(mockCore.info).toHaveBeenCalledWith("Parsed agent output with 0 entries");
    });

    it("should handle missing environment variables", async () => {
      // Don't set any environment variables

      await runScript();

      expect(mockCore.setOutput).toHaveBeenCalledWith("total_count", "0");
    });

    it("should add timestamp to reported tools", async () => {
      const testData = {
        items: [
          {
            type: "missing_tool",
            tool: "test-tool",
            reason: "testing timestamp",
          },
        ],
        errors: [],
      };

      setAgentOutput(testData);

      const beforeTime = new Date();
      await runScript();
      const afterTime = new Date();

      const toolsReportedCall = mockCore.setOutput.mock.calls.find(call => call[0] === "tools_reported");
      const reportedTools = JSON.parse(toolsReportedCall[1]);
      expect(reportedTools).toHaveLength(1);

      const timestamp = new Date(reportedTools[0].timestamp);
      expect(timestamp).toBeInstanceOf(Date);
      expect(timestamp.getTime()).toBeGreaterThanOrEqual(beforeTime.getTime());
      expect(timestamp.getTime()).toBeLessThanOrEqual(afterTime.getTime());
    });
  });
});
