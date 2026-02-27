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
});
