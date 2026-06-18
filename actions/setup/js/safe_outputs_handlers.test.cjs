import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
import { execSync } from "child_process";
import { createHandlers, hasUpdatePullRequestFields } from "./safe_outputs_handlers.cjs";
import {
  looksLikeExploratoryBranch,
  normalizeProbeValue,
  resolveIssueTitleForValidation,
  validateAddCommentIntent,
  validateCreateIssueIntent,
  validateCreatePullRequestIntent,
  validatePushToPullRequestBranchIntent,
} from "./intent_probe.cjs";

const LARGE_CONTENT_BODY = "A".repeat(70000);

// Mock the global objects that GitHub Actions provides
const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  notice: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
};

// Mock context object used by repo_helpers.cjs
const mockContext = {
  repo: {
    owner: "test-owner",
    repo: "test-repo",
  },
  eventName: "push",
  payload: {},
};

// Set up global mocks before importing the module
global.core = mockCore;
global.context = mockContext;

describe("safe_outputs_handlers", () => {
  let mockServer;
  let mockAppendSafeOutput;
  let handlers;
  let testWorkspaceDir;

  beforeEach(() => {
    vi.clearAllMocks();

    mockServer = {
      debug: vi.fn(),
    };

    mockAppendSafeOutput = vi.fn();

    // Create temporary workspace directory
    const testId = Math.random().toString(36).substring(7);
    testWorkspaceDir = `/tmp/test-handlers-workspace-${testId}`;
    fs.mkdirSync(testWorkspaceDir, { recursive: true });

    // Set environment variables
    process.env.GITHUB_WORKSPACE = testWorkspaceDir;
    process.env.GITHUB_SERVER_URL = "https://github.com";
    process.env.GITHUB_REPOSITORY = "owner/repo";
    process.env.GH_AW_WORKFLOW_ID = "test-workflow";

    handlers = createHandlers(mockServer, mockAppendSafeOutput);
  });

  afterEach(() => {
    // Clean up test files
    try {
      if (fs.existsSync(testWorkspaceDir)) {
        fs.rmSync(testWorkspaceDir, { recursive: true, force: true });
      }
    } catch (error) {
      // Ignore cleanup errors
    }

    // Clear environment variables
    delete process.env.GITHUB_WORKSPACE;
    delete process.env.GITHUB_SERVER_URL;
    delete process.env.GITHUB_REPOSITORY;
    delete process.env.GH_AW_WORKFLOW_ID;
    delete process.env.GH_AW_ASSETS_BRANCH;
    delete process.env.GH_AW_ASSETS_MAX_SIZE_KB;
    delete process.env.GH_AW_ASSETS_ALLOWED_EXTS;
  });

  describe("probe intent helpers", () => {
    it("normalizes probe values consistently", () => {
      expect(normalizeProbeValue("  Test   No   Base  ")).toBe("test no base");
      expect(normalizeProbeValue(null)).toBe("");
    });

    it("detects exploratory branch markers", () => {
      expect(looksLikeExploratoryBranch("docs/pr-17198-test-from-main-1853f10f924372d4")).toBe(true);
      expect(looksLikeExploratoryBranch("feature/probe-auth")).toBe(true);
      expect(looksLikeExploratoryBranch("feature/real-work")).toBe(false);
    });

    it("resolves issue title using the create_issue fallback order", () => {
      expect(resolveIssueTitleForValidation({ title: "Real title", body: "Ignored body" })).toBe("Real title");
      expect(resolveIssueTitleForValidation({ body: "Body title" })).toBe("Body title");
      expect(resolveIssueTitleForValidation({})).toBe("Agent Output");
    });

    it("rejects exploratory pull request payloads and allows real ones", () => {
      expect(
        validateCreatePullRequestIntent({
          branch: "docs/pr-17198-test-from-main-1853f10f924372d4",
          title: "test",
          body: "test",
        })
      ).toContain("Refusing to record an exploratory pull request");
      expect(
        validateCreatePullRequestIntent({
          branch: "feature/fix-real-bug",
          title: "Fix retry loop",
          body: "Describe the actual fix",
        })
      ).toBeNull();
    });

    it("rejects exploratory issue and comment payloads", () => {
      expect(validateCreateIssueIntent({ title: "test", body: "test" })).toContain("Refusing to record an exploratory issue");
      expect(validateCreateIssueIntent({ title: "Investigate flaky setup", body: "Track the real issue" })).toBeNull();
      expect(validateAddCommentIntent({ body: "test" })).toContain("Refusing to record an exploratory comment");
      expect(validateAddCommentIntent({ body: "This is the real follow-up comment." })).toBeNull();
    });

    it("rejects exploratory pull request branch updates and allows real ones", () => {
      expect(
        validatePushToPullRequestBranchIntent({
          branch: "feature/probe-auth",
          message: "test",
        })
      ).toContain("Refusing to record an exploratory pull request branch update");
      expect(
        validatePushToPullRequestBranchIntent({
          branch: "feature/real-follow-up",
          message: "Apply review fixes",
        })
      ).toBeNull();
    });
  });

  describe("defaultHandler", () => {
    it("should handle basic entry without large content", () => {
      const handler = handlers.defaultHandler("test-type");
      const args = { field1: "value1", field2: "value2" };

      const result = handler(args);

      expect(mockAppendSafeOutput).toHaveBeenCalledWith({
        field1: "value1",
        field2: "value2",
        type: "test-type",
      });
      expect(result).toEqual({
        content: [
          {
            type: "text",
            text: JSON.stringify({ result: "success" }),
          },
        ],
      });
    });

    it("should handle entry with large content", () => {
      const handler = handlers.defaultHandler("test-type");
      // Create content that exceeds 16000 tokens (roughly 64000 characters)
      const largeContent = "x".repeat(70000);
      const args = { largeField: largeContent, normalField: "normal" };

      const result = handler(args);

      // Should have written large content to file
      expect(mockAppendSafeOutput).toHaveBeenCalled();
      const appendedEntry = mockAppendSafeOutput.mock.calls[0][0];
      expect(appendedEntry.largeField).toContain("[Content too large, saved to file:");
      expect(appendedEntry.normalField).toBe("normal");
      expect(appendedEntry.type).toBe("test-type");

      // Result should contain file info
      expect(result.content[0].type).toBe("text");
      const fileInfo = JSON.parse(result.content[0].text);
      expect(fileInfo.filename).toBeDefined();
    });

    it("should handle null args", () => {
      const handler = handlers.defaultHandler("test-type");

      const result = handler(null);

      expect(mockAppendSafeOutput).toHaveBeenCalledWith({ type: "test-type" });
      expect(result.content[0].text).toBe(JSON.stringify({ result: "success" }));
    });

    it("should handle undefined args", () => {
      const handler = handlers.defaultHandler("test-type");

      const result = handler(undefined);

      expect(mockAppendSafeOutput).toHaveBeenCalledWith({ type: "test-type" });
      expect(result.content[0].text).toBe(JSON.stringify({ result: "success" }));
    });
  });

  describe("uploadAssetHandler", () => {
    let testRunnerTemp;

    beforeEach(() => {
      const testId = Math.random().toString(36).substring(7);
      testRunnerTemp = `/tmp/test-runner-temp-${testId}`;
      process.env.RUNNER_TEMP = testRunnerTemp;
    });

    afterEach(() => {
      delete process.env.RUNNER_TEMP;
      try {
        if (fs.existsSync(testRunnerTemp)) {
          fs.rmSync(testRunnerTemp, { recursive: true, force: true });
        }
      } catch {
        // Ignore cleanup errors
      }
    });

    it("should generate blob URL with raw=true for github.com", () => {
      process.env.GH_AW_ASSETS_BRANCH = "test-branch";
      process.env.GITHUB_SERVER_URL = "https://github.com";
      process.env.GITHUB_REPOSITORY = "myorg/myrepo";

      const testFile = path.join(testWorkspaceDir, "test.png");
      fs.writeFileSync(testFile, "test content");

      handlers.uploadAssetHandler({ path: testFile });

      const entry = mockAppendSafeOutput.mock.calls[0][0];
      expect(entry.url).toContain("github.com/myorg/myrepo/blob/test-branch");
      expect(entry.url).toContain("?raw=true");
      expect(entry.url).not.toContain("raw.githubusercontent.com");
    });

    it("should generate enterprise URL for GitHub Enterprise Server", () => {
      process.env.GH_AW_ASSETS_BRANCH = "test-branch";
      process.env.GITHUB_SERVER_URL = "https://github.example.com";
      process.env.GITHUB_REPOSITORY = "myorg/myrepo";

      const testFile = path.join(testWorkspaceDir, "test2.png");
      fs.writeFileSync(testFile, "test content");

      handlers = createHandlers(mockServer, mockAppendSafeOutput);
      handlers.uploadAssetHandler({ path: testFile });

      const entry = mockAppendSafeOutput.mock.calls[0][0];
      expect(entry.url).toContain("github.example.com");
      expect(entry.url).toContain("/raw/");
      expect(entry.url).not.toContain("raw.githubusercontent.com");
    });

    it("should validate and process valid asset upload", () => {
      process.env.GH_AW_ASSETS_BRANCH = "test-branch";

      // Create test file
      const testFile = path.join(testWorkspaceDir, "test.png");
      fs.writeFileSync(testFile, "test content");

      const args = { path: testFile };
      const result = handlers.uploadAssetHandler(args);

      expect(mockAppendSafeOutput).toHaveBeenCalled();
      const entry = mockAppendSafeOutput.mock.calls[0][0];
      expect(entry.type).toBe("upload_asset");
      expect(entry.fileName).toBe("test.png");
      expect(entry.sha).toBeDefined();
      expect(entry.url).toContain("test-branch");

      expect(result.content[0].type).toBe("text");
      const resultData = JSON.parse(result.content[0].text);
      expect(resultData.result).toContain("https://");
    });

    it("should stage asset file under RUNNER_TEMP not /tmp", () => {
      process.env.GH_AW_ASSETS_BRANCH = "test-branch";

      const testFile = path.join(testWorkspaceDir, "chart.png");
      fs.writeFileSync(testFile, "chart content");

      handlers.uploadAssetHandler({ path: testFile });

      // File must be staged under RUNNER_TEMP, not hardcoded /tmp
      const expectedDir = path.join(testRunnerTemp, "gh-aw", "safeoutputs", "assets");
      expect(fs.existsSync(path.join(expectedDir, "chart.png"))).toBe(true);
    });

    it("should throw error if GH_AW_ASSETS_BRANCH not set", () => {
      delete process.env.GH_AW_ASSETS_BRANCH;

      const args = { path: "/tmp/test.png" };

      expect(() => handlers.uploadAssetHandler(args)).toThrow("GH_AW_ASSETS_BRANCH not set");
    });

    it("should throw error if file not found", () => {
      process.env.GH_AW_ASSETS_BRANCH = "test-branch";

      // Use a path in the workspace that doesn't exist
      const args = { path: path.join(testWorkspaceDir, "nonexistent.png") };

      expect(() => handlers.uploadAssetHandler(args)).toThrow("File not found");
    });

    it("should throw error if file outside allowed directories", () => {
      process.env.GH_AW_ASSETS_BRANCH = "test-branch";

      const args = { path: "/etc/passwd" };

      expect(() => handlers.uploadAssetHandler(args)).toThrow("File path must be within workspace directory");
    });

    it("should allow files in /tmp directory", () => {
      process.env.GH_AW_ASSETS_BRANCH = "test-branch";

      // Create test file in /tmp
      const testFile = `/tmp/test-upload-${Date.now()}.png`;
      fs.writeFileSync(testFile, "test content");

      try {
        const args = { path: testFile };
        const result = handlers.uploadAssetHandler(args);

        expect(mockAppendSafeOutput).toHaveBeenCalled();
        expect(result.content[0].type).toBe("text");
      } finally {
        // Clean up
        if (fs.existsSync(testFile)) {
          fs.unlinkSync(testFile);
        }
      }
    });

    it("should reject file with disallowed extension", () => {
      process.env.GH_AW_ASSETS_BRANCH = "test-branch";

      // Create test file with .txt extension
      const testFile = path.join(testWorkspaceDir, "test.txt");
      fs.writeFileSync(testFile, "test content");

      const args = { path: testFile };

      expect(() => handlers.uploadAssetHandler(args)).toThrow("File extension '.txt' is not allowed");
    });

    it("should accept custom allowed extensions", () => {
      process.env.GH_AW_ASSETS_BRANCH = "test-branch";
      process.env.GH_AW_ASSETS_ALLOWED_EXTS = ".txt,.md";

      const testFile = path.join(testWorkspaceDir, "test.txt");
      fs.writeFileSync(testFile, "test content");

      const args = { path: testFile };
      const result = handlers.uploadAssetHandler(args);

      expect(mockAppendSafeOutput).toHaveBeenCalled();
      expect(result.content[0].type).toBe("text");
    });

    it("should normalize custom allowed extensions", () => {
      process.env.GH_AW_ASSETS_BRANCH = "test-branch";
      process.env.GH_AW_ASSETS_ALLOWED_EXTS = "TXT, md";

      const testFile = path.join(testWorkspaceDir, "test.txt");
      fs.writeFileSync(testFile, "test content");

      const args = { path: testFile };
      const result = handlers.uploadAssetHandler(args);

      expect(mockAppendSafeOutput).toHaveBeenCalled();
      expect(result.content[0].type).toBe("text");
    });

    it("should reject unresolved GitHub expression in allowed extensions", () => {
      process.env.GH_AW_ASSETS_BRANCH = "test-branch";
      process.env.GH_AW_ASSETS_ALLOWED_EXTS = "${{ inputs.allowed_exts }}";

      const testFile = path.join(testWorkspaceDir, "test.txt");
      fs.writeFileSync(testFile, "test content");

      const args = { path: testFile };
      expect(() => handlers.uploadAssetHandler(args)).toThrow("contains unresolved GitHub Actions expression");
    });

    it("should reject unresolved expression even when literal extension also matches", () => {
      process.env.GH_AW_ASSETS_BRANCH = "test-branch";
      process.env.GH_AW_ASSETS_ALLOWED_EXTS = ".txt,${{ inputs.allowed_exts }}";

      const testFile = path.join(testWorkspaceDir, "test.txt");
      fs.writeFileSync(testFile, "test content");

      const args = { path: testFile };
      expect(() => handlers.uploadAssetHandler(args)).toThrow("contains unresolved GitHub Actions expression");
    });

    it("should reject file exceeding size limit", () => {
      process.env.GH_AW_ASSETS_BRANCH = "test-branch";
      process.env.GH_AW_ASSETS_MAX_SIZE_KB = "1"; // 1 KB limit

      // Create file larger than 1KB
      const testFile = path.join(testWorkspaceDir, "large.png");
      fs.writeFileSync(testFile, "x".repeat(2048));

      const args = { path: testFile };

      expect(() => handlers.uploadAssetHandler(args)).toThrow("exceeds maximum allowed size");
    });
  });

  describe("uploadArtifactHandler", () => {
    let testStagingDir;

    beforeEach(() => {
      const testId = Math.random().toString(36).substring(7);
      testStagingDir = `/tmp/test-staging-${testId}`;
      process.env.RUNNER_TEMP = testStagingDir;
    });

    afterEach(() => {
      delete process.env.RUNNER_TEMP;
      try {
        if (fs.existsSync(testStagingDir)) {
          fs.rmSync(testStagingDir, { recursive: true, force: true });
        }
      } catch {
        // Ignore cleanup errors
      }
    });

    it("should copy absolute-path file to staging and rewrite path to basename", () => {
      const srcFile = path.join(testWorkspaceDir, "chart.png");
      fs.writeFileSync(srcFile, "png data");

      const result = handlers.uploadArtifactHandler({ path: srcFile });

      // File should be in staging
      const stagedPath = path.join(testStagingDir, "gh-aw", "safeoutputs", "upload-artifacts", "chart.png");
      expect(fs.existsSync(stagedPath)).toBe(true);
      expect(fs.readFileSync(stagedPath, "utf8")).toBe("png data");

      // JSONL entry should use the basename, not the absolute path
      expect(mockAppendSafeOutput).toHaveBeenCalledWith(expect.objectContaining({ type: "upload_artifact", path: "chart.png" }));

      // Response should be success
      const responseData = JSON.parse(result.content[0].text);
      expect(responseData.result).toBe("success");
    });

    it("should include temporary_id in response when provided", () => {
      const srcFile = path.join(testWorkspaceDir, "plot.png");
      fs.writeFileSync(srcFile, "png data");

      const result = handlers.uploadArtifactHandler({ path: srcFile, temporary_id: "aw_test123" });

      const responseData = JSON.parse(result.content[0].text);
      expect(responseData.result).toBe("success");
      expect(responseData.temporary_id).toBe("aw_test123");
    });

    it("should throw when absolute-path file does not exist", () => {
      expect(() => handlers.uploadArtifactHandler({ path: "/tmp/nonexistent-file.png" })).toThrow(expect.objectContaining({ message: expect.stringContaining("file not found") }));
    });

    it("should throw when path is a symlink", () => {
      const srcFile = path.join(testWorkspaceDir, "real.png");
      fs.writeFileSync(srcFile, "data");
      const linkPath = path.join(testWorkspaceDir, "link.png");
      fs.symlinkSync(srcFile, linkPath);

      expect(() => handlers.uploadArtifactHandler({ path: linkPath })).toThrow(expect.objectContaining({ message: expect.stringContaining("symlinks are not allowed") }));
    });

    it("should not overwrite existing staged file on duplicate call", () => {
      const srcFile = path.join(testWorkspaceDir, "chart.png");
      fs.writeFileSync(srcFile, "original");

      // First call stages the file
      handlers.uploadArtifactHandler({ path: srcFile });

      const stagedPath = path.join(testStagingDir, "gh-aw", "safeoutputs", "upload-artifacts", "chart.png");
      expect(fs.readFileSync(stagedPath, "utf8")).toBe("original");

      // Second call with modified source should not overwrite
      fs.writeFileSync(srcFile, "updated");
      handlers.uploadArtifactHandler({ path: srcFile });
      expect(fs.readFileSync(stagedPath, "utf8")).toBe("original");
    });

    it("should pass through relative path without copying to staging", () => {
      // Relative paths reference files already in staging - no copy needed
      const result = handlers.uploadArtifactHandler({ path: "already-staged.png" });

      // Staging dir should NOT have been created/written by the handler
      const stagingDir = path.join(testStagingDir, "gh-aw", "safeoutputs", "upload-artifacts");
      const stagedFile = path.join(stagingDir, "already-staged.png");
      expect(fs.existsSync(stagedFile)).toBe(false);

      // JSONL entry should preserve the relative path as-is
      expect(mockAppendSafeOutput).toHaveBeenCalledWith(expect.objectContaining({ type: "upload_artifact", path: "already-staged.png" }));

      const responseData = JSON.parse(result.content[0].text);
      expect(responseData.result).toBe("success");
    });

    it("should pass through filters-based request without file copy", () => {
      const result = handlers.uploadArtifactHandler({ filters: { include: ["**/*.png"] } });

      const stagingDir = path.join(testStagingDir, "gh-aw", "safeoutputs", "upload-artifacts");
      expect(fs.existsSync(stagingDir)).toBe(false);

      expect(mockAppendSafeOutput).toHaveBeenCalledWith(expect.objectContaining({ type: "upload_artifact", filters: { include: ["**/*.png"] } }));

      const responseData = JSON.parse(result.content[0].text);
      expect(responseData.result).toBe("success");
    });

    it("should recursively copy directory to staging", () => {
      const srcDir = path.join(testWorkspaceDir, "charts");
      fs.mkdirSync(path.join(srcDir, "sub"), { recursive: true });
      fs.writeFileSync(path.join(srcDir, "a.png"), "a");
      fs.writeFileSync(path.join(srcDir, "sub", "b.png"), "b");

      handlers.uploadArtifactHandler({ path: srcDir });

      const stagingBase = path.join(testStagingDir, "gh-aw", "safeoutputs", "upload-artifacts", "charts");
      expect(fs.existsSync(path.join(stagingBase, "a.png"))).toBe(true);
      expect(fs.existsSync(path.join(stagingBase, "sub", "b.png"))).toBe(true);

      // Entry path should be the directory basename
      expect(mockAppendSafeOutput).toHaveBeenCalledWith(expect.objectContaining({ type: "upload_artifact", path: "charts" }));
    });
  });

  describe("defaultHandler wildcard target validation", () => {
    it("should require explicit discussion_number when update_discussion target is '*'", () => {
      const wildcardHandlers = createHandlers(mockServer, mockAppendSafeOutput, {
        update_discussion: {
          target: "*",
        },
      });

      const result = wildcardHandlers.defaultHandler("update_discussion")({ body: "Updated discussion body." });

      expect(result.isError).toBe(true);
      const responseData = JSON.parse(result.content[0].text);
      expect(responseData.result).toBe("error");
      expect(responseData.error).toContain("requires discussion_number");
      expect(mockAppendSafeOutput).not.toHaveBeenCalled();
    });

    it("should require explicit pull_request_number when close_pull_request target is '*'", () => {
      const wildcardHandlers = createHandlers(mockServer, mockAppendSafeOutput, {
        close_pull_request: {
          target: "*",
        },
      });

      const result = wildcardHandlers.defaultHandler("close_pull_request")({ body: "Closing in favor of a newer PR." });

      expect(result.isError).toBe(true);
      const responseData = JSON.parse(result.content[0].text);
      expect(responseData.result).toBe("error");
      expect(responseData.error).toContain("requires pull_request_number");
      expect(mockAppendSafeOutput).not.toHaveBeenCalled();
    });

    it("should require explicit pull_request_number when create_check_run target is '*'", () => {
      const wildcardHandlers = createHandlers(mockServer, mockAppendSafeOutput, {
        create_check_run: {
          target: "*",
        },
      });

      const result = wildcardHandlers.defaultHandler("create_check_run")({ conclusion: "success", title: "Checks passed", summary: "All checks passed." });

      expect(result.isError).toBe(true);
      const responseData = JSON.parse(result.content[0].text);
      expect(responseData.result).toBe("error");
      expect(responseData.error).toContain("requires pull_request_number");
      expect(mockAppendSafeOutput).not.toHaveBeenCalled();
    });
  });

  describe("createPullRequestHandler", () => {
    /**
     * Creates a side-repo checkout where:
     * - main is the default branch
     * - release-1.12.x has an existing remote-tracked commit not on main
     * - the checked-out local release branch has one extra local-only commit
     *
     * This lets the test verify that create_pull_request diffs against
     * origin/release-1.12.x instead of origin/main, so only the local fix
     * ends up in the generated patch.
     */
    function createSideRepoOnReleaseBranchWithLocalCommit() {
      const targetRepoDir = path.join(testWorkspaceDir, "target-repo");
      fs.mkdirSync(targetRepoDir, { recursive: true });

      execSync("git init -b main", { cwd: targetRepoDir, stdio: "pipe" });
      execSync("git config user.email 'test@example.com'", { cwd: targetRepoDir, stdio: "pipe" });
      execSync("git config user.name 'Test User'", { cwd: targetRepoDir, stdio: "pipe" });

      fs.writeFileSync(path.join(targetRepoDir, "README.md"), "base\n");
      execSync("git add README.md", { cwd: targetRepoDir, stdio: "pipe" });
      execSync("git commit -m 'base commit'", { cwd: targetRepoDir, stdio: "pipe" });

      execSync("git checkout -b release-1.12.x", { cwd: targetRepoDir, stdio: "pipe" });
      fs.writeFileSync(path.join(targetRepoDir, "README.md"), "release tracked\n");
      execSync("git add README.md", { cwd: targetRepoDir, stdio: "pipe" });
      execSync("git commit -m 'release tracked commit'", { cwd: targetRepoDir, stdio: "pipe" });
      const releaseCommitSha = execSync("git rev-parse HEAD", { cwd: targetRepoDir, stdio: "pipe" }).toString().trim();

      execSync("git checkout main", { cwd: targetRepoDir, stdio: "pipe" });
      fs.writeFileSync(path.join(targetRepoDir, "MAIN_ONLY.md"), "main only\n");
      execSync("git add MAIN_ONLY.md", { cwd: targetRepoDir, stdio: "pipe" });
      execSync("git commit -m 'main only commit'", { cwd: targetRepoDir, stdio: "pipe" });
      const mainCommitSha = execSync("git rev-parse HEAD", { cwd: targetRepoDir, stdio: "pipe" }).toString().trim();

      execSync("git checkout release-1.12.x", { cwd: targetRepoDir, stdio: "pipe" });
      execSync("git remote add origin https://github.com/test-owner/test-repo.git", { cwd: targetRepoDir, stdio: "pipe" });
      execSync(`git update-ref refs/remotes/origin/main ${mainCommitSha}`, { cwd: targetRepoDir, stdio: "pipe" });
      execSync(`git update-ref refs/remotes/origin/release-1.12.x ${releaseCommitSha}`, { cwd: targetRepoDir, stdio: "pipe" });
      execSync("git symbolic-ref refs/remotes/origin/HEAD refs/remotes/origin/main", { cwd: targetRepoDir, stdio: "pipe" });

      fs.writeFileSync(path.join(targetRepoDir, "README.md"), "release local only\n");
      execSync("git add README.md", { cwd: targetRepoDir, stdio: "pipe" });
      execSync("git commit -m 'local only fix'", { cwd: targetRepoDir, stdio: "pipe" });

      return { targetRepoDir };
    }

    it("should be defined", () => {
      expect(handlers.createPullRequestHandler).toBeDefined();
    });

    it("should reject obvious exploratory test payloads before recording a PR intent", async () => {
      const result = await handlers.createPullRequestHandler({
        branch: "docs/pr-17198-test-from-main-1853f10f924372d4",
        title: "test",
        body: "test",
      });

      expect(result.isError).toBe(true);
      const responseData = JSON.parse(result.content[0].text);
      expect(responseData.result).toBe("error");
      expect(responseData.error).toContain("Refusing to record an exploratory pull request");
      expect(responseData.error).toContain("noop or report_incomplete");
      expect(mockAppendSafeOutput).not.toHaveBeenCalled();
    });

    it("should require temporary_id when configured", async () => {
      handlers = createHandlers(mockServer, mockAppendSafeOutput, {
        create_pull_request: {
          allow_empty: true,
          require_temporary_id: true,
        },
      });

      const result = await handlers.createPullRequestHandler({
        branch: "feature/test-change",
        title: "Test PR",
        body: "Test description",
      });

      expect(result.isError).toBe(true);
      const responseData = JSON.parse(result.content[0].text);
      expect(responseData.result).toBe("error");
      expect(responseData.error).toContain("requires 'temporary_id'");
      expect(mockAppendSafeOutput).not.toHaveBeenCalled();
    });

    it("should accept temporary_id when required and provided", async () => {
      handlers = createHandlers(mockServer, mockAppendSafeOutput, {
        create_pull_request: {
          allow_empty: true,
          require_temporary_id: true,
        },
      });

      const result = await handlers.createPullRequestHandler({
        branch: "feature/test-change",
        title: "Test PR",
        body: "Test description",
        temporary_id: "aw_pr1",
      });

      expect(result.isError).toBeUndefined();
      const responseData = JSON.parse(result.content[0].text);
      expect(responseData.result).toBe("success");
      expect(mockAppendSafeOutput).toHaveBeenCalledWith(
        expect.objectContaining({
          type: "create_pull_request",
          temporary_id: "aw_pr1",
        })
      );
    });

    it("should return error response when patch generation fails (not throw)", async () => {
      // This test verifies the error is returned as content, not thrown
      // Patch generation will fail because we're not in a git repo
      const args = {
        branch: "feature-branch",
        title: "Test PR",
        body: "Test description",
      };

      // The handler should NOT throw an error, it should return an error response
      const result = await handlers.createPullRequestHandler(args);

      // Verify it returns an error response structure
      expect(result).toBeDefined();
      expect(result.content).toBeDefined();
      expect(Array.isArray(result.content)).toBe(true);
      expect(result.content[0].type).toBe("text");
      expect(result.isError).toBe(true);

      // Parse the response
      const responseData = JSON.parse(result.content[0].text);
      expect(responseData.result).toBe("error");
      expect(responseData.error).toBeDefined();
      expect(responseData.error).toContain("Failed to pin branch");
      expect(responseData.error).toContain("before bundle generation");
      expect(responseData.details).toBeDefined();
      expect(responseData.details).toContain("Bundle transport requires branch pinning");

      // Should not have appended to safe output since patch generation failed
      expect(mockAppendSafeOutput).not.toHaveBeenCalled();
    });

    it("should include helpful details in error response", async () => {
      const args = {
        branch: "test-branch",
        title: "Test",
        body: "Description",
      };

      const result = await handlers.createPullRequestHandler(args);

      expect(result.isError).toBe(true);
      const responseData = JSON.parse(result.content[0].text);

      // Verify the details provide actionable guidance
      expect(responseData.details).toContain("Bundle transport requires branch pinning");
      expect(responseData.details).toContain("branch exists locally");
    });

    it("should return error when repo parameter is not in the allowed-repos list", async () => {
      const args = {
        branch: "feature-branch",
        title: "Test PR",
        body: "Test description",
        repo: "owner/non-existent-repo",
      };

      const result = await handlers.createPullRequestHandler(args);

      expect(result.isError).toBe(true);
      const responseData = JSON.parse(result.content[0].text);
      expect(responseData.result).toBe("error");
      expect(responseData.error).toContain("not in the allowed-repos list");
      expect(responseData.error).toContain("owner/non-existent-repo");
    });

    it("should treat empty repo string as workspace root", async () => {
      // Empty string should not trigger multi-repo code path
      const args = {
        branch: "feature-branch",
        title: "Test PR",
        body: "Test description",
        repo: "",
      };

      const result = await handlers.createPullRequestHandler(args);

      // Should proceed to patch generation (which will fail because not in git repo)
      // but NOT fail with repo not found error
      expect(result.isError).toBe(true);
      const responseData = JSON.parse(result.content[0].text);
      // Should be a patch generation error, not a repo not found error
      expect(responseData.error).not.toContain("not found in workspace");
      expect(responseData.error).toContain("Failed to pin branch");
    });

    it("should treat whitespace-only repo as workspace root", async () => {
      const args = {
        branch: "feature-branch",
        title: "Test PR",
        body: "Test description",
        repo: "   ",
      };

      const result = await handlers.createPullRequestHandler(args);

      // Should proceed to patch generation (which will fail because not in git repo)
      // but NOT fail with repo not found error
      expect(result.isError).toBe(true);
      const responseData = JSON.parse(result.content[0].text);
      expect(responseData.error).not.toContain("not found in workspace");
    });

    it("should prefer configured base_branch over trigger context base ref", async () => {
      handlers = createHandlers(mockServer, mockAppendSafeOutput, {
        create_pull_request: {
          allow_empty: true,
          base_branch: "main",
        },
      });

      process.env.GITHUB_BASE_REF = "master";
      process.env.GITHUB_HEAD_REF = "feature/test-change";
      process.env.GITHUB_REF_NAME = "feature/test-change";
      try {
        const result = await handlers.createPullRequestHandler({
          branch: "main",
          title: "Test PR",
          body: "Test description",
        });

        expect(result.isError).toBeUndefined();
        const responseData = JSON.parse(result.content[0].text);
        expect(responseData.result).toBe("success");
        expect(responseData.branch).toBe("feature/test-change");
        expect(mockServer.debug).toHaveBeenCalledWith(expect.stringContaining("Branch equals base branch (main)"));
        expect(mockAppendSafeOutput).toHaveBeenCalledWith(
          expect.objectContaining({
            type: "create_pull_request",
            branch: "feature/test-change",
          })
        );
      } finally {
        delete process.env.GITHUB_BASE_REF;
        delete process.env.GITHUB_HEAD_REF;
        delete process.env.GITHUB_REF_NAME;
      }
    });

    it("should reject create_pull_request when branch still equals base_branch after detection (unresolvable base)", async () => {
      // Simulates the scenario where getBaseBranch() incorrectly resolves to the
      // feature branch itself (e.g., GITHUB_BASE_REF set to the feature branch).
      // Detection calls getCurrentBranch() which also returns the feature branch,
      // so both entry.branch and entry.base_branch remain the same value.
      // This must be rejected with a clear error rather than writing a malformed
      // safe output that causes a cryptic git exit-1 in the safe_outputs job.
      handlers = createHandlers(mockServer, mockAppendSafeOutput, {
        create_pull_request: {
          allow_empty: true,
        },
      });

      // GITHUB_BASE_REF set to the feature branch (incorrectly), simulating the bug
      process.env.GITHUB_BASE_REF = "repo-assist/fix-issue-129";
      process.env.GITHUB_HEAD_REF = "repo-assist/fix-issue-129";
      process.env.GITHUB_REF_NAME = "repo-assist/fix-issue-129";
      try {
        const result = await handlers.createPullRequestHandler({
          branch: "repo-assist/fix-issue-129",
          title: "Fix issue 129",
          body: "Applies the fix for issue 129",
        });

        expect(result.isError).toBe(true);
        const responseData = JSON.parse(result.content[0].text);
        expect(responseData.result).toBe("error");
        expect(responseData.error).toContain("equals base_branch");
        expect(responseData.error).toContain("Cannot create a pull request from a branch into itself");
        expect(mockAppendSafeOutput).not.toHaveBeenCalled();
      } finally {
        delete process.env.GITHUB_BASE_REF;
        delete process.env.GITHUB_HEAD_REF;
        delete process.env.GITHUB_REF_NAME;
      }
    });

    it("should enforce create_pull_request allowed_branches against resolved branch", async () => {
      handlers = createHandlers(mockServer, mockAppendSafeOutput, {
        create_pull_request: {
          allow_empty: true,
          base_branch: "main",
          allowed_branches: ["feature/*"],
        },
      });

      process.env.GITHUB_HEAD_REF = "feature/from-detection";
      process.env.GITHUB_REF_NAME = "feature/from-detection";
      try {
        const result = await handlers.createPullRequestHandler({
          branch: "main",
          title: "Test PR",
          body: "Test description",
        });

        expect(result.isError).toBeUndefined();
        const responseData = JSON.parse(result.content[0].text);
        expect(responseData.result).toBe("success");
        expect(responseData.branch).toBe("feature/from-detection");
        expect(mockAppendSafeOutput).toHaveBeenCalledWith(
          expect.objectContaining({
            type: "create_pull_request",
            branch: "feature/from-detection",
          })
        );
      } finally {
        delete process.env.GITHUB_HEAD_REF;
        delete process.env.GITHUB_REF_NAME;
      }
    });

    it("should reject create_pull_request when branch does not match allowed_branches", async () => {
      handlers = createHandlers(mockServer, mockAppendSafeOutput, {
        create_pull_request: {
          allow_empty: true,
          allowed_branches: ["feature/*"],
        },
      });

      const result = await handlers.createPullRequestHandler({
        branch: "hotfix/not-allowed",
        title: "Test PR",
        body: "Test description",
      });

      expect(result.isError).toBe(true);
      const responseData = JSON.parse(result.content[0].text);
      expect(responseData.result).toBe("error");
      expect(responseData.error).toContain("does not match allowed-branches");
      expect(responseData.error).toContain("feature/*");
      expect(mockAppendSafeOutput).not.toHaveBeenCalled();
    });

    it("should validate the resolved current branch before recording a PR intent", async () => {
      handlers = createHandlers(mockServer, mockAppendSafeOutput, {
        create_pull_request: {
          allow_empty: true,
          base_branch: "main",
        },
      });

      process.env.GITHUB_HEAD_REF = "docs/pr-17198-test-from-main-1853f10f924372d4";
      process.env.GITHUB_REF_NAME = "docs/pr-17198-test-from-main-1853f10f924372d4";
      try {
        const result = await handlers.createPullRequestHandler({
          branch: "main",
          title: "Real looking title",
          body: "Real looking body",
        });

        expect(result.isError).toBe(true);
        const responseData = JSON.parse(result.content[0].text);
        expect(responseData.result).toBe("error");
        expect(responseData.error).toContain("Refusing to record an exploratory pull request");
        expect(mockAppendSafeOutput).not.toHaveBeenCalled();
      } finally {
        delete process.env.GITHUB_HEAD_REF;
        delete process.env.GITHUB_REF_NAME;
      }
    });

    it("should fail closed when patch_format resolves to an invalid value", async () => {
      handlers = createHandlers(mockServer, mockAppendSafeOutput, {
        create_pull_request: {
          patch_format: "invalid-format",
        },
      });

      const result = await handlers.createPullRequestHandler({
        branch: "feature-branch",
        title: "Test PR",
        body: "Test description",
      });

      expect(result.isError).toBe(true);
      const responseData = JSON.parse(result.content[0].text);
      expect(responseData.result).toBe("error");
      expect(responseData.error).toContain("Invalid patch_format");
      expect(responseData.error).toContain("am");
      expect(responseData.error).toContain("bundle");
      // Must not echo the raw resolved value (could be a secret expression result)
      expect(responseData.error).not.toContain("invalid-format");
      // Must not have appended any safe output
      expect(mockAppendSafeOutput).not.toHaveBeenCalled();
    });

    it("should fail closed when patch_format resolves to an empty string", async () => {
      handlers = createHandlers(mockServer, mockAppendSafeOutput, {
        create_pull_request: {
          patch_format: "",
        },
      });

      const result = await handlers.createPullRequestHandler({
        branch: "feature-branch",
        title: "Test PR",
        body: "Test description",
      });

      expect(result.isError).toBe(true);
      const responseData = JSON.parse(result.content[0].text);
      expect(responseData.result).toBe("error");
      expect(responseData.error).toContain("Invalid patch_format");
      expect(mockAppendSafeOutput).not.toHaveBeenCalled();
    });

    it("should store resolved base_branch in the safe output entry (allow-empty mode)", async () => {
      // Verifies that base_branch is embedded in the safe output payload so that
      // the apply-time checkout step can use it directly (self-describing safe output).
      handlers = createHandlers(mockServer, mockAppendSafeOutput, {
        create_pull_request: {
          allow_empty: true,
          base_branch: "release/v2.0",
        },
      });

      process.env.GITHUB_BASE_REF = "main"; // Would be wrong branch without self-describing
      try {
        const result = await handlers.createPullRequestHandler({
          branch: "feature/my-work",
          title: "Test PR",
          body: "Test description",
        });

        expect(result.isError).toBeUndefined();
        // base_branch should be stored in the appended entry
        expect(mockAppendSafeOutput).toHaveBeenCalledWith(
          expect.objectContaining({
            type: "create_pull_request",
            base_branch: "release/v2.0",
          })
        );
      } finally {
        delete process.env.GITHUB_BASE_REF;
      }
    });

    it("should store GITHUB_BASE_REF as base_branch when no config override (allow-empty mode)", async () => {
      // Verifies that the resolved base branch from event context is stored in the entry.
      handlers = createHandlers(mockServer, mockAppendSafeOutput, {
        create_pull_request: {
          allow_empty: true,
        },
      });

      process.env.GITHUB_BASE_REF = "feature/target-branch";
      try {
        const result = await handlers.createPullRequestHandler({
          branch: "feature/my-work",
          title: "Test PR",
          body: "Test description",
        });

        expect(result.isError).toBeUndefined();
        // base_branch should be the resolved branch from event context
        expect(mockAppendSafeOutput).toHaveBeenCalledWith(
          expect.objectContaining({
            type: "create_pull_request",
            base_branch: "feature/target-branch",
          })
        );
      } finally {
        delete process.env.GITHUB_BASE_REF;
      }
    });

    it("should use side-repo origin/HEAD base branch so patch includes branch commits since main", async () => {
      const { targetRepoDir } = createSideRepoOnReleaseBranchWithLocalCommit();

      handlers = createHandlers(mockServer, mockAppendSafeOutput, {
        create_pull_request: {
          "target-repo": "test-owner/test-repo",
          patch_format: "am",
        },
      });

      const result = await handlers.createPullRequestHandler({
        branch: "release-1.12.x",
        title: "Release fix",
        body: "Prepare release branch fix",
      });

      expect(result.isError).toBeUndefined();
      expect(mockServer.debug).toHaveBeenCalledWith(expect.stringContaining(`Found repo checkout at: ${targetRepoDir}`));
      // No base-branch override is configured. The checked-out branch is release-1.12.x,
      // but origin/HEAD points to origin/main, so base_branch must resolve to main.
      expect(mockAppendSafeOutput).toHaveBeenCalledWith(
        expect.objectContaining({
          type: "create_pull_request",
          base_branch: "main",
          branch: "release-1.12.x",
        })
      );
      const responseData = JSON.parse(result.content[0].text);
      const patchContent = fs.readFileSync(responseData.patch.path, "utf8");
      // Diffing release-1.12.x against base main includes both release-only commits:
      // the tracked release commit and the local-only fix.
      expect(patchContent).toContain("local only fix");
      expect(patchContent).toContain("release tracked commit");
      expect(patchContent).not.toContain("MAIN_ONLY.md");
    });

    it("should use patch_workspace_path when target repo resolves from GH_AW_TARGET_REPO_SLUG", async () => {
      createSideRepoOnReleaseBranchWithLocalCommit();
      process.env.GH_AW_TARGET_REPO_SLUG = "test-owner/test-repo";
      handlers = createHandlers(mockServer, mockAppendSafeOutput, {
        create_pull_request: {
          patch_workspace_path: "target-repo",
          current_checkout_repo: "test-owner/test-repo",
          patch_format: "am",
        },
      });

      try {
        const result = await handlers.createPullRequestHandler({
          branch: "release-1.12.x",
          title: "Release fix",
          body: "Prepare release branch fix",
        });

        expect(result.isError).toBeUndefined();
        expect(mockServer.debug).toHaveBeenCalledWith(expect.stringContaining("Using configured patch_workspace_path for create_pull_request"));
      } finally {
        delete process.env.GH_AW_TARGET_REPO_SLUG;
      }
    });
  });

  describe("pushToPullRequestBranchHandler", () => {
    // The agent never supplies a branch — the handler always derives the source
    // branch from getCurrentBranch(). In production this resolves the working
    // tree's HEAD; in unit tests the workspace is an empty temp dir with no git
    // repo, so seed the env-var fallback path so tests can focus on the
    // downstream patch-generation behavior they actually want to exercise.
    beforeEach(() => {
      process.env.GITHUB_REF_NAME = "feature-branch";
    });

    afterEach(() => {
      delete process.env.GITHUB_REF_NAME;
    });

    function createSideRepoWithTrackedAndLocalCommits() {
      const targetRepoDir = path.join(testWorkspaceDir, "target-repo");
      fs.mkdirSync(targetRepoDir, { recursive: true });

      execSync("git init -b main", { cwd: targetRepoDir, stdio: "pipe" });
      execSync("git config user.email 'test@example.com'", { cwd: targetRepoDir, stdio: "pipe" });
      execSync("git config user.name 'Test User'", { cwd: targetRepoDir, stdio: "pipe" });

      fs.writeFileSync(path.join(targetRepoDir, "README.md"), "base\n");
      execSync("git add README.md", { cwd: targetRepoDir, stdio: "pipe" });
      execSync("git commit -m 'base commit'", { cwd: targetRepoDir, stdio: "pipe" });

      execSync("git checkout -b feature/test-change", { cwd: targetRepoDir, stdio: "pipe" });
      fs.writeFileSync(path.join(targetRepoDir, "README.md"), "tracked\n");
      execSync("git add README.md", { cwd: targetRepoDir, stdio: "pipe" });
      execSync("git commit -m 'tracked commit'", { cwd: targetRepoDir, stdio: "pipe" });
      const trackedCommit = execSync("git rev-parse HEAD", { cwd: targetRepoDir, stdio: "pipe" }).toString().trim();

      execSync("git remote add origin https://github.com/test-owner/test-repo.git", { cwd: targetRepoDir, stdio: "pipe" });
      execSync(`git update-ref refs/remotes/origin/feature/test-change ${trackedCommit}`, { cwd: targetRepoDir, stdio: "pipe" });

      fs.writeFileSync(path.join(targetRepoDir, "README.md"), "local-only\n");
      execSync("git add README.md", { cwd: targetRepoDir, stdio: "pipe" });
      execSync("git commit -m 'local only commit'", { cwd: targetRepoDir, stdio: "pipe" });

      return { targetRepoDir };
    }

    it("should be defined", () => {
      expect(handlers.pushToPullRequestBranchHandler).toBeDefined();
    });

    it("should return error response when patch generation fails (not throw)", async () => {
      // This test verifies the error is returned as content, not thrown
      const args = {
        branch: "feature-branch",
      };

      // The handler should NOT throw an error, it should return an error response
      const result = await handlers.pushToPullRequestBranchHandler(args);

      // Verify it returns an error response structure
      expect(result).toBeDefined();
      expect(result.content).toBeDefined();
      expect(Array.isArray(result.content)).toBe(true);
      expect(result.content[0].type).toBe("text");
      expect(result.isError).toBe(true);

      // Parse the response
      const responseData = JSON.parse(result.content[0].text);
      expect(responseData.result).toBe("error");
      expect(responseData.error).toBeDefined();
      expect(responseData.error).toContain("Failed to pin branch");
      expect(responseData.details).toBeDefined();
      expect(responseData.details).toContain("Bundle transport requires branch pinning");

      // Should not have appended to safe output since patch generation failed
      expect(mockAppendSafeOutput).not.toHaveBeenCalled();
    });

    it("should require explicit pull_request_number when push_to_pull_request_branch target is '*'", async () => {
      const wildcardHandlers = createHandlers(mockServer, mockAppendSafeOutput, {
        push_to_pull_request_branch: {
          target: "*",
        },
      });

      const result = await wildcardHandlers.pushToPullRequestBranchHandler({ message: "Apply requested changes." });

      expect(result.isError).toBe(true);
      const responseData = JSON.parse(result.content[0].text);
      expect(responseData.result).toBe("error");
      expect(responseData.error).toContain("requires pull_request_number");
      expect(mockAppendSafeOutput).not.toHaveBeenCalled();
    });

    it("should reject obvious exploratory test payloads before recording a PR branch update intent", async () => {
      // The agent can no longer supply `branch`; the handler derives it from
      // the current working checkout. Model the failure mode where the
      // working tree itself is sitting on an obviously exploratory branch.
      const prevRefName = process.env.GITHUB_REF_NAME;
      process.env.GITHUB_REF_NAME = "docs/pr-17198-test-from-main-1853f10f924372d4";
      try {
        const result = await handlers.pushToPullRequestBranchHandler({
          message: "test",
        });

        expect(result.isError).toBe(true);
        const responseData = JSON.parse(result.content[0].text);
        expect(responseData.result).toBe("error");
        expect(responseData.error).toContain("Refusing to record an exploratory pull request branch update");
        expect(responseData.error).toContain("noop or report_incomplete");
        expect(mockAppendSafeOutput).not.toHaveBeenCalled();
      } finally {
        if (prevRefName === undefined) delete process.env.GITHUB_REF_NAME;
        else process.env.GITHUB_REF_NAME = prevRefName;
      }
    });

    it("should include helpful details in error response", async () => {
      const args = {
        branch: "test-branch",
      };

      const result = await handlers.pushToPullRequestBranchHandler(args);

      expect(result.isError).toBe(true);
      const responseData = JSON.parse(result.content[0].text);

      // Verify the details provide actionable guidance
      expect(responseData.details).toContain("Bundle transport requires branch pinning");
      expect(responseData.details).toContain("branch exists locally");
    });

    it("should return error when repo checkout is not found for explicit repo", async () => {
      const result = await handlers.pushToPullRequestBranchHandler({
        branch: "main",
        repo: "test-owner/test-repo",
      });

      expect(result.isError).toBe(true);
      const responseData = JSON.parse(result.content[0].text);
      expect(responseData.result).toBe("error");
      expect(responseData.error).toContain("Repository 'test-owner/test-repo' not found in workspace");
      expect(responseData.error).toContain("actions/checkout");
      expect(responseData.error).toContain("'path' input");
    });

    it("should return error when configured target-repo checkout is not found and entry.repo is not set", async () => {
      const configWithTarget = {
        push_to_pull_request_branch: { "target-repo": "test-owner/test-repo" },
      };
      const handlersWithTarget = createHandlers(mockServer, mockAppendSafeOutput, configWithTarget);

      const result = await handlersWithTarget.pushToPullRequestBranchHandler({
        branch: "feature/test-change",
      });

      expect(result.isError).toBe(true);
      const responseData = JSON.parse(result.content[0].text);
      expect(responseData.result).toBe("error");
      expect(responseData.error).toContain("Repository 'test-owner/test-repo' not found in workspace");
      expect(responseData.error).toContain("actions/checkout");
      expect(responseData.error).toContain("'path' input");
    });

    it("should use patch_workspace_path when target repo resolves from GH_AW_TARGET_REPO_SLUG", async () => {
      createSideRepoWithTrackedAndLocalCommits();
      process.env.GH_AW_TARGET_REPO_SLUG = "test-owner/test-repo";
      const handlersWithPatchWorkspace = createHandlers(mockServer, mockAppendSafeOutput, {
        push_to_pull_request_branch: {
          patch_workspace_path: "target-repo",
          current_checkout_repo: "test-owner/test-repo",
          patch_format: "am",
        },
      });

      process.env.GITHUB_BASE_REF = "main";
      try {
        const result = await handlersWithPatchWorkspace.pushToPullRequestBranchHandler({
          branch: "feature/test-change",
        });

        expect(result.isError).toBeFalsy();
        expect(mockServer.debug).toHaveBeenCalledWith(expect.stringContaining("Using configured patch_workspace_path for push_to_pull_request_branch"));
      } finally {
        delete process.env.GH_AW_TARGET_REPO_SLUG;
        delete process.env.GITHUB_BASE_REF;
      }
    });

    it("should reject push_to_pull_request_branch when branch still equals base_branch after detection", async () => {
      // Simulates the scenario where getBaseBranch() incorrectly resolves to the
      // feature branch itself. Detection cannot recover when getCurrentBranch()
      // also returns the same branch, so the handler must reject with a clear error.
      process.env.GITHUB_BASE_REF = "repo-assist/fix-issue-129";
      process.env.GITHUB_HEAD_REF = "repo-assist/fix-issue-129";
      process.env.GITHUB_REF_NAME = "repo-assist/fix-issue-129";
      try {
        const result = await handlers.pushToPullRequestBranchHandler({
          branch: "repo-assist/fix-issue-129",
        });

        expect(result.isError).toBe(true);
        const responseData = JSON.parse(result.content[0].text);
        expect(responseData.result).toBe("error");
        expect(responseData.error).toContain("equals base_branch");
        expect(responseData.error).toContain("checked out on the base branch");
        expect(mockAppendSafeOutput).not.toHaveBeenCalled();
      } finally {
        delete process.env.GITHUB_BASE_REF;
        delete process.env.GITHUB_HEAD_REF;
        delete process.env.GITHUB_REF_NAME;
      }
    });

    it("should detect branch from defaultTargetRepo checkout when entry.repo is not provided", async () => {
      const { targetRepoDir } = createSideRepoWithTrackedAndLocalCommits();

      const configWithTarget = {
        push_to_pull_request_branch: { "target-repo": "test-owner/test-repo" },
      };
      const handlersWithTarget = createHandlers(mockServer, mockAppendSafeOutput, configWithTarget);

      process.env.GITHUB_BASE_REF = "main";
      try {
        const result = await handlersWithTarget.pushToPullRequestBranchHandler({});

        expect(result.isError).toBeFalsy();
        expect(mockServer.debug).toHaveBeenCalledWith(expect.stringContaining("Looking for checkout of target repo: test-owner/test-repo"));
        expect(mockServer.debug).toHaveBeenCalledWith(expect.stringContaining(`Selected checkout folder for test-owner/test-repo: ${targetRepoDir}`));
        expect(mockServer.debug).toHaveBeenCalledWith(expect.stringContaining("Using current branch for push_to_pull_request_branch: feature/test-change"));
        expect(mockAppendSafeOutput).toHaveBeenCalledWith(
          expect.objectContaining({
            type: "push_to_pull_request_branch",
            branch: "feature/test-change",
            repo_cwd: targetRepoDir,
          })
        );
      } finally {
        delete process.env.GITHUB_BASE_REF;
      }
    });

    it("should detect branch from the checked out target repo when repo is provided", async () => {
      const { targetRepoDir } = createSideRepoWithTrackedAndLocalCommits();

      process.env.GITHUB_BASE_REF = "main";
      try {
        const result = await handlers.pushToPullRequestBranchHandler({
          repo: "test-owner/test-repo",
        });

        expect(result.isError).toBeFalsy();
        expect(mockServer.debug).toHaveBeenCalledWith(expect.stringContaining(`Selected checkout folder for test-owner/test-repo: ${targetRepoDir}`));
        expect(mockServer.debug).toHaveBeenCalledWith(expect.stringContaining("Using current branch for push_to_pull_request_branch: feature/test-change"));
        expect(mockAppendSafeOutput).toHaveBeenCalledWith(
          expect.objectContaining({
            type: "push_to_pull_request_branch",
            branch: "feature/test-change",
          })
        );
      } finally {
        delete process.env.GITHUB_BASE_REF;
      }
    });

    it("should include repo slug in incremental bundle filename for side-repo checkout (default mode)", async () => {
      const { targetRepoDir } = createSideRepoWithTrackedAndLocalCommits();

      process.env.GITHUB_BASE_REF = "main";
      try {
        const result = await handlers.pushToPullRequestBranchHandler({
          branch: "feature/test-change",
          repo: "test-owner/test-repo",
        });

        expect(result.isError).toBeFalsy();
        const responseData = JSON.parse(result.content[0].text);
        expect(responseData.result).toBe("success");
        expect(path.basename(responseData.bundle.path)).toBe("aw-test-owner-test-repo-feature-test-change.bundle");
        expect(path.basename(responseData.patch.path)).toBe("aw-test-owner-test-repo-feature-test-change.patch");

        expect(mockAppendSafeOutput).toHaveBeenCalledWith(
          expect.objectContaining({
            type: "push_to_pull_request_branch",
            repo_cwd: targetRepoDir,
          })
        );
      } finally {
        delete process.env.GITHUB_BASE_REF;
      }
    });

    it("should fail closed when patch_format resolves to an invalid value", async () => {
      handlers = createHandlers(mockServer, mockAppendSafeOutput, {
        push_to_pull_request_branch: {
          patch_format: "invalid-format",
        },
      });

      const result = await handlers.pushToPullRequestBranchHandler({
        branch: "feature-branch",
      });

      expect(result.isError).toBe(true);
      const responseData = JSON.parse(result.content[0].text);
      expect(responseData.result).toBe("error");
      expect(responseData.error).toContain("Invalid patch_format");
      expect(responseData.error).toContain("am");
      expect(responseData.error).toContain("bundle");
      // Must not echo the raw resolved value (could be a secret expression result)
      expect(responseData.error).not.toContain("invalid-format");
      // Must not have appended any safe output
      expect(mockAppendSafeOutput).not.toHaveBeenCalled();
    });

    it("should fail closed when patch_format resolves to an empty string", async () => {
      handlers = createHandlers(mockServer, mockAppendSafeOutput, {
        push_to_pull_request_branch: {
          patch_format: "",
        },
      });

      const result = await handlers.pushToPullRequestBranchHandler({
        branch: "feature-branch",
      });

      expect(result.isError).toBe(true);
      const responseData = JSON.parse(result.content[0].text);
      expect(responseData.result).toBe("error");
      expect(responseData.error).toContain("Invalid patch_format");
      expect(mockAppendSafeOutput).not.toHaveBeenCalled();
    });

    it("should store resolved base_branch in the safe output entry", async () => {
      // Verifies that base_branch is embedded in the safe output payload so that
      // the apply-time checkout step can use it directly (self-describing safe output).
      // Create a minimal git repo so the push succeeds
      const repoDir = path.join(testWorkspaceDir, "push-test-repo");
      fs.mkdirSync(repoDir, { recursive: true });
      try {
        execSync("git init -b main", { cwd: repoDir, stdio: "pipe" });
        execSync("git config user.name 'Test'", { cwd: repoDir, stdio: "pipe" });
        execSync("git config user.email 'test@test.com'", { cwd: repoDir, stdio: "pipe" });
        execSync("echo 'init' > file.txt && git add . && git commit -m init", { cwd: repoDir, stdio: "pipe" });
        execSync("git checkout -b feature/my-work", { cwd: repoDir, stdio: "pipe" });
        execSync("echo 'change' >> file.txt && git add . && git commit -m change", { cwd: repoDir, stdio: "pipe" });
      } catch {
        // Skip if git not available
        return;
      }

      process.env.GITHUB_BASE_REF = "feature/target-branch";
      process.env.GITHUB_WORKSPACE = repoDir;
      try {
        const result = await handlers.pushToPullRequestBranchHandler({
          branch: "feature/my-work",
        });

        // Whether success or failure, if appendSafeOutput was called the entry should have base_branch
        if (mockAppendSafeOutput.mock.calls.length > 0) {
          expect(mockAppendSafeOutput).toHaveBeenCalledWith(
            expect.objectContaining({
              type: "push_to_pull_request_branch",
              base_branch: "feature/target-branch",
            })
          );
        } else {
          // Patch generation may fail in test environment - just verify no error thrown
          expect(result).toBeDefined();
        }
      } finally {
        delete process.env.GITHUB_BASE_REF;
        process.env.GITHUB_WORKSPACE = testWorkspaceDir;
      }
    });

    /**
     * Reproduces the long-running-branch scenario from the issue:
     * the agent merged the default branch into the PR branch (creating a merge
     * commit), then committed additional work on top. The incremental range
     * origin/<branch>..<branch> therefore contains a merge commit. With bundle
     * now the default transport, this succeeds without requiring an auto-switch.
     */
    function createSideRepoWithMergeCommitInIncrementalRange() {
      const targetRepoDir = path.join(testWorkspaceDir, "target-repo-merge");
      fs.mkdirSync(targetRepoDir, { recursive: true });

      execSync("git init -b main", { cwd: targetRepoDir, stdio: "pipe" });
      execSync("git config user.email 'test@example.com'", { cwd: targetRepoDir, stdio: "pipe" });
      execSync("git config user.name 'Test User'", { cwd: targetRepoDir, stdio: "pipe" });

      // Initial commit on main
      fs.writeFileSync(path.join(targetRepoDir, "README.md"), "base\n");
      execSync("git add README.md", { cwd: targetRepoDir, stdio: "pipe" });
      execSync("git commit -m 'base commit'", { cwd: targetRepoDir, stdio: "pipe" });

      // Create the feature branch with one commit
      execSync("git checkout -b feature/test-change", { cwd: targetRepoDir, stdio: "pipe" });
      fs.writeFileSync(path.join(targetRepoDir, "feature.md"), "feature work\n");
      execSync("git add feature.md", { cwd: targetRepoDir, stdio: "pipe" });
      execSync("git commit -m 'feature commit'", { cwd: targetRepoDir, stdio: "pipe" });
      const featureCommit = execSync("git rev-parse HEAD", { cwd: targetRepoDir, stdio: "pipe" }).toString().trim();

      // Snapshot the remote tracking ref at this point — this is what the agent's
      // workflow checkout would see. Anything ahead of this is "to be pushed".
      execSync("git remote add origin https://github.com/test-owner/test-repo.git", { cwd: targetRepoDir, stdio: "pipe" });
      execSync(`git update-ref refs/remotes/origin/feature/test-change ${featureCommit}`, { cwd: targetRepoDir, stdio: "pipe" });

      // Advance main with new commits (simulating "branch falls behind")
      execSync("git checkout main", { cwd: targetRepoDir, stdio: "pipe" });
      fs.writeFileSync(path.join(targetRepoDir, "main-update.md"), "main moved on\n");
      execSync("git add main-update.md", { cwd: targetRepoDir, stdio: "pipe" });
      execSync("git commit -m 'main update'", { cwd: targetRepoDir, stdio: "pipe" });

      // Agent merges main into feature branch (creates a merge commit)
      execSync("git checkout feature/test-change", { cwd: targetRepoDir, stdio: "pipe" });
      execSync("git merge --no-ff main -m 'Merge main into feature'", { cwd: targetRepoDir, stdio: "pipe" });

      // Agent makes one more commit on top of the merge
      fs.writeFileSync(path.join(targetRepoDir, "feature.md"), "feature work updated\n");
      execSync("git add feature.md", { cwd: targetRepoDir, stdio: "pipe" });
      execSync("git commit -m 'follow-up after merge'", { cwd: targetRepoDir, stdio: "pipe" });
    }

    it("uses bundle transport by default when patch_format is not set", async () => {
      createSideRepoWithMergeCommitInIncrementalRange();

      process.env.GITHUB_BASE_REF = "main";
      try {
        const result = await handlers.pushToPullRequestBranchHandler({
          branch: "feature/test-change",
          repo: "test-owner/test-repo",
        });

        expect(result.isError).toBeFalsy();
        const responseData = JSON.parse(result.content[0].text);
        expect(responseData.result).toBe("success");
        // Must generate a bundle for transport and a patch for policy enforcement
        expect(responseData.bundle).toBeDefined();
        expect(responseData.patch).toBeDefined();
        expect(responseData.bundle.path).toMatch(/\.bundle$/);
        expect(responseData.patch.path).toMatch(/\.patch$/);

        // Default mode is already bundle, so no auto-switch message is required
        const autoSwitchCalls = mockServer.debug.mock.calls.filter(c => typeof c[0] === "string" && c[0].includes("auto-switching to bundle transport"));
        expect(autoSwitchCalls).toHaveLength(0);

        expect(mockAppendSafeOutput).toHaveBeenCalledWith(
          expect.objectContaining({
            type: "push_to_pull_request_branch",
          })
        );
        const appended = mockAppendSafeOutput.mock.calls[0][0];
        // diff_size must be recorded so the downstream push step can validate
        // max_patch_size against the net incremental diff (not the bundle size,
        // which on long-running branches accumulates packed git objects and can
        // exceed the limit even when the actual change is small).
        expect(typeof appended.diff_size).toBe("number");
        expect(appended.diff_size).toBeGreaterThanOrEqual(0);
      } finally {
        delete process.env.GITHUB_BASE_REF;
      }
    });

    it("respects explicit patch_format: am even when incremental range contains a merge commit", async () => {
      createSideRepoWithMergeCommitInIncrementalRange();

      handlers = createHandlers(mockServer, mockAppendSafeOutput, {
        push_to_pull_request_branch: {
          patch_format: "am",
        },
      });

      process.env.GITHUB_BASE_REF = "main";
      try {
        const result = await handlers.pushToPullRequestBranchHandler({
          branch: "feature/test-change",
          repo: "test-owner/test-repo",
        });

        // The user explicitly requested "am", so we must respect it and produce a patch
        // even though the range contains a merge commit. (The patch may later fail to
        // apply, but that is the user's explicit choice.)
        expect(result.isError).toBeFalsy();
        const responseData = JSON.parse(result.content[0].text);
        expect(responseData.result).toBe("success");
        expect(responseData.patch).toBeDefined();
        expect(responseData.bundle).toBeUndefined();

        // Auto-switch debug must NOT have fired
        const autoSwitchCalls = mockServer.debug.mock.calls.filter(c => typeof c[0] === "string" && c[0].includes("auto-switching to bundle transport"));
        expect(autoSwitchCalls).toHaveLength(0);
      } finally {
        delete process.env.GITHUB_BASE_REF;
      }
    });

    it("returns error when getCurrentBranch cannot resolve a branch", async () => {
      // Override the beforeEach seed so every getCurrentBranch resolution path
      // fails (no GITHUB_REF_NAME, no GITHUB_HEAD_REF, no git repo in the empty
      // test workspace) and the handler hits its try/catch.
      delete process.env.GITHUB_REF_NAME;
      delete process.env.GITHUB_HEAD_REF;
      try {
        const result = await handlers.pushToPullRequestBranchHandler({ message: "test" });
        expect(result.isError).toBe(true);
        const data = JSON.parse(result.content[0].text);
        expect(data.result).toBe("error");
        expect(data.error).toContain("Failed to determine source branch");
      } finally {
        process.env.GITHUB_REF_NAME = "feature-branch"; // restore for sibling tests
      }
    });
  });

  describe("handler structure", () => {
    it("should export all required handlers", () => {
      expect(handlers.defaultHandler).toBeDefined();
      expect(handlers.uploadAssetHandler).toBeDefined();
      expect(handlers.uploadArtifactHandler).toBeDefined();
      expect(handlers.createPullRequestHandler).toBeDefined();
      expect(handlers.pushToPullRequestBranchHandler).toBeDefined();
      expect(handlers.pushRepoMemoryHandler).toBeDefined();
      expect(handlers.createIssueHandler).toBeDefined();
      expect(handlers.addCommentHandler).toBeDefined();
      expect(handlers.createPullRequestReviewCommentHandler).toBeDefined();
      expect(handlers.submitPullRequestReviewHandler).toBeDefined();
    });

    it("should create handlers that return proper structure", () => {
      const handler = handlers.defaultHandler("test-type");
      const result = handler({ test: "data" });

      expect(result).toHaveProperty("content");
      expect(Array.isArray(result.content)).toBe(true);
      expect(result.content[0]).toHaveProperty("type");
      expect(result.content[0]).toHaveProperty("text");
    });
  });

  describe("addCommentHandler", () => {
    it("should auto-generate a temporary_id when not provided", () => {
      const result = handlers.addCommentHandler({ body: "Valid comment body" });

      expect(result).toHaveProperty("content");
      const responseData = JSON.parse(result.content[0].text);
      expect(responseData.result).toBe("success");
      expect(responseData.temporary_id).toBeDefined();
      expect(responseData.temporary_id).toMatch(/^aw_[A-Za-z0-9]{3,12}$/);
    });

    it("should use the provided temporary_id when given", () => {
      const result = handlers.addCommentHandler({ body: "Valid comment body", temporary_id: "aw_abc123" });

      expect(result).toHaveProperty("content");
      const responseData = JSON.parse(result.content[0].text);
      expect(responseData.result).toBe("success");
      expect(responseData.temporary_id).toBe("aw_abc123");
    });

    it("should return comment reference using temporary_id", () => {
      const result = handlers.addCommentHandler({ body: "Valid comment body" });

      const responseData = JSON.parse(result.content[0].text);
      expect(responseData.comment).toBe(`#${responseData.temporary_id}`);
    });

    it("should record the temporary_id in the NDJSON entry", () => {
      handlers.addCommentHandler({ body: "Valid comment body", temporary_id: "aw_test01" });

      expect(mockAppendSafeOutput).toHaveBeenCalledWith(
        expect.objectContaining({
          type: "add_comment",
          body: "Valid comment body",
          temporary_id: "aw_test01",
        })
      );
    });

    it("should throw validation error for oversized comment body", () => {
      const longBody = "a".repeat(70000);

      expect(() => handlers.addCommentHandler({ body: longBody })).toThrow();
    });

    it("should reject obvious exploratory placeholder comments before recording them", () => {
      const result = handlers.addCommentHandler({ body: "test" });

      expect(result.isError).toBe(true);
      const responseData = JSON.parse(result.content[0].text);
      expect(responseData.result).toBe("error");
      expect(responseData.error).toContain("Refusing to record an exploratory comment");
      expect(responseData.error).toContain("noop or report_incomplete");
      expect(mockAppendSafeOutput).not.toHaveBeenCalled();
    });

    it("should require explicit item_number when add_comment target is '*'", () => {
      const wildcardHandlers = createHandlers(mockServer, mockAppendSafeOutput, {
        add_comment: {
          target: "*",
        },
      });

      const result = wildcardHandlers.addCommentHandler({ body: "Post a real review summary." });

      expect(result.isError).toBe(true);
      const responseData = JSON.parse(result.content[0].text);
      expect(responseData.result).toBe("error");
      expect(responseData.error).toContain("requires item_number");
      expect(mockAppendSafeOutput).not.toHaveBeenCalled();
    });

    it("should refuse reply_to_id when discussions are not enabled in config", () => {
      // Default handlers have no discussions: true in config
      const result = handlers.addCommentHandler({
        body: "Reply to a discussion thread",
        reply_to_id: "DC_kwDOABcD1M4AaBbC",
      });

      expect(result.isError).toBe(true);
      const responseData = JSON.parse(result.content[0].text);
      expect(responseData.result).toBe("error");
      expect(responseData.error).toContain("discussion comments are not enabled");
      expect(responseData.error).toContain("discussions: true");
      expect(responseData.error).toContain("safe-outputs.add-comment");
      expect(mockAppendSafeOutput).not.toHaveBeenCalled();
    });

    it("should allow reply_to_id when discussions are enabled in config", () => {
      const discussionHandlers = createHandlers(mockServer, mockAppendSafeOutput, {
        "add-comment": { enabled: true, discussions: true },
      });

      const result = discussionHandlers.addCommentHandler({
        body: "Reply to a discussion thread with real content that is not a test placeholder",
        reply_to_id: "DC_kwDOABcD1M4AaBbC",
      });

      expect(result.isError).toBeUndefined();
      const responseData = JSON.parse(result.content[0].text);
      expect(responseData.result).toBe("success");
      expect(mockAppendSafeOutput).toHaveBeenCalledWith(
        expect.objectContaining({
          type: "add_comment",
          reply_to_id: "DC_kwDOABcD1M4AaBbC",
        })
      );
    });
  });

  describe("createIssueHandler", () => {
    it("should append create_issue entry when dedup is disabled", () => {
      handlers.createIssueHandler({ title: "Issue A", body: "Body A" });
      handlers.createIssueHandler({ title: "Issue A", body: "Body A again" });

      expect(mockAppendSafeOutput).toHaveBeenCalledTimes(2);
      const first = mockAppendSafeOutput.mock.calls[0][0];
      const second = mockAppendSafeOutput.mock.calls[1][0];
      expect(first.type).toBe("create_issue");
      expect(second.type).toBe("create_issue");
      expect(second._dropped_duplicate_by_title).toBeUndefined();
    });

    it("should drop duplicate create_issue titles in MCP pre-check when enabled", () => {
      const h = createHandlers(mockServer, mockAppendSafeOutput, {
        create_issue: {
          deduplicate_by_title: true,
        },
      });

      const first = h.createIssueHandler({ title: "Duplicate Issue", body: "First body" });
      const second = h.createIssueHandler({ title: "Duplicate Issue", body: "Second body" });

      const firstResponse = JSON.parse(first.content[0].text);
      const secondResponse = JSON.parse(second.content[0].text);
      expect(firstResponse.result).toBe("success");
      expect(secondResponse.result).toBe("duplicate_dropped");
      const droppedEntry = mockAppendSafeOutput.mock.calls[1][0];
      expect(droppedEntry._dropped_duplicate_by_title).toBe(true);
      expect(droppedEntry._duplicate_distance).toBe(0);
    });

    it("should support Levenshtein distance threshold in MCP pre-check", () => {
      const h = createHandlers(mockServer, mockAppendSafeOutput, {
        create_issue: {
          deduplicate_by_title: 1,
        },
      });

      h.createIssueHandler({ title: "Fix login bug", body: "A" });
      const second = h.createIssueHandler({ title: "Fix login bag", body: "B" });
      const secondResponse = JSON.parse(second.content[0].text);

      expect(secondResponse.result).toBe("duplicate_dropped");
    });

    it("should offload large body when appending duplicate create_issue entry", () => {
      const h = createHandlers(mockServer, mockAppendSafeOutput, {
        create_issue: {
          deduplicate_by_title: true,
        },
      });

      h.createIssueHandler({ title: "Duplicate Issue", body: "First body" });
      h.createIssueHandler({ title: "Duplicate Issue", body: LARGE_CONTENT_BODY });

      expect(mockAppendSafeOutput.mock.calls).toHaveLength(2);
      const droppedEntry = mockAppendSafeOutput.mock.calls[1][0];
      expect(droppedEntry._dropped_duplicate_by_title).toBe(true);
      expect(droppedEntry.body).toContain("[Content too large, saved to file:");
    });

    it("should reject invalid deduplicate-by-title configuration", () => {
      expect(() =>
        createHandlers(mockServer, mockAppendSafeOutput, {
          create_issue: {
            deduplicate_by_title: "invalid",
          },
        })
      ).toThrow("deduplicate-by-title");
    });

    it("should reject obvious exploratory placeholder issues before recording them", () => {
      const result = handlers.createIssueHandler({ title: "test", body: "test" });

      expect(result.isError).toBe(true);
      const responseData = JSON.parse(result.content[0].text);
      expect(responseData.result).toBe("error");
      expect(responseData.error).toContain("Refusing to record an exploratory issue");
      expect(responseData.error).toContain("noop or report_incomplete");
      expect(mockAppendSafeOutput).not.toHaveBeenCalled();
    });

    it("should require temporary_id when configured", () => {
      const h = createHandlers(mockServer, mockAppendSafeOutput, {
        create_issue: {
          require_temporary_id: true,
        },
      });
      const result = h.createIssueHandler({ title: "Issue A", body: "Body A" });

      expect(result.isError).toBe(true);
      const responseData = JSON.parse(result.content[0].text);
      expect(responseData.result).toBe("error");
      expect(responseData.error).toContain("requires 'temporary_id'");
      expect(mockAppendSafeOutput).not.toHaveBeenCalled();
    });

    it("should accept temporary_id when required and provided", () => {
      const h = createHandlers(mockServer, mockAppendSafeOutput, {
        create_issue: {
          require_temporary_id: true,
        },
      });
      const result = h.createIssueHandler({
        title: "Issue A",
        body: "Body A",
        temporary_id: "aw_issue1",
      });

      expect(result.isError).toBeUndefined();
      const responseData = JSON.parse(result.content[0].text);
      expect(responseData.result).toBe("success");
      expect(mockAppendSafeOutput).toHaveBeenCalledWith(
        expect.objectContaining({
          type: "create_issue",
          temporary_id: "aw_issue1",
        })
      );
    });
  });

  describe("pushRepoMemoryHandler", () => {
    let memoryDir;

    beforeEach(() => {
      const testId = Math.random().toString(36).substring(7);
      memoryDir = `/tmp/test-repo-memory-${testId}`;
    });

    afterEach(() => {
      try {
        if (fs.existsSync(memoryDir)) {
          fs.rmSync(memoryDir, { recursive: true, force: true });
        }
      } catch (_error) {
        // Ignore cleanup errors
      }
    });

    function makeHandlersWithMemory(overrides = {}) {
      const memConf = {
        id: "default",
        dir: memoryDir,
        max_file_size: 1024, // 1 KB
        max_patch_size: 2048, // 2 KB
        max_file_count: 5,
        ...overrides,
      };
      return createHandlers(mockServer, mockAppendSafeOutput, {
        push_repo_memory: { memories: [memConf] },
      });
    }

    it("should return success when no repo-memory is configured", () => {
      const h = createHandlers(mockServer, mockAppendSafeOutput, {});
      const result = h.pushRepoMemoryHandler({});
      const data = JSON.parse(result.content[0].text);
      expect(data.result).toBe("success");
      expect(data.message).toContain("No repo-memory configured");
    });

    it("should return error for unknown memory_id", () => {
      const h = makeHandlersWithMemory();
      fs.mkdirSync(memoryDir, { recursive: true });
      const result = h.pushRepoMemoryHandler({ memory_id: "nonexistent" });
      expect(result.isError).toBe(true);
      const data = JSON.parse(result.content[0].text);
      expect(data.result).toBe("error");
      expect(data.error).toContain("'nonexistent' not found");
      expect(data.error).toContain("default");
    });

    it("should return success when memory directory does not exist yet", () => {
      const h = makeHandlersWithMemory();
      // memoryDir not created
      const result = h.pushRepoMemoryHandler({ memory_id: "default" });
      const data = JSON.parse(result.content[0].text);
      expect(data.result).toBe("success");
      expect(data.message).toContain("does not exist yet");
    });

    it("should return success for valid files within limits", () => {
      const h = makeHandlersWithMemory();
      fs.mkdirSync(memoryDir, { recursive: true });
      fs.writeFileSync(path.join(memoryDir, "state.json"), "x".repeat(100));
      const result = h.pushRepoMemoryHandler({ memory_id: "default" });
      const data = JSON.parse(result.content[0].text);
      expect(data.result).toBe("success");
      expect(data.message).toContain("validation passed");
    });

    it("should return error when a file exceeds max_file_size", () => {
      const h = makeHandlersWithMemory({ max_file_size: 100 });
      fs.mkdirSync(memoryDir, { recursive: true });
      fs.writeFileSync(path.join(memoryDir, "big.json"), "x".repeat(200));
      const result = h.pushRepoMemoryHandler({ memory_id: "default" });
      expect(result.isError).toBe(true);
      const data = JSON.parse(result.content[0].text);
      expect(data.result).toBe("error");
      expect(data.error).toContain("big.json");
      expect(data.error).toContain("200 bytes");
    });

    it("should return error when file count exceeds max_file_count", () => {
      const h = makeHandlersWithMemory({ max_file_count: 2 });
      fs.mkdirSync(memoryDir, { recursive: true });
      for (let i = 0; i < 3; i++) {
        fs.writeFileSync(path.join(memoryDir, `file${i}.json`), "x".repeat(10));
      }
      const result = h.pushRepoMemoryHandler({ memory_id: "default" });
      expect(result.isError).toBe(true);
      const data = JSON.parse(result.content[0].text);
      expect(data.result).toBe("error");
      expect(data.error).toContain("Too many files");
      expect(data.error).toContain("3 files");
    });

    it("should return error when total size exceeds effective max_patch_size", () => {
      // max_patch_size = 500 bytes, effective limit = floor(500 * 1.2) = 600 bytes
      const h = makeHandlersWithMemory({ max_patch_size: 500, max_file_size: 1024 * 1024 });
      fs.mkdirSync(memoryDir, { recursive: true });
      // Write two files totaling 650 bytes (above the 600 byte effective limit)
      fs.writeFileSync(path.join(memoryDir, "a.json"), "x".repeat(350));
      fs.writeFileSync(path.join(memoryDir, "b.json"), "x".repeat(300));
      const result = h.pushRepoMemoryHandler({ memory_id: "default" });
      expect(result.isError).toBe(true);
      const data = JSON.parse(result.content[0].text);
      expect(data.result).toBe("error");
      expect(data.error).toContain("exceeds the allowed limit");
      expect(data.error).toContain("push_repo_memory again");
    });

    it("should use 'default' memory_id when memory_id is not specified", () => {
      const h = makeHandlersWithMemory();
      fs.mkdirSync(memoryDir, { recursive: true });
      fs.writeFileSync(path.join(memoryDir, "notes.md"), "hello");
      const result = h.pushRepoMemoryHandler({}); // no memory_id
      const data = JSON.parse(result.content[0].text);
      expect(data.result).toBe("success");
    });

    it("should scan files recursively in subdirectories", () => {
      // max_patch_size = 500 bytes, effective limit = 600 bytes
      const h = makeHandlersWithMemory({ max_patch_size: 500, max_file_size: 1024 * 1024 });
      const subDir = path.join(memoryDir, "history");
      fs.mkdirSync(subDir, { recursive: true });
      // Write a nested file that pushes total above effective limit
      fs.writeFileSync(path.join(subDir, "log.jsonl"), "x".repeat(700));
      const result = h.pushRepoMemoryHandler({ memory_id: "default" });
      expect(result.isError).toBe(true);
      const data = JSON.parse(result.content[0].text);
      expect(data.result).toBe("error");
      // The nested file path should appear correctly
      expect(data.error).toContain("exceeds the allowed limit");
    });

    it("should exclude .git directory from size calculation", () => {
      // Simulate the real scenario: memory directory is a git clone.
      // The .git directory can accumulate pack files across runs.
      // With max_patch_size = 500 bytes (effective limit = 600 bytes), actual memory
      // files are small but .git directory content is large — must not count toward limit.
      const h = makeHandlersWithMemory({ max_patch_size: 500, max_file_size: 1024 * 1024 });
      fs.mkdirSync(memoryDir, { recursive: true });
      // Small memory files (well within limit)
      fs.writeFileSync(path.join(memoryDir, "memory.json"), "x".repeat(100));
      fs.writeFileSync(path.join(memoryDir, "state.json"), "x".repeat(100));
      // Simulate a large .git directory (pack files accumulate with each run)
      const gitDir = path.join(memoryDir, ".git");
      const packDir = path.join(gitDir, "objects", "pack");
      fs.mkdirSync(packDir, { recursive: true });
      fs.writeFileSync(path.join(packDir, "pack-abc123.pack"), "x".repeat(30000));
      // Total without .git: 200 bytes (within 600 byte limit)
      // Total with .git: 30200 bytes (would exceed limit if .git were included)
      const result = h.pushRepoMemoryHandler({ memory_id: "default" });
      const data = JSON.parse(result.content[0].text);
      expect(data.result).toBe("success");
      expect(data.message).toContain("validation passed");
    });
  });

  describe("submitPullRequestReviewHandler", () => {
    // Each test gets fresh handlers via beforeEach, so inlineReviewCommentCount starts at 0.

    it("should write entry and return success when body is provided", () => {
      const result = handlers.submitPullRequestReviewHandler({ body: "Looks good!", event: "COMMENT" });
      expect(result).toHaveProperty("content");
      const data = JSON.parse(result.content[0].text);
      expect(data.result).toBe("success");
      expect(mockAppendSafeOutput).toHaveBeenCalledWith(expect.objectContaining({ type: "submit_pull_request_review", body: "Looks good!" }));
    });

    it("should write entry and return success when body is empty but inline comments were buffered", () => {
      // First buffer an inline comment
      handlers.createPullRequestReviewCommentHandler({ path: "src/foo.js", line: 1, body: "nit" });
      // Then submit with no body — should succeed because comments were buffered
      const result = handlers.submitPullRequestReviewHandler({ event: "COMMENT" });
      expect(result).toHaveProperty("content");
      const data = JSON.parse(result.content[0].text);
      expect(data.result).toBe("success");
    });

    it("should throw MCP error when body is empty and no inline comments were buffered", () => {
      expect(() => handlers.submitPullRequestReviewHandler({ event: "COMMENT" })).toThrow(
        expect.objectContaining({
          code: -32602,
          message: expect.stringContaining("review body is empty"),
        })
      );
    });

    it("should throw MCP error when body is whitespace-only and no inline comments were buffered", () => {
      expect(() => handlers.submitPullRequestReviewHandler({ body: "   ", event: "COMMENT" })).toThrow(expect.objectContaining({ code: -32602 }));
    });

    it("should throw MCP error when event is REQUEST_CHANGES and body is empty", () => {
      expect(() => handlers.submitPullRequestReviewHandler({ event: "REQUEST_CHANGES" })).toThrow(
        expect.objectContaining({
          code: -32602,
          message: expect.stringContaining("'body' is required when event is REQUEST_CHANGES"),
        })
      );
    });

    it("should throw MCP error when event is REQUEST_CHANGES, body is empty, and comments exist", () => {
      handlers.createPullRequestReviewCommentHandler({ path: "src/foo.js", line: 1, body: "nit" });
      expect(() => handlers.submitPullRequestReviewHandler({ event: "REQUEST_CHANGES" })).toThrow(
        expect.objectContaining({
          code: -32602,
          message: expect.stringContaining("'body' is required when event is REQUEST_CHANGES"),
        })
      );
    });

    it("should succeed when event is REQUEST_CHANGES and body is provided", () => {
      const result = handlers.submitPullRequestReviewHandler({ event: "REQUEST_CHANGES", body: "Please fix the issues." });
      const data = JSON.parse(result.content[0].text);
      expect(data.result).toBe("success");
    });

    it("should succeed for APPROVE with no body when inline comments are buffered", () => {
      handlers.createPullRequestReviewCommentHandler({ path: "src/foo.js", line: 1, body: "nit" });
      const result = handlers.submitPullRequestReviewHandler({ event: "APPROVE" });
      const data = JSON.parse(result.content[0].text);
      expect(data.result).toBe("success");
    });

    it("should default to COMMENT event when event is omitted", () => {
      const result = handlers.submitPullRequestReviewHandler({ body: "LGTM" });
      const data = JSON.parse(result.content[0].text);
      expect(data.result).toBe("success");
      expect(mockAppendSafeOutput).toHaveBeenCalledWith(expect.objectContaining({ type: "submit_pull_request_review" }));
    });
    it("should reset inline comment counter after a successful submit, allowing a second review to guard correctly", () => {
      // First review: submit with a body (succeeds, resets counter)
      handlers.createPullRequestReviewCommentHandler({ path: "src/a.js", line: 1, body: "nit" });
      handlers.submitPullRequestReviewHandler({ event: "COMMENT", body: "First review" });

      // Counter is now reset to 0. A second empty-body submit should be rejected.
      expect(() => handlers.submitPullRequestReviewHandler({ event: "COMMENT" })).toThrow(expect.objectContaining({ code: -32602 }));
    });

    it("should throw MCP error when event is an invalid value", () => {
      expect(() => handlers.submitPullRequestReviewHandler({ body: "LGTM", event: "COMMENTT" })).toThrow(
        expect.objectContaining({
          code: -32602,
          message: expect.stringContaining("invalid event 'COMMENTT'"),
        })
      );
    });

    it("should throw MCP error when event has leading/trailing whitespace that resolves to unknown value", () => {
      expect(() => handlers.submitPullRequestReviewHandler({ body: "LGTM", event: "APPROVE " })).toThrow(
        expect.objectContaining({
          code: -32602,
          message: expect.stringContaining("invalid event"),
        })
      );
    });

    it("should accept all valid event values case-insensitively", () => {
      // APPROVE (no body needed when comments buffered, but here we use body)
      expect(() => handlers.submitPullRequestReviewHandler({ body: "LGTM", event: "approve" })).not.toThrow();
      expect(() => handlers.submitPullRequestReviewHandler({ body: "LGTM", event: "comment" })).not.toThrow();
      expect(() => handlers.submitPullRequestReviewHandler({ body: "needs work", event: "request_changes" })).not.toThrow();
    });

    it("should require explicit pull_request_number when submit_pull_request_review target is '*'", () => {
      const wildcardHandlers = createHandlers(mockServer, mockAppendSafeOutput, {
        submit_pull_request_review: {
          target: "*",
        },
      });

      const result = wildcardHandlers.submitPullRequestReviewHandler({ body: "LGTM", event: "COMMENT" });

      expect(result.isError).toBe(true);
      const responseData = JSON.parse(result.content[0].text);
      expect(responseData.result).toBe("error");
      expect(responseData.error).toContain("requires pull_request_number");
      expect(mockAppendSafeOutput).not.toHaveBeenCalled();
    });
  });

  describe("createPullRequestReviewCommentHandler", () => {
    it("should write entry and return success", () => {
      const result = handlers.createPullRequestReviewCommentHandler({ path: "src/foo.js", line: 5, body: "Consider renaming." });
      expect(result).toHaveProperty("content");
      const data = JSON.parse(result.content[0].text);
      expect(data.result).toBe("success");
      expect(mockAppendSafeOutput).toHaveBeenCalledWith(expect.objectContaining({ type: "create_pull_request_review_comment", path: "src/foo.js" }));
    });

    it("should allow empty-body submit after buffering a comment", () => {
      handlers.createPullRequestReviewCommentHandler({ path: "src/bar.js", line: 10, body: "typo" });
      handlers.createPullRequestReviewCommentHandler({ path: "src/baz.js", line: 20, body: "unused import" });
      // Two inline comments buffered — empty-body submit must succeed
      expect(() => handlers.submitPullRequestReviewHandler({ event: "COMMENT" })).not.toThrow();
    });

    it("should not increment counter when the underlying append throws", () => {
      // Make the append call throw to simulate a failure after MCP validation
      mockAppendSafeOutput.mockImplementationOnce(() => {
        throw new Error("write error");
      });
      expect(() => handlers.createPullRequestReviewCommentHandler({ path: "src/foo.js", line: 1, body: "nit" })).toThrow();
      // Counter was NOT incremented, so empty-body submit should still be rejected
      expect(() => handlers.submitPullRequestReviewHandler({ event: "COMMENT" })).toThrow(expect.objectContaining({ code: -32602, message: expect.stringContaining("review body is empty") }));
    });

    it("should require explicit pull_request_number when review comment target is '*'", () => {
      const wildcardHandlers = createHandlers(mockServer, mockAppendSafeOutput, {
        create_pull_request_review_comment: {
          target: "*",
        },
      });

      const result = wildcardHandlers.createPullRequestReviewCommentHandler({ path: "src/foo.js", line: 5, body: "Consider renaming." });

      expect(result.isError).toBe(true);
      const responseData = JSON.parse(result.content[0].text);
      expect(responseData.result).toBe("error");
      expect(responseData.error).toContain("requires pull_request_number");
      expect(mockAppendSafeOutput).not.toHaveBeenCalled();
      expect(() => wildcardHandlers.submitPullRequestReviewHandler({ event: "COMMENT" })).toThrow(expect.objectContaining({ code: -32602, message: expect.stringContaining("review body is empty") }));
    });
  });

  describe("updatePullRequestHandler", () => {
    it("should throw MCP error when no fields are provided", () => {
      expect(() => handlers.updatePullRequestHandler({})).toThrow(
        expect.objectContaining({
          code: -32602,
          message: expect.stringContaining("requires at least one of"),
        })
      );
    });

    it("should throw MCP error when called with null/undefined args", () => {
      expect(() => handlers.updatePullRequestHandler(null)).toThrow(expect.objectContaining({ code: -32602 }));
      expect(() => handlers.updatePullRequestHandler(undefined)).toThrow(expect.objectContaining({ code: -32602 }));
    });

    it("should require explicit pull_request_number when update_pull_request target is '*'", () => {
      const wildcardHandlers = createHandlers(mockServer, mockAppendSafeOutput, {
        update_pull_request: {
          target: "*",
        },
      });

      const result = wildcardHandlers.updatePullRequestHandler({ body: "Update the PR body." });

      expect(result.isError).toBe(true);
      const responseData = JSON.parse(result.content[0].text);
      expect(responseData.result).toBe("error");
      expect(responseData.error).toContain("requires pull_request_number");
      expect(mockAppendSafeOutput).not.toHaveBeenCalled();
    });

    it("should throw MCP error when update_branch is explicitly false and no other fields", () => {
      expect(() => handlers.updatePullRequestHandler({ update_branch: false })).toThrow(expect.objectContaining({ code: -32602 }));
    });

    it("should throw MCP error when title is null", () => {
      expect(() => handlers.updatePullRequestHandler({ title: null })).toThrow(expect.objectContaining({ code: -32602 }));
    });

    it("should throw MCP error when body is null", () => {
      expect(() => handlers.updatePullRequestHandler({ body: null })).toThrow(expect.objectContaining({ code: -32602 }));
    });

    it("should throw MCP error when update_branch is null", () => {
      expect(() => handlers.updatePullRequestHandler({ update_branch: null })).toThrow(expect.objectContaining({ code: -32602 }));
    });

    it("should write entry and return success when title is provided", () => {
      const result = handlers.updatePullRequestHandler({ title: "New Title" });
      expect(result).toHaveProperty("content");
      const data = JSON.parse(result.content[0].text);
      expect(data.result).toBe("success");
      expect(mockAppendSafeOutput).toHaveBeenCalledWith(expect.objectContaining({ type: "update_pull_request", title: "New Title" }));
    });

    it("should write entry and return success when body is provided", () => {
      const result = handlers.updatePullRequestHandler({ body: "Updated body" });
      expect(result).toHaveProperty("content");
      const data = JSON.parse(result.content[0].text);
      expect(data.result).toBe("success");
      expect(mockAppendSafeOutput).toHaveBeenCalledWith(expect.objectContaining({ type: "update_pull_request", body: "Updated body" }));
    });

    it("should write entry and return success when update_branch is true", () => {
      const result = handlers.updatePullRequestHandler({ update_branch: true });
      expect(result).toHaveProperty("content");
      const data = JSON.parse(result.content[0].text);
      expect(data.result).toBe("success");
      expect(mockAppendSafeOutput).toHaveBeenCalledWith(expect.objectContaining({ type: "update_pull_request", update_branch: true }));
    });

    it("should write entry and return success when both title and body are provided", () => {
      const result = handlers.updatePullRequestHandler({ title: "New Title", body: "New body" });
      expect(result).toHaveProperty("content");
      const data = JSON.parse(result.content[0].text);
      expect(data.result).toBe("success");
      expect(mockAppendSafeOutput).toHaveBeenCalledWith(expect.objectContaining({ type: "update_pull_request", title: "New Title", body: "New body" }));
    });

    it("error message should mention all required fields", () => {
      try {
        handlers.updatePullRequestHandler({});
        expect.fail("Should have thrown");
      } catch (err) {
        expect(err.message).toContain("'title'");
        expect(err.message).toContain("'body'");
        expect(err.message).toContain("'update_branch'");
      }
    });
  });
});

