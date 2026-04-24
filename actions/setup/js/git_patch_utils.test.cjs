/**
 * Tests for git_patch_utils.cjs
 *
 * Pure helpers (sanitize, path, pathspec) are tested as simple unit tests.
 * `computeIncrementalDiffSize` is tested against a REAL local git repository
 * created in a temp directory, so it exercises `git diff --output=<file>`,
 * the O(1) stat-based size measurement, and temp-file cleanup end-to-end.
 *
 * These tests require `git` to be installed on the runner.
 */

import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import os from "os";
import path from "path";
import { spawnSync } from "child_process";

import { sanitizeForFilename, sanitizeBranchNameForPatch, sanitizeRepoSlugForPatch, getPatchPath, getPatchPathForRepo, buildExcludePathspecs, computeIncrementalDiffSize } from "./git_patch_utils.cjs";

// computeIncrementalDiffSize delegates to execGitSync from git_helpers.cjs,
// which calls the GitHub Actions `core.debug` / `core.error` globals. Stub
// them so this test suite can run outside of a workflow runner.
global.core = {
  debug: vi.fn(),
  error: vi.fn(),
  info: vi.fn(),
  warning: vi.fn(),
};

function execGit(args, options = {}) {
  const result = spawnSync("git", args, { encoding: "utf8", ...options });
  if (result.error) throw result.error;
  if (result.status !== 0 && !options.allowFailure) {
    throw new Error(`git ${args.join(" ")} failed: ${result.stderr}`);
  }
  return result;
}

function createTestRepo() {
  const repoDir = fs.mkdtempSync(path.join(os.tmpdir(), "git-patch-utils-"));
  execGit(["init", "-q"], { cwd: repoDir });
  execGit(["config", "user.name", "Test"], { cwd: repoDir });
  execGit(["config", "user.email", "test@example.com"], { cwd: repoDir });
  execGit(["config", "commit.gpgsign", "false"], { cwd: repoDir });
  fs.writeFileSync(path.join(repoDir, "README.md"), "# Test\n");
  execGit(["add", "."], { cwd: repoDir });
  execGit(["commit", "-q", "-m", "Initial commit"], { cwd: repoDir });
  execGit(["branch", "-M", "main"], { cwd: repoDir });
  return repoDir;
}

function cleanupRepo(repoDir) {
  if (repoDir && fs.existsSync(repoDir)) {
    fs.rmSync(repoDir, { recursive: true, force: true });
  }
}

describe("git_patch_utils - pure helpers", () => {
  describe("sanitizeForFilename", () => {
    it("returns fallback when value is empty or nullish", () => {
      expect(sanitizeForFilename("", "fallback")).toBe("fallback");
      expect(sanitizeForFilename(null, "fallback")).toBe("fallback");
      expect(sanitizeForFilename(undefined, "fallback")).toBe("fallback");
    });

    it("replaces path separators and special characters with dashes", () => {
      expect(sanitizeForFilename("feat/foo", "x")).toBe("feat-foo");
      expect(sanitizeForFilename('a\\b:c*d?e"f<g>h|i', "x")).toBe("a-b-c-d-e-f-g-h-i");
    });

    it("collapses runs of dashes and trims leading/trailing dashes", () => {
      expect(sanitizeForFilename("--foo//bar--", "x")).toBe("foo-bar");
    });

    it("lowercases the result", () => {
      expect(sanitizeForFilename("Feature/Mixed-Case", "x")).toBe("feature-mixed-case");
    });
  });

  describe("sanitizeBranchNameForPatch / sanitizeRepoSlugForPatch", () => {
    it("uses 'unknown' fallback for empty branch names", () => {
      expect(sanitizeBranchNameForPatch("")).toBe("unknown");
    });
    it("uses empty fallback for empty repo slugs", () => {
      expect(sanitizeRepoSlugForPatch("")).toBe("");
    });
    it("sanitizes owner/repo slugs", () => {
      expect(sanitizeRepoSlugForPatch("github/Gh-AW")).toBe("github-gh-aw");
    });
  });

  describe("getPatchPath / getPatchPathForRepo", () => {
    it("returns the /tmp/gh-aw path with the sanitized branch name", () => {
      expect(getPatchPath("feat/foo")).toBe("/tmp/gh-aw/aw-feat-foo.patch");
    });
    it("includes a sanitized repo slug for multi-repo scenarios", () => {
      expect(getPatchPathForRepo("feat/foo", "github/gh-aw")).toBe("/tmp/gh-aw/aw-github-gh-aw-feat-foo.patch");
    });
  });

  describe("buildExcludePathspecs", () => {
    it("returns [] for undefined/null/empty inputs", () => {
      expect(buildExcludePathspecs(undefined)).toEqual([]);
      expect(buildExcludePathspecs(null)).toEqual([]);
      expect(buildExcludePathspecs([])).toEqual([]);
    });

    it("produces [--, :(exclude)<pat>, ...] for non-empty inputs", () => {
      expect(buildExcludePathspecs(["*.lock", "dist/**"])).toEqual(["--", ":(exclude)*.lock", ":(exclude)dist/**"]);
    });

    it("ignores non-array inputs (returns [])", () => {
      // @ts-expect-error - exercising runtime guard
      expect(buildExcludePathspecs("not-an-array")).toEqual([]);
    });
  });
});

