// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { createRequire } from "module";
import * as fs from "fs";
import * as path from "path";
import * as os from "os";

const require = createRequire(import.meta.url);

describe("create_pull_request - draft policy enforcement", () => {
  let tempDir;
  let originalEnv;

  beforeEach(() => {
    originalEnv = { ...process.env };
    process.env.GH_AW_WORKFLOW_ID = "test-workflow";
    process.env.GITHUB_REPOSITORY = "test-owner/test-repo";
    process.env.GITHUB_BASE_REF = "main";
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-draft-test-"));

    global.core = {
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
    global.github = {
      rest: {
        pulls: {
          create: vi.fn().mockResolvedValue({ data: { number: 1, html_url: "https://github.com/test" } }),
        },
        repos: {
          get: vi.fn().mockResolvedValue({ data: { default_branch: "main" } }),
        },
        issues: {
          addLabels: vi.fn().mockResolvedValue({}),
        },
      },
      graphql: vi.fn(),
    };
    global.context = {
      eventName: "workflow_dispatch",
      repo: { owner: "test-owner", repo: "test-repo" },
      payload: {},
    };
    global.exec = {
      exec: vi.fn().mockResolvedValue(0),
      getExecOutput: vi.fn().mockResolvedValue({ exitCode: 0, stdout: "", stderr: "" }),
    };

    // Clear module cache so globals are picked up fresh
    delete require.cache[require.resolve("./create_pull_request.cjs")];
  });

  afterEach(() => {
    for (const key of Object.keys(process.env)) {
      if (!(key in originalEnv)) {
        delete process.env[key];
      }
    }
    Object.assign(process.env, originalEnv);

    if (tempDir && fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }

    delete global.core;
    delete global.github;
    delete global.context;
    delete global.exec;
    vi.clearAllMocks();
  });

  /** Returns the `core.warning` calls related to draft config override attempts. */
  function getDraftOverrideWarnings() {
    return global.core.warning.mock.calls.filter(args => String(args[0]).includes("Agent requested draft"));
  }

  it("should enforce draft: false from config even when agent requests draft: true", async () => {
    const { main } = require("./create_pull_request.cjs");
    const handler = await main({ draft: "false", allow_empty: true });

    const result = await handler({ title: "Test PR", body: "Test body", draft: true }, {});

    expect(result.success).toBe(true);
    expect(global.github.rest.pulls.create).toHaveBeenCalledWith(expect.objectContaining({ draft: false }));
  });

  it("should enforce draft: true from config even when agent requests draft: false", async () => {
    const { main } = require("./create_pull_request.cjs");
    const handler = await main({ draft: "true", allow_empty: true });

    const result = await handler({ title: "Test PR", body: "Test body", draft: false }, {});

    expect(result.success).toBe(true);
    expect(global.github.rest.pulls.create).toHaveBeenCalledWith(expect.objectContaining({ draft: true }));
  });

  it("should log a warning when agent attempts to override draft config", async () => {
    const { main } = require("./create_pull_request.cjs");
    const handler = await main({ draft: "false", allow_empty: true });

    await handler({ title: "Test PR", body: "Test body", draft: true }, {});

    expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("Agent requested draft: true, but configuration enforces draft: false"));
  });

  it("should not log a warning when agent draft matches config", async () => {
    const { main } = require("./create_pull_request.cjs");
    const handler = await main({ draft: "false", allow_empty: true });

    await handler({ title: "Test PR", body: "Test body", draft: false }, {});

    expect(getDraftOverrideWarnings()).toHaveLength(0);
  });

  it("should not log a warning when agent does not specify draft", async () => {
    const { main } = require("./create_pull_request.cjs");
    const handler = await main({ draft: "false", allow_empty: true });

    await handler({ title: "Test PR", body: "Test body" }, {});

    expect(getDraftOverrideWarnings()).toHaveLength(0);
  });
});

