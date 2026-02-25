import { describe, it, expect, vi, beforeEach } from "vitest";

describe("pr_helpers.cjs", () => {
  let detectForkPR;
  let getPullRequestNumber;

  // Import the helper before each test
  beforeEach(async () => {
    const helpers = await import("./pr_helpers.cjs");
    detectForkPR = helpers.detectForkPR;
    getPullRequestNumber = helpers.getPullRequestNumber;
  });

  describe("detectForkPR", () => {
    it("should detect fork using GitHub's fork flag", () => {
      const pullRequest = {
        head: {
          repo: {
            fork: true,
            full_name: "test-owner/test-repo",
          },
        },
        base: {
          repo: {
            full_name: "test-owner/test-repo",
          },
        },
      };

      const result = detectForkPR(pullRequest);

      expect(result.isFork).toBe(true);
      expect(result.reason).toBe("head.repo.fork flag is true");
    });

    it("should detect fork using different repository names", () => {
      const pullRequest = {
        head: {
          repo: {
            fork: false,
            full_name: "fork-owner/test-repo",
          },
        },
        base: {
          repo: {
            full_name: "original-owner/test-repo",
          },
        },
      };

      const result = detectForkPR(pullRequest);

      expect(result.isFork).toBe(true);
      expect(result.reason).toBe("different repository names");
    });

    it("should detect deleted fork (null head repo)", () => {
      const pullRequest = {
        head: {
          // repo is missing/null
        },
        base: {
          repo: {
            full_name: "original-owner/test-repo",
          },
        },
      };

      const result = detectForkPR(pullRequest);

      expect(result.isFork).toBe(true);
      expect(result.reason).toBe("head repository deleted (was likely a fork)");
    });

    it("should detect non-fork when repos match and fork flag is false", () => {
      const pullRequest = {
        head: {
          repo: {
            fork: false,
            full_name: "test-owner/test-repo",
          },
        },
        base: {
          repo: {
            full_name: "test-owner/test-repo",
          },
        },
      };

      const result = detectForkPR(pullRequest);

      expect(result.isFork).toBe(false);
      expect(result.reason).toBe("same repository");
    });

    it("should handle missing fork flag with same repo names", () => {
      const pullRequest = {
        head: {
          repo: {
            // fork flag not present
            full_name: "test-owner/test-repo",
          },
        },
        base: {
          repo: {
            full_name: "test-owner/test-repo",
          },
        },
      };

      const result = detectForkPR(pullRequest);

      expect(result.isFork).toBe(false);
      expect(result.reason).toBe("same repository");
    });

    it("should prioritize fork flag over repository name comparison", () => {
      // Edge case: fork flag is true even though names match
      // This could happen if a user forks and keeps the same name
      const pullRequest = {
        head: {
          repo: {
            fork: true,
            full_name: "test-owner/test-repo",
          },
        },
        base: {
          repo: {
            full_name: "test-owner/test-repo",
          },
        },
      };

      const result = detectForkPR(pullRequest);

      expect(result.isFork).toBe(true);
      expect(result.reason).toBe("head.repo.fork flag is true");
    });

    it("should handle null base repo gracefully", () => {
      const pullRequest = {
        head: {
          repo: {
            fork: false,
            full_name: "test-owner/test-repo",
          },
        },
        base: {
          // repo is missing/null
        },
      };

      const result = detectForkPR(pullRequest);

      // When base.repo is null, comparison with undefined returns true (different)
      expect(result.isFork).toBe(true);
      expect(result.reason).toBe("different repository names");
    });

    it("should handle both repos being null", () => {
      const pullRequest = {
        head: {
          // repo is missing/null
        },
        base: {
          // repo is missing/null
        },
      };

      const result = detectForkPR(pullRequest);

      // Deleted fork takes precedence
      expect(result.isFork).toBe(true);
      expect(result.reason).toBe("head repository deleted (was likely a fork)");
    });
  });

  describe("getPullRequestNumber", () => {
    it("should extract PR number from message", () => {
      const message = { pull_request_number: 123 };
      const context = { payload: {} };

      const result = getPullRequestNumber(message, context);

      expect(result.prNumber).toBe(123);
      expect(result.error).toBeNull();
    });

    it("should handle PR number as string", () => {
      const message = { pull_request_number: "456" };
      const context = { payload: {} };

      const result = getPullRequestNumber(message, context);

      expect(result.prNumber).toBe(456);
      expect(result.error).toBeNull();
    });

    it("should return error for invalid PR number", () => {
      const message = { pull_request_number: "invalid" };
      const context = { payload: {} };

      const result = getPullRequestNumber(message, context);

      expect(result.prNumber).toBeNull();
      expect(result.error).toBe("Invalid pull_request_number: invalid");
    });

    it("should return error for NaN PR number", () => {
      const message = { pull_request_number: NaN };
      const context = { payload: {} };

      const result = getPullRequestNumber(message, context);

      expect(result.prNumber).toBeNull();
      expect(result.error).toBe("Invalid pull_request_number: NaN");
    });

    it("should fall back to context when message has no PR number", () => {
      const message = {};
      const context = {
        payload: {
          pull_request: {
            number: 789,
          },
        },
      };

      const result = getPullRequestNumber(message, context);

      expect(result.prNumber).toBe(789);
      expect(result.error).toBeNull();
    });

    it("should fall back to context when message is undefined", () => {
      const context = {
        payload: {
          pull_request: {
            number: 101,
          },
        },
      };

      const result = getPullRequestNumber(undefined, context);

      expect(result.prNumber).toBe(101);
      expect(result.error).toBeNull();
    });

    it("should return error when no PR number is available", () => {
      const message = {};
      const context = { payload: {} };

      const result = getPullRequestNumber(message, context);

      expect(result.prNumber).toBeNull();
      expect(result.error).toBe("No pull_request_number provided and not in pull request context");
    });

    it("should prefer message PR number over context", () => {
      const message = { pull_request_number: 999 };
      const context = {
        payload: {
          pull_request: {
            number: 888,
          },
        },
      };

      const result = getPullRequestNumber(message, context);

      expect(result.prNumber).toBe(999);
      expect(result.error).toBeNull();
    });

    it("should handle context without payload", () => {
      const message = {};
      const context = {};

      const result = getPullRequestNumber(message, context);

      expect(result.prNumber).toBeNull();
      expect(result.error).toBe("No pull_request_number provided and not in pull request context");
    });

    it("should handle zero as a valid PR number from message", () => {
      const message = { pull_request_number: 0 };
      const context = { payload: {} };

      const result = getPullRequestNumber(message, context);

      expect(result.prNumber).toBe(0);
      expect(result.error).toBeNull();
    });
  });
});

