import { describe, it, expect, afterEach, vi } from "vitest";
import { createRequire } from "module";
import fs from "fs";
import os from "os";
import path from "path";
import { spawnSync } from "child_process";

const require = createRequire(import.meta.url);
const { getBundlePreApplyFiles, getBundleValidationBase, resolveBundleTrustedBase } = require("./push_to_pull_request_branch.cjs");
const { checkFileProtectionPostApply } = require("./manifest_file_helpers.cjs");

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

function writeRepoFile(repoDir, relativePath, content) {
  const fullPath = path.join(repoDir, relativePath);
  fs.mkdirSync(path.dirname(fullPath), { recursive: true });
  fs.writeFileSync(fullPath, content);
}

function createExecApi(cwd) {
  return {
    async getExecOutput(command, args = [], options = {}) {
      if (command !== "git") {
        throw new Error(`unexpected command: ${command}`);
      }
      const result = execGit(args, { cwd, allowFailure: true });
      if (result.status !== 0 && !options.ignoreReturnCode) {
        throw new Error(result.stderr || result.stdout);
      }
      return { exitCode: result.status, stdout: result.stdout, stderr: result.stderr };
    },
  };
}

function fetchBaseCommit(targetRepo, sourceRepo, baseSha, branchName) {
  execGit(["remote", "add", "origin", sourceRepo], { cwd: targetRepo });
  execGit(["fetch", "origin", baseSha], { cwd: targetRepo });
  execGit(["checkout", "-b", branchName, "FETCH_HEAD"], { cwd: targetRepo });
}

