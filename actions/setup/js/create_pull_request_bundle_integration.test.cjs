/**
 * Integration tests for create_pull_request bundle application.
 *
 * These tests run real git commands against temporary repositories to verify
 * bundle handling for checked-out target branches.
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

/**
 * create_pull_request.cjs reads the disclosure-header prompt template at module
 * load time. Ensure the file is present before the first `require` so the module
 * can be loaded successfully in any environment (local dev or CI).
 */
function ensureDisclosureHeaderPrompt() {
  const promptsDir = path.join(process.env.RUNNER_TEMP || os.tmpdir(), "gh-aw", "prompts");
  fs.mkdirSync(promptsDir, { recursive: true });
  fs.copyFileSync(path.join(promptsSourceDir, "safe_outputs_disclosure_header.md"), path.join(promptsDir, "safe_outputs_disclosure_header.md"));
}

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

  beforeAll(() => {
    ensureDisclosureHeaderPrompt();
  });

  afterEach(() => {
    for (const tempDir of tempDirs.splice(0)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
    vi.clearAllMocks();
  });

  it("applies a HEAD-only bundle (no refs/heads/* entry) using HEAD refspec fallback", async () => {
    const branchName = "docs/update-migration-version-2026-05-19";
    const sourceRepo = createRepo("create-pr-bundle-head-only-source-");
    const targetRepo = createRepo("create-pr-bundle-head-only-target-");
    tempDirs.push(sourceRepo, targetRepo);

    // Set up source with a shared base commit so target can accept the bundle
    fs.writeFileSync(path.join(sourceRepo, "file.txt"), "base\n");
    execGit(["add", "file.txt"], { cwd: sourceRepo });
    execGit(["commit", "-m", "base"], { cwd: sourceRepo });
    execGit(["branch", "-M", "main"], { cwd: sourceRepo });
    execGit(["checkout", "-b", branchName], { cwd: sourceRepo });
    fs.writeFileSync(path.join(sourceRepo, "file.txt"), "bundle tip\n");
    execGit(["commit", "-am", "bundle tip"], { cwd: sourceRepo });
    const expectedHead = execGit(["rev-parse", "HEAD"], { cwd: sourceRepo }).stdout.trim();
    const bundlePath = path.join(sourceRepo, "head-only.bundle");
    // Create a bundle with only HEAD — no named branch ref (reproduces the bug scenario)
    execGit(["bundle", "create", bundlePath, "HEAD"], { cwd: sourceRepo });

    // Verify that the bundle indeed contains only HEAD and no refs/heads/* entry
    const listHeadsOutput = execGit(["bundle", "list-heads", bundlePath], { cwd: sourceRepo }).stdout;
    expect(listHeadsOutput).toContain("HEAD");
    expect(listHeadsOutput).not.toMatch(/refs\/heads\//);

    // Target repo starts from the same base so bundle prerequisites are satisfied.
    // Fetch main from the source repo so the prerequisite commit is reachable.
    fs.writeFileSync(path.join(targetRepo, "file.txt"), "base\n");
    execGit(["add", "file.txt"], { cwd: targetRepo });
    execGit(["remote", "add", "origin", sourceRepo], { cwd: targetRepo });
    execGit(["fetch", "origin", "main"], { cwd: targetRepo });
    execGit(["checkout", "-b", branchName, "FETCH_HEAD"], { cwd: targetRepo });

    const { applyBundleToBranch } = require("./create_pull_request.cjs");
    // Pass a mismatched originalAgentBranch to trigger the fallback (as if the JSONL branch
    // name were different from any ref stored in the bundle)
    await applyBundleToBranch(bundlePath, branchName, "refs-that-dont-exist-in-bundle", createExecApi(targetRepo));

    const actualHead = execGit(["rev-parse", "HEAD"], { cwd: targetRepo }).stdout.trim();
    expect(actualHead).toBe(expectedHead);
    expect(fs.readFileSync(path.join(targetRepo, "file.txt"), "utf8")).toBe("bundle tip\n");
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

  it("captures a push rejection error after applying a reconcile-spark diverged-history bundle", async () => {
    // ─── Why this test exists ────────────────────────────────────────────────
    //
    // The "reconcile-spark" chaos scenario exposed a gap in the
    // create_pull_request bundle path:
    //
    //   1. An agent works on a feature branch and makes commits.
    //   2. The main branch receives new commits while the agent is working
    //      (history diverges).
    //   3. The agent reconciles by merging main into their branch, producing
    //      a non-linear (merge-commit) history — this is the "reconcile-spark"
    //      topology.
    //   4. A git bundle is created from that non-linear history.
    //   5. The bundle is applied to the safe-outputs runner via applyBundleToBranch.
    //   6. The subsequent push to origin fails because the remote branch has
    //      also diverged (or a policy hook rejects the push).
    //
    // Previously, pushSignedCommits attempted a linear cherry-pick replay of
    // the commit range onto the current GraphQL parent. That path choked on
    // merge commits and produced a CONFLICT error, dropping the flow into the
    // fallback-issue path with no useful context.
    //
    // The fix adds a sanitized `pushFailureMessage` to the fallback issue body
    // so that manual recovery is deterministic. This integration test verifies:
    //
    //   • applyBundleToBranch correctly imports a reconcile-spark merge topology
    //     (merge commits are preserved, not flattened).
    //   • A real git push to a diverged bare remote fails with an error — the
    //     kind of raw error string that create_pull_request captures and sanitizes
    //     before embedding in the fallback issue body.
    //   • The raw error produced by git contains content that sanitization must
    //     handle (the test injects an @-mention into the hook rejection message
    //     to document the attack surface; sanitizeContent strips it in prod).
    //
    // ─────────────────────────────────────────────────────────────────────────

    const branchName = "scratchpad/chaos/reconcile-spark";

    // 1. Set up a bare "origin" repo and a working clone — this mimics the
    //    relationship between GitHub and the safe-outputs runner checkout.
    const bareRemote = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-reconcile-spark-bare-"));
    const agentRepo = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-reconcile-spark-agent-"));
    const safeOutputsRepo = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-reconcile-spark-so-"));
    tempDirs.push(bareRemote, agentRepo, safeOutputsRepo);

    // Initialize the bare remote and push a first commit onto main.
    // Use -b main so we never need to run git symbolic-ref inside the bare repo
    // (git 2.36+ restricts bare-repo commands unless safe.bareRepository=all).
    execGit(["init", "--bare", "-b", "main"], { cwd: bareRemote });
    execGit(["clone", bareRemote, "."], { cwd: agentRepo });
    execGit(["config", "user.name", "Agent"], { cwd: agentRepo });
    execGit(["config", "user.email", "agent@example.com"], { cwd: agentRepo });

    fs.writeFileSync(path.join(agentRepo, "README.md"), "# Chaos scenario\n");
    execGit(["add", "README.md"], { cwd: agentRepo });
    execGit(["commit", "-m", "Initial commit on main"], { cwd: agentRepo });
    execGit(["branch", "-M", "main"], { cwd: agentRepo });
    execGit(["push", "-u", "origin", "main"], { cwd: agentRepo });
    core.info("[reconcile-spark] bare remote initialized with main");

    // 2. Agent creates the feature branch and makes a first content commit.
    execGit(["checkout", "-b", branchName], { cwd: agentRepo });
    fs.writeFileSync(path.join(agentRepo, "notes.md"), "# Agent notes\n");
    execGit(["add", "notes.md"], { cwd: agentRepo });
    execGit(["commit", "-m", "feat: add initial notes"], { cwd: agentRepo });
    core.info("[reconcile-spark] agent made first commit on feature branch");

    // 3. While the agent is working, a collaborator pushes a commit directly
    //    to main on the remote. The agent's local main diverges from origin/main.
    //    We simulate this by making a commit on main in the agent repo and then
    //    force-pushing it so the bare remote has a commit the agent branch doesn't.
    execGit(["checkout", "main"], { cwd: agentRepo });
    fs.writeFileSync(path.join(agentRepo, "collab.md"), "# Collaborator change\n");
    execGit(["add", "collab.md"], { cwd: agentRepo });
    execGit(["commit", "-m", "collab: landing from main"], { cwd: agentRepo });
    execGit(["push", "origin", "main"], { cwd: agentRepo });
    core.info("[reconcile-spark] collaborator commit pushed to origin/main — histories now diverged");

    // 4. Agent reconciles: merges the updated main back into the feature branch.
    //    This creates the "reconcile-spark" non-linear merge commit.
    execGit(["checkout", branchName], { cwd: agentRepo });
    execGit(["merge", "--no-ff", "main", "-m", "reconcile: merge main into feature"], { cwd: agentRepo });
    core.info("[reconcile-spark] merge commit created — non-linear history established");

    // Add one more commit after the reconcile merge to ensure the bundle tip is
    // beyond the merge (the pathological shape that the original linear-replay
    // path could not handle: a non-empty range starting with a merge parent).
    fs.writeFileSync(path.join(agentRepo, "notes.md"), "# Agent notes\n\nPost-reconcile edit\n");
    execGit(["commit", "-am", "feat: post-reconcile update"], { cwd: agentRepo });
    const expectedBundleTip = execGit(["rev-parse", "HEAD"], { cwd: agentRepo }).stdout.trim();
    const mergeCommitCount = Number(execGit(["rev-list", "--count", "--merges", `main..${branchName}`], { cwd: agentRepo }).stdout.trim());
    core.info(`[reconcile-spark] feature branch tip: ${expectedBundleTip.slice(0, 8)}, merge commits in range: ${mergeCommitCount}`);
    // Confirm the branch contains at least one merge commit — the test topology
    // is only valid when the reconcile merge is present.
    expect(mergeCommitCount).toBeGreaterThanOrEqual(1);

    // 5. Create a git bundle from the reconcile-spark branch.  The bundle
    //    includes the full history so that the safe-outputs runner can apply it
    //    without access to origin.
    const bundlePath = path.join(agentRepo, "reconcile-spark.bundle");
    execGit(["bundle", "create", bundlePath, `refs/heads/${branchName}`], { cwd: agentRepo });
    core.info(`[reconcile-spark] bundle created: ${bundlePath}`);

    // 6. Set up the safe-outputs runner checkout — a fresh clone of origin/main.
    //    This is the state the runner is in before it applies the agent's bundle.
    execGit(["clone", bareRemote, "."], { cwd: safeOutputsRepo });
    execGit(["config", "user.name", "Runner"], { cwd: safeOutputsRepo });
    execGit(["config", "user.email", "runner@example.com"], { cwd: safeOutputsRepo });
    execGit(["checkout", "-b", branchName], { cwd: safeOutputsRepo });
    core.info("[reconcile-spark] safe-outputs runner checkout ready");

    // 7. Apply the bundle via applyBundleToBranch — the function under test.
    const { applyBundleToBranch } = require("./create_pull_request.cjs");
    await applyBundleToBranch(bundlePath, branchName, `refs/heads/${branchName}`, createExecApi(safeOutputsRepo));

    // Verify that the merge-commit topology survived the bundle round-trip.
    const appliedTip = execGit(["rev-parse", "HEAD"], { cwd: safeOutputsRepo }).stdout.trim();
    const appliedMergeCount = Number(execGit(["rev-list", "--count", "--merges", "HEAD"], { cwd: safeOutputsRepo }).stdout.trim());
    core.info(`[reconcile-spark] bundle applied; tip: ${appliedTip.slice(0, 8)}, merges: ${appliedMergeCount}`);
    expect(appliedTip).toBe(expectedBundleTip);
    expect(appliedMergeCount).toBeGreaterThanOrEqual(1);
    expect(fs.readFileSync(path.join(safeOutputsRepo, "notes.md"), "utf8")).toContain("Post-reconcile edit");
    expect(fs.readFileSync(path.join(safeOutputsRepo, "collab.md"), "utf8")).toBe("# Collaborator change\n");

    // 8. Simulate a push rejection.
    //
    //    In the reconcile-spark scenario the push fails because:
    //    (a) The remote branch may not accept non-fast-forward pushes, or
    //    (b) A policy hook (e.g. "require signed commits") rejects the push.
    //
    //    We reproduce (b) by installing a pre-receive hook that emits a message
    //    containing an @-mention — deliberately chosen to document the attack
    //    surface that sanitizeContent must neutralise before the error is
    //    embedded in the fallback issue body.
    const hooksDir = path.join(bareRemote, "hooks");
    fs.mkdirSync(hooksDir, { recursive: true });
    const hookPath = path.join(hooksDir, "pre-receive");
    // The @org/team mention in the hook message is intentional: it demonstrates
    // the class of content (@ mentions, URLs, closing keywords) that can appear
    // in raw git push errors and must be stripped by sanitizeContent before the
    // message is interpolated into the fallback issue markdown body.
    fs.writeFileSync(
      hookPath,
      [
        "#!/bin/sh",
        "echo 'remote: error: pushSignedCommits: failed to rebase commit range onto current GraphQL parent (merge commit detected)' >&2",
        "echo 'remote: - CONFLICT (content): Merge conflict in scratchpad/chaos/reconcile-spark.md' >&2",
        "echo 'remote: - See @org/team for recovery steps.' >&2",
        "exit 1",
      ].join("\n") + "\n"
    );
    fs.chmodSync(hookPath, "0755");

    // Attempt the real git push — this MUST fail so we can capture the error.
    const pushResult = execGit(["push", "origin", branchName], { cwd: safeOutputsRepo, allowFailure: true });
    core.info(`[reconcile-spark] push exit code: ${pushResult.status}`);
    core.info(`[reconcile-spark] push stderr: ${pushResult.stderr.trim()}`);
    expect(pushResult.status).not.toBe(0);

    // 9. Verify the raw push error contains the content that create_pull_request
    //    must sanitize and embed in the fallback issue body.
    //    This is the value that will be passed through:
    //      sanitizeContent(neutralizeClosingKeywordsForIssueBody(pushError.message), ...)
    //    before being written into the fallback issue markdown.
    const rawPushError = pushResult.stderr.trim();
    expect(rawPushError).toContain("merge commit detected");
    expect(rawPushError).toContain("CONFLICT");
    // The @-mention in the hook output confirms that unsanitized error text can
    // contain @ tokens — sanitizeContent replaces them with safe equivalents.
    expect(rawPushError).toContain("@org/team");
    core.info("[reconcile-spark] push error captured — ready for sanitization and fallback issue embedding");
  });

  it("rewriteBundleBranchAsSingleCommit uses bundle prerequisite SHA to avoid including base-branch drift", async () => {
    // ─── Why this test exists ────────────────────────────────────────────────
    //
    // Production race: the agent creates its bundle while main is at commit A.
    // Between bundle creation and safe-outputs applying it, a collaborator
    // advances main to commit B (adding "base-drift.txt"). The safe-outputs
    // runner clones origin/main at B, applies the bundle, then calls
    // rewriteBundleBranchAsSingleCommit.
    //
    // With the naive fix (soft-reset to origin/main = B), the staged diff is
    // {agent-file.txt added, base-drift.txt deleted}, so the synthesized commit
    // incorrectly deletes base-drift.txt.
    //
    // The correct fix: read the prerequisite SHA from the bundle (= commit A,
    // the exact base the agent started from) and soft-reset to that SHA.
    // Staged diff becomes {agent-file.txt added} only.
    //
    // This test models the production race faithfully:
    //   1. Bare remote with initial commit A on main.
    //   2. Agent clones at A, creates feature branch, adds "agent-file.txt".
    //   3. Agent bundles while main is still at A  →  prereq = A.
    //   4. Collaborator advances main to B ("base-drift.txt").
    //   5. Safe-outputs clones updated origin (B on main), applies bundle.
    //   6. rewriteBundleBranchAsSingleCommit is called.
    //   7. Synthesized commit MUST contain only "agent-file.txt" (from agent).
    //   8. "base-drift.txt" MUST NOT appear anywhere in the synthesized diff
    //      (neither as added nor as deleted — it was never the agent's change).
    // ─────────────────────────────────────────────────────────────────────────

    const bareRemote = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-moving-base-bare-"));
    const agentRepo = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-moving-base-agent-"));
    const safeOutputsRepo = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-moving-base-so-"));
    tempDirs.push(bareRemote, agentRepo, safeOutputsRepo);

    // 1. Initialize bare remote and seed with initial commit A.
    execGit(["init", "--bare", "-b", "main"], { cwd: bareRemote });
    execGit(["clone", bareRemote, "."], { cwd: agentRepo });
    execGit(["config", "user.name", "Agent"], { cwd: agentRepo });
    execGit(["config", "user.email", "agent@example.com"], { cwd: agentRepo });
    fs.writeFileSync(path.join(agentRepo, "README.md"), "# Project\n");
    execGit(["add", "README.md"], { cwd: agentRepo });
    execGit(["commit", "-m", "Initial commit"], { cwd: agentRepo });
    execGit(["branch", "-M", "main"], { cwd: agentRepo });
    execGit(["push", "-u", "origin", "main"], { cwd: agentRepo });

    // Capture commit A — the exact base the agent starts from.
    const agentBaseCommit = execGit(["rev-parse", "HEAD"], { cwd: agentRepo }).stdout.trim();

    // 2. Agent creates feature branch and adds a file (main is still at A here).
    const branchName = "feature/moving-base-test";
    execGit(["checkout", "-b", branchName], { cwd: agentRepo });
    fs.writeFileSync(path.join(agentRepo, "agent-file.txt"), "agent change\n");
    execGit(["add", "agent-file.txt"], { cwd: agentRepo });
    execGit(["commit", "-m", "feat: agent adds a file"], { cwd: agentRepo });

    // 3. Agent bundles while main is STILL at A (before any drift).
    //    The bundle prerequisite is therefore A — the agent's actual base.
    const bundlePath = path.join(agentRepo, "feature.bundle");
    execGit(["bundle", "create", bundlePath, `${agentBaseCommit}..refs/heads/${branchName}`], { cwd: agentRepo });

    // Confirm the bundle's prerequisite is recorded as agentBaseCommit (A).
    const verifyOutput = execGit(["bundle", "verify", bundlePath], { cwd: agentRepo, allowFailure: true });
    const combinedVerify = `${verifyOutput.stdout || ""}\n${verifyOutput.stderr || ""}`;
    expect(combinedVerify).toMatch(new RegExp(agentBaseCommit.slice(0, 8), "i"));

    // 4. AFTER bundling: collaborator advances main to B (base-branch drift).
    execGit(["checkout", "main"], { cwd: agentRepo });
    fs.writeFileSync(path.join(agentRepo, "base-drift.txt"), "base drift added after agent checkout\n");
    execGit(["add", "base-drift.txt"], { cwd: agentRepo });
    execGit(["commit", "-m", "chore: add base-drift file"], { cwd: agentRepo });
    execGit(["push", "origin", "main"], { cwd: agentRepo });

    // 5. Safe-outputs runner clones the *updated* origin/main (B — includes base-drift.txt).
    execGit(["clone", bareRemote, "."], { cwd: safeOutputsRepo });
    execGit(["config", "user.name", "Runner"], { cwd: safeOutputsRepo });
    execGit(["config", "user.email", "runner@example.com"], { cwd: safeOutputsRepo });
    const newOriginMainSha = execGit(["rev-parse", "origin/main"], { cwd: safeOutputsRepo }).stdout.trim();
    expect(newOriginMainSha).not.toBe(agentBaseCommit); // Confirms base has moved.

    // Confirm base-drift.txt is on origin/main (the drift file is present on the remote).
    const driftOnOriginMain = execGit(["show", "origin/main:base-drift.txt"], { cwd: safeOutputsRepo, allowFailure: true });
    expect(driftOnOriginMain.status).toBe(0);

    // 6. Apply the bundle to the safe-outputs repo (bundle prereq = A, not B).
    const { applyBundleToBranch, rewriteBundleBranchAsSingleCommit } = require("./create_pull_request.cjs");
    await applyBundleToBranch(bundlePath, branchName, `refs/heads/${branchName}`, createExecApi(safeOutputsRepo), "main");

    // 7. Call rewriteBundleBranchAsSingleCommit with the bundle path so it reads prereq A.
    await rewriteBundleBranchAsSingleCommit("main", createExecApi(safeOutputsRepo), bundlePath);

    // 8. Verify the synthesized commit.
    //
    //   a) "agent-file.txt" must be in the working tree (agent's actual change was preserved).
    //   b) "base-drift.txt" must NOT appear in `git diff origin/main..HEAD` (the PR diff).
    //      After rebase-onto B the file is present in the working tree (inherited from origin/main),
    //      but must not appear as added or deleted relative to origin/main — the PR diff is clean.
    //   c) The diff HEAD^..HEAD must show "agent-file.txt" as added.
    //   d) "base-drift.txt" must NOT appear in HEAD^..HEAD — not deleted, not added.
    //      (With a naive soft-reset to origin/main=B, it would appear as "D base-drift.txt".)
    //   e) HEAD must have exactly one parent (linear, not a merge commit).
    const agentFilePresent = fs.existsSync(path.join(safeOutputsRepo, "agent-file.txt"));

    expect(agentFilePresent).toBe(true);
    // base-drift.txt must not appear in the PR diff (origin/main..HEAD). After rebase-onto B
    // the file is present in the working tree (inherited from origin/main) but must not be
    // listed as added or deleted relative to origin/main.
    const prDiffStat = execGit(["diff", "--name-status", "origin/main", "HEAD"], { cwd: safeOutputsRepo }).stdout;
    expect(prDiffStat).not.toMatch(/base-drift\.txt/);

    // The HEAD commit (linearized) should show "agent-file.txt" as added.
    const headStat = execGit(["show", "--stat", "HEAD"], { cwd: safeOutputsRepo }).stdout;
    expect(headStat).toContain("agent-file.txt");

    // The diff introduced by HEAD must NOT reference base-drift.txt at all.
    // If the prereq-SHA path were bypassed and origin/main (B) were used instead,
    // the diff would contain "D base-drift.txt" and this assertion would fail.
    const diffStat = execGit(["diff", "--name-status", "HEAD^", "HEAD"], { cwd: safeOutputsRepo }).stdout;
    expect(diffStat).not.toMatch(/base-drift\.txt/);

    // Verify the commit has exactly one parent (linearized, not a merge commit).
    const parentLine = execGit(["log", "-1", "--format=%P", "HEAD"], { cwd: safeOutputsRepo }).stdout.trim();
    const parentShas = parentLine.split(/\s+/).filter(Boolean);
    expect(parentShas).toHaveLength(1);
  });

  it("rewriteBundleBranchAsSingleCommit squashes multiple linear commits into one", async () => {
    // ─── Why this test exists ────────────────────────────────────────────────
    //
    // The signed-push path requires a single-commit head. An agent may produce
    // several linear commits (no merge topology). Verify that
    // rewriteBundleBranchAsSingleCommit collapses all of them into one commit
    // that contains every file change from the full range, and that the result
    // has exactly one parent rooted on origin/main.
    // ─────────────────────────────────────────────────────────────────────────

    const branchName = "feature/multi-commit-squash";

    const bareRemote = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-squash-bare-"));
    const agentRepo = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-squash-agent-"));
    const safeOutputsRepo = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-squash-so-"));
    tempDirs.push(bareRemote, agentRepo, safeOutputsRepo);

    // 1. Initialize bare remote and seed main.
    execGit(["init", "--bare", "-b", "main"], { cwd: bareRemote });
    execGit(["clone", bareRemote, "."], { cwd: agentRepo });
    execGit(["config", "user.name", "Agent"], { cwd: agentRepo });
    execGit(["config", "user.email", "agent@example.com"], { cwd: agentRepo });
    fs.writeFileSync(path.join(agentRepo, "README.md"), "# Project\n");
    execGit(["add", "README.md"], { cwd: agentRepo });
    execGit(["commit", "-m", "Initial commit"], { cwd: agentRepo });
    execGit(["branch", "-M", "main"], { cwd: agentRepo });
    execGit(["push", "-u", "origin", "main"], { cwd: agentRepo });

    const agentBaseCommit = execGit(["rev-parse", "HEAD"], { cwd: agentRepo }).stdout.trim();

    // 2. Agent creates the feature branch with three separate commits.
    execGit(["checkout", "-b", branchName], { cwd: agentRepo });
    fs.writeFileSync(path.join(agentRepo, "file-a.txt"), "content a\n");
    execGit(["add", "file-a.txt"], { cwd: agentRepo });
    execGit(["commit", "-m", "feat: add file-a"], { cwd: agentRepo });

    fs.writeFileSync(path.join(agentRepo, "file-b.txt"), "content b\n");
    execGit(["add", "file-b.txt"], { cwd: agentRepo });
    execGit(["commit", "-m", "feat: add file-b"], { cwd: agentRepo });

    fs.writeFileSync(path.join(agentRepo, "file-c.txt"), "content c\n");
    execGit(["add", "file-c.txt"], { cwd: agentRepo });
    execGit(["commit", "-m", "feat: add file-c"], { cwd: agentRepo });

    // Confirm three commits beyond main before bundling.
    const commitCount = Number(execGit(["rev-list", "--count", `${agentBaseCommit}..HEAD`], { cwd: agentRepo }).stdout.trim());
    expect(commitCount).toBe(3);

    // 3. Bundle the feature branch (prereq = agentBaseCommit).
    const bundlePath = path.join(agentRepo, "multi-commit.bundle");
    execGit(["bundle", "create", bundlePath, `${agentBaseCommit}..refs/heads/${branchName}`], { cwd: agentRepo });

    // 4. Safe-outputs runner: fresh clone of origin/main, apply bundle.
    execGit(["clone", bareRemote, "."], { cwd: safeOutputsRepo });
    execGit(["config", "user.name", "Runner"], { cwd: safeOutputsRepo });
    execGit(["config", "user.email", "runner@example.com"], { cwd: safeOutputsRepo });

    const { applyBundleToBranch, rewriteBundleBranchAsSingleCommit } = require("./create_pull_request.cjs");
    await applyBundleToBranch(bundlePath, branchName, `refs/heads/${branchName}`, createExecApi(safeOutputsRepo), "main");

    // 5. Rewrite the three commits to one.
    await rewriteBundleBranchAsSingleCommit("main", createExecApi(safeOutputsRepo), bundlePath);

    // 6. HEAD must have exactly one parent.
    const parentLine = execGit(["log", "-1", "--format=%P", "HEAD"], { cwd: safeOutputsRepo }).stdout.trim();
    const parentShas = parentLine.split(/\s+/).filter(Boolean);
    expect(parentShas).toHaveLength(1);

    // 7. All three files must be present in the working tree.
    expect(fs.existsSync(path.join(safeOutputsRepo, "file-a.txt"))).toBe(true);
    expect(fs.existsSync(path.join(safeOutputsRepo, "file-b.txt"))).toBe(true);
    expect(fs.existsSync(path.join(safeOutputsRepo, "file-c.txt"))).toBe(true);

    // 8. The diff origin/main..HEAD must contain exactly those three files.
    const diffNames = execGit(["diff", "--name-only", "origin/main..HEAD"], { cwd: safeOutputsRepo }).stdout.trim().split("\n").filter(Boolean).sort();
    expect(diffNames).toEqual(["file-a.txt", "file-b.txt", "file-c.txt"]);

    // 9. Exactly one commit beyond origin/main.
    const linearCommitCount = Number(execGit(["rev-list", "--count", "origin/main..HEAD"], { cwd: safeOutputsRepo }).stdout.trim());
    expect(linearCommitCount).toBe(1);
  });

  it("rewriteBundleBranchAsSingleCommit excludes specified files from the linearized commit", async () => {
    // ─── Why this test exists ────────────────────────────────────────────────
    //
    // When the caller specifies excludedFiles (e.g. secrets or build artifacts),
    // the rewritten single commit must not contain those files. This verifies
    // that excludedFiles are forwarded through rewriteBundleBranchAsSingleCommit
    // → linearizeRangeAsCommit and are unstaged before the squash commit is made.
    // ─────────────────────────────────────────────────────────────────────────

    const branchName = "feature/excluded-files-rewrite";

    const bareRemote = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-excl-rewrite-bare-"));
    const agentRepo = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-excl-rewrite-agent-"));
    const safeOutputsRepo = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-excl-rewrite-so-"));
    tempDirs.push(bareRemote, agentRepo, safeOutputsRepo);

    // 1. Initialize bare remote and seed main.
    execGit(["init", "--bare", "-b", "main"], { cwd: bareRemote });
    execGit(["clone", bareRemote, "."], { cwd: agentRepo });
    execGit(["config", "user.name", "Agent"], { cwd: agentRepo });
    execGit(["config", "user.email", "agent@example.com"], { cwd: agentRepo });
    fs.writeFileSync(path.join(agentRepo, "README.md"), "# Project\n");
    execGit(["add", "README.md"], { cwd: agentRepo });
    execGit(["commit", "-m", "Initial commit"], { cwd: agentRepo });
    execGit(["branch", "-M", "main"], { cwd: agentRepo });
    execGit(["push", "-u", "origin", "main"], { cwd: agentRepo });

    const agentBaseCommit = execGit(["rev-parse", "HEAD"], { cwd: agentRepo }).stdout.trim();

    // 2. Agent adds both a kept file and a file that should be excluded.
    execGit(["checkout", "-b", branchName], { cwd: agentRepo });
    fs.writeFileSync(path.join(agentRepo, "kept.txt"), "agent change\n");
    fs.writeFileSync(path.join(agentRepo, "secret.txt"), "sensitive data\n");
    execGit(["add", "kept.txt", "secret.txt"], { cwd: agentRepo });
    execGit(["commit", "-m", "feat: add kept and secret files"], { cwd: agentRepo });

    // 3. Bundle the feature branch.
    const bundlePath = path.join(agentRepo, "excluded-files.bundle");
    execGit(["bundle", "create", bundlePath, `${agentBaseCommit}..refs/heads/${branchName}`], { cwd: agentRepo });

    // 4. Safe-outputs runner: fresh clone, apply bundle.
    execGit(["clone", bareRemote, "."], { cwd: safeOutputsRepo });
    execGit(["config", "user.name", "Runner"], { cwd: safeOutputsRepo });
    execGit(["config", "user.email", "runner@example.com"], { cwd: safeOutputsRepo });

    const { applyBundleToBranch, rewriteBundleBranchAsSingleCommit } = require("./create_pull_request.cjs");
    await applyBundleToBranch(bundlePath, branchName, `refs/heads/${branchName}`, createExecApi(safeOutputsRepo), "main");

    // 5. Rewrite with secret.txt excluded.
    await rewriteBundleBranchAsSingleCommit("main", createExecApi(safeOutputsRepo), bundlePath, {
      excludedFiles: ["secret.txt"],
    });

    // 6. The diff origin/main..HEAD must contain only kept.txt.
    const diffNames = execGit(["diff", "--name-only", "origin/main..HEAD"], { cwd: safeOutputsRepo }).stdout.trim().split("\n").filter(Boolean).sort();
    expect(diffNames).toEqual(["kept.txt"]);

    // 7. secret.txt must not appear in the commit diff at all (not added, not deleted).
    const commitDiff = execGit(["diff", "--name-status", "HEAD^", "HEAD"], { cwd: safeOutputsRepo }).stdout;
    expect(commitDiff).not.toMatch(/secret\.txt/);

    // 8. Exactly one linearized commit beyond origin/main.
    const linearCommitCount = Number(execGit(["rev-list", "--count", "origin/main..HEAD"], { cwd: safeOutputsRepo }).stdout.trim());
    expect(linearCommitCount).toBe(1);
  });

  it("rewriteBundleBranchAsSingleCommit falls back to origin/main when bundle declares no prerequisites (self-contained bundle)", async () => {
    // ─── Why this test exists ────────────────────────────────────────────────
    //
    // Some agents bundle with `git bundle create <file> refs/heads/<branch>`
    // without a `<base>..` range exclusion. This produces a self-contained
    // bundle that includes all reachable history — meaning the bundle has NO
    // recorded prerequisites (getBundlePrerequisites returns []).
    //
    // rewriteBundleBranchAsSingleCommit must take the "prereqs.length === 0"
    // fallback branch and use origin/<base> as the linearization base, still
    // producing a single correct commit.
    // ─────────────────────────────────────────────────────────────────────────

    const branchName = "feature/self-contained-bundle";

    const bareRemote = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-no-prereq-bare-"));
    const agentRepo = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-no-prereq-agent-"));
    const safeOutputsRepo = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-no-prereq-so-"));
    tempDirs.push(bareRemote, agentRepo, safeOutputsRepo);

    // 1. Initialize bare remote and seed main.
    execGit(["init", "--bare", "-b", "main"], { cwd: bareRemote });
    execGit(["clone", bareRemote, "."], { cwd: agentRepo });
    execGit(["config", "user.name", "Agent"], { cwd: agentRepo });
    execGit(["config", "user.email", "agent@example.com"], { cwd: agentRepo });
    fs.writeFileSync(path.join(agentRepo, "README.md"), "# Project\n");
    execGit(["add", "README.md"], { cwd: agentRepo });
    execGit(["commit", "-m", "Initial commit"], { cwd: agentRepo });
    execGit(["branch", "-M", "main"], { cwd: agentRepo });
    execGit(["push", "-u", "origin", "main"], { cwd: agentRepo });

    // 2. Agent creates feature branch and adds a file.
    execGit(["checkout", "-b", branchName], { cwd: agentRepo });
    fs.writeFileSync(path.join(agentRepo, "agent-file.txt"), "agent change\n");
    execGit(["add", "agent-file.txt"], { cwd: agentRepo });
    execGit(["commit", "-m", "feat: agent adds a file"], { cwd: agentRepo });

    // 3. Create a SELF-CONTAINED bundle (no range exclusion → no prerequisites).
    //    The bundle includes the full reachable history and records no prereqs.
    const bundlePath = path.join(agentRepo, "self-contained.bundle");
    execGit(["bundle", "create", bundlePath, `refs/heads/${branchName}`], { cwd: agentRepo });

    // Confirm the bundle has no prerequisites (git reports "complete history").
    const verifyOut = execGit(["bundle", "verify", bundlePath], { cwd: agentRepo, allowFailure: true });
    const verifyText = `${verifyOut.stdout || ""}\n${verifyOut.stderr || ""}`;
    expect(verifyText).toMatch(/complete history/i);

    // 4. Safe-outputs runner: fresh clone of origin/main.
    execGit(["clone", bareRemote, "."], { cwd: safeOutputsRepo });
    execGit(["config", "user.name", "Runner"], { cwd: safeOutputsRepo });
    execGit(["config", "user.email", "runner@example.com"], { cwd: safeOutputsRepo });

    const { applyBundleToBranch, rewriteBundleBranchAsSingleCommit } = require("./create_pull_request.cjs");
    await applyBundleToBranch(bundlePath, branchName, `refs/heads/${branchName}`, createExecApi(safeOutputsRepo), "main");

    // 5. Rewrite using the self-contained bundle; function should log
    //    "Bundle declares no prerequisites; falling back to origin/main".
    await rewriteBundleBranchAsSingleCommit("main", createExecApi(safeOutputsRepo), bundlePath);

    // 6. Exactly one commit beyond origin/main.
    const linearCommitCount = Number(execGit(["rev-list", "--count", "origin/main..HEAD"], { cwd: safeOutputsRepo }).stdout.trim());
    expect(linearCommitCount).toBe(1);

    // 7. The diff origin/main..HEAD must contain only agent-file.txt.
    const diffNames = execGit(["diff", "--name-only", "origin/main..HEAD"], { cwd: safeOutputsRepo }).stdout.trim().split("\n").filter(Boolean).sort();
    expect(diffNames).toEqual(["agent-file.txt"]);

    // 8. Exactly one parent (linearized, not a merge commit).
    const parentLine = execGit(["log", "-1", "--format=%P", "HEAD"], { cwd: safeOutputsRepo }).stdout.trim();
    const parentShas = parentLine.split(/\s+/).filter(Boolean);
    expect(parentShas).toHaveLength(1);
  });

  it("rewriteBundleBranchAsSingleCommit falls back to origin/main when bundle has multiple prerequisites", async () => {
    // ─── Why this test exists ────────────────────────────────────────────────
    //
    // When the bundle range includes a commit whose parent is NOT reachable
    // from the range exclusion commit, the bundle records TWO prerequisites:
    //
    //   ROOT ─── MAIN (main branch, pushed to origin as agent base)
    //    └────── INDEPENDENT (side branch, never on main)
    //
    //   Feature: MAIN → FEAT(agent-work.txt) → MERGE(FEAT, INDEPENDENT)
    //   Bundle:  MAIN..refs/heads/feature
    //     • Included:      FEAT, INDEPENDENT, MERGE
    //     • Prerequisites: MAIN (parent of FEAT) + ROOT (parent of INDEPENDENT)
    //
    // rewriteBundleBranchAsSingleCommit falls back to origin/main when
    // prereqs.length > 1 and must still produce a single correct commit.
    // ─────────────────────────────────────────────────────────────────────────

    const branchName = "feature/multi-prereq-rewrite";

    const bareRemote = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-multi-prereq-bare-"));
    const agentRepo = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-multi-prereq-agent-"));
    const safeOutputsRepo = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-multi-prereq-so-"));
    tempDirs.push(bareRemote, agentRepo, safeOutputsRepo);

    // 1. Initialize bare remote.  ROOT is the initial commit; MAIN is the
    //    second commit on main.  The agent branches from MAIN.
    execGit(["init", "--bare", "-b", "main"], { cwd: bareRemote });
    execGit(["clone", bareRemote, "."], { cwd: agentRepo });
    execGit(["config", "user.name", "Agent"], { cwd: agentRepo });
    execGit(["config", "user.email", "agent@example.com"], { cwd: agentRepo });

    fs.writeFileSync(path.join(agentRepo, "README.md"), "# Project\n");
    execGit(["add", "README.md"], { cwd: agentRepo });
    execGit(["commit", "-m", "Initial commit"], { cwd: agentRepo });
    execGit(["branch", "-M", "main"], { cwd: agentRepo });
    execGit(["push", "-u", "origin", "main"], { cwd: agentRepo });
    const rootCommit = execGit(["rev-parse", "HEAD"], { cwd: agentRepo }).stdout.trim();

    fs.writeFileSync(path.join(agentRepo, "setup.txt"), "project setup\n");
    execGit(["add", "setup.txt"], { cwd: agentRepo });
    execGit(["commit", "-m", "chore: project setup"], { cwd: agentRepo });
    execGit(["push", "origin", "main"], { cwd: agentRepo });
    const agentBaseCommit = execGit(["rev-parse", "HEAD"], { cwd: agentRepo }).stdout.trim();

    // 2. Create an INDEPENDENT branch off ROOT (not on main's history).
    //    This branch is never pushed to origin.
    execGit(["checkout", "-b", "side-branch", rootCommit], { cwd: agentRepo });
    fs.writeFileSync(path.join(agentRepo, "side.txt"), "side branch content\n");
    execGit(["add", "side.txt"], { cwd: agentRepo });
    execGit(["commit", "-m", "feat: add side.txt from independent branch"], { cwd: agentRepo });

    // 3. Agent creates feature branch from MAIN and makes a commit.
    execGit(["checkout", "-b", branchName, agentBaseCommit], { cwd: agentRepo });
    fs.writeFileSync(path.join(agentRepo, "agent-work.txt"), "agent work\n");
    execGit(["add", "agent-work.txt"], { cwd: agentRepo });
    execGit(["commit", "-m", "feat: agent work"], { cwd: agentRepo });

    // 4. Agent merges the independent side branch into the feature branch.
    execGit(["merge", "--no-ff", "side-branch", "-m", "feat: merge side branch"], { cwd: agentRepo });

    // 5. Create the bundle with range MAIN..feature.
    //    FEAT and INDEPENDENT are included (reachable from feature but not from MAIN).
    //    Prerequisites: MAIN (parent of FEAT, not included) +
    //                   ROOT (parent of INDEPENDENT, not included) = TWO prereqs.
    const bundlePath = path.join(agentRepo, "multi-prereq.bundle");
    execGit(["bundle", "create", bundlePath, `${agentBaseCommit}..refs/heads/${branchName}`], { cwd: agentRepo });

    // Confirm the bundle records both prereq SHAs in git bundle verify output.
    const verifyOut = execGit(["bundle", "verify", bundlePath], { cwd: agentRepo, allowFailure: true });
    const verifyText = `${verifyOut.stdout || ""}\n${verifyOut.stderr || ""}`;
    expect(verifyText).toMatch(new RegExp(rootCommit.slice(0, 8), "i"));
    expect(verifyText).toMatch(new RegExp(agentBaseCommit.slice(0, 8), "i"));

    // 6. Safe-outputs runner: fresh clone of origin/main (has ROOT and MAIN,
    //    satisfying both bundle prerequisites).
    execGit(["clone", bareRemote, "."], { cwd: safeOutputsRepo });
    execGit(["config", "user.name", "Runner"], { cwd: safeOutputsRepo });
    execGit(["config", "user.email", "runner@example.com"], { cwd: safeOutputsRepo });

    const { applyBundleToBranch, rewriteBundleBranchAsSingleCommit } = require("./create_pull_request.cjs");
    await applyBundleToBranch(bundlePath, branchName, `refs/heads/${branchName}`, createExecApi(safeOutputsRepo), "main");

    // 7. Rewrite: 2 prereqs detected → falls back to origin/main (= MAIN).
    //    All changes relative to origin/main are captured in one squash commit.
    await rewriteBundleBranchAsSingleCommit("main", createExecApi(safeOutputsRepo), bundlePath);

    // 8. Exactly one commit beyond origin/main.
    const linearCommitCount = Number(execGit(["rev-list", "--count", "origin/main..HEAD"], { cwd: safeOutputsRepo }).stdout.trim());
    expect(linearCommitCount).toBe(1);

    // 9. The diff origin/main..HEAD covers both the agent's file and the merged-in
    //    side file (all changes relative to origin/main = MAIN).
    const diffNames = execGit(["diff", "--name-only", "origin/main..HEAD"], { cwd: safeOutputsRepo }).stdout.trim().split("\n").filter(Boolean).sort();
    expect(diffNames).toContain("agent-work.txt");
    expect(diffNames).toContain("side.txt");

    // 10. Exactly one parent (linearized, not a merge commit).
    const parentLine = execGit(["log", "-1", "--format=%P", "HEAD"], { cwd: safeOutputsRepo }).stdout.trim();
    const parentShas = parentLine.split(/\s+/).filter(Boolean);
    expect(parentShas).toHaveLength(1);
  });

  it("rewriteBundleBranchAsSingleCommit falls back to origin/main when bundle prerequisite SHA is not in local object store", async () => {
    // ─── Why this test exists ────────────────────────────────────────────────
    //
    // rewriteBundleBranchAsSingleCommit verifies reachability of the bundle's
    // prerequisite SHA via `git cat-file -e <sha>^{commit}`.  If the SHA is
    // not in the local object store (e.g., in a shallow clone, or when the
    // prereq comes from a repo whose history was never fetched), the function
    // falls back to origin/main.
    //
    // To trigger this path deterministically, we pass a bundle file whose
    // prerequisite commit is from an unrelated repository — a SHA that will
    // never exist in the safe-outputs checkout — while the feature branch
    // itself was correctly applied via a separate (full) bundle.
    // ─────────────────────────────────────────────────────────────────────────

    const branchName = "feature/prereq-inaccessible";

    const bareRemote = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-inaccessible-bare-"));
    const agentRepo = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-inaccessible-agent-"));
    const safeOutputsRepo = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-inaccessible-so-"));
    // Unrelated repo whose history never enters safeOutputsRepo.
    const unrelatedRepo = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-inaccessible-unrelated-"));
    tempDirs.push(bareRemote, agentRepo, safeOutputsRepo, unrelatedRepo);

    // 1. Normal project setup.
    execGit(["init", "--bare", "-b", "main"], { cwd: bareRemote });
    execGit(["clone", bareRemote, "."], { cwd: agentRepo });
    execGit(["config", "user.name", "Agent"], { cwd: agentRepo });
    execGit(["config", "user.email", "agent@example.com"], { cwd: agentRepo });
    fs.writeFileSync(path.join(agentRepo, "README.md"), "# Project\n");
    execGit(["add", "README.md"], { cwd: agentRepo });
    execGit(["commit", "-m", "Initial commit"], { cwd: agentRepo });
    execGit(["branch", "-M", "main"], { cwd: agentRepo });
    execGit(["push", "-u", "origin", "main"], { cwd: agentRepo });
    const agentBaseCommit = execGit(["rev-parse", "HEAD"], { cwd: agentRepo }).stdout.trim();

    // 2. Agent creates feature branch and adds a file.
    execGit(["checkout", "-b", branchName], { cwd: agentRepo });
    fs.writeFileSync(path.join(agentRepo, "agent-file.txt"), "agent change\n");
    execGit(["add", "agent-file.txt"], { cwd: agentRepo });
    execGit(["commit", "-m", "feat: agent adds a file"], { cwd: agentRepo });

    // 3. Create the real bundle (with correct prereq) for applyBundleToBranch.
    const realBundlePath = path.join(agentRepo, "real.bundle");
    execGit(["bundle", "create", realBundlePath, `${agentBaseCommit}..refs/heads/${branchName}`], { cwd: agentRepo });

    // 4. Create an UNRELATED repo and bundle from it.
    //    This bundle's prerequisite SHA is from unrelatedRepo and will NEVER
    //    exist in safeOutputsRepo.
    execGit(["init"], { cwd: unrelatedRepo });
    execGit(["config", "user.name", "Unrelated"], { cwd: unrelatedRepo });
    execGit(["config", "user.email", "unrelated@example.com"], { cwd: unrelatedRepo });
    fs.writeFileSync(path.join(unrelatedRepo, "base.txt"), "unrelated base\n");
    execGit(["add", "base.txt"], { cwd: unrelatedRepo });
    execGit(["commit", "-m", "unrelated base"], { cwd: unrelatedRepo });
    const unrelatedBase = execGit(["rev-parse", "HEAD"], { cwd: unrelatedRepo }).stdout.trim();
    execGit(["checkout", "-b", "feature-side"], { cwd: unrelatedRepo });
    fs.writeFileSync(path.join(unrelatedRepo, "side.txt"), "side\n");
    execGit(["add", "side.txt"], { cwd: unrelatedRepo });
    execGit(["commit", "-m", "side commit"], { cwd: unrelatedRepo });
    // Bundle with prereq = unrelatedBase (a SHA that will NOT be in safeOutputsRepo).
    const fakeBundlePath = path.join(unrelatedRepo, "fake.bundle");
    execGit(["bundle", "create", fakeBundlePath, `${unrelatedBase}..refs/heads/feature-side`], { cwd: unrelatedRepo });

    // Confirm the fake bundle's prereq is unrelatedBase.
    const fakeVerify = execGit(["bundle", "verify", fakeBundlePath], { cwd: unrelatedRepo, allowFailure: true });
    const fakeVerifyText = `${fakeVerify.stdout || ""}\n${fakeVerify.stderr || ""}`;
    expect(fakeVerifyText).toMatch(new RegExp(unrelatedBase.slice(0, 8), "i"));

    // 5. Safe-outputs runner: fresh clone, apply the REAL bundle.
    execGit(["clone", bareRemote, "."], { cwd: safeOutputsRepo });
    execGit(["config", "user.name", "Runner"], { cwd: safeOutputsRepo });
    execGit(["config", "user.email", "runner@example.com"], { cwd: safeOutputsRepo });

    const { applyBundleToBranch, rewriteBundleBranchAsSingleCommit } = require("./create_pull_request.cjs");
    await applyBundleToBranch(realBundlePath, branchName, `refs/heads/${branchName}`, createExecApi(safeOutputsRepo), "main");

    // 6. Confirm the unrelated prereq is not in safeOutputsRepo.
    const catFileResult = execGit(["cat-file", "-e", `${unrelatedBase}^{commit}`], { cwd: safeOutputsRepo, allowFailure: true });
    expect(catFileResult.status).not.toBe(0);

    // 7. Rewrite using the FAKE bundle path (whose prereq is not accessible).
    //    The function must detect `cat-file -e` failure and fall back to origin/main.
    await rewriteBundleBranchAsSingleCommit("main", createExecApi(safeOutputsRepo), fakeBundlePath);

    // 8. The diff origin/main..HEAD must contain only agent-file.txt (correct result
    //    from the fallback-to-origin/main path).
    const diffNames = execGit(["diff", "--name-only", "origin/main..HEAD"], { cwd: safeOutputsRepo }).stdout.trim().split("\n").filter(Boolean).sort();
    expect(diffNames).toEqual(["agent-file.txt"]);

    // 9. Exactly one linearized commit beyond origin/main.
    const linearCommitCount = Number(execGit(["rev-list", "--count", "origin/main..HEAD"], { cwd: safeOutputsRepo }).stdout.trim());
    expect(linearCommitCount).toBe(1);
  });

  it("rewriteBundleBranchAsSingleCommit correctly preserves a file deletion", async () => {
    // ─── Why this test exists ────────────────────────────────────────────────
    //
    // Existing tests cover file additions. This test verifies that when the
    // agent DELETES a file that was present on the base branch,
    // rewriteBundleBranchAsSingleCommit produces a linearized commit that also
    // deletes the file — the deletion is not silently dropped.
    // ─────────────────────────────────────────────────────────────────────────

    const branchName = "feature/delete-file-rewrite";

    const bareRemote = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-del-bare-"));
    const agentRepo = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-del-agent-"));
    const safeOutputsRepo = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-del-so-"));
    tempDirs.push(bareRemote, agentRepo, safeOutputsRepo);

    // 1. Initialize bare remote with a base commit that includes a file to delete.
    execGit(["init", "--bare", "-b", "main"], { cwd: bareRemote });
    execGit(["clone", bareRemote, "."], { cwd: agentRepo });
    execGit(["config", "user.name", "Agent"], { cwd: agentRepo });
    execGit(["config", "user.email", "agent@example.com"], { cwd: agentRepo });
    fs.writeFileSync(path.join(agentRepo, "README.md"), "# Project\n");
    fs.writeFileSync(path.join(agentRepo, "to-delete.txt"), "this file will be removed\n");
    execGit(["add", "README.md", "to-delete.txt"], { cwd: agentRepo });
    execGit(["commit", "-m", "Initial commit"], { cwd: agentRepo });
    execGit(["branch", "-M", "main"], { cwd: agentRepo });
    execGit(["push", "-u", "origin", "main"], { cwd: agentRepo });
    const agentBaseCommit = execGit(["rev-parse", "HEAD"], { cwd: agentRepo }).stdout.trim();

    // 2. Agent deletes to-delete.txt and adds a new file in the same commit.
    execGit(["checkout", "-b", branchName], { cwd: agentRepo });
    execGit(["rm", "to-delete.txt"], { cwd: agentRepo });
    fs.writeFileSync(path.join(agentRepo, "replacement.txt"), "replacement content\n");
    execGit(["add", "replacement.txt"], { cwd: agentRepo });
    execGit(["commit", "-m", "feat: remove to-delete.txt, add replacement.txt"], { cwd: agentRepo });

    // 3. Bundle the feature branch.
    const bundlePath = path.join(agentRepo, "delete-file.bundle");
    execGit(["bundle", "create", bundlePath, `${agentBaseCommit}..refs/heads/${branchName}`], { cwd: agentRepo });

    // 4. Safe-outputs runner: fresh clone, apply bundle.
    execGit(["clone", bareRemote, "."], { cwd: safeOutputsRepo });
    execGit(["config", "user.name", "Runner"], { cwd: safeOutputsRepo });
    execGit(["config", "user.email", "runner@example.com"], { cwd: safeOutputsRepo });

    const { applyBundleToBranch, rewriteBundleBranchAsSingleCommit } = require("./create_pull_request.cjs");
    await applyBundleToBranch(bundlePath, branchName, `refs/heads/${branchName}`, createExecApi(safeOutputsRepo), "main");

    // 5. Rewrite.
    await rewriteBundleBranchAsSingleCommit("main", createExecApi(safeOutputsRepo), bundlePath);

    // 6. to-delete.txt must be ABSENT from the working tree (it was deleted).
    expect(fs.existsSync(path.join(safeOutputsRepo, "to-delete.txt"))).toBe(false);

    // 7. replacement.txt must be present.
    expect(fs.existsSync(path.join(safeOutputsRepo, "replacement.txt"))).toBe(true);

    // 8. The PR diff (origin/main..HEAD) must show:
    //    - to-delete.txt as DELETED (D)
    //    - replacement.txt as ADDED (A)
    const prDiffStatus = execGit(["diff", "--name-status", "origin/main", "HEAD"], { cwd: safeOutputsRepo }).stdout;
    expect(prDiffStatus).toMatch(/^D\s+to-delete\.txt/m);
    expect(prDiffStatus).toMatch(/^A\s+replacement\.txt/m);

    // 9. The commit diff (HEAD^..HEAD) must also show the deletion.
    const commitDiffStatus = execGit(["diff", "--name-status", "HEAD^", "HEAD"], { cwd: safeOutputsRepo }).stdout;
    expect(commitDiffStatus).toMatch(/^D\s+to-delete\.txt/m);
    expect(commitDiffStatus).toMatch(/^A\s+replacement\.txt/m);

    // 10. Exactly one linearized commit.
    const linearCommitCount = Number(execGit(["rev-list", "--count", "origin/main..HEAD"], { cwd: safeOutputsRepo }).stdout.trim());
    expect(linearCommitCount).toBe(1);
  });

  it("rewriteBundleBranchAsSingleCommit correctly preserves file modifications", async () => {
    // ─── Why this test exists ────────────────────────────────────────────────
    //
    // Existing tests only cover new file additions. This test verifies that
    // when the agent MODIFIES an existing file (changes its content),
    // rewriteBundleBranchAsSingleCommit produces a linearized commit that
    // reflects the final modified state — both the diff and the working tree
    // must show the updated content.
    //
    // It also covers the case where the agent makes multiple sequential commits
    // that each update the same file: only the final state matters.
    // ─────────────────────────────────────────────────────────────────────────

    const branchName = "feature/modify-file-rewrite";

    const bareRemote = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-mod-bare-"));
    const agentRepo = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-mod-agent-"));
    const safeOutputsRepo = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-mod-so-"));
    tempDirs.push(bareRemote, agentRepo, safeOutputsRepo);

    // 1. Initialize bare remote with a base file to modify.
    execGit(["init", "--bare", "-b", "main"], { cwd: bareRemote });
    execGit(["clone", bareRemote, "."], { cwd: agentRepo });
    execGit(["config", "user.name", "Agent"], { cwd: agentRepo });
    execGit(["config", "user.email", "agent@example.com"], { cwd: agentRepo });
    fs.writeFileSync(path.join(agentRepo, "config.txt"), "version: 1\n");
    fs.writeFileSync(path.join(agentRepo, "README.md"), "# Project\n");
    execGit(["add", "config.txt", "README.md"], { cwd: agentRepo });
    execGit(["commit", "-m", "Initial commit"], { cwd: agentRepo });
    execGit(["branch", "-M", "main"], { cwd: agentRepo });
    execGit(["push", "-u", "origin", "main"], { cwd: agentRepo });
    const agentBaseCommit = execGit(["rev-parse", "HEAD"], { cwd: agentRepo }).stdout.trim();

    // 2. Agent modifies config.txt twice (tests that only the final state is captured)
    //    and appends to README.md.
    execGit(["checkout", "-b", branchName], { cwd: agentRepo });

    fs.writeFileSync(path.join(agentRepo, "config.txt"), "version: 2\n");
    execGit(["add", "config.txt"], { cwd: agentRepo });
    execGit(["commit", "-m", "refactor: bump version to 2"], { cwd: agentRepo });

    fs.writeFileSync(path.join(agentRepo, "config.txt"), "version: 3\nfeature-flag: true\n");
    fs.writeFileSync(path.join(agentRepo, "README.md"), "# Project\n\nUpdated docs.\n");
    execGit(["add", "config.txt", "README.md"], { cwd: agentRepo });
    execGit(["commit", "-m", "refactor: final config and readme update"], { cwd: agentRepo });

    // 3. Bundle the feature branch (two commits in range).
    const bundlePath = path.join(agentRepo, "modify-file.bundle");
    execGit(["bundle", "create", bundlePath, `${agentBaseCommit}..refs/heads/${branchName}`], { cwd: agentRepo });

    // Confirm two commits in range before squash.
    const commitCount = Number(execGit(["rev-list", "--count", `${agentBaseCommit}..HEAD`], { cwd: agentRepo }).stdout.trim());
    expect(commitCount).toBe(2);

    // 4. Safe-outputs runner: fresh clone, apply bundle.
    execGit(["clone", bareRemote, "."], { cwd: safeOutputsRepo });
    execGit(["config", "user.name", "Runner"], { cwd: safeOutputsRepo });
    execGit(["config", "user.email", "runner@example.com"], { cwd: safeOutputsRepo });

    const { applyBundleToBranch, rewriteBundleBranchAsSingleCommit } = require("./create_pull_request.cjs");
    await applyBundleToBranch(bundlePath, branchName, `refs/heads/${branchName}`, createExecApi(safeOutputsRepo), "main");

    // 5. Rewrite two commits into one.
    await rewriteBundleBranchAsSingleCommit("main", createExecApi(safeOutputsRepo), bundlePath);

    // 6. Working tree must reflect the final modified state.
    expect(fs.readFileSync(path.join(safeOutputsRepo, "config.txt"), "utf8")).toBe("version: 3\nfeature-flag: true\n");
    expect(fs.readFileSync(path.join(safeOutputsRepo, "README.md"), "utf8")).toBe("# Project\n\nUpdated docs.\n");

    // 7. The PR diff must show config.txt and README.md as modified (M).
    const prDiffStatus = execGit(["diff", "--name-status", "origin/main", "HEAD"], { cwd: safeOutputsRepo }).stdout;
    expect(prDiffStatus).toMatch(/^M\s+README\.md/m);
    expect(prDiffStatus).toMatch(/^M\s+config\.txt/m);

    // 8. Exactly one linearized commit beyond origin/main.
    const linearCommitCount = Number(execGit(["rev-list", "--count", "origin/main..HEAD"], { cwd: safeOutputsRepo }).stdout.trim());
    expect(linearCommitCount).toBe(1);

    // 9. Exactly one parent.
    const parentLine = execGit(["log", "-1", "--format=%P", "HEAD"], { cwd: safeOutputsRepo }).stdout.trim();
    const parentShas = parentLine.split(/\s+/).filter(Boolean);
    expect(parentShas).toHaveLength(1);
  });

  it("rewriteBundleBranchAsSingleCommit excludes a modified existing file (unstages modification, not deletion)", async () => {
    // ─── Why this test exists ────────────────────────────────────────────────
    //
    // The existing excluded-files test only covers newly-added files. When the
    // excluded file already exists on the base branch and the agent modifies it,
    // `git reset HEAD -- <file>` must restore the index to the base version
    // rather than removing the file entirely. This test verifies that:
    //
    //   1. The modified content is NOT committed (file stays at base version).
    //   2. The non-excluded new file IS committed.
    //   3. The excluded file is absent from both the PR diff and the commit diff.
    //   4. The working tree still reflects the base version of the excluded file
    //      (the rebase onto the current base preserves it correctly).
    // ─────────────────────────────────────────────────────────────────────────

    const branchName = "feature/exclude-modified-file";

    const bareRemote = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-excl-mod-bare-"));
    const agentRepo = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-excl-mod-agent-"));
    const safeOutputsRepo = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-excl-mod-so-"));
    tempDirs.push(bareRemote, agentRepo, safeOutputsRepo);

    // 1. Initialize bare remote with a base that contains the file to be modified.
    execGit(["init", "--bare", "-b", "main"], { cwd: bareRemote });
    execGit(["clone", bareRemote, "."], { cwd: agentRepo });
    execGit(["config", "user.name", "Agent"], { cwd: agentRepo });
    execGit(["config", "user.email", "agent@example.com"], { cwd: agentRepo });
    fs.writeFileSync(path.join(agentRepo, "config.txt"), "version: 1\n");
    fs.writeFileSync(path.join(agentRepo, "README.md"), "# Project\n");
    execGit(["add", "config.txt", "README.md"], { cwd: agentRepo });
    execGit(["commit", "-m", "Initial commit"], { cwd: agentRepo });
    execGit(["branch", "-M", "main"], { cwd: agentRepo });
    execGit(["push", "-u", "origin", "main"], { cwd: agentRepo });
    const agentBaseCommit = execGit(["rev-parse", "HEAD"], { cwd: agentRepo }).stdout.trim();

    // 2. Agent modifies config.txt (should be excluded) and adds a new file (kept).
    execGit(["checkout", "-b", branchName], { cwd: agentRepo });
    fs.writeFileSync(path.join(agentRepo, "config.txt"), "version: 2\nsecret-token: abc123\n");
    fs.writeFileSync(path.join(agentRepo, "new-feature.txt"), "new feature\n");
    execGit(["add", "config.txt", "new-feature.txt"], { cwd: agentRepo });
    execGit(["commit", "-m", "feat: update config and add feature"], { cwd: agentRepo });

    // 3. Bundle the feature branch.
    const bundlePath = path.join(agentRepo, "excl-mod.bundle");
    execGit(["bundle", "create", bundlePath, `${agentBaseCommit}..refs/heads/${branchName}`], { cwd: agentRepo });

    // 4. Safe-outputs runner: fresh clone, apply bundle.
    execGit(["clone", bareRemote, "."], { cwd: safeOutputsRepo });
    execGit(["config", "user.name", "Runner"], { cwd: safeOutputsRepo });
    execGit(["config", "user.email", "runner@example.com"], { cwd: safeOutputsRepo });

    const { applyBundleToBranch, rewriteBundleBranchAsSingleCommit } = require("./create_pull_request.cjs");
    await applyBundleToBranch(bundlePath, branchName, `refs/heads/${branchName}`, createExecApi(safeOutputsRepo), "main");

    // 5. Rewrite with config.txt excluded.
    await rewriteBundleBranchAsSingleCommit("main", createExecApi(safeOutputsRepo), bundlePath, {
      excludedFiles: ["config.txt"],
    });

    // 6. config.txt must be at its BASE version (v1, not v2) in the working tree.
    //    The excluded modification must not be committed.
    expect(fs.readFileSync(path.join(safeOutputsRepo, "config.txt"), "utf8")).toBe("version: 1\n");

    // 7. new-feature.txt must be present (non-excluded file was kept).
    expect(fs.existsSync(path.join(safeOutputsRepo, "new-feature.txt"))).toBe(true);

    // 8. PR diff must show only new-feature.txt as added; config.txt must be absent.
    const prDiffStatus = execGit(["diff", "--name-status", "origin/main", "HEAD"], { cwd: safeOutputsRepo }).stdout;
    expect(prDiffStatus).toMatch(/^A\s+new-feature\.txt/m);
    expect(prDiffStatus).not.toMatch(/config\.txt/);

    // 9. Commit diff (HEAD^..HEAD) must not mention config.txt.
    const commitDiff = execGit(["diff", "--name-status", "HEAD^", "HEAD"], { cwd: safeOutputsRepo }).stdout;
    expect(commitDiff).not.toMatch(/config\.txt/);

    // 10. Exactly one linearized commit.
    const linearCommitCount = Number(execGit(["rev-list", "--count", "origin/main..HEAD"], { cwd: safeOutputsRepo }).stdout.trim());
    expect(linearCommitCount).toBe(1);
  });

  it("rewriteBundleBranchAsSingleCommit handles excluded files together with base drift", async () => {
    // ─── Why this test exists ────────────────────────────────────────────────
    //
    // Tests excludedFiles and base-drift correction independently already exist,
    // but their COMBINATION is untested. This is a realistic production scenario:
    //
    //   1. Agent runs while main is at A, adds kept.txt + secret.txt.
    //   2. Main advances to B (drift.txt added) before the runner processes it.
    //   3. rewriteBundleBranchAsSingleCommit is called with secret.txt excluded.
    //
    // The linearized commit must contain ONLY kept.txt:
    //   • secret.txt excluded (excluded file feature).
    //   • drift.txt absent from PR diff (base-drift feature: prereq A, not B).
    // ─────────────────────────────────────────────────────────────────────────

    const branchName = "feature/excl-and-drift";

    const bareRemote = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-excl-drift-bare-"));
    const agentRepo = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-excl-drift-agent-"));
    const safeOutputsRepo = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-excl-drift-so-"));
    tempDirs.push(bareRemote, agentRepo, safeOutputsRepo);

    // 1. Initialize bare remote at commit A.
    execGit(["init", "--bare", "-b", "main"], { cwd: bareRemote });
    execGit(["clone", bareRemote, "."], { cwd: agentRepo });
    execGit(["config", "user.name", "Agent"], { cwd: agentRepo });
    execGit(["config", "user.email", "agent@example.com"], { cwd: agentRepo });
    fs.writeFileSync(path.join(agentRepo, "README.md"), "# Project\n");
    execGit(["add", "README.md"], { cwd: agentRepo });
    execGit(["commit", "-m", "Initial commit"], { cwd: agentRepo });
    execGit(["branch", "-M", "main"], { cwd: agentRepo });
    execGit(["push", "-u", "origin", "main"], { cwd: agentRepo });
    const agentBaseCommit = execGit(["rev-parse", "HEAD"], { cwd: agentRepo }).stdout.trim();

    // 2. Agent adds kept.txt and secret.txt while main is still at A.
    execGit(["checkout", "-b", branchName], { cwd: agentRepo });
    fs.writeFileSync(path.join(agentRepo, "kept.txt"), "agent change\n");
    fs.writeFileSync(path.join(agentRepo, "secret.txt"), "sensitive data\n");
    execGit(["add", "kept.txt", "secret.txt"], { cwd: agentRepo });
    execGit(["commit", "-m", "feat: add kept and secret files"], { cwd: agentRepo });

    // 3. Bundle while main is STILL at A (before drift).
    const bundlePath = path.join(agentRepo, "excl-drift.bundle");
    execGit(["bundle", "create", bundlePath, `${agentBaseCommit}..refs/heads/${branchName}`], { cwd: agentRepo });

    // 4. AFTER bundling: base drifts to B (drift.txt added).
    execGit(["checkout", "main"], { cwd: agentRepo });
    fs.writeFileSync(path.join(agentRepo, "drift.txt"), "base drift\n");
    execGit(["add", "drift.txt"], { cwd: agentRepo });
    execGit(["commit", "-m", "chore: base drift"], { cwd: agentRepo });
    execGit(["push", "origin", "main"], { cwd: agentRepo });

    // 5. Safe-outputs runner clones updated origin/main (B includes drift.txt).
    execGit(["clone", bareRemote, "."], { cwd: safeOutputsRepo });
    execGit(["config", "user.name", "Runner"], { cwd: safeOutputsRepo });
    execGit(["config", "user.email", "runner@example.com"], { cwd: safeOutputsRepo });
    const newOriginMainSha = execGit(["rev-parse", "origin/main"], { cwd: safeOutputsRepo }).stdout.trim();
    expect(newOriginMainSha).not.toBe(agentBaseCommit);

    const { applyBundleToBranch, rewriteBundleBranchAsSingleCommit } = require("./create_pull_request.cjs");
    await applyBundleToBranch(bundlePath, branchName, `refs/heads/${branchName}`, createExecApi(safeOutputsRepo), "main");

    // 6. Rewrite with secret.txt excluded.
    await rewriteBundleBranchAsSingleCommit("main", createExecApi(safeOutputsRepo), bundlePath, {
      excludedFiles: ["secret.txt"],
    });

    // 7. kept.txt must be present (agent's non-excluded change preserved).
    expect(fs.existsSync(path.join(safeOutputsRepo, "kept.txt"))).toBe(true);

    // 8. PR diff (origin/main..HEAD) must contain only kept.txt.
    //    - secret.txt excluded → not in diff.
    //    - drift.txt already on origin/main → not in diff (base-drift correction).
    const prDiffNames = execGit(["diff", "--name-only", "origin/main", "HEAD"], { cwd: safeOutputsRepo }).stdout.trim().split("\n").filter(Boolean).sort();
    expect(prDiffNames).toEqual(["kept.txt"]);

    // 9. Exactly one linearized commit.
    const linearCommitCount = Number(execGit(["rev-list", "--count", "origin/main..HEAD"], { cwd: safeOutputsRepo }).stdout.trim());
    expect(linearCommitCount).toBe(1);
  });

  it("rewriteBundleBranchAsSingleCommit falls back to origin/main when no bundleFilePath is supplied", async () => {
    // ─── Why this test exists ────────────────────────────────────────────────
    //
    // When rewriteBundleBranchAsSingleCommit is called without a bundle file
    // path (undefined/null), it must skip prerequisite extraction entirely and
    // fall back to origin/<baseBranch> as the linearization base. This is the
    // code path exercised when the signed-push rewrite is invoked outside the
    // bundle flow (e.g. after a direct branch push). Verify that:
    //
    //   • The function succeeds and produces a single linearized commit.
    //   • The PR diff contains only the agent's changes.
    // ─────────────────────────────────────────────────────────────────────────

    const branchName = "feature/no-bundle-path";

    const bareRemote = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-no-bundle-bare-"));
    const agentRepo = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-no-bundle-agent-"));
    const safeOutputsRepo = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-no-bundle-so-"));
    tempDirs.push(bareRemote, agentRepo, safeOutputsRepo);

    // 1. Initialize bare remote and seed main.
    execGit(["init", "--bare", "-b", "main"], { cwd: bareRemote });
    execGit(["clone", bareRemote, "."], { cwd: agentRepo });
    execGit(["config", "user.name", "Agent"], { cwd: agentRepo });
    execGit(["config", "user.email", "agent@example.com"], { cwd: agentRepo });
    fs.writeFileSync(path.join(agentRepo, "README.md"), "# Project\n");
    execGit(["add", "README.md"], { cwd: agentRepo });
    execGit(["commit", "-m", "Initial commit"], { cwd: agentRepo });
    execGit(["branch", "-M", "main"], { cwd: agentRepo });
    execGit(["push", "-u", "origin", "main"], { cwd: agentRepo });

    // 2. Agent creates feature branch with two commits.
    execGit(["checkout", "-b", branchName], { cwd: agentRepo });
    fs.writeFileSync(path.join(agentRepo, "alpha.txt"), "alpha\n");
    execGit(["add", "alpha.txt"], { cwd: agentRepo });
    execGit(["commit", "-m", "feat: add alpha"], { cwd: agentRepo });
    fs.writeFileSync(path.join(agentRepo, "beta.txt"), "beta\n");
    execGit(["add", "beta.txt"], { cwd: agentRepo });
    execGit(["commit", "-m", "feat: add beta"], { cwd: agentRepo });

    // 3. Safe-outputs runner: push the branch directly (simulate a non-bundle path).
    execGit(["clone", bareRemote, "."], { cwd: safeOutputsRepo });
    execGit(["config", "user.name", "Runner"], { cwd: safeOutputsRepo });
    execGit(["config", "user.email", "runner@example.com"], { cwd: safeOutputsRepo });
    // Copy the commits to the runner without using a bundle.
    execGit(["fetch", agentRepo, `${branchName}:${branchName}`], { cwd: safeOutputsRepo });
    execGit(["checkout", branchName], { cwd: safeOutputsRepo });

    const { rewriteBundleBranchAsSingleCommit } = require("./create_pull_request.cjs");

    // 4. Call without bundleFilePath (undefined) — must fall back to origin/main.
    await rewriteBundleBranchAsSingleCommit("main", createExecApi(safeOutputsRepo), undefined);

    // 5. Exactly one commit beyond origin/main.
    const linearCommitCount = Number(execGit(["rev-list", "--count", "origin/main..HEAD"], { cwd: safeOutputsRepo }).stdout.trim());
    expect(linearCommitCount).toBe(1);

    // 6. PR diff must contain both agent files and nothing else.
    const diffNames = execGit(["diff", "--name-only", "origin/main..HEAD"], { cwd: safeOutputsRepo }).stdout.trim().split("\n").filter(Boolean).sort();
    expect(diffNames).toEqual(["alpha.txt", "beta.txt"]);

    // 7. Exactly one parent (linearized, not a merge commit).
    const parentLine = execGit(["log", "-1", "--format=%P", "HEAD"], { cwd: safeOutputsRepo }).stdout.trim();
    const parentShas = parentLine.split(/\s+/).filter(Boolean);
    expect(parentShas).toHaveLength(1);
  });

  it("rewriteBundleBranchAsSingleCommit preserves a file rename", async () => {
    // ─── Why this test exists ────────────────────────────────────────────────
    //
    // File additions and deletions are tested separately. A rename is a logical
    // combination (delete + add the same content), but git tracks it explicitly
    // with rename detection. Verify that after linearization:
    //
    //   • The original filename is gone from the working tree and the PR diff
    //     shows it as deleted (or as a rename).
    //   • The new filename is present with the correct content.
    //   • No spurious extra changes appear in the PR diff.
    // ─────────────────────────────────────────────────────────────────────────

    const branchName = "feature/rename-file";

    const bareRemote = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-rename-bare-"));
    const agentRepo = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-rename-agent-"));
    const safeOutputsRepo = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-rename-so-"));
    tempDirs.push(bareRemote, agentRepo, safeOutputsRepo);

    // 1. Initialize bare remote with a file that will be renamed.
    execGit(["init", "--bare", "-b", "main"], { cwd: bareRemote });
    execGit(["clone", bareRemote, "."], { cwd: agentRepo });
    execGit(["config", "user.name", "Agent"], { cwd: agentRepo });
    execGit(["config", "user.email", "agent@example.com"], { cwd: agentRepo });
    fs.writeFileSync(path.join(agentRepo, "old-name.txt"), "file content\n");
    fs.writeFileSync(path.join(agentRepo, "README.md"), "# Project\n");
    execGit(["add", "old-name.txt", "README.md"], { cwd: agentRepo });
    execGit(["commit", "-m", "Initial commit"], { cwd: agentRepo });
    execGit(["branch", "-M", "main"], { cwd: agentRepo });
    execGit(["push", "-u", "origin", "main"], { cwd: agentRepo });
    const agentBaseCommit = execGit(["rev-parse", "HEAD"], { cwd: agentRepo }).stdout.trim();

    // 2. Agent renames old-name.txt → new-name.txt.
    execGit(["checkout", "-b", branchName], { cwd: agentRepo });
    execGit(["mv", "old-name.txt", "new-name.txt"], { cwd: agentRepo });
    execGit(["commit", "-m", "refactor: rename old-name.txt to new-name.txt"], { cwd: agentRepo });

    // 3. Bundle the feature branch.
    const bundlePath = path.join(agentRepo, "rename.bundle");
    execGit(["bundle", "create", bundlePath, `${agentBaseCommit}..refs/heads/${branchName}`], { cwd: agentRepo });

    // 4. Safe-outputs runner: fresh clone, apply bundle.
    execGit(["clone", bareRemote, "."], { cwd: safeOutputsRepo });
    execGit(["config", "user.name", "Runner"], { cwd: safeOutputsRepo });
    execGit(["config", "user.email", "runner@example.com"], { cwd: safeOutputsRepo });

    const { applyBundleToBranch, rewriteBundleBranchAsSingleCommit } = require("./create_pull_request.cjs");
    await applyBundleToBranch(bundlePath, branchName, `refs/heads/${branchName}`, createExecApi(safeOutputsRepo), "main");

    // 5. Rewrite.
    await rewriteBundleBranchAsSingleCommit("main", createExecApi(safeOutputsRepo), bundlePath);

    // 6. old-name.txt must be gone; new-name.txt must be present with correct content.
    expect(fs.existsSync(path.join(safeOutputsRepo, "old-name.txt"))).toBe(false);
    expect(fs.existsSync(path.join(safeOutputsRepo, "new-name.txt"))).toBe(true);
    expect(fs.readFileSync(path.join(safeOutputsRepo, "new-name.txt"), "utf8")).toBe("file content\n");

    // 7. PR diff must show old-name.txt removed and new-name.txt added
    //    (rename detection may show D/A or R depending on similarity threshold).
    const prDiffStatus = execGit(["diff", "--name-status", "origin/main", "HEAD"], { cwd: safeOutputsRepo }).stdout;
    expect(prDiffStatus).toMatch(/old-name\.txt/);
    expect(prDiffStatus).toMatch(/new-name\.txt/);

    // 8. README.md must NOT appear in the PR diff (unmodified base file).
    expect(prDiffStatus).not.toMatch(/README\.md/);

    // 9. Exactly one linearized commit.
    const linearCommitCount = Number(execGit(["rev-list", "--count", "origin/main..HEAD"], { cwd: safeOutputsRepo }).stdout.trim());
    expect(linearCommitCount).toBe(1);
  });

  it("rewriteBundleBranchAsSingleCommit uses the latest commit subject as the linearized commit message", async () => {
    // ─── Why this test exists ────────────────────────────────────────────────
    //
    // rewriteBundleBranchAsSingleCommit reads the HEAD commit subject with
    // `git log -1 --format=%s` and uses it as the message for the synthesized
    // single commit. When the agent has multiple commits, the LATEST subject
    // must be used (not the first, not a default fallback). This verifies that
    // the commit message forwarding works end-to-end in a real repository.
    // ─────────────────────────────────────────────────────────────────────────

    const branchName = "feature/commit-message-preservation";

    const bareRemote = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-msg-bare-"));
    const agentRepo = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-msg-agent-"));
    const safeOutputsRepo = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-msg-so-"));
    tempDirs.push(bareRemote, agentRepo, safeOutputsRepo);

    // 1. Initialize bare remote and seed main.
    execGit(["init", "--bare", "-b", "main"], { cwd: bareRemote });
    execGit(["clone", bareRemote, "."], { cwd: agentRepo });
    execGit(["config", "user.name", "Agent"], { cwd: agentRepo });
    execGit(["config", "user.email", "agent@example.com"], { cwd: agentRepo });
    fs.writeFileSync(path.join(agentRepo, "README.md"), "# Project\n");
    execGit(["add", "README.md"], { cwd: agentRepo });
    execGit(["commit", "-m", "Initial commit"], { cwd: agentRepo });
    execGit(["branch", "-M", "main"], { cwd: agentRepo });
    execGit(["push", "-u", "origin", "main"], { cwd: agentRepo });
    const agentBaseCommit = execGit(["rev-parse", "HEAD"], { cwd: agentRepo }).stdout.trim();

    // 2. Agent makes two commits; the second has the subject that should be used.
    execGit(["checkout", "-b", branchName], { cwd: agentRepo });
    fs.writeFileSync(path.join(agentRepo, "step1.txt"), "step 1\n");
    execGit(["add", "step1.txt"], { cwd: agentRepo });
    execGit(["commit", "-m", "wip: intermediate step"], { cwd: agentRepo });

    fs.writeFileSync(path.join(agentRepo, "step2.txt"), "step 2\n");
    execGit(["add", "step2.txt"], { cwd: agentRepo });
    execGit(["commit", "-m", "feat: implement feature X"], { cwd: agentRepo });

    // 3. Bundle the feature branch.
    const bundlePath = path.join(agentRepo, "msg.bundle");
    execGit(["bundle", "create", bundlePath, `${agentBaseCommit}..refs/heads/${branchName}`], { cwd: agentRepo });

    // 4. Safe-outputs runner: fresh clone, apply bundle.
    execGit(["clone", bareRemote, "."], { cwd: safeOutputsRepo });
    execGit(["config", "user.name", "Runner"], { cwd: safeOutputsRepo });
    execGit(["config", "user.email", "runner@example.com"], { cwd: safeOutputsRepo });

    const { applyBundleToBranch, rewriteBundleBranchAsSingleCommit } = require("./create_pull_request.cjs");
    await applyBundleToBranch(bundlePath, branchName, `refs/heads/${branchName}`, createExecApi(safeOutputsRepo), "main");

    // 5. Rewrite.
    await rewriteBundleBranchAsSingleCommit("main", createExecApi(safeOutputsRepo), bundlePath);

    // 6. The linearized commit message must match the LATEST agent commit subject.
    const commitSubject = execGit(["log", "-1", "--format=%s", "HEAD"], { cwd: safeOutputsRepo }).stdout.trim();
    expect(commitSubject).toBe("feat: implement feature X");

    // 7. Both files from the agent's commits must be in the PR diff.
    const diffNames = execGit(["diff", "--name-only", "origin/main..HEAD"], { cwd: safeOutputsRepo }).stdout.trim().split("\n").filter(Boolean).sort();
    expect(diffNames).toEqual(["step1.txt", "step2.txt"]);

    // 8. Exactly one linearized commit.
    const linearCommitCount = Number(execGit(["rev-list", "--count", "origin/main..HEAD"], { cwd: safeOutputsRepo }).stdout.trim());
    expect(linearCommitCount).toBe(1);
  });
});
