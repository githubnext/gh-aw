/**
 * Regression tests for create_pull_request validation/push file-set parity.
 *
 * Root cause (tracked in GitHub issue github/gh-aw#48999):
 *   Validation and push do not operate on the same object, so a passing patch
 *   validation does not reliably predict what files land on the remote.
 *
 * Two failure modes are tested:
 *
 *   1. Excluded-files path (non-rewrite):
 *      excluded-files patterns are applied to the patch but NOT to the bundle.
 *      The pushed commit therefore contains more files than the validated patch.
 *
 *   2. Merge-commit rewrite path:
 *      After applyBundleToBranch, linearizeRangeAsCommit performs a soft-reset
 *      to origin/base and re-commits ALL staged files — including excluded ones.
 *      The rewritten commit again contains more files than the validated patch.
 *
 * Both regression tests assert: file_set(patch) == file_set(pushed_commit).
 *
 * Both regressions now pass: filtered bundle synthesis keeps excluded files out
 * of the pushed commit, and the merge-commit rewrite path linearizes against
 * the bundle prerequisite so base-branch drift is not absorbed into the result.
 */

import { describe, it, expect, beforeAll, afterEach, vi } from "vitest";
import { createRequire } from "module";
import { fileURLToPath } from "url";
import fs from "fs";
import os from "os";
import path from "path";
import { spawnSync } from "child_process";

const require = createRequire(import.meta.url);
const promptsSourceDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../md");

global.core = {
  debug: vi.fn(),
  error: vi.fn(),
  info: vi.fn(),
  warning: vi.fn(),
};

/**
 * `create_pull_request.cjs` reads the disclosure-header prompt template at
 * module load time. Ensure the file is present before the first `require` so
 * the module can be loaded successfully. Re-creating the temp prompt directory
 * is intentional and idempotent (`mkdirSync(..., { recursive: true })`).
 */
function ensureDisclosureHeaderPrompt() {
  const promptsDir = path.join(process.env.RUNNER_TEMP || os.tmpdir(), "gh-aw", "prompts");
  fs.mkdirSync(promptsDir, { recursive: true });
  fs.copyFileSync(path.join(promptsSourceDir, "safe_outputs_disclosure_header.md"), path.join(promptsDir, "safe_outputs_disclosure_header.md"));
}

// ─── git helpers ─────────────────────────────────────────────────────────────

function execGit(args, options = {}) {
  const result = spawnSync("git", args, { encoding: "utf8", ...options });
  if (result.error) throw result.error;
  if (result.status !== 0 && !options.allowFailure) {
    throw new Error(`git ${args.join(" ")} failed:\n${result.stderr}`);
  }
  return result;
}

function createBareRepo(prefix) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), prefix));
  execGit(["init", "--bare", "-b", "main"], { cwd: dir });
  return dir;
}

function cloneRepo(remoteUrl, targetDir) {
  execGit(["clone", remoteUrl, "."], { cwd: targetDir });
  execGit(["config", "user.name", "Test"], { cwd: targetDir });
  execGit(["config", "user.email", "test@example.com"], { cwd: targetDir });
}

function createExecApi(cwd) {
  return {
    async exec(command, args = []) {
      if (command !== "git") throw new Error(`unexpected command: ${command}`);
      const result = execGit(args, { cwd, allowFailure: true });
      if (result.status !== 0) throw new Error(`git ${args.join(" ")} failed:\n${result.stderr || result.stdout}`);
      return result.status;
    },
    async getExecOutput(command, args = [], options = {}) {
      if (command !== "git") throw new Error(`unexpected command: ${command}`);
      const result = execGit(args, { cwd, allowFailure: true });
      if (result.status !== 0 && !options.ignoreReturnCode) {
        throw new Error(`git ${args.join(" ")} failed:\n${result.stderr || result.stdout}`);
      }
      return { exitCode: result.status, stdout: result.stdout, stderr: result.stderr };
    },
  };
}

