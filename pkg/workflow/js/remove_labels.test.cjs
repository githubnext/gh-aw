import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import fs from "fs";
import path from "path";

// Mock the global objects that GitHub Actions provides
const mockCore = {
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

const mockGithub = {
  rest: {
    issues: {
      removeLabel: vi.fn(),
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

describe("remove_labels.cjs", () => {
  let removeLabelsScript;

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

    // Reset environment variables
    delete process.env.GH_AW_AGENT_OUTPUT;
    delete process.env.GH_AW_LABELS_ALLOWED;
    delete process.env.GH_AW_LABELS_MAX_COUNT;

    // Reset context to default state
    global.context.eventName = "issues";
    global.context.payload.issue = { number: 123 };
    delete global.context.payload.pull_request;

    // Read the script content
    const scriptPath = path.join(process.cwd(), "remove_labels.cjs");
    removeLabelsScript = fs.readFileSync(scriptPath, "utf8");
  });

  afterEach(() => {
    // Clean up temporary file
    if (tempFilePath && require("fs").existsSync(tempFilePath)) {
      require("fs").unlinkSync(tempFilePath);
      tempFilePath = undefined;
    }
  });

  describe("Environment variable validation", () => {
    it("should skip when no agent output is provided", async () => {
      process.env.GH_AW_LABELS_ALLOWED = "bug,enhancement";
      delete process.env.GH_AW_AGENT_OUTPUT;

      // Execute the script
      await eval(`(async () => { ${removeLabelsScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith("No GH_AW_AGENT_OUTPUT environment variable found");
      expect(mockGithub.rest.issues.removeLabel).not.toHaveBeenCalled();
    });

    it("should skip when agent output is empty", async () => {
      setAgentOutput("");
      process.env.GH_AW_LABELS_ALLOWED = "bug,enhancement";

      // Execute the script
      await eval(`(async () => { ${removeLabelsScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith("Agent output content is empty");
      expect(mockGithub.rest.issues.removeLabel).not.toHaveBeenCalled();
    });

    it("should work when allowed labels are not provided (any labels allowed)", async () => {
      setAgentOutput({
        items: [
          {
            type: "remove_labels",
            labels: ["bug", "enhancement", "custom-label"],
          },
        ],
      });
      delete process.env.GH_AW_LABELS_ALLOWED;

      mockGithub.rest.issues.removeLabel.mockResolvedValue({});

      // Execute the script
      await eval(`(async () => { ${removeLabelsScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith("No label restrictions - any labels are allowed");
      expect(mockGithub.rest.issues.removeLabel).toHaveBeenCalledTimes(3);
    });

    it("should enforce allowed labels when restrictions are set", async () => {
      setAgentOutput({
        items: [
          {
            type: "remove_labels",
            labels: ["bug", "enhancement", "custom-label", "documentation"],
          },
        ],
      });
      process.env.GH_AW_LABELS_ALLOWED = "bug,enhancement";

      mockGithub.rest.issues.removeLabel.mockResolvedValue({});

      // Execute the script
      await eval(`(async () => { ${removeLabelsScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith(`Allowed labels: ${JSON.stringify(["bug", "enhancement"])}`);
      expect(mockGithub.rest.issues.removeLabel).toHaveBeenCalledTimes(2);
      expect(mockGithub.rest.issues.removeLabel).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 123,
        name: "bug",
      });
      expect(mockGithub.rest.issues.removeLabel).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 123,
        name: "enhancement",
      });
    });

    it("should fail when max count is invalid", async () => {
      setAgentOutput({
        items: [
          {
            type: "remove_labels",
            labels: ["bug", "enhancement"],
          },
        ],
      });
      process.env.GH_AW_LABELS_ALLOWED = "bug,enhancement";
      process.env.GH_AW_LABELS_MAX_COUNT = "invalid";

      // Execute the script
      await eval(`(async () => { ${removeLabelsScript} })()`);

      expect(mockCore.setFailed).toHaveBeenCalledWith("Invalid max value: invalid. Must be a positive integer");
      expect(mockGithub.rest.issues.removeLabel).not.toHaveBeenCalled();
    });

    it("should use default max count when not specified", async () => {
      setAgentOutput({
        items: [
          {
            type: "remove_labels",
            labels: ["bug", "enhancement", "feature", "documentation"],
          },
        ],
      });
      process.env.GH_AW_LABELS_ALLOWED = "bug,enhancement,feature,documentation";
      delete process.env.GH_AW_LABELS_MAX_COUNT;

      mockGithub.rest.issues.removeLabel.mockResolvedValue({});

      // Execute the script
      await eval(`(async () => { ${removeLabelsScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith("Max count: 3");
      expect(mockGithub.rest.issues.removeLabel).toHaveBeenCalledTimes(3); // Only first 3 due to default max count
    });
  });

  describe("Context validation", () => {
    it("should skip when not in issue or PR context (with default target)", async () => {
      setAgentOutput({
        items: [
          {
            type: "remove_labels",
            labels: ["bug", "enhancement"],
          },
        ],
      });
      process.env.GH_AW_LABELS_ALLOWED = "bug,enhancement";
      global.context.eventName = "push";

      // Execute the script
      await eval(`(async () => { ${removeLabelsScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith(
        'Target is "triggering" but not running in issue or pull request context, skipping label removal'
      );
      expect(mockGithub.rest.issues.removeLabel).not.toHaveBeenCalled();
    });

    it("should work with pull_request event", async () => {
      setAgentOutput({
        items: [
          {
            type: "remove_labels",
            labels: ["bug"],
          },
        ],
      });
      process.env.GH_AW_LABELS_ALLOWED = "bug,enhancement";
      global.context.eventName = "pull_request";
      global.context.payload.pull_request = { number: 456 };
      delete global.context.payload.issue;

      mockGithub.rest.issues.removeLabel.mockResolvedValue({});

      // Execute the script
      await eval(`(async () => { ${removeLabelsScript} })()`);

      expect(mockGithub.rest.issues.removeLabel).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 456,
        name: "bug",
      });
    });
  });

  describe("Label parsing and validation", () => {
    it("should parse labels from agent output and remove valid ones", async () => {
      setAgentOutput({
        items: [
          {
            type: "remove_labels",
            labels: ["bug", "enhancement", "documentation"],
          },
        ],
      });
      process.env.GH_AW_LABELS_ALLOWED = "bug,enhancement,feature";

      mockGithub.rest.issues.removeLabel.mockResolvedValue({});

      // Execute the script
      await eval(`(async () => { ${removeLabelsScript} })()`);

      expect(mockGithub.rest.issues.removeLabel).toHaveBeenCalledTimes(2); // Only bug and enhancement
      expect(mockCore.setOutput).toHaveBeenCalledWith("labels_removed", "bug\nenhancement");
      expect(mockCore.summary.addRaw).toHaveBeenCalled();
      expect(mockCore.summary.write).toHaveBeenCalled();
    });

    it("should remove duplicate labels", async () => {
      setAgentOutput({
        items: [
          {
            type: "remove_labels",
            labels: ["bug", "enhancement", "bug", "enhancement"],
          },
        ],
      });
      process.env.GH_AW_LABELS_ALLOWED = "bug,enhancement";

      mockGithub.rest.issues.removeLabel.mockResolvedValue({});

      // Execute the script
      await eval(`(async () => { ${removeLabelsScript} })()`);

      expect(mockGithub.rest.issues.removeLabel).toHaveBeenCalledTimes(2); // Duplicates removed
    });

    it("should enforce max count limit", async () => {
      setAgentOutput({
        items: [
          {
            type: "remove_labels",
            labels: ["bug", "enhancement", "feature", "documentation", "question"],
          },
        ],
      });
      process.env.GH_AW_LABELS_ALLOWED = "bug,enhancement,feature,documentation,question";
      process.env.GH_AW_LABELS_MAX_COUNT = "2";

      mockGithub.rest.issues.removeLabel.mockResolvedValue({});

      // Execute the script
      await eval(`(async () => { ${removeLabelsScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith("too many labels, keep 2");
      expect(mockGithub.rest.issues.removeLabel).toHaveBeenCalledTimes(2); // Only first 2
    });

    it("should skip when no valid labels found", async () => {
      setAgentOutput({
        items: [
          {
            type: "remove_labels",
            labels: ["invalid", "another-invalid"],
          },
        ],
      });
      process.env.GH_AW_LABELS_ALLOWED = "bug,enhancement";

      // Execute the script
      await eval(`(async () => { ${removeLabelsScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith("No labels to remove");
      expect(mockCore.setOutput).toHaveBeenCalledWith("labels_removed", "");
      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("No labels were removed"));
      expect(mockGithub.rest.issues.removeLabel).not.toHaveBeenCalled();
    });
  });

  describe("GitHub API integration", () => {
    it("should successfully remove labels from issue", async () => {
      setAgentOutput({
        items: [
          {
            type: "remove_labels",
            labels: ["bug", "enhancement"],
          },
        ],
      });
      process.env.GH_AW_LABELS_ALLOWED = "bug,enhancement,feature";

      mockGithub.rest.issues.removeLabel.mockResolvedValue({});

      // Execute the script
      await eval(`(async () => { ${removeLabelsScript} })()`);

      expect(mockGithub.rest.issues.removeLabel).toHaveBeenCalledTimes(2);
      expect(mockCore.info).toHaveBeenCalledWith("Successfully removed label 'bug' from issue #123");
      expect(mockCore.info).toHaveBeenCalledWith("Successfully removed label 'enhancement' from issue #123");
      expect(mockCore.setOutput).toHaveBeenCalledWith("labels_removed", "bug\nenhancement");

      const summaryCall = mockCore.summary.addRaw.mock.calls.find(call =>
        call[0].includes("Successfully removed 2 label(s) from issue #123")
      );
      expect(summaryCall).toBeDefined();
      expect(summaryCall[0]).toContain("- `bug`");
      expect(summaryCall[0]).toContain("- `enhancement`");
    });

    it("should handle labels that don't exist on the issue", async () => {
      setAgentOutput({
        items: [
          {
            type: "remove_labels",
            labels: ["bug", "nonexistent"],
          },
        ],
      });
      process.env.GH_AW_LABELS_ALLOWED = "bug,nonexistent";

      mockGithub.rest.issues.removeLabel
        .mockResolvedValueOnce({}) // bug removed successfully
        .mockRejectedValueOnce(new Error("Label does not exist")); // nonexistent label

      // Execute the script
      await eval(`(async () => { ${removeLabelsScript} })()`);

      expect(mockCore.warning).toHaveBeenCalledWith("Label 'nonexistent' not found on issue #123, skipping");
      expect(mockCore.setOutput).toHaveBeenCalledWith("labels_removed", "bug");
    });

    it("should handle GitHub API errors", async () => {
      setAgentOutput({
        items: [
          {
            type: "remove_labels",
            labels: ["bug"],
          },
        ],
      });
      process.env.GH_AW_LABELS_ALLOWED = "bug,enhancement";

      const apiError = new Error("API error occurred");
      mockGithub.rest.issues.removeLabel.mockRejectedValue(apiError);

      // Execute the script
      await eval(`(async () => { ${removeLabelsScript} })()`);

      expect(mockCore.error).toHaveBeenCalledWith("Failed to remove label 'bug': API error occurred");
      expect(mockCore.setFailed).toHaveBeenCalledWith("Failed to remove 1 label(s): bug");
    });

    it("should successfully remove labels from pull request", async () => {
      setAgentOutput({
        items: [
          {
            type: "remove_labels",
            labels: ["bug"],
          },
        ],
      });
      process.env.GH_AW_LABELS_ALLOWED = "bug,enhancement";
      global.context.eventName = "pull_request";
      global.context.payload.pull_request = { number: 456 };
      delete global.context.payload.issue;

      mockGithub.rest.issues.removeLabel.mockResolvedValue({});

      // Execute the script
      await eval(`(async () => { ${removeLabelsScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith("Successfully removed label 'bug' from pull request #456");

      const summaryCall = mockCore.summary.addRaw.mock.calls.find(call =>
        call[0].includes("Successfully removed 1 label(s) from pull request #456")
      );
      expect(summaryCall).toBeDefined();
    });
  });

  describe("Output and logging", () => {
    it("should log agent output content length", async () => {
      setAgentOutput({
        items: [
          {
            type: "remove_labels",
            labels: ["bug", "enhancement"],
          },
        ],
      });
      process.env.GH_AW_LABELS_ALLOWED = "bug,enhancement";

      mockGithub.rest.issues.removeLabel.mockResolvedValue({});

      // Execute the script
      await eval(`(async () => { ${removeLabelsScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith("Agent output content length: 67");
    });

    it("should log requested labels", async () => {
      setAgentOutput({
        items: [
          {
            type: "remove_labels",
            labels: ["bug", "enhancement", "invalid"],
          },
        ],
      });
      process.env.GH_AW_LABELS_ALLOWED = "bug,enhancement";

      mockGithub.rest.issues.removeLabel.mockResolvedValue({});

      // Execute the script
      await eval(`(async () => { ${removeLabelsScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith(`Requested labels: ${JSON.stringify(["bug", "enhancement", "invalid"])}`);
    });
  });

  describe("Target configuration", () => {
    beforeEach(() => {
      // Reset environment variables
      delete process.env.GH_AW_LABELS_TARGET;
    });

    it("should use explicit issue number from target configuration", async () => {
      setAgentOutput({
        items: [
          {
            type: "remove_labels",
            labels: ["bug", "urgent"],
          },
        ],
      });
      process.env.GH_AW_LABELS_TARGET = "999";

      // Context doesn't matter when explicit issue number is provided
      global.context.eventName = "push";
      delete global.context.payload.issue;

      mockGithub.rest.issues.removeLabel.mockResolvedValue({});

      // Execute the script
      await eval(`(async () => { ${removeLabelsScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith("Labels target configuration: 999");
      expect(mockGithub.rest.issues.removeLabel).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 999,
        name: "bug",
      });
    });

    it("should use item_number from labels item when target is '*'", async () => {
      setAgentOutput({
        items: [
          {
            type: "remove_labels",
            labels: ["documentation"],
            item_number: 555,
          },
        ],
      });
      process.env.GH_AW_LABELS_TARGET = "*";

      // Context doesn't matter when issue_number is provided in the item
      global.context.eventName = "push";
      delete global.context.payload.issue;

      mockGithub.rest.issues.removeLabel.mockResolvedValue({});

      // Execute the script
      await eval(`(async () => { ${removeLabelsScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith("Labels target configuration: *");
      expect(mockGithub.rest.issues.removeLabel).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 555,
        name: "documentation",
      });
    });

    it("should fail when target is '*' but no item_number in labels item", async () => {
      setAgentOutput({
        items: [
          {
            type: "remove_labels",
            labels: ["bug"],
          },
        ],
      });
      process.env.GH_AW_LABELS_TARGET = "*";

      // Execute the script
      await eval(`(async () => { ${removeLabelsScript} })()`);

      expect(mockCore.setFailed).toHaveBeenCalledWith('Target is "*" but no item_number specified in labels item');
      expect(mockGithub.rest.issues.removeLabel).not.toHaveBeenCalled();
    });
  });
});
