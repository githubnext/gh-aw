import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

describe("merge_pull_request branch validation", () => {
  beforeEach(() => {
    global.core = {
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
    };
  });

  afterEach(() => {
    vi.resetModules();
    vi.clearAllMocks();
    delete global.core;
  });

  it("sanitizes and rejects invalid branch names", async () => {
    const { __testables } = await import("./merge_pull_request.cjs");

    const valid = __testables.sanitizeBranchName("feature/ok-branch", "source");
    expect(valid).toEqual({ valid: true, value: "feature/ok-branch" });

    const invalid = __testables.sanitizeBranchName("feature/unsafe\nbranch", "source");
    expect(invalid.valid).toBe(false);
    expect(invalid.error).toContain("contains invalid characters");
  });

  it("marks protected base branch as protected", async () => {
    const { __testables } = await import("./merge_pull_request.cjs");

    const githubClient = {
      rest: {
        repos: {
          getBranch: vi.fn().mockResolvedValue({ data: { protected: true } }),
          get: vi.fn().mockResolvedValue({ data: { default_branch: "main" } }),
        },
      },
    };

    const policy = await __testables.getBranchPolicy(githubClient, "github", "gh-aw", "release/1.0");
    expect(policy.isProtected).toBe(true);
    expect(policy.requiredChecks).toEqual([]);
  });

  it("detects repository default branch", async () => {
    const { __testables } = await import("./merge_pull_request.cjs");

    const githubClient = {
      rest: {
        repos: {
          getBranch: vi.fn().mockResolvedValue({
            data: {
              protected: false,
            },
          }),
          getBranchProtection: vi.fn().mockResolvedValue({
            data: { required_status_checks: { contexts: ["ci/test"] } },
          }),
          get: vi.fn().mockResolvedValue({ data: { default_branch: "main" } }),
        },
      },
    };

    const policy = await __testables.getBranchPolicy(githubClient, "github", "gh-aw", "main");
    expect(policy.isDefault).toBe(true);
    expect(policy.requiredChecks).toEqual(["ci/test"]);
  });

  it("does not mark non-default branches as default", async () => {
    const { __testables } = await import("./merge_pull_request.cjs");

    const githubClient = {
      rest: {
        repos: {
          getBranch: vi.fn().mockResolvedValue({ data: { protected: false } }),
          getBranchProtection: vi.fn().mockRejectedValue({ status: 404 }),
          get: vi.fn().mockResolvedValue({ data: { default_branch: "main" } }),
        },
      },
    };

    const policy = await __testables.getBranchPolicy(githubClient, "github", "gh-aw", "feature-branch");
    expect(policy.isDefault).toBe(false);
  });

  it("rejects unsafe base branch names before branch policy lookup", async () => {
    const { __testables } = await import("./merge_pull_request.cjs");

    const githubClient = {
      rest: {
        repos: {
          getBranch: vi.fn(),
          get: vi.fn(),
        },
      },
    };

    await expect(__testables.getBranchPolicy(githubClient, "github", "gh-aw", "main;rm -rf /")).rejects.toThrow("E001: Invalid target base branch for policy evaluation");
    expect(githubClient.rest.repos.getBranch).not.toHaveBeenCalled();
  });

  it("findMissingRequiredLabels returns labels not present on the item (conjunctive check)", async () => {
    const { __testables } = await import("./merge_pull_request.cjs");

    // all required labels present → none missing
    expect(__testables.findMissingRequiredLabels(["automerge", "safe-to-merge"], ["automerge", "safe-to-merge"])).toEqual([]);
    // one required label missing
    expect(__testables.findMissingRequiredLabels(["automerge"], ["automerge", "safe-to-merge"])).toEqual(["safe-to-merge"]);
    // no required labels → none missing
    expect(__testables.findMissingRequiredLabels(["automerge"], [])).toEqual([]);
    // all required labels missing
    expect(__testables.findMissingRequiredLabels([], ["automerge", "safe-to-merge"])).toEqual(["automerge", "safe-to-merge"]);
    // case-sensitive: wrong case counts as missing
    expect(__testables.findMissingRequiredLabels(["AutoMerge"], ["automerge"])).toEqual(["automerge"]);
  });

  it("resolves temporary ID for pull_request_number", async () => {
    const { __testables } = await import("./merge_pull_request.cjs");
    const result = __testables.resolvePullRequestNumber({ pull_request_number: "aw_pr1" }, { aw_pr1: { number: 42 } });
    expect(result).toEqual({ success: true, pullNumber: 42, fromTemporaryId: true });
  });

  it("fails on unresolved temporary ID for pull_request_number", async () => {
    const { __testables } = await import("./merge_pull_request.cjs");
    const result = __testables.resolvePullRequestNumber({ pull_request_number: "aw_missing" }, {});
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error).toContain("Unresolved temporary ID");
    }
  });
});