// ─── file-set extraction helpers ─────────────────────────────────────────────

/**
 * Extract the sorted array of unique file paths touched by a patch.
 * Parses `diff --git` headers; works for additions, modifications, and deletions.
 * @param {string} patchContent
 * @returns {string[]}
 */
function fileListFromPatch(patchContent) {
  const { extractDiffGitHeaderEntries } = require("./patch_path_helpers.cjs");
  const entries = extractDiffGitHeaderEntries(patchContent);
  const files = new Set();
  for (const entry of entries) {
    const file = entry.newPath || entry.oldPath;
    if (file) files.add(file);
  }
  return [...files].sort();
}

function trackArtifactPath(result, key, createdArtifacts) {
  if (result && typeof result[key] === "string" && result[key]) {
    createdArtifacts.push(result[key]);
  }
}

/**
 * Return the sorted list of files changed between a base ref and HEAD via
 * `git diff --name-only <base>..HEAD`.
 * @param {string} cwd
 * @param {string} baseRef e.g. "origin/main"
 * @returns {string[]}
 */
function fileListFromPushedCommit(cwd, baseRef) {
  const { stdout } = execGit(["diff", "--name-only", `${baseRef}..HEAD`], { cwd });
  return stdout
    .split("\n")
    .map(f => f.trim())
    .filter(Boolean)
    .sort();
}

// ─── tests ───────────────────────────────────────────────────────────────────

