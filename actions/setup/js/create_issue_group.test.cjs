// @ts-check
/// <reference types="@actions/github-script" />

import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { createParentIssueTemplate, searchForExistingParent, getSubIssueCount } from "./create_issue.cjs";

describe("createParentIssueTemplate", () => {
  it("should create parent issue template with correct format", () => {
    const groupId = "test-workflow";
    const titlePrefix = "[Bot] ";
    const workflowName = "Test Workflow";
    const runUrl = "https://github.com/owner/repo/actions/runs/123";

    const result = createParentIssueTemplate(groupId, titlePrefix, workflowName, runUrl);

    expect(result.title).toBe("[Bot] test-workflow - Issue Group");
    expect(result.body).toContain("# test-workflow");
    expect(result.body).toContain("<!-- gh-aw-group: test-workflow -->");
    expect(result.body).toContain("- **Workflow**: Test Workflow");
    expect(result.body).toContain("- **Run**: https://github.com/owner/repo/actions/runs/123");
  });

  it("should handle empty title prefix", () => {
    const groupId = "test-workflow";
    const titlePrefix = "";
    const workflowName = "Test Workflow";
    const runUrl = "https://github.com/owner/repo/actions/runs/123";

    const result = createParentIssueTemplate(groupId, titlePrefix, workflowName, runUrl);

    expect(result.title).toBe("test-workflow - Issue Group");
  });

  it("should include group marker in body", () => {
    const groupId = "my-special-workflow";
    const titlePrefix = "";
    const workflowName = "My Workflow";
    const runUrl = "https://example.com/run/1";

    const result = createParentIssueTemplate(groupId, titlePrefix, workflowName, runUrl);

    expect(result.body).toContain("<!-- gh-aw-group: my-special-workflow -->");
  });
});

describe("searchForExistingParent", () => {
  let mockGithub;
  let mockCore;

  beforeEach(() => {
    // Create mock objects
    mockCore = {
      info: vi.fn(),
      warning: vi.fn(),
    };

    mockGithub = {
      rest: {
        search: {
          issuesAndPullRequests: vi.fn().mockResolvedValue({
            data: {
              total_count: 0,
              items: [],
            },
          }),
        },
      },
      graphql: vi.fn().mockResolvedValue({
        repository: {
          issue: {
            subIssues: {
              totalCount: 0,
            },
          },
        },
      }),
    };

    // Set global mocks
    global.github = mockGithub;
    global.core = mockCore;
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("should return null when no parent issues found", async () => {
    const result = await searchForExistingParent("owner", "repo", "<!-- gh-aw-group: test -->");

    expect(result).toBeNull();
  });

  it("should return issue number when open parent with available slots found", async () => {
    mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
      data: {
        total_count: 1,
        items: [
          {
            number: 42,
            title: "Parent Issue",
            state: "open",
          },
        ],
      },
    });

    mockGithub.graphql.mockResolvedValue({
      repository: {
        issue: {
          subIssues: {
            totalCount: 30,
          },
        },
      },
    });

    const result = await searchForExistingParent("owner", "repo", "<!-- gh-aw-group: test -->");

    expect(result).toBe(42);
  });

  it("should skip closed parent issues", async () => {
    mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
      data: {
        total_count: 1,
        items: [
          {
            number: 42,
            title: "Closed Parent",
            state: "closed",
          },
        ],
      },
    });

    const result = await searchForExistingParent("owner", "repo", "<!-- gh-aw-group: test -->");

    expect(result).toBeNull();
  });

  it("should skip full parent issues (64 sub-issues)", async () => {
    mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
      data: {
        total_count: 1,
        items: [
          {
            number: 42,
            title: "Full Parent",
            state: "open",
          },
        ],
      },
    });

    mockGithub.graphql.mockResolvedValue({
      repository: {
        issue: {
          subIssues: {
            totalCount: 64,
          },
        },
      },
    });

    const result = await searchForExistingParent("owner", "repo", "<!-- gh-aw-group: test -->");

    expect(result).toBeNull();
  });

  it("should find first available parent when multiple exist", async () => {
    mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
      data: {
        total_count: 3,
        items: [
          { number: 1, title: "Parent 1", state: "closed" },
          { number: 2, title: "Parent 2", state: "open" },
          { number: 3, title: "Parent 3", state: "open" },
        ],
      },
    });

    let callCount = 0;
    mockGithub.graphql.mockImplementation(() => {
      callCount++;
      return Promise.resolve({
        repository: {
          issue: {
            subIssues: {
              totalCount: 10,
            },
          },
        },
      });
    });

    const result = await searchForExistingParent("owner", "repo", "<!-- gh-aw-group: test -->");

    expect(result).toBe(2); // Should skip closed parent and return first open one
  });
});

describe("getSubIssueCount", () => {
  let mockGithub;
  let mockCore;

  beforeEach(() => {
    mockCore = {
      warning: vi.fn(),
    };

    mockGithub = {
      graphql: vi.fn().mockResolvedValue({
        repository: {
          issue: {
            subIssues: {
              totalCount: 0,
            },
          },
        },
      }),
    };

    global.github = mockGithub;
    global.core = mockCore;
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

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

    const result = await getSubIssueCount("owner", "repo", 42);

    expect(result).toBe(25);
  });

  it("should return 0 when no sub-issues exist", async () => {
    mockGithub.graphql.mockResolvedValue({
      repository: {
        issue: {
          subIssues: {
            totalCount: 0,
          },
        },
      },
    });

    const result = await getSubIssueCount("owner", "repo", 42);

    expect(result).toBe(0);
  });

  it("should return null when GraphQL query fails", async () => {
    mockGithub.graphql.mockRejectedValue(new Error("GraphQL error"));

    const result = await getSubIssueCount("owner", "repo", 42);

    expect(result).toBeNull();
  });

  it("should handle missing data in GraphQL response", async () => {
    mockGithub.graphql.mockResolvedValue({
      repository: null,
    });

    const result = await getSubIssueCount("owner", "repo", 42);

    expect(result).toBe(0);
  });
});
