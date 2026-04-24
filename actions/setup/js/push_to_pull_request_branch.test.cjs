import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import * as fs from "fs";
import * as path from "path";
import * as os from "os";

/**
 * Test Matrix for push_to_pull_request_branch.cjs
 *
 * This test file covers the following scenarios:
 *
 * 1. GitHub Actions Context Scenarios:
 *    - pull_request event (direct PR trigger)
 *    - issue_comment event (comment on PR)
 *    - schedule event (scheduled task)
 *    - workflow_dispatch (manual trigger)
 *
 * 2. Repository State Scenarios:
 *    - PR branch has had new pushes (needs merge/rebase)
 *    - Base branch (main/default) has new commits
 *    - PR branch has been force-pushed
 *    - Normal push scenario
 *
 * 3. Empty Commit Handling:
 *    - if-no-changes: warn (default)
 *    - if-no-changes: error
 *    - if-no-changes: ignore
 *
 * 4. Fork PR Scenarios:
 *    - Fork PR detection
 *    - Early failure for fork PRs (no push access)
 *    - Same-repo PR (normal case)
 *
 * 5. Error Scenarios:
 *    - git fetch failure
 *    - git checkout failure
 *    - git am (patch apply) failure
 *    - git push failure
 *    - Patch size too large
 *    - Patch file not found
 *    - Invalid patch content
 */

