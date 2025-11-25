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
  exportVariable: vi.fn(),
  setSecret: vi.fn(),
  getInput: vi.fn(),
  getBooleanInput: vi.fn(),
  getMultilineInput: vi.fn(),
  getState: vi.fn(),
  saveState: vi.fn(),
  startGroup: vi.fn(),
  endGroup: vi.fn(),
  group: vi.fn(),
  addPath: vi.fn(),
  setCommandEcho: vi.fn(),
  isDebug: vi.fn().mockReturnValue(false),
  getIDToken: vi.fn(),
  toPlatformPath: vi.fn(),
  toPosixPath: vi.fn(),
  toWin32Path: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

const mockGithub = {
  rest: {
    issues: {
      get: vi.fn(),
    },
  },
  graphql: vi.fn(),
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

describe("link_issues.cjs", () => {
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
    delete process.env.GH_AW_TARGET_REPO_SLUG;

    // Reset context to default state
    global.context.eventName = "issues";
    global.context.payload.issue = { number: 123 };
    delete global.context.payload.pull_request;

    // Clean up temp file if it exists
    if (tempFilePath && fs.existsSync(tempFilePath)) {
      fs.unlinkSync(tempFilePath);
      tempFilePath = null;
    }
  });

  afterEach(() => {
    // Clean up temp file if it exists
    if (tempFilePath && fs.existsSync(tempFilePath)) {
      fs.unlinkSync(tempFilePath);
      tempFilePath = null;
    }
  });

  describe("parseIssueReference", () => {
    it("should handle missing agent output", async () => {
      delete process.env.GH_AW_AGENT_OUTPUT;

      await import("./link_issues.cjs");

      expect(mockCore.info).toHaveBeenCalledWith("No GH_AW_AGENT_OUTPUT environment variable found");
    });

    it("should handle empty agent output", async () => {
      setAgentOutput({ items: [] });

      await import("./link_issues.cjs");

      expect(mockCore.info).toHaveBeenCalledWith("No link_issues items found in agent output");
    });

    it("should process link_issues items with sub relationship", async () => {
      // Mock GraphQL to return node IDs
      mockGithub.graphql.mockImplementation(async (query, variables) => {
        if (query.includes("query")) {
          return {
            repository: {
              issue: {
                id: `issue-node-id-${variables.number}`,
              },
            },
          };
        }
        // Mock mutation success
        return {
          addSubIssue: {
            issue: { id: "parent-id", number: 10 },
            subIssue: { id: "child-id", number: 20 },
          },
        };
      });

      setAgentOutput({
        items: [
          {
            type: "link_issues",
            parent_issue: 10,
            child_issue: 20,
            relationship: "sub",
          },
        ],
      });

      await import("./link_issues.cjs");

      expect(mockCore.info).toHaveBeenCalledWith("Found 1 link_issues item(s)");
      expect(mockCore.info).toHaveBeenCalledWith("Processing sub relationship: parent #10 -> child #20");
      expect(mockCore.setOutput).toHaveBeenCalledWith("links_created", "1");
    });

    it("should process link_issues items with blocks relationship", async () => {
      // Mock GraphQL to return node IDs
      mockGithub.graphql.mockImplementation(async (query, variables) => {
        if (query.includes("query")) {
          return {
            repository: {
              issue: {
                id: `issue-node-id-${variables.number}`,
              },
            },
          };
        }
        // Mock mutation success for addIssueDependency
        return {
          addIssueDependency: {
            issue: { id: "blocked-id", number: 20 },
          },
        };
      });

      setAgentOutput({
        items: [
          {
            type: "link_issues",
            parent_issue: 10,
            child_issue: 20,
            relationship: "blocks",
          },
        ],
      });

      await import("./link_issues.cjs");

      expect(mockCore.info).toHaveBeenCalledWith("Found 1 link_issues item(s)");
      expect(mockCore.info).toHaveBeenCalledWith("Processing blocks relationship: parent #10 -> child #20");
      expect(mockCore.setOutput).toHaveBeenCalledWith("links_created", "1");
    });

    it("should handle staged mode", async () => {
      process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";

      setAgentOutput({
        items: [
          {
            type: "link_issues",
            parent_issue: 10,
            child_issue: 20,
            relationship: "sub",
          },
        ],
      });

      await import("./link_issues.cjs");

      expect(mockCore.summary.addRaw).toHaveBeenCalled();
      expect(mockCore.summary.write).toHaveBeenCalled();
      // Should not make any GraphQL calls in staged mode
      expect(mockGithub.graphql).not.toHaveBeenCalled();
    });

    it("should parse URL-style issue references", async () => {
      // Mock GraphQL to return node IDs
      mockGithub.graphql.mockImplementation(async (query, variables) => {
        if (query.includes("query")) {
          return {
            repository: {
              issue: {
                id: `issue-node-id-${variables.number}`,
              },
            },
          };
        }
        // Mock mutation success
        return {
          addSubIssue: {
            issue: { id: "parent-id", number: 42 },
            subIssue: { id: "child-id", number: 99 },
          },
        };
      });

      setAgentOutput({
        items: [
          {
            type: "link_issues",
            parent_issue: "https://github.com/owner/repo/issues/42",
            child_issue: 99,
            relationship: "sub",
          },
        ],
      });

      await import("./link_issues.cjs");

      expect(mockCore.info).toHaveBeenCalledWith("Found 1 link_issues item(s)");
      expect(mockGithub.graphql).toHaveBeenCalled();
    });

    it("should handle issue not found error", async () => {
      // Mock GraphQL to return null for parent issue
      mockGithub.graphql.mockImplementation(async (query, variables) => {
        if (variables.number === 10) {
          return {
            repository: {
              issue: null,
            },
          };
        }
        return {
          repository: {
            issue: {
              id: `issue-node-id-${variables.number}`,
            },
          },
        };
      });

      setAgentOutput({
        items: [
          {
            type: "link_issues",
            parent_issue: 10,
            child_issue: 20,
            relationship: "sub",
          },
        ],
      });

      await import("./link_issues.cjs");

      expect(mockCore.error).toHaveBeenCalledWith("Could not find parent issue #10 in testowner/testrepo");
      expect(mockCore.setOutput).toHaveBeenCalledWith("links_failed", "1");
    });

    it("should default relationship to sub when not provided", async () => {
      // Mock GraphQL to return node IDs
      mockGithub.graphql.mockImplementation(async (query, variables) => {
        if (query.includes("query")) {
          return {
            repository: {
              issue: {
                id: `issue-node-id-${variables.number}`,
              },
            },
          };
        }
        return {
          addSubIssue: {
            issue: { id: "parent-id", number: 10 },
            subIssue: { id: "child-id", number: 20 },
          },
        };
      });

      setAgentOutput({
        items: [
          {
            type: "link_issues",
            parent_issue: 10,
            child_issue: 20,
            // No relationship specified - should default to "sub"
          },
        ],
      });

      await import("./link_issues.cjs");

      expect(mockCore.info).toHaveBeenCalledWith("Processing sub relationship: parent #10 -> child #20");
    });
  });
});
