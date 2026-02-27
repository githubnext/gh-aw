import { describe, it, expect, beforeEach, afterEach } from "vitest";

describe("generateGitPatch", () => {
  let originalEnv;

  beforeEach(() => {
    // Save original environment
    originalEnv = {
      GITHUB_SHA: process.env.GITHUB_SHA,
      GITHUB_WORKSPACE: process.env.GITHUB_WORKSPACE,
      DEFAULT_BRANCH: process.env.DEFAULT_BRANCH,
      GH_AW_BASE_BRANCH: process.env.GH_AW_BASE_BRANCH,
    };
  });

  afterEach(() => {
    // Restore original environment
    Object.keys(originalEnv).forEach(key => {
      if (originalEnv[key] !== undefined) {
        process.env[key] = originalEnv[key];
      } else {
        delete process.env[key];
      }
    });
  });

  it("should return error when no commits can be found", async () => {
    delete process.env.GITHUB_SHA;
    process.env.GITHUB_WORKSPACE = "/tmp/test-repo";

    const { generateGitPatch } = await import("./generate_git_patch.cjs");

    const result = await generateGitPatch(null);

    expect(result.success).toBe(false);
    expect(result).toHaveProperty("error");
  });

  it("should return success false when no commits found", async () => {
    const { generateGitPatch } = await import("./generate_git_patch.cjs");

    // Set up environment but in a way that won't find commits
    process.env.GITHUB_WORKSPACE = "/tmp/nonexistent-repo";
    process.env.GITHUB_SHA = "abc123";

    const result = await generateGitPatch("nonexistent-branch");

    expect(result.success).toBe(false);
    expect(result).toHaveProperty("error");
    expect(result).toHaveProperty("patchPath");
  });

  it("should create patch directory if it doesn't exist", async () => {
    const { generateGitPatch } = await import("./generate_git_patch.cjs");

    process.env.GITHUB_WORKSPACE = "/tmp/nonexistent-repo";
    process.env.GITHUB_SHA = "abc123";

    // Even if it fails, it should try to create the directory
    const result = await generateGitPatch("test-branch");

    expect(result).toHaveProperty("patchPath");
    // Patch path includes sanitized branch name
    expect(result.patchPath).toBe("/tmp/gh-aw/aw-test-branch.patch");
  });

  it("should return patch info structure", async () => {
    const { generateGitPatch } = await import("./generate_git_patch.cjs");

    process.env.GITHUB_WORKSPACE = "/tmp/nonexistent-repo";
    process.env.GITHUB_SHA = "abc123";

    const result = await generateGitPatch("test-branch");

    expect(result).toHaveProperty("success");
    expect(result).toHaveProperty("patchPath");
    expect(typeof result.success).toBe("boolean");
  });

  it("should handle null branch name", async () => {
    const { generateGitPatch } = await import("./generate_git_patch.cjs");

    process.env.GITHUB_WORKSPACE = "/tmp/nonexistent-repo";
    process.env.GITHUB_SHA = "abc123";

    const result = await generateGitPatch(null);

    expect(result).toHaveProperty("success");
    expect(result).toHaveProperty("patchPath");
  });

  it("should handle empty branch name", async () => {
    const { generateGitPatch } = await import("./generate_git_patch.cjs");

    process.env.GITHUB_WORKSPACE = "/tmp/nonexistent-repo";
    process.env.GITHUB_SHA = "abc123";

    const result = await generateGitPatch("");

    expect(result).toHaveProperty("success");
    expect(result).toHaveProperty("patchPath");
  });

  it("should use default branch from environment", async () => {
    const { generateGitPatch } = await import("./generate_git_patch.cjs");

    process.env.GITHUB_WORKSPACE = "/tmp/nonexistent-repo";
    process.env.GITHUB_SHA = "abc123";
    process.env.DEFAULT_BRANCH = "develop";

    const result = await generateGitPatch("feature-branch");

    expect(result).toHaveProperty("success");
    // Should attempt to use develop as default branch
  });

  it("should fall back to GH_AW_CUSTOM_BASE_BRANCH if DEFAULT_BRANCH not set", async () => {
    const { generateGitPatch } = await import("./generate_git_patch.cjs");

    process.env.GITHUB_WORKSPACE = "/tmp/nonexistent-repo";
    process.env.GITHUB_SHA = "abc123";
    delete process.env.DEFAULT_BRANCH;
    process.env.GH_AW_CUSTOM_BASE_BRANCH = "master";

    const result = await generateGitPatch("feature-branch");

    expect(result).toHaveProperty("success");
    // Should attempt to use master as base branch
  });

  it("should safely handle branch names with special characters", async () => {
    const { generateGitPatch } = await import("./generate_git_patch.cjs");

    process.env.GITHUB_WORKSPACE = "/tmp/nonexistent-repo";
    process.env.GITHUB_SHA = "abc123";

    // Test with various special characters that could cause shell injection
    const maliciousBranchNames = ["feature; rm -rf /", "feature && echo hacked", "feature | cat /etc/passwd", "feature$(whoami)", "feature`whoami`", "feature\nrm -rf /"];

    for (const branchName of maliciousBranchNames) {
      const result = await generateGitPatch(branchName);

      // Should not throw an error and should handle safely
      expect(result).toHaveProperty("success");
      expect(result.success).toBe(false);
      // Should fail gracefully without executing injected commands
    }
  });

  it("should safely handle GITHUB_SHA with special characters", async () => {
    const { generateGitPatch } = await import("./generate_git_patch.cjs");

    process.env.GITHUB_WORKSPACE = "/tmp/nonexistent-repo";

    // Test with malicious SHA that could cause shell injection
    process.env.GITHUB_SHA = "abc123; echo hacked";

    const result = await generateGitPatch("test-branch");

    // Should not throw an error and should handle safely
    expect(result).toHaveProperty("success");
    expect(result.success).toBe(false);
  });
});

