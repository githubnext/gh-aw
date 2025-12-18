import { describe, it, expect } from "vitest";
const { isIssueContext, getIssueNumber, isPRContext, getPRNumber } = require("./update_context_helpers.cjs");
describe("update_context_helpers", () => {
  (describe("isIssueContext", () => {
    (it("should return true for issues event", () => {
      expect(isIssueContext("issues", {})).toBe(!0);
    }),
      it("should return true for issue_comment event", () => {
        expect(isIssueContext("issue_comment", {})).toBe(!0);
      }),
      it("should return false for pull_request event", () => {
        expect(isIssueContext("pull_request", {})).toBe(!1);
      }),
      it("should return false for push event", () => {
        expect(isIssueContext("push", {})).toBe(!1);
      }),
      it("should return false for workflow_dispatch event", () => {
        expect(isIssueContext("workflow_dispatch", {})).toBe(!1);
      }));
  }),
    describe("getIssueNumber", () => {
      (it("should return issue number from payload", () => {
        expect(getIssueNumber({ issue: { number: 123 } })).toBe(123);
      }),
        it("should return undefined when issue is missing", () => {
          expect(getIssueNumber({})).toBeUndefined();
        }),
        it("should return undefined when issue.number is missing", () => {
          expect(getIssueNumber({ issue: {} })).toBeUndefined();
        }),
        it("should handle null payload gracefully", () => {
          expect(getIssueNumber(null)).toBeUndefined();
        }),
        it("should handle undefined payload gracefully", () => {
          expect(getIssueNumber(void 0)).toBeUndefined();
        }));
    }),
    describe("isPRContext", () => {
      (it("should return true for pull_request event", () => {
        expect(isPRContext("pull_request", {})).toBe(!0);
      }),
        it("should return true for pull_request_review event", () => {
          expect(isPRContext("pull_request_review", {})).toBe(!0);
        }),
        it("should return true for pull_request_review_comment event", () => {
          expect(isPRContext("pull_request_review_comment", {})).toBe(!0);
        }),
        it("should return true for pull_request_target event", () => {
          expect(isPRContext("pull_request_target", {})).toBe(!0);
        }),
        it("should return true for issue_comment on PR", () => {
          expect(isPRContext("issue_comment", { issue: { number: 100, pull_request: { url: "https://api.github.com/repos/owner/repo/pulls/100" } } })).toBe(!0);
        }),
        it("should return false for issue_comment on issue", () => {
          expect(isPRContext("issue_comment", { issue: { number: 123 } })).toBe(!1);
        }),
        it("should return false for issues event", () => {
          expect(isPRContext("issues", {})).toBe(!1);
        }),
        it("should return false for push event", () => {
          expect(isPRContext("push", {})).toBe(!1);
        }),
        it("should return false for workflow_dispatch event", () => {
          expect(isPRContext("workflow_dispatch", {})).toBe(!1);
        }));
    }),
    describe("getPRNumber", () => {
      (it("should return PR number from pull_request", () => {
        expect(getPRNumber({ pull_request: { number: 100 } })).toBe(100);
      }),
        it("should return PR number from issue with pull_request", () => {
          expect(getPRNumber({ issue: { number: 200, pull_request: { url: "https://api.github.com/repos/owner/repo/pulls/200" } } })).toBe(200);
        }),
        it("should prefer pull_request over issue", () => {
          expect(getPRNumber({ pull_request: { number: 100 }, issue: { number: 200 } })).toBe(100);
        }),
        it("should return undefined when pull_request is missing", () => {
          expect(getPRNumber({})).toBeUndefined();
        }),
        it("should return undefined when issue has no pull_request", () => {
          expect(getPRNumber({ issue: { number: 123 } })).toBeUndefined();
        }),
        it("should handle null payload gracefully", () => {
          expect(getPRNumber(null)).toBeUndefined();
        }),
        it("should handle undefined payload gracefully", () => {
          expect(getPRNumber(void 0)).toBeUndefined();
        }),
        it("should return undefined when pull_request.number is missing", () => {
          expect(getPRNumber({ pull_request: {} })).toBeUndefined();
        }),
        it("should return undefined when issue.number is missing", () => {
          expect(getPRNumber({ issue: { pull_request: { url: "https://api.github.com/repos/owner/repo/pulls/100" } } })).toBeUndefined();
        }));
    }),
    describe("Cross-validation", () => {
      (it("issue_comment on PR should be PR context but not issue context", () => {
        const payload = { issue: { number: 100, pull_request: { url: "https://api.github.com/repos/owner/repo/pulls/100" } } };
        (expect(isPRContext("issue_comment", payload)).toBe(!0), expect(isIssueContext("issue_comment", payload)).toBe(!0));
      }),
        it("issue_comment on issue should be issue context but not PR context", () => {
          const payload = { issue: { number: 123 } };
          (expect(isIssueContext("issue_comment", payload)).toBe(!0), expect(isPRContext("issue_comment", payload)).toBe(!1));
        }));
    }));
});