describe("create_pull_request – validation/push file-set parity", () => {
  const tempDirs = [];
  const createdArtifacts = [];

  beforeAll(() => {
    // create_pull_request.cjs reads the disclosure-header prompt at load time;
    // set it up once before any require() calls.
    ensureDisclosureHeaderPrompt();
  });

  afterEach(() => {
    for (const dir of tempDirs.splice(0)) {
      fs.rmSync(dir, { recursive: true, force: true });
    }
    for (const p of createdArtifacts.splice(0)) {
      try {
        fs.rmSync(p, { force: true });
      } catch {
        // best-effort cleanup
      }
    }
    vi.clearAllMocks();
  });

  /**
   * Positive sanity check: when no files are excluded the patch file set and
   * the pushed-commit file set are identical.  This should always pass
   * regardless of the companion fixes.
   */
  it("sanity – no excluded files: patch and pushed commit contain the same files", async () => {
    const { generateGitPatch } = require("./generate_git_patch.cjs");
    const { generateGitBundle } = require("./generate_git_bundle.cjs");
    const { applyBundleToBranch } = require("./create_pull_request.cjs");

    const branchName = "parity-sanity-no-exclusions";

    const bareRemote = createBareRepo("parity-sanity-bare-");
    const agentRepo = fs.mkdtempSync(path.join(os.tmpdir(), "parity-sanity-agent-"));
    tempDirs.push(bareRemote, agentRepo);
    cloneRepo(bareRemote, agentRepo);

    // Initial commit on main
    fs.writeFileSync(path.join(agentRepo, "README.md"), "base\n");
    execGit(["add", "README.md"], { cwd: agentRepo });
    execGit(["commit", "-m", "init"], { cwd: agentRepo });
    execGit(["push", "-u", "origin", "main"], { cwd: agentRepo });

    // Feature branch: add one file
    execGit(["checkout", "-b", branchName], { cwd: agentRepo });
    fs.writeFileSync(path.join(agentRepo, "main_file.txt"), "agent change\n");
    execGit(["add", "main_file.txt"], { cwd: agentRepo });
    execGit(["commit", "-m", "feat: add main_file"], { cwd: agentRepo });

    // Generate patch (no exclusions)
    const patchResult = await generateGitPatch(branchName, "main", { cwd: agentRepo });
    trackArtifactPath(patchResult, "patchPath", createdArtifacts);
    expect(patchResult.success, `patch generation failed: ${JSON.stringify(patchResult)}`).toBe(true);

    // Generate bundle
    const bundleResult = await generateGitBundle(branchName, "main", { cwd: agentRepo });
    trackArtifactPath(bundleResult, "bundlePath", createdArtifacts);
    expect(bundleResult.success, `bundle generation failed: ${bundleResult.error}`).toBe(true);

    // Apply bundle to a fresh safe-outputs checkout
    const safeOutputsRepo = fs.mkdtempSync(path.join(os.tmpdir(), "parity-sanity-so-"));
    tempDirs.push(safeOutputsRepo);
    cloneRepo(bareRemote, safeOutputsRepo);
    execGit(["checkout", "-b", branchName], { cwd: safeOutputsRepo });
    await applyBundleToBranch(bundleResult.bundlePath, branchName, "", createExecApi(safeOutputsRepo));

    // Compare file sets
    const patchContent = fs.readFileSync(patchResult.patchPath, "utf8");
    const fromPatch = fileListFromPatch(patchContent);
    const fromPush = fileListFromPushedCommit(safeOutputsRepo, "origin/main");

    expect(fromPatch, "patch should contain main_file.txt").toEqual(["main_file.txt"]);
    expect(fromPush, "pushed commit should match patch file set").toEqual(fromPatch);
  });

  /**
   * Regression test — excluded-files / non-rewrite path.
   *
   * The patch is generated with `excludedFiles` so it does NOT contain
   * `excluded_file.txt`. The bundle currently includes ALL file changes.
   * After applying the bundle the pushed commit MUST NOT contain
   * `excluded_file.txt`.
   *
   * This test FAILS on pre-fix code because the bundle always includes the
   * excluded file. It passes after the companion fix ensures that the
   * pushed commit matches the validated patch file set.
   */
  it("non-rewrite path: excluded files are absent from the pushed commit", async () => {
    const { generateGitPatch } = require("./generate_git_patch.cjs");
    const { generateGitBundle } = require("./generate_git_bundle.cjs");
    const { applyBundleToBranch } = require("./create_pull_request.cjs");

    const branchName = "parity-excl-no-rewrite";

    const bareRemote = createBareRepo("parity-excl-bare-");
    const agentRepo = fs.mkdtempSync(path.join(os.tmpdir(), "parity-excl-agent-"));
    tempDirs.push(bareRemote, agentRepo);
    cloneRepo(bareRemote, agentRepo);

    // Initial commit on main
    fs.writeFileSync(path.join(agentRepo, "README.md"), "base\n");
    execGit(["add", "README.md"], { cwd: agentRepo });
    execGit(["commit", "-m", "init"], { cwd: agentRepo });
    execGit(["push", "-u", "origin", "main"], { cwd: agentRepo });

    // Feature branch: modify both a regular file and an excluded file in one commit
    execGit(["checkout", "-b", branchName], { cwd: agentRepo });
    fs.writeFileSync(path.join(agentRepo, "main_file.txt"), "agent change\n");
    fs.writeFileSync(path.join(agentRepo, "excluded_file.txt"), "secret content\n");
    execGit(["add", "main_file.txt", "excluded_file.txt"], { cwd: agentRepo });
    execGit(["commit", "-m", "feat: add both files"], { cwd: agentRepo });

    // Generate patch: excluded_file.txt must be absent from the patch
    const patchResult = await generateGitPatch(branchName, "main", {
      cwd: agentRepo,
      excludedFiles: ["excluded_file.txt"],
    });
    trackArtifactPath(patchResult, "patchPath", createdArtifacts);
    expect(patchResult.success, `patch generation failed: ${JSON.stringify(patchResult)}`).toBe(true);

    // Verify the patch indeed excludes excluded_file.txt
    const patchContent = fs.readFileSync(patchResult.patchPath, "utf8");
    const fromPatch = fileListFromPatch(patchContent);
    expect(fromPatch, "patch should contain only main_file.txt").toEqual(["main_file.txt"]);

    // Generate bundle with the same exclusions used for patch validation.
    const bundleResult = await generateGitBundle(branchName, "main", {
      cwd: agentRepo,
      excludedFiles: ["excluded_file.txt"],
    });
    trackArtifactPath(bundleResult, "bundlePath", createdArtifacts);
    expect(bundleResult.success, `bundle generation failed: ${bundleResult.error}`).toBe(true);

    // Apply bundle to a fresh safe-outputs checkout
    const safeOutputsRepo = fs.mkdtempSync(path.join(os.tmpdir(), "parity-excl-so-"));
    tempDirs.push(safeOutputsRepo);
    cloneRepo(bareRemote, safeOutputsRepo);
    execGit(["checkout", "-b", branchName], { cwd: safeOutputsRepo });
    await applyBundleToBranch(bundleResult.bundlePath, branchName, "", createExecApi(safeOutputsRepo));

    // REGRESSION ASSERTION: the pushed commit must contain the same files as the patch.
    // On pre-fix code this fails because the bundle (and therefore the pushed commit)
    // contains excluded_file.txt even though the patch does not.
    const fromPush = fileListFromPushedCommit(safeOutputsRepo, "origin/main");
    expect(fromPush, "pushed commit should match patch file set (excluded_file.txt must not be pushed)").toEqual(fromPatch);
  });

  /**
   * Regression test — merge-commit rewrite path.
   *
   * After a bundle with merge commits is applied, `linearizeRangeAsCommit` is
   * called to collapse the branch history to a single commit (this is the path
   * taken by `rewriteBundleBranchAsSingleCommit` inside `create_pull_request`
   * when signed push refuses merge-commit topology).
   *
   * When base drift exists, the rewritten commit must still match the validated
   * patch file set rather than reverting or absorbing unrelated base changes.
   */
  it("merge-commit rewrite path: rewritten commit file set matches validated patch", async () => {
    const { generateGitPatch } = require("./generate_git_patch.cjs");
    const { generateGitBundle } = require("./generate_git_bundle.cjs");
    const { applyBundleToBranch, rewriteBundleBranchAsSingleCommit } = require("./create_pull_request.cjs");

    const branchName = "parity-excl-rewrite";

    const bareRemote = createBareRepo("parity-rewrite-bare-");
    const agentRepo = fs.mkdtempSync(path.join(os.tmpdir(), "parity-rewrite-agent-"));
    tempDirs.push(bareRemote, agentRepo);
    cloneRepo(bareRemote, agentRepo);

    // Initial commit on main
    fs.writeFileSync(path.join(agentRepo, "README.md"), "base\n");
    execGit(["add", "README.md"], { cwd: agentRepo });
    execGit(["commit", "-m", "init"], { cwd: agentRepo });
    execGit(["push", "-u", "origin", "main"], { cwd: agentRepo });

    // Feature branch: modify both a regular file and an excluded file.
    execGit(["checkout", "-b", branchName], { cwd: agentRepo });
    fs.writeFileSync(path.join(agentRepo, "main_file.txt"), "agent change\n");
    fs.writeFileSync(path.join(agentRepo, "excluded_file.txt"), "secret content\n");
    execGit(["add", "main_file.txt", "excluded_file.txt"], { cwd: agentRepo });
    execGit(["commit", "-m", "feat: add files"], { cwd: agentRepo });

    // Create merge topology on the feature branch without pulling base drift into it.
    const mergeSideBranch = `${branchName}-side`;
    execGit(["checkout", "-b", mergeSideBranch], { cwd: agentRepo });
    fs.writeFileSync(path.join(agentRepo, "main_file.txt"), "agent change from merged side branch\n");
    execGit(["add", "main_file.txt"], { cwd: agentRepo });
    execGit(["commit", "-m", "feat: side branch update"], { cwd: agentRepo });

    execGit(["checkout", branchName], { cwd: agentRepo });
    execGit(["merge", "--no-ff", mergeSideBranch, "-m", "reconcile: merge side branch"], { cwd: agentRepo });

    // Simulate base-branch drift from a separate clone after the merge topology exists.
    const collaboratorRepo = fs.mkdtempSync(path.join(os.tmpdir(), "parity-rewrite-collab-"));
    tempDirs.push(collaboratorRepo);
    cloneRepo(bareRemote, collaboratorRepo);
    fs.writeFileSync(path.join(collaboratorRepo, "drift.txt"), "collaborator change\n");
    execGit(["add", "drift.txt"], { cwd: collaboratorRepo });
    execGit(["commit", "-m", "chore: drift"], { cwd: collaboratorRepo });
    execGit(["push", "origin", "main"], { cwd: collaboratorRepo });

    // Verify the topology: the feature branch must contain at least one merge commit
    const mergeCount = Number(execGit(["rev-list", "--count", "--merges", `main..${branchName}`], { cwd: agentRepo }).stdout.trim());
    expect(mergeCount, "feature branch should contain at least one merge commit").toBeGreaterThanOrEqual(1);

    // Generate patch: excluded_file.txt must be absent
    // The patch covers commits between merge-base(origin/main, feature) and the branch tip.
    // Because format-patch ignores merge commits by default, the patch only reflects
    // the non-merge commits on the feature branch (i.e. the agent's own work).
    const patchResult = await generateGitPatch(branchName, "main", {
      cwd: agentRepo,
      excludedFiles: ["excluded_file.txt"],
    });
    trackArtifactPath(patchResult, "patchPath", createdArtifacts);
    expect(patchResult.success, `patch generation failed: ${JSON.stringify(patchResult)}`).toBe(true);

    // Verify the patch indeed excludes excluded_file.txt
    const patchContent = fs.readFileSync(patchResult.patchPath, "utf8");
    const fromPatch = fileListFromPatch(patchContent);
    expect(fromPatch, "patch should contain only main_file.txt").toEqual(["main_file.txt"]);

    // Generate bundle with the same exclusions used for patch validation.
    const bundleResult = await generateGitBundle(branchName, "main", {
      cwd: agentRepo,
      excludedFiles: ["excluded_file.txt"],
    });
    trackArtifactPath(bundleResult, "bundlePath", createdArtifacts);
    expect(bundleResult.success, `bundle generation failed: ${bundleResult.error}`).toBe(true);

    // Apply bundle to a fresh safe-outputs checkout.
    // The safe-outputs repo is cloned AFTER the drift commit was pushed, so
    // origin/main already includes the collaborator's change.
    const safeOutputsRepo = fs.mkdtempSync(path.join(os.tmpdir(), "parity-rewrite-so-"));
    tempDirs.push(safeOutputsRepo);
    cloneRepo(bareRemote, safeOutputsRepo);
    execGit(["checkout", "-b", branchName], { cwd: safeOutputsRepo });
    await applyBundleToBranch(bundleResult.bundlePath, branchName, "", createExecApi(safeOutputsRepo));

    // Simulate the production merge-commit rewrite path. This wrapper extracts the
    // bundle prerequisite as the linearization base, then forwards excludedFiles to
    // linearizeRangeAsCommit before creating the replacement single commit.
    await rewriteBundleBranchAsSingleCommit("main", createExecApi(safeOutputsRepo), bundleResult.bundlePath, {
      excludedFiles: ["excluded_file.txt"],
    });

    // REGRESSION ASSERTION: the rewritten commit must contain the same files as the patch.
    const fromPush = fileListFromPushedCommit(safeOutputsRepo, "origin/main");
    expect(fromPush, "rewritten commit should match patch file set after rewrite under base drift").toEqual(fromPatch);
  });
});
