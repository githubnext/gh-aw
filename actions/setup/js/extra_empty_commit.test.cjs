import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

describe("extra_empty_commit.cjs", () => {
  let mockCore;
  let mockExec;
  let pushExtraEmptyCommit;
  let originalEnv;
  let originalGithubRepo;
  let originalGithubServerUrl;

  beforeEach(() => {
    originalEnv = process.env.GH_AW_CI_TRIGGER_TOKEN;
    originalGithubRepo = process.env.GITHUB_REPOSITORY;
    originalGithubServerUrl = process.env.GITHUB_SERVER_URL;
    // Set GITHUB_REPOSITORY to match the default test owner/repo so the
    // cross-repo guard doesn't interfere with unrelated tests.
    process.env.GITHUB_REPOSITORY = "test-owner/test-repo";

    mockCore = {
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      setFailed: vi.fn(),
    };

    // Default exec mock: resolves successfully, no stdout output
    mockExec = {
      exec: vi.fn().mockResolvedValue(0),
    };

    global.core = mockCore;
    global.exec = mockExec;

    // Clear module cache so env changes take effect
    delete require.cache[require.resolve("./extra_empty_commit.cjs")];
  });

  afterEach(() => {
    if (originalEnv !== undefined) {
      process.env.GH_AW_CI_TRIGGER_TOKEN = originalEnv;
    } else {
      delete process.env.GH_AW_CI_TRIGGER_TOKEN;
    }
    if (originalGithubRepo !== undefined) {
      process.env.GITHUB_REPOSITORY = originalGithubRepo;
    } else {
      delete process.env.GITHUB_REPOSITORY;
    }
    if (originalGithubServerUrl !== undefined) {
      process.env.GITHUB_SERVER_URL = originalGithubServerUrl;
    } else {
      delete process.env.GITHUB_SERVER_URL;
    }
    delete global.core;
    delete global.exec;
    vi.clearAllMocks();
  });

  /**
   * Helper to configure the exec mock so that `git log` calls invoke the
   * stdout listener with the supplied output string, while all other
   * exec calls resolve normally.
   */
  function mockGitLogOutput(logOutput) {
    mockExec.exec.mockImplementation(async (cmd, args, options) => {
      if (cmd === "git" && args && args[0] === "log" && options && options.listeners && options.listeners.stdout) {
        options.listeners.stdout(Buffer.from(logOutput));
      }
      return 0;
    });
  }

  // ──────────────────────────────────────────────────────
  // Token presence
  // ──────────────────────────────────────────────────────

  describe("when no extra empty commit token is set", () => {
    it("should skip and return success with skipped=true", async () => {
      delete process.env.GH_AW_CI_TRIGGER_TOKEN;
      ({ pushExtraEmptyCommit } = require("./extra_empty_commit.cjs"));

      const result = await pushExtraEmptyCommit({
        branchName: "feature-branch",
        repoOwner: "test-owner",
        repoName: "test-repo",
      });

      expect(result).toEqual({ success: true, skipped: true });
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No extra empty commit token"));
      expect(mockExec.exec).not.toHaveBeenCalled();
    });

    it("should skip when token is empty string", async () => {
      process.env.GH_AW_CI_TRIGGER_TOKEN = "";
      ({ pushExtraEmptyCommit } = require("./extra_empty_commit.cjs"));

      const result = await pushExtraEmptyCommit({
        branchName: "feature-branch",
        repoOwner: "test-owner",
        repoName: "test-repo",
      });

      expect(result).toEqual({ success: true, skipped: true });
    });

    it("should skip when token is whitespace only", async () => {
      process.env.GH_AW_CI_TRIGGER_TOKEN = "   ";
      ({ pushExtraEmptyCommit } = require("./extra_empty_commit.cjs"));

      const result = await pushExtraEmptyCommit({
        branchName: "feature-branch",
        repoOwner: "test-owner",
        repoName: "test-repo",
      });

      expect(result).toEqual({ success: true, skipped: true });
    });
  });

  // ──────────────────────────────────────────────────────
  // Successful push (no cycle issues)
  // ──────────────────────────────────────────────────────

  describe("when token is set and no cycle issues", () => {
    beforeEach(() => {
      process.env.GH_AW_CI_TRIGGER_TOKEN = "ghp_test_token_123";
      // Simulate git log showing 5 commits, all with file changes (non-empty)
      const logOutput = ["COMMIT:aaa111", "file1.txt", "", "COMMIT:bbb222", "file2.txt", "file3.txt", "", "COMMIT:ccc333", "file4.txt", "", "COMMIT:ddd444", "file5.txt", "", "COMMIT:eee555", "file6.txt", ""].join("\n");
      mockGitLogOutput(logOutput);
      ({ pushExtraEmptyCommit } = require("./extra_empty_commit.cjs"));
    });

    it("should push an empty commit and return success", async () => {
      const result = await pushExtraEmptyCommit({
        branchName: "feature-branch",
        repoOwner: "test-owner",
        repoName: "test-repo",
      });

      expect(result).toEqual({ success: true });
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Extra empty commit token detected"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Cycle check passed"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Extra empty commit pushed"));
    });

    it("should add and remove a ci-trigger remote", async () => {
      await pushExtraEmptyCommit({
        branchName: "feature-branch",
        repoOwner: "test-owner",
        repoName: "test-repo",
      });

      const execCalls = mockExec.exec.mock.calls;
      // Find remote add call
      const addRemote = execCalls.find(c => c[0] === "git" && c[1] && c[1][0] === "remote" && c[1][1] === "add");
      expect(addRemote).toBeDefined();
      expect(addRemote[1]).toEqual(["remote", "add", "ci-trigger", expect.stringContaining("x-access-token:ghp_test_token_123")]);

      // Find remote remove cleanup call (after push)
      const removeRemoteCalls = execCalls.filter(c => c[0] === "git" && c[1] && c[1][0] === "remote" && c[1][1] === "remove");
      expect(removeRemoteCalls.length).toBeGreaterThanOrEqual(1);
    });

    it("should fetch and reset to remote branch before committing", async () => {
      await pushExtraEmptyCommit({
        branchName: "api-created-branch",
        repoOwner: "test-owner",
        repoName: "test-repo",
      });

      const execCalls = mockExec.exec.mock.calls;

      // Find the remote add call index so we can verify order
      const addRemoteIdx = execCalls.findIndex(c => c[0] === "git" && c[1] && c[1][0] === "remote" && c[1][1] === "add");
      expect(addRemoteIdx).toBeGreaterThanOrEqual(0);

      // fetch should come after remote add
      const fetchCall = execCalls.find(c => c[0] === "git" && c[1] && c[1][0] === "fetch" && c[1][1] === "ci-trigger");
      expect(fetchCall).toBeDefined();
      expect(fetchCall[1]).toEqual(["fetch", "ci-trigger", "api-created-branch"]);
      const fetchIdx = execCalls.indexOf(fetchCall);
      expect(fetchIdx).toBeGreaterThan(addRemoteIdx);

      // reset --hard should come after fetch
      const resetCall = execCalls.find(c => c[0] === "git" && c[1] && c[1][0] === "reset" && c[1][1] === "--hard");
      expect(resetCall).toBeDefined();
      expect(resetCall[1]).toEqual(["reset", "--hard", "ci-trigger/api-created-branch"]);
      const resetIdx = execCalls.indexOf(resetCall);
      expect(resetIdx).toBeGreaterThan(fetchIdx);

      // commit should come after reset
      const commitCall = execCalls.find(c => c[0] === "git" && c[1] && c[1][0] === "commit");
      expect(commitCall).toBeDefined();
      const commitIdx = execCalls.indexOf(commitCall);
      expect(commitIdx).toBeGreaterThan(resetIdx);

      // push should come after commit
      const pushCall = execCalls.find(c => c[0] === "git" && c[1] && c[1][0] === "push");
      expect(pushCall).toBeDefined();
      const pushIdx = execCalls.indexOf(pushCall);
      expect(pushIdx).toBeGreaterThan(commitIdx);
    });

    it("should succeed even when fetch/reset fails (branch not yet on remote)", async () => {
      mockExec.exec.mockImplementation(async (cmd, args, options) => {
        if (cmd === "git" && args && args[0] === "log" && options && options.listeners) {
          options.listeners.stdout(Buffer.from("COMMIT:abc123\nfile.txt\n"));
          return 0;
        }
        // Simulate fetch failing (branch does not yet exist on remote)
        if (cmd === "git" && args && args[0] === "fetch") {
          throw new Error("couldn't find remote ref api-created-branch");
        }
        return 0;
      });

      const result = await pushExtraEmptyCommit({
        branchName: "api-created-branch",
        repoOwner: "test-owner",
        repoName: "test-repo",
      });

      // Push should still be attempted and succeed (mock returns 0 for push)
      expect(result).toEqual({ success: true });
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Could not sync local branch"));

      // commit and push should still have been called
      const commitCall = mockExec.exec.mock.calls.find(c => c[0] === "git" && c[1] && c[1][0] === "commit");
      expect(commitCall).toBeDefined();
      const pushCall = mockExec.exec.mock.calls.find(c => c[0] === "git" && c[1] && c[1][0] === "push");
      expect(pushCall).toBeDefined();
    });

    it("should use github.com by default when GITHUB_SERVER_URL is not set", async () => {
      delete process.env.GITHUB_SERVER_URL;
      delete require.cache[require.resolve("./extra_empty_commit.cjs")];
      ({ pushExtraEmptyCommit } = require("./extra_empty_commit.cjs"));

      await pushExtraEmptyCommit({
        branchName: "feature-branch",
        repoOwner: "test-owner",
        repoName: "test-repo",
      });

      const addRemote = mockExec.exec.mock.calls.find(c => c[0] === "git" && c[1] && c[1][0] === "remote" && c[1][1] === "add");
      expect(addRemote).toBeDefined();
      expect(addRemote[1][3]).toContain("github.com/test-owner/test-repo.git");
    });

    it("should use GITHUB_SERVER_URL hostname for GitHub Enterprise", async () => {
      process.env.GITHUB_SERVER_URL = "https://github.example.com";
      delete require.cache[require.resolve("./extra_empty_commit.cjs")];
      ({ pushExtraEmptyCommit } = require("./extra_empty_commit.cjs"));

      await pushExtraEmptyCommit({
        branchName: "feature-branch",
        repoOwner: "test-owner",
        repoName: "test-repo",
      });

      const addRemote = mockExec.exec.mock.calls.find(c => c[0] === "git" && c[1] && c[1][0] === "remote" && c[1][1] === "add");
      expect(addRemote).toBeDefined();
      expect(addRemote[1][3]).toContain("github.example.com/test-owner/test-repo.git");
      expect(addRemote[1][3]).not.toContain("github.com/test-owner/test-repo.git");
    });

    it("should use default commit message when none provided", async () => {
      await pushExtraEmptyCommit({
        branchName: "feature-branch",
        repoOwner: "test-owner",
        repoName: "test-repo",
      });

      const commitCall = mockExec.exec.mock.calls.find(c => c[0] === "git" && c[1] && c[1][0] === "commit");
      expect(commitCall).toBeDefined();
      expect(commitCall[1]).toEqual(["commit", "--allow-empty", "-m", "ci: trigger checks"]);
    });

    it("should use custom commit message when provided", async () => {
      await pushExtraEmptyCommit({
        branchName: "feature-branch",
        repoOwner: "test-owner",
        repoName: "test-repo",
        commitMessage: "chore: custom CI trigger",
      });

      const commitCall = mockExec.exec.mock.calls.find(c => c[0] === "git" && c[1] && c[1][0] === "commit");
      expect(commitCall[1]).toEqual(["commit", "--allow-empty", "-m", "chore: custom CI trigger"]);
    });

    it("should push to the correct branch", async () => {
      await pushExtraEmptyCommit({
        branchName: "my-feature",
        repoOwner: "test-owner",
        repoName: "test-repo",
      });

      const pushCall = mockExec.exec.mock.calls.find(c => c[0] === "git" && c[1] && c[1][0] === "push");
      expect(pushCall).toBeDefined();
      expect(pushCall[1]).toEqual(["push", "ci-trigger", "my-feature"]);
    });
  });

  // ──────────────────────────────────────────────────────
  // Cycle prevention
  // ──────────────────────────────────────────────────────

  describe("cycle prevention", () => {
    beforeEach(() => {
      process.env.GH_AW_CI_TRIGGER_TOKEN = "ghp_test_token_123";
      ({ pushExtraEmptyCommit } = require("./extra_empty_commit.cjs"));
    });

    it("should skip when 30 or more empty commits found in last 60", async () => {
      // Build git log output with 30 empty commits (hash only, no files)
      const commits = [];
      for (let i = 0; i < 30; i++) {
        commits.push(`COMMIT:empty${i.toString().padStart(3, "0")}`);
        commits.push(""); // blank line = no files
      }
      // Add some non-empty commits too
      for (let i = 0; i < 10; i++) {
        commits.push(`COMMIT:real${i.toString().padStart(3, "0")}`);
        commits.push(`file${i}.txt`);
        commits.push("");
      }
      mockGitLogOutput(commits.join("\n"));

      const result = await pushExtraEmptyCommit({
        branchName: "feature-branch",
        repoOwner: "test-owner",
        repoName: "test-repo",
      });

      expect(result).toEqual({ success: true, skipped: true });
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Cycle prevention"));
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("30 empty commits"));

      // Should NOT have pushed (no commit or push calls after the log check)
      const commitCalls = mockExec.exec.mock.calls.filter(c => c[0] === "git" && c[1] && c[1][0] === "commit");
      expect(commitCalls).toHaveLength(0);
    });

    it("should allow push when fewer than 30 empty commits", async () => {
      // 29 empty commits - just under the limit
      const commits = [];
      for (let i = 0; i < 29; i++) {
        commits.push(`COMMIT:empty${i.toString().padStart(3, "0")}`);
        commits.push("");
      }
      for (let i = 0; i < 5; i++) {
        commits.push(`COMMIT:real${i.toString().padStart(3, "0")}`);
        commits.push(`file${i}.txt`);
        commits.push("");
      }
      mockGitLogOutput(commits.join("\n"));

      const result = await pushExtraEmptyCommit({
        branchName: "feature-branch",
        repoOwner: "test-owner",
        repoName: "test-repo",
      });

      expect(result).toEqual({ success: true });
      expect(mockCore.warning).not.toHaveBeenCalled();
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Cycle check passed: 29 empty commit"));
    });

    it("should allow push when no commits exist (empty repo)", async () => {
      mockGitLogOutput("");

      const result = await pushExtraEmptyCommit({
        branchName: "feature-branch",
        repoOwner: "test-owner",
        repoName: "test-repo",
      });

      expect(result).toEqual({ success: true });
    });

    it("should allow push when all commits have file changes", async () => {
      const commits = [];
      for (let i = 0; i < 50; i++) {
        commits.push(`COMMIT:hash${i.toString().padStart(3, "0")}`);
        commits.push(`src/file${i}.go`);
        commits.push("");
      }
      mockGitLogOutput(commits.join("\n"));

      const result = await pushExtraEmptyCommit({
        branchName: "feature-branch",
        repoOwner: "test-owner",
        repoName: "test-repo",
      });

      expect(result).toEqual({ success: true });
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Cycle check passed: 0 empty commit"));
    });

    it("should allow push if git log fails (defaults to 0 empty commits)", async () => {
      // Make git log throw an error
      mockExec.exec.mockImplementation(async (cmd, args, options) => {
        if (cmd === "git" && args && args[0] === "log") {
          throw new Error("git log failed");
        }
        return 0;
      });

      const result = await pushExtraEmptyCommit({
        branchName: "feature-branch",
        repoOwner: "test-owner",
        repoName: "test-repo",
      });

      expect(result).toEqual({ success: true });
    });

    it("should skip at exactly 30 empty commits (boundary)", async () => {
      const commits = [];
      // Exactly 30 empty commits
      for (let i = 0; i < 30; i++) {
        commits.push(`COMMIT:empty${i.toString().padStart(3, "0")}`);
        commits.push("");
      }
      // 30 non-empty commits to fill up to 60
      for (let i = 0; i < 30; i++) {
        commits.push(`COMMIT:real${i.toString().padStart(3, "0")}`);
        commits.push(`file${i}.txt`);
        commits.push("");
      }
      mockGitLogOutput(commits.join("\n"));

      const result = await pushExtraEmptyCommit({
        branchName: "feature-branch",
        repoOwner: "test-owner",
        repoName: "test-repo",
      });

      expect(result).toEqual({ success: true, skipped: true });
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Cycle prevention"));
    });
  });

  // ──────────────────────────────────────────────────────
  // newCommitCount restriction
  // ──────────────────────────────────────────────────────

  describe("newCommitCount restriction", () => {
    beforeEach(() => {
      process.env.GH_AW_CI_TRIGGER_TOKEN = "ghp_test_token_123";
      // No empty commits in log (so cycle prevention doesn't interfere)
      mockGitLogOutput("COMMIT:abc123\nfile.txt\n");
      ({ pushExtraEmptyCommit } = require("./extra_empty_commit.cjs"));
    });

    it("should proceed when newCommitCount is exactly 1", async () => {
      const result = await pushExtraEmptyCommit({
        branchName: "feature-branch",
        repoOwner: "test-owner",
        repoName: "test-repo",
        newCommitCount: 1,
      });

      expect(result).toEqual({ success: true });
      // Should have committed and pushed
      const commitCall = mockExec.exec.mock.calls.find(c => c[0] === "git" && c[1] && c[1][0] === "commit");
      expect(commitCall).toBeDefined();
    });

    it("should skip when newCommitCount is 0", async () => {
      const result = await pushExtraEmptyCommit({
        branchName: "feature-branch",
        repoOwner: "test-owner",
        repoName: "test-repo",
        newCommitCount: 0,
      });

      expect(result).toEqual({ success: true, skipped: true });
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("0 new commit(s)"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("only triggers for exactly 1 commit"));
      // Should NOT have committed
      const commitCall = mockExec.exec.mock.calls.find(c => c[0] === "git" && c[1] && c[1][0] === "commit");
      expect(commitCall).toBeUndefined();
    });

    it("should skip when newCommitCount is 2", async () => {
      const result = await pushExtraEmptyCommit({
        branchName: "feature-branch",
        repoOwner: "test-owner",
        repoName: "test-repo",
        newCommitCount: 2,
      });

      expect(result).toEqual({ success: true, skipped: true });
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("2 new commit(s)"));
      // Should NOT have committed
      const commitCall = mockExec.exec.mock.calls.find(c => c[0] === "git" && c[1] && c[1][0] === "commit");
      expect(commitCall).toBeUndefined();
    });

    it("should proceed when newCommitCount is not provided (backward compatibility)", async () => {
      const result = await pushExtraEmptyCommit({
        branchName: "feature-branch",
        repoOwner: "test-owner",
        repoName: "test-repo",
        // newCommitCount omitted
      });

      expect(result).toEqual({ success: true });
      // Should have committed (no restriction when parameter is absent)
      const commitCall = mockExec.exec.mock.calls.find(c => c[0] === "git" && c[1] && c[1][0] === "commit");
      expect(commitCall).toBeDefined();
    });
  });

  // ──────────────────────────────────────────────────────
  // Cross-repo guard
  // ──────────────────────────────────────────────────────

  describe("cross-repo guard", () => {
    let originalGithubRepo;

    beforeEach(() => {
      originalGithubRepo = process.env.GITHUB_REPOSITORY;
      process.env.GH_AW_CI_TRIGGER_TOKEN = "ghp_test_token_123";
      // No empty commits in log (so cycle prevention doesn't interfere)
      mockGitLogOutput("COMMIT:abc123\nfile.txt\n");
    });

    afterEach(() => {
      if (originalGithubRepo !== undefined) {
        process.env.GITHUB_REPOSITORY = originalGithubRepo;
      } else {
        delete process.env.GITHUB_REPOSITORY;
      }
    });

    it("should skip when target repo differs from GITHUB_REPOSITORY", async () => {
      process.env.GITHUB_REPOSITORY = "my-org/my-repo";
      delete require.cache[require.resolve("./extra_empty_commit.cjs")];
      ({ pushExtraEmptyCommit } = require("./extra_empty_commit.cjs"));

      const result = await pushExtraEmptyCommit({
        branchName: "feature-branch",
        repoOwner: "other-org",
        repoName: "other-repo",
      });

      expect(result).toEqual({ success: true, skipped: true });
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("cross-repo target"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("other-org/other-repo"));
      // Should NOT have committed or pushed
      const commitCall = mockExec.exec.mock.calls.find(c => c[0] === "git" && c[1] && c[1][0] === "commit");
      expect(commitCall).toBeUndefined();
    });

    it("should skip when only owner differs (cross-org)", async () => {
      process.env.GITHUB_REPOSITORY = "my-org/shared-repo";
      delete require.cache[require.resolve("./extra_empty_commit.cjs")];
      ({ pushExtraEmptyCommit } = require("./extra_empty_commit.cjs"));

      const result = await pushExtraEmptyCommit({
        branchName: "feature-branch",
        repoOwner: "other-org",
        repoName: "shared-repo",
      });

      expect(result).toEqual({ success: true, skipped: true });
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("cross-repo target"));
    });

    it("should proceed when target repo matches GITHUB_REPOSITORY", async () => {
      process.env.GITHUB_REPOSITORY = "my-org/my-repo";
      delete require.cache[require.resolve("./extra_empty_commit.cjs")];
      ({ pushExtraEmptyCommit } = require("./extra_empty_commit.cjs"));

      const result = await pushExtraEmptyCommit({
        branchName: "feature-branch",
        repoOwner: "my-org",
        repoName: "my-repo",
      });

      expect(result).toEqual({ success: true });
      // Should have committed and pushed
      const commitCall = mockExec.exec.mock.calls.find(c => c[0] === "git" && c[1] && c[1][0] === "commit");
      expect(commitCall).toBeDefined();
    });

    it("should compare repos case-insensitively", async () => {
      process.env.GITHUB_REPOSITORY = "My-Org/My-Repo";
      delete require.cache[require.resolve("./extra_empty_commit.cjs")];
      ({ pushExtraEmptyCommit } = require("./extra_empty_commit.cjs"));

      const result = await pushExtraEmptyCommit({
        branchName: "feature-branch",
        repoOwner: "my-org",
        repoName: "my-repo",
      });

      expect(result).toEqual({ success: true });
      // Should have committed and pushed (same repo, different case)
      const commitCall = mockExec.exec.mock.calls.find(c => c[0] === "git" && c[1] && c[1][0] === "commit");
      expect(commitCall).toBeDefined();
    });

    it("should proceed when GITHUB_REPOSITORY is not set", async () => {
      delete process.env.GITHUB_REPOSITORY;
      delete require.cache[require.resolve("./extra_empty_commit.cjs")];
      ({ pushExtraEmptyCommit } = require("./extra_empty_commit.cjs"));

      const result = await pushExtraEmptyCommit({
        branchName: "feature-branch",
        repoOwner: "test-owner",
        repoName: "test-repo",
      });

      expect(result).toEqual({ success: true });
    });
  });

  // ──────────────────────────────────────────────────────
  // Error handling
  // ──────────────────────────────────────────────────────

  describe("error handling", () => {
    beforeEach(() => {
      process.env.GH_AW_CI_TRIGGER_TOKEN = "ghp_test_token_123";
      // No empty commits in log
      mockGitLogOutput("COMMIT:abc123\nfile.txt\n");
      ({ pushExtraEmptyCommit } = require("./extra_empty_commit.cjs"));
    });

    it("should return error result when push fails", async () => {
      let callCount = 0;
      mockExec.exec.mockImplementation(async (cmd, args, options) => {
        // Let git log succeed with stdout listener
        if (cmd === "git" && args && args[0] === "log" && options && options.listeners) {
          options.listeners.stdout(Buffer.from("COMMIT:abc123\nfile.txt\n"));
          return 0;
        }
        // Let remote operations and commit succeed, but fail on push
        if (cmd === "git" && args && args[0] === "push") {
          throw new Error("authentication failed");
        }
        return 0;
      });

      const result = await pushExtraEmptyCommit({
        branchName: "feature-branch",
        repoOwner: "test-owner",
        repoName: "test-repo",
      });

      expect(result.success).toBe(false);
      expect(result.error).toContain("authentication failed");
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Failed to push extra empty commit"));
    });

    it("should clean up remote even when push fails", async () => {
      const remoteRemoveCalls = [];
      mockExec.exec.mockImplementation(async (cmd, args, options) => {
        if (cmd === "git" && args && args[0] === "log" && options && options.listeners) {
          options.listeners.stdout(Buffer.from("COMMIT:abc123\nfile.txt\n"));
          return 0;
        }
        if (cmd === "git" && args && args[0] === "remote" && args[1] === "remove") {
          remoteRemoveCalls.push(args);
          return 0;
        }
        if (cmd === "git" && args && args[0] === "push") {
          throw new Error("push failed");
        }
        return 0;
      });

      await pushExtraEmptyCommit({
        branchName: "feature-branch",
        repoOwner: "test-owner",
        repoName: "test-repo",
      });

      // Should have at least one remote remove call for cleanup
      const ciTriggerRemoveCall = remoteRemoveCalls.find(args => args[2] === "ci-trigger");
      expect(ciTriggerRemoveCall).toBeDefined();
    });
  });
});

