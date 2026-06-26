// @ts-check
import { describe, it, expect } from "vitest";
const { isIssueContext, getIssueNumber, isPRContext, getPRNumber, isDiscussionContext, getDiscussionNumber } = require("./update_context_helpers.cjs");

/** Reusable PR payload for issue_comment on a PR */
const PR_COMMENT_PAYLOAD = {
  issue: { number: 100, pull_request: { url: "https://api.github.com/repos/owner/repo/pulls/100" } },
};

describe("update_context_helpers", () => {
  describe("isIssueContext", () => {
    it("returns true for issues event", () => {
      expect(isIssueContext("issues", {})).toBe(true);
    });
    it("returns true for issue_comment event", () => {
      expect(isIssueContext("issue_comment", {})).toBe(true);
    });
    it("returns false for pull_request event", () => {
      expect(isIssueContext("pull_request", {})).toBe(false);
    });
    it("returns false for push event", () => {
      expect(isIssueContext("push", {})).toBe(false);
    });
    it("returns false for workflow_dispatch event", () => {
      expect(isIssueContext("workflow_dispatch", {})).toBe(false);
    });
  });

  describe("getIssueNumber", () => {
    it("returns issue number from payload", () => {
      expect(getIssueNumber({ issue: { number: 123 } })).toBe(123);
    });
    it("returns undefined when issue is missing", () => {
      expect(getIssueNumber({})).toBeUndefined();
    });
    it("returns undefined when issue.number is missing", () => {
      expect(getIssueNumber({ issue: {} })).toBeUndefined();
    });
    it("handles null payload gracefully", () => {
      expect(getIssueNumber(null)).toBeUndefined();
    });
    it("handles undefined payload gracefully", () => {
      expect(getIssueNumber(undefined)).toBeUndefined();
    });
  });

  describe("isPRContext", () => {
    it("returns true for pull_request event", () => {
      expect(isPRContext("pull_request", {})).toBe(true);
    });
    it("returns true for pull_request_review event", () => {
      expect(isPRContext("pull_request_review", {})).toBe(true);
    });
    it("returns true for pull_request_review_comment event", () => {
      expect(isPRContext("pull_request_review_comment", {})).toBe(true);
    });
    it("returns true for pull_request_target event", () => {
      expect(isPRContext("pull_request_target", {})).toBe(true);
    });
    it("returns true for issue_comment on PR", () => {
      expect(isPRContext("issue_comment", PR_COMMENT_PAYLOAD)).toBe(true);
    });
    it("returns false for issue_comment on issue", () => {
      expect(isPRContext("issue_comment", { issue: { number: 123 } })).toBe(false);
    });
    it("returns false for issue_comment with null payload", () => {
      expect(isPRContext("issue_comment", null)).toBe(false);
    });
    it("returns false for issue_comment with undefined payload", () => {
      expect(isPRContext("issue_comment", undefined)).toBe(false);
    });
    it("returns false for issue_comment with empty payload", () => {
      expect(isPRContext("issue_comment", {})).toBe(false);
    });
    it("returns false for issues event", () => {
      expect(isPRContext("issues", {})).toBe(false);
    });
    it("returns false for push event", () => {
      expect(isPRContext("push", {})).toBe(false);
    });
    it("returns false for workflow_dispatch event", () => {
      expect(isPRContext("workflow_dispatch", {})).toBe(false);
    });
  });

  describe("getPRNumber", () => {
    it("returns PR number from pull_request", () => {
      expect(getPRNumber({ pull_request: { number: 100 } })).toBe(100);
    });
    it("returns PR number from issue with pull_request", () => {
      expect(getPRNumber({ issue: { number: 200, pull_request: { url: "https://api.github.com/repos/owner/repo/pulls/200" } } })).toBe(200);
    });
    it("prefers pull_request over issue when both present", () => {
      expect(getPRNumber({ pull_request: { number: 100 }, issue: { number: 200 } })).toBe(100);
    });
    it("returns undefined when pull_request is missing", () => {
      expect(getPRNumber({})).toBeUndefined();
    });
    it("returns undefined when issue has no pull_request", () => {
      expect(getPRNumber({ issue: { number: 123 } })).toBeUndefined();
    });
    it("handles null payload gracefully", () => {
      expect(getPRNumber(null)).toBeUndefined();
    });
    it("handles undefined payload gracefully", () => {
      expect(getPRNumber(undefined)).toBeUndefined();
    });
    it("returns undefined when pull_request.number is missing", () => {
      expect(getPRNumber({ pull_request: {} })).toBeUndefined();
    });
    it("returns undefined when issue.number is missing but pull_request exists", () => {
      expect(getPRNumber({ issue: { pull_request: { url: "https://api.github.com/repos/owner/repo/pulls/100" } } })).toBeUndefined();
    });
  });

  describe("isDiscussionContext", () => {
    it("returns true for discussion event", () => {
      expect(isDiscussionContext("discussion", {})).toBe(true);
    });
    it("returns true for discussion_comment event", () => {
      expect(isDiscussionContext("discussion_comment", {})).toBe(true);
    });
    it("returns false for issues event", () => {
      expect(isDiscussionContext("issues", {})).toBe(false);
    });
    it("returns false for pull_request event", () => {
      expect(isDiscussionContext("pull_request", {})).toBe(false);
    });
    it("returns false for push event", () => {
      expect(isDiscussionContext("push", {})).toBe(false);
    });
    it("returns false for workflow_dispatch event", () => {
      expect(isDiscussionContext("workflow_dispatch", {})).toBe(false);
    });
  });

  describe("getDiscussionNumber", () => {
    it("returns discussion number from payload", () => {
      expect(getDiscussionNumber({ discussion: { number: 42 } })).toBe(42);
    });
    it("returns undefined when discussion is missing", () => {
      expect(getDiscussionNumber({})).toBeUndefined();
    });
    it("returns undefined when discussion.number is missing", () => {
      expect(getDiscussionNumber({ discussion: {} })).toBeUndefined();
    });
    it("handles null payload gracefully", () => {
      expect(getDiscussionNumber(null)).toBeUndefined();
    });
    it("handles undefined payload gracefully", () => {
      expect(getDiscussionNumber(undefined)).toBeUndefined();
    });
  });

  describe("Cross-validation", () => {
    it("issue_comment on PR is both PR context and issue context", () => {
      expect(isPRContext("issue_comment", PR_COMMENT_PAYLOAD)).toBe(true);
      expect(isIssueContext("issue_comment", PR_COMMENT_PAYLOAD)).toBe(true);
    });
    it("issue_comment on plain issue is issue context but not PR context", () => {
      const payload = { issue: { number: 123 } };
      expect(isIssueContext("issue_comment", payload)).toBe(true);
      expect(isPRContext("issue_comment", payload)).toBe(false);
    });
    it("discussion event is only discussion context", () => {
      expect(isDiscussionContext("discussion", {})).toBe(true);
      expect(isIssueContext("discussion", {})).toBe(false);
      expect(isPRContext("discussion", {})).toBe(false);
    });
  });
});
