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

describe("create_pull_request - auto-close-issue configuration", () => {
  let tempDir;
  let originalEnv;

  beforeEach(() => {
    originalEnv = { ...process.env };
    process.env.GH_AW_WORKFLOW_ID = "test-workflow";
    process.env.GITHUB_REPOSITORY = "test-owner/test-repo";
    process.env.GITHUB_BASE_REF = "main";
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-auto-close-test-"));

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
      eventName: "issues",
      repo: { owner: "test-owner", repo: "test-repo" },
      payload: {
        issue: { number: 42 },
      },
    };
    global.exec = {
      exec: vi.fn().mockResolvedValue(0),
      getExecOutput: vi.fn().mockResolvedValue({ exitCode: 0, stdout: "", stderr: "" }),
    };

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

  it("should auto-add 'Fixes #N' when triggered from an issue and auto_close_issue is not set (default)", async () => {
    const { main } = require("./create_pull_request.cjs");
    const handler = await main({ allow_empty: true });

    await handler({ title: "Test PR", body: "Test body" }, {});

    const createCall = global.github.rest.pulls.create.mock.calls[0]?.[0];
    expect(createCall?.body).toContain("Fixes #42");
    expect(global.core.info).toHaveBeenCalledWith(expect.stringContaining('Auto-added "Fixes #42"'));
  });

  it("should auto-add 'Fixes #N' when triggered from an issue and auto_close_issue is explicitly true", async () => {
    const { main } = require("./create_pull_request.cjs");
    const handler = await main({ allow_empty: true, auto_close_issue: true });

    await handler({ title: "Test PR", body: "Test body" }, {});

    const createCall = global.github.rest.pulls.create.mock.calls[0]?.[0];
    expect(createCall?.body).toContain("Fixes #42");
    expect(global.core.info).toHaveBeenCalledWith(expect.stringContaining('Auto-added "Fixes #42"'));
  });

  it("should NOT auto-add 'Fixes #N' when auto_close_issue is false", async () => {
    const { main } = require("./create_pull_request.cjs");
    const handler = await main({ allow_empty: true, auto_close_issue: false });

    await handler({ title: "Test PR", body: "Test body" }, {});

    const createCall = global.github.rest.pulls.create.mock.calls[0]?.[0];
    expect(createCall?.body).not.toContain("Fixes #42");
    expect(global.core.info).toHaveBeenCalledWith(expect.stringContaining("Skipping auto-close keyword for #42 (auto-close-issue: false)"));
  });

  it("should NOT auto-add 'Fixes #N' when body already contains a closing keyword, regardless of auto_close_issue", async () => {
    const { main } = require("./create_pull_request.cjs");
    const handler = await main({ allow_empty: true });

    await handler({ title: "Test PR", body: "Test body\n\nCloses #42" }, {});

    const createCall = global.github.rest.pulls.create.mock.calls[0]?.[0];
    // Should not duplicate the keyword
    const fixesCount = (createCall?.body?.match(/Fixes #42/gi) || []).length;
    const closesCount = (createCall?.body?.match(/Closes #42/gi) || []).length;
    expect(closesCount).toBe(1);
    expect(fixesCount).toBe(0);
  });

  it("should have no effect when not triggered from an issue, regardless of auto_close_issue value", async () => {
    // Override context to not be from an issue
    global.context = {
      eventName: "workflow_dispatch",
      repo: { owner: "test-owner", repo: "test-repo" },
      payload: {},
    };
    delete require.cache[require.resolve("./create_pull_request.cjs")];

    const { main } = require("./create_pull_request.cjs");
    const handler = await main({ allow_empty: true, auto_close_issue: true });

    await handler({ title: "Test PR", body: "Test body" }, {});

    const createCall = global.github.rest.pulls.create.mock.calls[0]?.[0];
    expect(createCall?.body).not.toContain("Fixes #");
  });

  it("should NOT add 'Fixes #N' when auto_close_issue is false even if body has no closing keyword", async () => {
    const { main } = require("./create_pull_request.cjs");
    const handler = await main({ allow_empty: true, auto_close_issue: false });

    await handler({ title: "Test PR", body: "Investigation findings - partial work only" }, {});

    const createCall = global.github.rest.pulls.create.mock.calls[0]?.[0];
    expect(createCall?.body).not.toContain("Fixes #");
    expect(createCall?.body).not.toContain("Closes #");
    expect(createCall?.body).not.toContain("Resolves #");
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
    // Create a patch with more than MAX_FILES (100) unique files
    const patchLines = [];
    for (let i = 0; i < 101; i++) {
      patchLines.push(`diff --git a/file${i}.txt b/file${i}.txt`);
      patchLines.push("index 1234567..abcdefg 100644");
      patchLines.push(`--- a/file${i}.txt`);
      patchLines.push(`+++ b/file${i}.txt`);
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
      patchLines.push(`--- a/file${i}.txt`);
      patchLines.push(`+++ b/file${i}.txt`);
      patchLines.push("@@ -1,1 +1,1 @@");
      patchLines.push("-old content");
      patchLines.push("+new content");
    }
    const patchContent = patchLines.join("\n");

    const { enforcePullRequestLimits } = require("./create_pull_request.cjs");

    // Should not throw
    expect(() => enforcePullRequestLimits(patchContent)).not.toThrow();
  });

  it("should count unique files across multi-commit patches (regression: long-running branch with multi-commit patches)", () => {
    // Simulate `git format-patch` output where the same files are modified across
    // multiple commits. Previously the limit check counted every `diff --git`
    // header (3 commits * 2 files = 6) instead of the 2 unique files actually
    // touched. After the fix it should count 2.
    const { countUniquePatchFiles, enforcePullRequestLimits } = require("./create_pull_request.cjs");

    const patchLines = [];
    for (let commit = 0; commit < 3; commit++) {
      for (const file of ["src/a.ts", "src/b.ts"]) {
        patchLines.push(`diff --git a/${file} b/${file}`);
        patchLines.push("index 1234567..abcdefg 100644");
        patchLines.push(`--- a/${file}`);
        patchLines.push(`+++ b/${file}`);
        patchLines.push("@@ -1,1 +1,1 @@");
        patchLines.push(`-old ${commit}`);
        patchLines.push(`+new ${commit}`);
      }
    }
    const patchContent = patchLines.join("\n");

    expect(countUniquePatchFiles(patchContent)).toBe(2);
    expect(() => enforcePullRequestLimits(patchContent, 5)).not.toThrow();
  });

  it("should accept a configurable max-files override", () => {
    const { enforcePullRequestLimits } = require("./create_pull_request.cjs");

    const patchLines = [];
    for (let i = 0; i < 150; i++) {
      patchLines.push(`diff --git a/file${i}.txt b/file${i}.txt`);
      patchLines.push("index 1234567..abcdefg 100644");
      patchLines.push(`--- a/file${i}.txt`);
      patchLines.push(`+++ b/file${i}.txt`);
      patchLines.push("@@ -1,1 +1,1 @@");
      patchLines.push("-old content");
      patchLines.push("+new content");
    }
    const patchContent = patchLines.join("\n");

    // With default limit (100) it should fail
    expect(() => enforcePullRequestLimits(patchContent)).toThrow("E003");
    // With raised limit it should pass
    expect(() => enforcePullRequestLimits(patchContent, 200)).not.toThrow();
    // With smaller limit, error message reflects override
    expect(() => enforcePullRequestLimits(patchContent, 50)).toThrow("more than 50 files");
    expect(() => enforcePullRequestLimits(patchContent, 50)).toThrow("received 150");
  });

  it("should return 0 for empty patches", () => {
    const { countUniquePatchFiles, enforcePullRequestLimits } = require("./create_pull_request.cjs");
    expect(countUniquePatchFiles("")).toBe(0);
    expect(countUniquePatchFiles("   \n\n")).toBe(0);
    expect(() => enforcePullRequestLimits("")).not.toThrow();
  });

  it("should handle quoted paths with C-style escapes", () => {
    // git emits quoted, escaped headers when filenames contain special chars
    // (e.g. embedded quotes or backslashes). The parser must treat these as
    // distinct unique files and never undercount.
    const { countUniquePatchFiles, parseDiffGitHeader } = require("./create_pull_request.cjs");

    // Embedded escaped quote: "a/foo\"bar" "b/foo\"bar"
    expect(parseDiffGitHeader('diff --git "a/foo\\"bar" "b/foo\\"bar"')).toBe('foo\\"bar');
    // Embedded backslash: "a/foo\\bar" "b/foo\\bar"
    expect(parseDiffGitHeader('diff --git "a/foo\\\\bar" "b/foo\\\\bar"')).toBe("foo\\\\bar");
    // Plain unquoted form
    expect(parseDiffGitHeader("diff --git a/foo.txt b/foo.txt")).toBe("foo.txt");
    // Path with spaces (git always emits quoted form when path contains spaces)
    expect(parseDiffGitHeader('diff --git "a/dir/with space/x" "b/dir/with space/x"')).toBe("dir/with space/x");

    // A patch with three different quoted/escaped files should count as 3.
    const patch = [
      'diff --git "a/foo\\"bar" "b/foo\\"bar"',
      "index 1234567..abcdefg 100644",
      "--- a/x",
      "+++ b/x",
      "@@ -1 +1 @@",
      "-old",
      "+new",
      'diff --git "a/foo\\\\bar" "b/foo\\\\bar"',
      "index 1234567..abcdefg 100644",
      "--- a/x",
      "+++ b/x",
      "@@ -1 +1 @@",
      "-old",
      "+new",
      "diff --git a/normal.txt b/normal.txt",
      "index 1234567..abcdefg 100644",
      "--- a/x",
      "+++ b/x",
      "@@ -1 +1 @@",
      "-old",
      "+new",
    ].join("\n");
    expect(countUniquePatchFiles(patch)).toBe(3);
  });

  it("should count unparseable headers conservatively (never undercount)", () => {
    // Each `diff --git` header that cannot be parsed contributes one entry
    // to the unique-file set, so a malformed header can never silently
    // bypass the limit.
    const { countUniquePatchFiles, enforcePullRequestLimits } = require("./create_pull_request.cjs");
    const patchContent = "diff --git \ndiff --git \ndiff --git \n";
    expect(countUniquePatchFiles(patchContent)).toBe(3);

    // Mixed: 2 parseable + 2 unparseable = 4 unique entries.
    const mixed = ["diff --git a/a.txt b/a.txt", "diff --git ", "diff --git b/b.txt c/b.txt", "diff --git "].join("\n");
    expect(countUniquePatchFiles(mixed)).toBe(4);

    // 200 unparseable headers must still trigger the default 100-file limit.
    const lots = Array.from({ length: 200 }, () => "diff --git ").join("\n");
    expect(() => enforcePullRequestLimits(lots)).toThrow("E003");
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

describe("create_pull_request - configured reviewers", () => {
  let tempDir;
  let originalEnv;

  beforeEach(() => {
    originalEnv = { ...process.env };
    process.env.GH_AW_WORKFLOW_ID = "test-workflow";
    process.env.GITHUB_REPOSITORY = "test-owner/test-repo";
    process.env.GITHUB_BASE_REF = "main";
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-reviewer-test-"));

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
          create: vi.fn().mockResolvedValue({ data: { number: 42, html_url: "https://github.com/test/pull/42", node_id: "PR_42" } }),
          requestReviewers: vi.fn().mockResolvedValue({}),
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

  it("should request configured reviewers after creating the PR", async () => {
    const { main } = require("./create_pull_request.cjs");
    const handler = await main({ reviewers: ["user1", "user2"], allow_empty: true });

    const result = await handler({ title: "Test PR", body: "Test body" }, {});

    expect(result.success).toBe(true);
    expect(global.github.rest.pulls.requestReviewers).toHaveBeenCalledWith(
      expect.objectContaining({
        owner: "test-owner",
        repo: "test-repo",
        pull_number: 42,
        reviewers: ["user1", "user2"],
      })
    );
  });

  it("should request configured team reviewers after creating the PR", async () => {
    const { main } = require("./create_pull_request.cjs");
    const handler = await main({ team_reviewers: ["platform-team"], allow_empty: true });

    const result = await handler({ title: "Test PR", body: "Test body" }, {});

    expect(result.success).toBe(true);
    expect(global.github.rest.pulls.requestReviewers).toHaveBeenCalledWith(
      expect.objectContaining({
        owner: "test-owner",
        repo: "test-repo",
        pull_number: 42,
        reviewers: [],
        team_reviewers: ["platform-team"],
      })
    );
  });

  it("should handle copilot reviewer separately from regular reviewers", async () => {
    const { main } = require("./create_pull_request.cjs");
    const handler = await main({ reviewers: ["user1", "copilot"], allow_empty: true });

    const result = await handler({ title: "Test PR", body: "Test body" }, {});

    expect(result.success).toBe(true);
    // Should be called twice: once for regular reviewers, once for copilot bot
    expect(global.github.rest.pulls.requestReviewers).toHaveBeenCalledTimes(2);
    expect(global.github.rest.pulls.requestReviewers).toHaveBeenCalledWith(expect.objectContaining({ reviewers: ["user1"] }));
    expect(global.github.rest.pulls.requestReviewers).toHaveBeenCalledWith(expect.objectContaining({ reviewers: ["copilot-pull-request-reviewer[bot]"] }));
  });

  it("should keep configured team reviewers with non-copilot reviewers when copilot is configured", async () => {
    const { main } = require("./create_pull_request.cjs");
    const handler = await main({ reviewers: ["user1", "copilot"], team_reviewers: ["platform-team"], allow_empty: true });

    const result = await handler({ title: "Test PR", body: "Test body" }, {});

    expect(result.success).toBe(true);
    expect(global.github.rest.pulls.requestReviewers).toHaveBeenCalledTimes(2);
    expect(global.github.rest.pulls.requestReviewers).toHaveBeenNthCalledWith(1, expect.objectContaining({ reviewers: ["user1"], team_reviewers: ["platform-team"] }));
    expect(global.github.rest.pulls.requestReviewers).toHaveBeenNthCalledWith(2, expect.objectContaining({ reviewers: ["copilot-pull-request-reviewer[bot]"] }));
  });

  it("should not call requestReviewers when no reviewers are configured", async () => {
    const { main } = require("./create_pull_request.cjs");
    const handler = await main({ allow_empty: true });

    const result = await handler({ title: "Test PR", body: "Test body" }, {});

    expect(result.success).toBe(true);
    expect(global.github.rest.pulls.requestReviewers).not.toHaveBeenCalled();
  });

  it("should continue successfully even if requestReviewers fails", async () => {
    global.github.rest.pulls.requestReviewers.mockRejectedValue(new Error("API error"));

    const { main } = require("./create_pull_request.cjs");
    const handler = await main({ reviewers: ["user1"], allow_empty: true });

    const result = await handler({ title: "Test PR", body: "Test body" }, {});

    expect(result.success).toBe(true);
    expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("Failed to request reviewers"));
  });

  it("should retry addLabels on race condition and warn after all retries exhausted", async () => {
    // GitHub API transiently fails to resolve the PR node ID immediately after creation.
    // withRetry retries 5 times (6 total calls); after exhaustion it should warn but NOT fall back to an issue.
    vi.useFakeTimers();
    try {
      global.github.rest.issues.addLabels.mockRejectedValue(new Error("Validation Failed: Could not resolve to a node with the global id of 'PR_kwDOPc1QR87OOJzM'."));

      const { main } = require("./create_pull_request.cjs");
      const handler = await main({ labels: ["automation"], allow_empty: true });

      const resultPromise = handler({ title: "Test PR", body: "Test body", labels: ["automation"] }, {});

      // Advance all fake timers to skip the retry delays (6s, 12s, 24s, 30s, 30s)
      await vi.runAllTimersAsync();

      const result = await resultPromise;

      expect(result.success).toBe(true);
      expect(result.fallback_used).toBeUndefined();
      // addLabels called once initially + 5 retries = 6 total
      expect(global.github.rest.issues.addLabels).toHaveBeenCalledTimes(6);
      expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("Failed to add labels to PR #42"));
    } finally {
      vi.useRealTimers();
    }
  });

  it("should succeed when addLabels recovers on a retry", async () => {
    // Simulates a transient race condition that resolves on the second attempt.
    vi.useFakeTimers();
    try {
      global.github.rest.issues.addLabels.mockRejectedValueOnce(new Error("Validation Failed: Could not resolve to a node with the global id of 'PR_kwDOPc1QR87OOJzM'.")).mockResolvedValue({});

      const { main } = require("./create_pull_request.cjs");
      const handler = await main({ labels: ["automation"], allow_empty: true });

      const resultPromise = handler({ title: "Test PR", body: "Test body", labels: ["automation"] }, {});

      await vi.runAllTimersAsync();

      const result = await resultPromise;

      expect(result.success).toBe(true);
      // addLabels called twice: first attempt fails, second succeeds
      expect(global.github.rest.issues.addLabels).toHaveBeenCalledTimes(2);
      // No warning about final failure — the retry succeeded
      expect(global.core.warning).not.toHaveBeenCalledWith(expect.stringContaining("Failed to add labels to PR #42"));
    } finally {
      vi.useRealTimers();
    }
  });

  it("should not retry addLabels for non-transient errors", async () => {
    // Non-transient errors (e.g., 404 label not found) should not be retried.
    global.github.rest.issues.addLabels.mockRejectedValue(new Error("Validation Failed: label does not exist"));

    const { main } = require("./create_pull_request.cjs");
    const handler = await main({ labels: ["nonexistent"], allow_empty: true });

    const result = await handler({ title: "Test PR", body: "Test body", labels: ["nonexistent"] }, {});

    expect(result.success).toBe(true);
    // No retry — called only once since the error is non-transient
    expect(global.github.rest.issues.addLabels).toHaveBeenCalledTimes(1);
    expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("Failed to add labels to PR #42"));
  });

  it("should accept reviewers as a comma-separated string", async () => {
    const { main } = require("./create_pull_request.cjs");
    const handler = await main({ reviewers: "user1,user2", allow_empty: true });

    const result = await handler({ title: "Test PR", body: "Test body" }, {});

    expect(result.success).toBe(true);
    expect(global.github.rest.pulls.requestReviewers).toHaveBeenCalledWith(expect.objectContaining({ reviewers: ["user1", "user2"] }));
  });
});

describe("create_pull_request - wildcard target-repo", () => {
  let tempDir;
  let originalEnv;

  beforeEach(() => {
    originalEnv = { ...process.env };
    process.env.GH_AW_WORKFLOW_ID = "test-workflow";
    process.env.GITHUB_REPOSITORY = "test-owner/test-repo";
    process.env.GITHUB_BASE_REF = "main";
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-wildcard-test-"));

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
          create: vi.fn().mockResolvedValue({ data: { number: 99, html_url: "https://github.com/any-org/any-repo/pull/99", node_id: "PR_99" } }),
          requestReviewers: vi.fn().mockResolvedValue({}),
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

  it('should create PR in any repo when target-repo is "*"', async () => {
    const { main } = require("./create_pull_request.cjs");
    const handler = await main({ "target-repo": "*", allow_empty: true });

    const result = await handler(
      {
        title: "Test PR",
        body: "Test body",
        repo: "any-org/any-repo",
      },
      {}
    );

    expect(result.success).toBe(true);
    expect(global.github.rest.pulls.create).toHaveBeenCalledWith(
      expect.objectContaining({
        owner: "any-org",
        repo: "any-repo",
      })
    );
  });

  it('should reject invalid repo slug when target-repo is "*"', async () => {
    const { main } = require("./create_pull_request.cjs");
    const handler = await main({ "target-repo": "*", allow_empty: true });

    const result = await handler(
      {
        title: "Test PR",
        body: "Test body",
        repo: "not-a-valid-slug",
      },
      {}
    );

    expect(result.success).toBe(false);
  });
});

describe("create_pull_request - base branch override policy", () => {
  let tempDir;
  let originalEnv;

  beforeEach(() => {
    originalEnv = { ...process.env };
    process.env.GH_AW_WORKFLOW_ID = "test-workflow";
    process.env.GITHUB_REPOSITORY = "test-owner/test-repo";
    process.env.GITHUB_BASE_REF = "main";
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-base-override-test-"));

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
          create: vi.fn().mockResolvedValue({ data: { number: 100, html_url: "https://github.com/test-owner/test-repo/pull/100", node_id: "PR_100" } }),
          requestReviewers: vi.fn().mockResolvedValue({}),
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

  it("should reject base override when allowed-base-branches is not configured", async () => {
    const { main } = require("./create_pull_request.cjs");
    const handler = await main({ allow_empty: true });

    const result = await handler({ title: "Test PR", body: "Test body", base: "release/1.0" }, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("Base branch override is not allowed");
    expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("allowed-base-branches is not configured"));
  });

  it("should allow base override when it matches allowed-base-branches", async () => {
    const { main } = require("./create_pull_request.cjs");
    const handler = await main({ allow_empty: true, allowed_base_branches: ["release/*", "main"] });

    const result = await handler({ title: "Test PR", body: "Test body", base: "release/1.0" }, {});

    expect(result.success).toBe(true);
    expect(global.github.rest.pulls.create).toHaveBeenCalledWith(expect.objectContaining({ base: "release/1.0" }));
    expect(global.core.info).toHaveBeenCalledWith(expect.stringContaining('Base branch override requested: "release/1.0"'));
    expect(global.core.info).toHaveBeenCalledWith(expect.stringContaining('Base branch override accepted: "release/1.0"'));
    expect(global.core.info).toHaveBeenCalledWith(expect.stringContaining("Using agent-provided base branch override: release/1.0"));
  });

  it("should reject base override when it does not match allowed-base-branches", async () => {
    const { main } = require("./create_pull_request.cjs");
    const handler = await main({ allow_empty: true, allowed_base_branches: ["release/*"] });

    const result = await handler({ title: "Test PR", body: "Test body", base: "main" }, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("Base branch override 'main' is not allowed");
    expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("does not match allowed patterns"));
  });
});

describe("create_pull_request - patch apply fallback to original base commit", () => {
  let tempDir;
  let originalEnv;
  let patchFilePath;

  const MOCK_BASE_COMMIT_SHA = "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef";
  // Minimal valid format-patch output
  const PATCH_CONTENT =
    `From a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2 Mon Sep 17 00:00:00 2001\n` +
    `From: Test Author <test@example.com>\n` +
    `Date: Wed, 26 Mar 2026 12:00:00 +0000\n` +
    `Subject: [PATCH] Test change\n\n` +
    `---\n` +
    ` file.txt | 1 +\n\n` +
    `diff --git a/file.txt b/file.txt\n` +
    `index 1234567..abcdefg 100644\n` +
    `--- a/file.txt\n` +
    `+++ b/file.txt\n` +
    `@@ -1 +1,2 @@\n` +
    ` existing content\n` +
    `+new content\n` +
    `--\n` +
    `2.39.0\n`;

  beforeEach(() => {
    originalEnv = { ...process.env };
    process.env.GH_AW_WORKFLOW_ID = "test-workflow";
    process.env.GITHUB_REPOSITORY = "test-owner/test-repo";
    process.env.GITHUB_BASE_REF = "main";

    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-fallback-test-"));
    patchFilePath = path.join(tempDir, "test.patch");
    fs.writeFileSync(patchFilePath, PATCH_CONTENT, "utf8");

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
          create: vi.fn().mockResolvedValue({ data: { number: 42, html_url: "https://github.com/test/pull/42", node_id: "PR_42" } }),
          requestReviewers: vi.fn().mockResolvedValue({}),
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
   * Helper to detect git am calls in both formats:
   * - exec("git", ["am", "--3way", path])  (array form)
   * - exec("git am --3way /path")          (string form)
   */
  function isGitAmCall(cmd, args) {
    if (cmd === "git" && Array.isArray(args) && args[0] === "am") return true;
    if (typeof cmd === "string" && cmd.startsWith("git am")) return true;
    return false;
  }

  function isGitAmAbort(cmd, args) {
    if (cmd === "git" && Array.isArray(args) && args[0] === "am" && args.includes("--abort")) return true;
    if (typeof cmd === "string" && cmd.includes("am --abort")) return true;
    return false;
  }

  function isGitAm3Way(cmd, args) {
    if (cmd === "git" && Array.isArray(args) && args[0] === "am" && args.includes("--3way")) return true;
    if (typeof cmd === "string" && cmd.startsWith("git am --3way")) return true;
    return false;
  }

  it("should fall back to original base commit when git am --3way fails with merge conflicts", async () => {
    let primaryAmAttempted = false;
    global.exec = {
      exec: vi.fn().mockImplementation((cmd, args) => {
        // Fail the first "git am --3way" call to simulate a merge conflict
        if (isGitAm3Way(cmd, args) && !primaryAmAttempted) {
          primaryAmAttempted = true;
          throw new Error("CONFLICT (content): Merge conflict in file.txt");
        }
        return Promise.resolve(0);
      }),
      getExecOutput: vi.fn().mockImplementation((cmd, args) => {
        return Promise.resolve({ exitCode: 0, stdout: "", stderr: "" });
      }),
    };

    const { main } = require("./create_pull_request.cjs");
    const handler = await main({});

    const result = await handler({ title: "Test PR", body: "Test body", patch_path: patchFilePath, branch: "test-branch", base_commit: MOCK_BASE_COMMIT_SHA }, {});

    expect(result.success).toBe(true);
    // Should warn that the PR will show merge conflicts
    expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("merge conflicts"));
  });

  it("should return error when both git am --3way and the fallback git am fail", async () => {
    global.exec = {
      exec: vi.fn().mockImplementation((cmd, args) => {
        // Fail all git am calls except git am --abort
        if (isGitAmCall(cmd, args) && !isGitAmAbort(cmd, args)) {
          throw new Error("CONFLICT (content): Merge conflict in file.txt");
        }
        return Promise.resolve(0);
      }),
      getExecOutput: vi.fn().mockImplementation((cmd, args) => {
        return Promise.resolve({ exitCode: 0, stdout: "", stderr: "" });
      }),
    };

    const { main } = require("./create_pull_request.cjs");
    const handler = await main({});

    const result = await handler({ title: "Test PR", body: "Test body", patch_path: patchFilePath, branch: "test-branch", base_commit: MOCK_BASE_COMMIT_SHA }, {});

    expect(result.success).toBe(false);
    expect(result.error).toBe("Failed to apply patch");
  });

  it("should return error when original base commit is not available (cross-repo scenario)", async () => {
    global.exec = {
      exec: vi.fn().mockImplementation((cmd, args) => {
        // Fail git am --3way
        if (isGitAm3Way(cmd, args)) {
          throw new Error("CONFLICT (content): Merge conflict in file.txt");
        }
        // Fail git cat-file to simulate commit not present in local repo
        if (cmd === "git" && Array.isArray(args) && args[0] === "cat-file") {
          throw new Error("Not a valid object name");
        }
        return Promise.resolve(0);
      }),
      getExecOutput: vi.fn().mockImplementation((cmd, args) => {
        return Promise.resolve({ exitCode: 0, stdout: "", stderr: "" });
      }),
    };

    const { main } = require("./create_pull_request.cjs");
    const handler = await main({});

    const result = await handler({ title: "Test PR", body: "Test body", patch_path: patchFilePath, branch: "test-branch", base_commit: MOCK_BASE_COMMIT_SHA }, {});

    expect(result.success).toBe(false);
    expect(result.error).toBe("Failed to apply patch");
  });

  it("should return error when no base_commit is provided and git am --3way fails", async () => {
    global.exec = {
      exec: vi.fn().mockImplementation((cmd, args) => {
        if (isGitAm3Way(cmd, args)) {
          throw new Error("CONFLICT (content): Merge conflict in file.txt");
        }
        return Promise.resolve(0);
      }),
      getExecOutput: vi.fn().mockImplementation(() => {
        return Promise.resolve({ exitCode: 0, stdout: "", stderr: "" });
      }),
    };

    const { main } = require("./create_pull_request.cjs");
    const handler = await main({});

    // No base_commit provided - fallback should not be possible
    const result = await handler({ title: "Test PR", body: "Test body", patch_path: patchFilePath, branch: "test-branch" }, {});

    expect(result.success).toBe(false);
    expect(result.error).toBe("Failed to apply patch");
    expect(global.core.warning).toHaveBeenCalledWith("No base_commit recorded in safe output entry - fallback not possible");
  });

  it("should fail loudly when preserve-branch-name is true and remote branch already exists", async () => {
    // Simulate the remote branch existing (ls-remote returns content)
    global.exec = {
      exec: vi.fn().mockResolvedValue(0),
      getExecOutput: vi.fn().mockImplementation((cmd, args) => {
        const cmdStr = typeof cmd === "string" ? cmd : `${cmd} ${(args || []).join(" ")}`;
        if (cmdStr.includes("ls-remote --heads origin")) {
          return Promise.resolve({ exitCode: 0, stdout: "abc123\trefs/heads/preserve-me\n", stderr: "" });
        }
        return Promise.resolve({ exitCode: 0, stdout: "", stderr: "" });
      }),
    };

    const { main } = require("./create_pull_request.cjs");
    const handler = await main({ preserve_branch_name: true, fallback_as_issue: false });

    const result = await handler({ title: "Test PR", body: "Test body", patch_path: patchFilePath, branch: "preserve-me", base_commit: MOCK_BASE_COMMIT_SHA }, {});

    expect(result.success).toBe(false);
    expect(result.error_type).toBe("push_failed");
    expect(result.error).toContain('Remote branch "preserve-me" already exists');
    expect(result.error).toContain("preserve-branch-name is enabled");
    // Critical: should NOT have warned about appending random suffix (silent bypass)
    const warningCalls = global.core.warning.mock.calls.map(call => String(call[0]));
    expect(warningCalls.some(msg => msg.includes("appending random suffix"))).toBe(false);
  });

  it("should append random suffix when preserve-branch-name is false and remote branch already exists", async () => {
    let renameCalled = false;
    global.exec = {
      exec: vi.fn().mockImplementation((cmd, args) => {
        const cmdStr = typeof cmd === "string" ? cmd : `${cmd} ${(args || []).join(" ")}`;
        if (cmdStr.includes("git branch -m")) {
          renameCalled = true;
        }
        return Promise.resolve(0);
      }),
      getExecOutput: vi.fn().mockImplementation((cmd, args) => {
        const cmdStr = typeof cmd === "string" ? cmd : `${cmd} ${(args || []).join(" ")}`;
        if (cmdStr.includes("ls-remote --heads origin")) {
          return Promise.resolve({ exitCode: 0, stdout: "abc123\trefs/heads/some-branch\n", stderr: "" });
        }
        return Promise.resolve({ exitCode: 0, stdout: "", stderr: "" });
      }),
    };

    const { main } = require("./create_pull_request.cjs");
    const handler = await main({});

    const result = await handler({ title: "Test PR", body: "Test body", patch_path: patchFilePath, branch: "some-branch", base_commit: MOCK_BASE_COMMIT_SHA }, {});

    expect(result.success).toBe(true);
    expect(renameCalled).toBe(true);
    const warningCalls = global.core.warning.mock.calls.map(call => String(call[0]));
    expect(warningCalls.some(msg => msg.includes("appending random suffix"))).toBe(true);
  });
});

describe("create_pull_request - copilot assignee on fallback issues", () => {
  let originalEnv;
  let tempDir;

  beforeEach(() => {
    originalEnv = { ...process.env };
    process.env.GH_AW_WORKFLOW_ID = "test-workflow";
    process.env.GITHUB_REPOSITORY = "test-owner/test-repo";
    process.env.GITHUB_BASE_REF = "main";
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-copilot-test-"));

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

    // Push fails to trigger the fallback-issue path; issue creation succeeds
    global.github = {
      rest: {
        pulls: {
          create: vi.fn().mockRejectedValue(Object.assign(new Error("Permission denied"), { status: 403 })),
          requestReviewers: vi.fn().mockResolvedValue({}),
        },
        repos: {
          get: vi.fn().mockResolvedValue({ data: { default_branch: "main" } }),
        },
        issues: {
          create: vi.fn().mockResolvedValue({ data: { number: 99, html_url: "https://github.com/test/issues/99" } }),
          addLabels: vi.fn().mockResolvedValue({}),
        },
      },
      graphql: vi.fn(),
    };

    global.context = {
      eventName: "issues",
      repo: { owner: "test-owner", repo: "test-repo" },
      payload: {},
      runId: "12345",
    };

    global.exec = {
      exec: vi.fn().mockResolvedValue(0),
      getExecOutput: vi.fn().mockImplementation(async (program, args) => {
        // Return empty for rev-list so pushSignedCommits exits early (no commits to replay).
        // These tests focus on copilot assignment, not the signed-commit push path.
        if (program === "git" && args[0] === "rev-list") {
          return { exitCode: 0, stdout: "", stderr: "" };
        }
        return { exitCode: 0, stdout: "main", stderr: "" };
      }),
    };

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

  it("should not call graphql for copilot assignment when GH_AW_ASSIGN_COPILOT is not set", async () => {
    delete process.env.GH_AW_ASSIGN_COPILOT;

    const { main } = require("./create_pull_request.cjs");
    const handler = await main({ assignees: ["copilot"], allow_empty: true });
    await handler({ title: "Test PR", body: "Test body" }, {});

    // No graphql calls for copilot assignment
    expect(global.github.graphql).not.toHaveBeenCalled();
  });

  it("should not call graphql when copilot is not in assignees even if GH_AW_ASSIGN_COPILOT is true", async () => {
    process.env.GH_AW_ASSIGN_COPILOT = "true";

    const { main } = require("./create_pull_request.cjs");
    const handler = await main({ assignees: ["user1"], allow_empty: true });
    await handler({ title: "Test PR", body: "Test body" }, {});

    expect(global.github.graphql).not.toHaveBeenCalled();
  });

  it("should strip copilot from REST assignees for fallback issue but assign via graphql when enabled", async () => {
    process.env.GH_AW_ASSIGN_COPILOT = "true";

    // Mock findAgent → getIssueDetails → assignAgentToIssue
    global.github.graphql
      .mockResolvedValueOnce({
        repository: {
          suggestedActors: {
            nodes: [{ id: "COPILOT_AGENT_ID", login: "copilot-swe-agent", __typename: "Bot" }],
          },
        },
      })
      .mockResolvedValueOnce({
        repository: {
          issue: {
            id: "ISSUE_NODE_ID",
            assignees: { nodes: [] },
          },
        },
      })
      .mockResolvedValueOnce({
        replaceActorsForAssignable: { __typename: "ReplaceActorsForAssignablePayload" },
      });

    const { main } = require("./create_pull_request.cjs");
    const handler = await main({ assignees: ["copilot", "user1"], allow_empty: true });
    await handler({ title: "Test PR", body: "Test body" }, {});

    // Copilot should NOT appear in the REST issue creation payload
    const issueCall = global.github.rest.issues.create.mock.calls[0][0];
    expect(issueCall.assignees).not.toContain("copilot");
    expect(issueCall.assignees).toContain("user1");

    // Graphql should be called for copilot assignment
    expect(global.github.graphql).toHaveBeenCalledTimes(3);
  });

  it("should use configured fallback_labels for fallback issues instead of PR labels", async () => {
    const { main } = require("./create_pull_request.cjs");
    const handler = await main({ allow_empty: true, fallback_labels: ["failure", "automated"] });
    await handler({ title: "Test PR", body: "Test body", labels: ["pr-label"] }, {});

    const issueCall = global.github.rest.issues.create.mock.calls[0][0];
    expect(issueCall.labels).toEqual(["agentic-workflows", "failure", "automated"]);
  });
});

describe("create_pull_request - threat detection caution", () => {
  let tempDir;
  let originalEnv;

  beforeEach(() => {
    originalEnv = { ...process.env };
    process.env.GH_AW_WORKFLOW_ID = "test-workflow";
    process.env.GITHUB_REPOSITORY = "test-owner/test-repo";
    process.env.GITHUB_BASE_REF = "main";
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "create-pr-threat-test-"));

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
          create: vi.fn().mockResolvedValue({ data: { number: 42, html_url: "https://github.com/test/pull/42", node_id: "PR_42" } }),
          requestReviewers: vi.fn().mockResolvedValue({}),
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

  it("should prepend caution alert at top of PR body when GH_AW_DETECTION_CONCLUSION is warning", async () => {
    process.env.GH_AW_DETECTION_CONCLUSION = "warning";

    const { main } = require("./create_pull_request.cjs");
    const handler = await main({ allow_empty: true });
    await handler({ title: "Test PR", body: "Agent body content" }, {});

    const prBody = global.github.rest.pulls.create.mock.calls[0][0].body;
    // Caution alert should appear before the agent body content
    const cautionIndex = prBody.indexOf("[!CAUTION]");
    const bodyIndex = prBody.indexOf("Agent body content");
    expect(cautionIndex).toBeGreaterThanOrEqual(0);
    expect(bodyIndex).toBeGreaterThan(cautionIndex);
  });

  it("should not include caution alert in PR body when GH_AW_DETECTION_CONCLUSION is not warning", async () => {
    delete process.env.GH_AW_DETECTION_CONCLUSION;

    const { main } = require("./create_pull_request.cjs");
    const handler = await main({ allow_empty: true });
    await handler({ title: "Test PR", body: "Agent body content" }, {});

    const prBody = global.github.rest.pulls.create.mock.calls[0][0].body;
    expect(prBody).not.toContain("[!CAUTION]");
  });

  it("should add agentic-threat-detected label when GH_AW_DETECTION_CONCLUSION is warning", async () => {
    process.env.GH_AW_DETECTION_CONCLUSION = "warning";

    const { main } = require("./create_pull_request.cjs");
    const handler = await main({ allow_empty: true });
    await handler({ title: "Test PR", body: "Agent body content" }, {});

    const labelsCall = global.github.rest.issues.addLabels.mock.calls[0][0];
    expect(labelsCall.labels).toContain("agentic-threat-detected");
  });

  it("should not add agentic-threat-detected label when GH_AW_DETECTION_CONCLUSION is not warning", async () => {
    delete process.env.GH_AW_DETECTION_CONCLUSION;

    const { main } = require("./create_pull_request.cjs");
    const handler = await main({ allow_empty: true });
    await handler({ title: "Test PR", body: "Agent body content", labels: ["automation"] }, {});

    const labelsCall = global.github.rest.issues.addLabels.mock.calls[0][0];
    expect(labelsCall.labels).not.toContain("agentic-threat-detected");
  });

  it("should separate caution alert from body content with blank lines", async () => {
    process.env.GH_AW_DETECTION_CONCLUSION = "warning";

    const { main } = require("./create_pull_request.cjs");
    const handler = await main({ allow_empty: true });
    await handler({ title: "Test PR", body: "Agent body content" }, {});

    const prBody = global.github.rest.pulls.create.mock.calls[0][0].body;
    const cautionIndex = prBody.indexOf("[!CAUTION]");
    const bodyIndex = prBody.indexOf("Agent body content");
    // There must be at least two newlines between the end of the caution block and the body content
    const between = prBody.slice(cautionIndex, bodyIndex);
    expect((between.match(/\n/g) || []).length).toBeGreaterThanOrEqual(2);
  });
});
