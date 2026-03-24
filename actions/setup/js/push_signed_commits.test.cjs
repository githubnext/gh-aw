/**
 * Integration tests for push_signed_commits.cjs
 *
 * These tests run REAL git commands to verify that pushSignedCommits:
 * 1. Correctly enumerates new commits via `git rev-list`
 * 2. Reads file contents and builds the GraphQL payload
 * 3. Calls the GitHub GraphQL `createCommitOnBranch` mutation for each commit
 * 4. Falls back to `git push` when the GraphQL mutation fails
 *
 * A bare git repository is used as the stand-in "remote" so that ls-remote
 * and push commands work without a real network connection.
 * The GraphQL layer is always mocked because it requires a real GitHub API.
 */

// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { createRequire } from "module";
import fs from "fs";
import path from "path";
import { spawnSync } from "child_process";
import os from "os";

const require = createRequire(import.meta.url);

// Import module once – globals are resolved at call time, not import time.
const { pushSignedCommits } = require("./push_signed_commits.cjs");

// ──────────────────────────────────────────────────────────────────────────────
// Git helpers (real subprocess – no mocking)
// ──────────────────────────────────────────────────────────────────────────────

/**
 * @param {string[]} args
 * @param {{ cwd?: string, allowFailure?: boolean }} [options]
 */
function execGit(args, options = {}) {
  const result = spawnSync("git", args, {
    encoding: "utf8",
    env: {
      ...process.env,
      GIT_CONFIG_NOSYSTEM: "1",
      HOME: os.tmpdir(),
    },
    ...options,
  });
  if (result.error) throw result.error;
  if (result.status !== 0 && !options.allowFailure) {
    throw new Error(`git ${args.join(" ")} failed (cwd=${options.cwd}):\n${result.stderr}`);
  }
  return result;
}

/**
 * Create a bare repository that acts as the remote "origin".
 * @returns {string} Path to the bare repository
 */
function createBareRepo() {
  const bareDir = fs.mkdtempSync(path.join(os.tmpdir(), "push-signed-bare-"));
  execGit(["init", "--bare"], { cwd: bareDir });
  // Ensure the bare repo uses "main" as the default branch
  execGit(["symbolic-ref", "HEAD", "refs/heads/main"], { cwd: bareDir });
  return bareDir;
}

/**
 * Clone the bare repo and set up a working copy with an initial commit on `main`.
 * @param {string} bareDir
 * @returns {string} Path to the working copy
 */
function createWorkingRepo(bareDir) {
  const workDir = fs.mkdtempSync(path.join(os.tmpdir(), "push-signed-work-"));
  execGit(["clone", bareDir, "."], { cwd: workDir });
  execGit(["config", "user.name", "Test User"], { cwd: workDir });
  execGit(["config", "user.email", "test@example.com"], { cwd: workDir });

  // Initial commit on main
  fs.writeFileSync(path.join(workDir, "README.md"), "# Test\n");
  execGit(["add", "."], { cwd: workDir });
  execGit(["commit", "-m", "Initial commit"], { cwd: workDir });
  // Rename to main if git defaulted to master
  execGit(["branch", "-M", "main"], { cwd: workDir });
  execGit(["push", "-u", "origin", "main"], { cwd: workDir });

  return workDir;
}

/** @param {string} dir */
function cleanupDir(dir) {
  if (dir && fs.existsSync(dir)) {
    fs.rmSync(dir, { recursive: true, force: true });
  }
}

// ──────────────────────────────────────────────────────────────────────────────
// Global stubs required by push_signed_commits.cjs
// ──────────────────────────────────────────────────────────────────────────────

/**
 * Build an `exec` global stub that runs real git commands via spawnSync.
 * @param {string} cwd
 */
function makeRealExec(cwd) {
  return {
    /**
     * @param {string} program
     * @param {string[]} args
     * @param {{ cwd?: string }} [opts]
     */
    getExecOutput: async (program, args, opts = {}) => {
      const result = spawnSync(program, args, {
        encoding: "utf8",
        cwd: opts.cwd ?? cwd,
        env: {
          ...process.env,
          GIT_CONFIG_NOSYSTEM: "1",
          HOME: os.tmpdir(),
        },
      });
      if (result.error) throw result.error;
      return { exitCode: result.status ?? 0, stdout: result.stdout ?? "", stderr: result.stderr ?? "" };
    },
    /**
     * @param {string} program
     * @param {string[]} args
     * @param {{ cwd?: string, env?: NodeJS.ProcessEnv }} [opts]
     */
    exec: async (program, args, opts = {}) => {
      const result = spawnSync(program, args, {
        encoding: "utf8",
        cwd: opts.cwd ?? cwd,
        env: opts.env ?? { ...process.env, GIT_CONFIG_NOSYSTEM: "1", HOME: os.tmpdir() },
      });
      if (result.error) throw result.error;
      if (result.status !== 0) {
        throw new Error(`${program} ${args.join(" ")} failed:\n${result.stderr}`);
      }
      return result.status ?? 0;
    },
  };
}

