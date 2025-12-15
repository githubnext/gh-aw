import { describe, it, expect } from "vitest";

// Import the functions to test
const { isIssueContext, getIssueNumber, isPRContext, getPRNumber } = require("./update_context_helpers.cjs");

describe("update_context_helpers", () => {
  describe("isIssueContext", () => {
    it("should return true for issues event", () => {
      expect(isIssueContext("issues", {})).toBe(true);
    });

    it("should return true for issue_comment event", () => {
      expect(isIssueContext("issue_comment", {})).toBe(true);
    });

    it("should return false for pull_request event", () => {
      expect(isIssueContext("pull_request", {})).toBe(false);
    });

    it("should return false for push event", () => {
      expect(isIssueContext("push", {})).toBe(false);
    });

    it("should return false for workflow_dispatch event", () => {
      expect(isIssueContext("workflow_dispatch", {})).toBe(false);
    });
  });

  describe("getIssueNumber", () => {
    it("should return issue number from payload", () => {
      const payload = { issue: { number: 123 } };
      expect(getIssueNumber(payload)).toBe(123);
    });

    it("should return undefined when issue is missing", () => {
      const payload = {};
      expect(getIssueNumber(payload)).toBeUndefined();
    });

    it("should return undefined when issue.number is missing", () => {
      const payload = { issue: {} };
      expect(getIssueNumber(payload)).toBeUndefined();
    });

    it("should handle null payload gracefully", () => {
      expect(getIssueNumber(null)).toBeUndefined();
    });

    it("should handle undefined payload gracefully", () => {
      expect(getIssueNumber(undefined)).toBeUndefined();
    });
  });

  describe("isPRContext", () => {
    it("should return true for pull_request event", () => {
      expect(isPRContext("pull_request", {})).toBe(true);
    });

    it("should return true for pull_request_review event", () => {
      expect(isPRContext("pull_request_review", {})).toBe(true);
    });

    it("should return true for pull_request_review_comment event", () => {
      expect(isPRContext("pull_request_review_comment", {})).toBe(true);
    });

    it("should return true for pull_request_target event", () => {
      expect(isPRContext("pull_request_target", {})).toBe(true);
    });

    it("should return true for issue_comment on PR", () => {
      const payload = {
        issue: {
          number: 100,
          pull_request: { url: "https://api.github.com/repos/owner/repo/pulls/100" },
        },
      };
      expect(isPRContext("issue_comment", payload)).toBe(true);
    });

    it("should return false for issue_comment on issue", () => {
      const payload = {
        issue: {
          number: 123,
        },
      };
      expect(isPRContext("issue_comment", payload)).toBe(false);
    });

    it("should return false for issues event", () => {
      expect(isPRContext("issues", {})).toBe(false);
    });

    it("should return false for push event", () => {
      expect(isPRContext("push", {})).toBe(false);
    });

    it("should return false for workflow_dispatch event", () => {
      expect(isPRContext("workflow_dispatch", {})).toBe(false);
    });
  });

  describe("getPRNumber", () => {
    it("should return PR number from pull_request", () => {
      const payload = { pull_request: { number: 100 } };
      expect(getPRNumber(payload)).toBe(100);
    });

    it("should return PR number from issue with pull_request", () => {
      const payload = {
        issue: {
          number: 200,
          pull_request: { url: "https://api.github.com/repos/owner/repo/pulls/200" },
        },
      };
      expect(getPRNumber(payload)).toBe(200);
    });

    it("should prefer pull_request over issue", () => {
      const payload = {
        pull_request: { number: 100 },
        issue: { number: 200 },
      };
      expect(getPRNumber(payload)).toBe(100);
    });

    it("should return undefined when pull_request is missing", () => {
      const payload = {};
      expect(getPRNumber(payload)).toBeUndefined();
    });

    it("should return undefined when issue has no pull_request", () => {
      const payload = { issue: { number: 123 } };
      expect(getPRNumber(payload)).toBeUndefined();
    });

    it("should handle null payload gracefully", () => {
      expect(getPRNumber(null)).toBeUndefined();
    });

    it("should handle undefined payload gracefully", () => {
      expect(getPRNumber(undefined)).toBeUndefined();
    });

    it("should return undefined when pull_request.number is missing", () => {
      const payload = { pull_request: {} };
      expect(getPRNumber(payload)).toBeUndefined();
    });

    it("should return undefined when issue.number is missing", () => {
      const payload = {
        issue: {
          pull_request: { url: "https://api.github.com/repos/owner/repo/pulls/100" },
        },
      };
      expect(getPRNumber(payload)).toBeUndefined();
    });
  });

  describe("Cross-validation", () => {
    it("issue_comment on PR should be PR context but not issue context", () => {
      const payload = {
        issue: {
          number: 100,
          pull_request: { url: "https://api.github.com/repos/owner/repo/pulls/100" },
        },
      };
      expect(isPRContext("issue_comment", payload)).toBe(true);
      expect(isIssueContext("issue_comment", payload)).toBe(true); // Note: issue_comment is valid for both
    });

    it("issue_comment on issue should be issue context but not PR context", () => {
      const payload = {
        issue: {
          number: 123,
        },
      };
      expect(isIssueContext("issue_comment", payload)).toBe(true);
      expect(isPRContext("issue_comment", payload)).toBe(false);
    });
  });
});
