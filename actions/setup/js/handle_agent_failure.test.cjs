// @ts-check

import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { createRequire } from "module";

const require = createRequire(import.meta.url);

describe("handle_agent_failure", () => {
  let buildCodePushFailureContext;
  let buildPushRepoMemoryFailureContext;

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
    ({ buildCodePushFailureContext, buildPushRepoMemoryFailureContext } = require("./handle_agent_failure.cjs"));
  });

  afterEach(() => {
    delete global.core;
    delete global.github;
    delete global.context;
    delete process.env.GITHUB_SHA;
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
      expect(result).toContain("agent-artifacts");
      expect(result).toContain("git am --3way");
      expect(result).toContain(runUrl);
    });

    it("shows generic download instructions when runUrl is not provided", () => {
      const errors = "create_pull_request:Failed to apply patch";
      const result = buildCodePushFailureContext(errors);
      expect(result).toContain("🔀 Patch Apply Failed");
      expect(result).toContain("git am --3way");
      // No specific run ID in instructions
      expect(result).not.toContain("gh run download");
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
});
