// @ts-check

import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { createRequire } from "module";

const require = createRequire(import.meta.url);

describe("handle_agent_failure", () => {
  let buildCodePushFailureContext;

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
    ({ buildCodePushFailureContext } = require("./handle_agent_failure.cjs"));
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
  });
});
