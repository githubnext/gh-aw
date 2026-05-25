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

  afterEach(() => {
    for (const tempDir of tempDirs.splice(0)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
  });

  it("excludes origin/base branch objects from incremental bundles when available", async () => {
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
    for (let i = 0; i < 4; i++) {
      fs.writeFileSync(path.join(workDir, `upstream-${i}.txt`), `upstream ${i}\n`);
      execGit(["add", `upstream-${i}.txt`], { cwd: workDir });
      execGit(["commit", "-m", `upstream ${i}`], { cwd: workDir });
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

    const naiveBundlePath = path.join(workDir, "naive.bundle");
    const optimizedBundlePath = path.join(workDir, "optimized.bundle");
    execGit(["bundle", "create", naiveBundlePath, "origin/pr-branch..pr-branch"], { cwd: workDir });
    execGit(["bundle", "create", optimizedBundlePath, "origin/pr-branch..pr-branch", "^origin/main"], { cwd: workDir });

    const generatedSize = fs.statSync(result.bundlePath).size;
    const naiveSize = fs.statSync(naiveBundlePath).size;
    const optimizedSize = fs.statSync(optimizedBundlePath).size;

    expect(generatedSize).toBe(optimizedSize);
    expect(generatedSize).toBeLessThan(naiveSize);
  });
});
