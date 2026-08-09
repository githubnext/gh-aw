// @ts-check

import { describe, it, expect, beforeEach, vi } from "vitest";
import { getIssueDetails, getPullRequestDetails, addIssueThreadComment, closeIssue, closePullRequest } from "./close_rest_helpers.cjs";

describe("close_rest_helpers", () => {
  let mockGithub;

  beforeEach(() => {
    vi.clearAllMocks();
    mockGithub = {
      rest: {
        issues: {
          get: vi.fn(),
          createComment: vi.fn(),
          update: vi.fn(),
        },
        pulls: {
          get: vi.fn(),
          update: vi.fn(),
        },
      },
    };
  });

  describe("getIssueDetails", () => {
    it("fetches the issue and returns its data", async () => {
      mockGithub.rest.issues.get.mockResolvedValue({ data: { number: 123, title: "Test issue" } });

      const result = await getIssueDetails(mockGithub, "owner", "repo", 123);

      expect(mockGithub.rest.issues.get).toHaveBeenCalledWith({ owner: "owner", repo: "repo", issue_number: 123 });
      expect(result).toEqual({ number: 123, title: "Test issue" });
    });

    it("throws a not-found error when the issue is missing", async () => {
      mockGithub.rest.issues.get.mockResolvedValue({ data: null });

      await expect(getIssueDetails(mockGithub, "owner", "repo", 123)).rejects.toThrow("ERR_NOT_FOUND: Issue #123 not found in owner/repo");
    });
  });

  describe("getPullRequestDetails", () => {
    it("fetches the pull request and returns its data", async () => {
      mockGithub.rest.pulls.get.mockResolvedValue({ data: { number: 42, title: "Test PR" } });

      const result = await getPullRequestDetails(mockGithub, "owner", "repo", 42);

      expect(mockGithub.rest.pulls.get).toHaveBeenCalledWith({ owner: "owner", repo: "repo", pull_number: 42 });
      expect(result).toEqual({ number: 42, title: "Test PR" });
    });

    it("throws a not-found error when the pull request is missing", async () => {
      mockGithub.rest.pulls.get.mockResolvedValue({ data: null });

      await expect(getPullRequestDetails(mockGithub, "owner", "repo", 42)).rejects.toThrow("ERR_NOT_FOUND: Pull request #42 not found in owner/repo");
    });
  });

  describe("addIssueThreadComment", () => {
    it("creates a comment on the issue thread without modifying the body", async () => {
      mockGithub.rest.issues.createComment.mockResolvedValue({ data: { id: 1, html_url: "https://github.com/owner/repo/issues/123#issuecomment-1" } });

      const result = await addIssueThreadComment(mockGithub, "owner", "repo", 123, "Hello");

      expect(mockGithub.rest.issues.createComment).toHaveBeenCalledWith({ owner: "owner", repo: "repo", issue_number: 123, body: "Hello" });
      expect(result).toEqual({ id: 1, html_url: "https://github.com/owner/repo/issues/123#issuecomment-1" });
    });
  });

  describe("closeIssue", () => {
    it("closes the issue with the provided state reason", async () => {
      mockGithub.rest.issues.update.mockResolvedValue({ data: { number: 123, html_url: "https://github.com/owner/repo/issues/123" } });

      const result = await closeIssue(mockGithub, "owner", "repo", 123, "completed");

      expect(mockGithub.rest.issues.update).toHaveBeenCalledWith({
        owner: "owner",
        repo: "repo",
        issue_number: 123,
        state: "closed",
        state_reason: "completed",
      });
      expect(result).toEqual({ number: 123, html_url: "https://github.com/owner/repo/issues/123" });
    });

    it("defaults the state reason to not_planned", async () => {
      mockGithub.rest.issues.update.mockResolvedValue({ data: { number: 123 } });

      await closeIssue(mockGithub, "owner", "repo", 123);

      expect(mockGithub.rest.issues.update).toHaveBeenCalledWith(expect.objectContaining({ state_reason: "not_planned" }));
    });
  });

  describe("closePullRequest", () => {
    it("closes the pull request", async () => {
      mockGithub.rest.pulls.update.mockResolvedValue({ data: { number: 42, html_url: "https://github.com/owner/repo/pull/42" } });

      const result = await closePullRequest(mockGithub, "owner", "repo", 42);

      expect(mockGithub.rest.pulls.update).toHaveBeenCalledWith({ owner: "owner", repo: "repo", pull_number: 42, state: "closed" });
      expect(result).toEqual({ number: 42, html_url: "https://github.com/owner/repo/pull/42" });
    });
  });
});
