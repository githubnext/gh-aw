import { describe, it, expect, afterEach } from "vitest";
import { createRequire } from "module";
import fs from "fs";
import os from "os";
import path from "path";
import { spawnSync } from "child_process";

const require = createRequire(import.meta.url);

global.core = {
  debug: () => {},
  error: () => {},
  info: () => {},
  warning: () => {},
};

function execGit(args, options = {}) {
  const result = spawnSync("git", args, {
    encoding: "utf8",
    ...options,
  });
  if (result.error) {
    throw result.error;
  }
  if (result.status !== 0 && !options.allowFailure) {
    throw new Error(`git ${args.join(" ")} failed: ${result.stderr}`);
  }
  return result;
}

describe("generateGitBundle (incremental)", () => {
  const tempDirs = [];
  const bundlePaths = [];

  afterEach(() => {
    for (const tempDir of tempDirs.splice(0)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
    for (const bundlePath of bundlePaths.splice(0)) {
      fs.rmSync(bundlePath, { force: true });
    }
  });

  it("reduces bundle size by excluding origin/base branch objects already on remote", async () => {
    const remoteDir = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-bundle-remote-"));
    const workDir = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-bundle-work-"));
    tempDirs.push(remoteDir, workDir);

    execGit(["init", "--bare"], { cwd: remoteDir });
    execGit(["clone", remoteDir, workDir]);
    execGit(["config", "user.name", "Test User"], { cwd: workDir });
    execGit(["config", "user.email", "test@example.com"], { cwd: workDir });

    fs.writeFileSync(path.join(workDir, "base.txt"), "base\n");
    execGit(["add", "base.txt"], { cwd: workDir });
    execGit(["commit", "-m", "base"], { cwd: workDir });
    execGit(["branch", "-M", "main"], { cwd: workDir });
    execGit(["push", "-u", "origin", "main"], { cwd: workDir });

    execGit(["checkout", "-b", "pr-branch"], { cwd: workDir });
    fs.writeFileSync(path.join(workDir, "pr.txt"), "pr start\n");
    execGit(["add", "pr.txt"], { cwd: workDir });
    execGit(["commit", "-m", "pr start"], { cwd: workDir });
    execGit(["push", "-u", "origin", "pr-branch"], { cwd: workDir });

    execGit(["checkout", "main"], { cwd: workDir });
    for (let commitIndex = 0; commitIndex < 4; commitIndex++) {
      fs.writeFileSync(path.join(workDir, `upstream-${commitIndex}.txt`), `upstream ${commitIndex}\n`);
      execGit(["add", `upstream-${commitIndex}.txt`], { cwd: workDir });
      execGit(["commit", "-m", `upstream ${commitIndex}`], { cwd: workDir });
    }
    execGit(["push", "origin", "main"], { cwd: workDir });

    execGit(["checkout", "pr-branch"], { cwd: workDir });
    execGit(["merge", "--no-ff", "origin/main", "-m", "merge main into pr"], { cwd: workDir });
    fs.writeFileSync(path.join(workDir, "resolution.txt"), "resolved\n");
    execGit(["add", "resolution.txt"], { cwd: workDir });
    execGit(["commit", "-m", "resolution commit"], { cwd: workDir });

    const { generateGitBundle } = require("./generate_git_bundle.cjs");
    const result = await generateGitBundle("pr-branch", "main", { mode: "incremental", cwd: workDir });
    expect(result.success).toBe(true);
    expect(result.bundlePath).toBeTruthy();
    bundlePaths.push(result.bundlePath);

    const naiveBundlePath = path.join(workDir, "naive.bundle");
    const optimizedBundlePath = path.join(workDir, "optimized.bundle");
    execGit(["bundle", "create", naiveBundlePath, "origin/pr-branch..pr-branch"], { cwd: workDir });
    execGit(["bundle", "create", optimizedBundlePath, "origin/pr-branch..pr-branch", "^origin/main"], { cwd: workDir });

    const prBranchHeadSha = execGit(["rev-parse", "pr-branch"], { cwd: workDir }).stdout.trim();
    const generatedBundleHeads = execGit(["bundle", "list-heads", result.bundlePath], { cwd: workDir }).stdout.trim();
    const optimizedBundleHeads = execGit(["bundle", "list-heads", optimizedBundlePath], { cwd: workDir }).stdout.trim();
    const generatedSize = fs.statSync(result.bundlePath).size;
    const naiveSize = fs.statSync(naiveBundlePath).size;

    expect(prBranchHeadSha).toBeTruthy();
    expect(prBranchHeadSha).toMatch(/^[a-f0-9]{40}$/);
    expect(optimizedBundleHeads).toContain(prBranchHeadSha);
    expect(generatedBundleHeads).toBe(optimizedBundleHeads);
    expect(generatedSize).toBeLessThan(naiveSize);
  });

  it("falls back to non-exclusion bundle generation when origin/base branch is unavailable", async () => {
    const remoteDir = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-bundle-remote-"));
    const workDir = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-bundle-work-"));
    tempDirs.push(remoteDir, workDir);

    execGit(["init", "--bare"], { cwd: remoteDir });
    execGit(["clone", remoteDir, workDir]);
    execGit(["config", "user.name", "Test User"], { cwd: workDir });
    execGit(["config", "user.email", "test@example.com"], { cwd: workDir });

    execGit(["checkout", "-b", "pr-branch"], { cwd: workDir });
    fs.writeFileSync(path.join(workDir, "pr.txt"), "pr start\n");
    execGit(["add", "pr.txt"], { cwd: workDir });
    execGit(["commit", "-m", "pr start"], { cwd: workDir });
    execGit(["push", "-u", "origin", "pr-branch"], { cwd: workDir });

    fs.writeFileSync(path.join(workDir, "pr-2.txt"), "pr second\n");
    execGit(["add", "pr-2.txt"], { cwd: workDir });
    execGit(["commit", "-m", "pr second"], { cwd: workDir });

    const { generateGitBundle } = require("./generate_git_bundle.cjs");
    const result = await generateGitBundle("pr-branch", "main", { mode: "incremental", cwd: workDir });
    expect(result.success).toBe(true);
    expect(result.bundlePath).toBeTruthy();
    bundlePaths.push(result.bundlePath);

    const naiveBundlePath = path.join(workDir, "naive.bundle");
    execGit(["bundle", "create", naiveBundlePath, "origin/pr-branch..pr-branch"], { cwd: workDir });
    expect(fs.existsSync(naiveBundlePath)).toBe(true);

    const generatedBundleHeads = execGit(["bundle", "list-heads", result.bundlePath], { cwd: workDir }).stdout.trim();
    const naiveBundleHeads = execGit(["bundle", "list-heads", naiveBundlePath], { cwd: workDir }).stdout.trim();

    expect(generatedBundleHeads).toBe(naiveBundleHeads);
  });

  it("returns actionable guidance when branch is missing in incremental mode", async () => {
    const { generateGitBundle } = require("./generate_git_bundle.cjs");

    const result = await generateGitBundle("feature-branch", "main", {
      mode: "incremental",
      cwd: "/tmp/nonexistent-repo",
    });

    expect(result.success).toBe(false);
    expect(result.error).toContain("wrong repository checkout");
    expect(result.error).toContain("GITHUB_WORKSPACE");
    expect(result.error).toContain("/tmp/nonexistent-repo");
  });
});