describe("create_pull_request - fallback-as-issue configuration", () => {
  describe("configuration parsing", () => {
    it("should default fallback_as_issue to true when not specified", () => {
      const config = {};
      const fallbackAsIssue = config.fallback_as_issue !== false;

      expect(fallbackAsIssue).toBe(true);
    });

    it("should respect fallback_as_issue when set to false", () => {
      const config = { fallback_as_issue: false };
      const fallbackAsIssue = config.fallback_as_issue !== false;

      expect(fallbackAsIssue).toBe(false);
    });

    it("should respect fallback_as_issue when explicitly set to true", () => {
      const config = { fallback_as_issue: true };
      const fallbackAsIssue = config.fallback_as_issue !== false;

      expect(fallbackAsIssue).toBe(true);
    });
  });

  describe("error type documentation", () => {
    it("should document expected error types", () => {
      // This test documents the expected error types for different failure scenarios
      const errorTypes = {
        push_failed: "Used when git push operation fails and fallback-as-issue is false",
        pr_creation_failed: "Used when PR creation fails (except permission errors) and fallback-as-issue is false",
        permission_denied: "Used when GitHub Actions lacks permission to create/approve PRs AND fallback issue creation also fails",
      };

      // Verify the error types are documented
      expect(errorTypes.push_failed).toBeDefined();
      expect(errorTypes.pr_creation_failed).toBeDefined();
      expect(errorTypes.permission_denied).toBeDefined();

      // These error types should be returned in the corresponding code paths:
      // - push failure with fallback disabled: error_type: "push_failed"
      // - PR creation failure with fallback disabled: error_type: "pr_creation_failed"
      // - Permission error with successful fallback issue: success=true, fallback_used=true
      // - Permission error when fallback issue also fails: error_type: "permission_denied"
    });
  });
});

describe("create_pull_request - max limit enforcement", () => {
  let mockFs;

  beforeEach(() => {
    // Mock fs module for patch reading
    mockFs = {
      existsSync: vi.fn().mockReturnValue(true),
      readFileSync: vi.fn(),
    };
  });

  it("should enforce max file limit on patch content", () => {
    // Create a patch with more than MAX_FILES (100) files
    const patchLines = [];
    for (let i = 0; i < 101; i++) {
      patchLines.push(`diff --git a/file${i}.txt b/file${i}.txt`);
      patchLines.push("index 1234567..abcdefg 100644");
      patchLines.push("--- a/file${i}.txt");
      patchLines.push("+++ b/file${i}.txt");
      patchLines.push("@@ -1,1 +1,1 @@");
      patchLines.push("-old content");
      patchLines.push("+new content");
    }
    const patchContent = patchLines.join("\n");

    // Import the enforcement function
    const { enforcePullRequestLimits } = require("./create_pull_request.cjs");

    // Should throw E003 error
    expect(() => enforcePullRequestLimits(patchContent)).toThrow("E003");
    expect(() => enforcePullRequestLimits(patchContent)).toThrow("Cannot create pull request with more than 100 files");
    expect(() => enforcePullRequestLimits(patchContent)).toThrow("received 101");
  });

  it("should allow patches under the file limit", () => {
    // Create a patch with exactly MAX_FILES (100) files
    const patchLines = [];
    for (let i = 0; i < 100; i++) {
      patchLines.push(`diff --git a/file${i}.txt b/file${i}.txt`);
      patchLines.push("index 1234567..abcdefg 100644");
      patchLines.push("--- a/file${i}.txt");
      patchLines.push("+++ b/file${i}.txt");
      patchLines.push("@@ -1,1 +1,1 @@");
      patchLines.push("-old content");
      patchLines.push("+new content");
    }
    const patchContent = patchLines.join("\n");

    const { enforcePullRequestLimits } = require("./create_pull_request.cjs");

    // Should not throw
    expect(() => enforcePullRequestLimits(patchContent)).not.toThrow();
  });
});

