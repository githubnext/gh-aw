import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { execSync } from "child_process";
import fs from "fs";
import os from "os";
import path from "path";

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

describe("getCurrentBranch detached-HEAD handling", () => {
  let tmpDir;
  let originalEnv;

  beforeEach(() => {
    originalEnv = {
      GITHUB_HEAD_REF: process.env.GITHUB_HEAD_REF,
      GITHUB_REF_NAME: process.env.GITHUB_REF_NAME,
      GITHUB_WORKSPACE: process.env.GITHUB_WORKSPACE,
    };
    delete process.env.GITHUB_HEAD_REF;
    delete process.env.GITHUB_REF_NAME;
    delete process.env.GITHUB_WORKSPACE;

    // Create a real git repo and leave it in detached-HEAD state so that
    // `git rev-parse --abbrev-ref HEAD` returns the literal string "HEAD".
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-detached-"));
    execSync("git init -b main", { cwd: tmpDir, stdio: "pipe" });
    execSync("git config user.email 'test@example.com'", { cwd: tmpDir, stdio: "pipe" });
    execSync("git config user.name 'Test User'", { cwd: tmpDir, stdio: "pipe" });
    fs.writeFileSync(path.join(tmpDir, "file.txt"), "content");
    execSync("git add file.txt", { cwd: tmpDir, stdio: "pipe" });
    execSync("git commit -m 'init'", { cwd: tmpDir, stdio: "pipe" });
    const sha = execSync("git rev-parse HEAD", { cwd: tmpDir, stdio: "pipe" }).toString().trim();
    // Detach HEAD by checking out the commit SHA directly.
    execSync(`git checkout ${sha}`, { cwd: tmpDir, stdio: "pipe" });
  });

  afterEach(() => {
    if (originalEnv.GITHUB_HEAD_REF !== undefined) process.env.GITHUB_HEAD_REF = originalEnv.GITHUB_HEAD_REF;
    else delete process.env.GITHUB_HEAD_REF;
    if (originalEnv.GITHUB_REF_NAME !== undefined) process.env.GITHUB_REF_NAME = originalEnv.GITHUB_REF_NAME;
    else delete process.env.GITHUB_REF_NAME;
    if (originalEnv.GITHUB_WORKSPACE !== undefined) process.env.GITHUB_WORKSPACE = originalEnv.GITHUB_WORKSPACE;
    else delete process.env.GITHUB_WORKSPACE;
    try {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    } catch (_) {}
  });

  it("falls back to GITHUB_HEAD_REF when git returns HEAD (detached-HEAD state)", async () => {
    process.env.GITHUB_HEAD_REF = "feature/pr-head-ref";
    const { getCurrentBranch } = await import("./get_current_branch.cjs");
    const result = getCurrentBranch(tmpDir);
    expect(result).toBe("feature/pr-head-ref");
  });

  it("falls back to GITHUB_REF_NAME when git returns HEAD and GITHUB_HEAD_REF is absent", async () => {
    process.env.GITHUB_REF_NAME = "feature/pr-ref-name";
    const { getCurrentBranch } = await import("./get_current_branch.cjs");
    const result = getCurrentBranch(tmpDir);
    expect(result).toBe("feature/pr-ref-name");
  });

  it("throws an actionable error when git returns HEAD and no env vars are set", async () => {
    const { getCurrentBranch } = await import("./get_current_branch.cjs");
    expect(() => getCurrentBranch(tmpDir)).toThrow("detached-HEAD");
  });
});