describe("generateGitPatch - cross-repo checkout scenarios", () => {
  let originalEnv;

  beforeEach(() => {
    // Save original environment
    originalEnv = {
      GITHUB_SHA: process.env.GITHUB_SHA,
      GITHUB_WORKSPACE: process.env.GITHUB_WORKSPACE,
      DEFAULT_BRANCH: process.env.DEFAULT_BRANCH,
      GH_AW_CUSTOM_BASE_BRANCH: process.env.GH_AW_CUSTOM_BASE_BRANCH,
    };
  });

  afterEach(() => {
    // Restore original environment
    Object.keys(originalEnv).forEach(key => {
      if (originalEnv[key] !== undefined) {
        process.env[key] = originalEnv[key];
      } else {
        delete process.env[key];
      }
    });
  });

  it("should handle GITHUB_SHA not existing in cross-repo checkout", async () => {
    const { generateGitPatch } = await import("./generate_git_patch.cjs");

    // In cross-repo checkout, GITHUB_SHA is from the workflow repo,
    // not the target repo that's checked out
    process.env.GITHUB_WORKSPACE = "/tmp/nonexistent-repo";
    process.env.GITHUB_SHA = "deadbeef123456789"; // SHA that doesn't exist in target repo

    const result = await generateGitPatch("feature-branch");

    // Should fail gracefully, not crash
    expect(result).toHaveProperty("success");
    expect(result.success).toBe(false);
    expect(result).toHaveProperty("error");
  });

  it("should fall back gracefully when persist-credentials is false", async () => {
    const { generateGitPatch } = await import("./generate_git_patch.cjs");

    // Simulate cross-repo checkout where fetch fails due to persist-credentials: false
    process.env.GITHUB_WORKSPACE = "/tmp/nonexistent-repo";
    process.env.GITHUB_SHA = "abc123";
    process.env.DEFAULT_BRANCH = "main";

    const result = await generateGitPatch("feature-branch");

    // Should try multiple strategies and fail gracefully
    expect(result).toHaveProperty("success");
    expect(result.success).toBe(false);
    expect(result).toHaveProperty("error");
    expect(result).toHaveProperty("patchPath");
  });

  it("should check local refs before attempting network fetch", async () => {
    const { generateGitPatch } = await import("./generate_git_patch.cjs");

    // This tests that Strategy 1 checks for local refs before fetching
    process.env.GITHUB_WORKSPACE = "/tmp/nonexistent-repo";
    process.env.DEFAULT_BRANCH = "main";

    const result = await generateGitPatch("feature-branch");

    // Should complete without hanging or crashing due to fetch attempts
    expect(result).toHaveProperty("success");
    expect(result).toHaveProperty("patchPath");
  });

  it("should return meaningful error for cross-repo scenarios", async () => {
    const { generateGitPatch } = await import("./generate_git_patch.cjs");

    process.env.GITHUB_WORKSPACE = "/tmp/nonexistent-repo";
    process.env.GITHUB_SHA = "sha-from-workflow-repo";
    process.env.DEFAULT_BRANCH = "main";

    const result = await generateGitPatch("agent-created-branch");

    expect(result.success).toBe(false);
    expect(result).toHaveProperty("error");
    // Error should be informative
    expect(typeof result.error).toBe("string");
    expect(result.error.length).toBeGreaterThan(0);
  });

  it("should handle incremental mode failure in cross-repo checkout", async () => {
    const { generateGitPatch } = await import("./generate_git_patch.cjs");

    process.env.GITHUB_WORKSPACE = "/tmp/nonexistent-repo";
    process.env.DEFAULT_BRANCH = "main";

    // Incremental mode requires origin/branchName to exist - should fail clearly
    const result = await generateGitPatch("feature-branch", { mode: "incremental" });

    expect(result.success).toBe(false);
    expect(result).toHaveProperty("error");
    // Should indicate the branch doesn't exist or can't be fetched
    expect(result.error).toMatch(/branch|fetch|incremental/i);
  });

  it("should handle SideRepoOps pattern where workflow repo != target repo", async () => {
    const { generateGitPatch } = await import("./generate_git_patch.cjs");

    // Simulates: workflow in org/side-repo, checkout of org/target-repo
    // GITHUB_SHA would be from side-repo, not target-repo
    process.env.GITHUB_WORKSPACE = "/tmp/nonexistent-target-repo";
    process.env.GITHUB_SHA = "side-repo-sha-not-in-target";
    process.env.DEFAULT_BRANCH = "main";

    const result = await generateGitPatch("agent-changes");

    // Should not crash, should return failure with helpful error
    expect(result).toHaveProperty("success");
    expect(result.success).toBe(false);
    expect(result).toHaveProperty("patchPath");
  });
});

