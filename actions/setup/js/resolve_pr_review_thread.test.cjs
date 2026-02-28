import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

global.core = mockCore;

const mockGraphql = vi.fn();
const mockGithub = {
  graphql: mockGraphql,
};

global.github = mockGithub;

const mockContext = {
  repo: { owner: "test-owner", repo: "test-repo" },
  runId: 12345,
  eventName: "pull_request",
  payload: {
    pull_request: { number: 42 },
    repository: { html_url: "https://github.com/test-owner/test-repo" },
  },
};

global.context = mockContext;

/**
 * Helper to set up mockGraphql to handle both the lookup query and the resolve mutation.
 * @param {number} lookupPRNumber - PR number returned by the thread lookup query
 * @param {string} [lookupRepo] - Repository nameWithOwner returned by the lookup query (default: "test-owner/test-repo")
 */
function mockGraphqlForThread(lookupPRNumber, lookupRepo = "test-owner/test-repo") {
  mockGraphql.mockImplementation(query => {
    if (query.includes("resolveReviewThread")) {
      // Mutation
      return Promise.resolve({
        resolveReviewThread: {
          thread: {
            id: "PRRT_kwDOABCD123456",
            isResolved: true,
          },
        },
      });
    }
    // Lookup query
    return Promise.resolve({
      node: {
        pullRequest: {
          number: lookupPRNumber,
          repository: { nameWithOwner: lookupRepo },
        },
      },
    });
  });
}

