/**
 * Integration tests for git patch generation and application
 *
 * These tests run REAL git commands to verify:
 * 1. git format-patch generates valid patches
 * 2. git am can apply patches correctly
 * 3. Emoji and unicode in commit messages work
 * 4. Merge conflict detection works
 * 5. Concurrent push scenarios are handled
 *
 * These tests require git to be installed and create temporary git repos.
 */

import { describe, it, expect, beforeEach, afterEach } from "vitest";
import fs from "fs";
import path from "path";
import { spawnSync } from "child_process";
import os from "os";
import { generateGitPatch } from "./generate_git_patch.cjs";

/**
 * Execute git command safely with args array
 */
function execGit(args, options = {}) {
  const result = spawnSync("git", args, {
    encoding: "utf8",
    ...options,
  });
  if (result.error) {
    throw result.error;
  }
  if (result.status !== 0 && !options.allowFailure) {
    throw new Error(`git ${args.join(" ")} failed: ${result.stderr}`);
  }
  return result;
}

/**
 * Create a temporary git repository for testing
 */
function createTestRepo() {
  const repoDir = fs.mkdtempSync(path.join(os.tmpdir(), "git-patch-test-"));

  execGit(["init"], { cwd: repoDir });
  execGit(["config", "user.name", "Test User"], { cwd: repoDir });
  execGit(["config", "user.email", "test@example.com"], { cwd: repoDir });

  // Create initial commit on main branch
  fs.writeFileSync(path.join(repoDir, "README.md"), "# Test Repo\n");
  execGit(["add", "."], { cwd: repoDir });
  execGit(["commit", "-m", "Initial commit"], { cwd: repoDir });

  // Create main as the default branch
  execGit(["branch", "-M", "main"], { cwd: repoDir });

  return repoDir;
}

/**
 * Clean up test repository
 */
function cleanupTestRepo(repoDir) {
  if (repoDir && fs.existsSync(repoDir)) {
    fs.rmSync(repoDir, { recursive: true, force: true });
  }
}

