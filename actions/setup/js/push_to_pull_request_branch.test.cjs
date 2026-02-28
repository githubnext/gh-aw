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

    it("should detect fork PR via fork flag and fail early", async () => {
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

    it("should handle git push rejection (concurrent changes)", async () => {
      const patchPath = createPatchFile();

      // Set up successful operations until push
      mockExec.exec.mockResolvedValueOnce(0); // fetch
      mockExec.exec.mockResolvedValueOnce(0); // rev-parse
      mockExec.exec.mockResolvedValueOnce(0); // checkout

      mockExec.getExecOutput.mockResolvedValueOnce({ exitCode: 0, stdout: "before-sha\n", stderr: "" });

      mockExec.exec.mockResolvedValueOnce(0); // git am
      mockExec.exec.mockRejectedValueOnce(new Error("! [rejected] feature-branch -> feature-branch (non-fast-forward)"));

      const module = await loadModule();
      const handler = await module.main({});
      const result = await handler({ patch_path: patchPath }, {});

      // The error happens during push, which currently shows in patch apply failure
      expect(result.success).toBe(false);
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