describe("push_to_pull_request_branch.cjs", () => {
  let mockCore;
  let mockExec;
  let mockContext;
  let mockGithub;
  let tempDir;
  let originalEnv;

  beforeEach(async () => {
    originalEnv = { ...process.env };

    // Set GITHUB_REPOSITORY to match the default test owner/repo so the
    // cross-repo guard in extra_empty_commit doesn't interfere.
    process.env.GITHUB_REPOSITORY = "test-owner/test-repo";

    // Create temp directory for test artifacts
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "push-to-pr-test-"));

    // Mock core actions methods
    mockCore = {
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      debug: vi.fn(),
      setFailed: vi.fn(),
      setOutput: vi.fn(),
      startGroup: vi.fn(),
      endGroup: vi.fn(),
      summary: {
        addRaw: vi.fn().mockReturnThis(),
        write: vi.fn().mockResolvedValue(undefined),
      },
    };

    // Default exec mock
    mockExec = {
      exec: vi.fn().mockResolvedValue(0),
      getExecOutput: vi.fn().mockResolvedValue({ exitCode: 0, stdout: "", stderr: "" }),
    };

    // Default pull request context
    mockContext = {
      eventName: "pull_request",
      sha: "abc123def456",
      repo: {
        owner: "test-owner",
        repo: "test-repo",
      },
      payload: {
        pull_request: {
          number: 123,
          state: "open",
          title: "Test PR",
          labels: [],
          head: {
            ref: "feature-branch",
            sha: "head-sha-123",
            repo: {
              full_name: "test-owner/test-repo",
              fork: false,
              owner: {
                login: "test-owner",
              },
            },
          },
          base: {
            ref: "main",
            sha: "base-sha-456",
            repo: {
              full_name: "test-owner/test-repo",
              owner: {
                login: "test-owner",
              },
            },
          },
        },
        repository: {
          html_url: "https://github.com/test-owner/test-repo",
        },
      },
    };

    // Default GitHub API mock
    mockGithub = {
      rest: {
        pulls: {
          get: vi.fn().mockResolvedValue({
            data: {
              head: {
                ref: "feature-branch",
                repo: {
                  full_name: "test-owner/test-repo",
                  fork: false,
                },
              },
              base: {
                repo: {
                  full_name: "test-owner/test-repo",
                },
              },
              title: "Test PR",
              labels: [],
            },
          }),
          create: vi.fn().mockResolvedValue({
            data: {
              number: 999,
              html_url: "https://github.com/test-owner/test-repo/pull/999",
            },
          }),
        },
        repos: {
          get: vi.fn().mockResolvedValue({
            data: { default_branch: "main" },
          }),
          getBranchProtection: vi.fn().mockRejectedValue(Object.assign(new Error("Branch not protected"), { status: 404 })),
        },
      },
      graphql: vi.fn(),
    };

    global.core = mockCore;
    global.exec = mockExec;
    global.context = mockContext;
    global.github = mockGithub;

    // Clear module cache
    delete require.cache[require.resolve("./push_to_pull_request_branch.cjs")];
    delete require.cache[require.resolve("./staged_preview.cjs")];
    delete require.cache[require.resolve("./update_activation_comment.cjs")];
    delete require.cache[require.resolve("./extra_empty_commit.cjs")];
  });

  afterEach(() => {
    // Restore environment by mutating process.env in place
    // (replacing process.env with a plain object breaks Node's special env handling)
    for (const key of Object.keys(process.env)) {
      if (!(key in originalEnv)) {
        delete process.env[key];
      }
    }
    Object.assign(process.env, originalEnv);

    // Clean up temp directory
    if (tempDir && fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }

    // Clean up globals
    delete global.core;
    delete global.exec;
    delete global.context;
    delete global.github;
    vi.clearAllMocks();
  });

  /**
   * Helper to load the module with mocked dependencies
   */
  async function loadModule() {
    // Module loads when imported
    const module = require("./push_to_pull_request_branch.cjs");
    return module;
  }

  /**
   * Helper to create a valid patch file
   */
  function createPatchFile(content = null) {
    const patchPath = path.join(tempDir, "test.patch");
    const defaultPatch = `From abc123 Mon Sep 17 00:00:00 2001
From: Test Author <test@example.com>
Date: Mon, 1 Jan 2024 00:00:00 +0000
Subject: [PATCH] Test commit

Test changes

diff --git a/test.txt b/test.txt
new file mode 100644
index 0000000..abc1234
--- /dev/null
+++ b/test.txt
@@ -0,0 +1 @@
+Hello World
--
2.34.1
`;
    fs.writeFileSync(patchPath, content !== null ? content : defaultPatch);
    return patchPath;
  }

  // ──────────────────────────────────────────────────────
  // Configuration Parsing Tests
  // ──────────────────────────────────────────────────────

  describe("configuration parsing", () => {
    it('should default target to "triggering" when not specified', async () => {
      const module = await loadModule();
      const handler = await module.main({});

      expect(mockCore.info).toHaveBeenCalledWith("Target: triggering");
    });

    it("should accept target from config", async () => {
      const module = await loadModule();
      const handler = await module.main({ target: "*" });

      expect(mockCore.info).toHaveBeenCalledWith("Target: *");
    });

    it("should accept numeric target for specific PR", async () => {
      const module = await loadModule();
      const handler = await module.main({ target: "456" });

      expect(mockCore.info).toHaveBeenCalledWith("Target: 456");
    });

    it('should default if_no_changes to "warn"', async () => {
      const module = await loadModule();
      const handler = await module.main({});

      expect(mockCore.info).toHaveBeenCalledWith("If no changes: warn");
    });

    it("should accept title_prefix configuration", async () => {
      const module = await loadModule();
      const handler = await module.main({ title_prefix: "[bot] " });

      expect(mockCore.info).toHaveBeenCalledWith("Title prefix: [bot] ");
    });

    it("should accept labels configuration as array", async () => {
      const module = await loadModule();
      const handler = await module.main({ labels: ["automated", "bot"] });

      expect(mockCore.info).toHaveBeenCalledWith("Required labels: automated, bot");
    });

    it("should accept labels configuration as comma-separated string", async () => {
      const module = await loadModule();
      const handler = await module.main({ labels: "automated,bot" });

      expect(mockCore.info).toHaveBeenCalledWith("Required labels: automated, bot");
    });

    it("should default max_patch_size to 1024 KB", async () => {
      const module = await loadModule();
      const handler = await module.main({});

      expect(mockCore.info).toHaveBeenCalledWith("Max patch size: 1024 KB");
    });
  });

  // ──────────────────────────────────────────────────────
  // GitHub Actions Context Tests
  // ──────────────────────────────────────────────────────

  describe("GitHub Actions context scenarios", () => {
    it('should handle pull_request event with target "triggering"', async () => {
      mockContext.eventName = "pull_request";
      const patchPath = createPatchFile();

      // Mock git commands
      mockExec.getExecOutput.mockResolvedValue({ exitCode: 0, stdout: "abc123\n", stderr: "" });

      const module = await loadModule();
      const handler = await module.main({ target: "triggering" });
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(true);
      expect(mockGithub.rest.pulls.get).toHaveBeenCalledWith({
        owner: "test-owner",
        repo: "test-repo",
        pull_number: 123,
      });
    });

    it("should handle issue_comment event (comment on PR)", async () => {
      mockContext.eventName = "issue_comment";
      mockContext.payload.issue = { number: 123 };
      const patchPath = createPatchFile();

      mockExec.getExecOutput.mockResolvedValue({ exitCode: 0, stdout: "abc123\n", stderr: "" });

      const module = await loadModule();
      const handler = await module.main({ target: "triggering" });
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(true);
    });

    it('should fail for schedule event with target "triggering"', async () => {
      mockContext.eventName = "schedule";
      delete mockContext.payload.pull_request;
      delete mockContext.payload.issue;

      const patchPath = createPatchFile();

      const module = await loadModule();
      const handler = await module.main({ target: "triggering" });
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("pull request context");
    });

    it('should fail gracefully with target "triggering" when context is undefined (MCP daemon mode)', async () => {
      delete global.context;

      const patchPath = createPatchFile();

      const module = await loadModule();
      const handler = await module.main({ target: "triggering" });
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("pull request context");
    });

    it('should handle schedule event with target "*" and explicit PR number', async () => {
      mockContext.eventName = "schedule";
      delete mockContext.payload.pull_request;
      const patchPath = createPatchFile();

      mockExec.getExecOutput.mockResolvedValue({ exitCode: 0, stdout: "abc123\n", stderr: "" });

      const module = await loadModule();
      const handler = await module.main({ target: "*" });
      const result = await handler({ patch_path: patchPath, pull_request_number: 456 }, {});

      expect(result.success).toBe(true);
      expect(mockGithub.rest.pulls.get).toHaveBeenCalledWith({
        owner: "test-owner",
        repo: "test-repo",
        pull_number: 456,
      });
    });

    it("should handle workflow_dispatch event with explicit PR number", async () => {
      mockContext.eventName = "workflow_dispatch";
      delete mockContext.payload.pull_request;
      const patchPath = createPatchFile();

      mockExec.getExecOutput.mockResolvedValue({ exitCode: 0, stdout: "abc123\n", stderr: "" });

      const module = await loadModule();
      const handler = await module.main({ target: "789" });
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(true);
      expect(mockGithub.rest.pulls.get).toHaveBeenCalledWith({
        owner: "test-owner",
        repo: "test-repo",
        pull_number: 789,
      });
    });
  });

  // ──────────────────────────────────────────────────────
  // Fork PR Detection Tests
  // ──────────────────────────────────────────────────────

  describe("fork PR detection and handling", () => {
    it("should detect fork PR via different repository names and fail early", async () => {
      // Set up fork scenario - head repo is different from base repo
      mockContext.payload.pull_request.head.repo.full_name = "fork-owner/test-repo";
      mockContext.payload.pull_request.head.repo.owner.login = "fork-owner";

      mockGithub.rest.pulls.get.mockResolvedValue({
        data: {
          head: {
            ref: "feature-branch",
            repo: {
              full_name: "fork-owner/test-repo",
              fork: true,
            },
          },
          base: {
            repo: {
              full_name: "test-owner/test-repo",
            },
          },
          title: "Fork PR",
          labels: [],
        },
      });

      const patchPath = createPatchFile();

      const module = await loadModule();
      const handler = await module.main({ target: "triggering" });
      const result = await handler({ patch_path: patchPath }, {});

      // Fork PRs should fail early with a clear error
      expect(result.success).toBe(false);
      expect(result.error).toContain("fork");
      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Cannot push to fork PR"));
      // Should not attempt any git operations
      expect(mockExec.exec).not.toHaveBeenCalled();
    });

    it("should NOT treat same-repo PR as fork even when repo has fork flag", async () => {
      // A repository that is itself a fork of another repo has fork=true,
      // but a same-repo PR within it is NOT a cross-repo fork PR (#24208)
      mockContext.payload.pull_request.head.repo.fork = true;

      mockGithub.rest.pulls.get.mockResolvedValue({
        data: {
          head: {
            ref: "feature-branch",
            repo: {
              full_name: "test-owner/test-repo",
              fork: true,
            },
          },
          base: {
            repo: {
              full_name: "test-owner/test-repo",
            },
          },
          title: "Same-repo PR in forked repo",
          labels: [],
        },
      });

      const patchPath = createPatchFile();

      const module = await loadModule();
      const handler = await module.main({ target: "triggering" });
      const result = await handler({ patch_path: patchPath }, {});

      // Same full_name means same repo — push should proceed, not fail
      expect(result.success).toBe(true);
    });

    it("should handle deleted head repo (likely a fork) and fail early", async () => {
      delete mockContext.payload.pull_request.head.repo;

      mockGithub.rest.pulls.get.mockResolvedValue({
        data: {
          head: {
            ref: "feature-branch",
            repo: null, // Deleted fork
          },
          base: {
            repo: {
              full_name: "test-owner/test-repo",
            },
          },
          title: "Deleted Fork PR",
          labels: [],
        },
      });

      const patchPath = createPatchFile();

      const module = await loadModule();
      const handler = await module.main({ target: "triggering" });
      const result = await handler({ patch_path: patchPath }, {});

      // When head.repo is null, this is likely a deleted fork
      // The handler should give a clear error about the fork
      expect(result.success).toBe(false);
      expect(result.error).toContain("fork");
      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Cannot push to fork PR"));
    });
  });

  // ──────────────────────────────────────────────────────
  // Protected Branch Detection Tests
  // ──────────────────────────────────────────────────────

  describe("protected branch detection", () => {
    it("should block push to the repository default branch", async () => {
      // PR head branch is the repo default branch
      mockGithub.rest.pulls.get.mockResolvedValue({
        data: {
          head: {
            ref: "main",
            repo: { full_name: "test-owner/test-repo", fork: false },
          },
          base: { repo: { full_name: "test-owner/test-repo" } },
          title: "Test PR",
          labels: [],
        },
      });
      mockGithub.rest.repos.get.mockResolvedValue({ data: { default_branch: "main" } });

      const patchPath = createPatchFile();
      const module = await loadModule();
      const handler = await module.main({ target: "triggering" });
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("default branch");
      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("default branch"));
      // Should not attempt any git operations
      expect(mockExec.exec).not.toHaveBeenCalled();
    });

    it("should block push to a branch with protection rules", async () => {
      // Branch has protection rules - getBranchProtection resolves successfully
      mockGithub.rest.repos.getBranchProtection.mockResolvedValue({ data: {} });

      const patchPath = createPatchFile();
      const module = await loadModule();
      const handler = await module.main({ target: "triggering" });
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("protection rules");
      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("protection rules"));
      // Should not attempt any git operations
      expect(mockExec.exec).not.toHaveBeenCalled();
    });

    it("should allow push when branch is not default and has no protection rules", async () => {
      // Default mock: repos.get returns "main", getBranchProtection returns 404
      // PR head is "feature-branch" (not the default)
      const patchPath = createPatchFile();
      mockExec.getExecOutput.mockResolvedValue({ exitCode: 0, stdout: "abc123\n", stderr: "" });

      const module = await loadModule();
      const handler = await module.main({});
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(true);
      expect(result.branch_name).toBe("feature-branch");
    });

    it("should warn and continue when repos.get fails", async () => {
      // repos.get fails - cannot determine default branch, warn and continue
      mockGithub.rest.repos.get.mockRejectedValue(new Error("API unavailable"));
      const patchPath = createPatchFile();
      mockExec.getExecOutput.mockResolvedValue({ exitCode: 0, stdout: "abc123\n", stderr: "" });

      const module = await loadModule();
      const handler = await module.main({});
      const result = await handler({ patch_path: patchPath }, {});

      // Should warn but still allow the push since we can't confirm it's the default branch
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Could not check repository default branch"));
      expect(result.success).toBe(true);
    });

    it("should warn and continue when getBranchProtection returns 403 (permission denied)", async () => {
      // getBranchProtection returns 403 (no permission to read protection)
      // Platform will still enforce protection at push time, so we warn and allow
      mockGithub.rest.repos.getBranchProtection.mockRejectedValue(Object.assign(new Error("Forbidden"), { status: 403 }));
      const patchPath = createPatchFile();
      mockExec.getExecOutput.mockResolvedValue({ exitCode: 0, stdout: "abc123\n", stderr: "" });

      const module = await loadModule();
      const handler = await module.main({});
      const result = await handler({ patch_path: patchPath }, {});

      // Should warn but allow push since the platform still enforces protection at push time
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Could not check branch protection rules"));
      expect(result.success).toBe(true);
    });

    it("should block push when getBranchProtection returns an unexpected error (fail closed)", async () => {
      // getBranchProtection returns 500 - unexpected error, fail closed for security
      mockGithub.rest.repos.getBranchProtection.mockRejectedValue(Object.assign(new Error("Internal Server Error"), { status: 500 }));
      const patchPath = createPatchFile();

      const module = await loadModule();
      const handler = await module.main({});
      const result = await handler({ patch_path: patchPath }, {});

      // Should block the push when branch protection status cannot be verified
      expect(result.success).toBe(false);
      expect(result.error).toContain("Cannot verify branch protection rules");
      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Cannot verify branch protection rules"));
      // Should not attempt any git operations
      expect(mockExec.exec).not.toHaveBeenCalled();
    });
  });

  // ──────────────────────────────────────────────────────
  // Repository State Scenarios
  // ──────────────────────────────────────────────────────

  describe("repository state scenarios", () => {
    it("should handle successful normal push", async () => {
      const patchPath = createPatchFile();
      mockExec.getExecOutput.mockResolvedValue({ exitCode: 0, stdout: "abc123\n", stderr: "" });

      const module = await loadModule();
      const handler = await module.main({});
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(true);
      expect(result.branch_name).toBe("feature-branch");
      expect(result.commit_url).toContain("test-owner/test-repo/commit/");
    });

    it("should use pushed commit SHA returned by pushSignedCommits for activation comment commit link", async () => {
      const patchPath = createPatchFile();
      const updateActivationCommentModule = require("./update_activation_comment.cjs");
      const updateCommitSpy = vi.spyOn(updateActivationCommentModule, "updateActivationCommentWithCommit").mockResolvedValue(undefined);
      const pushSignedCommitsModule = require("./push_signed_commits.cjs");
      const pushSignedSpy = vi.spyOn(pushSignedCommitsModule, "pushSignedCommits").mockResolvedValue("remote-head-after");

      try {
        mockExec.getExecOutput
          .mockResolvedValueOnce({ exitCode: 0, stdout: "preflight-sha\trefs/heads/feature-branch\n", stderr: "" }) // preflight ls-remote
          .mockResolvedValueOnce({ exitCode: 0, stdout: "local-head-before\n", stderr: "" }) // rev-parse HEAD before patch
          .mockResolvedValueOnce({ exitCode: 0, stdout: "1\n", stderr: "" }); // rev-list --count

        const module = await loadModule();
        const handler = await module.main({});
        const result = await handler({ patch_path: patchPath }, {});

        expect(result.success).toBe(true);
        expect(result.commit_sha).toBe("remote-head-after");
        expect(result.commit_url).toContain("/commit/remote-head-after");
        expect(updateCommitSpy).toHaveBeenCalledWith(mockGithub, mockContext, mockCore, "remote-head-after", "https://github.com/test-owner/test-repo/commit/remote-head-after", { targetIssueNumber: 123 });
      } finally {
        pushSignedSpy.mockRestore();
        updateCommitSpy.mockRestore();
      }
    });

    it("should detect deleted branch before fetch", async () => {
      const patchPath = createPatchFile();

      mockExec.getExecOutput.mockResolvedValueOnce({ exitCode: 2, stdout: "", stderr: "fatal: couldn't find remote ref feature-branch" });

      const module = await loadModule();
      const handler = await module.main({});
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("no longer exists on origin");
      expect(mockExec.exec).not.toHaveBeenCalled();
    });

    it("should skip deleted branch failure when ignore_missing_branch_failure is enabled", async () => {
      const patchPath = createPatchFile();

      mockExec.getExecOutput.mockResolvedValueOnce({ exitCode: 2, stdout: "", stderr: "fatal: couldn't find remote ref feature-branch" });

      const module = await loadModule();
      const handler = await module.main({ ignore_missing_branch_failure: true });
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(false);
      expect(result.skipped).toBe(true);
      expect(result.error).toContain("no longer exists on origin");
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("ignore-missing-branch-failure"));
      expect(mockExec.exec).not.toHaveBeenCalled();
    });

    it("should fail with diagnostic error when branch existence check fails for other reasons", async () => {
      const patchPath = createPatchFile();

      mockExec.getExecOutput.mockResolvedValueOnce({ exitCode: 128, stdout: "", stderr: "fatal: Authentication failed" });

      const module = await loadModule();
      const handler = await module.main({});
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("Failed to verify branch");
      expect(result.error).toContain("Authentication failed");
      expect(mockExec.exec).not.toHaveBeenCalled();
    });

    it("should handle git fetch failure", async () => {
      const patchPath = createPatchFile();

      // First exec call (git fetch) fails
      mockExec.exec.mockRejectedValueOnce(new Error("Failed to fetch: network error"));

      const module = await loadModule();
      const handler = await module.main({});
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("Failed to fetch branch");
    });

    it("should handle branch not existing on origin", async () => {
      const patchPath = createPatchFile();

      // git fetch succeeds, but git rev-parse fails
      mockExec.exec.mockResolvedValueOnce(0); // fetch
      mockExec.exec.mockRejectedValueOnce(new Error("fatal: Needed a single revision"));

      const module = await loadModule();
      const handler = await module.main({});
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("does not exist on origin");
    });

    it("should skip rev-parse missing branch failure when ignore_missing_branch_failure is enabled", async () => {
      const patchPath = createPatchFile();

      // git fetch succeeds, but git rev-parse fails
      mockExec.exec.mockResolvedValueOnce(0); // fetch
      mockExec.exec.mockRejectedValueOnce(new Error("fatal: Needed a single revision"));

      const module = await loadModule();
      const handler = await module.main({ ignore_missing_branch_failure: true });
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(false);
      expect(result.skipped).toBe(true);
      expect(result.error).toContain("no longer exists on origin");
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("ignore-missing-branch-failure"));
    });

    it("should handle git checkout failure", async () => {
      const patchPath = createPatchFile();

      mockExec.exec.mockResolvedValueOnce(0); // fetch
      mockExec.exec.mockResolvedValueOnce(0); // rev-parse
      mockExec.exec.mockRejectedValueOnce(new Error("checkout failed"));

      const module = await loadModule();
      const handler = await module.main({});
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("Failed to checkout branch");
    });

    it("should handle git am (patch apply) failure with investigation", async () => {
      const patchPath = createPatchFile();

      // Set up successful fetch, rev-parse, checkout
      mockExec.exec.mockResolvedValueOnce(0); // fetch
      mockExec.exec.mockResolvedValueOnce(0); // rev-parse
      mockExec.exec.mockResolvedValueOnce(0); // checkout

      // git rev-parse HEAD for commit tracking
      mockExec.getExecOutput.mockResolvedValueOnce({ exitCode: 0, stdout: "before-sha\n", stderr: "" });

      // git am fails
      mockExec.exec.mockRejectedValueOnce(new Error("Patch does not apply"));

      // Investigation commands succeed
      mockExec.getExecOutput.mockResolvedValueOnce({ exitCode: 0, stdout: "M file.txt\n", stderr: "" }); // git status
      mockExec.getExecOutput.mockResolvedValueOnce({ exitCode: 0, stdout: "abc123 commit 1\n", stderr: "" }); // git log
      mockExec.getExecOutput.mockResolvedValueOnce({ exitCode: 0, stdout: "", stderr: "" }); // git diff
      mockExec.getExecOutput.mockResolvedValueOnce({ exitCode: 0, stdout: "patch diff", stderr: "" }); // git am --show-current-patch=diff
      mockExec.getExecOutput.mockResolvedValueOnce({ exitCode: 0, stdout: "full patch", stderr: "" }); // git am --show-current-patch

      const module = await loadModule();
      const handler = await module.main({});
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("Failed to apply patch");
      expect(mockCore.info).toHaveBeenCalledWith("Investigating patch failure...");
    });

    it("should create fallback pull request on non-fast-forward push rejection by default", async () => {
      const patchPath = createPatchFile();

      // Set up successful operations until push
      mockExec.exec.mockResolvedValueOnce(0); // fetch
      mockExec.exec.mockResolvedValueOnce(0); // rev-parse
      mockExec.exec.mockResolvedValueOnce(0); // checkout

      mockExec.getExecOutput.mockResolvedValueOnce({ exitCode: 0, stdout: "before-sha\n", stderr: "" }); // git rev-parse HEAD (before patch)

      mockExec.exec.mockResolvedValueOnce(0); // git am

      // pushSignedCommits: git rev-list returns one SHA so the push is attempted
      mockExec.getExecOutput.mockResolvedValueOnce({ exitCode: 0, stdout: "abc123\n", stderr: "" }); // git rev-list
      // pushSignedCommits: git ls-remote returns remote HEAD OID
      mockExec.getExecOutput.mockResolvedValueOnce({ exitCode: 0, stdout: "remote-oid\trefs/heads/feature-branch\n", stderr: "" }); // git ls-remote
      // pushSignedCommits: git log -1 returns commit message
      mockExec.getExecOutput.mockResolvedValueOnce({ exitCode: 0, stdout: "Test commit\n", stderr: "" }); // git log -1
      // pushSignedCommits: git diff --name-status returns file changes
      mockExec.getExecOutput.mockResolvedValueOnce({ exitCode: 0, stdout: "", stderr: "" }); // git diff --name-status (empty - no files)

      // GraphQL call fails, triggering fallback to git push
      mockGithub.graphql.mockRejectedValueOnce(new Error("GraphQL error: branch protection"));

      // Fallback git push also fails with non-fast-forward
      mockExec.exec.mockRejectedValueOnce(new Error("! [rejected] feature-branch -> feature-branch (non-fast-forward)"));

      const module = await loadModule();
      const handler = await module.main({});
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(true);
      expect(result.fallback_used).toBe(true);
      expect(result.fallback_type).toBe("pull_request");
      expect(result.pull_request_number).toBe(999);
      expect(mockGithub.rest.pulls.create).toHaveBeenCalled();
    });

    it("should not create fallback pull request when fallback-as-pull-request is disabled", async () => {
      const patchPath = createPatchFile();

      mockExec.exec.mockResolvedValueOnce(0); // fetch
      mockExec.exec.mockResolvedValueOnce(0); // rev-parse
      mockExec.exec.mockResolvedValueOnce(0); // checkout

      mockExec.getExecOutput.mockResolvedValueOnce({ exitCode: 0, stdout: "before-sha\n", stderr: "" }); // git rev-parse HEAD (before patch)

      mockExec.exec.mockResolvedValueOnce(0); // git am

      mockExec.getExecOutput.mockResolvedValueOnce({ exitCode: 0, stdout: "abc123\n", stderr: "" }); // git rev-list
      mockExec.getExecOutput.mockResolvedValueOnce({ exitCode: 0, stdout: "remote-oid\trefs/heads/feature-branch\n", stderr: "" }); // git ls-remote
      mockExec.getExecOutput.mockResolvedValueOnce({ exitCode: 0, stdout: "Test commit\n", stderr: "" }); // git log -1
      mockExec.getExecOutput.mockResolvedValueOnce({ exitCode: 0, stdout: "", stderr: "" }); // git diff --name-status

      mockGithub.graphql.mockRejectedValueOnce(new Error("GraphQL error: branch protection"));
      mockExec.exec.mockRejectedValueOnce(new Error("! [rejected] feature-branch -> feature-branch (non-fast-forward)"));

      const module = await loadModule();
      const handler = await module.main({ fallback_as_pull_request: false });
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(false);
      expect(result.error_type).toBe("push_failed");
      expect(result.error).toContain("non-fast-forward");
      expect(mockGithub.rest.pulls.create).not.toHaveBeenCalled();
    });

    it("should diagnose deleted branch when push fails", async () => {
      const patchPath = createPatchFile();

      // Set up successful operations until push
      mockExec.exec.mockResolvedValueOnce(0); // fetch
      mockExec.exec.mockResolvedValueOnce(0); // rev-parse
      mockExec.exec.mockResolvedValueOnce(0); // checkout

      mockExec.getExecOutput.mockResolvedValueOnce({ exitCode: 0, stdout: "before-sha\n", stderr: "" }); // git rev-parse HEAD (before patch)

      mockExec.exec.mockResolvedValueOnce(0); // git am

      // pushSignedCommits + post-failure diagnosis responses
      const originalGetExecOutput = mockExec.getExecOutput;
      let exitCodeLsRemoteCallCount = 0;
      mockExec.getExecOutput = vi.fn().mockImplementation(async (cmd, args) => {
        const argList = Array.isArray(args) ? args : [];
        if (argList[0] === "rev-parse" && argList[1] === "HEAD") {
          return { exitCode: 0, stdout: "before-sha\n", stderr: "" };
        }
        if (argList[0] === "rev-list") {
          return { exitCode: 0, stdout: "abc123\n", stderr: "" };
        }
        if (argList[0] === "diff-tree") {
          return { exitCode: 0, stdout: "", stderr: "" };
        }
        if (argList[0] === "ls-remote" && argList[1] === "--exit-code") {
          exitCodeLsRemoteCallCount += 1;
          if (exitCodeLsRemoteCallCount === 1) {
            // Initial preflight check before fetch
            return { exitCode: 0, stdout: "remote-oid\trefs/heads/feature-branch\n", stderr: "" };
          }
          // Post-push diagnosis call from push_to_pull_request_branch catch block
          return { exitCode: 2, stdout: "", stderr: "fatal: couldn't find remote ref feature-branch" };
        }
        if (argList[0] === "ls-remote") {
          return { exitCode: 0, stdout: "remote-oid\trefs/heads/feature-branch\n", stderr: "" };
        }
        if (argList[0] === "log") {
          return { exitCode: 0, stdout: "Test commit\n", stderr: "" };
        }
        if (argList[0] === "diff" && argList[1] === "--name-status") {
          return { exitCode: 0, stdout: "", stderr: "" };
        }
        return originalGetExecOutput(cmd, args);
      });

      // GraphQL call fails, triggering fallback to git push
      mockGithub.graphql.mockRejectedValueOnce(new Error("GraphQL error: branch protection"));
      // Fallback git push fails
      mockExec.exec.mockRejectedValueOnce(new Error("remote: Internal Server Error"));

      const module = await loadModule();
      const handler = await module.main({});
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(false);
      expect(result.error_type).toBe("push_failed");
      expect(result.error).toContain("appears to have been deleted");
    });

    it("should detect force-pushed branch via ref mismatch", async () => {
      const patchPath = createPatchFile();

      // This tests the scenario where the branch SHA we have doesn't match remote
      // The current implementation doesn't explicitly detect this, but git operations would fail
      mockContext.payload.pull_request.head.sha = "old-sha-123";

      mockGithub.rest.pulls.get.mockResolvedValue({
        data: {
          head: {
            ref: "feature-branch",
            sha: "new-sha-456", // Remote has different SHA
          },
          title: "Test PR",
          labels: [],
        },
      });

      mockExec.exec.mockResolvedValueOnce(0); // fetch
      mockExec.exec.mockResolvedValueOnce(0); // rev-parse
      mockExec.exec.mockResolvedValueOnce(0); // checkout
      mockExec.getExecOutput.mockResolvedValueOnce({ exitCode: 0, stdout: "new-sha-456\n", stderr: "" });
      mockExec.exec.mockResolvedValueOnce(0); // git am
      mockExec.exec.mockResolvedValueOnce(0); // git push
      mockExec.getExecOutput.mockResolvedValueOnce({ exitCode: 0, stdout: "final-sha\n", stderr: "" });
      mockExec.getExecOutput.mockResolvedValueOnce({ exitCode: 0, stdout: "1\n", stderr: "" }); // commit count

      const module = await loadModule();
      const handler = await module.main({});
      const result = await handler({ patch_path: patchPath }, {});

      // The push proceeds - force-push detection would need to be added
      expect(mockGithub.rest.pulls.get).toHaveBeenCalled();
    });
  });

  // ──────────────────────────────────────────────────────
  // Empty Commit / No Changes Handling
  // ──────────────────────────────────────────────────────

  describe("empty commit / no changes handling", () => {
    it("should warn when patch is empty and if-no-changes is warn", async () => {
      const patchPath = createPatchFile(""); // Empty patch

      const module = await loadModule();
      const handler = await module.main({ if_no_changes: "warn" });
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(false);
      expect(result.skipped).toBe(true);
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("noop"));
    });

    it("should error when patch is empty and if-no-changes is error", async () => {
      const patchPath = createPatchFile(""); // Empty patch

      const module = await loadModule();
      const handler = await module.main({ if_no_changes: "error" });
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("No changes");
    });

    it("should silently skip when patch is empty and if-no-changes is ignore", async () => {
      const patchPath = createPatchFile(""); // Empty patch

      const module = await loadModule();
      const handler = await module.main({ if_no_changes: "ignore" });
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(false);
      expect(result.skipped).toBe(true);
    });

    it("should handle missing patch file", async () => {
      const module = await loadModule();
      const handler = await module.main({});
      const result = await handler({ patch_path: "/nonexistent/path.patch" }, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("No patch file");
    });

    it("should handle patch file with error message", async () => {
      const patchPath = createPatchFile("Failed to generate patch: some error");

      const module = await loadModule();
      const handler = await module.main({});
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("error message");
    });
  });

  // ──────────────────────────────────────────────────────
  // Patch Size Validation
  // ──────────────────────────────────────────────────────

  describe("patch size validation", () => {
    it("should reject patches exceeding max size", async () => {
      // Create a patch larger than 1KB
      const largePatch = "x".repeat(2 * 1024 * 1024); // 2MB
      const patchPath = createPatchFile(largePatch);

      const module = await loadModule();
      const handler = await module.main({ max_patch_size: 1024 }); // 1MB max
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("exceeds maximum");
    });

    it("should accept patches within max size", async () => {
      const patchPath = createPatchFile(); // Small valid patch

      mockExec.getExecOutput.mockResolvedValue({ exitCode: 0, stdout: "abc123\n", stderr: "" });

      const module = await loadModule();
      const handler = await module.main({ max_patch_size: 1024 });
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(true);
      expect(mockCore.info).toHaveBeenCalledWith("Patch size validation passed");
    });

    it("should prefer message.diff_size (incremental net diff) over patch file size", async () => {
      // Simulate the long-running branch case: a large format-patch file
      // (e.g. 2 MB of cumulative commit metadata + per-commit diffs) but a
      // tiny incremental net diff (e.g. 5 KB of actual changes since
      // origin/<branch>). The size check must use diff_size and accept the push.
      const largePatch = "x".repeat(2 * 1024 * 1024); // 2 MB format-patch file
      const patchPath = createPatchFile(largePatch);

      mockExec.getExecOutput.mockResolvedValue({ exitCode: 0, stdout: "abc123\n", stderr: "" });

      const module = await loadModule();
      const handler = await module.main({ max_patch_size: 1024 }); // 1 MB max
      const result = await handler({ patch_path: patchPath, diff_size: 5 * 1024 }, {});

      expect(result.success).toBe(true);
      expect(mockCore.info).toHaveBeenCalledWith("Patch size validation passed");
      // Verify the size check used the incremental (diff_size) value, not the
      // 2 MB file size.
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Incremental diff size: 5 KB"));
    });

    it("should reject when message.diff_size exceeds max size even if file size is small", async () => {
      // Inverse case: small file (defensive — shouldn't happen in practice)
      // but a recorded large diff_size should still cause rejection. This
      // proves diff_size is the source of truth for the size check.
      const patchPath = createPatchFile(); // small valid patch

      const module = await loadModule();
      const handler = await module.main({ max_patch_size: 1024 }); // 1 MB max
      const result = await handler({ patch_path: patchPath, diff_size: 2 * 1024 * 1024 }, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("exceeds maximum");
    });

    it("should fall back to patch file size when message.diff_size is not provided", async () => {
      // Backward-compat: older MCP servers (or non-incremental code paths)
      // do not set diff_size. The check must continue to work using the patch
      // file size as the measurement.
      const largePatch = "x".repeat(2 * 1024 * 1024); // 2 MB
      const patchPath = createPatchFile(largePatch);

      const module = await loadModule();
      const handler = await module.main({ max_patch_size: 1024 }); // 1 MB max
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("exceeds maximum");
    });

    it("should enforce max_patch_size against the bundle file size when bundle transport is used", async () => {
      // Bundle transport does not have a format-patch file. The max_patch_size
      // limit must still apply — using the on-disk bundle file size — so large
      // bundles cannot silently bypass the limit.
      const bundlePath = path.join(tempDir, "test.bundle");
      // 2 MB dummy bundle file (contents don't matter; only size is checked)
      fs.writeFileSync(bundlePath, Buffer.alloc(2 * 1024 * 1024));

      const module = await loadModule();
      const handler = await module.main({ max_patch_size: 1024 }); // 1 MB max
      const result = await handler({ bundle_path: bundlePath }, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("exceeds maximum");
      expect(result.error).toMatch(/Bundle size|Incremental diff size/);
    });

    it("should prefer diff_size over bundle file size for the limit check", async () => {
      // Bundle is 2 MB on disk, but the incremental net diff is only 5 KB:
      // the check must accept the push (limit reflects the real change, not the
      // compressed transport size).
      const bundlePath = path.join(tempDir, "test.bundle");
      fs.writeFileSync(bundlePath, Buffer.alloc(2 * 1024 * 1024));

      mockExec.getExecOutput.mockResolvedValue({ exitCode: 0, stdout: "abc123\n", stderr: "" });

      const module = await loadModule();
      const handler = await module.main({ max_patch_size: 1024 }); // 1 MB max
      const result = await handler({ bundle_path: bundlePath, diff_size: 5 * 1024 }, {});

      expect(result.success).toBe(true);
      expect(mockCore.info).toHaveBeenCalledWith("Patch size validation passed");
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Incremental diff size: 5 KB"));
    });
  });

  // ──────────────────────────────────────────────────────
  // Title Prefix and Labels Validation
  // ──────────────────────────────────────────────────────

  describe("title prefix and labels validation", () => {
    it("should reject PR without required title prefix", async () => {
      mockGithub.rest.pulls.get.mockResolvedValue({
        data: {
          head: {
            ref: "feature-branch",
            repo: { full_name: "test-owner/test-repo", fork: false },
          },
          base: { repo: { full_name: "test-owner/test-repo" } },
          title: "Some PR Title",
          labels: [],
        },
      });

      const patchPath = createPatchFile();

      const module = await loadModule();
      const handler = await module.main({ title_prefix: "[bot] " });
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("does not start with required prefix");
    });

    it("should accept PR with required title prefix", async () => {
      mockGithub.rest.pulls.get.mockResolvedValue({
        data: {
          head: {
            ref: "feature-branch",
            repo: { full_name: "test-owner/test-repo", fork: false },
          },
          base: { repo: { full_name: "test-owner/test-repo" } },
          title: "[bot] Automated PR",
          labels: [],
        },
      });

      const patchPath = createPatchFile();
      mockExec.getExecOutput.mockResolvedValue({ exitCode: 0, stdout: "abc123\n", stderr: "" });

      const module = await loadModule();
      const handler = await module.main({ title_prefix: "[bot] " });
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(true);
      expect(mockCore.info).toHaveBeenCalledWith('✓ Title prefix validation passed: "[bot] "');
    });

    it("should reject PR without required labels", async () => {
      mockGithub.rest.pulls.get.mockResolvedValue({
        data: {
          head: {
            ref: "feature-branch",
            repo: { full_name: "test-owner/test-repo", fork: false },
          },
          base: { repo: { full_name: "test-owner/test-repo" } },
          title: "Test PR",
          labels: [{ name: "bug" }],
        },
      });

      const patchPath = createPatchFile();

      const module = await loadModule();
      const handler = await module.main({ labels: ["automated", "enhancement"] });
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("missing required labels");
    });

    it("should accept PR with all required labels", async () => {
      mockGithub.rest.pulls.get.mockResolvedValue({
        data: {
          head: {
            ref: "feature-branch",
            repo: { full_name: "test-owner/test-repo", fork: false },
          },
          base: { repo: { full_name: "test-owner/test-repo" } },
          title: "Test PR",
          labels: [{ name: "automated" }, { name: "enhancement" }],
        },
      });

      const patchPath = createPatchFile();
      mockExec.getExecOutput.mockResolvedValue({ exitCode: 0, stdout: "abc123\n", stderr: "" });

      const module = await loadModule();
      const handler = await module.main({ labels: ["automated", "enhancement"] });
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(true);
      expect(mockCore.info).toHaveBeenCalledWith("✓ Labels validation passed: automated, enhancement");
    });
  });

  // ──────────────────────────────────────────────────────
  // Commit Title Suffix
  // ──────────────────────────────────────────────────────

  describe("commit title suffix", () => {
    it("should append suffix to commit messages in patch", async () => {
      const patchPath = createPatchFile();
      mockExec.getExecOutput.mockResolvedValue({ exitCode: 0, stdout: "abc123\n", stderr: "" });

      const module = await loadModule();
      const handler = await module.main({ commit_title_suffix: " [skip ci]" });
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(true);
      expect(mockCore.info).toHaveBeenCalledWith('Appending commit title suffix: " [skip ci]"');

      // Verify patch was modified
      const modifiedPatch = fs.readFileSync(patchPath, "utf8");
      expect(modifiedPatch).toContain("[skip ci]");
    });
  });

  // ──────────────────────────────────────────────────────
  // Staged Mode
  // ──────────────────────────────────────────────────────

  describe("staged mode", () => {
    it("should generate preview instead of pushing in staged mode", async () => {
      process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";
      const patchPath = createPatchFile();

      // Re-import the module to pick up env var
      delete require.cache[require.resolve("./push_to_pull_request_branch.cjs")];

      const module = require("./push_to_pull_request_branch.cjs");
      const handler = await module.main({});
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(true);
      expect(result.staged).toBe(true);
      // Should not have made any git commands
      expect(mockExec.exec).not.toHaveBeenCalled();
    });
  });

  // ──────────────────────────────────────────────────────
  // Max Count Limiting
  // ──────────────────────────────────────────────────────

  describe("max count limiting", () => {
    it("should skip messages after max count reached", async () => {
      const patchPath = createPatchFile();
      mockExec.getExecOutput.mockResolvedValue({ exitCode: 0, stdout: "abc123\n", stderr: "" });

      const module = await loadModule();
      const handler = await module.main({ max: 1 });

      // First call succeeds
      const result1 = await handler({ patch_path: patchPath }, {});
      expect(result1.success).toBe(true);

      // Second call is skipped
      const result2 = await handler({ patch_path: patchPath }, {});
      expect(result2.success).toBe(false);
      expect(result2.skipped).toBe(true);
      expect(result2.error).toContain("Max count");
    });
  });

  // ──────────────────────────────────────────────────────
  // Branch Name Sanitization
  // ──────────────────────────────────────────────────────

  describe("branch name sanitization", () => {
    it("should sanitize branch names with shell metacharacters", async () => {
      mockGithub.rest.pulls.get.mockResolvedValue({
        data: {
          head: {
            ref: "feature;rm -rf /",
            repo: { full_name: "test-owner/test-repo", fork: false },
          },
          base: { repo: { full_name: "test-owner/test-repo" } },
          title: "Test PR",
          labels: [],
        },
      });

      const patchPath = createPatchFile();

      const module = await loadModule();
      const handler = await module.main({});
      const result = await handler({ patch_path: patchPath }, {});

      // The normalizeBranchName should sanitize dangerous characters
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Branch name sanitized"));
    });

    it("should reject empty branch name after sanitization", async () => {
      mockGithub.rest.pulls.get.mockResolvedValue({
        data: {
          head: {
            ref: ";$`|&", // All dangerous chars
            repo: { full_name: "test-owner/test-repo", fork: false },
          },
          base: { repo: { full_name: "test-owner/test-repo" } },
          title: "Test PR",
          labels: [],
        },
      });

      const patchPath = createPatchFile();

      const module = await loadModule();
      const handler = await module.main({});
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("Invalid branch name");
    });
  });

  // ──────────────────────────────────────────────────────
  // PR API Error Handling
  // ──────────────────────────────────────────────────────

  describe("PR API error handling", () => {
    it("should handle PR not found error", async () => {
      mockGithub.rest.pulls.get.mockRejectedValue(new Error("Not Found"));

      const patchPath = createPatchFile();

      const module = await loadModule();
      const handler = await module.main({});
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("Failed to determine branch name");
    });

    it("should handle network error fetching PR", async () => {
      mockGithub.rest.pulls.get.mockRejectedValue(new Error("Network timeout"));

      const patchPath = createPatchFile();

      const module = await loadModule();
      const handler = await module.main({});
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("Failed to determine branch name");
    });
  });

  // ──────────────────────────────────────────────────────
  // Extra Empty Commit (CI Trigger)
  // ──────────────────────────────────────────────────────

  describe("extra empty commit for CI trigger", () => {
    it("should push extra empty commit when CI trigger token is set", async () => {
      process.env.GH_AW_CI_TRIGGER_TOKEN = "ghp_test_token";

      // Re-import modules to pick up env var
      delete require.cache[require.resolve("./push_to_pull_request_branch.cjs")];
      delete require.cache[require.resolve("./extra_empty_commit.cjs")];

      const patchPath = createPatchFile();

      // Mock successful commands
      mockExec.exec.mockResolvedValue(0);
      mockExec.getExecOutput.mockImplementation(async (cmd, args) => {
        if (args && args[0] === "rev-parse") {
          return { exitCode: 0, stdout: "abc123\n", stderr: "" };
        }
        if (args && args[0] === "rev-list") {
          return { exitCode: 0, stdout: "1\n", stderr: "" }; // 1 new commit
        }
        if (args && args[0] === "log") {
          return { exitCode: 0, stdout: "COMMIT:abc\nfile.txt\n", stderr: "" };
        }
        return { exitCode: 0, stdout: "", stderr: "" };
      });

      const module = require("./push_to_pull_request_branch.cjs");
      const handler = await module.main({});
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(true);
      // The extra empty commit should have been attempted
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Extra empty commit"));
    });
  });

  // ──────────────────────────────────────────────────────
  // Handler Type Export
  // ──────────────────────────────────────────────────────

  describe("module exports", () => {
    it("should export HANDLER_TYPE as push_to_pull_request_branch", async () => {
      const module = await loadModule();

      expect(module.HANDLER_TYPE).toBe("push_to_pull_request_branch");
    });

    it("should export main function", async () => {
      const module = await loadModule();

      expect(typeof module.main).toBe("function");
    });
  });

  // ──────────────────────────────────────────────────────
  // allowed-files strict allowlist
  // ──────────────────────────────────────────────────────

  describe("allowed-files strict allowlist", () => {
    /**
     * Helper to create a patch that touches only the given file path(s).
     * Produces minimal but valid `diff --git` headers so extractPathsFromPatch works.
     */
    function createPatchWithFiles(...filePaths) {
      const diffs = filePaths
        .map(
          p => `diff --git a/${p} b/${p}
new file mode 100644
index 0000000..abc1234
--- /dev/null
+++ b/${p}
@@ -0,0 +1 @@
+content
`
        )
        .join("\n");
      return `From abc123 Mon Sep 17 00:00:00 2001
From: Test Author <test@example.com>
Date: Mon, 1 Jan 2024 00:00:00 +0000
Subject: [PATCH] Test commit

${diffs}
--
2.34.1
`;
    }

    it("should reject files outside the allowed-files allowlist", async () => {
      const patchPath = createPatchFile(createPatchWithFiles("src/index.js"));

      const module = await loadModule();
      const handler = await module.main({ allowed_files: [".changeset/**"] });
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("outside the allowed-files list");
      expect(result.error).toContain("src/index.js");
    });

    it("should accept files that match the allowed-files pattern", async () => {
      const patchPath = createPatchFile(createPatchWithFiles(".changeset/my-feature-fix.md"));
      mockExec.getExecOutput.mockResolvedValue({ exitCode: 0, stdout: "abc123\n", stderr: "" });

      const module = await loadModule();
      const handler = await module.main({ allowed_files: [".changeset/**"] });
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(true);
    });

    it("should still block a protected file when it is in the allowlist but protected-files: allowed is not set", async () => {
      // allowed-files and protected-files are orthogonal: both checks must pass.
      // Matching the allowlist does NOT bypass the protected-files policy.
      const patchPath = createPatchFile(createPatchWithFiles("package.json"));

      const module = await loadModule();
      const handler = await module.main({
        allowed_files: ["package.json"],
        protected_files: ["package.json"],
        protected_files_policy: "blocked",
      });
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("protected files");
      expect(result.error).toContain("package.json");
    });

    it("should allow a protected file when both allowed-files matches and protected-files: allowed is set", async () => {
      // Both checks are satisfied explicitly: allowlist scope + protected-files permission.
      const patchPath = createPatchFile(createPatchWithFiles("package.json"));
      mockExec.getExecOutput.mockResolvedValue({ exitCode: 0, stdout: "abc123\n", stderr: "" });

      const module = await loadModule();
      const handler = await module.main({
        allowed_files: ["package.json"],
        protected_files: ["package.json"],
        protected_files_policy: "allowed",
      });
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(true);
    });

    it("should block a protected file when no allowed-files list is configured", async () => {
      const patchPath = createPatchFile(createPatchWithFiles("package.json"));

      const module = await loadModule();
      const handler = await module.main({
        protected_files: ["package.json"],
        protected_files_policy: "blocked",
      });
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("protected files");
      expect(result.error).toContain("package.json");
    });

    it("should reject a mixed patch where at least one file is outside the allowlist", async () => {
      const patchPath = createPatchFile(createPatchWithFiles(".changeset/my-fix.md", "src/index.js"));

      const module = await loadModule();
      const handler = await module.main({ allowed_files: [".changeset/**"] });
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("outside the allowed-files list");
      expect(result.error).toContain("src/index.js");
      expect(result.error).not.toContain(".changeset/my-fix.md");
    });
  });

  // excluded-files exclusion list
  // ──────────────────────────────────────────────────────

  describe("excluded-files exclusion list", () => {
    /**
     * Helper to create a patch that touches only the given file path(s).
     */
    function createPatchWithFiles(...filePaths) {
      const diffs = filePaths
        .map(
          p => `diff --git a/${p} b/${p}
new file mode 100644
index 0000000..abc1234
--- /dev/null
+++ b/${p}
@@ -0,0 +1 @@
+content
`
        )
        .join("\n");
      return `From abc123 Mon Sep 17 00:00:00 2001
From: Test Author <test@example.com>
Date: Mon, 1 Jan 2024 00:00:00 +0000
Subject: [PATCH] Test commit

${diffs}
--
2.34.1
`;
    }

    it("should ignore files matching excluded-files patterns (not blocked by allowed-files)", async () => {
      // excluded-files are excluded at patch generation time via git :(exclude) pathspecs.
      // Simulate post-generation: the patch already contains only the non-ignored file.
      const patchPath = createPatchFile(createPatchWithFiles("src/index.js"));
      mockExec.getExecOutput.mockResolvedValue({ exitCode: 0, stdout: "abc123\n", stderr: "" });

      const module = await loadModule();
      const handler = await module.main({
        excluded_files: ["auto-generated/**"],
        allowed_files: ["src/**"],
      });
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.error || "").not.toContain("outside the allowed-files list");
    });

    it("should still block non-ignored files that violate the allowed-files list", async () => {
      const patchPath = createPatchFile(createPatchWithFiles("src/index.js", "other/file.txt"));

      const module = await loadModule();
      const handler = await module.main({
        excluded_files: ["auto-generated/**"],
        allowed_files: ["src/**"],
      });
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("outside the allowed-files list");
      expect(result.error).toContain("other/file.txt");
      expect(result.error).not.toContain("src/index.js");
    });

    it("should ignore files matching excluded-files patterns (not blocked by protected-files)", async () => {
      // excluded-files are excluded at patch generation time via git :(exclude) pathspecs.
      // Simulate post-generation: the patch already contains only the non-ignored file.
      const patchPath = createPatchFile(createPatchWithFiles("src/index.js"));
      mockExec.getExecOutput.mockResolvedValue({ exitCode: 0, stdout: "abc123\n", stderr: "" });

      const module = await loadModule();
      const handler = await module.main({
        excluded_files: ["package.json"],
        protected_files: ["package.json"],
        protected_files_policy: "blocked",
      });
      const result = await handler({ patch_path: patchPath }, {});

      expect(result.error || "").not.toContain("protected files");
    });

    it("should allow when all patch files are ignored (even with allowed-files set)", async () => {
      // excluded-files are excluded at patch generation time via git :(exclude) pathspecs.
      // Simulate post-generation: all files were excluded so no patch file is produced.
      const nonexistentPath = path.join(tempDir, "nonexistent.patch");

      const module = await loadModule();
      const handler = await module.main({
        excluded_files: ["dist/**"],
        allowed_files: ["src/**"],
      });
      const result = await handler({ patch_path: nonexistentPath }, {});

      // No patch → treated as no changes, not an allowlist violation
      expect(result.error || "").not.toContain("outside the allowed-files list");
    });
  });
});

