import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

describe("getBaseBranch", () => {
  let originalEnv;

  beforeEach(() => {
    // Save original environment
    originalEnv = {
      GH_AW_CUSTOM_BASE_BRANCH: process.env.GH_AW_CUSTOM_BASE_BRANCH,
      GITHUB_BASE_REF: process.env.GITHUB_BASE_REF,
      DEFAULT_BRANCH: process.env.DEFAULT_BRANCH,
    };
    // Clear all base branch env vars
    delete process.env.GH_AW_CUSTOM_BASE_BRANCH;
    delete process.env.GITHUB_BASE_REF;
    delete process.env.DEFAULT_BRANCH;
    // Clear context and github globals
    delete global.context;
    delete global.github;
    delete global.core;
  });

  afterEach(() => {
    // Restore original environment
    for (const [key, value] of Object.entries(originalEnv)) {
      if (value !== undefined) {
        process.env[key] = value;
      } else {
        delete process.env[key];
      }
    }
    delete global.context;
    delete global.github;
    delete global.core;
    vi.resetModules();
  });

  it("should return main by default when no env vars set", async () => {
    const { getBaseBranch } = await import("./get_base_branch.cjs");
    const result = await getBaseBranch();
    expect(result).toBe("main");
  });

  it("should return GH_AW_CUSTOM_BASE_BRANCH if set (highest priority)", async () => {
    process.env.GH_AW_CUSTOM_BASE_BRANCH = "custom-base";
    process.env.GITHUB_BASE_REF = "pr-base";
    process.env.DEFAULT_BRANCH = "develop";

    const { getBaseBranch } = await import("./get_base_branch.cjs");
    const result = await getBaseBranch();
    expect(result).toBe("custom-base");
  });

  it("should return GITHUB_BASE_REF if GH_AW_CUSTOM_BASE_BRANCH not set", async () => {
    process.env.GITHUB_BASE_REF = "pr-base";
    process.env.DEFAULT_BRANCH = "develop";

    const { getBaseBranch } = await import("./get_base_branch.cjs");
    const result = await getBaseBranch();
    expect(result).toBe("pr-base");
  });

  it("should return DEFAULT_BRANCH as fallback", async () => {
    process.env.DEFAULT_BRANCH = "develop";

    const { getBaseBranch } = await import("./get_base_branch.cjs");
    const result = await getBaseBranch();
    expect(result).toBe("develop");
  });

  it("should handle various branch names", async () => {
    const { getBaseBranch } = await import("./get_base_branch.cjs");

    process.env.GH_AW_CUSTOM_BASE_BRANCH = "master";
    expect(await getBaseBranch()).toBe("master");

    process.env.GH_AW_CUSTOM_BASE_BRANCH = "release/v1.0";
    expect(await getBaseBranch()).toBe("release/v1.0");

    process.env.GH_AW_CUSTOM_BASE_BRANCH = "feature/new-feature";
    expect(await getBaseBranch()).toBe("feature/new-feature");
  });

  it("should accept optional targetRepo parameter without affecting simple cases", async () => {
    // When env vars are set, targetRepo doesn't change the result
    process.env.GH_AW_CUSTOM_BASE_BRANCH = "custom-branch";

    const { getBaseBranch } = await import("./get_base_branch.cjs");

    // With targetRepo parameter
    const result = await getBaseBranch({ owner: "other-owner", repo: "other-repo" });
    expect(result).toBe("custom-branch");

    // Without targetRepo parameter (null)
    const result2 = await getBaseBranch(null);
    expect(result2).toBe("custom-branch");

    // Without targetRepo parameter (undefined)
    const result3 = await getBaseBranch();
    expect(result3).toBe("custom-branch");
  });

  it("should return context.payload.repository.default_branch without an API call", async () => {
    const mockGetRepo = vi.fn();
    global.github = {
      rest: { repos: { get: mockGetRepo } },
    };
    global.context = {
      repo: { owner: "test-owner", repo: "test-repo" },
      eventName: "push",
      payload: { repository: { default_branch: "trunk" } },
    };

    const { getBaseBranch } = await import("./get_base_branch.cjs");
    const result = await getBaseBranch();

    expect(result).toBe("trunk");
    expect(mockGetRepo).not.toHaveBeenCalled();
  });

  it("should query GitHub API for repo default branch when payload does not have it", async () => {
    const mockGetRepo = vi.fn().mockResolvedValue({ data: { default_branch: "master" } });
    global.github = {
      rest: {
        repos: {
          get: mockGetRepo,
        },
      },
    };
    global.context = {
      repo: { owner: "test-owner", repo: "test-repo" },
      eventName: "workflow_dispatch",
      payload: {},
    };

    const { getBaseBranch } = await import("./get_base_branch.cjs");
    const result = await getBaseBranch();

    expect(mockGetRepo).toHaveBeenCalledWith({ owner: "test-owner", repo: "test-repo" });
    expect(result).toBe("master");
  });

  it("should use repos.get() for targetRepo (cross-repo) even when payload has default_branch", async () => {
    const mockGetRepo = vi.fn().mockResolvedValue({ data: { default_branch: "develop" } });
    global.github = {
      rest: { repos: { get: mockGetRepo } },
    };
    global.context = {
      repo: { owner: "workflow-owner", repo: "workflow-repo" },
      eventName: "issue_comment",
      payload: { repository: { default_branch: "main" } },
    };

    const { getBaseBranch } = await import("./get_base_branch.cjs");
    // targetRepo differs from the workflow repo - must use API, not payload
    const result = await getBaseBranch({ owner: "target-owner", repo: "target-repo" });

    expect(mockGetRepo).toHaveBeenCalledWith({ owner: "target-owner", repo: "target-repo" });
    expect(result).toBe("develop");
  });

  it("should use targetRepo for API default branch lookup", async () => {
    const mockGetRepo = vi.fn().mockResolvedValue({ data: { default_branch: "develop" } });
    global.github = {
      rest: {
        repos: {
          get: mockGetRepo,
        },
      },
    };

    const { getBaseBranch } = await import("./get_base_branch.cjs");
    const result = await getBaseBranch({ owner: "target-owner", repo: "target-repo" });

    expect(mockGetRepo).toHaveBeenCalledWith({ owner: "target-owner", repo: "target-repo" });
    expect(result).toBe("develop");
  });

  it("should fall through to DEFAULT_BRANCH when API lookup for repo default branch fails", async () => {
    const mockWarning = vi.fn();
    global.github = {
      rest: {
        repos: {
          get: vi.fn().mockRejectedValue(new Error("API error")),
        },
      },
    };
    global.core = { warning: mockWarning };
    process.env.DEFAULT_BRANCH = "trunk";

    const { getBaseBranch } = await import("./get_base_branch.cjs");
    const result = await getBaseBranch({ owner: "owner", repo: "repo" });

    expect(result).toBe("trunk");
    expect(mockWarning).toHaveBeenCalledWith(expect.stringContaining("Failed to fetch repository default branch"));
  });

  it("should fall through to hardcoded main when API lookup fails and no DEFAULT_BRANCH set", async () => {
    global.github = {
      rest: {
        repos: {
          get: vi.fn().mockRejectedValue(new Error("API error")),
        },
      },
    };
    global.core = { warning: vi.fn() };

    const { getBaseBranch } = await import("./get_base_branch.cjs");
    const result = await getBaseBranch({ owner: "owner", repo: "repo" });

    expect(result).toBe("main");
  });
});
