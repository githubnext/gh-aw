// @ts-check

import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";

describe("determine_conclusion_failure", () => {
  let originalEnv;
  let core;
  let tempFilePath;

  // Helper to set up agent output file
  const setAgentOutput = data => {
    tempFilePath = path.join("/tmp", `test_agent_output_${Date.now()}_${Math.random().toString(36).slice(2)}.json`);
    const content = typeof data === "string" ? data : JSON.stringify(data);
    fs.writeFileSync(tempFilePath, content);
    process.env.GH_AW_AGENT_OUTPUT = tempFilePath;
  };

  beforeEach(() => {
    // Save original environment
    originalEnv = { ...process.env };

    // Setup mock core
    core = {
      info: vi.fn(),
      error: vi.fn(),
      setFailed: vi.fn(),
    };

    // Make core available globally
    global.core = core;

    // Clear temp file path
    tempFilePath = undefined;
  });

  afterEach(() => {
    // Restore environment
    process.env = originalEnv;
    delete global.core;

    // Clean up temp file
    if (tempFilePath && fs.existsSync(tempFilePath)) {
      fs.unlinkSync(tempFilePath);
    }

    vi.clearAllMocks();
  });

  describe("when agent job did not succeed", () => {
    it("should skip check if agent job failed", async () => {
      process.env.GH_AW_AGENT_CONCLUSION = "failure";

      const { main } = await import("./determine_conclusion_failure.cjs");
      await main();

      expect(core.info).toHaveBeenCalledWith(expect.stringContaining("Agent job did not succeed"));
      expect(core.setFailed).not.toHaveBeenCalled();
    });

    it("should skip check if agent job was cancelled", async () => {
      process.env.GH_AW_AGENT_CONCLUSION = "cancelled";

      const { main } = await import("./determine_conclusion_failure.cjs");
      await main();

      expect(core.info).toHaveBeenCalledWith(expect.stringContaining("Agent job did not succeed"));
      expect(core.setFailed).not.toHaveBeenCalled();
    });

    it("should skip check if agent job was skipped", async () => {
      process.env.GH_AW_AGENT_CONCLUSION = "skipped";

      const { main } = await import("./determine_conclusion_failure.cjs");
      await main();

      expect(core.info).toHaveBeenCalledWith(expect.stringContaining("Agent job did not succeed"));
      expect(core.setFailed).not.toHaveBeenCalled();
    });
  });

  describe("when agent job succeeded", () => {
    beforeEach(() => {
      process.env.GH_AW_AGENT_CONCLUSION = "success";
    });

    it("should fail if no agent output file exists", async () => {
      // Don't set up any agent output file

      const { main } = await import("./determine_conclusion_failure.cjs");
      await main();

      expect(core.error).toHaveBeenCalledWith(expect.stringContaining("No agent output found"));
      expect(core.setFailed).toHaveBeenCalledWith(expect.stringContaining("No safe outputs were generated"));
    });

    it("should fail if agent output is empty (no items at all)", async () => {
      setAgentOutput({ items: [] });

      const { main } = await import("./determine_conclusion_failure.cjs");
      await main();

      expect(core.error).toHaveBeenCalledWith(expect.stringContaining("No safe output entries found"));
      expect(core.setFailed).toHaveBeenCalledWith(expect.stringContaining("No safe outputs were generated"));
    });

    it("should pass if agent output has non-noop entries", async () => {
      setAgentOutput({
        items: [{ type: "create_issue", message: "Created issue" }],
      });

      const { main } = await import("./determine_conclusion_failure.cjs");
      await main();

      expect(core.info).toHaveBeenCalledWith(expect.stringContaining("Safe output check passed"));
      expect(core.setFailed).not.toHaveBeenCalled();
    });

    it("should pass if agent output has noop entry", async () => {
      setAgentOutput({
        items: [{ type: "noop", message: "No action taken" }],
      });

      const { main } = await import("./determine_conclusion_failure.cjs");
      await main();

      expect(core.info).toHaveBeenCalledWith(expect.stringContaining("Safe output check passed"));
      expect(core.setFailed).not.toHaveBeenCalled();
    });

    it("should pass if agent output has multiple entries including noop", async () => {
      setAgentOutput({
        items: [
          { type: "create_issue", message: "Created issue" },
          { type: "noop", message: "No action taken" },
        ],
      });

      const { main } = await import("./determine_conclusion_failure.cjs");
      await main();

      expect(core.info).toHaveBeenCalledWith(expect.stringContaining("Safe output check passed"));
      expect(core.setFailed).not.toHaveBeenCalled();
    });

    it("should log correct count of non-noop items", async () => {
      setAgentOutput({
        items: [
          { type: "create_issue", message: "Created issue 1" },
          { type: "add_comment", message: "Added comment" },
          { type: "noop", message: "No action taken" },
        ],
      });

      const { main } = await import("./determine_conclusion_failure.cjs");
      await main();

      expect(core.info).toHaveBeenCalledWith("Found 3 safe output item(s)");
      expect(core.info).toHaveBeenCalledWith("Found 2 non-noop safe output item(s)");
      expect(core.setFailed).not.toHaveBeenCalled();
    });
  });

  describe("when GH_AW_AGENT_CONCLUSION is not set", () => {
    it("should default to empty string and skip check", async () => {
      delete process.env.GH_AW_AGENT_CONCLUSION;

      const { main } = await import("./determine_conclusion_failure.cjs");
      await main();

      expect(core.info).toHaveBeenCalledWith(expect.stringContaining("Agent conclusion:"));
      expect(core.setFailed).not.toHaveBeenCalled();
    });
  });
});