describe("hasUpdatePullRequestFields", () => {
  it("returns false for empty object", () => {
    expect(hasUpdatePullRequestFields({})).toBe(false);
  });

  it("returns false for null", () => {
    expect(hasUpdatePullRequestFields(null)).toBe(false);
  });

  it("returns false for undefined", () => {
    expect(hasUpdatePullRequestFields(undefined)).toBe(false);
  });

  it("returns false when update_branch is false", () => {
    expect(hasUpdatePullRequestFields({ update_branch: false })).toBe(false);
  });

  it("returns false when title is null", () => {
    expect(hasUpdatePullRequestFields({ title: null })).toBe(false);
  });

  it("returns false when body is null", () => {
    expect(hasUpdatePullRequestFields({ body: null })).toBe(false);
  });

  it("returns false when update_branch is null", () => {
    expect(hasUpdatePullRequestFields({ update_branch: null })).toBe(false);
  });

  it("returns true when title is a string", () => {
    expect(hasUpdatePullRequestFields({ title: "New Title" })).toBe(true);
  });

  it("returns true when body is a string", () => {
    expect(hasUpdatePullRequestFields({ body: "Updated body" })).toBe(true);
  });

  it("returns true when update_branch is exactly true", () => {
    expect(hasUpdatePullRequestFields({ update_branch: true })).toBe(true);
  });

  it("returns true when both title and body are provided", () => {
    expect(hasUpdatePullRequestFields({ title: "t", body: "b" })).toBe(true);
  });

  it("returns true for empty string title (typeof === 'string')", () => {
    expect(hasUpdatePullRequestFields({ title: "" })).toBe(true);
  });
});
