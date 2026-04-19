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

    await expect(__testables.getBranchPolicy(githubClient, "github", "gh-aw", "main;rm -rf /")).rejects.toThrow("Invalid target base branch for policy evaluation");
    expect(githubClient.rest.repos.getBranch).not.toHaveBeenCalled();
  });

  it("matches allowed labels by exact value (no glob matching)", async () => {
    const { __testables } = await import("./merge_pull_request.cjs");

    expect(__testables.findAllowedLabelMatches(["release/v1", "automerge/pr-1"], ["release/*", "automerge/*"])).toEqual([]);
    expect(__testables.findAllowedLabelMatches(["automerge", "release"], ["automerge", "deploy"])).toEqual(["automerge"]);
    expect(__testables.findAllowedLabelMatches(["release/*", "automerge/*"], ["release/*", "automerge/*"])).toEqual(["release/*", "automerge/*"]);
    expect(__testables.findAllowedLabelMatches([], ["automerge"])).toEqual([]);
    expect(__testables.findAllowedLabelMatches(["AutoMerge"], ["automerge"])).toEqual([]);
  });
});
