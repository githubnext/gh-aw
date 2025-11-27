import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
import os from "os";

// Mock the global core object that GitHub Actions provides
const mockCore = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

// Set up global variables
global.core = mockCore;

describe("safe_output_runner.cjs", () => {
  let runnerModule;
  let tempDir;
  let originalEnv;

  beforeEach(async () => {
    // Save original environment
    originalEnv = { ...process.env };

    // Create temp directory for test files
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "safe-output-runner-test-"));

    // Clear all mocks
    vi.clearAllMocks();

    // Reset environment
    delete process.env.GH_AW_SAFE_OUTPUTS_STAGED;

    // Dynamically import the module (fresh for each test)
    runnerModule = await import("./safe_output_runner.cjs");
  });

  afterEach(() => {
    // Restore original environment
    Object.keys(process.env).forEach(key => {
      if (!(key in originalEnv)) {
        delete process.env[key];
      }
    });
    Object.assign(process.env, originalEnv);

    // Clean up temp directory
    if (tempDir && fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
  });

  describe("runSafeOutput", () => {
    it("should return handled: true when no agent output file", async () => {
      delete process.env.GH_AW_AGENT_OUTPUT;

      const processItems = vi.fn();
      const result = await runnerModule.runSafeOutput({
        itemType: "add_labels",
        itemTypePlural: "add-labels",
        processItems,
      });

      expect(result.handled).toBe(true);
      expect(result.result).toBeUndefined();
      expect(processItems).not.toHaveBeenCalled();
    });

    it("should return handled: true and log info when no items found (default)", async () => {
      const testFile = path.join(tempDir, "no-items.json");
      fs.writeFileSync(testFile, JSON.stringify({ items: [{ type: "other_type" }] }));
      process.env.GH_AW_AGENT_OUTPUT = testFile;

      const processItems = vi.fn();
      const result = await runnerModule.runSafeOutput({
        itemType: "add_labels",
        itemTypePlural: "add-labels",
        processItems,
      });

      expect(result.handled).toBe(true);
      expect(mockCore.info).toHaveBeenCalledWith("No add-labels items found in agent output");
      expect(mockCore.warning).not.toHaveBeenCalled();
      expect(processItems).not.toHaveBeenCalled();
    });

    it("should return handled: true and log warning when no items found (warnIfNotFound: true)", async () => {
      const testFile = path.join(tempDir, "no-items.json");
      fs.writeFileSync(testFile, JSON.stringify({ items: [{ type: "other_type" }] }));
      process.env.GH_AW_AGENT_OUTPUT = testFile;

      const processItems = vi.fn();
      const result = await runnerModule.runSafeOutput({
        itemType: "add_labels",
        itemTypePlural: "add-labels",
        warnIfNotFound: true,
        processItems,
      });

      expect(result.handled).toBe(true);
      expect(mockCore.warning).toHaveBeenCalledWith("No add-labels items found in agent output");
      expect(processItems).not.toHaveBeenCalled();
    });

    it("should log item count when items are found", async () => {
      const testFile = path.join(tempDir, "items.json");
      const items = [
        { type: "add_labels", labels: ["bug"] },
        { type: "add_labels", labels: ["enhancement"] },
      ];
      fs.writeFileSync(testFile, JSON.stringify({ items }));
      process.env.GH_AW_AGENT_OUTPUT = testFile;

      const processItems = vi.fn().mockResolvedValue("done");
      await runnerModule.runSafeOutput({
        itemType: "add_labels",
        itemTypePlural: "add-labels",
        processItems,
      });

      expect(mockCore.info).toHaveBeenCalledWith("Found 2 add-labels item(s)");
    });

    it("should generate staged preview when in staged mode", async () => {
      const testFile = path.join(tempDir, "staged-items.json");
      const items = [{ type: "add_labels", labels: ["bug"] }];
      fs.writeFileSync(testFile, JSON.stringify({ items }));
      process.env.GH_AW_AGENT_OUTPUT = testFile;
      process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";

      const processItems = vi.fn();
      const renderStagedItem = vi.fn().mockReturnValue("**Labels:** bug\n");

      const result = await runnerModule.runSafeOutput({
        itemType: "add_labels",
        itemTypePlural: "add-labels",
        stagedTitle: "Add Labels",
        stagedDescription: "The following labels would be added:",
        renderStagedItem,
        processItems,
      });

      expect(result.handled).toBe(true);
      expect(processItems).not.toHaveBeenCalled();
      expect(renderStagedItem).toHaveBeenCalledWith(items[0], 0);
      expect(mockCore.summary.addRaw).toHaveBeenCalled();
      expect(mockCore.summary.write).toHaveBeenCalled();
    });

    it("should skip staged preview if missing staged config", async () => {
      const testFile = path.join(tempDir, "staged-items.json");
      const items = [{ type: "add_labels", labels: ["bug"] }];
      fs.writeFileSync(testFile, JSON.stringify({ items }));
      process.env.GH_AW_AGENT_OUTPUT = testFile;
      process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";

      const processItems = vi.fn();

      const result = await runnerModule.runSafeOutput({
        itemType: "add_labels",
        itemTypePlural: "add-labels",
        // No staged config provided
        processItems,
      });

      expect(result.handled).toBe(true);
      expect(processItems).not.toHaveBeenCalled();
      expect(mockCore.summary.addRaw).not.toHaveBeenCalled();
    });

    it("should call processItems with filtered items when not staged", async () => {
      const testFile = path.join(tempDir, "process-items.json");
      const labelItems = [
        { type: "add_labels", labels: ["bug"] },
        { type: "add_labels", labels: ["enhancement"] },
      ];
      const allItems = [...labelItems, { type: "other", data: "test" }];
      fs.writeFileSync(testFile, JSON.stringify({ items: allItems }));
      process.env.GH_AW_AGENT_OUTPUT = testFile;

      const processItems = vi.fn().mockResolvedValue({ success: true, count: 2 });

      const result = await runnerModule.runSafeOutput({
        itemType: "add_labels",
        itemTypePlural: "add-labels",
        processItems,
      });

      expect(result.handled).toBe(false);
      expect(result.result).toEqual({ success: true, count: 2 });
      expect(processItems).toHaveBeenCalledWith(labelItems);
    });

    it("should filter items correctly by type", async () => {
      const testFile = path.join(tempDir, "mixed-items.json");
      const items = [
        { type: "add_labels", labels: ["bug"] },
        { type: "create_issue", title: "Test" },
        { type: "add_labels", labels: ["enhancement"] },
        { type: "add_comment", body: "Hello" },
      ];
      fs.writeFileSync(testFile, JSON.stringify({ items }));
      process.env.GH_AW_AGENT_OUTPUT = testFile;

      const processItems = vi.fn().mockResolvedValue("done");

      await runnerModule.runSafeOutput({
        itemType: "add_labels",
        itemTypePlural: "add-labels",
        processItems,
      });

      const calledItems = processItems.mock.calls[0][0];
      expect(calledItems).toHaveLength(2);
      expect(calledItems[0]).toEqual({ type: "add_labels", labels: ["bug"] });
      expect(calledItems[1]).toEqual({ type: "add_labels", labels: ["enhancement"] });
    });
  });

  describe("runSingleItemSafeOutput", () => {
    it("should call processSingleItem with the first matching item", async () => {
      const testFile = path.join(tempDir, "single-item.json");
      const items = [
        { type: "add_labels", labels: ["first"] },
        { type: "add_labels", labels: ["second"] },
      ];
      fs.writeFileSync(testFile, JSON.stringify({ items }));
      process.env.GH_AW_AGENT_OUTPUT = testFile;

      const processSingleItem = vi.fn().mockResolvedValue({ success: true });

      const result = await runnerModule.runSingleItemSafeOutput({
        itemType: "add_labels",
        itemTypePlural: "add-labels",
        processSingleItem,
      });

      expect(result.handled).toBe(false);
      expect(result.result).toEqual({ success: true });
      expect(processSingleItem).toHaveBeenCalledWith({ type: "add_labels", labels: ["first"] });
    });

    it("should warn if no items found by default", async () => {
      const testFile = path.join(tempDir, "no-items.json");
      fs.writeFileSync(testFile, JSON.stringify({ items: [{ type: "other" }] }));
      process.env.GH_AW_AGENT_OUTPUT = testFile;

      const processSingleItem = vi.fn();

      await runnerModule.runSingleItemSafeOutput({
        itemType: "add_labels",
        itemTypePlural: "add-labels",
        processSingleItem,
      });

      expect(mockCore.warning).toHaveBeenCalledWith("No add-labels items found in agent output");
    });

    it("should respect warnIfNotFound: false override", async () => {
      const testFile = path.join(tempDir, "no-items.json");
      fs.writeFileSync(testFile, JSON.stringify({ items: [{ type: "other" }] }));
      process.env.GH_AW_AGENT_OUTPUT = testFile;

      const processSingleItem = vi.fn();

      await runnerModule.runSingleItemSafeOutput({
        itemType: "add_labels",
        itemTypePlural: "add-labels",
        warnIfNotFound: false,
        processSingleItem,
      });

      expect(mockCore.info).toHaveBeenCalledWith("No add-labels items found in agent output");
      expect(mockCore.warning).not.toHaveBeenCalled();
    });

    it("should log processing message", async () => {
      const testFile = path.join(tempDir, "single-item.json");
      const items = [{ type: "add_labels", labels: ["bug"] }];
      fs.writeFileSync(testFile, JSON.stringify({ items }));
      process.env.GH_AW_AGENT_OUTPUT = testFile;

      const processSingleItem = vi.fn().mockResolvedValue("done");

      await runnerModule.runSingleItemSafeOutput({
        itemType: "add_labels",
        itemTypePlural: "add-labels",
        processSingleItem,
      });

      expect(mockCore.info).toHaveBeenCalledWith("Processing add-labels item");
    });

    it("should handle staged mode correctly", async () => {
      const testFile = path.join(tempDir, "staged-single.json");
      const items = [{ type: "add_labels", labels: ["bug"] }];
      fs.writeFileSync(testFile, JSON.stringify({ items }));
      process.env.GH_AW_AGENT_OUTPUT = testFile;
      process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";

      const processSingleItem = vi.fn();
      const renderStagedItem = vi.fn().mockReturnValue("**Labels:** bug\n");

      const result = await runnerModule.runSingleItemSafeOutput({
        itemType: "add_labels",
        itemTypePlural: "add-labels",
        stagedTitle: "Add Labels",
        stagedDescription: "Labels to add:",
        renderStagedItem,
        processSingleItem,
      });

      expect(result.handled).toBe(true);
      expect(processSingleItem).not.toHaveBeenCalled();
      expect(renderStagedItem).toHaveBeenCalled();
    });
  });
});
