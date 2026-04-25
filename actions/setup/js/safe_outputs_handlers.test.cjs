import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
import { execSync } from "child_process";
import { createHandlers } from "./safe_outputs_handlers.cjs";

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

    handlers = createHandlers(mockServer, mockAppendSafeOutput);

    // Create temporary workspace directory
    const testId = Math.random().toString(36).substring(7);
    testWorkspaceDir = `/tmp/test-handlers-workspace-${testId}`;
    fs.mkdirSync(testWorkspaceDir, { recursive: true });

    // Set environment variables
    process.env.GITHUB_WORKSPACE = testWorkspaceDir;
    process.env.GITHUB_SERVER_URL = "https://github.com";
    process.env.GITHUB_REPOSITORY = "owner/repo";
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
    delete process.env.GH_AW_ASSETS_BRANCH;
    delete process.env.GH_AW_ASSETS_MAX_SIZE_KB;
    delete process.env.GH_AW_ASSETS_ALLOWED_EXTS;
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

  describe("createPullRequestHandler", () => {
    it("should be defined", () => {
      expect(handlers.createPullRequestHandler).toBeDefined();
    });

    it("should return error response when patch generation fails (not throw)", async () => {
      // This test verifies the error is returned as content, not thrown
      // The actual patch generation will fail because we're not in a git repo
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
      expect(responseData.error).toContain("Failed to generate patch");
      expect(responseData.details).toBeDefined();
      expect(responseData.details).toContain("Make sure you have committed your changes");
      expect(responseData.details).toContain("git add and git commit");

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
      expect(responseData.details).toContain("create a pull request");
      expect(responseData.details).toContain("git add");
      expect(responseData.details).toContain("git commit");
      expect(responseData.details).toContain("create_pull_request");
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
      // Should be a patch error, not a repo not found error
      expect(responseData.error).not.toContain("not found in workspace");
      expect(responseData.error).toContain("Failed to generate patch");
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
  });

  describe("pushToPullRequestBranchHandler", () => {
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
      expect(responseData.error).toContain("does not exist locally");
      expect(responseData.details).toBeDefined();
      expect(responseData.details).toContain("push to the pull request branch");
      expect(responseData.details).toContain("git add and git commit");

      // Should not have appended to safe output since patch generation failed
      expect(mockAppendSafeOutput).not.toHaveBeenCalled();
    });

    it("should include helpful details in error response", async () => {
      const args = {
        branch: "test-branch",
      };

      const result = await handlers.pushToPullRequestBranchHandler(args);

      expect(result.isError).toBe(true);
      const responseData = JSON.parse(result.content[0].text);

      // Verify the details provide actionable guidance
      expect(responseData.details).toContain("push to the pull request branch");
      expect(responseData.details).toContain("git add");
      expect(responseData.details).toContain("git commit");
      expect(responseData.details).toContain("push_to_pull_request_branch");
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

    it("should detect branch from defaultTargetRepo checkout when entry.repo is not provided", async () => {
      const { targetRepoDir } = createSideRepoWithTrackedAndLocalCommits();

      const configWithTarget = {
        push_to_pull_request_branch: { "target-repo": "test-owner/test-repo" },
      };
      const handlersWithTarget = createHandlers(mockServer, mockAppendSafeOutput, configWithTarget);

      process.env.GITHUB_BASE_REF = "main";
      try {
        const result = await handlersWithTarget.pushToPullRequestBranchHandler({
          branch: "main",
        });

        expect(result.isError).toBeFalsy();
        expect(mockServer.debug).toHaveBeenCalledWith(expect.stringContaining("Looking for checkout of target repo: test-owner/test-repo"));
        expect(mockServer.debug).toHaveBeenCalledWith(expect.stringContaining(`Selected checkout folder for test-owner/test-repo: ${targetRepoDir}`));
        expect(mockServer.debug).toHaveBeenCalledWith(expect.stringContaining("detecting actual working branch: feature/test-change"));
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
          branch: "main",
          repo: "test-owner/test-repo",
        });

        expect(result.isError).toBeFalsy();
        expect(mockServer.debug).toHaveBeenCalledWith(expect.stringContaining(`Selected checkout folder for test-owner/test-repo: ${targetRepoDir}`));
        expect(mockServer.debug).toHaveBeenCalledWith(expect.stringContaining("detecting actual working branch: feature/test-change"));
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

    it("should include repo slug in incremental patch filename for side-repo checkout", async () => {
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
        expect(path.basename(responseData.patch.path)).toBe("aw-test-owner-test-repo-feature-test-change.patch");

        expect(mockAppendSafeOutput).toHaveBeenCalledWith(
          expect.objectContaining({
            type: "push_to_pull_request_branch",
            repo_cwd: targetRepoDir,
            patch_path: expect.stringContaining("aw-test-owner-test-repo-feature-test-change.patch"),
          })
        );
      } finally {
        delete process.env.GITHUB_BASE_REF;
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
      expect(handlers.addCommentHandler).toBeDefined();
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
});
