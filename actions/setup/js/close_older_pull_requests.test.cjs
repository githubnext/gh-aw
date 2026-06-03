// @ts-check

import { describe, it, expect, beforeEach, vi } from "vitest";
import { closeOlderPullRequests, searchOlderPullRequests, addPullRequestComment, closePullRequest, getCloseOlderPullRequestMessage, MAX_CLOSE_COUNT } from "./close_older_pull_requests.cjs";

// Mock globals
global.core = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
};

describe("close_older_pull_requests", () => {
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
        },
        pulls: {
          update: vi.fn(),
        },
      },
    };
  });

  describe("searchOlderPullRequests", () => {
    it("should search for pull requests with workflow-id marker", async () => {
      mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
        data: {
          items: [
            {
              number: 123,
              title: "Chaos test - 2024-01",
              html_url: "https://github.com/owner/repo/pull/123",
              labels: [],
              pull_request: {},
              body: "<!-- gh-aw-workflow-id: chaos-pr-bundle-fuzzer -->",
            },
            {
              number: 124,
              title: "Chaos test - 2024-02",
              html_url: "https://github.com/owner/repo/pull/124",
              labels: [],
              pull_request: {},
              body: "<!-- gh-aw-workflow-id: chaos-pr-bundle-fuzzer -->",
            },
          ],
        },
      });

      const results = await searchOlderPullRequests(mockGithub, "owner", "repo", "chaos-pr-bundle-fuzzer", 125);

      expect(results).toHaveLength(2);
      expect(results[0].number).toBe(123);
      expect(results[1].number).toBe(124);
      expect(mockGithub.rest.search.issuesAndPullRequests).toHaveBeenCalledWith({
        q: 'repo:owner/repo is:pr is:open "gh-aw-workflow-id: chaos-pr-bundle-fuzzer" in:body',
        per_page: 50,
      });
    });

    it("should exclude the newly created pull request", async () => {
      mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
        data: {
          items: [
            {
              number: 123,
              title: "Chaos test - old",
              html_url: "https://github.com/owner/repo/pull/123",
              labels: [],
              pull_request: {},
              body: "<!-- gh-aw-workflow-id: chaos-pr-bundle-fuzzer -->",
            },
            {
              number: 124,
              title: "Chaos test - new",
              html_url: "https://github.com/owner/repo/pull/124",
              labels: [],
              pull_request: {},
              body: "<!-- gh-aw-workflow-id: chaos-pr-bundle-fuzzer -->",
            },
          ],
        },
      });

      const results = await searchOlderPullRequests(mockGithub, "owner", "repo", "chaos-pr-bundle-fuzzer", 124);

      expect(results).toHaveLength(1);
      expect(results[0].number).toBe(123);
    });

    it("should return empty array if no workflow-id provided", async () => {
      const results = await searchOlderPullRequests(mockGithub, "owner", "repo", "", 125);

      expect(results).toHaveLength(0);
      expect(mockGithub.rest.search.issuesAndPullRequests).not.toHaveBeenCalled();
    });

    it("should exclude issues (only return pull requests)", async () => {
      mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
        data: {
          items: [
            {
              number: 123,
              title: "Pull Request",
              html_url: "https://github.com/owner/repo/pull/123",
              labels: [],
              pull_request: {},
              body: "<!-- gh-aw-workflow-id: my-workflow -->",
            },
            {
              number: 124,
              title: "Issue - should be excluded",
              html_url: "https://github.com/owner/repo/issues/124",
              labels: [],
              // No pull_request property means it's an issue
              body: "<!-- gh-aw-workflow-id: my-workflow -->",
            },
          ],
        },
      });

      const results = await searchOlderPullRequests(mockGithub, "owner", "repo", "my-workflow", 125);

      expect(results).toHaveLength(1);
      expect(results[0].number).toBe(123);
    });

    it("should return empty array if no results", async () => {
      mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
        data: {
          items: [],
        },
      });

      const results = await searchOlderPullRequests(mockGithub, "owner", "repo", "my-workflow", 125);

      expect(results).toHaveLength(0);
    });

    it("should exclude PRs whose body does not contain the exact marker", async () => {
      mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
        data: {
          items: [
            {
              number: 123,
              title: "Matching PR",
              html_url: "https://github.com/owner/repo/pull/123",
              labels: [],
              pull_request: {},
              body: "Some content\n<!-- gh-aw-workflow-id: my-workflow -->",
            },
            {
              number: 124,
              title: "Substring match - should be excluded",
              html_url: "https://github.com/owner/repo/pull/124",
              labels: [],
              pull_request: {},
              // Body has a related-but-longer workflow ID - GitHub search may match this
              // but exact filtering should exclude it
              body: "Some content\n<!-- gh-aw-workflow-id: my-workflow-extended -->",
            },
            {
              number: 125,
              title: "No marker - should be excluded",
              html_url: "https://github.com/owner/repo/pull/125",
              labels: [],
              pull_request: {},
              body: "PR without any marker",
            },
          ],
        },
      });

      const results = await searchOlderPullRequests(mockGithub, "owner", "repo", "my-workflow", 999);

      expect(results).toHaveLength(1);
      expect(results[0].number).toBe(123);
    });

    it("should filter by gh-aw-workflow-call-id when callerWorkflowId is provided", async () => {
      mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
        data: {
          items: [
            {
              number: 123,
              title: "Same caller - should be included",
              html_url: "https://github.com/owner/repo/pull/123",
              labels: [],
              pull_request: {},
              body: "<!-- gh-aw-workflow-id: my-reusable-workflow -->\n<!-- gh-aw-workflow-call-id: owner/repo/CallerA -->",
            },
            {
              number: 124,
              title: "Different caller - should be excluded",
              html_url: "https://github.com/owner/repo/pull/124",
              labels: [],
              pull_request: {},
              body: "<!-- gh-aw-workflow-id: my-reusable-workflow -->\n<!-- gh-aw-workflow-call-id: owner/repo/CallerB -->",
            },
            {
              number: 125,
              title: "Old PR without call-id - should be excluded",
              html_url: "https://github.com/owner/repo/pull/125",
              labels: [],
              pull_request: {},
              body: "<!-- gh-aw-workflow-id: my-reusable-workflow -->",
            },
          ],
        },
      });

      const results = await searchOlderPullRequests(mockGithub, "owner", "repo", "my-reusable-workflow", 999, "owner/repo/CallerA");

      expect(results).toHaveLength(1);
      expect(results[0].number).toBe(123);
    });

    it("should use close-key marker as primary search term when closeOlderKey is provided", async () => {
      mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
        data: {
          items: [
            {
              number: 123,
              title: "Has close-key marker - should be included",
              html_url: "https://github.com/owner/repo/pull/123",
              labels: [],
              pull_request: {},
              body: "<!-- gh-aw-workflow-id: some-workflow -->\n<!-- gh-aw-close-key: my-stable-key -->",
            },
            {
              number: 124,
              title: "Missing close-key marker - should be excluded",
              html_url: "https://github.com/owner/repo/pull/124",
              labels: [],
              pull_request: {},
              body: "<!-- gh-aw-workflow-id: some-workflow -->",
            },
          ],
        },
      });

      const results = await searchOlderPullRequests(mockGithub, "owner", "repo", "some-workflow", 999, undefined, "my-stable-key");

      expect(results).toHaveLength(1);
      expect(results[0].number).toBe(123);
      expect(mockGithub.rest.search.issuesAndPullRequests).toHaveBeenCalledWith({
        q: 'repo:owner/repo is:pr is:open "gh-aw-close-key: my-stable-key" in:body',
        per_page: 50,
      });
    });

    it("should return empty array when neither workflowId nor closeOlderKey is provided", async () => {
      const results = await searchOlderPullRequests(mockGithub, "owner", "repo", "", 999, undefined, undefined);

      expect(results).toHaveLength(0);
      expect(mockGithub.rest.search.issuesAndPullRequests).not.toHaveBeenCalled();
    });
  });

  describe("addPullRequestComment", () => {
    it("should add comment to pull request", async () => {
      mockGithub.rest.issues.createComment.mockResolvedValue({
        data: {
          id: 456,
          html_url: "https://github.com/owner/repo/pull/123#issuecomment-456",
        },
      });

      const result = await addPullRequestComment(mockGithub, "owner", "repo", 123, "Test comment");

      expect(result).toEqual({
        id: 456,
        html_url: "https://github.com/owner/repo/pull/123#issuecomment-456",
      });
      expect(mockGithub.rest.issues.createComment).toHaveBeenCalledWith({
        owner: "owner",
        repo: "repo",
        issue_number: 123,
        body: "Test comment",
      });
    });
  });

  describe("closePullRequest", () => {
    it("should close pull request", async () => {
      mockGithub.rest.pulls.update.mockResolvedValue({
        data: {
          number: 123,
          html_url: "https://github.com/owner/repo/pull/123",
        },
      });

      const result = await closePullRequest(mockGithub, "owner", "repo", 123);

      expect(result).toEqual({
        number: 123,
        html_url: "https://github.com/owner/repo/pull/123",
      });
      expect(mockGithub.rest.pulls.update).toHaveBeenCalledWith({
        owner: "owner",
        repo: "repo",
        pull_number: 123,
        state: "closed",
      });
    });
  });

  describe("getCloseOlderPullRequestMessage", () => {
    it("should generate closing message", () => {
      const message = getCloseOlderPullRequestMessage({
        newPullRequestUrl: "https://github.com/owner/repo/pull/125",
        newPullRequestNumber: 125,
        workflowName: "Test Workflow",
        runUrl: "https://github.com/owner/repo/actions/runs/123",
      });

      expect(message).toContain("newer pull request has been created: #125");
      expect(message).toContain("https://github.com/owner/repo/pull/125");
      expect(message).toContain("Test Workflow");
      expect(message).toContain("https://github.com/owner/repo/actions/runs/123");
    });
  });

  describe("closeOlderPullRequests", () => {
    it("should close older pull requests successfully", async () => {
      mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
        data: {
          items: [
            {
              number: 123,
              title: "[chaos-test] Old PR",
              html_url: "https://github.com/owner/repo/pull/123",
              labels: [],
              pull_request: {},
              body: "<!-- gh-aw-workflow-id: chaos-pr-bundle-fuzzer -->",
            },
          ],
        },
      });

      mockGithub.rest.issues.createComment.mockResolvedValue({
        data: { id: 456, html_url: "https://github.com/owner/repo/pull/123#issuecomment-456" },
      });

      mockGithub.rest.pulls.update.mockResolvedValue({
        data: { number: 123, html_url: "https://github.com/owner/repo/pull/123" },
      });

      const newPR = { number: 125, html_url: "https://github.com/owner/repo/pull/125" };
      const results = await closeOlderPullRequests(mockGithub, "owner", "repo", "chaos-pr-bundle-fuzzer", newPR, "Chaos PR Bundle Fuzzer", "https://github.com/owner/repo/actions/runs/123");

      expect(results).toHaveLength(1);
      expect(results[0].number).toBe(123);
      expect(mockGithub.rest.issues.createComment).toHaveBeenCalled();
      expect(mockGithub.rest.pulls.update).toHaveBeenCalledWith({
        owner: "owner",
        repo: "repo",
        pull_number: 123,
        state: "closed",
      });
    });

    it("should limit to MAX_CLOSE_COUNT pull requests", async () => {
      const items = [];
      for (let i = 1; i <= 15; i++) {
        items.push({
          number: i,
          title: `PR ${i}`,
          html_url: `https://github.com/owner/repo/pull/${i}`,
          labels: [],
          pull_request: {},
          body: "<!-- gh-aw-workflow-id: my-workflow -->",
        });
      }

      mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
        data: { items },
      });

      mockGithub.rest.issues.createComment.mockResolvedValue({
        data: { id: 456, html_url: "https://github.com/owner/repo/pull/1#issuecomment-456" },
      });

      mockGithub.rest.pulls.update.mockResolvedValue({
        data: { number: 1, html_url: "https://github.com/owner/repo/pull/1" },
      });

      const newPR = { number: 20, html_url: "https://github.com/owner/repo/pull/20" };
      const results = await closeOlderPullRequests(mockGithub, "owner", "repo", "my-workflow", newPR, "My Workflow", "https://github.com/owner/repo/actions/runs/123");

      expect(results).toHaveLength(MAX_CLOSE_COUNT);
      expect(global.core.warning).toHaveBeenCalledWith(`⚠️  Found 15 older pull requests, but only closing the first ${MAX_CLOSE_COUNT}`);
    });

    it("should continue on error for individual pull requests", async () => {
      mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
        data: {
          items: [
            {
              number: 123,
              title: "PR 1",
              html_url: "https://github.com/owner/repo/pull/123",
              labels: [],
              pull_request: {},
              body: "<!-- gh-aw-workflow-id: my-workflow -->",
            },
            {
              number: 124,
              title: "PR 2",
              html_url: "https://github.com/owner/repo/pull/124",
              labels: [],
              pull_request: {},
              body: "<!-- gh-aw-workflow-id: my-workflow -->",
            },
          ],
        },
      });

      // First PR fails
      mockGithub.rest.issues.createComment.mockRejectedValueOnce(new Error("API Error"));

      // Second PR succeeds
      mockGithub.rest.issues.createComment.mockResolvedValueOnce({
        data: { id: 456, html_url: "https://github.com/owner/repo/pull/124#issuecomment-456" },
      });

      mockGithub.rest.pulls.update.mockResolvedValue({
        data: { number: 124, html_url: "https://github.com/owner/repo/pull/124" },
      });

      const newPR = { number: 125, html_url: "https://github.com/owner/repo/pull/125" };
      const results = await closeOlderPullRequests(mockGithub, "owner", "repo", "my-workflow", newPR, "My Workflow", "https://github.com/owner/repo/actions/runs/123");

      expect(results).toHaveLength(1);
      expect(results[0].number).toBe(124);
      expect(global.core.error).toHaveBeenCalledWith(expect.stringContaining("Failed to close pull request #123"));
    });

    it("should return empty array if no older pull requests found", async () => {
      mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
        data: { items: [] },
      });

      const newPR = { number: 125, html_url: "https://github.com/owner/repo/pull/125" };
      const results = await closeOlderPullRequests(mockGithub, "owner", "repo", "my-workflow", newPR, "My Workflow", "https://github.com/owner/repo/actions/runs/123");

      expect(results).toHaveLength(0);
      expect(global.core.info).toHaveBeenCalledWith("✓ No older pull requests found to close - operation complete");
    });
  });
});
