// @ts-check
import { describe, it, expect, beforeEach, vi } from "vitest";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
};

const mockGithub = {
  request: vi.fn(),
  graphql: vi.fn(),
};

const mockContext = {
  eventName: "issues",
  runId: 12345,
  repo: {
    owner: "testowner",
    repo: "testrepo",
  },
  payload: {
    issue: {
      number: 123,
    },
    repository: {
      html_url: "https://github.com/testowner/testrepo",
    },
  },
};

// Set up global mocks before importing the module
global.core = mockCore;
global.github = mockGithub;
global.context = mockContext;

describe("add_workflow_run_comment", () => {
  beforeEach(() => {
    // Reset all mocks before each test
    vi.clearAllMocks();
    vi.resetModules();

    // Reset environment variables
    delete process.env.GH_AW_WORKFLOW_NAME;
    delete process.env.GITHUB_WORKFLOW;
    delete process.env.GH_AW_TRACKER_ID;
    delete process.env.GH_AW_LOCK_FOR_AGENT;
    delete process.env.GITHUB_SERVER_URL;

    // Reset context to default
    global.context = {
      eventName: "issues",
      runId: 12345,
      repo: {
        owner: "testowner",
        repo: "testrepo",
      },
      payload: {
        issue: {
          number: 123,
        },
        repository: {
          html_url: "https://github.com/testowner/testrepo",
        },
      },
    };

    // Reset default mock implementations
    mockGithub.request.mockResolvedValue({
      data: {
        id: 67890,
        html_url: "https://github.com/testowner/testrepo/issues/123#issuecomment-67890",
      },
    });

    mockGithub.graphql.mockResolvedValue({
      repository: {
        discussion: {
          id: "D_kwDOTest123",
        },
      },
      addDiscussionComment: {
        comment: {
          id: "DC_kwDOTest456",
          url: "https://github.com/testowner/testrepo/discussions/10#discussioncomment-456",
        },
      },
    });
  });

  // Helper function to run the script
  async function runScript() {
    const { main } = await import("./add_workflow_run_comment.cjs?" + Date.now());
    await main();
  }

  // Helper function to run addCommentWithWorkflowLink
  async function runAddCommentWithWorkflowLink(endpoint, runUrl, eventName) {
    const { addCommentWithWorkflowLink } = await import("./add_workflow_run_comment.cjs?" + Date.now());
    await addCommentWithWorkflowLink(endpoint, runUrl, eventName);
  }

  describe("main() - issues event", () => {
    it("should create comment on an issue", async () => {
      global.context = {
        eventName: "issues",
        runId: 12345,
        repo: { owner: "testowner", repo: "testrepo" },
        payload: {
          issue: { number: 456 },
          repository: { html_url: "https://github.com/testowner/testrepo" },
        },
      };

      await runScript();

      expect(mockGithub.request).toHaveBeenCalledWith(
        expect.stringContaining("POST /repos/testowner/testrepo/issues/456/comments"),
        expect.objectContaining({
          body: expect.stringContaining("https://github.com/testowner/testrepo/actions/runs/12345"),
        })
      );
      expect(mockCore.setOutput).toHaveBeenCalledWith("comment-id", "67890");
      expect(mockCore.setOutput).toHaveBeenCalledWith("comment-url", expect.stringContaining("issuecomment-67890"));
      expect(mockCore.setOutput).toHaveBeenCalledWith("comment-repo", "testowner/testrepo");
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should fail when issue number is missing", async () => {
      global.context = {
        eventName: "issues",
        runId: 12345,
        repo: { owner: "testowner", repo: "testrepo" },
        payload: {},
      };

      await runScript();

      expect(mockCore.setFailed).toHaveBeenCalledWith("Issue number not found in event payload");
      expect(mockGithub.request).not.toHaveBeenCalled();
    });
  });

  describe("main() - issue_comment event", () => {
    it("should create comment on the issue", async () => {
      global.context = {
        eventName: "issue_comment",
        runId: 12345,
        repo: { owner: "testowner", repo: "testrepo" },
        payload: {
          issue: { number: 789 },
          repository: { html_url: "https://github.com/testowner/testrepo" },
        },
      };

      await runScript();

      expect(mockGithub.request).toHaveBeenCalledWith(
        expect.stringContaining("POST /repos/testowner/testrepo/issues/789/comments"),
        expect.objectContaining({
          body: expect.any(String),
        })
      );
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });
  });

  describe("main() - pull_request event", () => {
    it("should create comment on a pull request", async () => {
      global.context = {
        eventName: "pull_request",
        runId: 12345,
        repo: { owner: "testowner", repo: "testrepo" },
        payload: {
          pull_request: { number: 101 },
          repository: { html_url: "https://github.com/testowner/testrepo" },
        },
      };

      await runScript();

      expect(mockGithub.request).toHaveBeenCalledWith(
        expect.stringContaining("POST /repos/testowner/testrepo/issues/101/comments"),
        expect.objectContaining({
          body: expect.any(String),
        })
      );
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should fail when PR number is missing", async () => {
      global.context = {
        eventName: "pull_request",
        runId: 12345,
        repo: { owner: "testowner", repo: "testrepo" },
        payload: {},
      };

      await runScript();

      expect(mockCore.setFailed).toHaveBeenCalledWith("Pull request number not found in event payload");
      expect(mockGithub.request).not.toHaveBeenCalled();
    });
  });

  describe("main() - pull_request_review_comment event", () => {
    it("should create comment on the pull request", async () => {
      global.context = {
        eventName: "pull_request_review_comment",
        runId: 12345,
        repo: { owner: "testowner", repo: "testrepo" },
        payload: {
          pull_request: { number: 202 },
          repository: { html_url: "https://github.com/testowner/testrepo" },
        },
      };

      await runScript();

      expect(mockGithub.request).toHaveBeenCalledWith(
        expect.stringContaining("POST /repos/testowner/testrepo/issues/202/comments"),
        expect.objectContaining({
          body: expect.any(String),
        })
      );
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });
  });

  describe("main() - discussion event", () => {
    it("should create GraphQL comment on a discussion", async () => {
      global.context = {
        eventName: "discussion",
        runId: 12345,
        repo: { owner: "testowner", repo: "testrepo" },
        payload: {
          discussion: { number: 10 },
          repository: { html_url: "https://github.com/testowner/testrepo" },
        },
      };

      await runScript();

      expect(mockGithub.graphql).toHaveBeenCalled();
      expect(mockCore.setOutput).toHaveBeenCalledWith("comment-id", "DC_kwDOTest456");
      expect(mockCore.setOutput).toHaveBeenCalledWith("comment-url", expect.stringContaining("discussioncomment-456"));
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should fail when discussion number is missing", async () => {
      global.context = {
        eventName: "discussion",
        runId: 12345,
        repo: { owner: "testowner", repo: "testrepo" },
        payload: {},
      };

      await runScript();

      expect(mockCore.setFailed).toHaveBeenCalledWith("Discussion number not found in event payload");
    });
  });

  describe("main() - discussion_comment event", () => {
    it("should create threaded comment on a discussion with replyToId", async () => {
      global.context = {
        eventName: "discussion_comment",
        runId: 12345,
        repo: { owner: "testowner", repo: "testrepo" },
        payload: {
          discussion: { number: 15 },
          comment: { id: 999, node_id: "DC_kwDOOriginal" },
          repository: { html_url: "https://github.com/testowner/testrepo" },
        },
      };

      await runScript();

      expect(mockGithub.graphql).toHaveBeenCalled();
      const graphqlCalls = mockGithub.graphql.mock.calls;
      // Find the mutation call (second call)
      const mutationCall = graphqlCalls.find(call => call[0].includes("addDiscussionComment"));
      expect(mutationCall).toBeDefined();
      expect(mutationCall[1]).toMatchObject({
        replyToId: "DC_kwDOOriginal",
      });
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should fail when discussion or comment fields are missing", async () => {
      global.context = {
        eventName: "discussion_comment",
        runId: 12345,
        repo: { owner: "testowner", repo: "testrepo" },
        payload: {
          discussion: { number: 15 },
          // Missing comment field
        },
      };

      await runScript();

      expect(mockCore.setFailed).toHaveBeenCalledWith("Discussion or comment information not found in event payload");
    });
  });

  describe("main() - unsupported event types", () => {
    it("should fail for unsupported event type", async () => {
      global.context = {
        eventName: "unsupported_event",
        runId: 12345,
        repo: { owner: "testowner", repo: "testrepo" },
        payload: {},
      };

      await runScript();

      expect(mockCore.setFailed).toHaveBeenCalledWith("Unsupported event type: unsupported_event");
      expect(mockGithub.request).not.toHaveBeenCalled();
    });
  });

  describe("main() - API errors", () => {
    it("should warn but not fail on API error", async () => {
      mockGithub.request.mockRejectedValueOnce(new Error("API Error"));

      global.context = {
        eventName: "issues",
        runId: 12345,
        repo: { owner: "testowner", repo: "testrepo" },
        payload: {
          issue: { number: 456 },
          repository: { html_url: "https://github.com/testowner/testrepo" },
        },
      };

      await runScript();

      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Failed to create comment"));
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Failed to create comment with workflow link"));
      // Should NOT call setFailed - errors should only warn
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });
  });

  describe("addCommentWithWorkflowLink() - workflow-id marker", () => {
    it("should include workflow-id marker when GITHUB_WORKFLOW is set", async () => {
      process.env.GITHUB_WORKFLOW = "test-workflow.yml";

      await runAddCommentWithWorkflowLink("/repos/testowner/testrepo/issues/123/comments", "https://github.com/testowner/testrepo/actions/runs/12345", "issues");

      expect(mockGithub.request).toHaveBeenCalledWith(
        expect.stringContaining("POST"),
        expect.objectContaining({
          body: expect.stringContaining("<!-- gh-aw-workflow-id: test-workflow.yml -->"),
        })
      );
    });

    it("should include tracker-id marker when GH_AW_TRACKER_ID is set", async () => {
      process.env.GH_AW_TRACKER_ID = "tracker-123";

      await runAddCommentWithWorkflowLink("/repos/testowner/testrepo/issues/123/comments", "https://github.com/testowner/testrepo/actions/runs/12345", "issues");

      expect(mockGithub.request).toHaveBeenCalledWith(
        expect.stringContaining("POST"),
        expect.objectContaining({
          body: expect.stringContaining("<!-- gh-aw-tracker-id: tracker-123 -->"),
        })
      );
    });

    it("should always include reaction comment type marker", async () => {
      await runAddCommentWithWorkflowLink("/repos/testowner/testrepo/issues/123/comments", "https://github.com/testowner/testrepo/actions/runs/12345", "issues");

      expect(mockGithub.request).toHaveBeenCalledWith(
        expect.stringContaining("POST"),
        expect.objectContaining({
          body: expect.stringContaining("<!-- gh-aw-comment-type: reaction -->"),
        })
      );
    });
  });

  describe("addCommentWithWorkflowLink() - lock notice", () => {
    it("should add lock notice for issues event when GH_AW_LOCK_FOR_AGENT=true", async () => {
      process.env.GH_AW_LOCK_FOR_AGENT = "true";

      await runAddCommentWithWorkflowLink("/repos/testowner/testrepo/issues/123/comments", "https://github.com/testowner/testrepo/actions/runs/12345", "issues");

      expect(mockGithub.request).toHaveBeenCalledWith(
        expect.stringContaining("POST"),
        expect.objectContaining({
          body: expect.stringContaining("🔒 This issue has been locked while the workflow is running to prevent concurrent modifications."),
        })
      );
    });

    it("should not add lock notice for pull_request events", async () => {
      process.env.GH_AW_LOCK_FOR_AGENT = "true";

      await runAddCommentWithWorkflowLink("/repos/testowner/testrepo/issues/101/comments", "https://github.com/testowner/testrepo/actions/runs/12345", "pull_request");

      expect(mockGithub.request).toHaveBeenCalledWith(
        expect.stringContaining("POST"),
        expect.objectContaining({
          body: expect.not.stringContaining("🔒 This issue has been locked"),
        })
      );
    });
  });

  describe("addCommentWithWorkflowLink() - outputs", () => {
    it("should set all required outputs (comment-id, comment-url, comment-repo)", async () => {
      await runAddCommentWithWorkflowLink("/repos/testowner/testrepo/issues/123/comments", "https://github.com/testowner/testrepo/actions/runs/12345", "issues");

      expect(mockCore.setOutput).toHaveBeenCalledWith("comment-id", "67890");
      expect(mockCore.setOutput).toHaveBeenCalledWith("comment-url", expect.stringContaining("issuecomment-67890"));
      expect(mockCore.setOutput).toHaveBeenCalledWith("comment-repo", "testowner/testrepo");
    });
  });
});
