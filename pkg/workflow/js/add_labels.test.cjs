import { describe, it, expect, beforeEach, vi } from "vitest";
import fs from "fs";
import path from "path";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn(),
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
    const scriptPath = path.join(
      process.cwd(),
      "pkg/workflow/js/add_labels.cjs"
    );
    addLabelsScript = fs.readFileSync(scriptPath, "utf8");
  });

  describe("Environment variable validation", () => {
    it("should skip when no agent output is provided", async () => {
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";
      delete process.env.GITHUB_AW_AGENT_OUTPUT;

      const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(consoleSpy).toHaveBeenCalledWith(
        "No GITHUB_AW_AGENT_OUTPUT environment variable found"
      );
      expect(mockGithub.rest.issues.addLabels).not.toHaveBeenCalled();

      consoleSpy.mockRestore();
    });

    it("should skip when agent output is empty", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = "   ";
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";

      const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(consoleSpy).toHaveBeenCalledWith("Agent output content is empty");
      expect(mockGithub.rest.issues.addLabels).not.toHaveBeenCalled();

      consoleSpy.mockRestore();
    });

    it("should work when allowed labels are not provided (any labels allowed)", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add-issue-label",
            labels: ["bug", "enhancement", "custom-label"],
          },
        ],
      });
      delete process.env.GITHUB_AW_LABELS_ALLOWED;

      mockGithub.rest.issues.addLabels.mockResolvedValue({});

      const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(consoleSpy).toHaveBeenCalledWith(
        "No label restrictions - any labels are allowed"
      );
      expect(mockGithub.rest.issues.addLabels).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 123,
        labels: ["bug", "enhancement", "custom-label"],
      });

      consoleSpy.mockRestore();
    });

    it("should work when allowed labels list is empty (any labels allowed)", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add-issue-label",
            labels: ["bug", "enhancement", "custom-label"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "   ";

      mockGithub.rest.issues.addLabels.mockResolvedValue({});

      const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(consoleSpy).toHaveBeenCalledWith(
        "No label restrictions - any labels are allowed"
      );
      expect(mockGithub.rest.issues.addLabels).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 123,
        labels: ["bug", "enhancement", "custom-label"],
      });

      consoleSpy.mockRestore();
    });

    it("should enforce allowed labels when restrictions are set", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add-issue-label",
            labels: ["bug", "enhancement", "custom-label", "documentation"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";

      mockGithub.rest.issues.addLabels.mockResolvedValue({});

      const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(consoleSpy).toHaveBeenCalledWith("Allowed labels:", [
        "bug",
        "enhancement",
      ]);
      expect(mockGithub.rest.issues.addLabels).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 123,
        labels: ["bug", "enhancement"], // 'custom-label' and 'documentation' filtered out
      });

      consoleSpy.mockRestore();
    });

    it("should fail when max count is invalid", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add-issue-label",
            labels: ["bug", "enhancement"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";
      process.env.GITHUB_AW_LABELS_MAX_COUNT = "invalid";

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockCore.setFailed).toHaveBeenCalledWith(
        "Invalid max value: invalid. Must be a positive integer"
      );
      expect(mockGithub.rest.issues.addLabels).not.toHaveBeenCalled();
    });

    it("should fail when max count is zero", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add-issue-label",
            labels: ["bug", "enhancement"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";
      process.env.GITHUB_AW_LABELS_MAX_COUNT = "0";

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockCore.setFailed).toHaveBeenCalledWith(
        "Invalid max value: 0. Must be a positive integer"
      );
      expect(mockGithub.rest.issues.addLabels).not.toHaveBeenCalled();
    });

    it("should use default max count when not specified", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add-issue-label",
            labels: ["bug", "enhancement", "feature", "documentation"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED =
        "bug,enhancement,feature,documentation";
      delete process.env.GITHUB_AW_LABELS_MAX_COUNT;

      const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(consoleSpy).toHaveBeenCalledWith("Max count:", 3);
      expect(mockGithub.rest.issues.addLabels).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 123,
        labels: ["bug", "enhancement", "feature"], // Only first 3 due to default max count
      });

      consoleSpy.mockRestore();
    });
  });

  describe("Context validation", () => {
    it("should fail when not in issue or PR context", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add-issue-label",
            labels: ["bug", "enhancement"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";
      global.context.eventName = "push";

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockCore.setFailed).toHaveBeenCalledWith(
        "Not running in issue or pull request context, skipping label addition"
      );
      expect(mockGithub.rest.issues.addLabels).not.toHaveBeenCalled();
    });

    it("should work with issue_comment event", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add-issue-label",
            labels: ["bug"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";
      global.context.eventName = "issue_comment";

      const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockGithub.rest.issues.addLabels).toHaveBeenCalled();

      consoleSpy.mockRestore();
    });

    it("should work with pull_request event", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add-issue-label",
            labels: ["bug"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";
      global.context.eventName = "pull_request";
      global.context.payload.pull_request = { number: 456 };
      delete global.context.payload.issue;

      const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockGithub.rest.issues.addLabels).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 456,
        labels: ["bug"],
      });

      consoleSpy.mockRestore();
    });

    it("should work with pull_request_review event", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add-issue-label",
            labels: ["bug"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";
      global.context.eventName = "pull_request_review";
      global.context.payload.pull_request = { number: 789 };
      delete global.context.payload.issue;

      const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockGithub.rest.issues.addLabels).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 789,
        labels: ["bug"],
      });

      consoleSpy.mockRestore();
    });

    it("should fail when issue context detected but no issue in payload", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add-issue-label",
            labels: ["bug"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";
      global.context.eventName = "issues";
      delete global.context.payload.issue;

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockCore.setFailed).toHaveBeenCalledWith(
        "Issue context detected but no issue found in payload"
      );
      expect(mockGithub.rest.issues.addLabels).not.toHaveBeenCalled();
    });

    it("should fail when PR context detected but no PR in payload", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add-issue-label",
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

      expect(mockCore.setFailed).toHaveBeenCalledWith(
        "Pull request context detected but no pull request found in payload"
      );
      expect(mockGithub.rest.issues.addLabels).not.toHaveBeenCalled();
    });
  });

  describe("Label parsing and validation", () => {
    it("should parse labels from agent output and add valid ones", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add-issue-label",
            labels: ["bug", "enhancement", "documentation"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement,feature";

      const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockGithub.rest.issues.addLabels).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 123,
        labels: ["bug", "enhancement"], // 'documentation' not in allowed list
      });

      expect(mockCore.setOutput).toHaveBeenCalledWith(
        "labels_added",
        "bug\nenhancement"
      );
      expect(mockCore.summary.addRaw).toHaveBeenCalled();
      expect(mockCore.summary.write).toHaveBeenCalled();

      consoleSpy.mockRestore();
    });

    it("should skip empty lines in agent output", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add-issue-label",
            labels: ["bug", "enhancement"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";

      const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockGithub.rest.issues.addLabels).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 123,
        labels: ["bug", "enhancement"],
      });

      consoleSpy.mockRestore();
    });

    it("should fail when line starts with dash (removal indication)", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add-issue-label",
            labels: ["bug", "-enhancement"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockCore.setFailed).toHaveBeenCalledWith(
        "Label removal is not permitted. Found line starting with '-': -enhancement"
      );
      expect(mockGithub.rest.issues.addLabels).not.toHaveBeenCalled();
    });

    it("should remove duplicate labels", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add-issue-label",
            labels: ["bug", "enhancement", "bug", "enhancement"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";

      const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockGithub.rest.issues.addLabels).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 123,
        labels: ["bug", "enhancement"], // Duplicates removed
      });

      consoleSpy.mockRestore();
    });

    it("should enforce max count limit", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add-issue-label",
            labels: [
              "bug",
              "enhancement",
              "feature",
              "documentation",
              "question",
            ],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED =
        "bug,enhancement,feature,documentation,question";
      process.env.GITHUB_AW_LABELS_MAX_COUNT = "2";

      const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(consoleSpy).toHaveBeenCalledWith("too many labels, keep 2");
      expect(mockGithub.rest.issues.addLabels).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 123,
        labels: ["bug", "enhancement"], // Only first 2
      });

      consoleSpy.mockRestore();
    });

    it("should skip when no valid labels found", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add-issue-label",
            labels: ["invalid", "another-invalid"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";

      const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(consoleSpy).toHaveBeenCalledWith("No labels to add");
      expect(mockCore.setOutput).toHaveBeenCalledWith("labels_added", "");
      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(
        expect.stringContaining("No labels were added")
      );
      expect(mockGithub.rest.issues.addLabels).not.toHaveBeenCalled();

      consoleSpy.mockRestore();
    });
  });

  describe("GitHub API integration", () => {
    it("should successfully add labels to issue", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add-issue-label",
            labels: ["bug", "enhancement"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement,feature";

      mockGithub.rest.issues.addLabels.mockResolvedValue({});

      const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockGithub.rest.issues.addLabels).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 123,
        labels: ["bug", "enhancement"],
      });

      expect(consoleSpy).toHaveBeenCalledWith(
        "Successfully added 2 labels to issue #123"
      );
      expect(mockCore.setOutput).toHaveBeenCalledWith(
        "labels_added",
        "bug\nenhancement"
      );

      const summaryCall = mockCore.summary.addRaw.mock.calls.find(call =>
        call[0].includes("Successfully added 2 label(s) to issue #123")
      );
      expect(summaryCall).toBeDefined();
      expect(summaryCall[0]).toContain("- `bug`");
      expect(summaryCall[0]).toContain("- `enhancement`");

      consoleSpy.mockRestore();
    });

    it("should successfully add labels to pull request", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add-issue-label",
            labels: ["bug"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";
      global.context.eventName = "pull_request";
      global.context.payload.pull_request = { number: 456 };
      delete global.context.payload.issue;

      mockGithub.rest.issues.addLabels.mockResolvedValue({});

      const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(consoleSpy).toHaveBeenCalledWith(
        "Successfully added 1 labels to pull request #456"
      );

      const summaryCall = mockCore.summary.addRaw.mock.calls.find(call =>
        call[0].includes("Successfully added 1 label(s) to pull request #456")
      );
      expect(summaryCall).toBeDefined();

      consoleSpy.mockRestore();
    });

    it("should handle GitHub API errors", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add-issue-label",
            labels: ["bug"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";

      const apiError = new Error("Label does not exist");
      mockGithub.rest.issues.addLabels.mockRejectedValue(apiError);

      const consoleSpy = vi
        .spyOn(console, "error")
        .mockImplementation(() => {});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(consoleSpy).toHaveBeenCalledWith(
        "Failed to add labels:",
        "Label does not exist"
      );
      expect(mockCore.setFailed).toHaveBeenCalledWith(
        "Failed to add labels: Label does not exist"
      );

      consoleSpy.mockRestore();
    });

    it("should handle non-Error objects in catch block", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add-issue-label",
            labels: ["bug"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";

      const stringError = "Something went wrong";
      mockGithub.rest.issues.addLabels.mockRejectedValue(stringError);

      const consoleSpy = vi
        .spyOn(console, "error")
        .mockImplementation(() => {});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(consoleSpy).toHaveBeenCalledWith(
        "Failed to add labels:",
        "Something went wrong"
      );
      expect(mockCore.setFailed).toHaveBeenCalledWith(
        "Failed to add labels: Something went wrong"
      );

      consoleSpy.mockRestore();
    });
  });

  describe("Output and logging", () => {
    it("should log agent output content length", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add-issue-label",
            labels: ["bug", "enhancement"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";

      const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(consoleSpy).toHaveBeenCalledWith(
        "Agent output content length:",
        69
      );

      consoleSpy.mockRestore();
    });

    it("should log allowed labels and max count", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add-issue-label",
            labels: ["bug"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement,feature";
      process.env.GITHUB_AW_LABELS_MAX_COUNT = "5";

      const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(consoleSpy).toHaveBeenCalledWith("Allowed labels:", [
        "bug",
        "enhancement",
        "feature",
      ]);
      expect(consoleSpy).toHaveBeenCalledWith("Max count:", 5);

      consoleSpy.mockRestore();
    });

    it("should log requested labels", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add-issue-label",
            labels: ["bug", "enhancement", "invalid"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";

      const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(consoleSpy).toHaveBeenCalledWith("Requested labels:", [
        "bug",
        "enhancement",
        "invalid",
      ]);

      consoleSpy.mockRestore();
    });

    it("should log final labels being added", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add-issue-label",
            labels: ["bug", "enhancement"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";

      const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(consoleSpy).toHaveBeenCalledWith(
        "Adding 2 labels to issue #123:",
        ["bug", "enhancement"]
      );

      consoleSpy.mockRestore();
    });
  });

  describe("Edge cases", () => {
    it("should handle whitespace in allowed labels", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add-issue-label",
            labels: ["bug", "enhancement"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = " bug , enhancement , feature ";

      const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(consoleSpy).toHaveBeenCalledWith("Allowed labels:", [
        "bug",
        "enhancement",
        "feature",
      ]);
      expect(mockGithub.rest.issues.addLabels).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 123,
        labels: ["bug", "enhancement"],
      });

      consoleSpy.mockRestore();
    });

    it("should handle empty entries in allowed labels", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add-issue-label",
            labels: ["bug"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,,enhancement,";

      const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(consoleSpy).toHaveBeenCalledWith("Allowed labels:", [
        "bug",
        "enhancement",
      ]);

      consoleSpy.mockRestore();
    });

    it("should handle single label output", async () => {
      process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
        items: [
          {
            type: "add-issue-label",
            labels: ["bug"],
          },
        ],
      });
      process.env.GITHUB_AW_LABELS_ALLOWED = "bug,enhancement";

      mockGithub.rest.issues.addLabels.mockResolvedValue({});

      const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

      // Execute the script
      await eval(`(async () => { ${addLabelsScript} })()`);

      expect(mockGithub.rest.issues.addLabels).toHaveBeenCalledWith({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 123,
        labels: ["bug"],
      });

      expect(mockCore.setOutput).toHaveBeenCalledWith("labels_added", "bug");

      consoleSpy.mockRestore();
    });
  });
});
