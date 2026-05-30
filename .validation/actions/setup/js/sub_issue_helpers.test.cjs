// @ts-check
/// <reference types="@actions/github-script" />

import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

describe("sub_issue_helpers.cjs", () => {
  let getSubIssueCount;
  let MAX_SUB_ISSUES;
  let mockCore;
  let mockGithub;

  beforeEach(async () => {
    // Mock core
    mockCore = {
      warning: vi.fn(),
    };
    global.core = mockCore;

    // Mock github
    mockGithub = {
      graphql: vi.fn(),
    };
    global.github = mockGithub;

    // Load the module
    const module = await import("./sub_issue_helpers.cjs");
    getSubIssueCount = module.getSubIssueCount;
    MAX_SUB_ISSUES = module.MAX_SUB_ISSUES;
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  describe("MAX_SUB_ISSUES", () => {
    it("should be 64", () => {
      expect(MAX_SUB_ISSUES).toBe(64);
    });
  });

  describe("getSubIssueCount", () => {
    it("should return sub-issue count from GraphQL", async () => {
      mockGithub.graphql.mockResolvedValue({
        repository: {
          issue: {
            subIssues: {
              totalCount: 25,
            },
          },
        },
      });

      const count = await getSubIssueCount("test-owner", "test-repo", 123);

      expect(count).toBe(25);
      expect(mockGithub.graphql).toHaveBeenCalledWith(expect.stringContaining("subIssues"), {
        owner: "test-owner",
        repo: "test-repo",
        issueNumber: 123,
      });
    });

    it("should return 0 when sub-issue count is 0", async () => {
      mockGithub.graphql.mockResolvedValue({
        repository: {
          issue: {
            subIssues: {
              totalCount: 0,
            },
          },
        },
      });

      const count = await getSubIssueCount("test-owner", "test-repo", 456);

      expect(count).toBe(0);
    });

    it("should return 0 when GraphQL response is malformed", async () => {
      mockGithub.graphql.mockResolvedValue({});

      const count = await getSubIssueCount("test-owner", "test-repo", 789);

      expect(count).toBe(0);
    });

    it("should return null and log warning when GraphQL query fails", async () => {
      mockGithub.graphql.mockRejectedValue(new Error("GraphQL API Error"));

      const count = await getSubIssueCount("test-owner", "test-repo", 999);

      expect(count).toBe(null);
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Could not check sub-issue count for #999"));
    });

    it("should use MAX_SUB_ISSUES + 1 in GraphQL query", async () => {
      mockGithub.graphql.mockResolvedValue({
        repository: {
          issue: {
            subIssues: {
              totalCount: 10,
            },
          },
        },
      });

      await getSubIssueCount("test-owner", "test-repo", 100);

      const queryCall = mockGithub.graphql.mock.calls[0][0];
      expect(queryCall).toContain(`subIssues(first: ${MAX_SUB_ISSUES + 1})`);
    });
  });
});