describe("create_pull_request - security: branch name sanitization", () => {
  it("should sanitize branch names with shell metacharacters", () => {
    const { normalizeBranchName } = require("./normalize_branch_name.cjs");

    // Test shell injection attempts - forward slashes and dots are valid in git branch names
    const dangerousNames = [
      { input: "feature; rm -rf /", expected: "feature-rm-rf-/" },
      { input: "branch$(malicious)", expected: "branch-malicious" },
      { input: "branch`backdoor`", expected: "branch-backdoor" },
      { input: "branch| curl evil.com", expected: "branch-curl-evil.com" },
      { input: "branch && echo hacked", expected: "branch-echo-hacked" },
      { input: "branch || evil", expected: "branch-evil" },
      { input: "branch > /etc/passwd", expected: "branch-/etc/passwd" },
      { input: "branch < input.txt", expected: "branch-input.txt" },
      { input: "branch\x00null", expected: "branch-null" }, // Actual null byte, not escaped string
      { input: "branch\\x00null", expected: "branch-x00null" }, // Escaped string representation
    ];

    for (const { input, expected } of dangerousNames) {
      const result = normalizeBranchName(input);
      expect(result).toBe(expected);
      // Verify dangerous shell metacharacters are removed
      expect(result).not.toContain(";");
      expect(result).not.toContain("$");
      expect(result).not.toContain("`");
      expect(result).not.toContain("|");
      expect(result).not.toContain("&");
      expect(result).not.toContain(">");
      expect(result).not.toContain("<");
      expect(result).not.toContain("\x00"); // Actual null byte
      expect(result).not.toContain("\\x00"); // Escaped string
    }
  });

  it("should sanitize branch names with newlines and control characters", () => {
    const { normalizeBranchName } = require("./normalize_branch_name.cjs");

    const controlCharNames = [
      { input: "branch\nwith\nnewlines", expected: "branch-with-newlines" },
      { input: "branch\rwith\rcarriage", expected: "branch-with-carriage" },
      { input: "branch\twith\ttabs", expected: "branch-with-tabs" },
      { input: "branch\x1b[31mwith\x1b[0mescapes", expected: "branch-31mwith-0mescapes" },
    ];

    for (const { input, expected } of controlCharNames) {
      const result = normalizeBranchName(input);
      expect(result).toBe(expected);
      expect(result).not.toContain("\n");
      expect(result).not.toContain("\r");
      expect(result).not.toContain("\t");
      expect(result).not.toMatch(/\x1b/);
    }
  });

  it("should sanitize branch names with spaces and special characters", () => {
    const { normalizeBranchName } = require("./normalize_branch_name.cjs");

    const specialCharNames = [
      { input: "branch with spaces", expected: "branch-with-spaces" },
      { input: "branch!@#$%^&*()", expected: "branch" },
      { input: "branch[brackets]", expected: "branch-brackets" },
      { input: "branch{braces}", expected: "branch-braces" },
      { input: "branch:colon", expected: "branch-colon" },
      { input: 'branch"quotes"', expected: "branch-quotes" },
      { input: "branch'single'quotes", expected: "branch-single-quotes" },
    ];

    for (const { input, expected } of specialCharNames) {
      const result = normalizeBranchName(input);
      expect(result).toBe(expected);
    }
  });

  it("should preserve valid branch name characters", () => {
    const { normalizeBranchName } = require("./normalize_branch_name.cjs");

    const validNames = [
      { input: "feature/my-branch_v1.0", expected: "feature/my-branch_v1.0" },
      { input: "hotfix-123", expected: "hotfix-123" },
      { input: "release_v2.0.0", expected: "release_v2.0.0" },
    ];

    for (const { input, expected } of validNames) {
      const result = normalizeBranchName(input);
      expect(result).toBe(expected);
    }
  });

  it("should handle empty strings after sanitization", () => {
    const { normalizeBranchName } = require("./normalize_branch_name.cjs");

    // Branch names that become empty after sanitization
    const emptyAfterSanitization = ["!@#$%^&*()", ";;;", "|||", "---"];

    for (const input of emptyAfterSanitization) {
      const result = normalizeBranchName(input);
      expect(result).toBe("");
    }
  });

  it("should truncate long branch names to 128 characters", () => {
    const { normalizeBranchName } = require("./normalize_branch_name.cjs");

    const longBranchName = "a".repeat(200);
    const result = normalizeBranchName(longBranchName);
    expect(result.length).toBeLessThanOrEqual(128);
  });

  it("should collapse multiple dashes to single dash", () => {
    const { normalizeBranchName } = require("./normalize_branch_name.cjs");

    expect(normalizeBranchName("branch---with---dashes")).toBe("branch-with-dashes");
    expect(normalizeBranchName("branch  with  spaces")).toBe("branch-with-spaces");
  });

  it("should remove leading and trailing dashes", () => {
    const { normalizeBranchName } = require("./normalize_branch_name.cjs");

    expect(normalizeBranchName("---branch---")).toBe("branch");
    expect(normalizeBranchName("---")).toBe("");
  });

  it("should preserve original casing (no lowercase conversion)", () => {
    const { normalizeBranchName } = require("./normalize_branch_name.cjs");

    expect(normalizeBranchName("Feature/MyBranch")).toBe("Feature/MyBranch");
    expect(normalizeBranchName("UPPERCASE")).toBe("UPPERCASE");
    // Motivating use-case: Jira keys stay uppercase
    expect(normalizeBranchName("bugfix/BR-329-red")).toBe("bugfix/BR-329-red");
  });
});

