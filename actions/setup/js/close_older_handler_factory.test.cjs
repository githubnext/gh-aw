// @ts-check

import { describe, it, expect, beforeEach, vi } from "vitest";
import { adaptCloseMessage, createCloseOlderHandler, searchOlderIssueLikeEntities } from "./close_older_handler_factory.cjs";
import { generateWorkflowIdMarker } from "./generate_footer.cjs";

// Mock globals
global.core = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
};

describe("close_older_handler_factory", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("adaptCloseMessage", () => {
    it("maps generic params to entity-specific message keys", () => {
      const getMessage = vi.fn().mockReturnValue("message");

      const result = adaptCloseMessage(
        "PullRequest",
        getMessage
      )({
        newEntityUrl: "https://example.com/pr/2",
        newEntityNumber: 2,
        workflowName: "Test Workflow",
        runUrl: "https://example.com/run/1",
      });

      expect(result).toBe("message");
      expect(getMessage).toHaveBeenCalledWith({
        newPullRequestUrl: "https://example.com/pr/2",
        newPullRequestNumber: 2,
        workflowName: "Test Workflow",
        runUrl: "https://example.com/run/1",
      });
    });
  });

  describe("createCloseOlderHandler", () => {
    it("comments, closes, and maps results using the configured url key", async () => {
      const addComment = vi.fn().mockResolvedValue({ id: 1 });
      const closeEntity = vi.fn().mockResolvedValue({ number: 1 });
      const searchOlderEntities = vi.fn().mockResolvedValue([{ number: 1, title: "Older", html_url: "https://example.com/issues/1" }]);

      const handler = createCloseOlderHandler({
        entityType: "issue",
        entityTypePlural: "issues",
        entityKey: "Issue",
        urlKey: "html_url",
        getMessage: ({ newIssueNumber }) => `superseded by #${newIssueNumber}`,
        addComment,
        closeEntity,
        delayMs: 0,
      });

      const github = {};
      const result = await handler(github, "owner", "repo", "test-workflow", { number: 2, html_url: "https://example.com/issues/2" }, "Test Workflow", "https://example.com/run/1", searchOlderEntities);

      expect(searchOlderEntities).toHaveBeenCalledWith(github, "owner", "repo", "test-workflow", 2);
      expect(addComment).toHaveBeenCalledWith(github, "owner", "repo", 1, "superseded by #2");
      expect(closeEntity).toHaveBeenCalledWith(github, "owner", "repo", 1);
      expect(result).toEqual([{ number: 1, html_url: "https://example.com/issues/1" }]);
    });

    it("supports custom entity ids, url keys, and extra search args", async () => {
      const addComment = vi.fn().mockResolvedValue({ id: "c1" });
      const closeEntity = vi.fn().mockResolvedValue({ id: "d1" });
      const searchOlderEntities = vi.fn().mockResolvedValue([{ id: "d1", number: 1, title: "Older", url: "https://example.com/discussions/1" }]);

      const handler = createCloseOlderHandler({
        entityType: "discussion",
        entityTypePlural: "discussions",
        entityKey: "Discussion",
        urlKey: "url",
        getMessage: () => "closing",
        addComment,
        closeEntity,
        delayMs: 0,
        getEntityId: entity => entity.id,
      });

      const github = {};
      const result = await handler(github, "owner", "repo", "test-workflow", { number: 2, url: "https://example.com/discussions/2" }, "Test Workflow", "https://example.com/run/1", searchOlderEntities, "category-1");

      expect(searchOlderEntities).toHaveBeenCalledWith(github, "owner", "repo", "test-workflow", "category-1", 2);
      expect(addComment).toHaveBeenCalledWith(github, "owner", "repo", "d1", "closing");
      expect(closeEntity).toHaveBeenCalledWith(github, "owner", "repo", "d1");
      expect(result).toEqual([{ number: 1, url: "https://example.com/discussions/1" }]);
    });
  });

  describe("searchOlderIssueLikeEntities", () => {
    const marker = generateWorkflowIdMarker("test-workflow");

    /** @param {Array<any>} items */
    function makeGithub(items) {
      return {
        rest: {
          search: {
            issuesAndPullRequests: vi.fn().mockResolvedValue({ data: { items } }),
          },
        },
      };
    }

    it("filters out pull requests when searching issues", async () => {
      const github = makeGithub([
        { number: 1, title: "Issue", html_url: "https://example.com/issues/1", body: marker, created_at: "2024-01-01" },
        { number: 2, title: "PR", html_url: "https://example.com/pull/2", body: marker, pull_request: {}, created_at: "2024-01-01" },
      ]);

      const result = await searchOlderIssueLikeEntities({
        github,
        owner: "owner",
        repo: "repo",
        workflowId: "test-workflow",
        excludeNumber: 99,
        isPullRequest: false,
      });

      expect(github.rest.search.issuesAndPullRequests).toHaveBeenCalledWith(expect.objectContaining({ q: expect.stringContaining("is:issue") }));
      expect(result).toEqual([{ number: 1, title: "Issue", html_url: "https://example.com/issues/1", labels: [], created_at: "2024-01-01" }]);
    });

    it("filters out issues when searching pull requests", async () => {
      const github = makeGithub([
        { number: 1, title: "Issue", html_url: "https://example.com/issues/1", body: marker, created_at: "2024-01-01" },
        { number: 2, title: "PR", html_url: "https://example.com/pull/2", body: marker, pull_request: {}, created_at: "2024-01-01" },
      ]);

      const result = await searchOlderIssueLikeEntities({
        github,
        owner: "owner",
        repo: "repo",
        workflowId: "test-workflow",
        excludeNumber: 99,
        isPullRequest: true,
      });

      expect(github.rest.search.issuesAndPullRequests).toHaveBeenCalledWith(expect.objectContaining({ q: expect.stringContaining("is:pr") }));
      expect(result).toEqual([{ number: 2, title: "PR", html_url: "https://example.com/pull/2", labels: [], created_at: "2024-01-01" }]);
    });

    it("excludes the new entity and additional excluded numbers", async () => {
      const github = makeGithub([
        { number: 1, title: "Older", html_url: "https://example.com/issues/1", body: marker, created_at: "2024-01-01" },
        { number: 2, title: "Same run", html_url: "https://example.com/issues/2", body: marker, created_at: "2024-01-01" },
        { number: 3, title: "New", html_url: "https://example.com/issues/3", body: marker, created_at: "2024-01-01" },
      ]);

      const result = await searchOlderIssueLikeEntities({
        github,
        owner: "owner",
        repo: "repo",
        workflowId: "test-workflow",
        excludeNumber: 3,
        isPullRequest: false,
        additionalExcludeNumbers: new Set([2]),
      });

      expect(result.map(item => item.number)).toEqual([1]);
    });
  });
});
