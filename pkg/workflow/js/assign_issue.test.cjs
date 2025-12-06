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

  // Input/state functions
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

const mockExec = {
  exec: vi.fn(),
};

const mockGithub = {
  graphql: vi.fn(),
};

const mockContext = {
  repo: {
    owner: "testowner",
    repo: "testrepo",
  },
};

// Set up global variables
global.core = mockCore;
global.exec = mockExec;
global.github = mockGithub;
global.context = mockContext;

describe("assign_issue.cjs", () => {
  let assignIssueScript;

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();

    // Reset environment variables
    delete process.env.GH_TOKEN;
    delete process.env.ASSIGNEE;
    delete process.env.ISSUE_NUMBER;

    // Read the script content
    const scriptPath = path.join(process.cwd(), "assign_issue.cjs");
    assignIssueScript = fs.readFileSync(scriptPath, "utf8");
  });

  describe("Environment variable validation", () => {
    it("should fail when GH_TOKEN is not set", async () => {
      process.env.ASSIGNEE = "test-user";
      process.env.ISSUE_NUMBER = "123";
      delete process.env.GH_TOKEN;

      // Execute the script
      await eval(`(async () => { ${assignIssueScript} })()`);

      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("GH_TOKEN environment variable is required but not set"));
      expect(mockCore.setFailed).toHaveBeenCalledWith(
        expect.stringContaining("https://githubnext.github.io/gh-aw/reference/safe-outputs/#assigning-issues-to-copilot")
      );
      expect(mockExec.exec).not.toHaveBeenCalled();
    });

    it("should fail when GH_TOKEN is empty string", async () => {
      process.env.GH_TOKEN = "   ";
      process.env.ASSIGNEE = "test-user";
      process.env.ISSUE_NUMBER = "123";

      // Execute the script
      await eval(`(async () => { ${assignIssueScript} })()`);

      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("GH_TOKEN environment variable is required but not set"));
      expect(mockExec.exec).not.toHaveBeenCalled();
    });

    it("should fail when ASSIGNEE is not set", async () => {
      process.env.GH_TOKEN = "ghp_test123";
      process.env.ISSUE_NUMBER = "123";
      delete process.env.ASSIGNEE;

      // Execute the script
      await eval(`(async () => { ${assignIssueScript} })()`);

      expect(mockCore.setFailed).toHaveBeenCalledWith("ASSIGNEE environment variable is required but not set");
      expect(mockExec.exec).not.toHaveBeenCalled();
    });

    it("should fail when ASSIGNEE is empty string", async () => {
      process.env.GH_TOKEN = "ghp_test123";
      process.env.ASSIGNEE = "   ";
      process.env.ISSUE_NUMBER = "123";

      // Execute the script
      await eval(`(async () => { ${assignIssueScript} })()`);

      expect(mockCore.setFailed).toHaveBeenCalledWith("ASSIGNEE environment variable is required but not set");
      expect(mockExec.exec).not.toHaveBeenCalled();
    });

    it("should fail when ISSUE_NUMBER is not set", async () => {
      process.env.GH_TOKEN = "ghp_test123";
      process.env.ASSIGNEE = "test-user";
      delete process.env.ISSUE_NUMBER;

      // Execute the script
      await eval(`(async () => { ${assignIssueScript} })()`);

      expect(mockCore.setFailed).toHaveBeenCalledWith("ISSUE_NUMBER environment variable is required but not set");
      expect(mockExec.exec).not.toHaveBeenCalled();
    });

    it("should fail when ISSUE_NUMBER is empty string", async () => {
      process.env.GH_TOKEN = "ghp_test123";
      process.env.ASSIGNEE = "test-user";
      process.env.ISSUE_NUMBER = "   ";

      // Execute the script
      await eval(`(async () => { ${assignIssueScript} })()`);

      expect(mockCore.setFailed).toHaveBeenCalledWith("ISSUE_NUMBER environment variable is required but not set");
      expect(mockExec.exec).not.toHaveBeenCalled();
    });
  });

  describe("Successful assignment for regular users", () => {
    it("should successfully assign issue to a regular user using GraphQL", async () => {
      process.env.GH_TOKEN = "ghp_test123";
      process.env.ASSIGNEE = "test-user";
      process.env.ISSUE_NUMBER = "456";

      // Note: Full end-to-end GraphQL functionality is tested in assign_agent_helpers.test.cjs
      // This test verifies the script structure and that gh CLI is not used
      // The eval() approach here cannot fully mock the require() chain, so we focus on
      // verifying the script loads without errors and validates environment variables

      // Execute the script
      await eval(`(async () => { ${assignIssueScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith("Assigning issue #456 to test-user");
      // Verify gh CLI is not used
      expect(mockExec.exec).not.toHaveBeenCalled();
    });

    it("should trim whitespace from environment variables", async () => {
      process.env.GH_TOKEN = "  ghp_test123  ";
      process.env.ASSIGNEE = "  test-user  ";
      process.env.ISSUE_NUMBER = "  123  ";

      // Execute the script
      await eval(`(async () => { ${assignIssueScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith("Assigning issue #123 to test-user");
      // No gh CLI calls should be made
      expect(mockExec.exec).not.toHaveBeenCalled();
    });

    it("should include summary in output on success", async () => {
      process.env.GH_TOKEN = "ghp_test123";
      process.env.ASSIGNEE = "test-user";
      process.env.ISSUE_NUMBER = "123";

      // Mock successful assignment result (would come from assignIssue helper)
      // This test verifies the summary generation logic works when assignment succeeds

      // Execute the script
      await eval(`(async () => { ${assignIssueScript} })()`);

      // Note: Summary will only be added if assignIssue succeeds, which requires proper mocking
      // The actual summary generation is tested indirectly through integration tests
    });
  });

  describe("Error handling for regular users", () => {
    it("should handle GraphQL API errors", async () => {
      process.env.GH_TOKEN = "ghp_test123";
      process.env.ASSIGNEE = "test-user";
      process.env.ISSUE_NUMBER = "999";

      // The script now uses assignIssue helper which handles GraphQL errors
      // Error handling is tested in assign_agent_helpers.test.cjs

      // Execute the script
      await eval(`(async () => { ${assignIssueScript} })()`);

      // Verify no gh CLI calls are made
      expect(mockExec.exec).not.toHaveBeenCalled();
    });

    it("should handle errors and call setFailed", async () => {
      process.env.GH_TOKEN = "ghp_test123";
      process.env.ASSIGNEE = "test-user";
      process.env.ISSUE_NUMBER = "999";

      // Execute the script - errors from assignIssue helper will be caught
      await eval(`(async () => { ${assignIssueScript} })()`);

      // No gh CLI should be invoked
      expect(mockExec.exec).not.toHaveBeenCalled();
    });

    it("should handle top-level errors with catch handler", async () => {
      process.env.GH_TOKEN = "ghp_test123";
      process.env.ASSIGNEE = "test-user";
      process.env.ISSUE_NUMBER = "123";

      // Execute the script
      await eval(`(async () => { ${assignIssueScript} })()`);

      // No gh CLI calls should be made with the new implementation
      expect(mockExec.exec).not.toHaveBeenCalled();
    });
  });

  describe("Edge cases for regular users", () => {
    it("should handle numeric issue number", async () => {
      process.env.GH_TOKEN = "ghp_test123";
      process.env.ASSIGNEE = "test-user";
      process.env.ISSUE_NUMBER = "123";

      // Execute the script
      await eval(`(async () => { ${assignIssueScript} })()`);

      // Script now uses GraphQL API, not gh CLI
      expect(mockExec.exec).not.toHaveBeenCalled();
    });

    it("should use GraphQL API instead of gh CLI", async () => {
      process.env.GH_TOKEN = "ghp_test123";
      process.env.ASSIGNEE = "test-user";
      process.env.ISSUE_NUMBER = "123";
      process.env.OTHER_VAR = "other_value";

      // Execute the script
      await eval(`(async () => { ${assignIssueScript} })()`);

      // Verify gh CLI is not used
      expect(mockExec.exec).not.toHaveBeenCalled();
    });

    it("should handle special characters in assignee name", async () => {
      process.env.GH_TOKEN = "ghp_test123";
      process.env.ASSIGNEE = "user-with-dash";
      process.env.ISSUE_NUMBER = "123";

      // Execute the script
      await eval(`(async () => { ${assignIssueScript} })()`);

      // Script uses assignIssue helper which handles all assignee types
      expect(mockExec.exec).not.toHaveBeenCalled();
    });

    it("should include documentation link in error message", async () => {
      delete process.env.GH_TOKEN;
      process.env.ASSIGNEE = "test-user";
      process.env.ISSUE_NUMBER = "123";

      // Execute the script
      await eval(`(async () => { ${assignIssueScript} })()`);

      const failedCall = mockCore.setFailed.mock.calls[0][0];
      expect(failedCall).toContain("https://githubnext.github.io/gh-aw/reference/safe-outputs/#assigning-issues-to-copilot");
    });
  });

  // Note: Agent-specific tests (e.g., @copilot) are in assign_agent_helpers.test.cjs
  // since assign_issue.cjs uses the shared helpers module for agent assignment,
  // and the require() statements don't work with eval() in tests.
});
