import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
describe("noop", () => {
  let mockCore, noopScript, tempFilePath;
  const setAgentOutput = data => {
    tempFilePath = path.join("/tmp", `test_agent_output_${Date.now()}_${Math.random().toString(36).slice(2)}.json`);
    const content = "string" == typeof data ? data : JSON.stringify(data);
    (fs.writeFileSync(tempFilePath, content), (process.env.GH_AW_AGENT_OUTPUT = tempFilePath));
  };
  (beforeEach(() => {
    ((mockCore = { debug: vi.fn(), info: vi.fn(), warning: vi.fn(), error: vi.fn(), setFailed: vi.fn(), setOutput: vi.fn(), exportVariable: vi.fn(), summary: { addRaw: vi.fn().mockReturnThis(), write: vi.fn().mockResolvedValue() } }),
      (global.core = mockCore),
      (global.fs = fs));
    const scriptPath = path.join(process.cwd(), "noop.cjs");
    ((noopScript = fs.readFileSync(scriptPath, "utf8")), delete process.env.GH_AW_SAFE_OUTPUTS_STAGED, delete process.env.GH_AW_AGENT_OUTPUT);
  }),
    afterEach(() => {
      tempFilePath && fs.existsSync(tempFilePath) && (fs.unlinkSync(tempFilePath), (tempFilePath = void 0));
    }),
    it("should handle empty agent output", async () => {
      (setAgentOutput({ items: [], errors: [] }), await eval(`(async () => { ${noopScript}; await main(); })()`), expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No noop items found")));
    }),
    it("should process single noop message", async () => {
      (setAgentOutput({ items: [{ type: "noop", message: "No issues found in this review" }], errors: [] }),
        await eval(`(async () => { ${noopScript}; await main(); })()`),
        expect(mockCore.info).toHaveBeenCalledWith("Found 1 noop item(s)"),
        expect(mockCore.info).toHaveBeenCalledWith("No-op message 1: No issues found in this review"),
        expect(mockCore.setOutput).toHaveBeenCalledWith("noop_message", "No issues found in this review"),
        expect(mockCore.exportVariable).toHaveBeenCalledWith("GH_AW_NOOP_MESSAGE", "No issues found in this review"),
        expect(mockCore.summary.addRaw).toHaveBeenCalled(),
        expect(mockCore.summary.write).toHaveBeenCalled());
    }),
    it("should process multiple noop messages", async () => {
      (setAgentOutput({
        items: [
          { type: "noop", message: "First message" },
          { type: "noop", message: "Second message" },
          { type: "noop", message: "Third message" },
        ],
        errors: [],
      }),
        await eval(`(async () => { ${noopScript}; await main(); })()`),
        expect(mockCore.info).toHaveBeenCalledWith("Found 3 noop item(s)"),
        expect(mockCore.info).toHaveBeenCalledWith("No-op message 1: First message"),
        expect(mockCore.info).toHaveBeenCalledWith("No-op message 2: Second message"),
        expect(mockCore.info).toHaveBeenCalledWith("No-op message 3: Third message"),
        expect(mockCore.setOutput).toHaveBeenCalledWith("noop_message", "First message"),
        expect(mockCore.exportVariable).toHaveBeenCalledWith("GH_AW_NOOP_MESSAGE", "First message"));
    }),
    it("should show preview in staged mode", async () => {
      ((process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true"),
        setAgentOutput({ items: [{ type: "noop", message: "Test message in staged mode" }], errors: [] }),
        await eval(`(async () => { ${noopScript}; await main(); })()`),
        expect(mockCore.info).toHaveBeenCalledWith("Found 1 noop item(s)"),
        expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("📝 No-op message preview written to step summary")),
        expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("🎭 Staged Mode")),
        expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("Test message in staged mode")),
        expect(mockCore.setOutput).not.toHaveBeenCalled());
    }),
    it("should ignore non-noop items", async () => {
      (setAgentOutput({
        items: [
          { type: "create_issue", title: "Test Issue", body: "Test body" },
          { type: "noop", message: "This is the only noop" },
          { type: "add_comment", body: "Test comment" },
        ],
        errors: [],
      }),
        await eval(`(async () => { ${noopScript}; await main(); })()`),
        expect(mockCore.info).toHaveBeenCalledWith("Found 1 noop item(s)"),
        expect(mockCore.info).toHaveBeenCalledWith("No-op message 1: This is the only noop"));
    }),
    it("should handle missing agent output file", async () => {
      // Note: loadAgentOutput uses core.info (not core.error) for missing files
      // because this is a normal scenario when the agent fails before producing safe-outputs
      ((process.env.GH_AW_AGENT_OUTPUT = "/tmp/nonexistent.json"), await eval(`(async () => { ${noopScript}; await main(); })()`), expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Error reading agent output file")));
    }),
    it("should generate proper step summary format", async () => {
      (setAgentOutput({
        items: [
          { type: "noop", message: "Analysis complete" },
          { type: "noop", message: "No action required" },
        ],
        errors: [],
      }),
        await eval(`(async () => { ${noopScript}; await main(); })()`));
      const summaryCall = mockCore.summary.addRaw.mock.calls[0][0];
      (expect(summaryCall).toContain("## No-Op Messages"), expect(summaryCall).toContain("- Analysis complete"), expect(summaryCall).toContain("- No action required"));
    }));
});
