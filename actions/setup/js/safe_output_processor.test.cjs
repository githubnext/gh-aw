import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
import os from "os";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  notice: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

const mockContext = {
  eventName: "issues",
  repo: {
    owner: "testowner",
    repo: "testrepo",
  },
  payload: {
    issue: {
      number: 123,
    },
  },
};

// Set up global mocks before importing the module
global.core = mockCore;
global.context = mockContext;

describe("safe_output_processor", () => {
  let tempDir;
  let outputFile;
  let processor;

  beforeEach(async () => {
    // Create a temporary directory for test files
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "processor-test-"));
    outputFile = path.join(tempDir, "agent-output.json");

    // Reset all mocks before each test
    vi.clearAllMocks();
    vi.resetModules();

    // Clear environment variables
    delete process.env.GH_AW_AGENT_OUTPUT;
    delete process.env.GH_AW_SAFE_OUTPUTS_STAGED;
    delete process.env.GH_AW_TEST_ALLOWED;
    delete process.env.GH_AW_TEST_MAX_COUNT;
    delete process.env.GH_AW_TEST_TARGET;

    // Reset context to default
    global.context = {
      eventName: "issues",
      repo: {
        owner: "testowner",
        repo: "testrepo",
      },
      payload: {
        issue: {
          number: 123,
        },
      },
    };

    // Import the module fresh for each test
    processor = await import("./safe_output_processor.cjs");
  });

  afterEach(() => {
    // Clean up temporary files
    if (fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
  });

  describe("processSafeOutput", () => {
    const defaultConfig = {
      itemType: "test_type",
      configKey: "test_type",
      displayName: "Test",
      itemTypeName: "test item",
      supportsPR: false,
      supportsIssue: true,
      envVars: {
        allowed: "GH_AW_TEST_ALLOWED",
        maxCount: "GH_AW_TEST_MAX_COUNT",
        target: "GH_AW_TEST_TARGET",
      },
    };

    const defaultStagedOptions = {
      title: "Test Preview",
      description: "Test description",
      renderItem: item => `**Test:** ${JSON.stringify(item)}\n`,
    };

    it("should return unsuccessful when agent output is not available", async () => {
      delete process.env.GH_AW_AGENT_OUTPUT;

      const result = await processor.processSafeOutput(defaultConfig, defaultStagedOptions);

      expect(result.success).toBe(false);
      expect(result.reason).toBe("Agent output not available");
    });

    it("should return unsuccessful when no matching items found", async () => {
      const agentOutput = {
        items: [{ type: "other_type", data: "test" }],
      };
      fs.writeFileSync(outputFile, JSON.stringify(agentOutput));
      process.env.GH_AW_AGENT_OUTPUT = outputFile;

      const result = await processor.processSafeOutput(defaultConfig, defaultStagedOptions);

      expect(result.success).toBe(false);
      expect(result.reason).toContain("No test_type item found");
      expect(mockCore.warning).toHaveBeenCalledWith("No test-type item found in agent output");
    });

    it("should handle staged mode and generate preview", async () => {
      const agentOutput = {
        items: [{ type: "test_type", value: "test_value" }],
      };
      fs.writeFileSync(outputFile, JSON.stringify(agentOutput));
      process.env.GH_AW_AGENT_OUTPUT = outputFile;
      process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";

      const result = await processor.processSafeOutput(defaultConfig, defaultStagedOptions);

      expect(result.success).toBe(false);
      expect(result.reason).toBe("Staged mode - preview generated");
      expect(mockCore.summary.addRaw).toHaveBeenCalled();
      expect(mockCore.summary.write).toHaveBeenCalled();
    });

    it("should parse configuration from environment variables", async () => {
      const agentOutput = {
        items: [{ type: "test_type", value: "test_value" }],
      };
      fs.writeFileSync(outputFile, JSON.stringify(agentOutput));
      process.env.GH_AW_AGENT_OUTPUT = outputFile;
      process.env.GH_AW_TEST_ALLOWED = "item1,item2,item3";
      process.env.GH_AW_TEST_MAX_COUNT = "5";
      process.env.GH_AW_TEST_TARGET = "triggering";

      const result = await processor.processSafeOutput(defaultConfig, defaultStagedOptions);

      expect(result.success).toBe(true);
      expect(result.config.allowed).toEqual(["item1", "item2", "item3"]);
      expect(result.config.maxCount).toBe(5);
      expect(result.config.target).toBe("triggering");
    });

    it("should fail when max count is invalid", async () => {
      const agentOutput = {
        items: [{ type: "test_type", value: "test_value" }],
      };
      fs.writeFileSync(outputFile, JSON.stringify(agentOutput));
      process.env.GH_AW_AGENT_OUTPUT = outputFile;
      process.env.GH_AW_TEST_MAX_COUNT = "invalid";

      const result = await processor.processSafeOutput(defaultConfig, defaultStagedOptions);

      expect(result.success).toBe(false);
      expect(result.reason).toBe("Invalid max count configuration");
      expect(mockCore.setFailed).toHaveBeenCalledWith("Invalid max value: invalid. Must be a positive integer");
    });

    it("should resolve target successfully for issue context", async () => {
      const agentOutput = {
        items: [{ type: "test_type", value: "test_value" }],
      };
      fs.writeFileSync(outputFile, JSON.stringify(agentOutput));
      process.env.GH_AW_AGENT_OUTPUT = outputFile;
      process.env.GH_AW_TEST_TARGET = "triggering";

      const result = await processor.processSafeOutput(defaultConfig, defaultStagedOptions);

      expect(result.success).toBe(true);
      expect(result.targetResult.number).toBe(123);
      expect(result.targetResult.contextType).toBe("issue");
    });

    it("should handle explicit target number", async () => {
      const agentOutput = {
        items: [{ type: "test_type", value: "test_value" }],
      };
      fs.writeFileSync(outputFile, JSON.stringify(agentOutput));
      process.env.GH_AW_AGENT_OUTPUT = outputFile;
      process.env.GH_AW_TEST_TARGET = "456";

      const result = await processor.processSafeOutput(defaultConfig, defaultStagedOptions);

      expect(result.success).toBe(true);
      expect(result.targetResult.number).toBe(456);
    });

    it("should handle * target with item_number", async () => {
      const agentOutput = {
        items: [{ type: "test_type", value: "test_value", item_number: 789 }],
      };
      fs.writeFileSync(outputFile, JSON.stringify(agentOutput));
      process.env.GH_AW_AGENT_OUTPUT = outputFile;
      process.env.GH_AW_TEST_TARGET = "*";

      const result = await processor.processSafeOutput(defaultConfig, defaultStagedOptions);

      expect(result.success).toBe(true);
      expect(result.targetResult.number).toBe(789);
    });

    it("should fail when * target but no item_number provided", async () => {
      const agentOutput = {
        items: [{ type: "test_type", value: "test_value" }],
      };
      fs.writeFileSync(outputFile, JSON.stringify(agentOutput));
      process.env.GH_AW_AGENT_OUTPUT = outputFile;
      process.env.GH_AW_TEST_TARGET = "*";

      const result = await processor.processSafeOutput(defaultConfig, defaultStagedOptions);

      expect(result.success).toBe(false);
      expect(mockCore.setFailed).toHaveBeenCalled();
    });

    it("should find multiple items when findMultiple is true", async () => {
      const agentOutput = {
        items: [
          { type: "test_type", value: "value1" },
          { type: "test_type", value: "value2" },
          { type: "other_type", value: "other" },
        ],
      };
      fs.writeFileSync(outputFile, JSON.stringify(agentOutput));
      process.env.GH_AW_AGENT_OUTPUT = outputFile;

      const multiConfig = { ...defaultConfig, findMultiple: true };
      const result = await processor.processSafeOutput(multiConfig, defaultStagedOptions);

      expect(result.success).toBe(true);
      expect(result.items).toHaveLength(2);
      expect(result.items[0].value).toBe("value1");
      expect(result.items[1].value).toBe("value2");
      // Multiple items don't resolve target
      expect(result.targetResult).toBeUndefined();
    });

    it("should log labels count for label items", async () => {
      const agentOutput = {
        items: [{ type: "test_type", labels: ["bug", "enhancement", "help wanted"] }],
      };
      fs.writeFileSync(outputFile, JSON.stringify(agentOutput));
      process.env.GH_AW_AGENT_OUTPUT = outputFile;

      await processor.processSafeOutput(defaultConfig, defaultStagedOptions);

      expect(mockCore.info).toHaveBeenCalledWith("Found test-type item with 3 labels");
    });

    it("should log reviewers count for reviewer items", async () => {
      const agentOutput = {
        items: [{ type: "test_type", reviewers: ["user1", "user2"] }],
      };
      fs.writeFileSync(outputFile, JSON.stringify(agentOutput));
      process.env.GH_AW_AGENT_OUTPUT = outputFile;

      await processor.processSafeOutput(defaultConfig, defaultStagedOptions);

      expect(mockCore.info).toHaveBeenCalledWith("Found test-type item with 2 reviewers");
    });
  });

  describe("sanitizeItems", () => {
    it("should filter out null and undefined values", () => {
      const result = processor.sanitizeItems([null, "valid", undefined, "also valid"]);
      expect(result).toEqual(["valid", "also valid"]);
    });

    it("should filter out false and zero values", () => {
      const result = processor.sanitizeItems([false, "valid", 0, "also valid"]);
      expect(result).toEqual(["valid", "also valid"]);
    });

    it("should convert to string and trim", () => {
      const result = processor.sanitizeItems([123, "  spaced  ", "normal"]);
      expect(result).toEqual(["123", "spaced", "normal"]);
    });

    it("should remove empty strings after trimming", () => {
      const result = processor.sanitizeItems(["valid", "   ", "", "also valid"]);
      expect(result).toEqual(["valid", "also valid"]);
    });

    it("should deduplicate items", () => {
      const result = processor.sanitizeItems(["a", "b", "a", "c", "b"]);
      expect(result).toEqual(["a", "b", "c"]);
    });

    it("should handle complex deduplication with trimming", () => {
      const result = processor.sanitizeItems(["  a  ", "a", " a", "b"]);
      expect(result).toEqual(["a", "b"]);
    });
  });

  describe("filterByAllowed", () => {
    it("should return all items when allowed is undefined", () => {
      const result = processor.filterByAllowed(["a", "b", "c"], undefined);
      expect(result).toEqual(["a", "b", "c"]);
    });

    it("should return all items when allowed is empty", () => {
      const result = processor.filterByAllowed(["a", "b", "c"], []);
      expect(result).toEqual(["a", "b", "c"]);
    });

    it("should filter items by allowed list", () => {
      const result = processor.filterByAllowed(["a", "b", "c", "d"], ["a", "c"]);
      expect(result).toEqual(["a", "c"]);
    });

    it("should return empty array when no items are allowed", () => {
      const result = processor.filterByAllowed(["a", "b", "c"], ["x", "y", "z"]);
      expect(result).toEqual([]);
    });
  });

  describe("limitToMaxCount", () => {
    it("should return all items when under limit", () => {
      const result = processor.limitToMaxCount(["a", "b", "c"], 5);
      expect(result).toEqual(["a", "b", "c"]);
    });

    it("should return all items when exactly at limit", () => {
      const result = processor.limitToMaxCount(["a", "b", "c"], 3);
      expect(result).toEqual(["a", "b", "c"]);
    });

    it("should truncate items when over limit", () => {
      const result = processor.limitToMaxCount(["a", "b", "c", "d", "e"], 3);
      expect(result).toEqual(["a", "b", "c"]);
      expect(mockCore.info).toHaveBeenCalledWith("Too many items (5), limiting to 3");
    });

    it("should handle limit of 1", () => {
      const result = processor.limitToMaxCount(["a", "b", "c"], 1);
      expect(result).toEqual(["a"]);
    });
  });

  describe("processItems", () => {
    it("should apply the full pipeline: filter, sanitize, dedupe, limit", () => {
      const raw = ["a", "b", "a", "c", "d", null, "e", "  a  "];
      const allowed = ["a", "c", "e"];
      const maxCount = 2;

      const result = processor.processItems(raw, allowed, maxCount);

      // Filter by allowed: ["a", "a", "c", "e", "  a  "]
      // Sanitize and dedupe: ["a", "c", "e"]
      // Limit to 2: ["a", "c"]
      expect(result).toEqual(["a", "c"]);
    });

    it("should handle no allowed restrictions", () => {
      const raw = ["a", "b", "c", "a"];
      const result = processor.processItems(raw, undefined, 10);
      expect(result).toEqual(["a", "b", "c"]);
    });

    it("should handle all values filtered out", () => {
      const raw = ["x", "y", "z"];
      const allowed = ["a", "b", "c"];
      const result = processor.processItems(raw, allowed, 10);
      expect(result).toEqual([]);
    });
  });
});
