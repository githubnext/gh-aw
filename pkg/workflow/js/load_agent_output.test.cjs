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
};

// Set up global variables
global.core = mockCore;

describe("load_agent_output.cjs", () => {
  let loadAgentOutputModule;
  let tempDir;
  let originalEnv;

  beforeEach(async () => {
    // Save original environment
    originalEnv = process.env.GH_AW_AGENT_OUTPUT;

    // Create temp directory for test files
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "load-agent-output-test-"));

    // Clear all mocks
    vi.clearAllMocks();

    // Dynamically import the module (fresh for each test)
    loadAgentOutputModule = await import("./load_agent_output.cjs");
  });

  afterEach(() => {
    // Restore original environment
    if (originalEnv !== undefined) {
      process.env.GH_AW_AGENT_OUTPUT = originalEnv;
    } else {
      delete process.env.GH_AW_AGENT_OUTPUT;
    }

    // Clean up temp directory
    if (tempDir && fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
  });

  describe("loadAgentOutput", () => {
    it("should return success: false when GH_AW_AGENT_OUTPUT is not set", () => {
      delete process.env.GH_AW_AGENT_OUTPUT;

      const result = loadAgentOutputModule.loadAgentOutput();

      expect(result.success).toBe(false);
      expect(result.items).toBeUndefined();
      expect(mockCore.info).toHaveBeenCalledWith("No GH_AW_AGENT_OUTPUT environment variable found");
    });

    it("should return success: false and set failure when file cannot be read", () => {
      process.env.GH_AW_AGENT_OUTPUT = "/nonexistent/file.json";

      const result = loadAgentOutputModule.loadAgentOutput();

      expect(result.success).toBe(false);
      expect(result.error).toMatch(/Error reading agent output file/);
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Error reading agent output file"));
    });

    it("should return success: false when file content is empty", () => {
      const emptyFile = path.join(tempDir, "empty.json");
      fs.writeFileSync(emptyFile, "");
      process.env.GH_AW_AGENT_OUTPUT = emptyFile;

      const result = loadAgentOutputModule.loadAgentOutput();

      expect(result.success).toBe(false);
      expect(result.items).toBeUndefined();
      expect(mockCore.info).toHaveBeenCalledWith("Agent output content is empty");
    });

    it("should return success: false when file content is only whitespace", () => {
      const whitespaceFile = path.join(tempDir, "whitespace.json");
      fs.writeFileSync(whitespaceFile, "   \n\t   ");
      process.env.GH_AW_AGENT_OUTPUT = whitespaceFile;

      const result = loadAgentOutputModule.loadAgentOutput();

      expect(result.success).toBe(false);
      expect(result.items).toBeUndefined();
      expect(mockCore.info).toHaveBeenCalledWith("Agent output content is empty");
    });

    it("should return success: false and set failure when JSON is invalid", () => {
      const invalidJsonFile = path.join(tempDir, "invalid.json");
      fs.writeFileSync(invalidJsonFile, "{ invalid json }");
      process.env.GH_AW_AGENT_OUTPUT = invalidJsonFile;

      const result = loadAgentOutputModule.loadAgentOutput();

      expect(result.success).toBe(false);
      expect(result.error).toMatch(/Error parsing agent output JSON/);
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Error parsing agent output JSON"));
    });

    it("should return success: false when items field is missing", () => {
      const noItemsFile = path.join(tempDir, "no-items.json");
      fs.writeFileSync(noItemsFile, JSON.stringify({ other: "data" }));
      process.env.GH_AW_AGENT_OUTPUT = noItemsFile;

      const result = loadAgentOutputModule.loadAgentOutput();

      expect(result.success).toBe(false);
      expect(result.items).toBeUndefined();
      expect(mockCore.info).toHaveBeenCalledWith("No valid items found in agent output");
    });

    it("should return success: false when items field is not an array", () => {
      const invalidItemsFile = path.join(tempDir, "invalid-items.json");
      fs.writeFileSync(invalidItemsFile, JSON.stringify({ items: "not-an-array" }));
      process.env.GH_AW_AGENT_OUTPUT = invalidItemsFile;

      const result = loadAgentOutputModule.loadAgentOutput();

      expect(result.success).toBe(false);
      expect(result.items).toBeUndefined();
      expect(mockCore.info).toHaveBeenCalledWith("No valid items found in agent output");
    });

    it("should return success: true with empty items array", () => {
      const emptyItemsFile = path.join(tempDir, "empty-items.json");
      fs.writeFileSync(emptyItemsFile, JSON.stringify({ items: [] }));
      process.env.GH_AW_AGENT_OUTPUT = emptyItemsFile;

      const result = loadAgentOutputModule.loadAgentOutput();

      expect(result.success).toBe(true);
      expect(result.items).toEqual([]);
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Agent output content length:"));
    });

    it("should return success: true with valid items", () => {
      const validFile = path.join(tempDir, "valid.json");
      const items = [
        { type: "create_issue", title: "Test Issue" },
        { type: "add_comment", body: "Test Comment" },
      ];
      fs.writeFileSync(validFile, JSON.stringify({ items }));
      process.env.GH_AW_AGENT_OUTPUT = validFile;

      const result = loadAgentOutputModule.loadAgentOutput();

      expect(result.success).toBe(true);
      expect(result.items).toEqual(items);
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Agent output content length:"));
    });

    it("should log file content length on successful parse", () => {
      const validFile = path.join(tempDir, "valid.json");
      const content = JSON.stringify({ items: [{ type: "test" }] });
      fs.writeFileSync(validFile, content);
      process.env.GH_AW_AGENT_OUTPUT = validFile;

      loadAgentOutputModule.loadAgentOutput();

      expect(mockCore.info).toHaveBeenCalledWith(`Agent output content length: ${content.length}`);
    });

    it("should handle complex nested items structure", () => {
      const complexFile = path.join(tempDir, "complex.json");
      const items = [
        {
          type: "create_issue",
          title: "Complex Issue",
          labels: ["bug", "high-priority"],
          metadata: { nested: { data: "value" } },
        },
      ];
      fs.writeFileSync(complexFile, JSON.stringify({ items }));
      process.env.GH_AW_AGENT_OUTPUT = complexFile;

      const result = loadAgentOutputModule.loadAgentOutput();

      expect(result.success).toBe(true);
      expect(result.items).toEqual(items);
    });
  });

  describe("processAgentOutput", () => {
    let validOutputFile;

    beforeEach(() => {
      // Create a valid output file with mixed item types
      validOutputFile = path.join(tempDir, "mixed-items.json");
      const items = [
        { type: "create_issue", title: "Issue 1", body: "Body 1" },
        { type: "create_issue", title: "Issue 2", body: "Body 2" },
        { type: "add_comment", body: "Comment 1" },
        { type: "add_labels", labels: ["bug", "enhancement"] },
        { type: "create_discussion", title: "Discussion 1" },
      ];
      fs.writeFileSync(validOutputFile, JSON.stringify({ items }));
      process.env.GH_AW_AGENT_OUTPUT = validOutputFile;
    });

    it("should return success: false when agent output loading fails", async () => {
      delete process.env.GH_AW_AGENT_OUTPUT;

      const result = await loadAgentOutputModule.processAgentOutput({
        itemType: "create_issue",
      });

      expect(result.success).toBe(false);
      expect(result.items).toBeUndefined();
    });

    it("should filter items by type and return matching items", async () => {
      const result = await loadAgentOutputModule.processAgentOutput({
        itemType: "create_issue",
      });

      expect(result.success).toBe(true);
      expect(result.isStaged).toBe(false);
      expect(result.items).toHaveLength(2);
      expect(result.items[0].type).toBe("create_issue");
      expect(result.items[1].type).toBe("create_issue");
    });

    it("should return success: false when no items match the type", async () => {
      const result = await loadAgentOutputModule.processAgentOutput({
        itemType: "nonexistent_type",
      });

      expect(result.success).toBe(false);
      expect(mockCore.info).toHaveBeenCalledWith("No nonexistent-type items found in agent output");
    });

    it("should use core.warning for empty results when useWarningForEmpty is true", async () => {
      const result = await loadAgentOutputModule.processAgentOutput({
        itemType: "nonexistent_type",
        useWarningForEmpty: true,
      });

      expect(result.success).toBe(false);
      expect(mockCore.warning).toHaveBeenCalledWith("No nonexistent-type items found in agent output");
    });

    it("should log the number of found items", async () => {
      await loadAgentOutputModule.processAgentOutput({
        itemType: "create_issue",
      });

      expect(mockCore.info).toHaveBeenCalledWith("Found 2 create_issue item(s)");
    });

    it("should find a single item when findOne is true", async () => {
      const result = await loadAgentOutputModule.processAgentOutput({
        itemType: "add_labels",
        findOne: true,
      });

      expect(result.success).toBe(true);
      expect(result.isStaged).toBe(false);
      expect(result.items).toHaveLength(1);
      expect(result.items[0].type).toBe("add_labels");
    });

    it("should return success: false when findOne is true but no item found", async () => {
      const result = await loadAgentOutputModule.processAgentOutput({
        itemType: "nonexistent_type",
        findOne: true,
      });

      expect(result.success).toBe(false);
    });

    it("should handle staged mode without preview config", async () => {
      process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";

      const result = await loadAgentOutputModule.processAgentOutput({
        itemType: "create_issue",
      });

      expect(result.success).toBe(true);
      expect(result.isStaged).toBe(false); // No preview was generated
      expect(result.items).toHaveLength(2);
    });

    it("should handle staged mode with preview config", async () => {
      process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";

      // Mock core.summary
      const mockSummary = {
        addRaw: vi.fn().mockReturnThis(),
        write: vi.fn().mockResolvedValue(undefined),
      };
      global.core.summary = mockSummary;

      const result = await loadAgentOutputModule.processAgentOutput({
        itemType: "create_issue",
        stagedPreview: {
          title: "Create Issues",
          description: "The following issues would be created:",
          renderItem: (item, index) => `### Issue ${index + 1}\n**Title:** ${item.title}\n\n`,
        },
      });

      expect(result.success).toBe(true);
      expect(result.isStaged).toBe(true);
      expect(result.items).toHaveLength(2);
      expect(mockSummary.addRaw).toHaveBeenCalled();
      expect(mockSummary.write).toHaveBeenCalled();
    });

    it("should not trigger staged preview when not in staged mode", async () => {
      delete process.env.GH_AW_SAFE_OUTPUTS_STAGED;

      const mockSummary = {
        addRaw: vi.fn().mockReturnThis(),
        write: vi.fn().mockResolvedValue(undefined),
      };
      global.core.summary = mockSummary;

      const result = await loadAgentOutputModule.processAgentOutput({
        itemType: "create_issue",
        stagedPreview: {
          title: "Create Issues",
          description: "The following issues would be created:",
          renderItem: item => `Item: ${item.title}`,
        },
      });

      expect(result.success).toBe(true);
      expect(result.isStaged).toBe(false);
      expect(mockSummary.addRaw).not.toHaveBeenCalled();
    });

    it("should handle different item types correctly", async () => {
      const commentResult = await loadAgentOutputModule.processAgentOutput({
        itemType: "add_comment",
      });
      expect(commentResult.success).toBe(true);
      expect(commentResult.items).toHaveLength(1);
      expect(commentResult.items[0].type).toBe("add_comment");

      const discussionResult = await loadAgentOutputModule.processAgentOutput({
        itemType: "create_discussion",
      });
      expect(discussionResult.success).toBe(true);
      expect(discussionResult.items).toHaveLength(1);
      expect(discussionResult.items[0].type).toBe("create_discussion");
    });

    it("should preserve item data during filtering", async () => {
      const result = await loadAgentOutputModule.processAgentOutput({
        itemType: "create_issue",
      });

      expect(result.items[0].title).toBe("Issue 1");
      expect(result.items[0].body).toBe("Body 1");
      expect(result.items[1].title).toBe("Issue 2");
      expect(result.items[1].body).toBe("Body 2");
    });

    it("should handle empty items array gracefully", async () => {
      const emptyFile = path.join(tempDir, "empty-items.json");
      fs.writeFileSync(emptyFile, JSON.stringify({ items: [] }));
      process.env.GH_AW_AGENT_OUTPUT = emptyFile;

      const result = await loadAgentOutputModule.processAgentOutput({
        itemType: "create_issue",
      });

      expect(result.success).toBe(false);
    });
  });
});
