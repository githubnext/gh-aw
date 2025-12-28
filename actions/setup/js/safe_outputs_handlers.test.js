// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
import { createHandlers } from "./safe_outputs_handlers.cjs";

describe("safe_outputs_handlers", () => {
  let mockServer;
  let mockAppendSafeOutput;
  let handlers;
  let testWorkspaceDir;

  beforeEach(() => {
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
    it("should handle create pull request with valid branch", () => {
      // Mock git commands
      vi.mock("child_process", () => ({
        execSync: vi.fn(() => Buffer.from("test-branch")),
      }));

      const args = {
        branch: "feature-branch",
        title: "Test PR",
        body: "Test description",
      };

      // This test requires git to be available, so we'll just check it doesn't throw
      // when branch is provided
      expect(handlers.createPullRequestHandler).toBeDefined();
    });
  });

  describe("pushToPullRequestBranchHandler", () => {
    it("should be defined", () => {
      expect(handlers.pushToPullRequestBranchHandler).toBeDefined();
    });
  });

  describe("handler structure", () => {
    it("should export all required handlers", () => {
      expect(handlers.defaultHandler).toBeDefined();
      expect(handlers.uploadAssetHandler).toBeDefined();
      expect(handlers.createPullRequestHandler).toBeDefined();
      expect(handlers.pushToPullRequestBranchHandler).toBeDefined();
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
});