describe("resolvePullRequestRepo", () => {
  const { resolvePullRequestRepo } = require("./pr_helpers.cjs");

  it("returns repoId, effectiveBaseBranch from explicit config, and resolvedDefaultBranch", async () => {
    const fakeGithub = {
      graphql: vi.fn().mockResolvedValue({ repository: { id: "repo-id", defaultBranchRef: { name: "develop" } } }),
    };
    const result = await resolvePullRequestRepo(fakeGithub, "owner", "repo", "feature");
    expect(result.repoId).toBe("repo-id");
    expect(result.resolvedDefaultBranch).toBe("develop");
    // explicit config wins over fetched default
    expect(result.effectiveBaseBranch).toBe("feature");
  });

  it("falls back to repo default branch when no explicit base branch configured", async () => {
    const fakeGithub = {
      graphql: vi.fn().mockResolvedValue({ repository: { id: "repo-id", defaultBranchRef: { name: "trunk" } } }),
    };
    const result = await resolvePullRequestRepo(fakeGithub, "owner", "repo", undefined);
    expect(result.repoId).toBe("repo-id");
    expect(result.resolvedDefaultBranch).toBe("trunk");
    expect(result.effectiveBaseBranch).toBe("trunk");
  });

  it("handles missing defaultBranchRef gracefully", async () => {
    const fakeGithub = {
      graphql: vi.fn().mockResolvedValue({ repository: { id: "repo-id", defaultBranchRef: null } }),
    };
    const result = await resolvePullRequestRepo(fakeGithub, "owner", "repo", undefined);
    expect(result.repoId).toBe("repo-id");
    expect(result.resolvedDefaultBranch).toBeNull();
    expect(result.effectiveBaseBranch).toBeNull();
  });
});

describe("buildBranchInstruction", () => {
  const { buildBranchInstruction } = require("./pr_helpers.cjs");

  it("produces a plain instruction when effective branch equals resolved default", () => {
    const instruction = buildBranchInstruction("main", "main");
    expect(instruction).toBe("IMPORTANT: Create your branch from the 'main' branch.");
    expect(instruction).not.toContain("NOT from");
  });

  it("includes NOT clause when effective branch differs from resolved default", () => {
    const instruction = buildBranchInstruction("feature", "develop");
    expect(instruction).toBe("IMPORTANT: Create your branch from the 'feature' branch, NOT from 'develop'.");
  });

  it("omits NOT clause when resolvedDefaultBranch is null", () => {
    const instruction = buildBranchInstruction("feature", null);
    expect(instruction).toBe("IMPORTANT: Create your branch from the 'feature' branch.");
    expect(instruction).not.toContain("NOT from");
  });
});
