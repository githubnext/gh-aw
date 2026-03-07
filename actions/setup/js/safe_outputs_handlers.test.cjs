import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
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
    it("should generate raw.githubusercontent.com URL for github.com", () => {
      process.env.GH_AW_ASSETS_BRANCH = "test-branch";
      process.env.GITHUB_SERVER_URL = "https://github.com";
      process.env.GITHUB_REPOSITORY = "myorg/myrepo";

      const testFile = path.join(testWorkspaceDir, "test.png");
      fs.writeFileSync(testFile, "test content");

      handlers.uploadAssetHandler({ path: testFile });

      const entry = mockAppendSafeOutput.mock.calls[0][0];
      expect(entry.url).toContain("raw.githubusercontent.com");
      expect(entry.url).toContain("myorg/myrepo");
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
  });

  describe("pushToPullRequestBranchHandler", () => {
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
  });

  describe("handler structure", () => {
    it("should export all required handlers", () => {
      expect(handlers.defaultHandler).toBeDefined();
      expect(handlers.uploadAssetHandler).toBeDefined();
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
  });
});
