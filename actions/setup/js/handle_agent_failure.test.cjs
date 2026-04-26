// @ts-check

import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { createRequire } from "module";

const require = createRequire(import.meta.url);

describe("handle_agent_failure", () => {
  let main;
  let buildCodePushFailureContext;
  let buildPushRepoMemoryFailureContext;
  let getActionFailureIssueExpiresHours;

  beforeEach(() => {
    // Provide minimal GitHub Actions globals expected by require-time code
    global.core = {
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      debug: vi.fn(),
      setOutput: vi.fn(),
      setFailed: vi.fn(),
    };
    global.github = {};
    global.context = { repo: { owner: "owner", repo: "repo" } };

    // Reset module registry so each test gets a fresh require
    vi.resetModules();
    ({ main, buildCodePushFailureContext, buildPushRepoMemoryFailureContext, getActionFailureIssueExpiresHours } = require("./handle_agent_failure.cjs"));
  });

  afterEach(() => {
    delete global.core;
    delete global.github;
    delete global.context;
    delete process.env.GITHUB_SHA;
    delete process.env.GH_AW_ACTION_FAILURE_ISSUE_EXPIRES_HOURS;
  });

  describe("getActionFailureIssueExpiresHours", () => {
    it("returns default when env var is missing", () => {
      expect(getActionFailureIssueExpiresHours()).toBe(168);
    });

    it("returns configured value when env var is a positive integer", () => {
      process.env.GH_AW_ACTION_FAILURE_ISSUE_EXPIRES_HOURS = "48";
      expect(getActionFailureIssueExpiresHours()).toBe(48);
    });

    it("returns default for invalid values", () => {
      process.env.GH_AW_ACTION_FAILURE_ISSUE_EXPIRES_HOURS = "0";
      expect(getActionFailureIssueExpiresHours()).toBe(168);
      process.env.GH_AW_ACTION_FAILURE_ISSUE_EXPIRES_HOURS = "invalid";
      expect(getActionFailureIssueExpiresHours()).toBe(168);
    });
  });

  describe("detection caution placement in main()", () => {
    const fs = require("fs");
    const path = require("path");
    const os = require("os");

    /** @type {string} */
    let tmpDir;
    /** @type {string} */
    let promptsDir;

    beforeEach(() => {
      tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "aw-handle-agent-failure-"));
      promptsDir = path.join(tmpDir, "gh-aw", "prompts");
      fs.mkdirSync(promptsDir, { recursive: true });

      // Minimal templates used by main()
      fs.writeFileSync(path.join(promptsDir, "agent_failure_comment.md"), "COMMENT TEMPLATE CONTENT");
      fs.writeFileSync(path.join(promptsDir, "agent_failure_issue.md"), "ISSUE TEMPLATE CONTENT");

      process.env.RUNNER_TEMP = tmpDir;
      process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";
      process.env.GH_AW_WORKFLOW_ID = "test-workflow";
      process.env.GH_AW_RUN_URL = "https://github.com/owner/repo/actions/runs/123456";
      process.env.GH_AW_AGENT_CONCLUSION = "failure";
      process.env.GH_AW_DETECTION_CONCLUSION = "warning";
      process.env.GH_AW_DETECTION_REASON = "threat_detected";
    });

    afterEach(() => {
      delete process.env.RUNNER_TEMP;
      delete process.env.GH_AW_WORKFLOW_NAME;
      delete process.env.GH_AW_WORKFLOW_ID;
      delete process.env.GH_AW_RUN_URL;
      delete process.env.GH_AW_AGENT_CONCLUSION;
      delete process.env.GH_AW_DETECTION_CONCLUSION;
      delete process.env.GH_AW_DETECTION_REASON;

      if (tmpDir && fs.existsSync(tmpDir)) {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("prepends caution callout to existing-issue comment body and includes it only once", async () => {
      /** @type {string} */
      let capturedCommentBody = "";

      global.github = {
        rest: {
          search: {
            issuesAndPullRequests: vi.fn(async ({ q }) => {
              if (q.includes("is:pr")) {
                return { data: { total_count: 0, items: [] } };
              }
              return {
                data: {
                  total_count: 1,
                  items: [{ number: 42, html_url: "https://github.com/owner/repo/issues/42" }],
                },
              };
            }),
          },
          issues: {
            createComment: vi.fn(async ({ body }) => {
              capturedCommentBody = body;
              return { data: { id: 1001 } };
            }),
          },
          pulls: {
            get: vi.fn(),
          },
        },
        graphql: vi.fn(),
      };

      await main();

      expect(capturedCommentBody).toBeTruthy();
      expect(capturedCommentBody.startsWith("> [!CAUTION]")).toBe(true);
      expect(capturedCommentBody.indexOf("> [!CAUTION]")).toBeLessThan(capturedCommentBody.indexOf("COMMENT TEMPLATE CONTENT"));
      expect((capturedCommentBody.match(/> \[!CAUTION\]/g) || []).length).toBe(1);
      expect(capturedCommentBody).toContain("> Generated from [Test Workflow]");
    });

    it("prepends caution callout to new issue body and includes it only once", async () => {
      /** @type {string} */
      let capturedIssueBody = "";

      global.github = {
        rest: {
          search: {
            issuesAndPullRequests: vi.fn(async ({ q }) => {
              if (q.includes("is:pr")) {
                return { data: { total_count: 0, items: [] } };
              }
              return { data: { total_count: 0, items: [] } };
            }),
          },
          issues: {
            create: vi.fn(async ({ body }) => {
              capturedIssueBody = body;
              return {
                data: { number: 101, html_url: "https://github.com/owner/repo/issues/101", node_id: "I_123" },
              };
            }),
          },
          pulls: {
            get: vi.fn(),
          },
        },
        graphql: vi.fn(),
      };

      await main();

      expect(capturedIssueBody).toBeTruthy();
      expect(capturedIssueBody.startsWith("> [!CAUTION]")).toBe(true);
      expect(capturedIssueBody.indexOf("> [!CAUTION]")).toBeLessThan(capturedIssueBody.indexOf("ISSUE TEMPLATE CONTENT"));
      expect((capturedIssueBody.match(/> \[!CAUTION\]/g) || []).length).toBe(1);
      expect(capturedIssueBody).toContain("> Generated from [Test Workflow]");
    });
  });

  describe("buildCodePushFailureContext", () => {
    it("returns empty string when no errors", () => {
      expect(buildCodePushFailureContext("")).toBe("");
      expect(buildCodePushFailureContext(null)).toBe("");
      expect(buildCodePushFailureContext(undefined)).toBe("");
    });

    it("shows protected file protection section for protected file errors", () => {
      const errors = "create_pull_request:Cannot create pull request: patch modifies protected files (package.json). Set manifest-files: fallback-to-issue to create a review issue instead.";
      const result = buildCodePushFailureContext(errors);
      expect(result).toContain("🛡️ Protected Files");
      expect(result).toContain("package.json");
      expect(result).toContain("protected-files: fallback-to-issue");
      // Should NOT contain generic "Code Push Failed" for pure manifest errors
      expect(result).not.toContain("Code Push Failed");
    });

    it("shows protected file protection section for legacy 'package manifest files' error messages", () => {
      // Old error message format – must still be detected
      const errors = "create_pull_request:Cannot create pull request: patch modifies package manifest files (package.json). Set allow-manifest-files: true in your workflow to allow this.";
      const result = buildCodePushFailureContext(errors);
      expect(result).toContain("🛡️ Protected Files");
      expect(result).not.toContain("Code Push Failed");
    });

    it("shows protected file protection section for push_to_pull_request_branch errors", () => {
      const errors = "push_to_pull_request_branch:Cannot push to pull request branch: patch modifies protected files (go.mod, go.sum). Set manifest-files: fallback-to-issue to create a review issue.";
      const result = buildCodePushFailureContext(errors);
      expect(result).toContain("🛡️ Protected Files");
      expect(result).toContain("go.mod");
      expect(result).toContain("`push_to_pull_request_branch`");
      expect(result).not.toContain("Code Push Failed");
    });

    it("shows protected file protection for .github/ protected path errors", () => {
      const errors = "create_pull_request:Cannot create pull request: patch modifies protected files (.github/workflows/ci.yml). Set manifest-files: fallback-to-issue to create a review issue.";
      const result = buildCodePushFailureContext(errors);
      expect(result).toContain("🛡️ Protected Files");
      expect(result).toContain(".github/workflows/ci.yml");
    });

    it("includes PR link in protected file protection section when PR is provided", () => {
      const errors = "create_pull_request:Cannot create pull request: patch modifies package manifest files (package.json). Set allow-manifest-files: true in your workflow to allow this.";
      const pullRequest = { number: 42, html_url: "https://github.com/owner/repo/pull/42" };
      const result = buildCodePushFailureContext(errors, pullRequest);
      expect(result).toContain("🛡️ Protected Files");
      expect(result).toContain("#42");
      expect(result).toContain("https://github.com/owner/repo/pull/42");
      // PR state diagnostics should NOT appear for protected-file-only failures
      expect(result).not.toContain("PR State at Push Time");
    });

    it("shows generic code push failure section for non-manifest errors", () => {
      const errors = "push_to_pull_request_branch:Branch not found";
      const result = buildCodePushFailureContext(errors);
      expect(result).toContain("Code Push Failed");
      expect(result).toContain("Branch not found");
      expect(result).not.toContain("Protected Files");
    });

    it("shows both sections when protected file and non-protected-file errors are mixed", () => {
      const errors = [
        "create_pull_request:Cannot create pull request: patch modifies package manifest files (package.json). Set allow-manifest-files: true in your workflow to allow this.",
        "push_to_pull_request_branch:Branch not found",
      ].join("\n");
      const result = buildCodePushFailureContext(errors);
      expect(result).toContain("🛡️ Protected Files");
      expect(result).toContain("Code Push Failed");
      expect(result).toContain("package.json");
      expect(result).toContain("Branch not found");
    });

    it("includes yaml remediation snippet in protected file protection section", () => {
      const errors = "create_pull_request:Cannot create pull request: patch modifies package manifest files (requirements.txt). Set allow-manifest-files: true in your workflow to allow this.";
      const result = buildCodePushFailureContext(errors);
      expect(result).toContain("```yaml");
      expect(result).toContain("create-pull-request:");
      expect(result).toContain("protected-files: fallback-to-issue");
    });

    it("uses push-to-pull-request-branch key in yaml snippet for push type", () => {
      const errors = "push_to_pull_request_branch:Cannot push to pull request branch: patch modifies package manifest files (go.mod). Set manifest-files: fallback-to-issue in your workflow to allow this.";
      const result = buildCodePushFailureContext(errors);
      expect(result).toContain("push-to-pull-request-branch:");
      expect(result).toContain("protected-files: fallback-to-issue");
      expect(result).not.toContain("create-pull-request:");
    });

    it("includes both yaml keys when both types have protected file errors", () => {
      const errors = [
        "create_pull_request:Cannot create pull request: patch modifies package manifest files (package.json). Set manifest-files: fallback-to-issue in your workflow to allow this.",
        "push_to_pull_request_branch:Cannot push to pull request branch: patch modifies package manifest files (go.mod). Set manifest-files: fallback-to-issue in your workflow to allow this.",
      ].join("\n");
      const result = buildCodePushFailureContext(errors);
      expect(result).toContain("create-pull-request:");
      expect(result).toContain("push-to-pull-request-branch:");
    });

    // ──────────────────────────────────────────────────────
    // Patch Size Exceeded
    // ──────────────────────────────────────────────────────

    it("shows patch size exceeded section for create_pull_request patch size error", () => {
      const errors = "create_pull_request:Patch size (2048 KB) exceeds maximum allowed size (1024 KB)";
      const result = buildCodePushFailureContext(errors);
      expect(result).toContain("📦 Patch Size Exceeded");
      expect(result).toContain("create-pull-request:");
      expect(result).toContain("max-patch-size:");
      expect(result).not.toContain("Code Push Failed");
      expect(result).not.toContain("Protected Files");
    });

    it("shows patch size exceeded section for push_to_pull_request_branch patch size error", () => {
      const errors = "push_to_pull_request_branch:Patch size (3072 KB) exceeds maximum allowed size (1024 KB)";
      const result = buildCodePushFailureContext(errors);
      expect(result).toContain("📦 Patch Size Exceeded");
      expect(result).toContain("push-to-pull-request-branch:");
      expect(result).toContain("max-patch-size:");
      expect(result).not.toContain("Code Push Failed");
    });

    it("shows patch size exceeded yaml snippet with both types when both have patch size errors", () => {
      const errors = ["create_pull_request:Patch size (2048 KB) exceeds maximum allowed size (1024 KB)", "push_to_pull_request_branch:Patch size (3072 KB) exceeds maximum allowed size (1024 KB)"].join("\n");
      const result = buildCodePushFailureContext(errors);
      expect(result).toContain("📦 Patch Size Exceeded");
      expect(result).toContain("create-pull-request:");
      expect(result).toContain("push-to-pull-request-branch:");
      expect(result).toContain("max-patch-size:");
    });

    it("includes PR link in patch size exceeded section when PR is provided", () => {
      const errors = "create_pull_request:Patch size (2048 KB) exceeds maximum allowed size (1024 KB)";
      const pullRequest = { number: 99, html_url: "https://github.com/owner/repo/pull/99" };
      const result = buildCodePushFailureContext(errors, pullRequest);
      expect(result).toContain("📦 Patch Size Exceeded");
      expect(result).toContain("#99");
      expect(result).toContain("https://github.com/owner/repo/pull/99");
    });

    it("does not show patch size section for generic errors", () => {
      const errors = "push_to_pull_request_branch:Branch not found";
      const result = buildCodePushFailureContext(errors);
      expect(result).not.toContain("📦 Patch Size Exceeded");
    });

    it("shows both patch size and generic sections when mixed", () => {
      const errors = ["create_pull_request:Patch size (2048 KB) exceeds maximum allowed size (1024 KB)", "push_to_pull_request_branch:Branch not found"].join("\n");
      const result = buildCodePushFailureContext(errors);
      expect(result).toContain("📦 Patch Size Exceeded");
      expect(result).toContain("Code Push Failed");
      expect(result).toContain("Branch not found");
    });

    // ──────────────────────────────────────────────────────
    // Patch Apply Failed (merge conflict)
    // ──────────────────────────────────────────────────────

    it("shows patch apply failed section for create_pull_request patch apply error", () => {
      const errors = "create_pull_request:Failed to apply patch";
      const result = buildCodePushFailureContext(errors);
      expect(result).toContain("🔀 Patch Apply Failed");
      expect(result).toContain("merge conflict");
      expect(result).toContain("`create_pull_request`");
      expect(result).toContain("Failed to apply patch");
      // Should NOT show generic "Code Push Failed" for pure patch apply errors
      expect(result).not.toContain("Code Push Failed");
    });

    it("shows patch apply failed section for push_to_pull_request_branch patch apply error", () => {
      const errors = "push_to_pull_request_branch:Failed to apply patch";
      const result = buildCodePushFailureContext(errors);
      expect(result).toContain("🔀 Patch Apply Failed");
      expect(result).toContain("`push_to_pull_request_branch`");
      expect(result).not.toContain("Code Push Failed");
    });

    it("includes PR link in patch apply failed section when PR is provided", () => {
      const errors = "create_pull_request:Failed to apply patch";
      const pullRequest = { number: 77, html_url: "https://github.com/owner/repo/pull/77" };
      const result = buildCodePushFailureContext(errors, pullRequest);
      expect(result).toContain("🔀 Patch Apply Failed");
      expect(result).toContain("#77");
      expect(result).toContain("https://github.com/owner/repo/pull/77");
    });

    it("includes patch download instructions with run ID when runUrl is provided", () => {
      const errors = "create_pull_request:Failed to apply patch";
      const runUrl = "https://github.com/owner/repo/actions/runs/12345678";
      const result = buildCodePushFailureContext(errors, null, runUrl);
      expect(result).toContain("🔀 Patch Apply Failed");
      expect(result).toContain("gh run download 12345678");
      expect(result).toContain("-n agent");
      expect(result).toContain("/tmp/agent-");
      expect(result).toContain("git am --3way");
      expect(result).toContain(runUrl);
      // Should use progressive disclosure for the apply commands
      expect(result).toContain("<details>");
      expect(result).toContain("Apply the patch manually");
    });

    it("shows generic download instructions when runUrl is not provided", () => {
      const errors = "create_pull_request:Failed to apply patch";
      const result = buildCodePushFailureContext(errors);
      expect(result).toContain("🔀 Patch Apply Failed");
      expect(result).toContain("git am --3way");
      // No specific run ID in instructions
      expect(result).not.toContain("gh run download");
      // Should still use progressive disclosure
      expect(result).toContain("<details>");
      expect(result).toContain("Apply the patch manually");
    });

    it("shows both patch apply failed and generic sections when mixed", () => {
      const errors = ["create_pull_request:Failed to apply patch", "push_to_pull_request_branch:Branch not found"].join("\n");
      const result = buildCodePushFailureContext(errors);
      expect(result).toContain("🔀 Patch Apply Failed");
      expect(result).toContain("Code Push Failed");
      expect(result).toContain("Branch not found");
    });

    it("does not show patch apply section for generic errors", () => {
      const errors = "push_to_pull_request_branch:Branch not found";
      const result = buildCodePushFailureContext(errors);
      expect(result).not.toContain("🔀 Patch Apply Failed");
    });
  });

  // ──────────────────────────────────────────────────────
  // buildPushRepoMemoryFailureContext
  // ──────────────────────────────────────────────────────

  describe("buildPushRepoMemoryFailureContext", () => {
    it("returns empty string when no failure", () => {
      expect(buildPushRepoMemoryFailureContext(false, [], "https://example.com/run")).toBe("");
    });

    it("shows generic failure message when failure but no patch size exceeded", () => {
      const result = buildPushRepoMemoryFailureContext(true, [], "https://example.com/run");
      expect(result).toContain("⚠️ Repo-Memory Push Failed");
      expect(result).toContain("https://example.com/run");
      expect(result).not.toContain("📦 Repo-Memory Patch Size Exceeded");
    });

    it("shows patch size exceeded message with front matter example when patch size exceeded", () => {
      const result = buildPushRepoMemoryFailureContext(true, ["default"], "https://example.com/run");
      expect(result).toContain("📦 Repo-Memory Patch Size Exceeded");
      expect(result).toContain("`default`");
      expect(result).toContain("max-patch-size:");
      expect(result).toContain("repo-memory:");
      expect(result).not.toContain("⚠️ Repo-Memory Push Failed");
    });

    it("includes all affected memory IDs in patch size exceeded message", () => {
      const result = buildPushRepoMemoryFailureContext(true, ["default", "secondary"], "https://example.com/run");
      expect(result).toContain("`default`");
      expect(result).toContain("`secondary`");
      expect(result).toContain("id: default");
      expect(result).toContain("id: secondary");
    });

    it("shows yaml front matter snippet for each affected memory ID", () => {
      const result = buildPushRepoMemoryFailureContext(true, ["my-memory"], "https://example.com/run");
      expect(result).toContain("```yaml");
      expect(result).toContain("repo-memory:");
      expect(result).toContain("id: my-memory");
      expect(result).toContain("max-patch-size: 51200");
    });
  });

  // ──────────────────────────────────────────────────────
  // buildAppTokenMintingFailedContext
  // ──────────────────────────────────────────────────────

  describe("buildAppTokenMintingFailedContext", () => {
    let buildAppTokenMintingFailedContext;
    const fs = require("fs");
    const path = require("path");
    const templateContent = fs.readFileSync(path.join(__dirname, "../md/app_token_minting_failed.md"), "utf8");
    const originalReadFileSync = fs.readFileSync.bind(fs);

    beforeEach(() => {
      vi.resetModules();
      // Stub readFileSync so the runtime path resolves to the source-tree template
      fs.readFileSync = (filePath, encoding) => {
        if (typeof filePath === "string" && filePath.includes("app_token_minting_failed.md")) {
          return templateContent;
        }
        return originalReadFileSync(filePath, encoding);
      };
      ({ buildAppTokenMintingFailedContext } = require("./handle_agent_failure.cjs"));
    });

    afterEach(() => {
      fs.readFileSync = originalReadFileSync;
    });

    it("returns empty string when no failure", () => {
      expect(buildAppTokenMintingFailedContext(false)).toBe("");
    });

    it("returns formatted error message when app token minting failed", () => {
      const result = buildAppTokenMintingFailedContext(true);
      expect(result).toContain("GitHub App Authentication Failed");
      expect(result).toContain("App ID");
      expect(result).toContain("private key");
      expect(result).toContain("installed");
    });

    it("includes actionable remediation steps", () => {
      const result = buildAppTokenMintingFailedContext(true);
      expect(result).toContain("required permissions");
      expect(result).toContain("https://github.github.com/gh-aw/reference/safe-outputs/");
    });
  });

  // ──────────────────────────────────────────────────────
  // buildLockdownCheckFailedContext
  // ──────────────────────────────────────────────────────

  describe("buildLockdownCheckFailedContext", () => {
    let buildLockdownCheckFailedContext;
    const fs = require("fs");
    const path = require("path");
    const templateContent = fs.readFileSync(path.join(__dirname, "../md/lockdown_check_failed.md"), "utf8");
    const originalReadFileSync = fs.readFileSync.bind(fs);

    beforeEach(() => {
      vi.resetModules();
      // Stub readFileSync so the runtime path resolves to the source-tree template
      fs.readFileSync = (filePath, encoding) => {
        if (typeof filePath === "string" && filePath.includes("lockdown_check_failed.md")) {
          return templateContent;
        }
        return originalReadFileSync(filePath, encoding);
      };
      ({ buildLockdownCheckFailedContext } = require("./handle_agent_failure.cjs"));
    });

    afterEach(() => {
      fs.readFileSync = originalReadFileSync;
    });

    it("returns empty string when no failure", () => {
      expect(buildLockdownCheckFailedContext(false)).toBe("");
    });

    it("returns formatted error message when lockdown check failed", () => {
      const result = buildLockdownCheckFailedContext(true);
      expect(result).toContain("Lockdown Check Failed");
    });

    it("includes token configuration guidance", () => {
      const result = buildLockdownCheckFailedContext(true);
      expect(result).toContain("GH_AW_GITHUB_TOKEN");
      expect(result).toContain("gh aw secrets set");
    });

    it("includes strict mode guidance", () => {
      const result = buildLockdownCheckFailedContext(true);
      expect(result).toContain("gh aw compile --strict");
    });
  });

  // ──────────────────────────────────────────────────────
  // buildStaleLockFileFailedContext
  // ──────────────────────────────────────────────────────

  describe("buildStaleLockFileFailedContext", () => {
    let buildStaleLockFileFailedContext;
    const fs = require("fs");
    const path = require("path");
    const templateContent = fs.readFileSync(path.join(__dirname, "../md/stale_lock_file_failed.md"), "utf8");
    const originalReadFileSync = fs.readFileSync.bind(fs);

    beforeEach(() => {
      vi.resetModules();
      fs.readFileSync = (filePath, encoding) => {
        if (typeof filePath === "string" && filePath.includes("stale_lock_file_failed.md")) {
          return templateContent;
        }
        return originalReadFileSync(filePath, encoding);
      };
      ({ buildStaleLockFileFailedContext } = require("./handle_agent_failure.cjs"));
    });

    afterEach(() => {
      fs.readFileSync = originalReadFileSync;
    });

    it("returns empty string when check did not fail", () => {
      expect(buildStaleLockFileFailedContext(false)).toBe("");
    });

    it("returns formatted context when stale lock file check failed", () => {
      const result = buildStaleLockFileFailedContext(true);
      expect(result).toBeTruthy();
      expect(result.length).toBeGreaterThan(0);
    });

    it("includes recompile guidance", () => {
      const result = buildStaleLockFileFailedContext(true);
      expect(result).toContain("gh aw compile");
    });

    it("includes guidance on how to disable the check", () => {
      const result = buildStaleLockFileFailedContext(true);
      expect(result).toContain("stale-check: false");
    });

    it("includes debug logging guidance", () => {
      const result = buildStaleLockFileFailedContext(true);
      expect(result).toContain("[hash-debug]");
    });
  });

  // ──────────────────────────────────────────────────────
  // buildTimeoutContext
  // ──────────────────────────────────────────────────────

  describe("buildTimeoutContext", () => {
    let buildTimeoutContext;
    const fs = require("fs");
    const path = require("path");
    const templateContent = fs.readFileSync(path.join(__dirname, "../md/agent_timeout.md"), "utf8");
    const originalReadFileSync = fs.readFileSync.bind(fs);

    beforeEach(() => {
      vi.resetModules();
      // Stub readFileSync so the runtime path resolves to the source-tree template
      fs.readFileSync = (filePath, encoding) => {
        if (typeof filePath === "string" && filePath.includes("agent_timeout.md")) {
          return templateContent;
        }
        return originalReadFileSync(filePath, encoding);
      };
      ({ buildTimeoutContext } = require("./handle_agent_failure.cjs"));
    });

    afterEach(() => {
      fs.readFileSync = originalReadFileSync;
    });

    it("returns empty string when not timed out", () => {
      expect(buildTimeoutContext(false, "20")).toBe("");
      expect(buildTimeoutContext(false, "")).toBe("");
    });

    it("returns formatted error message when timed out", () => {
      const result = buildTimeoutContext(true, "20");
      expect(result).toContain("Agent Timed Out");
      expect(result).toContain("20");
      expect(result).toContain("30");
      expect(result).toContain("timeout-minutes");
    });

    it("uses default of 20 minutes when timeoutMinutes is empty", () => {
      const result = buildTimeoutContext(true, "");
      expect(result).toContain("20");
      expect(result).toContain("30");
    });

    it("suggests current + 10 minutes", () => {
      const result = buildTimeoutContext(true, "45");
      expect(result).toContain("45");
      expect(result).toContain("55");
    });
  });

  // ──────────────────────────────────────────────────────
  // timeout classification (isTimedOut logic in main)
  // ──────────────────────────────────────────────────────

  describe("timeout classification", () => {
    // Mirrors the classification logic in main():
    //   const isTimedOut = agentConclusion === "timed_out" || agenticEngineTimeout;
    // This ensures step-level timeouts (detected via signal in the engine log)
    // are treated as timeouts even when agentConclusion is "failure".
    function classifyTimeout(agentConclusion, agenticEngineTimeout) {
      return agentConclusion === "timed_out" || agenticEngineTimeout;
    }

    it("detects job-level timeout (agentConclusion === 'timed_out')", () => {
      expect(classifyTimeout("timed_out", false)).toBe(true);
    });

    it("detects step-level timeout (agentConclusion === 'failure' with agenticEngineTimeout)", () => {
      expect(classifyTimeout("failure", true)).toBe(true);
    });

    it("detects timeout when both indicators are present", () => {
      expect(classifyTimeout("timed_out", true)).toBe(true);
    });

    it("does not flag timeout for plain failure without engine timeout signal", () => {
      expect(classifyTimeout("failure", false)).toBe(false);
    });

    it("does not flag timeout for successful completion", () => {
      expect(classifyTimeout("success", false)).toBe(false);
    });

    it("does not flag timeout for cancelled job", () => {
      expect(classifyTimeout("cancelled", false)).toBe(false);
    });
  });

  // ──────────────────────────────────────────────────────
  // buildEngineFailureContext
  // ──────────────────────────────────────────────────────

  describe("buildEngineFailureContext", () => {
    let buildEngineFailureContext;
    const fs = require("fs");
    const path = require("path");
    const os = require("os");

    /** @type {string} */
    let tmpDir;
    /** @type {string} */
    let stdioLogPath;

    /** @type {string} */
    let promptsDir;

    beforeEach(() => {
      vi.resetModules();
      tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "aw-test-"));
      stdioLogPath = path.join(tmpDir, "agent-stdio.log");
      promptsDir = path.join(tmpDir, "gh-aw", "prompts");
      fs.mkdirSync(promptsDir, { recursive: true });
      process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "agent_output.json");
      process.env.RUNNER_TEMP = tmpDir;
      ({ buildEngineFailureContext } = require("./handle_agent_failure.cjs"));
    });

    afterEach(() => {
      delete process.env.GH_AW_AGENT_OUTPUT;
      delete process.env.GH_AW_ENGINE_ID;
      delete process.env.RUNNER_TEMP;
      // Clean up temp dir
      if (fs.existsSync(tmpDir)) {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("returns empty string when log file does not exist", () => {
      // stdioLogPath not written — file does not exist
      expect(buildEngineFailureContext()).toBe("");
    });

    it("returns empty string when log file is empty", () => {
      fs.writeFileSync(stdioLogPath, "");
      expect(buildEngineFailureContext()).toBe("");
    });

    it("returns empty string when log file contains only whitespace", () => {
      fs.writeFileSync(stdioLogPath, "   \n\n   ");
      expect(buildEngineFailureContext()).toBe("");
    });

    it("detects ERROR: prefix pattern (Codex/generic CLI)", () => {
      fs.writeFileSync(stdioLogPath, "ERROR: quota exceeded\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("Engine Failure");
      expect(result).toContain("quota exceeded");
      expect(result).toContain("Error details:");
    });

    it("detects Error: prefix pattern (Node.js style)", () => {
      fs.writeFileSync(stdioLogPath, "Error: connect ECONNREFUSED 127.0.0.1:8080\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("Engine Failure");
      expect(result).toContain("connect ECONNREFUSED 127.0.0.1:8080");
    });

    it("detects Fatal: prefix pattern", () => {
      fs.writeFileSync(stdioLogPath, "Fatal: out of memory\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("Engine Failure");
      expect(result).toContain("out of memory");
    });

    it("detects FATAL: prefix pattern", () => {
      fs.writeFileSync(stdioLogPath, "FATAL: unexpected shutdown\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("Engine Failure");
      expect(result).toContain("unexpected shutdown");
    });

    it("detects panic: prefix pattern (Go runtime)", () => {
      fs.writeFileSync(stdioLogPath, "panic: runtime error: index out of range\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("Engine Failure");
      expect(result).toContain("runtime error: index out of range");
    });

    it("detects Reconnecting pattern", () => {
      fs.writeFileSync(stdioLogPath, "Reconnecting... 1/3 (connection lost)\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("Engine Failure");
      expect(result).toContain("connection lost");
    });

    it("deduplicates repeated error messages", () => {
      fs.writeFileSync(stdioLogPath, "ERROR: quota exceeded\nERROR: quota exceeded\nERROR: quota exceeded\n");
      const result = buildEngineFailureContext();
      const count = (result.match(/quota exceeded/g) || []).length;
      expect(count).toBe(1);
    });

    it("collects multiple distinct error messages", () => {
      fs.writeFileSync(stdioLogPath, "ERROR: quota exceeded\nERROR: auth failed\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("quota exceeded");
      expect(result).toContain("auth failed");
    });

    it("falls back to last lines when no known error patterns match", () => {
      const logLines = ["Starting agent...", "Running tool: list_branches", '{"branches": ["main"]}', "Running tool: get_file_contents", "Agent interrupted"];
      fs.writeFileSync(stdioLogPath, logLines.join("\n") + "\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("Engine Failure");
      expect(result).toContain("Last agent output");
      expect(result).toContain("Agent interrupted");
    });

    it("fallback includes at most 10 non-empty lines", () => {
      const lines = Array.from({ length: 20 }, (_, i) => `line ${i + 1}`);
      fs.writeFileSync(stdioLogPath, lines.join("\n") + "\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("line 20");
      expect(result).toContain("line 11");
      // Lines 1-10 should not appear in the tail
      expect(result).not.toContain("line 10\n");
      expect(result).not.toContain("line 1\n");
    });

    it("fallback ignores empty lines when counting tail", () => {
      const lines = ["line 1", "", "line 2", "", "line 3", "", "", "line 4"];
      fs.writeFileSync(stdioLogPath, lines.join("\n") + "\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("Last agent output");
      expect(result).toContain("line 4");
      expect(result).toContain("line 1");
    });

    it("shows startup-failure message when log contains only AWF infrastructure lines", () => {
      // This is the exact pattern from the Apr 8 systemic failure incident:
      // containers stop cleanly, engine exits with code 1, no substantive output produced.
      const infraLines = [
        " Container awf-squid  Removing",
        " Container awf-squid  Removed",
        "[SUCCESS] Containers stopped successfully",
        "[INFO] Agent session state preserved at: /tmp/awf-agent-session-state-abc123",
        "[INFO] API proxy logs available at: /tmp/gh-aw/sandbox/firewall/logs/api-proxy-logs",
        "[WARN] Command completed with exit code: 1",
        "Process exiting with code: 1",
      ];
      fs.writeFileSync(stdioLogPath, infraLines.join("\n") + "\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("Engine Failure");
      expect(result).toContain("terminated before producing output");
      expect(result).toContain("transient infrastructure issue");
      // Infrastructure lines should NOT appear as "Last agent output"
      expect(result).not.toContain("Last agent output");
      expect(result).not.toContain("awf-squid");
      expect(result).not.toContain("Command completed with exit code");
      expect(result).not.toContain("Process exiting with code");
    });

    it("filters infrastructure lines from fallback tail when mixed with real agent output", () => {
      // Real agent output followed by AWF infrastructure shutdown lines.
      // Only the real agent output should appear in the fallback.
      const logLines = [
        "Starting agent...",
        "● list_files",
        "  └ Found 12 files",
        " Container awf-squid  Removing",
        " Container awf-squid  Removed",
        "[SUCCESS] Containers stopped successfully",
        "[WARN] Command completed with exit code: 1",
        "Process exiting with code: 1",
      ];
      fs.writeFileSync(stdioLogPath, logLines.join("\n") + "\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("Last agent output");
      expect(result).toContain("Starting agent");
      expect(result).toContain("Found 12 files");
      // Infrastructure lines must be excluded from the displayed output
      expect(result).not.toContain("awf-squid");
      expect(result).not.toContain("Command completed with exit code");
      expect(result).not.toContain("Process exiting with code");
    });

    it("includes [entrypoint] and [health-check] infra lines in the infra filter", () => {
      // AWF container scripts emit lowercase [entrypoint] and [health-check] prefixes.
      // The INFRA_LINE_RE pattern is intentionally case-sensitive and matches exactly
      // the casing produced by each AWF component (consistent with parse_copilot_log.cjs).
      const lines = ["[entrypoint] Starting firewall...", "[health-check] Proxy ready", "[INFO] API proxy logs available at: /tmp/gh-aw/logs", "Process exiting with code: 1"];
      fs.writeFileSync(stdioLogPath, lines.join("\n") + "\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("Engine Failure");
      expect(result).toContain("terminated before producing output");
      // None of the infra lines should appear
      expect(result).not.toContain("entrypoint");
      expect(result).not.toContain("health-check");
      expect(result).not.toContain("API proxy");
    });

    it("includes engine ID in startup-failure message", () => {
      process.env.GH_AW_ENGINE_ID = "copilot";
      vi.resetModules();
      ({ buildEngineFailureContext } = require("./handle_agent_failure.cjs"));
      const infraLines = ["[WARN] Command completed with exit code: 1", "Process exiting with code: 1"];
      fs.writeFileSync(stdioLogPath, infraLines.join("\n") + "\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("`copilot` engine");
      expect(result).toContain("terminated before producing output");
      // Copilot-specific status page guidance
      expect(result).toContain("GitHub Copilot status page");
    });

    it("shows provider-agnostic status page guidance for non-copilot engines", () => {
      process.env.GH_AW_ENGINE_ID = "claude";
      vi.resetModules();
      ({ buildEngineFailureContext } = require("./handle_agent_failure.cjs"));
      const infraLines = ["[WARN] Command completed with exit code: 1", "Process exiting with code: 1"];
      fs.writeFileSync(stdioLogPath, infraLines.join("\n") + "\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("`claude` engine");
      expect(result).toContain("terminated before producing output");
      // Generic guidance for non-copilot engines
      expect(result).toContain("provider status page");
      expect(result).not.toContain("GitHub Copilot status page");
    });

    it("includes engine ID in failure message when GH_AW_ENGINE_ID is set", () => {
      process.env.GH_AW_ENGINE_ID = "copilot";
      vi.resetModules();
      ({ buildEngineFailureContext } = require("./handle_agent_failure.cjs"));
      fs.writeFileSync(stdioLogPath, "ERROR: quota exceeded\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("`copilot` engine");
    });

    it("includes engine ID in fallback message when GH_AW_ENGINE_ID is set", () => {
      process.env.GH_AW_ENGINE_ID = "claude";
      vi.resetModules();
      ({ buildEngineFailureContext } = require("./handle_agent_failure.cjs"));
      fs.writeFileSync(stdioLogPath, "Agent did something unexpected\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("`claude` engine");
    });

    it("uses generic 'AI engine' label when GH_AW_ENGINE_ID is not set", () => {
      fs.writeFileSync(stdioLogPath, "ERROR: connection reset\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("The AI engine");
    });

    it("returns dedicated cyber_policy_violation message when template exists", () => {
      const templateContent = "**OpenAI Cyber Policy Violation**: The Codex engine was blocked by OpenAI's safety policy.";
      fs.writeFileSync(path.join(promptsDir, "cyber_policy_violation.md"), templateContent);
      fs.writeFileSync(stdioLogPath, "ERROR: cyber_policy_violation\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("Cyber Policy Violation");
      expect(result).not.toContain("Engine Failure");
      expect(result).not.toContain("cyber_policy_violation");
    });

    it("falls back to generic message when cyber_policy_violation template is missing", () => {
      // No template file written — promptsDir exists but template is absent
      fs.writeFileSync(stdioLogPath, "ERROR: cyber_policy_violation\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("Engine Failure");
      expect(result).toContain("cyber_policy_violation");
    });

    it("returns dedicated message when cyber_policy_violation appears among multiple errors", () => {
      const templateContent = "**OpenAI Cyber Policy Violation**: The Codex engine was blocked by OpenAI's safety policy.";
      fs.writeFileSync(path.join(promptsDir, "cyber_policy_violation.md"), templateContent);
      fs.writeFileSync(stdioLogPath, "ERROR: connection reset\nERROR: cyber_policy_violation\n");
      const result = buildEngineFailureContext();
      expect(result).toContain("Cyber Policy Violation");
      expect(result).not.toContain("Engine Failure");
    });
  });

  // ──────────────────────────────────────────────────────
  // buildMCPPolicyErrorContext
  // ──────────────────────────────────────────────────────

  describe("buildMCPPolicyErrorContext", () => {
    let buildMCPPolicyErrorContext;
    const fs = require("fs");
    const path = require("path");
    const os = require("os");

    /** @type {string} */
    let tmpDir;

    /** @type {string} */
    let promptsDir;

    beforeEach(() => {
      vi.resetModules();
      tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "aw-test-mcp-"));
      promptsDir = path.join(tmpDir, "gh-aw", "prompts");
      fs.mkdirSync(promptsDir, { recursive: true });
      process.env.RUNNER_TEMP = tmpDir;
      ({ buildMCPPolicyErrorContext } = require("./handle_agent_failure.cjs"));
    });

    afterEach(() => {
      delete process.env.RUNNER_TEMP;
      if (fs.existsSync(tmpDir)) {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("returns empty string when no MCP policy error", () => {
      expect(buildMCPPolicyErrorContext(false)).toBe("");
    });

    it("returns template content when MCP policy error and template exists", () => {
      const templateContent = "\n**🔒 MCP Servers Blocked by Policy**: Test message.\n";
      fs.writeFileSync(path.join(promptsDir, "mcp_policy_error.md"), templateContent);
      const result = buildMCPPolicyErrorContext(true);
      expect(result).toContain("MCP Servers Blocked by Policy");
    });

    it("includes link to official documentation when template exists", () => {
      const templateContent = "**🔒 MCP Servers Blocked by Policy**: See [docs](https://docs.github.com/en/copilot/how-tos/administer-copilot/manage-mcp-usage/configure-mcp-server-access).\n";
      fs.writeFileSync(path.join(promptsDir, "mcp_policy_error.md"), templateContent);
      const result = buildMCPPolicyErrorContext(true);
      expect(result).toContain("docs.github.com/en/copilot/how-tos/administer-copilot/manage-mcp-usage/configure-mcp-server-access");
    });

    it("returns inline fallback message when template is missing", () => {
      // No template file written
      const result = buildMCPPolicyErrorContext(true);
      expect(result).toContain("MCP Servers Blocked by Policy");
      expect(result).toContain("configure-mcp-server-access");
    });
  });

  // buildModelNotSupportedErrorContext
  // ──────────────────────────────────────────────────────

  describe("buildModelNotSupportedErrorContext", () => {
    let buildModelNotSupportedErrorContext;
    const fs = require("fs");
    const path = require("path");
    const os = require("os");

    /** @type {string} */
    let tmpDir;

    /** @type {string} */
    let promptsDir;

    beforeEach(() => {
      vi.resetModules();
      tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "aw-test-model-not-supported-"));
      promptsDir = path.join(tmpDir, "gh-aw", "prompts");
      fs.mkdirSync(promptsDir, { recursive: true });
      process.env.RUNNER_TEMP = tmpDir;
      ({ buildModelNotSupportedErrorContext } = require("./handle_agent_failure.cjs"));
    });

    afterEach(() => {
      delete process.env.RUNNER_TEMP;
      if (fs.existsSync(tmpDir)) {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("returns empty string when no model-not-supported error", () => {
      expect(buildModelNotSupportedErrorContext(false)).toBe("");
    });

    it("returns template content when model-not-supported error and template exists", () => {
      const templateContent = "\n**🚫 Model Not Supported**: Test message.\n";
      fs.writeFileSync(path.join(promptsDir, "model_not_supported_error.md"), templateContent);
      const result = buildModelNotSupportedErrorContext(true);
      expect(result).toContain("Model Not Supported");
    });

    it("returns inline fallback message when template is missing", () => {
      // No template file written
      const result = buildModelNotSupportedErrorContext(true);
      expect(result).toContain("Model Not Supported");
      expect(result).toContain("gpt-5-mini");
    });
  });

  // buildMissingDataContext
  // ──────────────────────────────────────────────────────

  describe("buildMissingDataContext", () => {
    let buildMissingDataContext;
    const fs = require("fs");
    const path = require("path");
    const os = require("os");

    /** @type {string} */
    let tmpDir;

    /** @type {string} */
    let promptsDir;

    beforeEach(() => {
      vi.resetModules();
      tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "aw-test-missing-data-"));
      promptsDir = path.join(tmpDir, "gh-aw", "prompts");
      fs.mkdirSync(promptsDir, { recursive: true });
      process.env.RUNNER_TEMP = tmpDir;
      process.env.GH_AW_AGENT_OUTPUT = path.join(tmpDir, "agent_output.json");
      ({ buildMissingDataContext } = require("./handle_agent_failure.cjs"));
    });

    afterEach(() => {
      delete process.env.RUNNER_TEMP;
      delete process.env.GH_AW_AGENT_OUTPUT;
      if (fs.existsSync(tmpDir)) {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("returns empty string when agent output file does not exist", () => {
      expect(buildMissingDataContext(false)).toBe("");
      expect(buildMissingDataContext(true)).toBe("");
    });

    it("returns empty string when agent output has no missing_data items", () => {
      fs.writeFileSync(path.join(tmpDir, "agent_output.json"), JSON.stringify({ items: [{ type: "noop", reason: "done" }] }));
      vi.resetModules();
      ({ buildMissingDataContext } = require("./handle_agent_failure.cjs"));
      expect(buildMissingDataContext(false)).toBe("");
      expect(buildMissingDataContext(true)).toBe("");
    });

    it("returns missing data context without cache warning when cacheMemoryEnabled is false", () => {
      fs.writeFileSync(
        path.join(tmpDir, "agent_output.json"),
        JSON.stringify({
          items: [{ type: "missing_data", data_type: "cache_memory", reason: "cache_memory_miss" }],
        })
      );
      vi.resetModules();
      ({ buildMissingDataContext } = require("./handle_agent_failure.cjs"));
      const result = buildMissingDataContext(false);
      expect(result).toContain("Missing Data Reported");
      expect(result).toContain("cache\\_memory"); // data_type after markdown escaping
      expect(result).not.toContain("Cache Configuration Problem");
    });

    it("appends cache configuration warning when cacheMemoryEnabled is true and cache_memory_miss item present", () => {
      fs.writeFileSync(
        path.join(tmpDir, "agent_output.json"),
        JSON.stringify({
          items: [{ type: "missing_data", data_type: "cache_memory", reason: "cache_memory_miss" }],
        })
      );
      fs.writeFileSync(
        path.join(promptsDir, "cache_memory_miss.md"),
        "> [!WARNING]\n> **Cache Configuration Problem**: cache miss detected despite cache-memory being configured.\n\n<details>\n<summary>How to fix</summary>\n\nReview the [cache-memory configuration](https://github.github.com/gh-aw/reference/cache-memory/) and ensure the agent prompt correctly references files inside the cache directory.\n\n**File naming convention:** Cache files are stored at `/tmp/gh-aw/cache-memory/`.\n\n</details>\n"
      );
      vi.resetModules();
      ({ buildMissingDataContext } = require("./handle_agent_failure.cjs"));
      const result = buildMissingDataContext(true);
      expect(result).toContain("Missing Data Reported");
      expect(result).toContain("Cache Configuration Problem");
      expect(result).toContain("> [!WARNING]");
      expect(result).toContain("<details>");
      expect(result).toContain("/gh-aw/reference/cache-memory/");
      expect(result).toContain("File naming convention");
    });

    it("captures reason-only missing_data items (no data_type) and detects cache miss", () => {
      // Agents may emit missing_data with only reason (no data_type) — ensure it is still captured
      fs.writeFileSync(
        path.join(tmpDir, "agent_output.json"),
        JSON.stringify({
          items: [{ type: "missing_data", reason: "cache_memory_miss" }],
        })
      );
      fs.writeFileSync(
        path.join(promptsDir, "cache_memory_miss.md"),
        "> [!WARNING]\n> **Cache Configuration Problem**: cache miss detected despite cache-memory being configured.\n\n<details>\n<summary>How to fix</summary>\n\nDetails here.\n\n</details>\n"
      );
      vi.resetModules();
      ({ buildMissingDataContext } = require("./handle_agent_failure.cjs"));
      const result = buildMissingDataContext(true);
      expect(result).toContain("Missing Data Reported");
      expect(result).toContain("Cache Configuration Problem");
      expect(result).toContain("> [!WARNING]");
      expect(result).toContain("<details>");
    });

    it("does not append cache warning for unrelated missing_data reasons when cacheMemoryEnabled is true", () => {
      fs.writeFileSync(
        path.join(tmpDir, "agent_output.json"),
        JSON.stringify({
          items: [{ type: "missing_data", data_type: "user_data", reason: "not_provided" }],
        })
      );
      vi.resetModules();
      ({ buildMissingDataContext } = require("./handle_agent_failure.cjs"));
      const result = buildMissingDataContext(true);
      expect(result).toContain("Missing Data Reported");
      expect(result).not.toContain("Cache Configuration Problem");
    });
  });
});
