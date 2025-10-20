import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
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
  setCancelled: vi.fn(),
  setError: vi.fn(),

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

const mockGithub = {
  request: vi.fn().mockResolvedValue({
    data: {
      id: 123456,
      html_url: "https://github.com/owner/repo/issues/1#issuecomment-123456",
    },
  }),
  graphql: vi.fn().mockResolvedValue({
    updateDiscussionComment: {
      comment: {
        id: "DC_kwDOABCDEF4ABCDEF",
        url: "https://github.com/owner/repo/discussions/1#discussioncomment-123456",
      },
    },
  }),
};

const mockContext = {
  repo: {
    owner: "testowner",
    repo: "testrepo",
  },
};

// Set up global variables
global.core = mockCore;
global.github = mockGithub;
global.context = mockContext;

describe("notify_comment_error.cjs", () => {
  let notifyCommentScript;
  let originalEnv;

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();

    // Store original environment
    originalEnv = {
      GITHUB_AW_COMMENT_ID: process.env.GITHUB_AW_COMMENT_ID,
      GITHUB_AW_COMMENT_REPO: process.env.GITHUB_AW_COMMENT_REPO,
      GITHUB_AW_RUN_URL: process.env.GITHUB_AW_RUN_URL,
      GITHUB_AW_WORKFLOW_NAME: process.env.GITHUB_AW_WORKFLOW_NAME,
      GITHUB_AW_AGENT_CONCLUSION: process.env.GITHUB_AW_AGENT_CONCLUSION,
    };

    // Read the script content
    const scriptPath = path.join(process.cwd(), "notify_comment_error.cjs");
    notifyCommentScript = fs.readFileSync(scriptPath, "utf8");
  });

  afterEach(() => {
    // Restore original environment
    Object.keys(originalEnv).forEach(key => {
      if (originalEnv[key] !== undefined) {
        process.env[key] = originalEnv[key];
      } else {
        delete process.env[key];
      }
    });
  });

  describe("when comment ID is not provided", () => {
    it("should skip comment update", async () => {
      delete process.env.GITHUB_AW_COMMENT_ID;
      process.env.GITHUB_AW_RUN_URL = "https://github.com/owner/repo/actions/runs/123";
      process.env.GITHUB_AW_WORKFLOW_NAME = "test-workflow";
      process.env.GITHUB_AW_AGENT_CONCLUSION = "failure";

      await eval(`(async () => { ${notifyCommentScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith("No comment ID found, skipping comment update");
      expect(mockGithub.request).not.toHaveBeenCalled();
      expect(mockGithub.graphql).not.toHaveBeenCalled();
    });
  });

  describe("when run URL is not provided", () => {
    it("should fail with error", async () => {
      process.env.GITHUB_AW_COMMENT_ID = "123456";
      delete process.env.GITHUB_AW_RUN_URL;
      process.env.GITHUB_AW_WORKFLOW_NAME = "test-workflow";
      process.env.GITHUB_AW_AGENT_CONCLUSION = "failure";

      await eval(`(async () => { ${notifyCommentScript} })()`);

      expect(mockCore.setFailed).toHaveBeenCalledWith("Run URL is required");
      expect(mockGithub.request).not.toHaveBeenCalled();
      expect(mockGithub.graphql).not.toHaveBeenCalled();
    });
  });

  describe("when updating an issue/PR comment", () => {
    it("should update with success message when agent succeeds", async () => {
      process.env.GITHUB_AW_COMMENT_ID = "123456";
      process.env.GITHUB_AW_RUN_URL = "https://github.com/owner/repo/actions/runs/123";
      process.env.GITHUB_AW_WORKFLOW_NAME = "test-workflow";
      process.env.GITHUB_AW_AGENT_CONCLUSION = "success";

      await eval(`(async () => { ${notifyCommentScript} })()`);

      expect(mockGithub.request).toHaveBeenCalledWith(
        "PATCH /repos/{owner}/{repo}/issues/comments/{comment_id}",
        expect.objectContaining({
          owner: "testowner",
          repo: "testrepo",
          comment_id: 123456,
          body: expect.stringMatching(/✅.*completed successfully/s),
        })
      );
      expect(mockCore.info).toHaveBeenCalledWith("Successfully updated comment");
    });

    it("should update with failure message when agent fails", async () => {
      process.env.GITHUB_AW_COMMENT_ID = "123456";
      process.env.GITHUB_AW_RUN_URL = "https://github.com/owner/repo/actions/runs/123";
      process.env.GITHUB_AW_WORKFLOW_NAME = "test-workflow";
      process.env.GITHUB_AW_AGENT_CONCLUSION = "failure";

      await eval(`(async () => { ${notifyCommentScript} })()`);

      expect(mockGithub.request).toHaveBeenCalledWith(
        "PATCH /repos/{owner}/{repo}/issues/comments/{comment_id}",
        expect.objectContaining({
          owner: "testowner",
          repo: "testrepo",
          comment_id: 123456,
          body: expect.stringMatching(/❌.*failed.*wasn't able to produce a result/s),
        })
      );
      expect(mockCore.info).toHaveBeenCalledWith("Successfully updated comment");
    });

    it("should update with cancelled message when agent is cancelled", async () => {
      process.env.GITHUB_AW_COMMENT_ID = "123456";
      process.env.GITHUB_AW_RUN_URL = "https://github.com/owner/repo/actions/runs/123";
      process.env.GITHUB_AW_WORKFLOW_NAME = "test-workflow";
      process.env.GITHUB_AW_AGENT_CONCLUSION = "cancelled";

      await eval(`(async () => { ${notifyCommentScript} })()`);

      expect(mockGithub.request).toHaveBeenCalledWith(
        "PATCH /repos/{owner}/{repo}/issues/comments/{comment_id}",
        expect.objectContaining({
          body: expect.stringMatching(/🚫.*was cancelled/s),
        })
      );
    });

    it("should update with timeout message when agent times out", async () => {
      process.env.GITHUB_AW_COMMENT_ID = "123456";
      process.env.GITHUB_AW_RUN_URL = "https://github.com/owner/repo/actions/runs/123";
      process.env.GITHUB_AW_WORKFLOW_NAME = "test-workflow";
      process.env.GITHUB_AW_AGENT_CONCLUSION = "timed_out";

      await eval(`(async () => { ${notifyCommentScript} })()`);

      expect(mockGithub.request).toHaveBeenCalledWith(
        "PATCH /repos/{owner}/{repo}/issues/comments/{comment_id}",
        expect.objectContaining({
          body: expect.stringMatching(/⏱️.*timed out/s),
        })
      );
    });

    it("should update with skipped message when agent is skipped", async () => {
      process.env.GITHUB_AW_COMMENT_ID = "123456";
      process.env.GITHUB_AW_RUN_URL = "https://github.com/owner/repo/actions/runs/123";
      process.env.GITHUB_AW_WORKFLOW_NAME = "test-workflow";
      process.env.GITHUB_AW_AGENT_CONCLUSION = "skipped";

      await eval(`(async () => { ${notifyCommentScript} })()`);

      expect(mockGithub.request).toHaveBeenCalledWith(
        "PATCH /repos/{owner}/{repo}/issues/comments/{comment_id}",
        expect.objectContaining({
          body: expect.stringMatching(/⏭️.*was skipped/s),
        })
      );
    });

    it("should use custom comment repo when provided", async () => {
      process.env.GITHUB_AW_COMMENT_ID = "123456";
      process.env.GITHUB_AW_COMMENT_REPO = "customowner/customrepo";
      process.env.GITHUB_AW_RUN_URL = "https://github.com/owner/repo/actions/runs/123";
      process.env.GITHUB_AW_WORKFLOW_NAME = "test-workflow";
      process.env.GITHUB_AW_AGENT_CONCLUSION = "success";

      await eval(`(async () => { ${notifyCommentScript} })()`);

      expect(mockGithub.request).toHaveBeenCalledWith(
        "PATCH /repos/{owner}/{repo}/issues/comments/{comment_id}",
        expect.objectContaining({
          owner: "customowner",
          repo: "customrepo",
        })
      );
    });
  });

  describe("when updating a discussion comment", () => {
    it("should use GraphQL API for discussion comments on success", async () => {
      process.env.GITHUB_AW_COMMENT_ID = "DC_kwDOABCDEF4ABCDEF";
      process.env.GITHUB_AW_RUN_URL = "https://github.com/owner/repo/actions/runs/123";
      process.env.GITHUB_AW_WORKFLOW_NAME = "test-workflow";
      process.env.GITHUB_AW_AGENT_CONCLUSION = "success";

      await eval(`(async () => { ${notifyCommentScript} })()`);

      expect(mockGithub.graphql).toHaveBeenCalledWith(
        expect.stringContaining("updateDiscussionComment"),
        expect.objectContaining({
          commentId: "DC_kwDOABCDEF4ABCDEF",
          body: expect.stringMatching(/✅.*completed successfully/s),
        })
      );
      expect(mockCore.info).toHaveBeenCalledWith("Successfully updated discussion comment");
    });

    it("should use GraphQL API for discussion comments on failure", async () => {
      process.env.GITHUB_AW_COMMENT_ID = "DC_kwDOABCDEF4ABCDEF";
      process.env.GITHUB_AW_RUN_URL = "https://github.com/owner/repo/actions/runs/123";
      process.env.GITHUB_AW_WORKFLOW_NAME = "test-workflow";
      process.env.GITHUB_AW_AGENT_CONCLUSION = "failure";

      await eval(`(async () => { ${notifyCommentScript} })()`);

      expect(mockGithub.graphql).toHaveBeenCalledWith(
        expect.stringContaining("updateDiscussionComment"),
        expect.objectContaining({
          commentId: "DC_kwDOABCDEF4ABCDEF",
          body: expect.stringMatching(/❌.*failed/s),
        })
      );
    });
  });

  describe("error handling", () => {
    it("should warn but not fail when comment update fails", async () => {
      process.env.GITHUB_AW_COMMENT_ID = "123456";
      process.env.GITHUB_AW_RUN_URL = "https://github.com/owner/repo/actions/runs/123";
      process.env.GITHUB_AW_WORKFLOW_NAME = "test-workflow";
      process.env.GITHUB_AW_AGENT_CONCLUSION = "success";

      mockGithub.request.mockRejectedValueOnce(new Error("API error"));

      await eval(`(async () => { ${notifyCommentScript} })()`);

      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Failed to update comment"));
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("API error"));
      // Should not fail the workflow
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });
  });
});