// ──────────────────────────────────────────────────────
// normalizeBranchName: salt argument
// ──────────────────────────────────────────────────────

describe("create_pull_request - normalizeBranchName: salt argument", () => {
  it("should append salt suffix when salt argument is provided", () => {
    const { normalizeBranchName } = require("./normalize_branch_name.cjs");

    expect(normalizeBranchName("feature/my-branch", "abc123")).toBe("feature/my-branch-abc123");
    expect(normalizeBranchName("bugfix/BR-329-red", "cde2a954af3b8fa8")).toBe("bugfix/BR-329-red-cde2a954af3b8fa8");
  });

  it("should preserve original casing and add salt (default behaviour)", () => {
    const { normalizeBranchName } = require("./normalize_branch_name.cjs");

    // Default: preserve case + salt
    expect(normalizeBranchName("bugfix/BR-329-red", "cde2a954")).toBe("bugfix/BR-329-red-cde2a954");

    // preserve-branch-name=true: no salt
    expect(normalizeBranchName("bugfix/BR-329-red")).toBe("bugfix/BR-329-red");
  });

  it("should still replace shell metacharacters for security even when preserving case (CWE-78)", () => {
    const { normalizeBranchName } = require("./normalize_branch_name.cjs");

    const dangerousNames = [
      { input: "Feature; rm -rf /", expected: "Feature-rm-rf-/" },
      { input: "Branch$(malicious)", expected: "Branch-malicious" },
      { input: "BRANCH`backdoor`", expected: "BRANCH-backdoor" },
      { input: "Branch| curl EVIL.COM", expected: "Branch-curl-EVIL.COM" },
      { input: "Branch && echo HACKED", expected: "Branch-echo-HACKED" },
    ];

    for (const { input, expected } of dangerousNames) {
      const result = normalizeBranchName(input);
      expect(result).toBe(expected);
      expect(result).not.toContain(";");
      expect(result).not.toContain("$");
      expect(result).not.toContain("`");
      expect(result).not.toContain("|");
      expect(result).not.toContain("&");
    }
  });
});

// ──────────────────────────────────────────────────────
// allowed-files strict allowlist
// ──────────────────────────────────────────────────────

