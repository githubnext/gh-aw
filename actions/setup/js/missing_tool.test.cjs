import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
describe("missing_tool.cjs", () => {
  let mockCore, missingToolScript, originalConsole, tempFilePath;
  const setAgentOutput = data => {
    tempFilePath = path.join("/tmp", `test_agent_output_${Date.now()}_${Math.random().toString(36).slice(2)}.json`);
    const content = "string" == typeof data ? data : JSON.stringify(data);
    (fs.writeFileSync(tempFilePath, content), (process.env.GH_AW_AGENT_OUTPUT = tempFilePath));
  };
  (beforeEach(() => {
    ((originalConsole = global.console),
      (global.console = { log: vi.fn(), error: vi.fn() }),
      (mockCore = {
        debug: vi.fn(),
        info: vi.fn(),
        notice: vi.fn(),
        warning: vi.fn(),
        error: vi.fn(),
        setFailed: vi.fn(),
        setOutput: vi.fn(),
        exportVariable: vi.fn(),
        setSecret: vi.fn(),
        getInput: vi.fn(),
        getBooleanInput: vi.fn(),
        getMultilineInput: vi.fn(),
        getState: vi.fn(),
        saveState: vi.fn(),
        startGroup: vi.fn(),
        endGroup: vi.fn(),
        group: vi.fn(),
        addPath: vi.fn(),
        setCommandEcho: vi.fn(),
        isDebug: vi.fn().mockReturnValue(!1),
        getIDToken: vi.fn(),
        toPlatformPath: vi.fn(),
        toPosixPath: vi.fn(),
        toWin32Path: vi.fn(),
        summary: { addRaw: vi.fn().mockReturnThis(), addHeading: vi.fn().mockReturnThis(), write: vi.fn().mockResolvedValue() },
      }),
      (global.core = mockCore),
      (global.module = { exports: {} }), // Add module for exports
      (global.require = vi.fn().mockImplementation(module => {
        if ("fs" === module) return fs;
        if ("@actions/core" === module) return mockCore;
        if ("./error_helpers.cjs" === module) return { getErrorMessage: error => (error instanceof Error ? error.message : String(error)) };
        throw new Error(`Module not found: ${module}`);
      })));
    const scriptPath = path.join(__dirname, "missing_tool.cjs");
    missingToolScript = fs.readFileSync(scriptPath, "utf8");
  }),
    afterEach(() => {
      (tempFilePath && require("fs").existsSync(tempFilePath) && (require("fs").unlinkSync(tempFilePath), (tempFilePath = void 0)),
        delete process.env.GH_AW_AGENT_OUTPUT,
        delete process.env.GH_AW_MISSING_TOOL_MAX,
        (global.console = originalConsole),
        delete global.core,
        delete global.module,
        delete global.require);
    }));
  const runScript = async () => {
    new Function(missingToolScript)();
    if (global.module.exports.main) {
      await global.module.exports.main();
    }
  };
  (describe("JSON Array Input Format", () => {
    (it("should parse JSON array with missing-tool entries", async () => {
      (setAgentOutput({
        items: [
          { type: "missing_tool", tool: "docker", reason: "Need containerization support", alternatives: "Use VM or manual setup" },
          { type: "missing_tool", tool: "kubectl", reason: "Kubernetes cluster management required" },
        ],
        errors: [],
      }),
        await runScript(),
        expect(mockCore.setOutput).toHaveBeenCalledWith("total_count", "2"));
      const toolsReportedCall = mockCore.setOutput.mock.calls.find(call => "tools_reported" === call[0]);
      expect(toolsReportedCall).toBeDefined();
      const reportedTools = JSON.parse(toolsReportedCall[1]);
      (expect(reportedTools).toHaveLength(2),
        expect(reportedTools[0].tool).toBe("docker"),
        expect(reportedTools[0].reason).toBe("Need containerization support"),
        expect(reportedTools[0].alternatives).toBe("Use VM or manual setup"),
        expect(reportedTools[1].tool).toBe("kubectl"),
        expect(reportedTools[1].reason).toBe("Kubernetes cluster management required"),
        expect(reportedTools[1].alternatives).toBe(null));
    }),
      it("should filter out non-missing-tool entries", async () => {
        (setAgentOutput({
          items: [
            { type: "missing_tool", tool: "docker", reason: "Need containerization" },
            { type: "other-type", data: "should be ignored" },
            { type: "missing_tool", tool: "kubectl", reason: "Need k8s support" },
          ],
          errors: [],
        }),
          await runScript(),
          expect(mockCore.setOutput).toHaveBeenCalledWith("total_count", "2"));
        const toolsReportedCall = mockCore.setOutput.mock.calls.find(call => "tools_reported" === call[0]),
          reportedTools = JSON.parse(toolsReportedCall[1]);
        (expect(reportedTools).toHaveLength(2), expect(reportedTools[0].tool).toBe("docker"), expect(reportedTools[1].tool).toBe("kubectl"));
      }));
  }),
    describe("Validation", () => {
      (it("should skip entries missing tool field", async () => {
        const testData = {
          items: [
            { type: "missing_tool", reason: "No tool specified" },
            { type: "missing_tool", tool: "valid-tool", reason: "This should work" },
          ],
          errors: [],
        };
        (setAgentOutput(testData),
          await runScript(),
          expect(mockCore.setOutput).toHaveBeenCalledWith("total_count", "1"),
          expect(mockCore.warning).toHaveBeenCalledWith(`missing-tool entry missing 'tool' field: ${JSON.stringify(testData.items[0])}`));
      }),
        it("should skip entries missing reason field", async () => {
          const testData = {
            items: [
              { type: "missing_tool", tool: "some-tool" },
              { type: "missing_tool", tool: "valid-tool", reason: "This should work" },
            ],
            errors: [],
          };
          (setAgentOutput(testData),
            await runScript(),
            expect(mockCore.setOutput).toHaveBeenCalledWith("total_count", "1"),
            expect(mockCore.warning).toHaveBeenCalledWith(`missing-tool entry missing 'reason' field: ${JSON.stringify(testData.items[0])}`));
        }));
    }),
    describe("Max Reports Limit", () => {
      (it("should respect max reports limit", async () => {
        (setAgentOutput({
          items: [
            { type: "missing_tool", tool: "tool1", reason: "reason1" },
            { type: "missing_tool", tool: "tool2", reason: "reason2" },
            { type: "missing_tool", tool: "tool3", reason: "reason3" },
            { type: "missing_tool", tool: "tool4", reason: "reason4" },
          ],
          errors: [],
        }),
          (process.env.GH_AW_MISSING_TOOL_MAX = "2"),
          await runScript(),
          expect(mockCore.setOutput).toHaveBeenCalledWith("total_count", "2"),
          expect(mockCore.info).toHaveBeenCalledWith("Reached maximum number of missing tool reports (2)"));
        const toolsReportedCall = mockCore.setOutput.mock.calls.find(call => "tools_reported" === call[0]),
          reportedTools = JSON.parse(toolsReportedCall[1]);
        (expect(reportedTools).toHaveLength(2), expect(reportedTools[0].tool).toBe("tool1"), expect(reportedTools[1].tool).toBe("tool2"));
      }),
        it("should work without max limit", async () => {
          (setAgentOutput({
            items: [
              { type: "missing_tool", tool: "tool1", reason: "reason1" },
              { type: "missing_tool", tool: "tool2", reason: "reason2" },
              { type: "missing_tool", tool: "tool3", reason: "reason3" },
            ],
            errors: [],
          }),
            await runScript(),
            expect(mockCore.setOutput).toHaveBeenCalledWith("total_count", "3"));
        }));
    }),
    describe("Edge Cases", () => {
      (it("should handle empty agent output", async () => {
        (setAgentOutput(""), await runScript(), expect(mockCore.setOutput).toHaveBeenCalledWith("total_count", "0"), expect(mockCore.info).toHaveBeenCalledWith("No agent output to process"));
      }),
        it("should handle agent output with empty items array", async () => {
          (setAgentOutput({ items: [], errors: [] }), await runScript(), expect(mockCore.setOutput).toHaveBeenCalledWith("total_count", "0"), expect(mockCore.info).toHaveBeenCalledWith("Parsed agent output with 0 entries"));
        }),
        it("should handle missing environment variables", async () => {
          (await runScript(), expect(mockCore.setOutput).toHaveBeenCalledWith("total_count", "0"));
        }),
        it("should add timestamp to reported tools", async () => {
          setAgentOutput({ items: [{ type: "missing_tool", tool: "test-tool", reason: "testing timestamp" }], errors: [] });
          const beforeTime = new Date();
          await runScript();
          const afterTime = new Date(),
            toolsReportedCall = mockCore.setOutput.mock.calls.find(call => "tools_reported" === call[0]),
            reportedTools = JSON.parse(toolsReportedCall[1]);
          expect(reportedTools).toHaveLength(1);
          const timestamp = new Date(reportedTools[0].timestamp);
          (expect(timestamp).toBeInstanceOf(Date), expect(timestamp.getTime()).toBeGreaterThanOrEqual(beforeTime.getTime()), expect(timestamp.getTime()).toBeLessThanOrEqual(afterTime.getTime()));
        }),
        it("should handle missing agent output file gracefully with info message", async () => {
          ((process.env.GH_AW_AGENT_OUTPUT = "/nonexistent/path/to/file.json"),
            await runScript(),
            expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Agent output file not found or unreadable")),
            expect(mockCore.setFailed).not.toHaveBeenCalled(),
            expect(mockCore.setOutput).toHaveBeenCalledWith("total_count", "0"),
            expect(mockCore.setOutput).toHaveBeenCalledWith("tools_reported", JSON.stringify([])));
        }));
    }));
});