describe("isCrossRepoTarget", () => {
  let originalGithubRepo;
  let isCrossRepoTarget;

  beforeEach(() => {
    originalGithubRepo = process.env.GITHUB_REPOSITORY;
    // Provide minimal globals so the module loads
    global.core = { info: vi.fn(), warning: vi.fn(), error: vi.fn(), setFailed: vi.fn() };
    global.exec = { exec: vi.fn().mockResolvedValue(0) };
    delete require.cache[require.resolve("./extra_empty_commit.cjs")];
    ({ isCrossRepoTarget } = require("./extra_empty_commit.cjs"));
  });

  afterEach(() => {
    if (originalGithubRepo !== undefined) {
      process.env.GITHUB_REPOSITORY = originalGithubRepo;
    } else {
      delete process.env.GITHUB_REPOSITORY;
    }
    delete global.core;
    delete global.exec;
  });

  it("returns true when target repo differs from GITHUB_REPOSITORY", () => {
    process.env.GITHUB_REPOSITORY = "my-org/my-repo";
    expect(isCrossRepoTarget("other-org", "other-repo")).toBe(true);
  });

  it("returns true when only owner differs", () => {
    process.env.GITHUB_REPOSITORY = "my-org/shared-repo";
    expect(isCrossRepoTarget("other-org", "shared-repo")).toBe(true);
  });

  it("returns false when target repo matches GITHUB_REPOSITORY", () => {
    process.env.GITHUB_REPOSITORY = "my-org/my-repo";
    expect(isCrossRepoTarget("my-org", "my-repo")).toBe(false);
  });

  it("compares case-insensitively", () => {
    process.env.GITHUB_REPOSITORY = "My-Org/My-Repo";
    expect(isCrossRepoTarget("my-org", "my-repo")).toBe(false);
  });

  it("returns false when GITHUB_REPOSITORY is not set", () => {
    delete process.env.GITHUB_REPOSITORY;
    expect(isCrossRepoTarget("any-org", "any-repo")).toBe(false);
  });

  it("returns false when GITHUB_REPOSITORY is empty string", () => {
    process.env.GITHUB_REPOSITORY = "";
    expect(isCrossRepoTarget("any-org", "any-repo")).toBe(false);
  });
});