describe("push_to_pull_request_branch bundle integration", () => {
  const tempDirs = [];

  afterEach(() => {
    for (const tempDir of tempDirs.splice(0)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
    vi.clearAllMocks();
  });

  it.each([
    [false, false],
    [true, false],
    [false, true],
    [false, false, true],
    [true, false, true],
    [false, true, true],
  ])("excludes trusted base changes while counting agent changes (conflict: %s, reverted change: %s, advanced base: %s)", async (conflict, revertedChange, baseAdvanced = false) => {
    const repo = createRepo("push-pr-validation-base-");
    tempDirs.push(repo);
    const git = args => execGit(args, { cwd: repo });
    writeRepoFile(repo, "conflict.txt", "original\n");
    git(["add", "."]);
    git(["commit", "-m", "common base"]);
    git(["branch", "-M", "main"]);
    git(["checkout", "-b", "feature"]);
    writeRepoFile(repo, "existing-pr.txt", "existing PR work\n");
    if (conflict) writeRepoFile(repo, "conflict.txt", "PR version\n");
    git(["add", "."]);
    git(["commit", "-m", "existing PR"]);
    const oldHead = git(["rev-parse", "HEAD"]).stdout.trim();
    git(["checkout", "main"]);
    writeRepoFile(repo, "upstream.txt", "upstream\n".repeat(10000));
    writeRepoFile(repo, "package.json", "{}\n");
    if (conflict) writeRepoFile(repo, "conflict.txt", "base version\n");
    git(["add", "."]);
    git(["commit", "-m", "trusted upstream changes"]);
    const trustedBase = git(["rev-parse", "HEAD"]).stdout.trim();
    git(["checkout", "feature"]);
    const merge = execGit(["merge", "--no-ff", "main", "-m", "merge base"], { cwd: repo, allowFailure: true });
    expect(merge.status).toBe(conflict ? 1 : 0);
    if (conflict) {
      writeRepoFile(repo, "conflict.txt", "agent resolution\n");
      git(["add", "."]);
      git(["commit", "-m", "resolve conflict"]);
    }
    if (revertedChange) {
      writeRepoFile(repo, "package.json", '{"agent":true}\n');
      git(["commit", "-am", "agent modifies protected file"]);
      writeRepoFile(repo, "package.json", "{}\n");
      git(["commit", "-am", "agent reverts protected file"]);
    }
    writeRepoFile(repo, "agent.txt", "agent change\n");
    git(["add", "."]);
    git(["commit", "-m", "agent work"]);

    const exec = createExecApi(repo);
    let trustedTip = trustedBase;
    if (baseAdvanced) {
      git(["checkout", "main"]);
      writeRepoFile(repo, "later.txt", "later base work\n");
      git(["add", "."]);
      git(["commit", "-m", "base advances after agent merge"]);
      trustedTip = git(["rev-parse", "HEAD"]).stdout.trim();
      git(["checkout", "feature"]);
    }
    const compareCommits = vi.fn(async ({ base, head }) => {
      expect(base).toBe(trustedTip);
      const ancestor = execGit(["merge-base", "--is-ancestor", head, base], { cwd: repo, allowFailure: true });
      return { data: { status: ancestor.status === 0 ? "behind" : "diverged" } };
    });
    const recognizedBase = await resolveBundleTrustedBase(exec, {}, oldHead, "HEAD", trustedTip, { rest: { repos: { compareCommits } } }, { owner: "test-owner", repo: "test-repo" });
    expect(recognizedBase).toBe(trustedBase);
    expect(compareCommits).toHaveBeenCalledTimes(baseAdvanced ? 1 : 0);
    const { ref: baseline, conflictedFiles } = await getBundleValidationBase(exec, {}, oldHead, "HEAD", recognizedBase);
    expect(baseline).not.toBe(oldHead);
    const files = await getBundlePreApplyFiles(exec, {}, baseline, "HEAD", conflictedFiles);
    expect(files.sort()).toEqual(conflict ? ["agent.txt", "conflict.txt"] : ["agent.txt"]);
    expect(checkFileProtectionPostApply(files, { protected_files: ["package.json"] }).action).toBe("allow");
    const originalPatch = git(["format-patch", "--stdout", `${oldHead}..HEAD`]).stdout;
    const agentPatch = git(["diff", "--binary", baseline, "HEAD"]).stdout;
    expect(Buffer.byteLength(originalPatch)).toBeGreaterThan(10240);
    expect(Buffer.byteLength(agentPatch)).toBeLessThan(1024);
    expect(files.reduce((size, file) => size + fs.statSync(path.join(repo, file)).size, 0)).toBeLessThan(1024);
    expect(git(["rev-list", "--count", "--merges", `${oldHead}..HEAD`]).stdout.trim()).toBe("1");
    const historyFiles = git(["log", "--format=", "--name-only", "--no-renames", "--diff-merges=remerge", "-z", "HEAD", "--not", oldHead, trustedBase]).stdout.split("\0").filter(Boolean);
    expect([...new Set(historyFiles)].sort()).toEqual(revertedChange ? ["agent.txt", "package.json"] : conflict ? ["agent.txt", "conflict.txt"] : ["agent.txt"]);
    expect(checkFileProtectionPostApply([...files, ...historyFiles], { protected_files: ["package.json"] }).action).toBe(revertedChange ? "deny" : "allow");
  });

  it.each([false, true])("detects protected changes made and restored only in agent merge commits (octopus: %s)", async octopus => {
    const repo = createRepo("push-pr-merge-history-protection-");
    tempDirs.push(repo);
    const git = args => execGit(args, { cwd: repo });
    writeRepoFile(repo, "package.json", "{}\n");
    git(["add", "."]);
    git(["commit", "-m", "base"]);
    git(["branch", "-M", "main"]);
    git(["checkout", "-b", "feature"]);
    writeRepoFile(repo, "existing.txt", "PR work\n");
    git(["add", "."]);
    git(["commit", "-m", "existing PR"]);
    const oldHead = git(["rev-parse", "HEAD"]).stdout.trim();
    git(["checkout", "main"]);
    writeRepoFile(repo, "upstream.txt", "base work\n");
    git(["add", "."]);
    git(["commit", "-m", "trusted base work"]);
    const trustedBase = git(["rev-parse", "HEAD"]).stdout.trim();
    if (octopus) {
      git(["checkout", "-b", "first-side", oldHead]);
      writeRepoFile(repo, "first-side.txt", "first side\n");
      git(["add", "."]);
      git(["commit", "-m", "first side"]);
    }
    git(["checkout", "feature"]);
    git(["merge", "--no-ff", "--no-commit", "main", ...(octopus ? ["first-side"] : [])]);
    writeRepoFile(repo, "package.json", '{"agent":true}\n');
    git(["commit", "-am", "merge-only protected change"]);
    git(["checkout", "-b", "side"]);
    writeRepoFile(repo, "side.txt", "side work\n");
    git(["add", "."]);
    git(["commit", "-m", "side work"]);
    if (octopus) {
      git(["checkout", "-b", "second-side", "feature"]);
      writeRepoFile(repo, "second-side.txt", "second side\n");
      git(["add", "."]);
      git(["commit", "-m", "second side"]);
    }
    git(["checkout", "feature"]);
    git(["merge", "--no-ff", "--no-commit", "side", ...(octopus ? ["second-side"] : [])]);
    writeRepoFile(repo, "package.json", "{}\n");
    git(["commit", "-am", "merge-only protected restoration"]);

    const exec = createExecApi(repo);
    const { ref, conflictedFiles } = await getBundleValidationBase(exec, {}, oldHead, "HEAD", trustedBase);
    const files = await getBundlePreApplyFiles(exec, {}, ref, "HEAD", conflictedFiles);
    expect(files.sort()).toEqual(octopus ? ["first-side.txt", "second-side.txt", "side.txt"] : ["side.txt"]);
    const octopusCommits = git(["rev-list", "--min-parents=3", "HEAD", "--not", oldHead, trustedBase]).stdout.trim();
    if (octopus) {
      expect(octopusCommits.split("\n")).toHaveLength(2);
      return;
    }
    expect(octopusCommits).toBe("");
    const historyFiles = git(["log", "--format=", "--name-only", "--no-renames", "--diff-merges=remerge", "-z", "HEAD", "--not", oldHead, trustedBase]).stdout.split("\0").filter(Boolean);
    expect(historyFiles).toContain("package.json");
    expect(historyFiles).not.toContain("upstream.txt");
    expect(checkFileProtectionPostApply([...files, ...historyFiles], { protected_files: ["package.json"] }).action).toBe("deny");
  });

  it.each(["text", "binary", "modify/delete"])("keeps protected %s conflict resolutions in the validation file set", async kind => {
    const repo = createRepo("push-pr-protected-resolution-");
    tempDirs.push(repo);
    const git = args => execGit(args, { cwd: repo });
    writeRepoFile(repo, "package.json", kind === "binary" ? Buffer.from([0, 1]) : '{"version":"1"}\n');
    git(["add", "."]);
    git(["commit", "-m", "base"]);
    git(["branch", "-M", "main"]);
    git(["checkout", "-b", "feature"]);
    const prContent = kind === "binary" ? Buffer.from([0, 2]) : '{"version":"2"}\n';
    writeRepoFile(repo, "package.json", prContent);
    git(["commit", "-am", "PR version"]);
    const oldHead = git(["rev-parse", "HEAD"]).stdout.trim();
    git(["checkout", "main"]);
    if (kind === "modify/delete") {
      git(["rm", "package.json"]);
    } else {
      writeRepoFile(repo, "package.json", kind === "binary" ? Buffer.from([0, 3]) : '{"version":"3"}\n');
    }
    git(["commit", "-am", "base version"]);
    const trustedBase = git(["rev-parse", "HEAD"]).stdout.trim();
    git(["checkout", "feature"]);
    expect(execGit(["merge", "main"], { cwd: repo, allowFailure: true }).status).toBe(1);
    writeRepoFile(repo, "package.json", kind === "text" ? '{"version":"4"}\n' : prContent);
    git(["add", "package.json"]);
    git(["commit", "-m", "agent resolution"]);
    const exec = createExecApi(repo);
    const { ref: baseline, conflictedFiles } = await getBundleValidationBase(exec, {}, oldHead, "HEAD", trustedBase);
    const files = await getBundlePreApplyFiles(exec, {}, baseline, "HEAD", conflictedFiles);
    expect(conflictedFiles).toEqual(["package.json"]);
    if (kind !== "text") expect(git(["diff", "--name-only", baseline, "HEAD"]).stdout).toBe("");
    expect(files).toEqual(["package.json"]);
    expect(checkFileProtectionPostApply(files, { protected_files: ["package.json"] }).action).toBe("deny");
  });

  it("does not adjust the baseline for absent, unmerged, already merged, or unrelated base commits", async () => {
    const repo = createRepo("push-pr-baseline-fallback-");
    tempDirs.push(repo);
    const git = args => execGit(args, { cwd: repo });
    writeRepoFile(repo, "base.txt", "base\n");
    git(["add", "."]);
    git(["commit", "-m", "base"]);
    const oldHead = git(["rev-parse", "HEAD"]).stdout.trim();
    git(["checkout", "-b", "other"]);
    writeRepoFile(repo, "other.txt", "unmerged\n");
    git(["add", "."]);
    git(["commit", "-m", "unmerged"]);
    const unmerged = git(["rev-parse", "HEAD"]).stdout.trim();
    git(["checkout", "-b", "candidate", oldHead]);
    writeRepoFile(repo, "agent.txt", "agent\n");
    git(["add", "."]);
    git(["commit", "-m", "agent"]);
    const exec = createExecApi(repo);
    for (const trustedBase of ["", "f".repeat(40), unmerged, oldHead]) {
      expect(await getBundleValidationBase(exec, {}, oldHead, "HEAD", trustedBase)).toEqual({ ref: oldHead, conflictedFiles: [] });
    }
    expect(await getBundleValidationBase(exec, {}, unmerged, "HEAD", oldHead)).toEqual({ ref: unmerged, conflictedFiles: [] });
  });

  it("lists files from a fetched bundle before applying it", async () => {
    const branchName = "autoloop/simple-bundle";
    const sourceRepo = createRepo("push-pr-bundle-source-");
    const targetRepo = createRepo("push-pr-bundle-target-");
    tempDirs.push(sourceRepo, targetRepo);

    writeRepoFile(sourceRepo, "README.md", "base\n");
    execGit(["add", "README.md"], { cwd: sourceRepo });
    execGit(["commit", "-m", "base"], { cwd: sourceRepo });
    execGit(["branch", "-M", "main"], { cwd: sourceRepo });
    const baseSha = execGit(["rev-parse", "HEAD"], { cwd: sourceRepo }).stdout.trim();

    execGit(["checkout", "-b", branchName], { cwd: sourceRepo });
    writeRepoFile(sourceRepo, ".changeset/fix.md", "patch\n");
    writeRepoFile(sourceRepo, "docs/guide.md", "guide\n");
    execGit(["add", ".changeset/fix.md", "docs/guide.md"], { cwd: sourceRepo });
    execGit(["commit", "-m", "bundle change"], { cwd: sourceRepo });

    const bundlePath = path.join(sourceRepo, "bundle.bundle");
    execGit(["bundle", "create", bundlePath, `refs/heads/${branchName}`], { cwd: sourceRepo });

    fetchBaseCommit(targetRepo, sourceRepo, baseSha, branchName);
    const bundleRef = "refs/bundles/test-simple-bundle";
    execGit(["fetch", bundlePath, `refs/heads/${branchName}:${bundleRef}`], { cwd: targetRepo });

    const actualFiles = await getBundlePreApplyFiles(createExecApi(targetRepo), {}, baseSha, bundleRef);

    expect(actualFiles.sort()).toEqual([".changeset/fix.md", "docs/guide.md"]);
  });

  it("checks only bundle commits added after an existing PR head", async () => {
    const branchName = "feature/existing-pr";
    const sourceRepo = createRepo("push-pr-existing-source-");
    const targetRepo = createRepo("push-pr-existing-target-");
    tempDirs.push(sourceRepo, targetRepo);

    writeRepoFile(sourceRepo, "README.md", "base\n");
    execGit(["add", "README.md"], { cwd: sourceRepo });
    execGit(["commit", "-m", "base"], { cwd: sourceRepo });
    execGit(["checkout", "-b", branchName], { cwd: sourceRepo });
    writeRepoFile(sourceRepo, "tools/__pycache__/author.pyc", "author file\n");
    execGit(["add", "tools/__pycache__/author.pyc"], { cwd: sourceRepo });
    execGit(["commit", "-m", "author change"], { cwd: sourceRepo });
    const existingHead = execGit(["rev-parse", "HEAD"], { cwd: sourceRepo }).stdout.trim();

    writeRepoFile(sourceRepo, "config/agent.json", "{}\n");
    execGit(["add", "config/agent.json"], { cwd: sourceRepo });
    execGit(["commit", "-m", "agent change"], { cwd: sourceRepo });
    const bundlePath = path.join(sourceRepo, "incremental.bundle");
    execGit(["bundle", "create", bundlePath, `refs/heads/${branchName}`], { cwd: sourceRepo });

    fetchBaseCommit(targetRepo, sourceRepo, existingHead, branchName);
    const bundleRef = "refs/bundles/test-existing-pr";
    execGit(["fetch", bundlePath, `refs/heads/${branchName}:${bundleRef}`], { cwd: targetRepo });
    const files = await getBundlePreApplyFiles(createExecApi(targetRepo), {}, existingHead, bundleRef);

    expect(files).toEqual(["config/agent.json"]);
    expect(checkFileProtectionPostApply(files, { allowed_files: ["**/*.json"] }).action).toBe("allow");
  });

  it("fetches a HEAD-only bundle after its named branch ref is absent", () => {
    const branchName = "autoloop/head-only-bundle";
    const sourceRepo = createRepo("push-pr-head-only-source-");
    const targetRepo = createRepo("push-pr-head-only-target-");
    tempDirs.push(sourceRepo, targetRepo);

    writeRepoFile(sourceRepo, "README.md", "base\n");
    execGit(["add", "README.md"], { cwd: sourceRepo });
    execGit(["commit", "-m", "base"], { cwd: sourceRepo });
    execGit(["branch", "-M", "main"], { cwd: sourceRepo });
    const baseSha = execGit(["rev-parse", "HEAD"], { cwd: sourceRepo }).stdout.trim();

    execGit(["checkout", "-b", branchName], { cwd: sourceRepo });
    writeRepoFile(sourceRepo, "head-only.txt", "bundle change\n");
    execGit(["add", "head-only.txt"], { cwd: sourceRepo });
    execGit(["commit", "-m", "HEAD-only bundle change"], { cwd: sourceRepo });
    const bundleHead = execGit(["rev-parse", "HEAD"], { cwd: sourceRepo }).stdout.trim();

    const bundlePath = path.join(sourceRepo, "head-only.bundle");
    execGit(["bundle", "create", bundlePath, "HEAD"], { cwd: sourceRepo });

    fetchBaseCommit(targetRepo, sourceRepo, baseSha, branchName);
    const bundleRef = "refs/bundles/test-head-only-bundle";
    const namedRefFetch = execGit(["fetch", bundlePath, `refs/heads/${branchName}:${bundleRef}`], { cwd: targetRepo, allowFailure: true });
    expect(namedRefFetch.status).not.toBe(0);
    expect(execGit(["bundle", "list-heads", bundlePath], { cwd: targetRepo }).stdout).toBe(`${bundleHead} HEAD\n`);

    execGit(["fetch", bundlePath, `HEAD:${bundleRef}`], { cwd: targetRepo });

    expect(execGit(["rev-parse", bundleRef], { cwd: targetRepo }).stdout.trim()).toBe(bundleHead);
  });

  it("includes files introduced through merge-commit bundle history", async () => {
    const branchName = "autoloop/merge-bundle";
    const sourceRepo = createRepo("push-pr-merge-source-");
    const targetRepo = createRepo("push-pr-merge-target-");
    tempDirs.push(sourceRepo, targetRepo);

    writeRepoFile(sourceRepo, "README.md", "base\n");
    execGit(["add", "README.md"], { cwd: sourceRepo });
    execGit(["commit", "-m", "base"], { cwd: sourceRepo });
    execGit(["branch", "-M", "main"], { cwd: sourceRepo });
    const baseSha = execGit(["rev-parse", "HEAD"], { cwd: sourceRepo }).stdout.trim();

    execGit(["checkout", "-b", "feature"], { cwd: sourceRepo });
    writeRepoFile(sourceRepo, "feature.txt", "feature branch change\n");
    execGit(["add", "feature.txt"], { cwd: sourceRepo });
    execGit(["commit", "-m", "feature commit"], { cwd: sourceRepo });

    execGit(["checkout", "main"], { cwd: sourceRepo });
    writeRepoFile(sourceRepo, "main.txt", "main branch change\n");
    execGit(["add", "main.txt"], { cwd: sourceRepo });
    execGit(["commit", "-m", "main commit"], { cwd: sourceRepo });
    execGit(["merge", "--no-ff", "feature", "-m", "merge feature"], { cwd: sourceRepo });
    execGit(["checkout", "-b", branchName], { cwd: sourceRepo });

    const bundlePath = path.join(sourceRepo, "merge.bundle");
    execGit(["bundle", "create", bundlePath, `refs/heads/${branchName}`], { cwd: sourceRepo });

    fetchBaseCommit(targetRepo, sourceRepo, baseSha, branchName);
    const bundleRef = "refs/bundles/test-merge-bundle";
    execGit(["fetch", bundlePath, `refs/heads/${branchName}:${bundleRef}`], { cwd: targetRepo });

    const actualFiles = await getBundlePreApplyFiles(createExecApi(targetRepo), {}, baseSha, bundleRef);

    expect(actualFiles.sort()).toEqual(["feature.txt", "main.txt"]);
  });

  it("resolves filenames that git would otherwise quote (non-ASCII characters)", async () => {
    const branchName = "autoloop/quoted-path-bundle";
    const sourceRepo = createRepo("push-pr-quoted-source-");
    const targetRepo = createRepo("push-pr-quoted-target-");
    tempDirs.push(sourceRepo, targetRepo);

    writeRepoFile(sourceRepo, "README.md", "base\n");
    execGit(["add", "README.md"], { cwd: sourceRepo });
    execGit(["commit", "-m", "base"], { cwd: sourceRepo });
    execGit(["branch", "-M", "main"], { cwd: sourceRepo });
    const baseSha = execGit(["rev-parse", "HEAD"], { cwd: sourceRepo }).stdout.trim();

    execGit(["checkout", "-b", branchName], { cwd: sourceRepo });
    const quotedFileName = "résumé.txt";
    writeRepoFile(sourceRepo, quotedFileName, "quoted path change\n");
    execGit(["add", quotedFileName], { cwd: sourceRepo });
    execGit(["commit", "-m", "quoted path change"], { cwd: sourceRepo });

    const bundlePath = path.join(sourceRepo, "quoted.bundle");
    execGit(["bundle", "create", bundlePath, `refs/heads/${branchName}`], { cwd: sourceRepo });

    fetchBaseCommit(targetRepo, sourceRepo, baseSha, branchName);
    const bundleRef = "refs/bundles/test-quoted-bundle";
    execGit(["fetch", bundlePath, `refs/heads/${branchName}:${bundleRef}`], { cwd: targetRepo });

    const actualFiles = await getBundlePreApplyFiles(createExecApi(targetRepo), {}, baseSha, bundleRef);

    // Without NUL-delimited (`-z`) diff output, git would quote this path (e.g. "r\303\251sum\303\251.txt"),
    // causing downstream filesystem lookups keyed on the raw name to fail and silently drop the file
    // from size/protection checks.
    expect(actualFiles).toEqual([quotedFileName]);
  });
});