describe("resolve_pr_review_thread", () => {
  let handler;
  const originalPayload = mockContext.payload;

  beforeEach(async () => {
    vi.resetModules();
    vi.clearAllMocks();

    // Default: thread belongs to triggering PR #42
    mockGraphqlForThread(42);

    const { main } = require("./resolve_pr_review_thread.cjs");
    handler = await main({ max: 10 });
  });

  afterEach(() => {
    // Always restore the global context payload, even if assertions threw
    global.context.payload = originalPayload;
  });

  it("should return a function from main()", async () => {
    const { main } = require("./resolve_pr_review_thread.cjs");
    const result = await main({});
    expect(typeof result).toBe("function");
  });

  it("should successfully resolve a review thread on the triggering PR", async () => {
    const message = {
      type: "resolve_pull_request_review_thread",
      thread_id: "PRRT_kwDOABCD123456",
    };

    const result = await handler(message, {});

    expect(result.success).toBe(true);
    expect(result.thread_id).toBe("PRRT_kwDOABCD123456");
    expect(result.is_resolved).toBe(true);
    // Should have made two GraphQL calls: lookup + resolve
    expect(mockGraphql).toHaveBeenCalledTimes(2);
    expect(mockGraphql).toHaveBeenCalledWith(expect.stringContaining("resolveReviewThread"), expect.objectContaining({ threadId: "PRRT_kwDOABCD123456" }));
  });

  it("should reject a thread that belongs to a different PR", async () => {
    // Thread belongs to PR #99, not triggering PR #42
    mockGraphqlForThread(99);

    const { main } = require("./resolve_pr_review_thread.cjs");
    const freshHandler = await main({ max: 10 });

    const message = {
      type: "resolve_pull_request_review_thread",
      thread_id: "PRRT_kwDOOtherThread",
    };

    const result = await freshHandler(message, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("PR #99");
    expect(result.error).toContain("triggering PR #42");
  });

  it("should reject when thread is not found", async () => {
    mockGraphql.mockImplementation(query => {
      if (query.includes("resolveReviewThread")) {
        return Promise.resolve({});
      }
      // Lookup returns null node
      return Promise.resolve({ node: null });
    });

    const { main } = require("./resolve_pr_review_thread.cjs");
    const freshHandler = await main({ max: 10 });

    const message = {
      type: "resolve_pull_request_review_thread",
      thread_id: "PRRT_invalid",
    };

    const result = await freshHandler(message, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("not found");
  });

  it("should reject when not in a pull request context", async () => {
    // Override context to non-PR event (afterEach restores the original payload)
    global.context.payload = {
      repository: { html_url: "https://github.com/test-owner/test-repo" },
    };

    const { main } = require("./resolve_pr_review_thread.cjs");
    const freshHandler = await main({ max: 10 });

    const message = {
      type: "resolve_pull_request_review_thread",
      thread_id: "PRRT_kwDOABCD123456",
    };

    const result = await freshHandler(message, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("pull request context");
  });

  it("should fail when thread_id is missing", async () => {
    const message = {
      type: "resolve_pull_request_review_thread",
    };

    const result = await handler(message, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("thread_id");
  });

  it("should fail when thread_id is empty string", async () => {
    const message = {
      type: "resolve_pull_request_review_thread",
      thread_id: "",
    };

    const result = await handler(message, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("thread_id");
  });

  it("should fail when thread_id is whitespace only", async () => {
    const message = {
      type: "resolve_pull_request_review_thread",
      thread_id: "   ",
    };

    const result = await handler(message, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("thread_id");
  });

  it("should fail when thread_id is not a string", async () => {
    const message = {
      type: "resolve_pull_request_review_thread",
      thread_id: 12345,
    };

    const result = await handler(message, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("thread_id");
  });

  it("should respect max count limit", async () => {
    const { main } = require("./resolve_pr_review_thread.cjs");
    const limitedHandler = await main({ max: 2 });

    const message = {
      type: "resolve_pull_request_review_thread",
      thread_id: "PRRT_kwDOABCD123456",
    };

    const result1 = await limitedHandler(message, {});
    const result2 = await limitedHandler(message, {});
    const result3 = await limitedHandler(message, {});

    expect(result1.success).toBe(true);
    expect(result2.success).toBe(true);
    expect(result3.success).toBe(false);
    expect(result3.error).toContain("Max count of 2 reached");
  });

  it("should handle API errors gracefully", async () => {
    mockGraphql.mockRejectedValue(new Error("Could not resolve. Thread not found."));

    const message = {
      type: "resolve_pull_request_review_thread",
      thread_id: "PRRT_invalid",
    };

    const result = await handler(message, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("Could not resolve");
  });

  it("should handle unexpected resolve failure", async () => {
    mockGraphql.mockImplementation(query => {
      if (query.includes("resolveReviewThread")) {
        return Promise.resolve({
          resolveReviewThread: {
            thread: {
              id: "PRRT_kwDOABCD123456",
              isResolved: false,
            },
          },
        });
      }
      // Lookup succeeds - thread is on triggering PR
      return Promise.resolve({
        node: { pullRequest: { number: 42 } },
      });
    });

    const { main } = require("./resolve_pr_review_thread.cjs");
    const freshHandler = await main({ max: 10 });

    const message = {
      type: "resolve_pull_request_review_thread",
      thread_id: "PRRT_kwDOABCD123456",
    };

    const result = await freshHandler(message, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("Failed to resolve");
  });

  it("should default max to 10", async () => {
    const { main } = require("./resolve_pr_review_thread.cjs");
    const defaultHandler = await main({});

    const message = {
      type: "resolve_pull_request_review_thread",
      thread_id: "PRRT_kwDOABCD123456",
    };

    // Process 10 messages successfully
    for (let i = 0; i < 10; i++) {
      const result = await defaultHandler(message, {});
      expect(result.success).toBe(true);
    }

    // 11th should fail
    const result = await defaultHandler(message, {});
    expect(result.success).toBe(false);
    expect(result.error).toContain("Max count of 10 reached");
  });

  it("should work when triggered from issue_comment on a PR", async () => {
    // Simulate issue_comment event on a PR (afterEach restores the original payload)
    global.context.payload = {
      issue: { number: 42, pull_request: { url: "https://api.github.com/..." } },
      repository: { html_url: "https://github.com/test-owner/test-repo" },
    };

    const { main } = require("./resolve_pr_review_thread.cjs");
    const freshHandler = await main({ max: 10 });

    const message = {
      type: "resolve_pull_request_review_thread",
      thread_id: "PRRT_kwDOABCD123456",
    };

    const result = await freshHandler(message, {});

    expect(result.success).toBe(true);
  });
});

describe("resolve_pr_review_thread - cross-repo support", () => {
  const originalPayload = mockContext.payload;

  beforeEach(() => {
    vi.resetModules();
    vi.clearAllMocks();
  });

  afterEach(() => {
    global.context.payload = originalPayload;
  });

  it("should allow resolving a thread in target-repo when configured", async () => {
    mockGraphqlForThread(10, "other-owner/other-repo");

    const { main } = require("./resolve_pr_review_thread.cjs");
    const freshHandler = await main({
      max: 10,
      "target-repo": "other-owner/other-repo",
      target: "*",
    });

    const message = {
      type: "resolve_pull_request_review_thread",
      thread_id: "PRRT_kwDOCrossRepo",
    };

    const result = await freshHandler(message, {});

    expect(result.success).toBe(true);
    expect(result.thread_id).toBe("PRRT_kwDOCrossRepo");
    expect(result.is_resolved).toBe(true);
  });

  it("should reject a thread whose repo is not in allowed-repos", async () => {
    mockGraphqlForThread(10, "other-owner/other-repo");

    const { main } = require("./resolve_pr_review_thread.cjs");
    const freshHandler = await main({
      max: 10,
      "target-repo": "allowed-owner/allowed-repo",
      target: "*",
    });

    const message = {
      type: "resolve_pull_request_review_thread",
      thread_id: "PRRT_kwDOCrossRepo",
    };

    const result = await freshHandler(message, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("other-owner/other-repo");
    expect(result.error).toContain("allowed");
  });

  it("should allow cross-repo thread in allowed_repos list", async () => {
    mockGraphqlForThread(10, "extra-owner/extra-repo");

    const { main } = require("./resolve_pr_review_thread.cjs");
    const freshHandler = await main({
      max: 10,
      "target-repo": "allowed-owner/allowed-repo",
      allowed_repos: ["extra-owner/extra-repo"],
      target: "*",
    });

    const message = {
      type: "resolve_pull_request_review_thread",
      thread_id: "PRRT_kwDOCrossRepo",
    };

    const result = await freshHandler(message, {});

    expect(result.success).toBe(true);
  });

  it("should reject thread not on target PR when target is an explicit PR number", async () => {
    mockGraphqlForThread(99, "other-owner/other-repo");

    const { main } = require("./resolve_pr_review_thread.cjs");
    const freshHandler = await main({
      max: 10,
      "target-repo": "other-owner/other-repo",
      target: "55",
    });

    const message = {
      type: "resolve_pull_request_review_thread",
      thread_id: "PRRT_kwDOCrossRepo",
    };

    const result = await freshHandler(message, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("PR #99");
    expect(result.error).toContain("PR #55");
  });

  it("should allow thread on correct explicit target PR", async () => {
    mockGraphqlForThread(55, "other-owner/other-repo");

    const { main } = require("./resolve_pr_review_thread.cjs");
    const freshHandler = await main({
      max: 10,
      "target-repo": "other-owner/other-repo",
      target: "55",
    });

    const message = {
      type: "resolve_pull_request_review_thread",
      thread_id: "PRRT_kwDOCrossRepo",
    };

    const result = await freshHandler(message, {});

    expect(result.success).toBe(true);
  });

  it("should require triggering PR context when target=triggering with cross-repo config", async () => {
    global.context.payload = {
      repository: { html_url: "https://github.com/test-owner/test-repo" },
    };
    mockGraphqlForThread(10, "other-owner/other-repo");

    const { main } = require("./resolve_pr_review_thread.cjs");
    const freshHandler = await main({
      max: 10,
      "target-repo": "other-owner/other-repo",
      target: "triggering",
    });

    const message = {
      type: "resolve_pull_request_review_thread",
      thread_id: "PRRT_kwDOCrossRepo",
    };

    const result = await freshHandler(message, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("pull request context");
  });

  it("should allow resolving when allowed_repos uses '*' wildcard", async () => {
    mockGraphqlForThread(10, "any-owner/any-repo");

    const { main } = require("./resolve_pr_review_thread.cjs");
    const freshHandler = await main({
      max: 10,
      "target-repo": "default-owner/default-repo",
      allowed_repos: ["*"],
      target: "*",
    });

    const message = {
      type: "resolve_pull_request_review_thread",
      thread_id: "PRRT_kwDOWildcard",
    };

    const result = await freshHandler(message, {});

    expect(result.success).toBe(true);
  });

  it("should allow resolving when allowed_repos uses org/* pattern", async () => {
    mockGraphqlForThread(10, "other-owner/specific-repo");

    const { main } = require("./resolve_pr_review_thread.cjs");
    const freshHandler = await main({
      max: 10,
      "target-repo": "default-owner/default-repo",
      allowed_repos: ["other-owner/*"],
      target: "*",
    });

    const message = {
      type: "resolve_pull_request_review_thread",
      thread_id: "PRRT_kwDOOrgWildcard",
    };

    const result = await freshHandler(message, {});

    expect(result.success).toBe(true);
  });

  it("should reject resolving when org/* pattern does not match", async () => {
    mockGraphqlForThread(10, "wrong-owner/specific-repo");

    const { main } = require("./resolve_pr_review_thread.cjs");
    const freshHandler = await main({
      max: 10,
      "target-repo": "default-owner/default-repo",
      allowed_repos: ["other-owner/*"],
      target: "*",
    });

    const message = {
      type: "resolve_pull_request_review_thread",
      thread_id: "PRRT_kwDOOrgWildcard",
    };

    const result = await freshHandler(message, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("not in the allowed-repos list");
  });

  it("should fail closed when threadRepo is null in cross-repo mode", async () => {
    // Simulate GraphQL returning a thread with no repository info
    mockGraphql.mockImplementation(query => {
      if (query.includes("resolveReviewThread")) {
        return Promise.resolve({ resolveReviewThread: { thread: { id: "PRRT_x", isResolved: true } } });
      }
      return Promise.resolve({
        node: { pullRequest: { number: 10, repository: null } },
      });
    });

    const { main } = require("./resolve_pr_review_thread.cjs");
    const freshHandler = await main({
      max: 10,
      "target-repo": "other-owner/other-repo",
      target: "*",
    });

    const message = {
      type: "resolve_pull_request_review_thread",
      thread_id: "PRRT_kwDONoRepo",
    };

    const result = await freshHandler(message, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("Could not determine");
  });
});

describe("getPRNumber (shared helper in update_context_helpers)", () => {
  it("should return pull_request.number for pull_request events", () => {
    const { getPRNumber } = require("./update_context_helpers.cjs");
    const payload = { pull_request: { number: 7 } };
    expect(getPRNumber(payload)).toBe(7);
  });

  it("should return issue.number for issue_comment events on a PR", () => {
    const { getPRNumber } = require("./update_context_helpers.cjs");
    const payload = { issue: { number: 15, pull_request: { url: "https://api.github.com/..." } } };
    expect(getPRNumber(payload)).toBe(15);
  });

  it("should return undefined when payload has no PR context", () => {
    const { getPRNumber } = require("./update_context_helpers.cjs");
    const payload = { repository: { html_url: "https://github.com/owner/repo" } };
    expect(getPRNumber(payload)).toBeUndefined();
  });

  it("should return undefined for an empty payload", () => {
    const { getPRNumber } = require("./update_context_helpers.cjs");
    expect(getPRNumber({})).toBeUndefined();
  });

  it("should return undefined for a nullish payload", () => {
    const { getPRNumber } = require("./update_context_helpers.cjs");
    expect(getPRNumber(null)).toBeUndefined();
    expect(getPRNumber(undefined)).toBeUndefined();
  });

  it("should prefer pull_request.number over issue.number", () => {
    const { getPRNumber } = require("./update_context_helpers.cjs");
    const payload = {
      pull_request: { number: 10 },
      issue: { number: 20, pull_request: { url: "https://api.github.com/..." } },
    };
    expect(getPRNumber(payload)).toBe(10);
  });

  it("should not return issue.number when issue has no pull_request field", () => {
    const { getPRNumber } = require("./update_context_helpers.cjs");
    const payload = { issue: { number: 30 } };
    expect(getPRNumber(payload)).toBeUndefined();
  });
});