describe("git_patch_utils.computeIncrementalDiffSize - real git repo", () => {
  let repoDir;

  beforeEach(() => {
    repoDir = createTestRepo();
  });

  afterEach(() => {
    cleanupRepo(repoDir);
  });

  it("returns a positive size that matches the actual diff bytes for a single-file change", () => {
    const baseSha = execGit(["rev-parse", "HEAD"], { cwd: repoDir }).stdout.trim();
    const body = "line\n".repeat(200); // deterministic content
    fs.writeFileSync(path.join(repoDir, "file.txt"), body);
    execGit(["add", "."], { cwd: repoDir });
    execGit(["commit", "-q", "-m", "add file"], { cwd: repoDir });
    const headSha = execGit(["rev-parse", "HEAD"], { cwd: repoDir }).stdout.trim();

    const tmpPath = path.join(repoDir, ".diffsize.tmp");
    const size = computeIncrementalDiffSize({
      baseRef: baseSha,
      headRef: headSha,
      cwd: repoDir,
      tmpPath,
    });

    // Cross-check against `git diff` run independently.
    const expected = Buffer.byteLength(execGit(["diff", "--binary", `${baseSha}..${headSha}`], { cwd: repoDir }).stdout, "utf8");

    expect(size).toBe(expected);
    expect(size).toBeGreaterThan(body.length - 50); // at minimum ~the file contents
  });

  it("always cleans up the temp file, even on success", () => {
    const baseSha = execGit(["rev-parse", "HEAD"], { cwd: repoDir }).stdout.trim();
    fs.writeFileSync(path.join(repoDir, "b.txt"), "hello\n");
    execGit(["add", "."], { cwd: repoDir });
    execGit(["commit", "-q", "-m", "add b"], { cwd: repoDir });
    const headSha = execGit(["rev-parse", "HEAD"], { cwd: repoDir }).stdout.trim();

    const tmpPath = path.join(repoDir, ".diffsize.tmp");
    computeIncrementalDiffSize({ baseRef: baseSha, headRef: headSha, cwd: repoDir, tmpPath });

    expect(fs.existsSync(tmpPath)).toBe(false);
  });

  it("returns 0 when the two refs are identical (empty net diff)", () => {
    const sha = execGit(["rev-parse", "HEAD"], { cwd: repoDir }).stdout.trim();
    const tmpPath = path.join(repoDir, ".diffsize.tmp");
    const size = computeIncrementalDiffSize({ baseRef: sha, headRef: sha, cwd: repoDir, tmpPath });
    expect(size).toBe(0);
    expect(fs.existsSync(tmpPath)).toBe(false);
  });

  it("honors excludedFiles pathspecs (excluded content does not contribute to diff size)", () => {
    const baseSha = execGit(["rev-parse", "HEAD"], { cwd: repoDir }).stdout.trim();
    // Two files: one kept, one excluded. The excluded file is much larger.
    fs.writeFileSync(path.join(repoDir, "keep.txt"), "keep\n");
    fs.writeFileSync(path.join(repoDir, "big.lock"), "x".repeat(20 * 1024));
    execGit(["add", "."], { cwd: repoDir });
    execGit(["commit", "-q", "-m", "add both"], { cwd: repoDir });
    const headSha = execGit(["rev-parse", "HEAD"], { cwd: repoDir }).stdout.trim();

    const tmpPath = path.join(repoDir, ".diffsize.tmp");
    const sizeWithExclude = computeIncrementalDiffSize({
      baseRef: baseSha,
      headRef: headSha,
      cwd: repoDir,
      tmpPath,
      excludedFiles: ["*.lock"],
    });
    const sizeNoExclude = computeIncrementalDiffSize({
      baseRef: baseSha,
      headRef: headSha,
      cwd: repoDir,
      tmpPath,
    });

    expect(sizeWithExclude).toBeGreaterThan(0);
    expect(sizeNoExclude).toBeGreaterThan(sizeWithExclude);
    // The excluded 20 KB lock file should account for the majority of the delta.
    expect(sizeNoExclude - sizeWithExclude).toBeGreaterThan(10 * 1024);
  });

  it("returns null for an invalid baseRef (and still cleans up)", () => {
    const tmpPath = path.join(repoDir, ".diffsize.tmp");
    const size = computeIncrementalDiffSize({
      baseRef: "does-not-exist-ref",
      headRef: "HEAD",
      cwd: repoDir,
      tmpPath,
    });
    expect(size).toBeNull();
    expect(fs.existsSync(tmpPath)).toBe(false);
  });

  it("returns null when required arguments are missing", () => {
    expect(computeIncrementalDiffSize({ baseRef: "", headRef: "HEAD", cwd: "/tmp", tmpPath: "/tmp/x" })).toBeNull();
    expect(computeIncrementalDiffSize({ baseRef: "HEAD", headRef: "", cwd: "/tmp", tmpPath: "/tmp/x" })).toBeNull();
    expect(computeIncrementalDiffSize({ baseRef: "HEAD", headRef: "HEAD", cwd: "", tmpPath: "/tmp/x" })).toBeNull();
    expect(computeIncrementalDiffSize({ baseRef: "HEAD", headRef: "HEAD", cwd: "/tmp", tmpPath: "" })).toBeNull();
  });
});