describe("git patch integration tests", () => {
  let repoDir;
  let patchDir;

  beforeEach(() => {
    repoDir = createTestRepo();
    patchDir = fs.mkdtempSync(path.join(os.tmpdir(), "git-patch-output-"));
  });

  afterEach(() => {
    cleanupTestRepo(repoDir);
    cleanupTestRepo(patchDir);
  });

  // ──────────────────────────────────────────────────────
  // Basic Patch Generation and Application
  // ──────────────────────────────────────────────────────

  describe("basic patch generation and application", () => {
    it("should generate and apply a simple patch", () => {
      // Create feature branch with changes
      execGit(["checkout", "-b", "feature-branch"], { cwd: repoDir });

      fs.writeFileSync(path.join(repoDir, "test.txt"), "Hello World\n");
      execGit(["add", "test.txt"], { cwd: repoDir });
      execGit(["commit", "-m", "Add test file"], { cwd: repoDir });

      // Generate patch
      const patchPath = path.join(patchDir, "test.patch");
      const patchResult = execGit(["format-patch", "main..feature-branch", "--stdout"], { cwd: repoDir });
      fs.writeFileSync(patchPath, patchResult.stdout);

      // Verify patch has content
      const patchContent = fs.readFileSync(patchPath, "utf8");
      expect(patchContent).toContain("Subject:");
      expect(patchContent).toContain("Add test file");
      expect(patchContent).toContain("Hello World");

      // Create a clean branch to test apply
      execGit(["checkout", "main"], { cwd: repoDir });
      execGit(["checkout", "-b", "apply-test"], { cwd: repoDir });

      // Apply patch
      const applyResult = execGit(["am", patchPath], { cwd: repoDir });
      expect(applyResult.status).toBe(0);

      // Verify file was created
      const fileContent = fs.readFileSync(path.join(repoDir, "test.txt"), "utf8");
      expect(fileContent).toBe("Hello World\n");
    });

    it("should handle multi-commit patches", () => {
      execGit(["checkout", "-b", "multi-commit"], { cwd: repoDir });

      // First commit
      fs.writeFileSync(path.join(repoDir, "file1.txt"), "File 1\n");
      execGit(["add", "file1.txt"], { cwd: repoDir });
      execGit(["commit", "-m", "Add file 1"], { cwd: repoDir });

      // Second commit
      fs.writeFileSync(path.join(repoDir, "file2.txt"), "File 2\n");
      execGit(["add", "file2.txt"], { cwd: repoDir });
      execGit(["commit", "-m", "Add file 2"], { cwd: repoDir });

      // Generate patch
      const patchPath = path.join(patchDir, "multi.patch");
      const patchResult = execGit(["format-patch", "main..multi-commit", "--stdout"], { cwd: repoDir });
      fs.writeFileSync(patchPath, patchResult.stdout);

      // Apply to clean branch
      execGit(["checkout", "main"], { cwd: repoDir });
      execGit(["checkout", "-b", "apply-multi"], { cwd: repoDir });

      const applyResult = execGit(["am", patchPath], { cwd: repoDir });
      expect(applyResult.status).toBe(0);

      // Verify both files exist
      expect(fs.existsSync(path.join(repoDir, "file1.txt"))).toBe(true);
      expect(fs.existsSync(path.join(repoDir, "file2.txt"))).toBe(true);
    });
  });

  // ──────────────────────────────────────────────────────
  // Emoji and Unicode in Commit Messages
  // ──────────────────────────────────────────────────────

  describe("emoji and unicode handling", () => {
    it("should handle emoji in commit message", () => {
      execGit(["checkout", "-b", "emoji-branch"], { cwd: repoDir });

      fs.writeFileSync(path.join(repoDir, "emoji.txt"), "Bug fixed!\n");
      execGit(["add", "emoji.txt"], { cwd: repoDir });
      execGit(["commit", "-m", "🐛 Fix bug with login"], { cwd: repoDir });

      // Generate patch
      const patchPath = path.join(patchDir, "emoji.patch");
      const patchResult = execGit(["format-patch", "main..emoji-branch", "--stdout"], { cwd: repoDir });
      fs.writeFileSync(patchPath, patchResult.stdout);

      // Check patch content - git may RFC2047 encode or keep UTF-8
      const patchContent = fs.readFileSync(patchPath, "utf8");
      // Either raw emoji or RFC2047 encoded is acceptable
      const hasEmoji = patchContent.includes("🐛") || patchContent.includes("=?UTF-8?");
      expect(hasEmoji).toBe(true);

      // Apply to clean branch
      execGit(["checkout", "main"], { cwd: repoDir });
      execGit(["checkout", "-b", "apply-emoji"], { cwd: repoDir });

      const applyResult = execGit(["am", patchPath], { cwd: repoDir });
      expect(applyResult.status).toBe(0);

      // Verify commit message was preserved
      const logResult = execGit(["log", "-1", "--format=%s"], { cwd: repoDir });
      expect(logResult.stdout.trim()).toContain("Fix bug");
    });

    it("should handle unicode characters in commit message", () => {
      execGit(["checkout", "-b", "unicode-branch"], { cwd: repoDir });

      fs.writeFileSync(path.join(repoDir, "unicode.txt"), "International text\n");
      execGit(["add", "unicode.txt"], { cwd: repoDir });
      execGit(["commit", "-m", "日本語のコミットメッセージ"], { cwd: repoDir });

      // Generate patch
      const patchPath = path.join(patchDir, "unicode.patch");
      const patchResult = execGit(["format-patch", "main..unicode-branch", "--stdout"], { cwd: repoDir });
      fs.writeFileSync(patchPath, patchResult.stdout);

      // Apply to clean branch
      execGit(["checkout", "main"], { cwd: repoDir });
      execGit(["checkout", "-b", "apply-unicode"], { cwd: repoDir });

      const applyResult = execGit(["am", patchPath], { cwd: repoDir });
      expect(applyResult.status).toBe(0);
    });

    it("should handle multi-line commit messages", () => {
      execGit(["checkout", "-b", "multiline-branch"], { cwd: repoDir });

      fs.writeFileSync(path.join(repoDir, "multiline.txt"), "Content\n");
      execGit(["add", "multiline.txt"], { cwd: repoDir });

      // Commit with multi-line message
      const commitMsg = "Short title\n\nThis is the body of the commit.\nIt has multiple lines.\n\n- Item 1\n- Item 2";
      execGit(["commit", "-m", commitMsg], { cwd: repoDir });

      // Generate patch
      const patchPath = path.join(patchDir, "multiline.patch");
      const patchResult = execGit(["format-patch", "main..multiline-branch", "--stdout"], { cwd: repoDir });
      fs.writeFileSync(patchPath, patchResult.stdout);

      // Verify patch structure
      const patchContent = fs.readFileSync(patchPath, "utf8");
      expect(patchContent).toContain("Subject:");
      expect(patchContent).toContain("Short title");

      // Apply to clean branch
      execGit(["checkout", "main"], { cwd: repoDir });
      execGit(["checkout", "-b", "apply-multiline"], { cwd: repoDir });

      const applyResult = execGit(["am", patchPath], { cwd: repoDir });
      expect(applyResult.status).toBe(0);

      // Verify body was preserved
      const logResult = execGit(["log", "-1", "--format=%B"], { cwd: repoDir });
      expect(logResult.stdout).toContain("body of the commit");
    });

    it("should handle special characters in commit message", () => {
      execGit(["checkout", "-b", "special-chars"], { cwd: repoDir });

      fs.writeFileSync(path.join(repoDir, "special.txt"), "Content\n");
      execGit(["add", "special.txt"], { cwd: repoDir });

      // Commit with special characters that might cause issues
      const commitMsg = "Fix: Handle \"quotes\" and 'apostrophes' and $variables and `backticks`";
      execGit(["commit", "-m", commitMsg], { cwd: repoDir });

      // Generate patch
      const patchPath = path.join(patchDir, "special.patch");
      const patchResult = execGit(["format-patch", "main..special-chars", "--stdout"], { cwd: repoDir });
      fs.writeFileSync(patchPath, patchResult.stdout);

      // Apply to clean branch
      execGit(["checkout", "main"], { cwd: repoDir });
      execGit(["checkout", "-b", "apply-special"], { cwd: repoDir });

      const applyResult = execGit(["am", patchPath], { cwd: repoDir });
      expect(applyResult.status).toBe(0);
    });
  });

  // ──────────────────────────────────────────────────────
  // Merge Conflict Scenarios
  // ──────────────────────────────────────────────────────

  describe("merge conflict scenarios", () => {
    it("should detect and report patch application failure due to conflict", () => {
      // Create conflicting changes
      fs.writeFileSync(path.join(repoDir, "conflict.txt"), "Original content\n");
      execGit(["add", "conflict.txt"], { cwd: repoDir });
      execGit(["commit", "-m", "Add conflict file"], { cwd: repoDir });

      // Create feature branch with one change
      execGit(["checkout", "-b", "feature-conflict"], { cwd: repoDir });
      fs.writeFileSync(path.join(repoDir, "conflict.txt"), "Feature branch content\n");
      execGit(["add", "conflict.txt"], { cwd: repoDir });
      execGit(["commit", "-m", "Feature change"], { cwd: repoDir });

      // Generate patch
      const patchPath = path.join(patchDir, "conflict.patch");
      const patchResult = execGit(["format-patch", "main~1..feature-conflict", "--stdout"], { cwd: repoDir });
      fs.writeFileSync(patchPath, patchResult.stdout);

      // Go back to main and make a different change
      execGit(["checkout", "main"], { cwd: repoDir });
      fs.writeFileSync(path.join(repoDir, "conflict.txt"), "Main branch different content\n");
      execGit(["add", "conflict.txt"], { cwd: repoDir });
      execGit(["commit", "-m", "Main change"], { cwd: repoDir });

      // Try to apply patch - should fail
      const applyResult = execGit(["am", patchPath], { cwd: repoDir, allowFailure: true });
      expect(applyResult.status).not.toBe(0);
      // Error message varies - could be "patch does not apply" or "already exists in index"
      const errorOutput = applyResult.stderr.toLowerCase();
      expect(errorOutput.includes("patch does not apply") || errorOutput.includes("already exists") || errorOutput.includes("conflict")).toBe(true);

      // Abort the failed am
      execGit(["am", "--abort"], { cwd: repoDir, allowFailure: true });
    });

    it("should handle patch based on old commit that no longer exists in history", () => {
      // This simulates force-push scenario where base commit was rewritten
      execGit(["checkout", "-b", "force-push-branch"], { cwd: repoDir });

      fs.writeFileSync(path.join(repoDir, "force.txt"), "Content\n");
      execGit(["add", "force.txt"], { cwd: repoDir });
      execGit(["commit", "-m", "Initial force change"], { cwd: repoDir });

      // Get the commit SHA
      const shaResult = execGit(["rev-parse", "HEAD"], { cwd: repoDir });
      const originalSha = shaResult.stdout.trim();

      // Generate patch
      const patchPath = path.join(patchDir, "force.patch");
      const patchResult = execGit(["format-patch", "main..force-push-branch", "--stdout"], { cwd: repoDir });
      fs.writeFileSync(patchPath, patchResult.stdout);

      // Simulate force-push by amending the commit
      fs.writeFileSync(path.join(repoDir, "force.txt"), "Different content\n");
      execGit(["add", "force.txt"], { cwd: repoDir });
      execGit(["commit", "--amend", "-m", "Amended force change"], { cwd: repoDir });

      // The original SHA should no longer match HEAD
      const newShaResult = execGit(["rev-parse", "HEAD"], { cwd: repoDir });
      expect(newShaResult.stdout.trim()).not.toBe(originalSha);

      // The patch should still be applicable to main (it's based on main)
      execGit(["checkout", "main"], { cwd: repoDir });
      execGit(["checkout", "-b", "apply-force"], { cwd: repoDir });

      const applyResult = execGit(["am", patchPath], { cwd: repoDir });
      expect(applyResult.status).toBe(0);
    });
  });

  // ──────────────────────────────────────────────────────
  // Concurrent Push Scenarios
  // ──────────────────────────────────────────────────────

  describe("concurrent push scenarios", () => {
    let bareRepoDir;
    let workingRepo1;
    let workingRepo2;

    beforeEach(() => {
      // Create a bare repository to simulate remote
      bareRepoDir = fs.mkdtempSync(path.join(os.tmpdir(), "bare-repo-"));
      execGit(["init", "--bare"], { cwd: bareRepoDir });

      // Clone it twice to simulate two workers
      workingRepo1 = fs.mkdtempSync(path.join(os.tmpdir(), "working1-"));
      execGit(["clone", bareRepoDir, "."], { cwd: workingRepo1 });
      execGit(["config", "user.name", "User 1"], { cwd: workingRepo1 });
      execGit(["config", "user.email", "user1@example.com"], { cwd: workingRepo1 });

      // Make initial commit
      fs.writeFileSync(path.join(workingRepo1, "README.md"), "# Repo\n");
      execGit(["add", "."], { cwd: workingRepo1 });
      execGit(["commit", "-m", "Initial"], { cwd: workingRepo1 });
      execGit(["push", "-u", "origin", "main"], { cwd: workingRepo1, allowFailure: true });

      workingRepo2 = fs.mkdtempSync(path.join(os.tmpdir(), "working2-"));
      execGit(["clone", bareRepoDir, "."], { cwd: workingRepo2 });
      execGit(["config", "user.name", "User 2"], { cwd: workingRepo2 });
      execGit(["config", "user.email", "user2@example.com"], { cwd: workingRepo2 });
    });

    afterEach(() => {
      cleanupTestRepo(bareRepoDir);
      cleanupTestRepo(workingRepo1);
      cleanupTestRepo(workingRepo2);
    });

    it("should fail when branch was updated after patch was generated", () => {
      // Create PR branch in working repo 1
      execGit(["checkout", "-b", "pr-branch"], { cwd: workingRepo1 });
      fs.writeFileSync(path.join(workingRepo1, "file1.txt"), "User 1 content\n");
      execGit(["add", "file1.txt"], { cwd: workingRepo1 });
      execGit(["commit", "-m", "User 1 commit"], { cwd: workingRepo1 });
      execGit(["push", "-u", "origin", "pr-branch"], { cwd: workingRepo1 });

      // Fetch in working repo 2 and make changes
      execGit(["fetch", "origin"], { cwd: workingRepo2 });
      execGit(["checkout", "-b", "pr-branch", "--track", "origin/pr-branch"], { cwd: workingRepo2 });

      fs.writeFileSync(path.join(workingRepo2, "file2.txt"), "User 2 content\n");
      execGit(["add", "file2.txt"], { cwd: workingRepo2 });
      execGit(["commit", "-m", "User 2 commit"], { cwd: workingRepo2 });

      // Generate patch in repo 2 (before pushing)
      const patchPath = path.join(patchDir, "concurrent.patch");
      const patchResult = execGit(["format-patch", "origin/pr-branch..pr-branch", "--stdout"], { cwd: workingRepo2 });
      fs.writeFileSync(patchPath, patchResult.stdout);

      // Simulate concurrent push - User 1 pushes first
      fs.writeFileSync(path.join(workingRepo1, "file3.txt"), "Another User 1 commit\n");
      execGit(["add", "file3.txt"], { cwd: workingRepo1 });
      execGit(["commit", "-m", "Concurrent commit from User 1"], { cwd: workingRepo1 });
      execGit(["push", "origin", "pr-branch"], { cwd: workingRepo1 });

      // Now User 2 tries to push (without fetching) - should fail with non-fast-forward
      const pushResult = execGit(["push", "origin", "pr-branch"], { cwd: workingRepo2, allowFailure: true });
      expect(pushResult.status).not.toBe(0);
      // Error message varies by git version but contains rejection info
      const errorOutput = pushResult.stderr.toLowerCase();
      expect(errorOutput.includes("rejected") || errorOutput.includes("failed") || errorOutput.includes("non-fast-forward")).toBe(true);
    });
  });

  // ──────────────────────────────────────────────────────
  // Commit Title Suffix Modification
  // ──────────────────────────────────────────────────────

  describe("commit title suffix modification", () => {
    it("should correctly modify Subject line to add suffix", () => {
      execGit(["checkout", "-b", "suffix-test"], { cwd: repoDir });

      fs.writeFileSync(path.join(repoDir, "suffix.txt"), "Content\n");
      execGit(["add", "suffix.txt"], { cwd: repoDir });
      execGit(["commit", "-m", "Original title"], { cwd: repoDir });

      // Generate patch
      const patchPath = path.join(patchDir, "suffix.patch");
      const patchResult = execGit(["format-patch", "main..suffix-test", "--stdout"], { cwd: repoDir });
      let patchContent = patchResult.stdout;

      // Simulate the suffix modification logic from push_to_pull_request_branch.cjs
      const commitTitleSuffix = " [bot]";
      patchContent = patchContent.replace(/^Subject: (?:\[PATCH\] )?(.*)$/gm, (match, title) => `Subject: [PATCH] ${title}${commitTitleSuffix}`);

      fs.writeFileSync(patchPath, patchContent);

      // Verify the modification
      const modifiedPatch = fs.readFileSync(patchPath, "utf8");
      expect(modifiedPatch).toContain("Subject: [PATCH] Original title [bot]");

      // Apply the modified patch
      execGit(["checkout", "main"], { cwd: repoDir });
      execGit(["checkout", "-b", "apply-suffix"], { cwd: repoDir });

      const applyResult = execGit(["am", patchPath], { cwd: repoDir });
      expect(applyResult.status).toBe(0);

      // Verify commit message has suffix
      const logResult = execGit(["log", "-1", "--format=%s"], { cwd: repoDir });
      expect(logResult.stdout.trim()).toBe("Original title [bot]");
    });

    it("should handle emoji in commit title with suffix modification", () => {
      execGit(["checkout", "-b", "emoji-suffix"], { cwd: repoDir });

      fs.writeFileSync(path.join(repoDir, "emoji.txt"), "Content\n");
      execGit(["add", "emoji.txt"], { cwd: repoDir });
      execGit(["commit", "-m", "🚀 Launch feature"], { cwd: repoDir });

      // Generate patch
      const patchPath = path.join(patchDir, "emoji-suffix.patch");
      const patchResult = execGit(["format-patch", "main..emoji-suffix", "--stdout"], { cwd: repoDir });
      let patchContent = patchResult.stdout;

      // Check if emoji is RFC2047 encoded
      const isEncoded = patchContent.includes("=?UTF-8?");

      // Apply suffix modification
      const commitTitleSuffix = " [bot]";

      if (isEncoded) {
        // For RFC2047 encoded subjects, we need different handling
        // This test documents the current limitation
        console.log("Note: Emoji commit titles are RFC2047 encoded - suffix cannot be applied with simple regex");
        // The regex won't match encoded subjects properly
      } else {
        // If not encoded, apply the suffix
        patchContent = patchContent.replace(/^Subject: (?:\[PATCH\] )?(.*)$/gm, (match, title) => `Subject: [PATCH] ${title}${commitTitleSuffix}`);
      }

      fs.writeFileSync(patchPath, patchContent);

      // Apply the patch
      execGit(["checkout", "main"], { cwd: repoDir });
      execGit(["checkout", "-b", "apply-emoji-suffix"], { cwd: repoDir });

      const applyResult = execGit(["am", patchPath], { cwd: repoDir });
      expect(applyResult.status).toBe(0);

      // The commit should be applied (suffix may or may not be present depending on encoding)
      const logResult = execGit(["log", "-1", "--format=%s"], { cwd: repoDir });
      expect(logResult.stdout).toContain("Launch feature");
    });
  });

  // ──────────────────────────────────────────────────────
  // Incremental Mode Tests
  // ──────────────────────────────────────────────────────

  describe("incremental mode", () => {
    let bareRepoDir;
    let workingRepo;

    beforeEach(() => {
      // Create a bare repo to simulate remote
      bareRepoDir = fs.mkdtempSync(path.join(os.tmpdir(), "bare-incremental-"));
      execGit(["init", "--bare", "--initial-branch=main"], { cwd: bareRepoDir });

      // Clone the bare repo
      workingRepo = fs.mkdtempSync(path.join(os.tmpdir(), "working-incremental-"));
      execGit(["clone", bareRepoDir, "."], { cwd: workingRepo });

      // Set up git config (after clone, we have a proper repo)
      execGit(["config", "user.email", "test@test.com"], { cwd: workingRepo });
      execGit(["config", "user.name", "Test User"], { cwd: workingRepo });

      // Checkout main (might be master on some systems)
      execGit(["checkout", "-b", "main"], { cwd: workingRepo, allowFailure: true });

      // Create initial commit on main
      fs.writeFileSync(path.join(workingRepo, "README.md"), "# Initial\n");
      execGit(["add", "README.md"], { cwd: workingRepo });
      execGit(["commit", "-m", "Initial commit"], { cwd: workingRepo });

      // Push main to origin (this MUST succeed for full mode tests)
      execGit(["push", "-u", "origin", "main"], { cwd: workingRepo });
    });

    afterEach(() => {
      // Clean up
      cleanupTestRepo(bareRepoDir);
      cleanupTestRepo(workingRepo);
    });

    it("should only include new commits after origin/branch in incremental mode", async () => {
      // Create a feature branch with first commit
      execGit(["checkout", "-b", "feature-branch"], { cwd: workingRepo });
      fs.writeFileSync(path.join(workingRepo, "feature.txt"), "First commit content\n");
      execGit(["add", "feature.txt"], { cwd: workingRepo });
      execGit(["commit", "-m", "First commit on feature branch"], { cwd: workingRepo });

      // Push to origin
      execGit(["push", "-u", "origin", "feature-branch"], { cwd: workingRepo });

      // Add a second commit (the "new" commit)
      fs.writeFileSync(path.join(workingRepo, "feature2.txt"), "Second commit content\n");
      execGit(["add", "feature2.txt"], { cwd: workingRepo });
      execGit(["commit", "-m", "Second commit - the new one"], { cwd: workingRepo });

      // Delete the origin/feature-branch tracking ref to simulate a fresh checkout
      // that hasn't fetched the remote branch yet
      execGit(["update-ref", "-d", "refs/remotes/origin/feature-branch"], { cwd: workingRepo });

      // Generate patch in incremental mode
      // Set environment
      const origWorkspace = process.env.GITHUB_WORKSPACE;
      const origDefaultBranch = process.env.DEFAULT_BRANCH;
      process.env.GITHUB_WORKSPACE = workingRepo;
      process.env.DEFAULT_BRANCH = "main";

      try {
        const result = await generateGitPatch("feature-branch", { mode: "incremental" });

        expect(result.success).toBe(true);
        expect(result.patchPath).toBeDefined();

        // Read the patch content
        const patchContent = fs.readFileSync(result.patchPath, "utf8");

        // Should only have ONE patch [PATCH 1/1], not [PATCH 1/2], [PATCH 2/2]
        expect(patchContent).toContain("Subject:");

        // Should contain the second commit
        expect(patchContent).toContain("Second commit - the new one");

        // Should NOT contain the first commit (it's already on origin/feature-branch)
        expect(patchContent).not.toContain("First commit on feature branch");

        // Verify it's [PATCH 1/1] or just [PATCH], not [PATCH 1/2]
        const patchHeaders = patchContent.match(/\[PATCH[^\]]*\]/g);
        // If there are numbered patches, should be just 1
        if (patchHeaders) {
          expect(patchHeaders.length).toBe(1);
        }
      } finally {
        process.env.GITHUB_WORKSPACE = origWorkspace;
        process.env.DEFAULT_BRANCH = origDefaultBranch;
      }
    });

    it("should fail clearly when origin/branch doesnt exist in incremental mode", async () => {
      // Create a local branch without pushing
      execGit(["checkout", "-b", "local-only-branch"], { cwd: workingRepo });
      fs.writeFileSync(path.join(workingRepo, "local.txt"), "Local content\n");
      execGit(["add", "local.txt"], { cwd: workingRepo });
      execGit(["commit", "-m", "Local commit"], { cwd: workingRepo });

      // Don't push - origin/local-only-branch doesn't exist

      const origWorkspace = process.env.GITHUB_WORKSPACE;
      const origDefaultBranch = process.env.DEFAULT_BRANCH;
      process.env.GITHUB_WORKSPACE = workingRepo;
      process.env.DEFAULT_BRANCH = "main";

      try {
        const result = await generateGitPatch("local-only-branch", { mode: "incremental" });

        // Should fail with a clear error message
        expect(result.success).toBe(false);
        expect(result.error).toContain("Cannot generate incremental patch");
        expect(result.error).toContain("origin/local-only-branch");
      } finally {
        process.env.GITHUB_WORKSPACE = origWorkspace;
        process.env.DEFAULT_BRANCH = origDefaultBranch;
      }
    });

    it("should include all commits in full mode even when origin/branch exists", async () => {
      // Create a feature branch with first commit
      execGit(["checkout", "-b", "full-mode-branch"], { cwd: workingRepo });
      fs.writeFileSync(path.join(workingRepo, "full1.txt"), "First commit\n");
      execGit(["add", "full1.txt"], { cwd: workingRepo });
      execGit(["commit", "-m", "First commit in full mode test"], { cwd: workingRepo });

      // Push to origin
      execGit(["push", "-u", "origin", "full-mode-branch"], { cwd: workingRepo });

      // Add second commit
      fs.writeFileSync(path.join(workingRepo, "full2.txt"), "Second commit\n");
      execGit(["add", "full2.txt"], { cwd: workingRepo });
      execGit(["commit", "-m", "Second commit in full mode test"], { cwd: workingRepo });

      // Delete origin ref to force merge-base fallback
      execGit(["update-ref", "-d", "refs/remotes/origin/full-mode-branch"], { cwd: workingRepo });

      // Fetch origin/main so merge-base can work
      execGit(["fetch", "origin", "main"], { cwd: workingRepo });

      const origWorkspace = process.env.GITHUB_WORKSPACE;
      const origDefaultBranch = process.env.DEFAULT_BRANCH;
      process.env.GITHUB_WORKSPACE = workingRepo;
      process.env.DEFAULT_BRANCH = "main";

      try {
        // Full mode (default) - should fall back to merge-base and include all commits
        const result = await generateGitPatch("full-mode-branch", { mode: "full" });

        // Debug output if test fails
        if (!result.success) {
          console.log("Full mode test failed with error:", result.error);
        }

        expect(result.success).toBe(true);

        const patchContent = fs.readFileSync(result.patchPath, "utf8");

        // Should contain BOTH commits (using merge-base with main)
        expect(patchContent).toContain("First commit in full mode test");
        expect(patchContent).toContain("Second commit in full mode test");
      } finally {
        process.env.GITHUB_WORKSPACE = origWorkspace;
        process.env.DEFAULT_BRANCH = origDefaultBranch;
      }
    });
  });
});
