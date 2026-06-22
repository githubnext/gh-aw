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
});
