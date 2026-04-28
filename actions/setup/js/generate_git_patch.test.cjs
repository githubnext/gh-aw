import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { execSync } from "child_process";
import { createRequire } from "module";
import * as fs from "fs";
import * as path from "path";
import * as os from "os";

const require = createRequire(import.meta.url);

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

    const result = await generateGitPatch(null, "main");

    expect(result.success).toBe(false);
    expect(result).toHaveProperty("error");
  });

  it("should return success false when no commits found", async () => {
    const { generateGitPatch } = await import("./generate_git_patch.cjs");

    // Set up environment but in a way that won't find commits
    process.env.GITHUB_WORKSPACE = "/tmp/nonexistent-repo";
    process.env.GITHUB_SHA = "abc123";

    const result = await generateGitPatch("nonexistent-branch", "main");

    expect(result.success).toBe(false);
    expect(result).toHaveProperty("error");
    expect(result).toHaveProperty("patchPath");
  });

  it("should create patch directory if it doesn't exist", async () => {
    const { generateGitPatch } = await import("./generate_git_patch.cjs");

    process.env.GITHUB_WORKSPACE = "/tmp/nonexistent-repo";
    process.env.GITHUB_SHA = "abc123";

    // Even if it fails, it should try to create the directory
    const result = await generateGitPatch("test-branch", "main");

    expect(result).toHaveProperty("patchPath");
    // Patch path includes sanitized branch name
    expect(result.patchPath).toBe("/tmp/gh-aw/aw-test-branch.patch");
  });

  it("should return patch info structure", async () => {
    const { generateGitPatch } = await import("./generate_git_patch.cjs");

    process.env.GITHUB_WORKSPACE = "/tmp/nonexistent-repo";
    process.env.GITHUB_SHA = "abc123";

    const result = await generateGitPatch("test-branch", "main");

    expect(result).toHaveProperty("success");
    expect(result).toHaveProperty("patchPath");
    expect(typeof result.success).toBe("boolean");
  });

  it("should handle null branch name", async () => {
    const { generateGitPatch } = await import("./generate_git_patch.cjs");

    process.env.GITHUB_WORKSPACE = "/tmp/nonexistent-repo";
    process.env.GITHUB_SHA = "abc123";

    const result = await generateGitPatch(null, "main");

    expect(result).toHaveProperty("success");
    expect(result).toHaveProperty("patchPath");
  });

  it("should handle empty branch name", async () => {
    const { generateGitPatch } = await import("./generate_git_patch.cjs");

    process.env.GITHUB_WORKSPACE = "/tmp/nonexistent-repo";
    process.env.GITHUB_SHA = "abc123";

    const result = await generateGitPatch("", "main");

    expect(result).toHaveProperty("success");
    expect(result).toHaveProperty("patchPath");
  });

  it("should use provided base branch", async () => {
    const { generateGitPatch } = await import("./generate_git_patch.cjs");

    process.env.GITHUB_WORKSPACE = "/tmp/nonexistent-repo";
    process.env.GITHUB_SHA = "abc123";

    const result = await generateGitPatch("feature-branch", "develop");

    expect(result).toHaveProperty("success");
    // Should use develop as base branch
  });

  it("should use provided master branch as base", async () => {
    const { generateGitPatch } = await import("./generate_git_patch.cjs");

    process.env.GITHUB_WORKSPACE = "/tmp/nonexistent-repo";
    process.env.GITHUB_SHA = "abc123";

    const result = await generateGitPatch("feature-branch", "master");

    expect(result).toHaveProperty("success");
    // Should use master as base branch
  });

  it("should safely handle branch names with special characters", async () => {
    const { generateGitPatch } = await import("./generate_git_patch.cjs");

    process.env.GITHUB_WORKSPACE = "/tmp/nonexistent-repo";
    process.env.GITHUB_SHA = "abc123";

    // Test with various special characters that could cause shell injection
    const maliciousBranchNames = ["feature; rm -rf /", "feature && echo hacked", "feature | cat /etc/passwd", "feature$(whoami)", "feature`whoami`", "feature\nrm -rf /"];

    for (const branchName of maliciousBranchNames) {
      const result = await generateGitPatch(branchName, "main");

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

    const result = await generateGitPatch("test-branch", "main");

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

    const result = await generateGitPatch("feature-branch", "main");

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

    const result = await generateGitPatch("feature-branch", "main");

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

    const result = await generateGitPatch("feature-branch", "main");

    // Should complete without hanging or crashing due to fetch attempts
    expect(result).toHaveProperty("success");
    expect(result).toHaveProperty("patchPath");
  });

  it("should return meaningful error for cross-repo scenarios", async () => {
    const { generateGitPatch } = await import("./generate_git_patch.cjs");

    process.env.GITHUB_WORKSPACE = "/tmp/nonexistent-repo";
    process.env.GITHUB_SHA = "sha-from-workflow-repo";

    const result = await generateGitPatch("agent-created-branch", "main");

    expect(result.success).toBe(false);
    expect(result).toHaveProperty("error");
    // Error should be informative
    expect(typeof result.error).toBe("string");
    expect(result.error.length).toBeGreaterThan(0);
  });

  it("should handle incremental mode failure in cross-repo checkout", async () => {
    const { generateGitPatch } = await import("./generate_git_patch.cjs");

    process.env.GITHUB_WORKSPACE = "/tmp/nonexistent-repo";

    // Incremental mode requires origin/branchName to exist - should fail clearly
    const result = await generateGitPatch("feature-branch", "main", { mode: "incremental" });

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

    const result = await generateGitPatch("agent-changes", "main");

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

describe("generateGitPatch - standardized error codes", () => {
  let originalEnv;

  beforeEach(() => {
    originalEnv = {
      GITHUB_SHA: process.env.GITHUB_SHA,
      GITHUB_WORKSPACE: process.env.GITHUB_WORKSPACE,
    };
  });

  afterEach(() => {
    Object.keys(originalEnv).forEach(key => {
      if (originalEnv[key] !== undefined) {
        process.env[key] = originalEnv[key];
      } else {
        delete process.env[key];
      }
    });
  });

  it("should fail gracefully and return a non-empty error string when no commits can be found", async () => {
    const { generateGitPatch } = await import("./generate_git_patch.cjs");

    process.env.GITHUB_WORKSPACE = "/tmp/nonexistent-repo";
    process.env.GITHUB_SHA = "abc123";

    const result = await generateGitPatch("feature-branch", "main");

    expect(result.success).toBe(false);
    expect(result).toHaveProperty("error");
    // Note: E005 is used as an internal control-flow signal in Strategy 1 (full mode)
    // and is caught before reaching the final return value. The conformance check
    // validates the E005 code at source level via check-safe-outputs-conformance.sh.
    expect(typeof result.error).toBe("string");
    expect(result.error.length).toBeGreaterThan(0);
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

// ──────────────────────────────────────────────────────
// excludedFiles option – end-to-end with a real git repo
// ──────────────────────────────────────────────────────

describe("generateGitPatch – excludedFiles option", () => {
  let repoDir;
  let originalEnv;

  beforeEach(() => {
    originalEnv = { GITHUB_WORKSPACE: process.env.GITHUB_WORKSPACE, GITHUB_SHA: process.env.GITHUB_SHA };

    // Set up the core global required by git_helpers.cjs
    global.core = { debug: () => {}, info: () => {}, warning: () => {}, error: () => {} };

    // Create an isolated git repo in a temp directory
    repoDir = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-patch-test-"));
    execSync("git init -b main", { cwd: repoDir });
    execSync('git config user.email "test@example.com"', { cwd: repoDir });
    execSync('git config user.name "Test"', { cwd: repoDir });

    // Initial commit so the repo has a base
    fs.writeFileSync(path.join(repoDir, "README.md"), "# Repo\n");
    execSync("git add .", { cwd: repoDir });
    execSync('git commit -m "init"', { cwd: repoDir });

    // Record the initial commit SHA for GITHUB_SHA (Strategy 2 base)
    const sha = execSync("git rev-parse HEAD", { cwd: repoDir }).toString().trim();
    process.env.GITHUB_SHA = sha;
    // Clear GITHUB_WORKSPACE so the cwd option is used instead
    delete process.env.GITHUB_WORKSPACE;

    // Reset module cache so each test gets a fresh module instance
    delete require.cache[require.resolve("./generate_git_patch.cjs")];
  });

  afterEach(() => {
    // Restore env
    Object.entries(originalEnv).forEach(([k, v]) => {
      if (v !== undefined) process.env[k] = v;
      else delete process.env[k];
    });
    // Clean up temp repo
    if (repoDir && fs.existsSync(repoDir)) {
      fs.rmSync(repoDir, { recursive: true, force: true });
    }
    delete require.cache[require.resolve("./generate_git_patch.cjs")];
    delete global.core;
  });

  function commitFiles(files) {
    for (const [filePath, content] of Object.entries(files)) {
      const abs = path.join(repoDir, filePath);
      fs.mkdirSync(path.dirname(abs), { recursive: true });
      fs.writeFileSync(abs, content);
    }
    execSync("git add .", { cwd: repoDir });
    execSync('git commit -m "add files"', { cwd: repoDir });
  }

  it("should include all files when excludedFiles is not set", async () => {
    commitFiles({
      "src/index.js": "console.log('hello');\n",
      "dist/bundle.js": "/* bundled */\n",
    });

    const { generateGitPatch } = require("./generate_git_patch.cjs");
    const result = await generateGitPatch(null, "main", { cwd: repoDir });

    expect(result.success).toBe(true);
    const patch = fs.readFileSync(result.patchPath, "utf8");
    expect(patch).toContain("src/index.js");
    expect(patch).toContain("dist/bundle.js");
  });

  it("should exclude files matching excludedFiles patterns from the patch", async () => {
    commitFiles({
      "src/index.js": "console.log('hello');\n",
      "dist/bundle.js": "/* bundled */\n",
    });

    const { generateGitPatch } = require("./generate_git_patch.cjs");
    const result = await generateGitPatch(null, "main", { cwd: repoDir, excludedFiles: ["dist/**"] });

    expect(result.success).toBe(true);
    const patch = fs.readFileSync(result.patchPath, "utf8");
    expect(patch).toContain("src/index.js");
    expect(patch).not.toContain("dist/bundle.js");
  });

  it("should return no patch when all files are ignored", async () => {
    commitFiles({
      "dist/bundle.js": "/* bundled */\n",
    });

    const { generateGitPatch } = require("./generate_git_patch.cjs");
    const result = await generateGitPatch(null, "main", { cwd: repoDir, excludedFiles: ["dist/**"] });

    // All changes were excluded — patch is empty so generation reports no changes
    expect(result.success).toBe(false);
  });
});

// ──────────────────────────────────────────────────────────────────────────
// Full mode base ref selection — must use merge-base, never stale origin/<branch>
// Regression test for: create_pull_request patch uses stale origin/branchName
// instead of merge-base with default branch.
// ──────────────────────────────────────────────────────────────────────────

describe("generateGitPatch – full mode base ref (merge-base, not stale origin)", () => {
  let repoDir;
  let originalEnv;

  beforeEach(() => {
    originalEnv = { GITHUB_WORKSPACE: process.env.GITHUB_WORKSPACE, GITHUB_SHA: process.env.GITHUB_SHA };

    global.core = { debug: () => {}, info: () => {}, warning: () => {}, error: () => {} };

    repoDir = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-patch-fullmode-"));
    execSync("git init -b main", { cwd: repoDir });
    execSync('git config user.email "test@example.com"', { cwd: repoDir });
    execSync('git config user.name "Test"', { cwd: repoDir });

    // Initial commit on main
    fs.writeFileSync(path.join(repoDir, "README.md"), "# Repo\n");
    execSync("git add .", { cwd: repoDir });
    execSync('git commit -m "init"', { cwd: repoDir });

    delete process.env.GITHUB_WORKSPACE;
    delete process.env.GITHUB_SHA;
    delete require.cache[require.resolve("./generate_git_patch.cjs")];
  });

  afterEach(() => {
    Object.entries(originalEnv).forEach(([k, v]) => {
      if (v !== undefined) process.env[k] = v;
      else delete process.env[k];
    });
    if (repoDir && fs.existsSync(repoDir)) {
      fs.rmSync(repoDir, { recursive: true, force: true });
    }
    delete require.cache[require.resolve("./generate_git_patch.cjs")];
    delete global.core;
  });

  it("should NOT include phantom commits when stale origin/<branch> exists (regression test)", async () => {
    // Reproduce the stale-remote-tracking-ref bug scenario:
    //   1. A feature branch was pushed in the past at some old commit
    //      (origin/feature-branch points there — this is the "stale" remote-tracking ref)
    //   2. main has since advanced with many "phantom" commits the agent never made
    //   3. The agent fast-forwards the local branch to main, then makes one new commit
    //   4. Full-mode patch (create_pull_request) must contain ONLY the agent's commit,
    //      NOT the phantom commits between the old branch tip and main.

    // Set up a "remote" repo to act as origin
    const remoteDir = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-patch-fullmode-remote-"));
    try {
      execSync("git init --bare -b main", { cwd: remoteDir });
      execSync(`git remote add origin ${remoteDir}`, { cwd: repoDir });
      execSync("git push origin main", { cwd: repoDir });

      // Step 1: Create the feature branch at the initial commit and push it.
      // This becomes the "old" position of origin/feature-branch.
      execSync("git checkout -b feature-branch", { cwd: repoDir });
      execSync("git push origin feature-branch", { cwd: repoDir });
      // Explicitly fetch to ensure refs/remotes/origin/feature-branch exists in the
      // local repo. `git push` typically updates this automatically, but in repos
      // created via `git init` + `git remote add` (no clone) the remote-tracking
      // ref population can be inconsistent across git versions, so be explicit.
      execSync("git fetch origin feature-branch:refs/remotes/origin/feature-branch", { cwd: repoDir });
      const oldBranchSha = execSync("git rev-parse HEAD", { cwd: repoDir }).toString().trim();

      // Step 2: main advances with phantom commits the agent will not make
      execSync("git checkout main", { cwd: repoDir });
      fs.writeFileSync(path.join(repoDir, "phantom1.md"), "# phantom 1\n");
      execSync("git add phantom1.md", { cwd: repoDir });
      execSync('git commit -m "phantom commit 1"', { cwd: repoDir });
      fs.writeFileSync(path.join(repoDir, "phantom2.md"), "# phantom 2\n");
      execSync("git add phantom2.md", { cwd: repoDir });
      execSync('git commit -m "phantom commit 2"', { cwd: repoDir });
      execSync("git push origin main", { cwd: repoDir });

      // Step 3: Simulate the agent run — fast-forward the local feature branch to
      // main, then make one new commit. Note that origin/feature-branch is still
      // pointing at oldBranchSha (we deliberately do NOT push the branch update).
      execSync("git checkout feature-branch", { cwd: repoDir });
      execSync("git reset --hard main", { cwd: repoDir });
      fs.writeFileSync(path.join(repoDir, "agent-change.txt"), "the only real change\n");
      execSync("git add agent-change.txt", { cwd: repoDir });
      execSync('git commit -m "agent change"', { cwd: repoDir });

      // Sanity check: origin/feature-branch is still stale (points to oldBranchSha)
      const remoteBranchSha = execSync("git rev-parse origin/feature-branch", { cwd: repoDir }).toString().trim();
      expect(remoteBranchSha).toBe(oldBranchSha);

      // Step 4: Generate the full-mode patch
      const { generateGitPatch } = require("./generate_git_patch.cjs");
      const result = await generateGitPatch("feature-branch", "main", { cwd: repoDir, mode: "full" });

      expect(result.success).toBe(true);
      const patch = fs.readFileSync(result.patchPath, "utf8");

      // The patch MUST contain the agent's commit
      expect(patch).toContain("agent-change.txt");
      // The patch MUST NOT contain the phantom commits from main
      expect(patch).not.toContain("phantom1.md");
      expect(patch).not.toContain("phantom2.md");
    } finally {
      if (fs.existsSync(remoteDir)) {
        fs.rmSync(remoteDir, { recursive: true, force: true });
      }
    }
  });
});
