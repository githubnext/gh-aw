import { describe, it, expect, beforeEach, vi } from "vitest";
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

describe("add_labels.cjs", () => {
  let addLabelsScript;

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();

    // Reset environment variables
    delete process.env.GITHUB_AW_AGENT_OUTPUT;
    delete process.env.GITHUB_AW_LABELS_ALLOWED;
    delete process.env.GITHUB_AW_LABELS_MAX_COUNT;

    // Reset context to default state
    global.context.eventName = "issues";
    global.context.payload.issue = { number: 123 };
    delete global.context.payload.pull_request;

    // Read the script content
    const scriptPath = path.join(process.cwd(), "add_labels.cjs");
    addLabelsScript = fs.readFileSync(scriptPath, "utf8");
  });

  describe("Environment variable validation", () => {
    it("should skip when no agent output is provided", async () => {
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";
      delete process.env.GITHUB_AW_AGENT_OUTPUT;

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith("No GITHUB_AW_AGENT_OUTPUT environment variable found");
      expect(mockGithub.rest.issues.addLabels).not.toHaveBeenCalled();
    });

    it("should skip when agent output is empty", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = "   ";
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith("Agent output content is empty");
      expect(mockGithub.rest.issues.addLabels).not.toHaveBeenCalled();
    });

    it("should work when allowed labels are not provided (any labels allowed)", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["bug", "enhancement", "custom-label"],
          },
        ],
      });
      delete process.env.GITHUB_AW_LABELS_ALLOWED;

      mockGithub.rest.issues.addLabels.mockResolvedValue({});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith("No label restrictions - any labels are allowed");
      expect(mockGithub.rest.issues.addLabels).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 123,
        labels: ["bug", "enhancement", "custom-label"],
      });
    });

    it("should work when allowed labels list is empty (any labels allowed)", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["bug", "enhancement", "custom-label"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "   ";

      mockGithub.rest.issues.addLabels.mockResolvedValue({});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith("No label restrictions - any labels are allowed");
      expect(mockGithub.rest.issues.addLabels).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 123,
        labels: ["bug", "enhancement", "custom-label"],
      });
    });

    it("should enforce allowed labels when restrictions are set", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["bug", "enhancement", "custom-label", "documentation"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";

      mockGithub.rest.issues.addLabels.mockResolvedValue({});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith(`Allowed labels: ${JSON.stringify(["bug", "enhancement"])}`);
      expect(mockGithub.rest.issues.addLabels).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 123,
        labels: ["bug", "enhancement"], // 'custom-label' and 'documentation' filtered out
      });
    });

    it("should fail when max count is invalid", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["bug", "enhancement"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";
      process.env.GITHUB_AW_LABELS_MAX_COUNT = "invalid";

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockCore.setFailed).toHaveBeenCalledWith("Invalid max value: invalid. Must be a positive integer");
      expect(mockGithub.rest.issues.addLabels).not.toHaveBeenCalled();
    });

    it("should fail when max count is zero", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["bug", "enhancement"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";
      process.env.GITHUB_AW_LABELS_MAX_COUNT = "0";

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockCore.setFailed).toHaveBeenCalledWith("Invalid max value: 0. Must be a positive integer");
      expect(mockGithub.rest.issues.addLabels).not.toHaveBeenCalled();
    });

    it("should use default max count when not specified", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["bug", "enhancement", "feature", "documentation"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement,feature,documentation";
      delete process.env.GITHUB_AW_LABELS_MAX_COUNT;

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith("Max count: 3");
      expect(mockGithub.rest.issues.addLabels).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 123,
        labels: ["bug", "enhancement", "feature"], // Only first 3 due to default max count
      });
    });
  });

  describe("Context validation", () => {
    it("should skip when not in issue or PR context (with default target)", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["bug", "enhancement"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";
      global.context.eventName = "push";

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith(
        'Target is "triggering" but not running in issue or pull request context, skipping label addition'
      );
      expect(mockGithub.rest.issues.addLabels).not.toHaveBeenCalled();
    });

    it("should work with issue_comment event", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["bug"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";
      global.context.eventName = "issue_comment";

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockGithub.rest.issues.addLabels).toHaveBeenCalled();
    });

    it("should work with pull_request event", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["bug"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";
      global.context.eventName = "pull_request";
      global.context.payload.pull_request = { number: 456 };
      delete global.context.payload.issue;

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockGithub.rest.issues.addLabels).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 456,
        labels: ["bug"],
      });
    });

    it("should work with pull_request_review event", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["bug"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";
      global.context.eventName = "pull_request_review";
      global.context.payload.pull_request = { number: 789 };
      delete global.context.payload.issue;

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockGithub.rest.issues.addLabels).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 789,
        labels: ["bug"],
      });
    });

    it("should fail when issue context detected but no issue in payload", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["bug"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";
      global.context.eventName = "issues";
      delete global.context.payload.issue;

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockCore.setFailed).toHaveBeenCalledWith("Issue context detected but no issue found in payload");
      expect(mockGithub.rest.issues.addLabels).not.toHaveBeenCalled();
    });

    it("should fail when PR context detected but no PR in payload", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["bug"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";
      global.context.eventName = "pull_request";
      delete global.context.payload.issue;
      delete global.context.payload.pull_request;

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockCore.setFailed).toHaveBeenCalledWith("Pull request context detected but no pull request found in payload");
      expect(mockGithub.rest.issues.addLabels).not.toHaveBeenCalled();
    });
  });

  describe("Label parsing and validation", () => {
    it("should parse labels from agent output and add valid ones", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["bug", "enhancement", "documentation"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement,feature";

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockGithub.rest.issues.addLabels).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 123,
        labels: ["bug", "enhancement"], // 'documentation' not in allowed list
      });

      expect(mockCore.setOutput).toHaveBeenCalledWith("labels_added", "bug\nenhancement");
      expect(mockCore.summary.addRaw).toHaveBeenCalled();
      expect(mockCore.summary.write).toHaveBeenCalled();
    });

    it("should skip empty lines in agent output", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["bug", "enhancement"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockGithub.rest.issues.addLabels).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 123,
        labels: ["bug", "enhancement"],
      });
    });

    it("should fail when line starts with dash (removal indication)", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["bug", "-enhancement"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockCore.setFailed).toHaveBeenCalledWith("Label removal is not permitted. Found line starting with '-': -enhancement");
      expect(mockGithub.rest.issues.addLabels).not.toHaveBeenCalled();
    });

    it("should remove duplicate labels", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["bug", "enhancement", "bug", "enhancement"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockGithub.rest.issues.addLabels).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 123,
        labels: ["bug", "enhancement"], // Duplicates removed
      });
    });

    it("should enforce max count limit", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["bug", "enhancement", "feature", "documentation", "question"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement,feature,documentation,question";
      process.env.GITHUB_AW_LABELS_MAX_COUNT = "2";

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith("too many labels, keep 2");
      expect(mockGithub.rest.issues.addLabels).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 123,
        labels: ["bug", "enhancement"], // Only first 2
      });
    });

    it("should skip when no valid labels found", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["invalid", "another-invalid"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith("No labels to add");
      expect(mockCore.setOutput).toHaveBeenCalledWith("labels_added", "");
      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("No labels were added"));
      expect(mockGithub.rest.issues.addLabels).not.toHaveBeenCalled();
    });
  });

  describe("GitHub API integration", () => {
    it("should successfully add labels to issue", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["bug", "enhancement"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement,feature";

      mockGithub.rest.issues.addLabels.mockResolvedValue({});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockGithub.rest.issues.addLabels).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 123,
        labels: ["bug", "enhancement"],
      });

      expect(mockCore.info).toHaveBeenCalledWith("Successfully added 2 labels to issue #123");
      expect(mockCore.setOutput).toHaveBeenCalledWith("labels_added", "bug\nenhancement");

      const summaryCall = mockCore.summary.addRaw.mock.calls.find(call => call[0].includes("Successfully added 2 label(s) to issue #123"));
      expect(summaryCall).toBeDefined();
      expect(summaryCall[0]).toContain("- `bug`");
      expect(summaryCall[0]).toContain("- `enhancement`");
    });

    it("should successfully add labels to pull request", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["bug"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";
      global.context.eventName = "pull_request";
      global.context.payload.pull_request = { number: 456 };
      delete global.context.payload.issue;

      mockGithub.rest.issues.addLabels.mockResolvedValue({});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith("Successfully added 1 labels to pull request #456");

      const summaryCall = mockCore.summary.addRaw.mock.calls.find(call =>
        call[0].includes("Successfully added 1 label(s) to pull request #456")
      );
      expect(summaryCall).toBeDefined();
    });

    it("should handle GitHub API errors", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["bug"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";

      const apiError = new Error("Label does not exist");
      mockGithub.rest.issues.addLabels.mockRejectedValue(apiError);

      const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockCore.error).toHaveBeenCalledWith("Failed to add labels: Label does not exist");
      expect(mockCore.setFailed).toHaveBeenCalledWith("Failed to add labels: Label does not exist");
    });

    it("should handle non-Error objects in catch block", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["bug"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";

      const stringError = "Something went wrong";
      mockGithub.rest.issues.addLabels.mockRejectedValue(stringError);

      const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockCore.error).toHaveBeenCalledWith("Failed to add labels: Something went wrong");
      expect(mockCore.setFailed).toHaveBeenCalledWith("Failed to add labels: Something went wrong");
    });
  });

  describe("Output and logging", () => {
    it("should log agent output content length", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["bug", "enhancement"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith("Agent output content length: 64");
    });

    it("should log allowed labels and max count", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["bug"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement,feature";
      process.env.GITHUB_AW_LABELS_MAX_COUNT = "5";

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith(`Allowed labels: ${JSON.stringify(["bug", "enhancement", "feature"])}`);
      expect(mockCore.info).toHaveBeenCalledWith("Max count: 5");
    });

    it("should log requested labels", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["bug", "enhancement", "invalid"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith(`Requested labels: ${JSON.stringify(["bug", "enhancement", "invalid"])}`);
    });

    it("should log final labels being added", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["bug", "enhancement"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith(`Adding 2 labels to issue #123: ${JSON.stringify(["bug", "enhancement"])}`);
    });
  });

  describe("Edge cases", () => {
    it("should handle whitespace in allowed labels", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["bug", "enhancement"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = " bug , enhancement , feature ";

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith(`Allowed labels: ${JSON.stringify(["bug", "enhancement", "feature"])}`);
      expect(mockGithub.rest.issues.addLabels).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 123,
        labels: ["bug", "enhancement"],
      });
    });

    it("should handle empty entries in allowed labels", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["bug"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,,enhancement,";

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith(`Allowed labels: ${JSON.stringify(["bug", "enhancement"])}`);
    });

    it("should handle single label output", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["bug"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";

      mockGithub.rest.issues.addLabels.mockResolvedValue({});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockGithub.rest.issues.addLabels).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 123,
        labels: ["bug"],
      });

      expect(mockCore.setOutput).toHaveBeenCalledWith("labels_added", "bug");
    });

    it("should handle duplicate labels by removing duplicates", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["bug", "enhancement", "bug", "automation", "enhancement"],
          },
        ],
      });

      mockGithub.rest.issues.addLabels.mockResolvedValue({});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockGithub.rest.issues.addLabels).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 123,
        labels: ["bug", "enhancement", "automation"],
      });
    });

    it("should sanitize labels by removing problematic characters", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["bug<script>", "enhancement@user", "automation&test", "normal-label"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_MAX_COUNT = "5"; // Allow more than 4 labels

      mockGithub.rest.issues.addLabels.mockResolvedValue({});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      const callArgs = mockGithub.rest.issues.addLabels.mock.calls[0][0];
      // Should sanitize problematic characters but keep valid labels
      expect(callArgs.labels).toContain("bugscript");
      expect(callArgs.labels).toContain("enhancement@user");
      expect(callArgs.labels).toContain("automationtest");
      expect(callArgs.labels).toContain("normal-label");
      expect(callArgs.labels).toHaveLength(4);
    });

    it("should limit label length to 64 characters", async () => {
      const longLabel = "a".repeat(100);
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: [longLabel, "short"],
          },
        ],
      });

      mockGithub.rest.issues.addLabels.mockResolvedValue({});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      const callArgs = mockGithub.rest.issues.addLabels.mock.calls[0][0];
      expect(callArgs.labels[0]).toHaveLength(64);
      expect(callArgs.labels[0]).toBe("a".repeat(64));
      expect(callArgs.labels[1]).toBe("short");
    });

    it("should remove empty and whitespace-only labels", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["bug", "", "   ", "enhancement", null, undefined, 0, false],
          },
        ],
      });

      mockGithub.rest.issues.addLabels.mockResolvedValue({});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockGithub.rest.issues.addLabels).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 123,
        labels: ["bug", "enhancement"],
      });
    });
  });

  describe("Target configuration", () => {
    beforeEach(() => {
      // Reset environment variables
      delete process.env.GITHUB_AW_LABELS_TARGET;
    });

    it("should use triggering issue when target is not specified (default behavior)", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["bug"],
          },
        ],
      });

      global.context.eventName = "issues";
      global.context.payload.issue = { number: 456 };

      mockGithub.rest.issues.addLabels.mockResolvedValue({});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith("Labels target configuration: triggering");
      expect(mockGithub.rest.issues.addLabels).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 456,
        labels: ["bug"],
      });
    });

    it("should use triggering issue when target is explicitly set to 'triggering'", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["enhancement"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_TARGET = "triggering";

      global.context.eventName = "issues";
      global.context.payload.issue = { number: 789 };

      mockGithub.rest.issues.addLabels.mockResolvedValue({});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith("Labels target configuration: triggering");
      expect(mockGithub.rest.issues.addLabels).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 789,
        labels: ["enhancement"],
      });
    });

    it("should skip when target is 'triggering' but not in issue context", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["bug"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_TARGET = "triggering";

      // Set context to something other than issues or PR
      global.context.eventName = "push";
      delete global.context.payload.issue;
      delete global.context.payload.pull_request;

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith(
        'Target is "triggering" but not running in issue or pull request context, skipping label addition'
      );
      expect(mockGithub.rest.issues.addLabels).not.toHaveBeenCalled();
    });

    it("should use explicit issue number from target configuration", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["bug", "urgent"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_TARGET = "999";

      // Context doesn't matter when explicit issue number is provided
      global.context.eventName = "push";
      delete global.context.payload.issue;

      mockGithub.rest.issues.addLabels.mockResolvedValue({});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith("Labels target configuration: 999");
      expect(mockGithub.rest.issues.addLabels).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 999,
        labels: ["bug", "urgent"],
      });
    });

    it("should use item_number from labels item when target is '*'", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["documentation"],
            item_number: 555,
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_TARGET = "*";

      // Context doesn't matter when issue_number is provided in the item
      global.context.eventName = "push";
      delete global.context.payload.issue;

      mockGithub.rest.issues.addLabels.mockResolvedValue({});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith("Labels target configuration: *");
      expect(mockGithub.rest.issues.addLabels).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 555,
        labels: ["documentation"],
      });
    });

    it("should fail when target is '*' but no item_number in labels item", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["bug"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_TARGET = "*";

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockCore.setFailed).toHaveBeenCalledWith('Target is "*" but no item_number specified in labels item');
      expect(mockGithub.rest.issues.addLabels).not.toHaveBeenCalled();
    });

    it("should fail when target has invalid issue number", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["bug"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_TARGET = "invalid";

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockCore.setFailed).toHaveBeenCalledWith("Invalid issue number in target configuration: invalid");
      expect(mockGithub.rest.issues.addLabels).not.toHaveBeenCalled();
    });

    it("should fail when target is '*' and issue_number in item is invalid", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add_labels",
            labels: ["bug"],
            item_number: -5,
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_TARGET = "*";

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockCore.setFailed).toHaveBeenCalledWith("Invalid item_number specified: -5");
      expect(mockGithub.rest.issues.addLabels).not.toHaveBeenCalled();
    });
  });
});
