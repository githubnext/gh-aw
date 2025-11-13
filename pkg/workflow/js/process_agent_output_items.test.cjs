import { describe, it, expect, beforeEach, vi } from "vitest";
import fs from "fs";
import os from "os";
import path from "path";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  debug: vi.fn(),
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

// Set up global mocks before importing the module
global.core = mockCore;

describe("processAgentOutputItems", () => {
  let tempDir;
  let tempFile;

  beforeEach(() => {
    // Reset mocks before each test
    vi.clearAllMocks();

    // Create temp directory for test files
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "test-process-items-"));
    tempFile = path.join(tempDir, "agent_output.json");

    // Clear environment variables
    delete process.env.GH_AW_AGENT_OUTPUT;
    delete process.env.GH_AW_SAFE_OUTPUTS_STAGED;
  });

  afterEach(() => {
    // Clean up temp directory
    if (fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
  });

  describe("output initialization", () => {
    it("should initialize outputs to empty strings", async () => {
      const { processAgentOutputItems } = await import("./process_agent_output_items.cjs");

      // No agent output file
      const result = await processAgentOutputItems({
        itemType: "create_issue",
        outputs: {
          issue_number: "",
          issue_url: "",
        },
      });

      expect(mockCore.setOutput).toHaveBeenCalledWith("issue_number", "");
      expect(mockCore.setOutput).toHaveBeenCalledWith("issue_url", "");
      expect(result.processed).toBe(false);
    });

    it("should not initialize outputs if none provided", async () => {
      const { processAgentOutputItems } = await import("./process_agent_output_items.cjs");

      const result = await processAgentOutputItems({
        itemType: "create_issue",
      });

      expect(mockCore.setOutput).not.toHaveBeenCalled();
      expect(result.processed).toBe(false);
    });
  });

  describe("agent output loading", () => {
    it("should return early if no agent output file specified", async () => {
      const { processAgentOutputItems } = await import("./process_agent_output_items.cjs");

      const result = await processAgentOutputItems({
        itemType: "create_issue",
      });

      expect(result.processed).toBe(false);
      expect(result.staged).toBe(false);
      expect(mockCore.info).toHaveBeenCalledWith("No GH_AW_AGENT_OUTPUT environment variable found");
    });

    it("should return early if agent output file is empty", async () => {
      const { processAgentOutputItems } = await import("./process_agent_output_items.cjs");

      // Create empty file
      fs.writeFileSync(tempFile, "");
      process.env.GH_AW_AGENT_OUTPUT = tempFile;

      const result = await processAgentOutputItems({
        itemType: "create_issue",
      });

      expect(result.processed).toBe(false);
      expect(mockCore.info).toHaveBeenCalledWith("Agent output content is empty");
    });

    it("should return early if agent output has no items", async () => {
      const { processAgentOutputItems } = await import("./process_agent_output_items.cjs");

      fs.writeFileSync(tempFile, JSON.stringify({ items: [] }));
      process.env.GH_AW_AGENT_OUTPUT = tempFile;

      const result = await processAgentOutputItems({
        itemType: "create_issue",
      });

      expect(result.processed).toBe(false);
      expect(result.items).toEqual([]);
      expect(mockCore.info).toHaveBeenCalledWith("No create_issue items found in agent output");
    });
  });

  describe("item filtering", () => {
    it("should filter items by specified type", async () => {
      const { processAgentOutputItems } = await import("./process_agent_output_items.cjs");

      const agentOutput = {
        items: [
          { type: "create_issue", title: "Issue 1" },
          { type: "add_comment", body: "Comment 1" },
          { type: "create_issue", title: "Issue 2" },
          { type: "create_discussion", title: "Discussion 1" },
        ],
      };

      fs.writeFileSync(tempFile, JSON.stringify(agentOutput));
      process.env.GH_AW_AGENT_OUTPUT = tempFile;

      const result = await processAgentOutputItems({
        itemType: "create_issue",
      });

      expect(result.processed).toBe(true);
      expect(result.items).toHaveLength(2);
      expect(result.items[0].title).toBe("Issue 1");
      expect(result.items[1].title).toBe("Issue 2");
      expect(mockCore.info).toHaveBeenCalledWith("Found 2 create_issue item(s)");
    });

    it("should return empty array if no items match type", async () => {
      const { processAgentOutputItems } = await import("./process_agent_output_items.cjs");

      const agentOutput = {
        items: [
          { type: "add_comment", body: "Comment 1" },
          { type: "create_discussion", title: "Discussion 1" },
        ],
      };

      fs.writeFileSync(tempFile, JSON.stringify(agentOutput));
      process.env.GH_AW_AGENT_OUTPUT = tempFile;

      const result = await processAgentOutputItems({
        itemType: "create_issue",
      });

      expect(result.processed).toBe(false);
      expect(result.items).toEqual([]);
      expect(mockCore.info).toHaveBeenCalledWith("No create_issue items found in agent output");
    });
  });

  describe("staged mode handling", () => {
    it("should detect staged mode from environment variable", async () => {
      const { processAgentOutputItems } = await import("./process_agent_output_items.cjs");

      const agentOutput = {
        items: [{ type: "create_issue", title: "Issue 1", body: "Test body" }],
      };

      fs.writeFileSync(tempFile, JSON.stringify(agentOutput));
      process.env.GH_AW_AGENT_OUTPUT = tempFile;
      process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";

      const result = await processAgentOutputItems({
        itemType: "create_issue",
        stagedPreview: {
          title: "Create Issues",
          description: "Preview of issues",
          renderItem: item => `**Title:** ${item.title}\n**Body:** ${item.body}\n`,
        },
      });

      expect(result.processed).toBe(true);
      expect(result.staged).toBe(true);
      expect(mockCore.summary.addRaw).toHaveBeenCalled();
      expect(mockCore.summary.write).toHaveBeenCalled();
    });

    it("should generate staged preview with custom renderer", async () => {
      const { processAgentOutputItems } = await import("./process_agent_output_items.cjs");

      const agentOutput = {
        items: [
          { type: "create_issue", title: "Issue 1", body: "Body 1" },
          { type: "create_issue", title: "Issue 2", body: "Body 2" },
        ],
      };

      fs.writeFileSync(tempFile, JSON.stringify(agentOutput));
      process.env.GH_AW_AGENT_OUTPUT = tempFile;
      process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";

      const renderItem = vi.fn((item, index) => `### Item ${index + 1}\n**Title:** ${item.title}\n`);

      const result = await processAgentOutputItems({
        itemType: "create_issue",
        stagedPreview: {
          title: "Create Issues",
          description: "The following issues would be created",
          renderItem,
        },
      });

      expect(result.processed).toBe(true);
      expect(result.staged).toBe(true);
      expect(renderItem).toHaveBeenCalledTimes(2);
      expect(renderItem).toHaveBeenNthCalledWith(1, agentOutput.items[0], 0);
      expect(renderItem).toHaveBeenNthCalledWith(2, agentOutput.items[1], 1);
    });

    it("should skip staged preview if not configured", async () => {
      const { processAgentOutputItems } = await import("./process_agent_output_items.cjs");

      const agentOutput = {
        items: [{ type: "create_issue", title: "Issue 1" }],
      };

      fs.writeFileSync(tempFile, JSON.stringify(agentOutput));
      process.env.GH_AW_AGENT_OUTPUT = tempFile;
      process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";

      const result = await processAgentOutputItems({
        itemType: "create_issue",
        // No stagedPreview provided
      });

      expect(result.processed).toBe(true);
      expect(result.staged).toBe(true);
      expect(mockCore.summary.addRaw).not.toHaveBeenCalled();
    });
  });

  describe("live mode processing", () => {
    it("should call processItems handler in live mode", async () => {
      const { processAgentOutputItems } = await import("./process_agent_output_items.cjs");

      const agentOutput = {
        items: [
          { type: "create_issue", title: "Issue 1" },
          { type: "create_issue", title: "Issue 2" },
        ],
      };

      fs.writeFileSync(tempFile, JSON.stringify(agentOutput));
      process.env.GH_AW_AGENT_OUTPUT = tempFile;
      process.env.GH_AW_SAFE_OUTPUTS_STAGED = "false";

      const processItems = vi.fn(async items => {
        expect(items).toHaveLength(2);
      });

      const result = await processAgentOutputItems({
        itemType: "create_issue",
        processItems,
      });

      expect(result.processed).toBe(true);
      expect(result.staged).toBe(false);
      expect(processItems).toHaveBeenCalledWith(agentOutput.items);
    });

    it("should skip processItems if not provided", async () => {
      const { processAgentOutputItems } = await import("./process_agent_output_items.cjs");

      const agentOutput = {
        items: [{ type: "create_issue", title: "Issue 1" }],
      };

      fs.writeFileSync(tempFile, JSON.stringify(agentOutput));
      process.env.GH_AW_AGENT_OUTPUT = tempFile;
      process.env.GH_AW_SAFE_OUTPUTS_STAGED = "false";

      const result = await processAgentOutputItems({
        itemType: "create_issue",
        // No processItems handler
      });

      expect(result.processed).toBe(true);
      expect(result.staged).toBe(false);
    });
  });

  describe("integration scenarios", () => {
    it("should handle complete create_issue workflow", async () => {
      const { processAgentOutputItems } = await import("./process_agent_output_items.cjs");

      const agentOutput = {
        items: [
          { type: "create_issue", title: "Bug report", body: "Found a bug" },
          { type: "create_issue", title: "Feature request", body: "Add feature" },
        ],
      };

      fs.writeFileSync(tempFile, JSON.stringify(agentOutput));
      process.env.GH_AW_AGENT_OUTPUT = tempFile;

      const result = await processAgentOutputItems({
        itemType: "create_issue",
        outputs: {
          issue_number: "",
          issue_url: "",
        },
        stagedPreview: {
          title: "Create Issues",
          description: "The following issues would be created if staged mode was disabled:",
          renderItem: (item, index) => {
            let content = `### Issue ${index + 1}\n`;
            content += `**Title:** ${item.title}\n\n`;
            if (item.body) {
              content += `**Body:**\n${item.body}\n\n`;
            }
            return content;
          },
        },
      });

      expect(mockCore.setOutput).toHaveBeenCalledWith("issue_number", "");
      expect(mockCore.setOutput).toHaveBeenCalledWith("issue_url", "");
      expect(result.processed).toBe(true);
      expect(result.items).toHaveLength(2);
    });

    it("should handle complete add_comment workflow", async () => {
      const { processAgentOutputItems } = await import("./process_agent_output_items.cjs");

      const agentOutput = {
        items: [
          { type: "add_comment", body: "Great work!" },
          { type: "add_comment", body: "Thanks for the PR" },
        ],
      };

      fs.writeFileSync(tempFile, JSON.stringify(agentOutput));
      process.env.GH_AW_AGENT_OUTPUT = tempFile;

      const processItems = vi.fn();

      const result = await processAgentOutputItems({
        itemType: "add_comment",
        processItems,
      });

      expect(result.processed).toBe(true);
      expect(result.items).toHaveLength(2);
      expect(processItems).toHaveBeenCalledWith(agentOutput.items);
    });
  });
});