// ──────────────────────────────────────────────────────────────────────────────
// Tests
// ──────────────────────────────────────────────────────────────────────────────

describe("push_signed_commits integration tests", () => {
  let bareDir;
  let workDir;
  let mockCore;
  let capturedGraphQLCalls;

  /** @returns {any} */
  function makeMockGithubClient(options = {}) {
    const { failWithError = null, oid = "signed-oid-abc123" } = options;
    return {
      graphql: vi.fn(async query => {
        if (failWithError) throw failWithError;
        capturedGraphQLCalls.push({ oid, query });
        return { createCommitOnBranch: { commit: { oid } } };
      }),
      rest: {
        git: {
          createRef: vi.fn(async () => ({ data: {} })),
        },
      },
    };
  }

  beforeEach(() => {
    bareDir = createBareRepo();
    workDir = createWorkingRepo(bareDir);
    capturedGraphQLCalls = [];

    mockCore = {
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      debug: vi.fn(),
    };

    global.core = mockCore;
  });

  afterEach(() => {
    cleanupDir(bareDir);
    cleanupDir(workDir);
    delete global.core;
    delete global.exec;
    vi.clearAllMocks();
  });

  // ──────────────────────────────────────────────────────
  // Happy path – GraphQL succeeds
  // ──────────────────────────────────────────────────────

  describe("GraphQL signed commits (happy path)", () => {
    it("should call GraphQL for a single new commit", async () => {
      // Create a feature branch with one new file
      execGit(["checkout", "-b", "feature-branch"], { cwd: workDir });
      fs.writeFileSync(path.join(workDir, "hello.txt"), "Hello World\n");
      execGit(["add", "hello.txt"], { cwd: workDir });
      execGit(["commit", "-m", "Add hello.txt"], { cwd: workDir });
      // Push the branch so ls-remote can resolve its OID
      execGit(["push", "-u", "origin", "feature-branch"], { cwd: workDir });

      global.exec = makeRealExec(workDir);
      const githubClient = makeMockGithubClient();

      await pushSignedCommits({
        githubClient,
        owner: "test-owner",
        repo: "test-repo",
        branch: "feature-branch",
        baseRef: "origin/main",
        cwd: workDir,
      });

      expect(githubClient.graphql).toHaveBeenCalledTimes(1);
      // Verify the mutation query targets createCommitOnBranch
      const [query, variables] = githubClient.graphql.mock.calls[0];
      expect(query).toContain("createCommitOnBranch");
      expect(query).toContain("CreateCommitOnBranchInput");
      // Verify the input structure
      expect(variables.input.branch.branchName).toBe("feature-branch");
      expect(variables.input.branch.repositoryNameWithOwner).toBe("test-owner/test-repo");
      expect(variables.input.message.headline).toBe("Add hello.txt");
      // hello.txt should appear in additions with base64 content
      expect(variables.input.fileChanges.additions).toHaveLength(1);
      expect(variables.input.fileChanges.additions[0].path).toBe("hello.txt");
      expect(Buffer.from(variables.input.fileChanges.additions[0].contents, "base64").toString()).toBe("Hello World\n");
    });

    it("should call GraphQL once per commit for multiple new commits", async () => {
      execGit(["checkout", "-b", "multi-commit-branch"], { cwd: workDir });

      fs.writeFileSync(path.join(workDir, "file-a.txt"), "File A\n");
      execGit(["add", "file-a.txt"], { cwd: workDir });
      execGit(["commit", "-m", "Add file-a.txt"], { cwd: workDir });

      fs.writeFileSync(path.join(workDir, "file-b.txt"), "File B\n");
      execGit(["add", "file-b.txt"], { cwd: workDir });
      execGit(["commit", "-m", "Add file-b.txt"], { cwd: workDir });

      execGit(["push", "-u", "origin", "multi-commit-branch"], { cwd: workDir });

      global.exec = makeRealExec(workDir);
      const githubClient = makeMockGithubClient();

      await pushSignedCommits({
        githubClient,
        owner: "test-owner",
        repo: "test-repo",
        branch: "multi-commit-branch",
        baseRef: "origin/main",
        cwd: workDir,
      });

      expect(githubClient.graphql).toHaveBeenCalledTimes(2);
      const headlines = githubClient.graphql.mock.calls.map(c => c[1].input.message.headline);
      expect(headlines).toEqual(["Add file-a.txt", "Add file-b.txt"]);
    });

    it("should include deletions when files are removed in a commit", async () => {
      execGit(["checkout", "-b", "delete-branch"], { cwd: workDir });

      // First add a file, push, then delete it
      fs.writeFileSync(path.join(workDir, "to-delete.txt"), "Will be deleted\n");
      execGit(["add", "to-delete.txt"], { cwd: workDir });
      execGit(["commit", "-m", "Add file to delete"], { cwd: workDir });
      execGit(["push", "-u", "origin", "delete-branch"], { cwd: workDir });

      // Now delete the file
      fs.unlinkSync(path.join(workDir, "to-delete.txt"));
      execGit(["add", "-u"], { cwd: workDir });
      execGit(["commit", "-m", "Delete file"], { cwd: workDir });
      execGit(["push", "origin", "delete-branch"], { cwd: workDir });

      global.exec = makeRealExec(workDir);
      const githubClient = makeMockGithubClient();

      await pushSignedCommits({
        githubClient,
        owner: "test-owner",
        repo: "test-repo",
        branch: "delete-branch",
        // Only replay the delete commit
        baseRef: "delete-branch^",
        cwd: workDir,
      });

      expect(githubClient.graphql).toHaveBeenCalledTimes(1);
      const callArg = githubClient.graphql.mock.calls[0][1].input;
      expect(callArg.fileChanges.deletions).toEqual([{ path: "to-delete.txt" }]);
      expect(callArg.fileChanges.additions).toHaveLength(0);
    });

    it("should handle commit with no file changes (empty commit)", async () => {
      execGit(["checkout", "-b", "empty-diff-branch"], { cwd: workDir });
      execGit(["push", "-u", "origin", "empty-diff-branch"], { cwd: workDir });

      // Allow an empty commit
      execGit(["commit", "--allow-empty", "-m", "Empty commit"], { cwd: workDir });
      execGit(["push", "origin", "empty-diff-branch"], { cwd: workDir });

      global.exec = makeRealExec(workDir);
      const githubClient = makeMockGithubClient();

      await pushSignedCommits({
        githubClient,
        owner: "test-owner",
        repo: "test-repo",
        branch: "empty-diff-branch",
        baseRef: "origin/main",
        cwd: workDir,
      });

      expect(githubClient.graphql).toHaveBeenCalledTimes(1);
      const callArg = githubClient.graphql.mock.calls[0][1].input;
      expect(callArg.fileChanges.additions).toHaveLength(0);
      expect(callArg.fileChanges.deletions).toHaveLength(0);
    });

    it("should do nothing when there are no new commits", async () => {
      execGit(["checkout", "-b", "no-commits-branch"], { cwd: workDir });
      execGit(["push", "-u", "origin", "no-commits-branch"], { cwd: workDir });

      global.exec = makeRealExec(workDir);
      const githubClient = makeMockGithubClient();

      // baseRef points to the same HEAD – no commits to replay
      await pushSignedCommits({
        githubClient,
        owner: "test-owner",
        repo: "test-repo",
        branch: "no-commits-branch",
        baseRef: "origin/no-commits-branch",
        cwd: workDir,
      });

      expect(githubClient.graphql).not.toHaveBeenCalled();
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("no new commits"));
    });
  });

  // ──────────────────────────────────────────────────────
  // New branch – branch does not yet exist on remote
  // ──────────────────────────────────────────────────────

  describe("new branch (does not exist on remote)", () => {
    it("should create remote branch via REST and use parent OID for first commit (single commit)", async () => {
      // Create a local branch with one commit but do NOT push it
      execGit(["checkout", "-b", "new-unpushed-branch"], { cwd: workDir });
      fs.writeFileSync(path.join(workDir, "new-file.txt"), "New file content\n");
      execGit(["add", "new-file.txt"], { cwd: workDir });
      execGit(["commit", "-m", "Add new-file.txt"], { cwd: workDir });

      // Capture the local parent OID (main HEAD before the new commit)
      const expectedParentOid = execGit(["rev-parse", "HEAD^"], { cwd: workDir }).stdout.trim();

      global.exec = makeRealExec(workDir);
      const githubClient = makeMockGithubClient();

      await pushSignedCommits({
        githubClient,
        owner: "test-owner",
        repo: "test-repo",
        branch: "new-unpushed-branch",
        baseRef: "origin/main",
        cwd: workDir,
      });

      // Branch must be created via REST before the GraphQL mutation
      expect(githubClient.rest.git.createRef).toHaveBeenCalledTimes(1);
      expect(githubClient.rest.git.createRef).toHaveBeenCalledWith({
        owner: "test-owner",
        repo: "test-repo",
        ref: "refs/heads/new-unpushed-branch",
        sha: expectedParentOid,
      });

      expect(githubClient.graphql).toHaveBeenCalledTimes(1);
      const callArg = githubClient.graphql.mock.calls[0][1].input;
      // expectedHeadOid must be the parent commit OID, not empty
      expect(callArg.expectedHeadOid).toBe(expectedParentOid);
      expect(callArg.branch.branchName).toBe("new-unpushed-branch");
      expect(callArg.message.headline).toBe("Add new-file.txt");
      // Verify the info log was emitted
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("not yet on the remote"));
    });

    it("should create remote branch once then chain GraphQL OIDs for multiple commits on a new branch", async () => {
      // Create a local branch with two commits but do NOT push it
      execGit(["checkout", "-b", "new-multi-commit-branch"], { cwd: workDir });

      fs.writeFileSync(path.join(workDir, "alpha.txt"), "Alpha\n");
      execGit(["add", "alpha.txt"], { cwd: workDir });
      execGit(["commit", "-m", "Add alpha.txt"], { cwd: workDir });

      fs.writeFileSync(path.join(workDir, "beta.txt"), "Beta\n");
      execGit(["add", "beta.txt"], { cwd: workDir });
      execGit(["commit", "-m", "Add beta.txt"], { cwd: workDir });

      // The parent OID of the first commit is main's HEAD (two commits back from current)
      const expectedParentOid = execGit(["rev-parse", "HEAD^^"], { cwd: workDir }).stdout.trim();

      global.exec = makeRealExec(workDir);
      // Mock returns the same OID for all calls; second call must use that OID
      const githubClient = makeMockGithubClient({ oid: "signed-oid-first" });

      await pushSignedCommits({
        githubClient,
        owner: "test-owner",
        repo: "test-repo",
        branch: "new-multi-commit-branch",
        baseRef: "origin/main",
        cwd: workDir,
      });

      // Branch must be created via REST only once (for the first commit)
      expect(githubClient.rest.git.createRef).toHaveBeenCalledTimes(1);
      expect(githubClient.rest.git.createRef).toHaveBeenCalledWith({
        owner: "test-owner",
        repo: "test-repo",
        ref: "refs/heads/new-multi-commit-branch",
        sha: expectedParentOid,
      });

      expect(githubClient.graphql).toHaveBeenCalledTimes(2);

      // First call: expectedHeadOid is the parent commit OID (resolved via git rev-parse)
      const firstCallArg = githubClient.graphql.mock.calls[0][1].input;
      expect(firstCallArg.expectedHeadOid).toBe(expectedParentOid);
      expect(firstCallArg.message.headline).toBe("Add alpha.txt");

      // Second call: expectedHeadOid is the OID returned by the first GraphQL mutation
      const secondCallArg = githubClient.graphql.mock.calls[1][1].input;
      expect(secondCallArg.expectedHeadOid).toBe("signed-oid-first");
      expect(secondCallArg.message.headline).toBe("Add beta.txt");
    });

    it("should continue with signed commits when createRef returns 422 (concurrent branch creation)", async () => {
      execGit(["checkout", "-b", "race-condition-branch"], { cwd: workDir });
      fs.writeFileSync(path.join(workDir, "race.txt"), "Race content\n");
      execGit(["add", "race.txt"], { cwd: workDir });
      execGit(["commit", "-m", "Race commit"], { cwd: workDir });

      const expectedParentOid = execGit(["rev-parse", "HEAD^"], { cwd: workDir }).stdout.trim();

      global.exec = makeRealExec(workDir);

      // Simulate concurrent branch creation: createRef throws 422 (GitHub API exact format)
      const concurrentError = Object.assign(new Error("Reference refs/heads/race-condition-branch already exists"), { status: 422 });
      const githubClient = {
        graphql: vi.fn(async () => ({ createCommitOnBranch: { commit: { oid: "signed-oid-race" } } })),
        rest: {
          git: {
            createRef: vi.fn(async () => {
              throw concurrentError;
            }),
          },
        },
      };

      await pushSignedCommits({
        githubClient,
        owner: "test-owner",
        repo: "test-repo",
        branch: "race-condition-branch",
        baseRef: "origin/main",
        cwd: workDir,
      });

      // createRef was attempted but threw 422 – should continue, not fall back
      expect(githubClient.rest.git.createRef).toHaveBeenCalledTimes(1);
      expect(githubClient.graphql).toHaveBeenCalledTimes(1);
      const callArg = githubClient.graphql.mock.calls[0][1].input;
      expect(callArg.expectedHeadOid).toBe(expectedParentOid);
      expect(callArg.message.headline).toBe("Race commit");
      // Should log the concurrent-creation info message
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("created concurrently"));
    });
  });

  // ──────────────────────────────────────────────────────
  // Fallback path – GraphQL fails → git push
  // ──────────────────────────────────────────────────────

  describe("git push fallback when GraphQL fails", () => {
    it("should fall back to git push when GraphQL throws", async () => {
      execGit(["checkout", "-b", "fallback-branch"], { cwd: workDir });
      fs.writeFileSync(path.join(workDir, "fallback.txt"), "Fallback content\n");
      execGit(["add", "fallback.txt"], { cwd: workDir });
      execGit(["commit", "-m", "Fallback commit"], { cwd: workDir });
      execGit(["push", "-u", "origin", "fallback-branch"], { cwd: workDir });

      // Add another commit that will be pushed via git push fallback
      fs.writeFileSync(path.join(workDir, "extra.txt"), "Extra content\n");
      execGit(["add", "extra.txt"], { cwd: workDir });
      execGit(["commit", "-m", "Extra commit"], { cwd: workDir });

      global.exec = makeRealExec(workDir);
      const githubClient = makeMockGithubClient({ failWithError: new Error("GraphQL: not supported on GHES") });

      await pushSignedCommits({
        githubClient,
        owner: "test-owner",
        repo: "test-repo",
        branch: "fallback-branch",
        baseRef: "origin/fallback-branch",
        cwd: workDir,
      });

      // Should warn and fall back
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("falling back to git push"));

      // The commit should now be on the remote (verified via ls-remote)
      const lsRemote = execGit(["ls-remote", bareDir, "refs/heads/fallback-branch"], { cwd: workDir });
      const remoteOid = lsRemote.stdout.trim().split(/\s+/)[0];
      const localOid = execGit(["rev-parse", "HEAD"], { cwd: workDir }).stdout.trim();
      expect(remoteOid).toBe(localOid);
    });
  });

  // ──────────────────────────────────────────────────────
  // Commit message – multi-line body
  // ──────────────────────────────────────────────────────

  describe("commit message handling", () => {
    it("should include the commit body when present", async () => {
      execGit(["checkout", "-b", "body-branch"], { cwd: workDir });
      fs.writeFileSync(path.join(workDir, "described.txt"), "content\n");
      execGit(["add", "described.txt"], { cwd: workDir });
      execGit(["commit", "-m", "Subject line\n\nDetailed body text\n\nMore details here"], { cwd: workDir });
      execGit(["push", "-u", "origin", "body-branch"], { cwd: workDir });

      global.exec = makeRealExec(workDir);
      const githubClient = makeMockGithubClient();

      await pushSignedCommits({
        githubClient,
        owner: "test-owner",
        repo: "test-repo",
        branch: "body-branch",
        baseRef: "origin/main",
        cwd: workDir,
      });

      const callArg = githubClient.graphql.mock.calls[0][1].input;
      expect(callArg.message.headline).toBe("Subject line");
      expect(callArg.message.body).toContain("Detailed body text");
    });

    it("should omit the body field when commit message has no body", async () => {
      execGit(["checkout", "-b", "no-body-branch"], { cwd: workDir });
      fs.writeFileSync(path.join(workDir, "nodesc.txt"), "nodesc\n");
      execGit(["add", "nodesc.txt"], { cwd: workDir });
      execGit(["commit", "-m", "Subject only"], { cwd: workDir });
      execGit(["push", "-u", "origin", "no-body-branch"], { cwd: workDir });

      global.exec = makeRealExec(workDir);
      const githubClient = makeMockGithubClient();

      await pushSignedCommits({
        githubClient,
        owner: "test-owner",
        repo: "test-repo",
        branch: "no-body-branch",
        baseRef: "origin/main",
        cwd: workDir,
      });

      const callArg = githubClient.graphql.mock.calls[0][1].input;
      expect(callArg.message.headline).toBe("Subject only");
      expect(callArg.message.body).toBeUndefined();
    });
  });
});
