import { describe, it, expect, beforeEach, vi } from "vitest";
import fs from "fs";
import path from "path";

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

const mockGithub = {
  rest: {
    issues: {
      addLabels: vi.fn(),
    },
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

// Set up global variables
global.core = mockCore;
global.github = mockGithub;
global.context = mockContext;

describe("safe_output_list_action.cjs", () => {
  let tempFilePath;

  // Helper function to set agent output via file
  const setAgentOutput = data => {
    tempFilePath = path.join("/tmp", `test_agent_output_${Date.now()}_${Math.random().toString(36).slice(2)}.json`);
    const content = typeof data === "string" ? data : JSON.stringify(data);
    fs.writeFileSync(tempFilePath, content);
    process.env.GH_AW_AGENT_OUTPUT = tempFilePath;
  };

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();
    vi.resetModules();

    // Reset environment variables
    delete process.env.GH_AW_AGENT_OUTPUT;
    delete process.env.GH_AW_SAFE_OUTPUTS_STAGED;
    delete process.env.GH_AW_LABELS_ALLOWED;
    delete process.env.GH_AW_LABELS_MAX_COUNT;
    delete process.env.GH_AW_LABELS_TARGET;

    // Reset context to default state
    global.context.eventName = "issues";
    global.context.payload.issue = { number: 123 };
    delete global.context.payload.pull_request;
  });

  afterEach(() => {
    // Clean up temporary file
    if (tempFilePath && fs.existsSync(tempFilePath)) {
      fs.unlinkSync(tempFilePath);
      tempFilePath = undefined;
    }
  });

  describe("validateListItems", () => {
    it("should return error for non-array input", async () => {
      const { validateListItems } = await import("./safe_output_list_action.cjs");
      const result = validateListItems("not an array", undefined, 3);
      expect(result.valid).toBe(false);
      expect(result.error).toBe("items must be an array");
    });

    it("should filter by allowed items", async () => {
      const { validateListItems } = await import("./safe_output_list_action.cjs");
      const result = validateListItems(["item1", "item2", "item3"], ["item1", "item2"], 10);
      expect(result.valid).toBe(true);
      expect(result.value).toEqual(["item1", "item2"]);
    });

    it("should apply max count limit", async () => {
      const { validateListItems } = await import("./safe_output_list_action.cjs");
      const result = validateListItems(["item1", "item2", "item3", "item4"], undefined, 2);
      expect(result.valid).toBe(true);
      expect(result.value).toEqual(["item1", "item2"]);
    });

    it("should deduplicate items", async () => {
      const { validateListItems } = await import("./safe_output_list_action.cjs");
      const result = validateListItems(["item1", "item2", "item1", "item2"], undefined, 10);
      expect(result.valid).toBe(true);
      expect(result.value).toEqual(["item1", "item2"]);
    });

    it("should filter out null, false, and 0 values", async () => {
      const { validateListItems } = await import("./safe_output_list_action.cjs");
      const result = validateListItems(["item1", null, "item2", false, 0, "item3"], undefined, 10);
      expect(result.valid).toBe(true);
      expect(result.value).toEqual(["item1", "item2", "item3"]);
    });

    it("should trim whitespace from items", async () => {
      const { validateListItems } = await import("./safe_output_list_action.cjs");
      const result = validateListItems(["  item1  ", " item2 "], undefined, 10);
      expect(result.valid).toBe(true);
      expect(result.value).toEqual(["item1", "item2"]);
    });

    it("should return error when no valid items remain", async () => {
      const { validateListItems } = await import("./safe_output_list_action.cjs");
      const result = validateListItems(["item1", "item2"], ["item3"], 10);
      expect(result.valid).toBe(false);
      expect(result.error).toBe("No valid items found after filtering");
    });

    it("should allow all items when no allowed list provided", async () => {
      const { validateListItems } = await import("./safe_output_list_action.cjs");
      const result = validateListItems(["item1", "item2", "item3"], undefined, 10);
      expect(result.valid).toBe(true);
      expect(result.value).toEqual(["item1", "item2", "item3"]);
    });
  });

  describe("renderMarkdownList", () => {
    it("should render items as a markdown list", async () => {
      const { renderMarkdownList } = await import("./safe_output_list_action.cjs");
      const result = renderMarkdownList(["item1", "item2", "item3"]);
      expect(result).toBe("- `item1`\n- `item2`\n- `item3`");
    });

    it("should handle empty array", async () => {
      const { renderMarkdownList } = await import("./safe_output_list_action.cjs");
      const result = renderMarkdownList([]);
      expect(result).toBe("");
    });

    it("should handle single item", async () => {
      const { renderMarkdownList } = await import("./safe_output_list_action.cjs");
      const result = renderMarkdownList(["single"]);
      expect(result).toBe("- `single`");
    });
  });

  describe("executeListAction", () => {
    it("should handle missing agent output gracefully", async () => {
      delete process.env.GH_AW_AGENT_OUTPUT;

      const { executeListAction } = await import("./safe_output_list_action.cjs");
      await executeListAction({
        itemType: "add_labels",
        singularNoun: "label",
        pluralNoun: "labels",
        itemsField: "labels",
        configKey: "add_labels",
        configAllowedField: "allowed",
        envAllowedVar: "GH_AW_LABELS_ALLOWED",
        envMaxCountVar: "GH_AW_LABELS_MAX_COUNT",
        envTargetVar: "GH_AW_LABELS_TARGET",
        targetNumberField: "item_number",
        supportsPR: true,
        stagedPreviewTitle: "Add Labels",
        stagedPreviewDescription: "The following labels would be added:",
        renderStagedItem: () => "content",
        applyAction: async () => {},
        outputField: "labels_added",
        summaryTitle: "Label Addition",
        renderSuccessSummary: () => "summary",
      });

      expect(mockCore.info).toHaveBeenCalledWith("No GH_AW_AGENT_OUTPUT environment variable found");
    });

    it("should warn when no matching item found", async () => {
      setAgentOutput({
        items: [{ type: "other_type", data: "value" }],
      });

      const { executeListAction } = await import("./safe_output_list_action.cjs");
      await executeListAction({
        itemType: "add_labels",
        singularNoun: "label",
        pluralNoun: "labels",
        itemsField: "labels",
        configKey: "add_labels",
        configAllowedField: "allowed",
        envAllowedVar: "GH_AW_LABELS_ALLOWED",
        envMaxCountVar: "GH_AW_LABELS_MAX_COUNT",
        envTargetVar: "GH_AW_LABELS_TARGET",
        targetNumberField: "item_number",
        supportsPR: true,
        stagedPreviewTitle: "Add Labels",
        stagedPreviewDescription: "The following labels would be added:",
        renderStagedItem: () => "content",
        applyAction: async () => {},
        outputField: "labels_added",
        summaryTitle: "Label Addition",
        renderSuccessSummary: () => "summary",
      });

      expect(mockCore.warning).toHaveBeenCalledWith("No add-labels item found in agent output");
    });

    it("should generate staged preview in staged mode", async () => {
      setAgentOutput({
        items: [{ type: "add_labels", labels: ["bug", "enhancement"] }],
      });
      process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";

      const { executeListAction } = await import("./safe_output_list_action.cjs");
      await executeListAction({
        itemType: "add_labels",
        singularNoun: "label",
        pluralNoun: "labels",
        itemsField: "labels",
        configKey: "add_labels",
        configAllowedField: "allowed",
        envAllowedVar: "GH_AW_LABELS_ALLOWED",
        envMaxCountVar: "GH_AW_LABELS_MAX_COUNT",
        envTargetVar: "GH_AW_LABELS_TARGET",
        targetNumberField: "item_number",
        supportsPR: true,
        stagedPreviewTitle: "Add Labels",
        stagedPreviewDescription: "The following labels would be added:",
        renderStagedItem: () => "**Test content**",
        applyAction: async () => {},
        outputField: "labels_added",
        summaryTitle: "Label Addition",
        renderSuccessSummary: () => "summary",
      });

      expect(mockCore.summary.addRaw).toHaveBeenCalled();
      expect(mockCore.summary.write).toHaveBeenCalled();
    });

    it("should call applyAction with processed items", async () => {
      setAgentOutput({
        items: [{ type: "add_labels", labels: ["bug", "enhancement"] }],
      });
      process.env.GH_AW_LABELS_MAX_COUNT = "10";

      const mockApplyAction = vi.fn();

      const { executeListAction } = await import("./safe_output_list_action.cjs");
      await executeListAction({
        itemType: "add_labels",
        singularNoun: "label",
        pluralNoun: "labels",
        itemsField: "labels",
        configKey: "add_labels",
        configAllowedField: "allowed",
        envAllowedVar: "GH_AW_LABELS_ALLOWED",
        envMaxCountVar: "GH_AW_LABELS_MAX_COUNT",
        envTargetVar: "GH_AW_LABELS_TARGET",
        targetNumberField: "item_number",
        supportsPR: true,
        stagedPreviewTitle: "Add Labels",
        stagedPreviewDescription: "The following labels would be added:",
        renderStagedItem: () => "content",
        applyAction: mockApplyAction,
        outputField: "labels_added",
        summaryTitle: "Label Addition",
        renderSuccessSummary: () => "summary",
      });

      expect(mockApplyAction).toHaveBeenCalledWith(["bug", "enhancement"], "issue", 123);
    });

    it("should use custom validateItems function when provided", async () => {
      setAgentOutput({
        items: [{ type: "add_labels", labels: ["bug", "feature", "enhancement"] }],
      });

      const mockApplyAction = vi.fn();
      const customValidate = vi.fn().mockReturnValue({ valid: true, value: ["custom-validated"] });

      const { executeListAction } = await import("./safe_output_list_action.cjs");
      await executeListAction({
        itemType: "add_labels",
        singularNoun: "label",
        pluralNoun: "labels",
        itemsField: "labels",
        configKey: "add_labels",
        configAllowedField: "allowed",
        envAllowedVar: "GH_AW_LABELS_ALLOWED",
        envMaxCountVar: "GH_AW_LABELS_MAX_COUNT",
        envTargetVar: "GH_AW_LABELS_TARGET",
        targetNumberField: "item_number",
        supportsPR: true,
        stagedPreviewTitle: "Add Labels",
        stagedPreviewDescription: "The following labels would be added:",
        renderStagedItem: () => "content",
        validateItems: customValidate,
        applyAction: mockApplyAction,
        outputField: "labels_added",
        summaryTitle: "Label Addition",
        renderSuccessSummary: () => "summary",
      });

      expect(customValidate).toHaveBeenCalledWith(["bug", "feature", "enhancement"]);
      expect(mockApplyAction).toHaveBeenCalledWith(["custom-validated"], "issue", 123);
    });
  });
});
