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
    it("should be defined", () => {
      expect(handlers.createPullRequestHandler).toBeDefined();
    });

    it("should return error response when patch generation fails (not throw)", () => {
      // This test verifies the error is returned as content, not thrown
      // The actual patch generation will fail because we're not in a git repo
      const args = {
        branch: "feature-branch",
        title: "Test PR",
        body: "Test description",
      };

      // The handler should NOT throw an error, it should return an error response
      const result = handlers.createPullRequestHandler(args);

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

    it("should include helpful details in error response", () => {
      const args = {
        branch: "test-branch",
        title: "Test",
        body: "Description",
      };

      const result = handlers.createPullRequestHandler(args);

      expect(result.isError).toBe(true);
      const responseData = JSON.parse(result.content[0].text);

      // Verify the details provide actionable guidance
      expect(responseData.details).toContain("create a pull request");
      expect(responseData.details).toContain("git add");
      expect(responseData.details).toContain("git commit");
      expect(responseData.details).toContain("create_pull_request");
    });
  });

  describe("pushToPullRequestBranchHandler", () => {
    it("should be defined", () => {
      expect(handlers.pushToPullRequestBranchHandler).toBeDefined();
    });

    it("should return error response when patch generation fails (not throw)", () => {
      // This test verifies the error is returned as content, not thrown
      const args = {
        branch: "feature-branch",
      };

      // The handler should NOT throw an error, it should return an error response
      const result = handlers.pushToPullRequestBranchHandler(args);

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
      expect(responseData.details).toContain("push to the pull request branch");
      expect(responseData.details).toContain("git add and git commit");

      // Should not have appended to safe output since patch generation failed
      expect(mockAppendSafeOutput).not.toHaveBeenCalled();
    });

    it("should include helpful details in error response", () => {
      const args = {
        branch: "test-branch",
      };

      const result = handlers.pushToPullRequestBranchHandler(args);

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
