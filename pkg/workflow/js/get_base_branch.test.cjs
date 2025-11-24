import { describe, it, expect, beforeEach, afterEach } from "vitest";

describe("getBaseBranch", () => {
  let originalEnv;

  beforeEach(() => {
    // Save original environment
    originalEnv = process.env.GH_AW_BASE_BRANCH;
  });

  afterEach(() => {
    // Restore original environment
    if (originalEnv !== undefined) {
      process.env.GH_AW_BASE_BRANCH = originalEnv;
    } else {
      delete process.env.GH_AW_BASE_BRANCH;
    }
  });

  it("should return main by default", async () => {
    delete process.env.GH_AW_BASE_BRANCH;
    const { getBaseBranch } = await import("./get_base_branch.cjs");

    expect(getBaseBranch()).toBe("main");
  });

  it("should return environment variable value", async () => {
    process.env.GH_AW_BASE_BRANCH = "develop";
    const { getBaseBranch } = await import("./get_base_branch.cjs");

    expect(getBaseBranch()).toBe("develop");
  });

  it("should handle various branch names", async () => {
    const { getBaseBranch } = await import("./get_base_branch.cjs");

    process.env.GH_AW_BASE_BRANCH = "master";
    expect(getBaseBranch()).toBe("master");

    process.env.GH_AW_BASE_BRANCH = "release/v1.0";
    expect(getBaseBranch()).toBe("release/v1.0");

    process.env.GH_AW_BASE_BRANCH = "feature/new-feature";
    expect(getBaseBranch()).toBe("feature/new-feature");
  });

  it("should return main if environment variable is empty string", async () => {
    process.env.GH_AW_BASE_BRANCH = "";
    const { getBaseBranch } = await import("./get_base_branch.cjs");

    // Empty string is falsy, so || operator returns "main"
    expect(getBaseBranch()).toBe("main");
  });
});
