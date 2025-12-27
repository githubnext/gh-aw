import { describe, it, expect, beforeEach, vi } from "vitest";

describe("assign_to_user.cjs", () => {
  let mockCore;
  let mockGithub;
  let mockContext;
  let mockProcessSafeOutput;
  let mockProcessItems;

  beforeEach(() => {
    // Mock core actions methods
    mockCore = {
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

    // Mock GitHub API
    mockGithub = {
      rest: {
        issues: {
          addAssignees: vi.fn(),
        },
      },
    };

    // Mock context
    mockContext = {
      eventName: "issues",
      actor: "testuser",
      repo: {
        owner: "testorg",
        repo: "testrepo",
      },
      payload: {
        issue: {
          number: 42,
        },
      },
    };

    // Mock safe_output_processor functions
    mockProcessSafeOutput = vi.fn();
    mockProcessItems = vi.fn();

    global.core = mockCore;
    global.github = mockGithub;
    global.context = mockContext;
  });

  afterEach(() => {
    delete global.core;
    delete global.github;
    delete global.context;
    delete process.env.GH_AW_ASSIGNEES_ALLOWED;
    delete process.env.GH_AW_ASSIGNEES_MAX_COUNT;
    delete process.env.GH_AW_TARGET_REPO_SLUG;
  });

  const runScript = async () => {
    const fs = await import("fs");
    const path = await import("path");
    const scriptPath = path.join(import.meta.dirname, "assign_to_user.cjs");
    const scriptContent = fs.readFileSync(scriptPath, "utf8");

    // Create a mock require function
    const mockRequire = modulePath => {
      if (modulePath === "./safe_output_processor.cjs") {
        return {
          processSafeOutput: mockProcessSafeOutput,
          processItems: mockProcessItems,
        };
      }
      throw new Error(`Module not found: ${modulePath}`);
    };

    // Remove the main() call/export at the end and execute
    const scriptWithoutMain = scriptContent.replace("module.exports = { main };", "");
    const scriptFunction = new Function("core", "github", "context", "process", "require", scriptWithoutMain + "\nreturn main();");
    await scriptFunction(mockCore, mockGithub, mockContext, process, mockRequire);
  };

  describe("basic functionality", () => {
    it("should return early if processSafeOutput returns not success", async () => {
      mockProcessSafeOutput.mockResolvedValue({ success: false });

      await runScript();

      expect(mockProcessSafeOutput).toHaveBeenCalled();
      expect(mockGithub.rest.issues.addAssignees).not.toHaveBeenCalled();
    });

    it("should handle singular assignee field", async () => {
      mockProcessSafeOutput.mockResolvedValue({
        success: true,
        item: { assignee: "user1" },
        config: { allowed: [], maxCount: 10 },
        targetResult: { number: 42 },
      });
      mockProcessItems.mockReturnValue(["user1"]);
      mockGithub.rest.issues.addAssignees.mockResolvedValue({});

      await runScript();

      expect(mockProcessItems).toHaveBeenCalledWith(["user1"], [], 10);
      expect(mockGithub.rest.issues.addAssignees).toHaveBeenCalledWith({
        owner: "testorg",
        repo: "testrepo",
        issue_number: 42,
        assignees: ["user1"],
      });
    });

    it("should handle plural assignees field", async () => {
      mockProcessSafeOutput.mockResolvedValue({
        success: true,
        item: { assignees: ["user1", "user2"] },
        config: { allowed: [], maxCount: 10 },
        targetResult: { number: 42 },
      });
      mockProcessItems.mockReturnValue(["user1", "user2"]);
      mockGithub.rest.issues.addAssignees.mockResolvedValue({});

      await runScript();

      expect(mockProcessItems).toHaveBeenCalledWith(["user1", "user2"], [], 10);
      expect(mockGithub.rest.issues.addAssignees).toHaveBeenCalledWith({
        owner: "testorg",
        repo: "testrepo",
        issue_number: 42,
        assignees: ["user1", "user2"],
      });
    });

    it("should handle empty assignees array", async () => {
      mockProcessSafeOutput.mockResolvedValue({
        success: true,
        item: { assignees: [] },
        config: { allowed: [], maxCount: 10 },
        targetResult: { number: 42 },
      });
      mockProcessItems.mockReturnValue([]);

      await runScript();

      expect(mockCore.info).toHaveBeenCalledWith("No assignees to add");
      expect(mockCore.setOutput).toHaveBeenCalledWith("assigned_users", "");
      expect(mockGithub.rest.issues.addAssignees).not.toHaveBeenCalled();
    });

    it("should handle missing assignee fields", async () => {
      mockProcessSafeOutput.mockResolvedValue({
        success: true,
        item: {},
        config: { allowed: [], maxCount: 10 },
        targetResult: { number: 42 },
      });
      mockProcessItems.mockReturnValue([]);

      await runScript();

      expect(mockProcessItems).toHaveBeenCalledWith([], [], 10);
      expect(mockCore.info).toHaveBeenCalledWith("No assignees to add");
    });
  });

  describe("target repository handling", () => {
    it("should use current repository by default", async () => {
      mockProcessSafeOutput.mockResolvedValue({
        success: true,
        item: { assignees: ["user1"] },
        config: { allowed: [], maxCount: 10 },
        targetResult: { number: 42 },
      });
      mockProcessItems.mockReturnValue(["user1"]);
      mockGithub.rest.issues.addAssignees.mockResolvedValue({});

      await runScript();

      expect(mockGithub.rest.issues.addAssignees).toHaveBeenCalledWith({
        owner: "testorg",
        repo: "testrepo",
        issue_number: 42,
        assignees: ["user1"],
      });
    });

    it("should use target repository from environment variable", async () => {
      process.env.GH_AW_TARGET_REPO_SLUG = "otherorg/otherrepo";
      mockProcessSafeOutput.mockResolvedValue({
        success: true,
        item: { assignees: ["user1"] },
        config: { allowed: [], maxCount: 10 },
        targetResult: { number: 42 },
      });
      mockProcessItems.mockReturnValue(["user1"]);
      mockGithub.rest.issues.addAssignees.mockResolvedValue({});

      await runScript();

      expect(mockCore.info).toHaveBeenCalledWith("Using target repository: otherorg/otherrepo");
      expect(mockGithub.rest.issues.addAssignees).toHaveBeenCalledWith({
        owner: "otherorg",
        repo: "otherrepo",
        issue_number: 42,
        assignees: ["user1"],
      });
    });

    it("should handle invalid target repository format", async () => {
      process.env.GH_AW_TARGET_REPO_SLUG = "invalid-format";
      mockProcessSafeOutput.mockResolvedValue({
        success: true,
        item: { assignees: ["user1"] },
        config: { allowed: [], maxCount: 10 },
        targetResult: { number: 42 },
      });
      mockProcessItems.mockReturnValue(["user1"]);
      mockGithub.rest.issues.addAssignees.mockResolvedValue({});

      await runScript();

      // Should fall back to current repository
      expect(mockGithub.rest.issues.addAssignees).toHaveBeenCalledWith({
        owner: "testorg",
        repo: "testrepo",
        issue_number: 42,
        assignees: ["user1"],
      });
    });

    it("should handle empty target repository environment variable", async () => {
      process.env.GH_AW_TARGET_REPO_SLUG = "   ";
      mockProcessSafeOutput.mockResolvedValue({
        success: true,
        item: { assignees: ["user1"] },
        config: { allowed: [], maxCount: 10 },
        targetResult: { number: 42 },
      });
      mockProcessItems.mockReturnValue(["user1"]);
      mockGithub.rest.issues.addAssignees.mockResolvedValue({});

      await runScript();

      // Should fall back to current repository
      expect(mockGithub.rest.issues.addAssignees).toHaveBeenCalledWith({
        owner: "testorg",
        repo: "testrepo",
        issue_number: 42,
        assignees: ["user1"],
      });
    });
  });

  describe("output generation", () => {
    it("should set outputs and generate summary on success", async () => {
      mockProcessSafeOutput.mockResolvedValue({
        success: true,
        item: { assignees: ["user1", "user2"] },
        config: { allowed: [], maxCount: 10 },
        targetResult: { number: 42 },
      });
      mockProcessItems.mockReturnValue(["user1", "user2"]);
      mockGithub.rest.issues.addAssignees.mockResolvedValue({});

      await runScript();

      expect(mockCore.setOutput).toHaveBeenCalledWith("assigned_users", "user1\nuser2");
      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("Successfully assigned 2 user(s) to issue #42"));
      expect(mockCore.summary.write).toHaveBeenCalled();
    });

    it("should generate summary with no assignees", async () => {
      mockProcessSafeOutput.mockResolvedValue({
        success: true,
        item: {},
        config: { allowed: [], maxCount: 10 },
        targetResult: { number: 42 },
      });
      mockProcessItems.mockReturnValue([]);

      await runScript();

      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("No users were assigned"));
    });
  });

  describe("error handling", () => {
    it("should handle API errors gracefully", async () => {
      mockProcessSafeOutput.mockResolvedValue({
        success: true,
        item: { assignees: ["user1"] },
        config: { allowed: [], maxCount: 10 },
        targetResult: { number: 42 },
      });
      mockProcessItems.mockReturnValue(["user1"]);
      const apiError = new Error("API rate limit exceeded");
      mockGithub.rest.issues.addAssignees.mockRejectedValue(apiError);

      await runScript();

      expect(mockCore.error).toHaveBeenCalledWith("Failed to assign users: API rate limit exceeded");
      expect(mockCore.setFailed).toHaveBeenCalledWith("Failed to assign users: API rate limit exceeded");
    });

    it("should handle non-Error failures", async () => {
      mockProcessSafeOutput.mockResolvedValue({
        success: true,
        item: { assignees: ["user1"] },
        config: { allowed: [], maxCount: 10 },
        targetResult: { number: 42 },
      });
      mockProcessItems.mockReturnValue(["user1"]);
      mockGithub.rest.issues.addAssignees.mockRejectedValue("String error");

      await runScript();

      expect(mockCore.error).toHaveBeenCalledWith("Failed to assign users: String error");
      expect(mockCore.setFailed).toHaveBeenCalledWith("Failed to assign users: String error");
    });

    it("should handle undefined config gracefully", async () => {
      mockProcessSafeOutput.mockResolvedValue({
        success: true,
        item: { assignees: ["user1"] },
        config: undefined,
        targetResult: { number: 42 },
      });

      await runScript();

      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Internal error"));
    });

    it("should handle undefined targetResult gracefully", async () => {
      mockProcessSafeOutput.mockResolvedValue({
        success: true,
        item: { assignees: ["user1"] },
        config: { allowed: [], maxCount: 10 },
        targetResult: undefined,
      });

      await runScript();

      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Internal error"));
    });
  });
});
