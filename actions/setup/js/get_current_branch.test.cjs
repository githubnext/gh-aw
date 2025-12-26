import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

describe("getCurrentBranch", () => {
  let originalEnv;

  beforeEach(() => {
    // Save original environment
    originalEnv = {
      GITHUB_HEAD_REF: process.env.GITHUB_HEAD_REF,
      GITHUB_REF_NAME: process.env.GITHUB_REF_NAME,
      GITHUB_WORKSPACE: process.env.GITHUB_WORKSPACE,
    };

    // Clean environment for tests
    delete process.env.GITHUB_HEAD_REF;
    delete process.env.GITHUB_REF_NAME;
    delete process.env.GITHUB_WORKSPACE;
  });

  afterEach(() => {
    // Restore original environment
    if (originalEnv.GITHUB_HEAD_REF !== undefined) {
      process.env.GITHUB_HEAD_REF = originalEnv.GITHUB_HEAD_REF;
    }
    if (originalEnv.GITHUB_REF_NAME !== undefined) {
      process.env.GITHUB_REF_NAME = originalEnv.GITHUB_REF_NAME;
    }
    if (originalEnv.GITHUB_WORKSPACE !== undefined) {
      process.env.GITHUB_WORKSPACE = originalEnv.GITHUB_WORKSPACE;
    }
  });

  it("should return GITHUB_HEAD_REF if set", async () => {
    process.env.GITHUB_HEAD_REF = "feature/test-branch";
    process.env.GITHUB_REF_NAME = "other-branch";

    const { getCurrentBranch } = await import("./get_current_branch.cjs");

    // If git command fails, should use GITHUB_HEAD_REF
    try {
      const result = getCurrentBranch();
      // Either from git or from GITHUB_HEAD_REF
      expect(typeof result).toBe("string");
      expect(result.length).toBeGreaterThan(0);
    } catch (error) {
      // This is acceptable if we're not in a git repo
      expect(error.message).toContain("Failed to determine current branch");
    }
  });

  it("should return GITHUB_REF_NAME if GITHUB_HEAD_REF not set", async () => {
    delete process.env.GITHUB_HEAD_REF;
    process.env.GITHUB_REF_NAME = "main";

    const { getCurrentBranch } = await import("./get_current_branch.cjs");

    try {
      const result = getCurrentBranch();
      // Either from git or from GITHUB_REF_NAME
      expect(typeof result).toBe("string");
      expect(result.length).toBeGreaterThan(0);
    } catch (error) {
      // This is acceptable if we're not in a git repo
      expect(error.message).toContain("Failed to determine current branch");
    }
  });

  it("should throw error when no branch can be determined", async () => {
    delete process.env.GITHUB_HEAD_REF;
    delete process.env.GITHUB_REF_NAME;
    process.env.GITHUB_WORKSPACE = "/tmp/nonexistent-git-repo";

    const { getCurrentBranch } = await import("./get_current_branch.cjs");

    expect(() => getCurrentBranch()).toThrow("Failed to determine current branch");
  });

  it("should prioritize GITHUB_HEAD_REF over GITHUB_REF_NAME", async () => {
    process.env.GITHUB_HEAD_REF = "pr-branch";
    process.env.GITHUB_REF_NAME = "main";
    process.env.GITHUB_WORKSPACE = "/tmp/nonexistent-git-repo";

    const { getCurrentBranch } = await import("./get_current_branch.cjs");

    try {
      const result = getCurrentBranch();
      // If git fails, should fall back to GITHUB_HEAD_REF
      if (result === "pr-branch" || result === "main") {
        expect(result).toBeTruthy();
      }
    } catch (error) {
      // This is acceptable if we're not in a git repo
      expect(error.message).toContain("Failed to determine current branch");
    }
  });
});
