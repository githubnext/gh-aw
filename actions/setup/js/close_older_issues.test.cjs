// @ts-check

import { describe, it, expect, beforeEach, vi } from "vitest";
import { closeOlderIssues, searchOlderIssues, addIssueComment, closeIssueAsNotPlanned, getCloseOlderIssueMessage, MAX_CLOSE_COUNT } from "./close_older_issues.cjs";

// Mock globals
global.core = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
};

describe("close_older_issues", () => {
  let mockGithub;

  beforeEach(() => {
    vi.clearAllMocks();
    mockGithub = {
      rest: {
        search: {
          issuesAndPullRequests: vi.fn(),
        },
        issues: {
          createComment: vi.fn(),
          update: vi.fn(),
        },
      },
    };
  });

  describe("searchOlderIssues", () => {
    it("should search for issues with workflow-id marker", async () => {
      mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
        data: {
          items: [
            {
              number: 123,
              title: "Weekly Report - 2024-01",
              html_url: "https://github.com/owner/repo/issues/123",
              labels: [],
            },
            {
              number: 124,
              title: "Weekly Report - 2024-02",
              html_url: "https://github.com/owner/repo/issues/124",
              labels: [],
            },
          ],
        },
      });

      const results = await searchOlderIssues(mockGithub, "owner", "repo", "test-workflow", 125);

      expect(results).toHaveLength(2);
      expect(results[0].number).toBe(123);
      expect(results[1].number).toBe(124);
      expect(mockGithub.rest.search.issuesAndPullRequests).toHaveBeenCalledWith({
        q: 'repo:owner/repo is:issue is:open "gh-aw-workflow-id: test-workflow" in:body',
        per_page: 50,
      });
    });

    it("should exclude the newly created issue", async () => {
      mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
        data: {
          items: [
            {
              number: 123,
              title: "Weekly Report - 2024-01",
              html_url: "https://github.com/owner/repo/issues/123",
              labels: [],
            },
            {
              number: 124,
              title: "Weekly Report - 2024-02",
              html_url: "https://github.com/owner/repo/issues/124",
              labels: [],
            },
          ],
        },
      });

      const results = await searchOlderIssues(mockGithub, "owner", "repo", "test-workflow", 124);

      expect(results).toHaveLength(1);
      expect(results[0].number).toBe(123);
    });

    it("should return empty array if no workflow-id provided", async () => {
      const results = await searchOlderIssues(mockGithub, "owner", "repo", "", 125);

      expect(results).toHaveLength(0);
      expect(mockGithub.rest.search.issuesAndPullRequests).not.toHaveBeenCalled();
    });

    it("should exclude pull requests", async () => {
      mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
        data: {
          items: [
            {
              number: 123,
              title: "Issue",
              html_url: "https://github.com/owner/repo/issues/123",
              labels: [],
            },
            {
              number: 124,
              title: "Pull Request",
              html_url: "https://github.com/owner/repo/pull/124",
              labels: [],
              pull_request: {},
            },
          ],
        },
      });

      const results = await searchOlderIssues(mockGithub, "owner", "repo", "test-workflow", 125);

      expect(results).toHaveLength(1);
      expect(results[0].number).toBe(123);
    });

    it("should return empty array if no results", async () => {
      mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
        data: {
          items: [],
        },
      });

      const results = await searchOlderIssues(mockGithub, "owner", "repo", "test-workflow", 125);

      expect(results).toHaveLength(0);
    });
  });

  describe("addIssueComment", () => {
    it("should add comment to issue", async () => {
      mockGithub.rest.issues.createComment.mockResolvedValue({
        data: {
          id: 456,
          html_url: "https://github.com/owner/repo/issues/123#issuecomment-456",
        },
      });

      const result = await addIssueComment(mockGithub, "owner", "repo", 123, "Test comment");

      expect(result).toEqual({
        id: 456,
        html_url: "https://github.com/owner/repo/issues/123#issuecomment-456",
      });
      expect(mockGithub.rest.issues.createComment).toHaveBeenCalledWith({
        owner: "owner",
        repo: "repo",
        issue_number: 123,
        body: "Test comment",
      });
    });
  });

  describe("closeIssueAsNotPlanned", () => {
    it("should close issue as not planned", async () => {
      mockGithub.rest.issues.update.mockResolvedValue({
        data: {
          number: 123,
          html_url: "https://github.com/owner/repo/issues/123",
        },
      });

      const result = await closeIssueAsNotPlanned(mockGithub, "owner", "repo", 123);

      expect(result).toEqual({
        number: 123,
        html_url: "https://github.com/owner/repo/issues/123",
      });
      expect(mockGithub.rest.issues.update).toHaveBeenCalledWith({
        owner: "owner",
        repo: "repo",
        issue_number: 123,
        state: "closed",
        state_reason: "not_planned",
      });
    });
  });

  describe("getCloseOlderIssueMessage", () => {
    it("should generate closing message", () => {
      const message = getCloseOlderIssueMessage({
        newIssueUrl: "https://github.com/owner/repo/issues/125",
        newIssueNumber: 125,
        workflowName: "Test Workflow",
        runUrl: "https://github.com/owner/repo/actions/runs/123",
      });

      expect(message).toContain("newer issue has been created: #125");
      expect(message).toContain("https://github.com/owner/repo/issues/125");
      expect(message).toContain("Test Workflow");
      expect(message).toContain("https://github.com/owner/repo/actions/runs/123");
    });
  });

  describe("closeOlderIssues", () => {
    it("should close older issues successfully", async () => {
      mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
        data: {
          items: [
            {
              number: 123,
              title: "Prefix - Old Issue",
              html_url: "https://github.com/owner/repo/issues/123",
              labels: [],
            },
          ],
        },
      });

      mockGithub.rest.issues.createComment.mockResolvedValue({
        data: { id: 456, html_url: "https://github.com/owner/repo/issues/123#issuecomment-456" },
      });

      mockGithub.rest.issues.update.mockResolvedValue({
        data: { number: 123, html_url: "https://github.com/owner/repo/issues/123" },
      });

      const newIssue = { number: 125, html_url: "https://github.com/owner/repo/issues/125" };
      const results = await closeOlderIssues(mockGithub, "owner", "repo", "test-workflow", newIssue, "Test Workflow", "https://github.com/owner/repo/actions/runs/123");

      expect(results).toHaveLength(1);
      expect(results[0].number).toBe(123);
      expect(mockGithub.rest.issues.createComment).toHaveBeenCalled();
      expect(mockGithub.rest.issues.update).toHaveBeenCalled();
    });

    it("should limit to MAX_CLOSE_COUNT issues", async () => {
      const items = [];
      for (let i = 1; i <= 15; i++) {
        items.push({
          number: i,
          title: `Issue ${i}`,
          html_url: `https://github.com/owner/repo/issues/${i}`,
          labels: [],
        });
      }

      mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
        data: { items },
      });

      mockGithub.rest.issues.createComment.mockResolvedValue({
        data: { id: 456, html_url: "https://github.com/owner/repo/issues/1#issuecomment-456" },
      });

      mockGithub.rest.issues.update.mockResolvedValue({
        data: { number: 1, html_url: "https://github.com/owner/repo/issues/1" },
      });

      const newIssue = { number: 20, html_url: "https://github.com/owner/repo/issues/20" };
      const results = await closeOlderIssues(mockGithub, "owner", "repo", "test-workflow", newIssue, "Test Workflow", "https://github.com/owner/repo/actions/runs/123");

      expect(results).toHaveLength(MAX_CLOSE_COUNT);
      expect(global.core.warning).toHaveBeenCalledWith(`⚠️  Found 15 older issues, but only closing the first ${MAX_CLOSE_COUNT}`);
    });

    it("should continue on error for individual issues", async () => {
      mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
        data: {
          items: [
            {
              number: 123,
              title: "Issue 1",
              html_url: "https://github.com/owner/repo/issues/123",
              labels: [],
            },
            {
              number: 124,
              title: "Issue 2",
              html_url: "https://github.com/owner/repo/issues/124",
              labels: [],
            },
          ],
        },
      });

      // First issue fails
      mockGithub.rest.issues.createComment.mockRejectedValueOnce(new Error("API Error"));

      // Second issue succeeds
      mockGithub.rest.issues.createComment.mockResolvedValueOnce({
        data: { id: 456, html_url: "https://github.com/owner/repo/issues/124#issuecomment-456" },
      });

      mockGithub.rest.issues.update.mockResolvedValue({
        data: { number: 124, html_url: "https://github.com/owner/repo/issues/124" },
      });

      const newIssue = { number: 125, html_url: "https://github.com/owner/repo/issues/125" };
      const results = await closeOlderIssues(mockGithub, "owner", "repo", "test-workflow", newIssue, "Test Workflow", "https://github.com/owner/repo/actions/runs/123");

      expect(results).toHaveLength(1);
      expect(results[0].number).toBe(124);
      expect(global.core.error).toHaveBeenCalledWith(expect.stringContaining("Failed to close issue #123"));
    });

    it("should return empty array if no older issues found", async () => {
      mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
        data: { items: [] },
      });

      const newIssue = { number: 125, html_url: "https://github.com/owner/repo/issues/125" };
      const results = await closeOlderIssues(mockGithub, "owner", "repo", "test-workflow", newIssue, "Test Workflow", "https://github.com/owner/repo/actions/runs/123");

      expect(results).toHaveLength(0);
      expect(global.core.info).toHaveBeenCalledWith("✓ No older issues found to close - operation complete");
    });
  });
});