// ──────────────────────────────────────────────────────
// Integration Test Recommendations
// ──────────────────────────────────────────────────────

/**
 * INTEGRATION TEST RECOMMENDATIONS
 *
 * The following scenarios require integration testing with real Git repositories.
 * These tests should be added to a separate file (e.g., push_to_pull_request_branch.integration.test.cjs)
 * and run in a CI environment with actual Git operations.
 *
 * 1. CONCURRENT PUSH SCENARIOS:
 *    - Create a test repo with a PR branch
 *    - Push a commit to the PR branch from another process
 *    - Attempt to push from the handler
 *    - Verify proper error handling for non-fast-forward rejection
 *
 * 2. FORCE-PUSH DETECTION:
 *    - Create a test repo with a PR branch
 *    - Force-push to rewrite the branch history
 *    - Attempt to apply a patch based on old history
 *    - Verify proper error message about force-push
 *
 * 3. BASE BRANCH UPDATES:
 *    - Create a test repo with a PR branch
 *    - Push commits to the base branch
 *    - Verify that pushing to PR branch still works
 *    - (Note: This should work as PR branch is independent)
 *
 * 4. MERGE CONFLICT SCENARIOS:
 *    - Create a test repo with conflicting changes
 *    - Attempt to apply a patch that conflicts
 *    - Verify clear error message about merge conflict
 *
 * 5. FORK PR EARLY FAILURE:
 *    - Create a fork repository (or simulate fork context)
 *    - Verify early failure before attempting push
 *    - Verify clear error message about fork permissions
 *
 * TEST SETUP RECOMMENDATIONS:
 * - Use GitHub API to create test repositories
 * - Use git commands to set up test scenarios
 * - Clean up test repos after each test
 * - Run these tests on a schedule, not on every commit
 */
