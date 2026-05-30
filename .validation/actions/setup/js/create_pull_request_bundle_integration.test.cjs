/**
 * Integration tests for create_pull_request bundle application.
 *
 * These tests run real git commands against temporary repositories to verify
 * bundle handling for checked-out target branches.
 */

import { describe, it, expect, afterEach, vi } from "vitest";
import { createRequire } from "module";
import fs from "fs";
import os from "os";
import path from "path";
import { spawnSync } from "child_process";

const require = createRequire(import.meta.url);

global.core = {
  debug: vi.fn(),
  error: vi.fn(),
  info: vi.fn(),
  warning: vi.fn(),
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

function createRepo(prefix) {
  const repoDir = fs.mkdtempSync(path.join(os.tmpdir(), prefix));
  execGit(["init"], { cwd: repoDir });
  execGit(["config", "user.name", "Test User"], { cwd: repoDir });
  execGit(["config", "user.email", "test@example.com"], { cwd: repoDir });
  return repoDir;
}

function createExecApi(cwd, onExec) {
  return {
    async exec(command, args = []) {
      if (command !== "git") {
        throw new Error(`unexpected command: ${command}`);
      }
      const result = execGit(args, { cwd, allowFailure: true });
      if (result.status !== 0) {
        throw new Error(result.stderr || result.stdout);
      }
      if (onExec) {
        onExec(args);
      }
      return result.status;
    },
    async getExecOutput(command, args = [], options = {}) {
      if (command !== "git") {
        throw new Error(`unexpected command: ${command}`);
      }
      const result = execGit(args, { cwd, allowFailure: true });
      if (result.status !== 0 && !options.ignoreReturnCode) {
        throw new Error(result.stderr || result.stdout);
      }
      if (onExec) {
        onExec(args);
      }
      return { exitCode: result.status, stdout: result.stdout, stderr: result.stderr };
    },
  };
}

describe("create_pull_request bundle integration", () => {
  const tempDirs = [];

  afterEach(() => {
    for (const tempDir of tempDirs.splice(0)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
    vi.clearAllMocks();
  });

  it("applies a bundle when the target branch is currently checked out", async () => {
    const branchName = "autoloop/perf-comparison";
    const sourceRepo = createRepo("create-pr-bundle-source-");
    const targetRepo = createRepo("create-pr-bundle-target-");
    tempDirs.push(sourceRepo, targetRepo);

    fs.writeFileSync(path.join(sourceRepo, "file.txt"), "base\n");
    execGit(["add", "file.txt"], { cwd: sourceRepo });
    execGit(["commit", "-m", "base"], { cwd: sourceRepo });
    execGit(["branch", "-M", "main"], { cwd: sourceRepo });
    execGit(["checkout", "-b", branchName], { cwd: sourceRepo });
    fs.writeFileSync(path.join(sourceRepo, "file.txt"), "bundle tip\n");
    execGit(["commit", "-am", "bundle tip"], { cwd: sourceRepo });
    const expectedHead = execGit(["rev-parse", "HEAD"], { cwd: sourceRepo }).stdout.trim();
    const bundlePath = path.join(sourceRepo, "change.bundle");
    execGit(["bundle", "create", bundlePath, `refs/heads/${branchName}`], { cwd: sourceRepo });

    fs.writeFileSync(path.join(targetRepo, "file.txt"), "checked out branch before bundle\n");
    execGit(["add", "file.txt"], { cwd: targetRepo });
    execGit(["commit", "-m", "old branch state"], { cwd: targetRepo });
    execGit(["checkout", "-b", branchName], { cwd: targetRepo });

    const checkedOutBranchFetchResult = execGit(["fetch", bundlePath, `refs/heads/${branchName}:refs/heads/${branchName}`], { cwd: targetRepo, allowFailure: true });
    expect(checkedOutBranchFetchResult.status).not.toBe(0);
    expect(checkedOutBranchFetchResult.stderr).toContain("refusing to fetch into branch");

    let bundleTempRef = "";
    const { applyBundleToBranch } = require("./create_pull_request.cjs");
    await applyBundleToBranch(
      bundlePath,
      branchName,
      "",
      createExecApi(targetRepo, args => {
        if (args[0] === "fetch" && args[1] === bundlePath) {
          bundleTempRef = args[2].split(":")[1];
          expect(execGit(["show-ref", "--verify", bundleTempRef], { cwd: targetRepo }).status).toBe(0);
        }
      })
    );

    const actualHead = execGit(["rev-parse", "HEAD"], { cwd: targetRepo }).stdout.trim();
    expect(actualHead).toBe(expectedHead);
    expect(fs.readFileSync(path.join(targetRepo, "file.txt"), "utf8")).toBe("bundle tip\n");
    expect(bundleTempRef).toMatch(/^refs\/bundles\/create-pr-autoloop-perf-comparison-[a-f0-9]{8}$/);
    expect(execGit(["show-ref", "--verify", bundleTempRef], { cwd: targetRepo, allowFailure: true }).status).not.toBe(0);
  });

  it("cleans up the temp ref when updating the target branch fails", async () => {
    const branchName = "autoloop/perf-comparison";
    const sourceRepo = createRepo("create-pr-bundle-source-");
    const targetRepo = createRepo("create-pr-bundle-target-");
    tempDirs.push(sourceRepo, targetRepo);

    fs.writeFileSync(path.join(sourceRepo, "file.txt"), "base\n");
    execGit(["add", "file.txt"], { cwd: sourceRepo });
    execGit(["commit", "-m", "base"], { cwd: sourceRepo });
    execGit(["branch", "-M", "main"], { cwd: sourceRepo });
    execGit(["checkout", "-b", branchName], { cwd: sourceRepo });
    fs.writeFileSync(path.join(sourceRepo, "file.txt"), "bundle tip\n");
    execGit(["commit", "-am", "bundle tip"], { cwd: sourceRepo });
    const bundlePath = path.join(sourceRepo, "change.bundle");
    execGit(["bundle", "create", bundlePath, `refs/heads/${branchName}`], { cwd: sourceRepo });

    fs.writeFileSync(path.join(targetRepo, "file.txt"), "old branch state\n");
    execGit(["add", "file.txt"], { cwd: targetRepo });
    execGit(["commit", "-m", "old branch state"], { cwd: targetRepo });
    execGit(["checkout", "-b", branchName], { cwd: targetRepo });
    const originalHead = execGit(["rev-parse", `refs/heads/${branchName}`], { cwd: targetRepo }).stdout.trim();

    let bundleTempRef = "";
    const execApi = createExecApi(targetRepo, args => {
      if (args[0] === "fetch" && args[1] === bundlePath) {
        bundleTempRef = args[2].split(":")[1];
      }
    });
    const { applyBundleToBranch } = require("./create_pull_request.cjs");

    await expect(
      applyBundleToBranch(bundlePath, branchName, "", {
        ...execApi,
        async exec(command, args = []) {
          if (command === "git" && args[0] === "update-ref" && args[1] === `refs/heads/${branchName}`) {
            throw new Error("simulated update-ref failure");
          }
          return execApi.exec(command, args);
        },
      })
    ).rejects.toThrow("simulated update-ref failure");

    expect(bundleTempRef).toMatch(/^refs\/bundles\/create-pr-autoloop-perf-comparison-[a-f0-9]{8}$/);
    expect(execGit(["show-ref", "--verify", bundleTempRef], { cwd: targetRepo, allowFailure: true }).status).not.toBe(0);
    expect(execGit(["rev-parse", `refs/heads/${branchName}`], { cwd: targetRepo }).stdout.trim()).toBe(originalHead);
  });

  it("applies bundle route with merge-commit history intact", async () => {
    const branchName = "autoloop/merge-bundle";
    const sourceRepo = createRepo("create-pr-bundle-merge-source-");
    const targetRepo = createRepo("create-pr-bundle-merge-target-");
    tempDirs.push(sourceRepo, targetRepo);

    fs.writeFileSync(path.join(sourceRepo, "file.txt"), "base\n");
    execGit(["add", "file.txt"], { cwd: sourceRepo });
    execGit(["commit", "-m", "base"], { cwd: sourceRepo });
    execGit(["branch", "-M", "main"], { cwd: sourceRepo });

    execGit(["checkout", "-b", "feature"], { cwd: sourceRepo });
    fs.writeFileSync(path.join(sourceRepo, "feature.txt"), "feature branch commit\n");
    execGit(["add", "feature.txt"], { cwd: sourceRepo });
    execGit(["commit", "-m", "feature commit"], { cwd: sourceRepo });

    execGit(["checkout", "main"], { cwd: sourceRepo });
    fs.writeFileSync(path.join(sourceRepo, "main.txt"), "main branch commit\n");
    execGit(["add", "main.txt"], { cwd: sourceRepo });
    execGit(["commit", "-m", "main commit"], { cwd: sourceRepo });
    execGit(["merge", "--no-ff", "feature", "-m", "merge feature"], { cwd: sourceRepo });
    execGit(["checkout", "-b", branchName], { cwd: sourceRepo });

    const expectedHead = execGit(["rev-parse", "HEAD"], { cwd: sourceRepo }).stdout.trim();
    const bundlePath = path.join(sourceRepo, "merge.bundle");
    execGit(["bundle", "create", bundlePath, `refs/heads/${branchName}`], { cwd: sourceRepo });

    fs.writeFileSync(path.join(targetRepo, "file.txt"), "target divergent history\n");
    execGit(["add", "file.txt"], { cwd: targetRepo });
    execGit(["commit", "-m", "target state"], { cwd: targetRepo });
    execGit(["checkout", "-b", branchName], { cwd: targetRepo });

    const { applyBundleToBranch } = require("./create_pull_request.cjs");
    await applyBundleToBranch(bundlePath, branchName, "", createExecApi(targetRepo));

    const actualHead = execGit(["rev-parse", "HEAD"], { cwd: targetRepo }).stdout.trim();
    const mergeCount = Number(execGit(["rev-list", "--count", "--merges", "HEAD"], { cwd: targetRepo }).stdout.trim());
    expect(actualHead).toBe(expectedHead);
    expect(mergeCount).toBeGreaterThanOrEqual(1);
    expect(fs.readFileSync(path.join(targetRepo, "feature.txt"), "utf8")).toBe("feature branch commit\n");
    expect(fs.readFileSync(path.join(targetRepo, "main.txt"), "utf8")).toBe("main branch commit\n");
  });
});
