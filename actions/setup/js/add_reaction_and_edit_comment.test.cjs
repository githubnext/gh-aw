// @ts-check
import { describe, it, expect, beforeEach, vi } from "vitest";
const { ERR_NOT_FOUND, ERR_VALIDATION, ERR_API } = require("./error_codes.cjs");

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
  summary: { addRaw: vi.fn().mockReturnThis(), write: vi.fn().mockResolvedValue(undefined) },
};

const mockGithub = {
  request: vi.fn(),
  graphql: vi.fn(),
  rest: { issues: { createComment: vi.fn() } },
};

const mockContext = {
  eventName: "issues",
  runId: 12345,
  repo: { owner: "testowner", repo: "testrepo" },
  payload: {
    issue: { number: 123 },
    repository: { html_url: "https://github.com/testowner/testrepo" },
  },
};

global.core = mockCore;
global.github = mockGithub;
global.context = mockContext;

// Helper to import the module fresh (bust module cache)
async function loadModule() {
  const { main, addCommentWithWorkflowLink, addReaction, addDiscussionReaction } = await import("./add_reaction_and_edit_comment.cjs?" + Date.now());
  return { main, addCommentWithWorkflowLink, addReaction, addDiscussionReaction };
}

describe("add_reaction_and_edit_comment.cjs", () => {
  beforeEach(() => {
    vi.resetAllMocks();
    delete process.env.GH_AW_REACTION;
    delete process.env.GH_AW_COMMAND;
    delete process.env.GH_AW_WORKFLOW_NAME;
    delete process.env.GH_AW_LOCK_FOR_AGENT;
    delete process.env.GITHUB_WORKFLOW;
    delete process.env.GH_AW_TRACKER_ID;
    delete process.env.GITHUB_SERVER_URL;

    global.context = {
      eventName: "issues",
      runId: 12345,
      repo: { owner: "testowner", repo: "testrepo" },
      payload: {
        issue: { number: 123 },
        repository: { html_url: "https://github.com/testowner/testrepo" },
      },
    };

    mockGithub.request.mockResolvedValue({ data: { id: 456, html_url: "https://github.com/testowner/testrepo/issues/123#issuecomment-456" } });
    mockGithub.graphql.mockResolvedValue({
      repository: { discussion: { id: "D_kwDOABcD1M4AaBbC", url: "https://github.com/testowner/testrepo/discussions/10" } },
      addReaction: { reaction: { id: "MDg6UmVhY3Rpb24xMjM0NTY3ODk=", content: "EYES" } },
      addDiscussionComment: { comment: { id: "DC_kwDOABcD1M4AaBbE", url: "https://github.com/testowner/testrepo/discussions/10#discussioncomment-999" } },
    });
  });

  describe("Issue reactions", () => {
    it("should add reaction to issue successfully", async () => {
      process.env.GH_AW_REACTION = "eyes";
      global.context.eventName = "issues";
      global.context.payload = { issue: { number: 123 }, repository: { html_url: "https://github.com/testowner/testrepo" } };
      mockGithub.request.mockResolvedValueOnce({ data: { id: 456 } });

      const { main } = await loadModule();
      await main();

      expect(mockGithub.request).toHaveBeenCalledWith("POST /repos/testowner/testrepo/issues/123/reactions", expect.objectContaining({ content: "eyes" }));
      expect(mockCore.setOutput).toHaveBeenCalledWith("reaction-id", "456");
    });

    it("should reject invalid reaction type", async () => {
      process.env.GH_AW_REACTION = "invalid";
      global.context.eventName = "issues";

      const { main } = await loadModule();
      await main();

      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Invalid reaction type: invalid"));
      expect(mockGithub.request).not.toHaveBeenCalled();
    });

    it("should fail when issue number is missing", async () => {
      process.env.GH_AW_REACTION = "eyes";
      global.context.eventName = "issues";
      global.context.payload = {};

      const { main } = await loadModule();
      await main();

      expect(mockCore.setFailed).toHaveBeenCalledWith(`${ERR_NOT_FOUND}: Issue number not found in event payload`);
      expect(mockGithub.request).not.toHaveBeenCalled();
    });
  });

  describe("Pull request reactions", () => {
    it("should add reaction to pull request and create comment", async () => {
      process.env.GH_AW_REACTION = "heart";
      process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";
      global.context.eventName = "pull_request";
      global.context.payload = {
        pull_request: { number: 456 },
        repository: { html_url: "https://github.com/testowner/testrepo" },
      };
      mockGithub.request.mockResolvedValueOnce({ data: { id: 789 } }).mockResolvedValueOnce({ data: { id: 999, html_url: "https://github.com/testowner/testrepo/pull/456#issuecomment-999" } });

      const { main } = await loadModule();
      await main();

      expect(mockGithub.request).toHaveBeenCalledWith("POST /repos/testowner/testrepo/issues/456/reactions", expect.objectContaining({ content: "heart" }));
      expect(mockGithub.request).toHaveBeenCalledWith("POST /repos/testowner/testrepo/issues/456/comments", expect.objectContaining({ body: expect.stringContaining("has started processing this pull request") }));
      expect(mockCore.setOutput).toHaveBeenCalledWith("reaction-id", "789");
      expect(mockCore.setOutput).toHaveBeenCalledWith("comment-id", "999");
      expect(mockCore.setOutput).toHaveBeenCalledWith("comment-url", "https://github.com/testowner/testrepo/pull/456#issuecomment-999");
    });

    it("should fail when PR number is missing", async () => {
      process.env.GH_AW_REACTION = "eyes";
      global.context.eventName = "pull_request";
      global.context.payload = {};

      const { main } = await loadModule();
      await main();

      expect(mockCore.setFailed).toHaveBeenCalledWith(`${ERR_NOT_FOUND}: Pull request number not found in event payload`);
    });
  });

  describe("Issue comment reactions", () => {
    it("should create new comment for issue_comment event (not edit)", async () => {
      process.env.GH_AW_REACTION = "eyes";
      process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";
      global.context.eventName = "issue_comment";
      global.context.payload = {
        issue: { number: 123 },
        comment: { id: 456 },
        repository: { html_url: "https://github.com/testowner/testrepo" },
      };
      mockGithub.request.mockResolvedValueOnce({ data: { id: 111 } }).mockResolvedValueOnce({ data: { id: 789, html_url: "https://github.com/testowner/testrepo/issues/123#issuecomment-789" } });

      const { main } = await loadModule();
      await main();

      expect(mockGithub.request).toHaveBeenCalledWith("POST /repos/testowner/testrepo/issues/123/comments", expect.objectContaining({ body: expect.stringContaining("has started processing this issue comment") }));
      expect(mockCore.setOutput).toHaveBeenCalledWith("comment-id", "789");
      expect(mockCore.setOutput).toHaveBeenCalledWith("comment-url", "https://github.com/testowner/testrepo/issues/123#issuecomment-789");
      expect(mockCore.setOutput).toHaveBeenCalledWith("comment-repo", "testowner/testrepo");
    });

    it("should fail when comment ID is missing", async () => {
      process.env.GH_AW_REACTION = "eyes";
      global.context.eventName = "issue_comment";
      global.context.payload = { issue: { number: 123 } };

      const { main } = await loadModule();
      await main();

      expect(mockCore.setFailed).toHaveBeenCalledWith(`${ERR_VALIDATION}: Comment ID not found in event payload`);
    });
  });

  describe("Pull request review comment reactions", () => {
    it("should create new comment for pull_request_review_comment event (not edit)", async () => {
      process.env.GH_AW_REACTION = "rocket";
      process.env.GH_AW_WORKFLOW_NAME = "PR Review Bot";
      global.context.eventName = "pull_request_review_comment";
      global.context.payload = {
        pull_request: { number: 456 },
        comment: { id: 789 },
        repository: { html_url: "https://github.com/testowner/testrepo" },
      };
      mockGithub.request.mockResolvedValueOnce({ data: { id: 222 } }).mockResolvedValueOnce({ data: { id: 999, html_url: "https://github.com/testowner/testrepo/pull/456#discussion_r999" } });

      const { main } = await loadModule();
      await main();

      expect(mockGithub.request).toHaveBeenCalledWith("POST /repos/testowner/testrepo/issues/456/comments", expect.objectContaining({ body: expect.stringContaining("has started processing this pull request review comment") }));
      expect(mockCore.setOutput).toHaveBeenCalledWith("comment-id", "999");
      expect(mockCore.setOutput).toHaveBeenCalledWith("comment-url", "https://github.com/testowner/testrepo/pull/456#discussion_r999");
      expect(mockCore.setOutput).toHaveBeenCalledWith("comment-repo", "testowner/testrepo");
    });
  });

  describe("Discussion reactions", () => {
    it("should add reaction to discussion using GraphQL", async () => {
      mockGithub.graphql.mockResolvedValueOnce({ addReaction: { reaction: { id: "MDg6UmVhY3Rpb24xMjM0NTY3ODk=", content: "ROCKET" } } });

      const { addDiscussionReaction } = await loadModule();
      await addDiscussionReaction("D_kwDOABcD1M4AaBbC", "rocket");

      expect(mockGithub.graphql).toHaveBeenCalledWith(expect.stringContaining("mutation"), expect.objectContaining({ subjectId: "D_kwDOABcD1M4AaBbC", content: "ROCKET" }));
      expect(mockCore.setOutput).toHaveBeenCalledWith("reaction-id", "MDg6UmVhY3Rpb24xMjM0NTY3ODk=");
    });

    it("should map all reaction types correctly for GraphQL", async () => {
      const reactionTests = [
        { input: "+1", expected: "THUMBS_UP" },
        { input: "-1", expected: "THUMBS_DOWN" },
        { input: "laugh", expected: "LAUGH" },
        { input: "confused", expected: "CONFUSED" },
        { input: "heart", expected: "HEART" },
        { input: "hooray", expected: "HOORAY" },
        { input: "rocket", expected: "ROCKET" },
        { input: "eyes", expected: "EYES" },
      ];

      for (const test of reactionTests) {
        vi.clearAllMocks();
        mockGithub.graphql.mockResolvedValueOnce({ addReaction: { reaction: { id: "abc", content: test.expected } } });

        const { addDiscussionReaction } = await loadModule();
        await addDiscussionReaction("D_kwDOABcD1M4AaBbC", test.input);

        expect(mockGithub.graphql).toHaveBeenCalledWith(expect.stringContaining("mutation"), expect.objectContaining({ content: test.expected }));
      }
    });

    it("should create comment on discussion", async () => {
      process.env.GH_AW_REACTION = "eyes";
      process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";
      global.context.eventName = "discussion";
      global.context.payload = {
        discussion: { number: 10 },
        repository: { html_url: "https://github.com/testowner/testrepo" },
      };
      mockGithub.graphql
        .mockResolvedValueOnce({ repository: { discussion: { id: "D_kwDOABcD1M4AaBbC", url: "https://github.com/testowner/testrepo/discussions/10" } } })
        .mockResolvedValueOnce({ addReaction: { reaction: { id: "MDg6UmVhY3Rpb24xMjM0NTY3ODk=", content: "EYES" } } })
        .mockResolvedValueOnce({ repository: { discussion: { id: "D_kwDOABcD1M4AaBbC", url: "https://github.com/testowner/testrepo/discussions/10" } } })
        .mockResolvedValueOnce({ addDiscussionComment: { comment: { id: "DC_kwDOABcD1M4AaBbE", url: "https://github.com/testowner/testrepo/discussions/10#discussioncomment-999" } } });

      const { main } = await loadModule();
      await main();

      expect(mockGithub.graphql).toHaveBeenCalledTimes(4);
      expect(mockGithub.graphql).toHaveBeenCalledWith(expect.stringContaining("addDiscussionComment"), expect.objectContaining({ dId: "D_kwDOABcD1M4AaBbC", body: expect.stringContaining("has started processing this discussion") }));
      expect(mockCore.setOutput).toHaveBeenCalledWith("reaction-id", "MDg6UmVhY3Rpb24xMjM0NTY3ODk=");
      expect(mockCore.setOutput).toHaveBeenCalledWith("comment-id", "DC_kwDOABcD1M4AaBbE");
      expect(mockCore.setOutput).toHaveBeenCalledWith("comment-url", "https://github.com/testowner/testrepo/discussions/10#discussioncomment-999");
    });

    it("should fail when discussion number is missing", async () => {
      process.env.GH_AW_REACTION = "eyes";
      global.context.eventName = "discussion";
      global.context.payload = { repository: { html_url: "https://github.com/testowner/testrepo" } };

      const { main } = await loadModule();
      await main();

      expect(mockCore.setFailed).toHaveBeenCalledWith(`${ERR_NOT_FOUND}: Discussion number not found in event payload`);
    });
  });

  describe("Discussion comment reactions", () => {
    it("should add reaction to discussion comment using GraphQL", async () => {
      process.env.GH_AW_REACTION = "heart";
      global.context.eventName = "discussion_comment";
      global.context.payload = {
        discussion: { number: 10 },
        comment: { id: 123, node_id: "DC_kwDOABcD1M4AaBbC", html_url: "https://github.com/testowner/testrepo/discussions/10#discussioncomment-123" },
        repository: { html_url: "https://github.com/testowner/testrepo" },
      };
      mockGithub.graphql.mockResolvedValueOnce({ addReaction: { reaction: { id: "MDg6UmVhY3Rpb24xMjM0NTY3ODk=", content: "HEART" } } });

      const { addDiscussionReaction } = await loadModule();
      await addDiscussionReaction("DC_kwDOABcD1M4AaBbC", "heart");

      expect(mockGithub.graphql).toHaveBeenCalledWith(expect.stringContaining("mutation"), expect.objectContaining({ subjectId: "DC_kwDOABcD1M4AaBbC", content: "HEART" }));
      expect(mockCore.setOutput).toHaveBeenCalledWith("reaction-id", "MDg6UmVhY3Rpb24xMjM0NTY3ODk=");
    });

    it("should fail when discussion comment node_id is missing", async () => {
      process.env.GH_AW_REACTION = "eyes";
      global.context.eventName = "discussion_comment";
      global.context.payload = {
        discussion: { number: 10 },
        comment: { id: 123 },
        repository: { html_url: "https://github.com/testowner/testrepo" },
      };

      const { main } = await loadModule();
      await main();

      expect(mockCore.setFailed).toHaveBeenCalledWith(`${ERR_NOT_FOUND}: Discussion comment node ID not found in event payload`);
      expect(mockGithub.graphql).not.toHaveBeenCalled();
    });

    it("should create threaded comment for discussion_comment events", async () => {
      process.env.GH_AW_REACTION = "eyes";
      process.env.GH_AW_WORKFLOW_NAME = "Discussion Bot";
      global.context.eventName = "discussion_comment";
      global.context.payload = {
        discussion: { number: 10 },
        comment: { id: 123, node_id: "DC_kwDOABcD1M4AaBbC" },
        repository: { html_url: "https://github.com/testowner/testrepo" },
      };
      mockGithub.graphql
        .mockResolvedValueOnce({ addReaction: { reaction: { id: "MDg6UmVhY3Rpb24xMjM0NTY3ODk=", content: "EYES" } } })
        .mockResolvedValueOnce({ repository: { discussion: { id: "D_kwDOABcD1M4AaBbC", url: "https://github.com/testowner/testrepo/discussions/10" } } })
        .mockResolvedValueOnce({
          addDiscussionComment: {
            comment: { id: "DC_kwDOABcD1M4AaBbE", url: "https://github.com/testowner/testrepo/discussions/10#discussioncomment-789" },
          },
        });

      const { main } = await loadModule();
      await main();

      expect(mockGithub.graphql).toHaveBeenCalledWith(
        expect.stringContaining("addDiscussionComment"),
        expect.objectContaining({
          dId: "D_kwDOABcD1M4AaBbC",
          body: expect.stringContaining("has started processing this discussion comment"),
          replyToId: "DC_kwDOABcD1M4AaBbC",
        })
      );
      expect(mockCore.setOutput).toHaveBeenCalledWith("comment-id", "DC_kwDOABcD1M4AaBbE");
      expect(mockCore.setOutput).toHaveBeenCalledWith("comment-url", "https://github.com/testowner/testrepo/discussions/10#discussioncomment-789");
      expect(mockCore.setOutput).toHaveBeenCalledWith("comment-repo", "testowner/testrepo");
    });

    it("should fail when discussion or comment fields are missing", async () => {
      process.env.GH_AW_REACTION = "eyes";
      global.context.eventName = "discussion_comment";
      global.context.payload = {
        discussion: { number: 10 },
        // Missing comment field
      };

      const { main } = await loadModule();
      await main();

      expect(mockCore.setFailed).toHaveBeenCalledWith(`${ERR_NOT_FOUND}: Discussion or comment information not found in event payload`);
    });
  });

  describe("Unsupported event types", () => {
    it("should fail for unsupported event type", async () => {
      process.env.GH_AW_REACTION = "eyes";
      global.context.eventName = "push";
      global.context.payload = { repository: { html_url: "https://github.com/testowner/testrepo" } };

      const { main } = await loadModule();
      await main();

      expect(mockCore.setFailed).toHaveBeenCalledWith(`${ERR_VALIDATION}: Unsupported event type: push`);
    });
  });

  describe("Error handling", () => {
    it("should silently ignore locked issue errors (status 403 + locked message)", async () => {
      const lockedError = new Error("Issue is locked");
      /** @type {any} */ lockedError.status = 403;
      process.env.GH_AW_REACTION = "eyes";
      global.context.eventName = "issues";
      global.context.payload = { issue: { number: 123 }, repository: { html_url: "https://github.com/testowner/testrepo" } };
      mockGithub.request.mockRejectedValueOnce(lockedError);

      const { main } = await loadModule();
      await main();

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("resource is locked"));
      expect(mockCore.error).not.toHaveBeenCalled();
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should fail for errors with 'locked' message but non-403 status", async () => {
      const lockedError = new Error("Lock conversation is enabled");
      /** @type {any} */ lockedError.status = 500;
      process.env.GH_AW_REACTION = "eyes";
      global.context.eventName = "issues";
      global.context.payload = { issue: { number: 123 }, repository: { html_url: "https://github.com/testowner/testrepo" } };
      mockGithub.request.mockRejectedValueOnce(lockedError);

      const { main } = await loadModule();
      await main();

      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Failed to process reaction"));
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Failed to process reaction"));
    });

    it("should fail for 403 errors that don't mention locked", async () => {
      const forbiddenError = new Error("Forbidden: insufficient permissions");
      /** @type {any} */ forbiddenError.status = 403;
      process.env.GH_AW_REACTION = "eyes";
      global.context.eventName = "issues";
      global.context.payload = { issue: { number: 123 }, repository: { html_url: "https://github.com/testowner/testrepo" } };
      mockGithub.request.mockRejectedValueOnce(forbiddenError);

      const { main } = await loadModule();
      await main();

      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Failed to process reaction"));
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Failed to process reaction"));
    });

    it("should fail for other non-403 errors", async () => {
      const serverError = new Error("Internal server error");
      /** @type {any} */ serverError.status = 500;
      process.env.GH_AW_REACTION = "eyes";
      global.context.eventName = "issues";
      global.context.payload = { issue: { number: 123 }, repository: { html_url: "https://github.com/testowner/testrepo" } };
      mockGithub.request.mockRejectedValueOnce(serverError);

      const { main } = await loadModule();
      await main();

      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Failed to process reaction"));
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining(`${ERR_API}: Failed to process reaction`));
    });
  });

  describe("addCommentWithWorkflowLink() - markers", () => {
    it("should include workflow-id marker when GITHUB_WORKFLOW is set", async () => {
      process.env.GITHUB_WORKFLOW = "test-workflow.yml";
      mockGithub.request.mockResolvedValueOnce({ data: { id: 123, html_url: "https://example.com" } });

      const { addCommentWithWorkflowLink } = await loadModule();
      await addCommentWithWorkflowLink("/repos/testowner/testrepo/issues/123/comments", "https://github.com/testowner/testrepo/actions/runs/12345", "issues");

      expect(mockGithub.request).toHaveBeenCalledWith(expect.stringContaining("POST"), expect.objectContaining({ body: expect.stringContaining("<!-- gh-aw-workflow-id: test-workflow.yml -->") }));
    });

    it("should include tracker-id marker when GH_AW_TRACKER_ID is set", async () => {
      process.env.GH_AW_TRACKER_ID = "tracker-123";
      mockGithub.request.mockResolvedValueOnce({ data: { id: 123, html_url: "https://example.com" } });

      const { addCommentWithWorkflowLink } = await loadModule();
      await addCommentWithWorkflowLink("/repos/testowner/testrepo/issues/123/comments", "https://github.com/testowner/testrepo/actions/runs/12345", "issues");

      expect(mockGithub.request).toHaveBeenCalledWith(expect.stringContaining("POST"), expect.objectContaining({ body: expect.stringContaining("<!-- gh-aw-tracker-id: tracker-123 -->") }));
    });

    it("should always include reaction comment type marker", async () => {
      mockGithub.request.mockResolvedValueOnce({ data: { id: 123, html_url: "https://example.com" } });

      const { addCommentWithWorkflowLink } = await loadModule();
      await addCommentWithWorkflowLink("/repos/testowner/testrepo/issues/123/comments", "https://github.com/testowner/testrepo/actions/runs/12345", "issues");

      expect(mockGithub.request).toHaveBeenCalledWith(expect.stringContaining("POST"), expect.objectContaining({ body: expect.stringContaining("<!-- gh-aw-comment-type: reaction -->") }));
    });

    it("should add lock notice for issues event when GH_AW_LOCK_FOR_AGENT=true", async () => {
      process.env.GH_AW_LOCK_FOR_AGENT = "true";
      mockGithub.request.mockResolvedValueOnce({ data: { id: 123, html_url: "https://example.com" } });

      const { addCommentWithWorkflowLink } = await loadModule();
      await addCommentWithWorkflowLink("/repos/testowner/testrepo/issues/123/comments", "https://github.com/testowner/testrepo/actions/runs/12345", "issues");

      expect(mockGithub.request).toHaveBeenCalledWith(expect.stringContaining("POST"), expect.objectContaining({ body: expect.stringContaining("🔒 This issue has been locked") }));
    });

    it("should not add lock notice for pull_request events", async () => {
      process.env.GH_AW_LOCK_FOR_AGENT = "true";
      mockGithub.request.mockResolvedValueOnce({ data: { id: 123, html_url: "https://example.com" } });

      const { addCommentWithWorkflowLink } = await loadModule();
      await addCommentWithWorkflowLink("/repos/testowner/testrepo/issues/123/comments", "https://github.com/testowner/testrepo/actions/runs/12345", "pull_request");

      expect(mockGithub.request).toHaveBeenCalledWith(expect.stringContaining("POST"), expect.objectContaining({ body: expect.not.stringContaining("🔒 This issue has been locked") }));
    });
  });

  describe("addReaction()", () => {
    it("should add reaction via REST API and set output", async () => {
      mockGithub.request.mockResolvedValueOnce({ data: { id: 789 } });

      const { addReaction } = await loadModule();
      await addReaction("/repos/testowner/testrepo/issues/123/reactions", "eyes");

      expect(mockGithub.request).toHaveBeenCalledWith("POST /repos/testowner/testrepo/issues/123/reactions", expect.objectContaining({ content: "eyes" }));
      expect(mockCore.setOutput).toHaveBeenCalledWith("reaction-id", "789");
    });

    it("should set empty reaction-id when response has no id", async () => {
      mockGithub.request.mockResolvedValueOnce({ data: {} });

      const { addReaction } = await loadModule();
      await addReaction("/repos/testowner/testrepo/issues/123/reactions", "eyes");

      expect(mockCore.setOutput).toHaveBeenCalledWith("reaction-id", "");
    });
  });
});