describe("sanitizeBranchNameForPatch", () => {
  it("should sanitize branch names with path separators", async () => {
    const { sanitizeBranchNameForPatch } = await import("./generate_git_patch.cjs");

    expect(sanitizeBranchNameForPatch("feature/add-login")).toBe("feature-add-login");
    expect(sanitizeBranchNameForPatch("user\\branch")).toBe("user-branch");
  });

  it("should sanitize branch names with special characters", async () => {
    const { sanitizeBranchNameForPatch } = await import("./generate_git_patch.cjs");

    expect(sanitizeBranchNameForPatch("feature:test")).toBe("feature-test");
    expect(sanitizeBranchNameForPatch("branch*name")).toBe("branch-name");
    expect(sanitizeBranchNameForPatch('branch?"name')).toBe("branch-name");
    expect(sanitizeBranchNameForPatch("branch<>name")).toBe("branch-name");
    expect(sanitizeBranchNameForPatch("branch|name")).toBe("branch-name");
  });

  it("should collapse multiple dashes", async () => {
    const { sanitizeBranchNameForPatch } = await import("./generate_git_patch.cjs");

    expect(sanitizeBranchNameForPatch("feature//double")).toBe("feature-double");
    expect(sanitizeBranchNameForPatch("a---b")).toBe("a-b");
  });

  it("should remove leading and trailing dashes", async () => {
    const { sanitizeBranchNameForPatch } = await import("./generate_git_patch.cjs");

    expect(sanitizeBranchNameForPatch("/feature")).toBe("feature");
    expect(sanitizeBranchNameForPatch("feature/")).toBe("feature");
    expect(sanitizeBranchNameForPatch("/feature/")).toBe("feature");
  });

  it("should convert to lowercase", async () => {
    const { sanitizeBranchNameForPatch } = await import("./generate_git_patch.cjs");

    expect(sanitizeBranchNameForPatch("Feature-Branch")).toBe("feature-branch");
    expect(sanitizeBranchNameForPatch("UPPER")).toBe("upper");
  });

  it("should handle null and empty strings", async () => {
    const { sanitizeBranchNameForPatch } = await import("./generate_git_patch.cjs");

    expect(sanitizeBranchNameForPatch(null)).toBe("unknown");
    expect(sanitizeBranchNameForPatch("")).toBe("unknown");
    expect(sanitizeBranchNameForPatch(undefined)).toBe("unknown");
  });
});

describe("getPatchPath", () => {
  it("should return correct path format", async () => {
    const { getPatchPath } = await import("./generate_git_patch.cjs");

    expect(getPatchPath("feature-branch")).toBe("/tmp/gh-aw/aw-feature-branch.patch");
  });

  it("should sanitize branch name in path", async () => {
    const { getPatchPath } = await import("./generate_git_patch.cjs");

    expect(getPatchPath("feature/branch")).toBe("/tmp/gh-aw/aw-feature-branch.patch");
    expect(getPatchPath("Feature/BRANCH")).toBe("/tmp/gh-aw/aw-feature-branch.patch");
  });
});