describe("create_pull_request - allowed-files strict allowlist", () => {
  let tempDir;
  let originalEnv;

  beforeEach(() => {
    originalEnv = { ...process.env };
    process.env.GH_AW_WORKFLOW_ID = "test-workflow";
    process.env.GITHUB_REPOSITORY = "test-owner/test-repo";
    process.env.GITHUB_BASE_REF = "main";
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-allowed-test-"));

    global.core = {
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
    global.github = {
      rest: {
        pulls: {
          create: vi.fn().mockResolvedValue({ data: { number: 1, html_url: "https://github.com/test" } }),
        },
        repos: {
          get: vi.fn().mockResolvedValue({ data: { default_branch: "main" } }),
        },
      },
      graphql: vi.fn(),
    };
    global.context = {
      eventName: "workflow_dispatch",
      repo: { owner: "test-owner", repo: "test-repo" },
      payload: {},
    };
    global.exec = {
      exec: vi.fn().mockResolvedValue(0),
      getExecOutput: vi.fn().mockResolvedValue({ exitCode: 0, stdout: "abc123\n", stderr: "" }),
    };

    // Clear module cache so globals are picked up fresh
    delete require.cache[require.resolve("./create_pull_request.cjs")];
  });

  afterEach(() => {
    for (const key of Object.keys(process.env)) {
      if (!(key in originalEnv)) {
        delete process.env[key];
      }
    }
    Object.assign(process.env, originalEnv);

    if (tempDir && fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }

    delete global.core;
    delete global.github;
    delete global.context;
    delete global.exec;
    vi.clearAllMocks();
  });

  /**
   * Creates a minimal git patch touching the given file paths.
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

  function writePatch(content) {
    const p = path.join(tempDir, "test.patch");
    fs.writeFileSync(p, content);
    return p;
  }

  it("should reject files outside the allowed-files allowlist", async () => {
    const patchPath = writePatch(createPatchWithFiles("src/index.js"));

    const { main } = require("./create_pull_request.cjs");
    const handler = await main({ allowed_files: [".github/aw/**"] });
    const result = await handler({ patch_path: patchPath, title: "Test PR", body: "" }, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("outside the allowed-files list");
    expect(result.error).toContain("src/index.js");
  });

  it("should reject a mixed patch where some files are outside the allowlist", async () => {
    const patchPath = writePatch(createPatchWithFiles(".github/aw/github-agentic-workflows.md", "src/index.js"));

    const { main } = require("./create_pull_request.cjs");
    const handler = await main({ allowed_files: [".github/aw/**"] });
    const result = await handler({ patch_path: patchPath, title: "Test PR", body: "" }, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("outside the allowed-files list");
    expect(result.error).toContain("src/index.js");
    expect(result.error).not.toContain(".github/aw/github-agentic-workflows.md");
  });

  it("should still enforce protected-files when allowed-files matches (orthogonal checks)", async () => {
    // allowed-files and protected-files are orthogonal: both checks must pass.
    // Matching the allowlist does NOT bypass the protected-files policy.
    const patchPath = writePatch(createPatchWithFiles(".github/aw/instructions.md"));

    const { main } = require("./create_pull_request.cjs");
    const handler = await main({
      allowed_files: [".github/aw/**"],
      protected_path_prefixes: [".github/"],
      protected_files_policy: "blocked",
    });
    const result = await handler({ patch_path: patchPath, title: "Test PR", body: "" }, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("protected files");
  });

  it("should allow a protected file when both allowed-files matches and protected-files: allowed is set", async () => {
    // Both checks are satisfied explicitly: allowlist scope + protected-files permission.
    const patchPath = writePatch(createPatchWithFiles(".github/aw/instructions.md"));

    const { main } = require("./create_pull_request.cjs");
    const handler = await main({
      allowed_files: [".github/aw/**"],
      protected_path_prefixes: [".github/"],
      protected_files_policy: "allowed",
    });
    const result = await handler({ patch_path: patchPath, title: "Test PR", body: "" }, {});

    // Should not be blocked by either check
    expect(result.error || "").not.toContain("protected files");
    expect(result.error || "").not.toContain("outside the allowed-files list");
  });

  it("should still enforce protected-files when allowed-files is not set", async () => {
    const patchPath = writePatch(createPatchWithFiles(".github/aw/instructions.md"));

    const { main } = require("./create_pull_request.cjs");
    const handler = await main({
      protected_path_prefixes: [".github/"],
      protected_files_policy: "blocked",
    });
    const result = await handler({ patch_path: patchPath, title: "Test PR", body: "" }, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("protected files");
  });
});

// excluded-files exclusion list
// ──────────────────────────────────────────────────────

describe("create_pull_request - excluded-files exclusion list", () => {
  let tempDir;
  let originalEnv;

  beforeEach(() => {
    originalEnv = { ...process.env };
    process.env.GH_AW_WORKFLOW_ID = "test-workflow";
    process.env.GITHUB_REPOSITORY = "test-owner/test-repo";
    process.env.GITHUB_BASE_REF = "main";
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-ignored-test-"));

    global.core = {
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
    global.github = {
      rest: {
        pulls: {
          create: vi.fn().mockResolvedValue({ data: { number: 1, html_url: "https://github.com/test" } }),
        },
        repos: {
          get: vi.fn().mockResolvedValue({ data: { default_branch: "main" } }),
        },
      },
      graphql: vi.fn(),
    };
    global.context = {
      eventName: "workflow_dispatch",
      repo: { owner: "test-owner", repo: "test-repo" },
      payload: {},
    };
    global.exec = {
      exec: vi.fn().mockResolvedValue(0),
      getExecOutput: vi.fn().mockResolvedValue({ exitCode: 0, stdout: "abc123\n", stderr: "" }),
    };

    // Clear module cache so globals are picked up fresh
    delete require.cache[require.resolve("./create_pull_request.cjs")];
  });

  afterEach(() => {
    for (const key of Object.keys(process.env)) {
      if (!(key in originalEnv)) {
        delete process.env[key];
      }
    }
    Object.assign(process.env, originalEnv);

    if (tempDir && fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }

    delete global.core;
    delete global.github;
    delete global.context;
    delete global.exec;
    vi.clearAllMocks();
  });

  /**
   * Creates a minimal git patch touching the given file paths.
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

  function writePatch(content) {
    const p = path.join(tempDir, "test.patch");
    fs.writeFileSync(p, content);
    return p;
  }

  it("should ignore files matching excluded-files patterns (not blocked by allowed-files)", async () => {
    // excluded-files are excluded at patch generation time via git :(exclude) pathspecs.
    // Simulate post-generation: the patch already contains only the non-ignored file.
    const patchPath = writePatch(createPatchWithFiles("src/index.js"));

    const { main } = require("./create_pull_request.cjs");
    const handler = await main({
      excluded_files: ["auto-generated/**"],
      allowed_files: ["src/**"],
    });
    const result = await handler({ patch_path: patchPath, title: "Test PR", body: "" }, {});

    expect(result.error || "").not.toContain("outside the allowed-files list");
  });

  it("should still block non-ignored files that violate the allowed-files list", async () => {
    const patchPath = writePatch(createPatchWithFiles("src/index.js", "other/file.txt"));

    const { main } = require("./create_pull_request.cjs");
    const handler = await main({
      excluded_files: ["auto-generated/**"],
      allowed_files: ["src/**"],
    });
    const result = await handler({ patch_path: patchPath, title: "Test PR", body: "" }, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("outside the allowed-files list");
    expect(result.error).toContain("other/file.txt");
    expect(result.error).not.toContain("src/index.js");
  });

  it("should ignore files matching excluded-files patterns (not blocked by protected-files)", async () => {
    // excluded-files are excluded at patch generation time via git :(exclude) pathspecs.
    // Simulate post-generation: the patch already contains only the non-ignored file.
    const patchPath = writePatch(createPatchWithFiles("src/index.js"));

    const { main } = require("./create_pull_request.cjs");
    const handler = await main({
      excluded_files: ["package.json"],
      protected_files: ["package.json"],
      protected_files_policy: "blocked",
    });
    const result = await handler({ patch_path: patchPath, title: "Test PR", body: "" }, {});

    expect(result.error || "").not.toContain("protected files");
  });

  it("should allow when all patch files are ignored (even with allowed-files set)", async () => {
    // excluded-files are excluded at patch generation time via git :(exclude) pathspecs.
    // Simulate post-generation: all files were excluded so the patch file is absent.
    const { main } = require("./create_pull_request.cjs");
    const handler = await main({
      excluded_files: ["dist/**"],
      allowed_files: ["src/**"],
    });
    // No patch file — simulates all changes being ignored at generation time
    const result = await handler({ patch_path: path.join(tempDir, "nonexistent.patch"), title: "Test PR", body: "" }, {});

    // No patch → treated as no changes, not an allowlist violation
    expect(result.error || "").not.toContain("outside the allowed-files list");
  });
});
